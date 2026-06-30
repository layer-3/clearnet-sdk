package evm

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/log"
)

// configWatcherPollInterval controls how often the watcher polls for new
// confirmed ConfigSet events. Matches the BLS cache cadence; config changes are
// rare and gated on a confirmation window, so a few seconds of latency is fine.
const configWatcherPollInterval = 2 * time.Second

// ConfigState is the cached view of one registry key: the latest content
// checksum and the epoch (write count) it was committed at.
type ConfigState struct {
	Checksum [32]byte
	Epoch    uint64
}

// ConfigWatcherReader is the subset of the Config binding the watcher needs.
// The generated *Config satisfies it; tests inject a fake.
type ConfigWatcherReader interface {
	LatestConfigChecksum(opts *bind.CallOpts, key [32]byte) ([32]byte, error)
	ConfigEpoch(opts *bind.CallOpts, key [32]byte) (uint64, error)
	FilterConfigSet(opts *bind.FilterOpts, key [][32]byte, writer []common.Address) (*ConfigConfigSetIterator, error)
}

var _ ConfigWatcherReader = (*Config)(nil)

// ConfigWatcher maintains an in-memory, confirmation-gated view of one or more
// Config registry keys (ADR-017), modelled on BLSPubkeyCache: a startup
// Backfill of latestConfigChecksum(key)/configEpoch(key), then a Watch loop that
// applies ConfigSet events once they reach `confirmations` depth. It is the
// shared read primitive behind both the daemon's ACTIVATE loop and clearnet's
// RegistrySignerSource.
//
// A confirmed epoch advance fires OnChange (if set) so the consumer can react —
// the daemon resolves the checksum to a held config_versions payload and either
// hot-swaps or restarts; the receipt verifier reloads the signer set. Lookup is
// lock-free in the common case and never issues an RPC.
type ConfigWatcher struct {
	// client is used for head polling. Optional — when nil the watcher is
	// populated via Backfill only and Watch is a no-op (test wiring).
	client *ethclient.Client

	reg           ConfigWatcherReader
	keys          [][32]byte
	confirmations uint64
	pollInterval  time.Duration
	logger        log.Logger

	mu        sync.RWMutex
	state     map[[32]byte]ConfigState
	watermark uint64
	onChange  func(key [32]byte, st ConfigState)
	// started guards the documented "Backfill before Watch" precondition: once
	// Watch is running a second Backfill (e.g. a reconnect path) could race a
	// poll cycle and silently regress state to an older snapshot, so it errors.
	started bool
}

// NewConfigWatcher constructs an empty watcher over the given keys. Call
// Backfill(ctx) before Watch(ctx). client may be nil for unit-test wiring that
// only exercises Lookup; production callers supply the live ethclient.
//
// keys must be non-empty: an empty indexed-key filter is a wildcard in
// eth_getLogs, which would match every ConfigSet event on the contract.
func NewConfigWatcher(client *ethclient.Client, reg ConfigWatcherReader, keys [][32]byte, confirmations uint64) (*ConfigWatcher, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("config watcher: no keys to watch")
	}
	cp := make([][32]byte, len(keys))
	copy(cp, keys)
	return &ConfigWatcher{
		client:        client,
		reg:           reg,
		keys:          cp,
		confirmations: confirmations,
		pollInterval:  configWatcherPollInterval,
		logger:        log.NewNoopLogger(),
		state:         make(map[[32]byte]ConfigState),
	}, nil
}

// SetLogger sets the watcher's logger (defaults to a no-op). Call before
// Backfill or Watch; not safe to call concurrently with them.
func (w *ConfigWatcher) SetLogger(l log.Logger) {
	if l == nil {
		l = log.NewNoopLogger()
	}
	w.logger = l
}

// SetOnChange registers a callback fired whenever a confirmed ConfigSet advances
// a watched key's epoch. Call before Watch; the callback runs on the Watch
// goroutine and must not block.
func (w *ConfigWatcher) SetOnChange(fn func(key [32]byte, st ConfigState)) {
	w.onChange = fn
}

// Lookup returns the cached state for key and whether it is known.
func (w *ConfigWatcher) Lookup(key [32]byte) (ConfigState, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	st, ok := w.state[key]
	return st, ok
}

// LatestChecksum returns the latest confirmed checksum for key and whether it
// is known and non-zero. It is the flattened accessor consumed by
// receipt.RegistrySignerSource (which stays decoupled from this package by
// depending only on this method's shape).
func (w *ConfigWatcher) LatestChecksum(key [32]byte) ([32]byte, bool) {
	st, ok := w.Lookup(key)
	if !ok || st.Checksum == ([32]byte{}) {
		return [32]byte{}, false
	}
	return st.Checksum, true
}

// Watermark returns the highest confirmed L1 block whose events are committed.
func (w *ConfigWatcher) Watermark() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watermark
}

// Backfill seeds the watcher from the registry: for each watched key it reads
// latestConfigChecksum + configEpoch, recording the block height snapshot as the
// watermark so Watch starts exactly at watermark+1.
func (w *ConfigWatcher) Backfill(ctx context.Context) error {
	if w.reg == nil {
		return fmt.Errorf("config watcher: registry not configured")
	}
	w.mu.RLock()
	started := w.started
	w.mu.RUnlock()
	if started {
		return fmt.Errorf("config watcher: Backfill called after Watch started")
	}
	opts := &bind.CallOpts{Context: ctx}

	// Snapshot block height FIRST so any later event is guaranteed not already
	// reflected in the snapshot.
	var startBlock uint64
	if w.client != nil {
		bn, err := w.client.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("config watcher: block number: %w", err)
		}
		startBlock = bn
	}

	snapshot := make(map[[32]byte]ConfigState, len(w.keys))
	for _, key := range w.keys {
		checksum, err := w.reg.LatestConfigChecksum(opts, key)
		if err != nil {
			return fmt.Errorf("config watcher: latest checksum: %w", err)
		}
		epoch, err := w.reg.ConfigEpoch(opts, key)
		if err != nil {
			return fmt.Errorf("config watcher: config epoch: %w", err)
		}
		// epoch 0 / zero checksum means "nothing committed yet" — record it so
		// Lookup can distinguish genesis from an unwatched key.
		snapshot[key] = ConfigState{Checksum: checksum, Epoch: epoch}
	}

	w.mu.Lock()
	w.state = snapshot
	if startBlock > w.watermark { // monotonic watermark
		w.watermark = startBlock
	}
	w.mu.Unlock()

	w.logger.Info("ConfigWatcher backfill complete", "keys", len(w.keys), "watermark", startBlock)
	return nil
}

// Watch polls the chain for ConfigSet events on the watched keys and applies
// them once they reach `confirmations` depth. Returns when ctx is cancelled.
func (w *ConfigWatcher) Watch(ctx context.Context) error {
	w.mu.Lock()
	w.started = true
	w.mu.Unlock()
	if w.client == nil {
		<-ctx.Done()
		return nil
	}
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := w.pollOnce(ctx); err != nil {
				w.logger.Debug("ConfigWatcher poll failed", "error", err)
			}
		}
	}
}

// pollOnce applies every ConfigSet log on the watched keys between
// (watermark, head-confirmations] in order, advancing each key's cached state
// when its epoch increases and firing OnChange for confirmed advances.
func (w *ConfigWatcher) pollOnce(ctx context.Context) error {
	head, err := w.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("block number: %w", err)
	}
	if head < w.confirmations {
		return nil
	}
	confirmed := head - w.confirmations

	w.mu.RLock()
	from := w.watermark + 1
	w.mu.RUnlock()
	if from > confirmed {
		return nil
	}

	it, err := w.reg.FilterConfigSet(
		&bind.FilterOpts{Context: ctx, Start: from, End: &confirmed},
		w.keys, // indexed-key filter
		nil,
	)
	if err != nil {
		return fmt.Errorf("filter ConfigSet: %w", err)
	}
	defer it.Close()

	type change struct {
		key [32]byte
		st  ConfigState
	}
	var changes []change

	w.mu.Lock()
	for it.Next() {
		ev := it.Event
		cur, ok := w.state[ev.Key]
		if !ok {
			// An empty indexed-key filter is a wildcard, so a foreign writer's
			// ConfigSet on an unwatched key can surface here. Only Backfill-seeded
			// (watched) keys belong in state — drop the rest.
			continue
		}
		if ev.Epoch <= cur.Epoch {
			continue // stale or out-of-order log; keep the higher epoch
		}
		st := ConfigState{Checksum: ev.Checksum, Epoch: ev.Epoch}
		w.state[ev.Key] = st
		changes = append(changes, change{key: ev.Key, st: st})
	}
	// Check the iterator error BEFORE advancing the watermark: a mid-stream
	// failure means some logs in (watermark, confirmed] went unprocessed, so the
	// watermark must not move past them or the next poll would skip them forever.
	if err := it.Error(); err != nil {
		w.mu.Unlock()
		return fmt.Errorf("iterate ConfigSet: %w", err)
	}
	w.watermark = confirmed
	w.mu.Unlock()

	for _, c := range changes {
		w.logger.Info("ConfigWatcher: ConfigSet committed",
			"key", "0x"+common.Bytes2Hex(c.key[:]),
			"checksum", "0x"+common.Bytes2Hex(c.st.Checksum[:]),
			"epoch", c.st.Epoch,
		)
		if w.onChange != nil {
			w.onChange(c.key, c.st)
		}
	}
	return nil
}
