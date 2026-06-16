package evm

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// BLSPubkeyCacheSize is the expected G2 serialization length (ADR-008 / ISSUE-035
// WS-3): 128 bytes in the X.A1 || X.A0 || Y.A1 || Y.A0 layout.
const BLSPubkeyCacheSize = 128

// g2Zero reports whether a [4]*big.Int G2 pubkey equals the zero point.
// Registry-side a zero key indicates "no BLS key bound to this slot" — should
// not happen for active or unbonding nodes, but the cache treats them as absent.
func g2Zero(g2 [4]*big.Int) bool {
	for _, c := range g2 {
		if c != nil && c.Sign() != 0 {
			return false
		}
	}
	return true
}

// serializeG2 converts a [4]*big.Int G2 pubkey (x_im, x_re, y_im, y_re) into
// the 128-byte serialized form consumed by `consensus.DeserializeG2` and
// carried on block attestations. Layout matches `consensus.SerializeG2`:
// [X.A1 || X.A0 || Y.A1 || Y.A0].
func serializeG2(g2 [4]*big.Int) []byte {
	buf := make([]byte, BLSPubkeyCacheSize)
	for i, c := range g2 {
		if c == nil {
			continue
		}
		c.FillBytes(buf[i*32 : (i+1)*32])
	}
	return buf
}

// BLSPubkeyCache maintains an in-memory map of nodeId → 128-byte G2 pubkey
// populated from the on-chain Registry (ADR-008 / 2026-05-16 amendment: BLS
// keys live on NodeRecord directly).
//
// Lifecycle:
//  1. Backfill(ctx) — startup full-sync via paginated `GetNodeIds` + `GetNodes`.
//     Records the block height (watermark) so follow-up subscription starts
//     exactly at the next block — no gap, no duplicate processing.
//  2. Watch(ctx) — poll for NodeActivated + NodeReleased events and update the
//     map. Held until N confirmations deep to absorb reorgs.
//
// Lookup is lock-free in the common case (sync.RWMutex read path) and never
// issues an RPC — missing entries return nil and the caller hard-fails per
// ADR-008 "drop the NodeID fallback" step.
type BLSPubkeyCache struct {
	// client is used for log subscriptions (FilterLogs) and head polling.
	// Optional — when nil the cache is populated via Backfill only and
	// Watch is a no-op (used by tests that don't need live updates).
	client *ethclient.Client

	// registryAddr is required when client is set, so FilterLogs can filter
	// logs by contract address.
	registryAddr common.Address

	// reg is the thin binding surface needed for the initial full-sync and
	// ad-hoc single-node lookups.
	reg BLSPubkeyCacheRegistry

	// confirmations is the number of L1 block confirmations to wait before
	// committing a NodeActivated / NodeReleased event into the cache.
	confirmations uint64

	mu sync.RWMutex
	// keys is the forward index: nodeId → 128-byte G2 pubkey.
	keys map[core.NodeID][]byte
	// byPubkey is the reverse index: string(pubkey) → nodeId.
	byPubkey  map[string]core.NodeID
	watermark uint64 // highest block height whose events are committed
}

// BLSPubkeyCacheRegistry is the subset of the Registry binding used by the
// cache. Decouples from *Registry so tests can inject a stub.
//
// TODO: refactor the cache off this binding-shaped interface. Backfill can use
// core.RegistryReader directly — GetNodes returns []*core.Slot, where each
// Slot already carries .ID (nodeId) and .BLSPubKey (the serialized G2), so the
// GetNodeIds call and the local g2Zero/serializeG2 helpers become unnecessary.
// Watch cannot: it needs ethclient.FilterLogs over NodeActivated/NodeReleased,
// which is event-subscription, not a contract read. Doing this properly means
// introducing a log-subscription seam (so the cache depends on RegistryReader
// for sync + a log source for updates) rather than the raw *ethclient.Client.
type BLSPubkeyCacheRegistry interface {
	TotalNodes(opts *bind.CallOpts) (*big.Int, error)
	GetNodeIds(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([][32]byte, error)
	GetNodes(opts *bind.CallOpts, offset *big.Int, limit *big.Int) ([]NodeRecord, error)
	GetNodeById(opts *bind.CallOpts, nodeId [32]byte) (NodeRecord, error)
}

// compile-time check: the generated Registry binding satisfies the cache's
// read surface (methods are hoisted via RegistryCaller embedding).
var _ BLSPubkeyCacheRegistry = (*Registry)(nil)

// NewBLSPubkeyCache constructs an empty cache. Call Backfill(ctx) before
// Watch(ctx). The client may be nil for unit-test wiring that only exercises
// the lookup path; production callers supply the live ethclient.
func NewBLSPubkeyCache(client *ethclient.Client, registryAddr common.Address, reg BLSPubkeyCacheRegistry, confirmations uint64) *BLSPubkeyCache {
	return &BLSPubkeyCache{
		client:        client,
		registryAddr:  registryAddr,
		reg:           reg,
		confirmations: confirmations,
		keys:          make(map[core.NodeID][]byte),
		byPubkey:      make(map[string]core.NodeID),
	}
}

// assignLocked writes (id → pubkey) into both indices. Caller MUST hold the
// write lock. pubkey is expected to be exactly BLSPubkeyCacheSize bytes; the
// caller already validated length. If id previously mapped to a different
// pubkey (key rotation, re-lock), the stale reverse entry is removed first
// to keep the indices consistent.
func (c *BLSPubkeyCache) assignLocked(id core.NodeID, pubkey []byte) {
	if old, ok := c.keys[id]; ok {
		delete(c.byPubkey, string(old))
	}
	c.keys[id] = pubkey
	c.byPubkey[string(pubkey)] = id
}

// removeLocked deletes id from both indices. Caller MUST hold the write lock.
func (c *BLSPubkeyCache) removeLocked(id core.NodeID) {
	if old, ok := c.keys[id]; ok {
		delete(c.byPubkey, string(old))
	}
	delete(c.keys, id)
}

// Lookup returns the cached 128-byte G2 pubkey for a Slot, or nil if unknown.
// Wired into `cluster.SigningCoordinator.BLSPubKeyLookup` in production so
// SealBlock populates block.Validators with Registry-authoritative pubkeys.
func (c *BLSPubkeyCache) Lookup(id core.NodeID) []byte {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.keys[id]
}

// Size returns the current cache size. Intended for diagnostics / tests.
func (c *BLSPubkeyCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.keys)
}

// Watermark returns the highest confirmed L1 block whose events have been
// committed into the cache. Advances on Backfill + on each confirmed event
// batch. Intended for diagnostics.
func (c *BLSPubkeyCache) Watermark() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.watermark
}

// Put records a (nodeId → G2 pubkey) entry. Exposed so tests can seed the
// cache without a chain adapter; production code always goes through
// Backfill / Watch. pubkey must be exactly 128 bytes (BLSPubkeyCacheSize).
func (c *BLSPubkeyCache) Put(id core.NodeID, pubkey []byte) {
	if len(pubkey) != BLSPubkeyCacheSize {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	buf := make([]byte, BLSPubkeyCacheSize)
	copy(buf, pubkey)
	c.assignLocked(id, buf)
}

// Delete removes an entry. Invoked by Watch on a confirmed NodeReleased event.
func (c *BLSPubkeyCache) Delete(id core.NodeID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeLocked(id)
}

// NodeIDForPubkey returns the NodeID that registered the given 128-byte G2
// pubkey on chain, or (zero, false) if no active node carries that pubkey.
// Off-chain verifiers (custody) use this to authorize signers carried in a
// block's Attestation.Validators against the live Registry — without it, the
// pairing check alone would accept any well-formed signature, including those
// produced by self-elected attacker keys.
func (c *BLSPubkeyCache) NodeIDForPubkey(pubkey []byte) (core.NodeID, bool) {
	if len(pubkey) != BLSPubkeyCacheSize {
		return core.NodeID{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	id, ok := c.byPubkey[string(pubkey)]
	return id, ok
}

// Backfill seeds the cache from the Registry by paginating GetNodeIds +
// GetNodes (both iterate `_nodeIds` in identical order, so positions match
// by index). Records the block height at which the snapshot was taken as
// the watermark so subsequent Watch starts exactly at watermark+1.
//
// Returns the set of (active + unbonding) nodes whose BLS keys we will need
// during the unbonding window — slashing evidence against an unbonding node
// must still verify, so the cache holds keys until `NodeReleased`.
func (c *BLSPubkeyCache) Backfill(ctx context.Context) error {
	if c.reg == nil {
		return fmt.Errorf("bls cache: registry not configured")
	}
	opts := &bind.CallOpts{Context: ctx}

	// Snapshot block height FIRST so we know the floor for Watch — any event
	// that arrives at a later block is guaranteed not already in the snapshot.
	var startBlock uint64
	if c.client != nil {
		bn, err := c.client.BlockNumber(ctx)
		if err != nil {
			return fmt.Errorf("bls cache: block number: %w", err)
		}
		startBlock = bn
	}

	total, err := c.reg.TotalNodes(opts)
	if err != nil {
		return fmt.Errorf("bls cache: total nodes: %w", err)
	}
	if total == nil || total.Sign() == 0 {
		c.setWatermark(startBlock)
		return nil
	}

	ids, err := c.reg.GetNodeIds(opts, big.NewInt(0), total)
	if err != nil {
		return fmt.Errorf("bls cache: get node ids: %w", err)
	}
	if len(ids) == 0 {
		c.setWatermark(startBlock)
		return nil
	}

	records, err := c.reg.GetNodes(opts, big.NewInt(0), total)
	if err != nil {
		return fmt.Errorf("bls cache: get nodes: %w", err)
	}
	if len(records) != len(ids) {
		return fmt.Errorf("bls cache: record count mismatch: got %d records, want %d", len(records), len(ids))
	}

	c.mu.Lock()
	for i, id := range ids {
		if g2Zero(records[i].BlsPubkeyG2) {
			continue
		}
		c.assignLocked(core.NodeID(id), serializeG2(records[i].BlsPubkeyG2))
	}
	// B6: monotonic watermark — never rewind past a prior Watch-driven advance.
	if startBlock > c.watermark {
		c.watermark = startBlock
	}
	c.mu.Unlock()

	slog.Info("BLSPubkeyCache backfill complete",
		"entries", c.Size(),
		"watermark", startBlock,
		"total_nodes", total.String(),
	)
	return nil
}

func (c *BLSPubkeyCache) setWatermark(h uint64) {
	c.mu.Lock()
	// B6: monotonic watermark — never regress below a prior advance.
	if h > c.watermark {
		c.watermark = h
	}
	c.mu.Unlock()
}

// Event signatures for the post 2026-05-16 Registry shape:
//
//	event NodeActivated(
//	    address indexed operator,    // topic[1]
//	    bytes32 indexed nodeId,      // topic[2]
//	    uint32  indexed tokenId,     // topic[3]
//	    uint256 collateral,          // data[0..32)
//	    uint64  vestedAt,            // data[32..64)
//	    uint256[4] blsPubkeyG2       // data[64..192)
//	);
//
//	event NodeReleased(
//	    address indexed operator,    // topic[1]
//	    bytes32 indexed nodeId,      // topic[2]
//	    uint32  indexed tokenId,     // topic[3]
//	    uint256 collateral           // data[0..32)
//	);
var (
	nodeActivatedEventSig = crypto.Keccak256Hash([]byte("NodeActivated(address,bytes32,uint32,uint256,uint64,uint256[4])"))
	nodeReleasedEventSig  = crypto.Keccak256Hash([]byte("NodeReleased(address,bytes32,uint32,uint256)"))

	// blsCachePollInterval controls how often the watcher polls for new
	// confirmed events. The cache is used by SigningCoordinator.BLSPubKeyLookup
	// which fires on every block seal, so missing a newly-activated node for
	// up to one interval is acceptable (PeerAnnouncement fast-path covers
	// the gap).
	blsCachePollInterval = 2 * time.Second
)

// Watch polls the chain for NodeActivated / NodeReleased events and updates
// the cache once they reach c.confirmations depth. Returns when ctx is
// cancelled.
//
// Design: HTTP-poll (vs WatchLogs subscription) — subscriptions require a
// websocket client which is not always available (QA deploys use plain HTTP);
// polling is the common denominator. The poll interval is small enough (2s)
// that the fallback to PeerAnnouncement's fast-path covers any in-flight gap.
func (c *BLSPubkeyCache) Watch(ctx context.Context) error {
	if c.client == nil {
		<-ctx.Done()
		return nil
	}
	if c.registryAddr == (common.Address{}) {
		return fmt.Errorf("bls cache: registry address required for Watch")
	}

	ticker := time.NewTicker(blsCachePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.pollOnce(ctx); err != nil {
				slog.Debug("BLSPubkeyCache poll failed", "error", err)
			}
		}
	}
}

// pollOnce fetches every NodeActivated + NodeReleased log between
// (watermark, head-confirmations] and applies them to the cache in order.
// Safe to call concurrently with Lookup — only takes the write lock while
// mutating the map.
func (c *BLSPubkeyCache) pollOnce(ctx context.Context) error {
	head, err := c.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("block number: %w", err)
	}
	if head < c.confirmations {
		return nil
	}
	confirmed := head - c.confirmations

	c.mu.RLock()
	from := c.watermark + 1
	c.mu.RUnlock()
	if from > confirmed {
		return nil // nothing new to commit
	}

	query := ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(from),
		ToBlock:   new(big.Int).SetUint64(confirmed),
		Addresses: []common.Address{c.registryAddr},
		Topics: [][]common.Hash{
			{nodeActivatedEventSig, nodeReleasedEventSig},
		},
	}

	logs, err := c.client.FilterLogs(ctx, query)
	if err != nil {
		return fmt.Errorf("filter logs: %w", err)
	}

	for _, l := range logs {
		c.applyLog(l)
	}

	c.mu.Lock()
	c.watermark = confirmed
	c.mu.Unlock()
	return nil
}

// applyLog parses a single NodeActivated / NodeReleased log and updates the
// cache. Unknown topics are ignored so a future ABI extension doesn't break
// the poller.
func (c *BLSPubkeyCache) applyLog(l types.Log) {
	if len(l.Topics) < 3 {
		return
	}
	// Topic[2] = indexed nodeId (bytes32) for NodeActivated and NodeReleased.
	var nodeID core.NodeID
	copy(nodeID[:], l.Topics[2].Bytes())

	switch l.Topics[0] {
	case nodeActivatedEventSig:
		// Event data layout (non-indexed fields, 32-byte slots):
		//   [0..32)    uint256 collateral
		//   [32..64)   uint64 vestedAt (right-padded)
		//   [64..192)  uint256[4] blsPubkeyG2 — [x_im, x_re, y_im, y_re]
		if len(l.Data) < 192 {
			slog.Warn("NodeActivated log has short data", "len", len(l.Data), "tx", l.TxHash.Hex())
			return
		}
		pubkey := make([]byte, BLSPubkeyCacheSize)
		copy(pubkey, l.Data[64:192])
		c.mu.Lock()
		c.assignLocked(nodeID, pubkey)
		c.mu.Unlock()
		slog.Debug("BLSPubkeyCache: NodeActivated committed",
			"node", fmt.Sprintf("%x", nodeID[:8]),
			"block", l.BlockNumber,
		)
	case nodeReleasedEventSig:
		c.mu.Lock()
		c.removeLocked(nodeID)
		c.mu.Unlock()
		slog.Debug("BLSPubkeyCache: NodeReleased committed",
			"node", fmt.Sprintf("%x", nodeID[:8]),
			"block", l.BlockNumber,
		)
	}
}

// LookupWithRefresh returns the pubkey for id, issuing a single cold-miss
// `GetNodeById` RPC if the cache does not yet carry an entry. Intended as a
// narrow escape hatch: a freshly-activated node whose NodeActivated event
// has not yet cleared the confirmation window but whose partial signature
// the sealer wants to include. Returns nil + false when both the cache and
// the on-chain lookup return zero.
func (c *BLSPubkeyCache) LookupWithRefresh(ctx context.Context, id core.NodeID) ([]byte, bool) {
	if k := c.Lookup(id); len(k) > 0 {
		return k, true
	}
	if c.reg == nil {
		return nil, false
	}
	rec, err := c.reg.GetNodeById(&bind.CallOpts{Context: ctx}, [32]byte(id))
	if err != nil {
		slog.Debug("BLSPubkeyCache: cold-miss lookup failed", "error", err, "node", fmt.Sprintf("%x", id[:8]))
		return nil, false
	}
	if g2Zero(rec.BlsPubkeyG2) {
		return nil, false
	}
	buf := serializeG2(rec.BlsPubkeyG2)
	c.Put(id, buf)
	return buf, true
}
