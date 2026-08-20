package evm

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/log"
)

// ConfigRegistryEventHandler receives confirmed registry events in chain order.
// It should persist the config record and cursor atomically; the watcher advances
// only after HandleConfigRegistryEvent returns nil.
type ConfigRegistryEventHandler interface {
	HandleConfigRegistryEvent(ctx context.Context, ev core.ConfigRegistryEvent) error
}

// ConfigRegistryCursorSource returns the latest cursor previously persisted by
// the handler/store. If no cursor exists, the watcher starts from a recent
// lookback window.
type ConfigRegistryCursorSource interface {
	LatestConfigRegistryCursor(ctx context.Context, registry common.Address) (core.ConfigRegistryCursor, bool, error)
}

// ConfigRegistryWatcherReader supplies decoded registry events over a block
// range. The production adapter wraps the generated *ConfigRegistry binding;
// tests inject fakes without constructing abigen iterators.
type ConfigRegistryWatcherReader interface {
	FilterConfigCommittedEvents(ctx context.Context, from, to uint64) ([]*ConfigRegistryConfigCommitted, error)
	FilterConfigWithDataCommittedEvents(ctx context.Context, from, to uint64) ([]*ConfigRegistryConfigWithDataCommitted, error)
}

type blockNumberReader interface {
	BlockNumber(ctx context.Context) (uint64, error)
}

var _ blockNumberReader = (*ethclient.Client)(nil)

// ConfigRegistryWatcher polls ConfigRegistry for confirmed ConfigCommitted and
// ConfigWithDataCommitted events with broad filters, normalizes them, and
// delivers them to a handler. It keeps no signer/config business state.
type ConfigRegistryWatcher struct {
	client       blockNumberReader
	registry     ConfigRegistryWatcherReader
	registryAddr common.Address
	handler      ConfigRegistryEventHandler
	cursorSource ConfigRegistryCursorSource

	confirmations  uint64
	lookbackBlocks uint64
	pollInterval   time.Duration
	logger         log.Logger

	mu        sync.RWMutex
	watermark uint64
	cursor    core.ConfigRegistryCursor
	hasCursor bool
	started   bool
}

func NewConfigRegistryWatcher(client *ethclient.Client, registryAddr common.Address, registry *ConfigRegistry, confirmations uint64, handler ConfigRegistryEventHandler) (*ConfigRegistryWatcher, error) {
	if registry == nil {
		return nil, fmt.Errorf("config registry watcher: nil registry")
	}
	return newConfigRegistryWatcher(client, registryAddr, configRegistryBindingReader{registry: registry}, confirmations, handler)
}

func newConfigRegistryWatcher(client blockNumberReader, registryAddr common.Address, registry ConfigRegistryWatcherReader, confirmations uint64, handler ConfigRegistryEventHandler) (*ConfigRegistryWatcher, error) {
	if registry == nil {
		return nil, fmt.Errorf("config registry watcher: nil registry")
	}
	if handler == nil {
		return nil, fmt.Errorf("config registry watcher: nil handler")
	}
	return &ConfigRegistryWatcher{
		client:         client,
		registry:       registry,
		registryAddr:   registryAddr,
		handler:        handler,
		confirmations:  confirmations,
		lookbackBlocks: defaultConfigLookupWindow,
		pollInterval:   configWatcherPollInterval,
		logger:         log.NewNoopLogger(),
	}, nil
}

func (w *ConfigRegistryWatcher) SetLogger(l log.Logger) {
	if l == nil {
		l = log.NewNoopLogger()
	}
	w.logger = l
}

func (w *ConfigRegistryWatcher) SetCursorSource(src ConfigRegistryCursorSource) {
	w.cursorSource = src
}

func (w *ConfigRegistryWatcher) SetInitialLookback(blocks uint64) {
	w.lookbackBlocks = blocks
}

func (w *ConfigRegistryWatcher) SetPollInterval(interval time.Duration) {
	if interval > 0 {
		w.pollInterval = interval
	}
}

func (w *ConfigRegistryWatcher) Watermark() uint64 {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.watermark
}

func (w *ConfigRegistryWatcher) Cursor() (core.ConfigRegistryCursor, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.cursor, w.hasCursor
}

// Backfill initializes the in-memory cursor from the durable cursor source. If
// no cursor exists, it starts from a recent confirmed-block lookback so payload
// events committed shortly before startup can still be replayed into the store.
func (w *ConfigRegistryWatcher) Backfill(ctx context.Context) error {
	w.mu.RLock()
	started := w.started
	w.mu.RUnlock()
	if started {
		return fmt.Errorf("config registry watcher: Backfill called after Watch started")
	}

	if w.cursorSource != nil {
		cur, ok, err := w.cursorSource.LatestConfigRegistryCursor(ctx, w.registryAddr)
		if err != nil {
			return fmt.Errorf("config registry watcher: load cursor: %w", err)
		}
		if ok {
			w.mu.Lock()
			w.cursor = cur
			w.hasCursor = true
			w.watermark = cur.BlockNumber
			w.mu.Unlock()
			w.logger.Info("ConfigRegistryWatcher cursor restored",
				"registry", w.registryAddr.Hex(), "block", cur.BlockNumber, "logIndex", cur.LogIndex)
			return nil
		}
	}

	var start uint64
	if w.client != nil {
		head, err := w.client.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("config registry watcher: block number: %w", err)
		}
		var confirmed uint64
		if head >= w.confirmations {
			confirmed = head - w.confirmations
		}
		if confirmed > w.lookbackBlocks {
			start = confirmed - w.lookbackBlocks
		}
	}

	w.mu.Lock()
	w.cursor = core.ConfigRegistryCursor{}
	w.hasCursor = false
	w.watermark = start
	w.mu.Unlock()
	w.logger.Info("ConfigRegistryWatcher backfill complete",
		"registry", w.registryAddr.Hex(), "watermark", start, "lookback", w.lookbackBlocks)
	return nil
}

func (w *ConfigRegistryWatcher) Watch(ctx context.Context) error {
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
				w.logger.Debug("ConfigRegistryWatcher poll failed", "error", err)
			}
		}
	}
}

func (w *ConfigRegistryWatcher) pollOnce(ctx context.Context) error {
	if w.client == nil {
		return nil
	}
	head, err := w.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("block number: %w", err)
	}
	if head < w.confirmations {
		return nil
	}
	confirmed := head - w.confirmations

	w.mu.RLock()
	watermark := w.watermark
	cur := w.cursor
	hasCursor := w.hasCursor
	w.mu.RUnlock()

	from := watermark
	if !hasCursor && from < confirmed {
		from++
	}
	if from > confirmed {
		return nil
	}

	events, err := w.fetchEvents(ctx, from, confirmed)
	if err != nil {
		return err
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].BlockNumber != events[j].BlockNumber {
			return events[i].BlockNumber < events[j].BlockNumber
		}
		return events[i].LogIndex < events[j].LogIndex
	})

	for _, ev := range events {
		if hasCursor && ev.BlockNumber == cur.BlockNumber && ev.LogIndex <= cur.LogIndex {
			continue
		}
		if err := w.handler.HandleConfigRegistryEvent(ctx, ev); err != nil {
			return fmt.Errorf("handle config registry event: %w", err)
		}
		w.mu.Lock()
		w.cursor = core.ConfigRegistryCursor{Registry: w.registryAddr, BlockNumber: ev.BlockNumber, LogIndex: ev.LogIndex, TxHash: ev.TxHash}
		w.hasCursor = true
		w.watermark = ev.BlockNumber
		w.mu.Unlock()
		hasCursor = true
		cur = core.ConfigRegistryCursor{Registry: w.registryAddr, BlockNumber: ev.BlockNumber, LogIndex: ev.LogIndex, TxHash: ev.TxHash}
	}

	w.mu.Lock()
	if confirmed > w.watermark {
		w.watermark = confirmed
	}
	w.mu.Unlock()
	return nil
}

func (w *ConfigRegistryWatcher) fetchEvents(ctx context.Context, from, to uint64) ([]core.ConfigRegistryEvent, error) {
	out := make([]core.ConfigRegistryEvent, 0)

	committed, err := w.registry.FilterConfigCommittedEvents(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("filter ConfigCommitted: %w", err)
	}
	for _, ev := range committed {
		out = append(out, core.ConfigRegistryEvent{
			Registry:    w.registryAddr,
			IssuerID:    ev.IssuerId,
			Key:         ev.Key,
			Checksum:    ev.Checksum,
			HasData:     false,
			NewNonce:    cloneBig(ev.NewNonce),
			BlockNumber: ev.Raw.BlockNumber,
			LogIndex:    ev.Raw.Index,
			TxHash:      ev.Raw.TxHash,
		})
	}

	withData, err := w.registry.FilterConfigWithDataCommittedEvents(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("filter ConfigWithDataCommitted: %w", err)
	}
	for _, ev := range withData {
		data := append([]byte(nil), ev.Data...)
		if crypto.Keccak256Hash(data) != ev.Checksum {
			return nil, fmt.Errorf("ConfigWithDataCommitted checksum mismatch: issuer=%s key=0x%s tx=%s",
				ev.IssuerId.Hex(), common.Bytes2Hex(ev.Key[:]), ev.Raw.TxHash.Hex())
		}
		out = append(out, core.ConfigRegistryEvent{
			Registry:    w.registryAddr,
			IssuerID:    ev.IssuerId,
			Key:         ev.Key,
			Checksum:    ev.Checksum,
			Data:        data,
			HasData:     true,
			NewNonce:    cloneBig(ev.NewNonce),
			BlockNumber: ev.Raw.BlockNumber,
			LogIndex:    ev.Raw.Index,
			TxHash:      ev.Raw.TxHash,
		})
	}

	return out, nil
}

type configRegistryBindingReader struct {
	registry *ConfigRegistry
}

func (r configRegistryBindingReader) FilterConfigCommittedEvents(ctx context.Context, from, to uint64) ([]*ConfigRegistryConfigCommitted, error) {
	it, err := r.registry.FilterConfigCommitted(&bind.FilterOpts{Context: ctx, Start: from, End: &to}, nil, nil)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	var out []*ConfigRegistryConfigCommitted
	for it.Next() {
		out = append(out, it.Event)
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r configRegistryBindingReader) FilterConfigWithDataCommittedEvents(ctx context.Context, from, to uint64) ([]*ConfigRegistryConfigWithDataCommitted, error) {
	it, err := r.registry.FilterConfigWithDataCommitted(&bind.FilterOpts{Context: ctx, Start: from, End: &to}, nil, nil)
	if err != nil {
		return nil, err
	}
	defer it.Close()
	var out []*ConfigRegistryConfigWithDataCommitted
	for it.Next() {
		out = append(out, it.Event)
	}
	if err := it.Error(); err != nil {
		return nil, err
	}
	return out, nil
}

func cloneBig(v *big.Int) *big.Int {
	if v == nil {
		return nil
	}
	return new(big.Int).Set(v)
}
