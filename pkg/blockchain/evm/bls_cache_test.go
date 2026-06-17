package evm

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// TestBLSPubkeyCache_PutDeleteLookup exercises the low-level mutation helpers
// without touching a chain adapter.
func TestBLSPubkeyCache_PutDeleteLookup(t *testing.T) {
	cache := NewBLSPubkeyCache(nil, common.Address{}, nil, 0)

	id := core.NodeID{0x01}
	good := make([]byte, BLSPubkeyCacheSize)
	for i := range good {
		good[i] = byte(i)
	}

	// Wrong size is silently dropped — a mis-decoded event must not poison
	// the cache.
	cache.Put(id, good[:10])
	if cache.Lookup(id) != nil {
		t.Fatal("Put accepted wrong-sized pubkey — cache may have been poisoned")
	}

	cache.Put(id, good)
	got := cache.Lookup(id)
	if len(got) != BLSPubkeyCacheSize {
		t.Fatalf("Lookup returned %d bytes, want %d", len(got), BLSPubkeyCacheSize)
	}
	for i, b := range good {
		if got[i] != b {
			t.Fatalf("byte %d mismatch: got %02x, want %02x", i, got[i], b)
		}
	}

	cache.Delete(id)
	if cache.Lookup(id) != nil {
		t.Fatal("Lookup returned an entry after Delete")
	}
}

// TestBLSPubkeyCache_ApplyLogDecodesNodeActivated pins the event-payload layout
// the poller consumes. A regenerated binding with a re-ordered struct would
// silently flip bytes otherwise. Topic[2] is the indexed nodeId; data carries
// (collateral, vestedAt, blsPubkeyG2[4]) as 6 ABI slots after the 2026-05-16
// event slim-down.
func TestBLSPubkeyCache_ApplyLogDecodesNodeActivated(t *testing.T) {
	var nodeID [32]byte
	nodeID[0] = 0xAA
	nodeID[31] = 0xBB

	data := make([]byte, 6*32)
	// The first two slots (collateral, vestedAt) are ignored by the cache;
	// fill them with junk to prove the slice boundaries are correct.
	for j := 0; j < 64; j++ {
		data[j] = 0xEF
	}
	// Per-quadrant marker so a mis-aligned slice would corrupt the result.
	for i := 0; i < 4; i++ {
		marker := byte(0x10 + i)
		for j := 0; j < 32; j++ {
			data[64+i*32+j] = marker
		}
	}

	log := types.Log{
		Topics: []common.Hash{nodeActivatedEventSig, common.Hash{}, common.BytesToHash(nodeID[:]), common.Hash{}},
		Data:   data,
	}

	cache := NewBLSPubkeyCache(nil, common.Address{}, nil, 0)
	cache.applyLog(log)

	got := cache.Lookup(core.NodeID(nodeID))
	if len(got) != 128 {
		t.Fatalf("post-applyLog pubkey length = %d, want 128", len(got))
	}
	for i := 0; i < 4; i++ {
		want := byte(0x10 + i)
		for j := 0; j < 32; j++ {
			if got[i*32+j] != want {
				t.Fatalf("pubkey byte %d = %02x, want %02x (field %d) — NodeActivated event slice is misaligned",
					i*32+j, got[i*32+j], want, i)
			}
		}
	}
}

// TestBLSPubkeyCache_ApplyLogEvictsOnRelease pins that NodeReleased clears
// the entry so a subsequent Lookup returns nil — a post-slashing operator
// re-activating on a fresh nodeId must not inherit the old pubkey.
func TestBLSPubkeyCache_ApplyLogEvictsOnRelease(t *testing.T) {
	var nodeID [32]byte
	nodeID[0] = 0xCC

	cache := NewBLSPubkeyCache(nil, common.Address{}, nil, 0)
	cache.Put(core.NodeID(nodeID), make([]byte, BLSPubkeyCacheSize))
	if cache.Lookup(core.NodeID(nodeID)) == nil {
		t.Fatal("Put did not populate the cache")
	}

	log := types.Log{
		Topics: []common.Hash{nodeReleasedEventSig, common.Hash{}, common.BytesToHash(nodeID[:]), common.Hash{}},
		Data:   []byte{},
	}
	cache.applyLog(log)

	if cache.Lookup(core.NodeID(nodeID)) != nil {
		t.Fatal("NodeReleased did not evict the entry")
	}
}

// TestBLSPubkeyCache_G2Zero_FourFieldGuard pins invariant B2: `g2Zero` must
// check ALL four big.Int fields of the [4]*big.Int G2 pubkey. A regression
// that drops any single field check would silently treat a partial-zero point
// (e.g. a valid G2 with one field == 0 by coincidence) as "unlocked" and evict
// the cache entry, causing downstream BLS verify failures. Or worse: a non-zero
// point whose unchecked field happens to be zero passes the null-guard and
// falsely-present entries poison the cache.
//
// ADR-008 §cache lookup semantics requires the Registry-returned zero point
// (all four fields = 0) to be treated as "not locked"; anything else is a
// locked node and must populate the cache.
func TestBLSPubkeyCache_G2Zero_FourFieldGuard(t *testing.T) {
	// All-zero → treated as absent (the one true case where the predicate holds).
	allZero := [4]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}
	if !g2Zero(allZero) {
		t.Fatal("g2Zero(all-zero) = false, want true")
	}

	// Each of the 4 fields individually non-zero → must be treated as present.
	// A mutant that drops the Sign() check on any one field will see one of
	// these four sub-cases return true (= cache-evict) and fail.
	cases := []struct {
		name string
		p    [4]*big.Int
	}{
		{"[0] non-zero", [4]*big.Int{big.NewInt(1), big.NewInt(0), big.NewInt(0), big.NewInt(0)}},
		{"[1] non-zero", [4]*big.Int{big.NewInt(0), big.NewInt(1), big.NewInt(0), big.NewInt(0)}},
		{"[2] non-zero", [4]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(1), big.NewInt(0)}},
		{"[3] non-zero", [4]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(1)}},
	}
	for _, tc := range cases {
		if g2Zero(tc.p) {
			t.Fatalf("%s: g2Zero returned true; partial-zero must be treated as PRESENT (mutation-kill: dropping the Sign() check on that field would misclassify a live node as unlocked)", tc.name)
		}
	}
}

// stubBLSRegistry is a minimal BLSPubkeyCacheRegistry fake for watermark /
// backfill tests that don't need live block-number tracking.
type stubBLSRegistry struct {
	total    *big.Int
	ids      [][32]byte
	pubkeys  [][4]*big.Int
	idsErr   error
	totalErr error
}

func (s *stubBLSRegistry) TotalNodes(_ *bind.CallOpts) (*big.Int, error) {
	return s.total, s.totalErr
}
func (s *stubBLSRegistry) GetNodeIds(_ *bind.CallOpts, _ *big.Int, _ *big.Int) ([][32]byte, error) {
	return s.ids, s.idsErr
}
func (s *stubBLSRegistry) GetNodes(_ *bind.CallOpts, _ *big.Int, _ *big.Int) ([]NodeRecord, error) {
	records := make([]NodeRecord, len(s.ids))
	for i := range s.ids {
		records[i].BlsPubkeyG2 = s.pubkeys[i]
	}
	return records, nil
}
func (s *stubBLSRegistry) GetNodeById(_ *bind.CallOpts, id [32]byte) (NodeRecord, error) {
	for i, known := range s.ids {
		if known == id {
			return NodeRecord{BlsPubkeyG2: s.pubkeys[i]}, nil
		}
	}
	return NodeRecord{BlsPubkeyG2: [4]*big.Int{big.NewInt(0), big.NewInt(0), big.NewInt(0), big.NewInt(0)}}, nil
}

// TestBLSPubkeyCache_Watermark_NeverRegresses pins invariant B6: the cache
// watermark never goes backward across Backfill calls. If the empty-total
// branch of Backfill dropped the setWatermark call (or if setWatermark were
// accidentally a no-op), a second Backfill after events have been polled into
// the cache would leave the watermark advanced-then-frozen-then-rewound,
// causing Watch.pollOnce to re-fetch the same log range on every tick.
//
// The test uses a stub registry + nil client so it can control the watermark
// transition directly: simulate a prior Watch that advanced the cache's
// watermark, then call Backfill and assert the watermark is not rewound below
// the prior value.
func TestBLSPubkeyCache_Watermark_NeverRegresses(t *testing.T) {
	reg := &stubBLSRegistry{total: big.NewInt(0)}
	cache := NewBLSPubkeyCache(nil, common.Address{}, reg, 0)

	// Simulate some prior Watch-driven advance (e.g. after confirmed events).
	cache.setWatermark(42)
	if got := cache.Watermark(); got != 42 {
		t.Fatalf("pre-condition: Watermark = %d, want 42", got)
	}

	// Second Backfill, against an empty registry + nil client, takes the
	// `startBlock=0` path. Monotonicity must still hold — the watermark must
	// NEVER regress from 42.
	if err := cache.Backfill(context.Background()); err != nil {
		t.Fatalf("Backfill: %v", err)
	}

	if wm := cache.Watermark(); wm < 42 {
		t.Fatalf("watermark regressed after Backfill: got %d, want >=42 (Backfill unconditionally assigned a lower startBlock — would cause double-application of confirmed events in Watch.pollOnce on next tick)", wm)
	}
}

// TestBLSPubkeyCache_NodeIDForPubkey pins the reverse index used by off-chain
// verifiers (custody) to authorize block-attached pubkeys against the live
// Registry. Forward and reverse indices MUST stay in lockstep across Put,
// Delete, NodeLocked, NodeWithdrawn, and re-Put (key rotation).
func TestBLSPubkeyCache_NodeIDForPubkey(t *testing.T) {
	cache := NewBLSPubkeyCache(nil, common.Address{}, nil, 0)

	id := core.NodeID{0xAB, 0xCD}
	pk := make([]byte, BLSPubkeyCacheSize)
	for i := range pk {
		pk[i] = byte(i)
	}

	// Unknown pubkey before any Put.
	if _, ok := cache.NodeIDForPubkey(pk); ok {
		t.Fatal("NodeIDForPubkey returned true for never-seen pubkey")
	}

	// Wrong-size input is rejected without touching the lock or map.
	if _, ok := cache.NodeIDForPubkey(pk[:64]); ok {
		t.Fatal("NodeIDForPubkey accepted a 64-byte input")
	}

	cache.Put(id, pk)
	got, ok := cache.NodeIDForPubkey(pk)
	if !ok {
		t.Fatal("NodeIDForPubkey did not find pubkey after Put")
	}
	if got != id {
		t.Fatalf("reverse lookup mapped to wrong NodeID: got %x, want %x", got, id)
	}

	// Re-Put with a different pubkey for the same NodeID — typical key rotation
	// shape. The old reverse entry MUST be evicted so a stale pubkey can't
	// authorize a signature after the chain has moved on.
	rotated := make([]byte, BLSPubkeyCacheSize)
	for i := range rotated {
		rotated[i] = byte(0xFF - i)
	}
	cache.Put(id, rotated)
	if _, ok := cache.NodeIDForPubkey(pk); ok {
		t.Fatal("stale pubkey still reverse-resolves after key rotation; reverse index leaked")
	}
	if got, ok := cache.NodeIDForPubkey(rotated); !ok || got != id {
		t.Fatalf("rotated pubkey did not reverse-resolve to %x: got %x ok=%v", id, got, ok)
	}

	// Delete must clear both directions.
	cache.Delete(id)
	if _, ok := cache.NodeIDForPubkey(rotated); ok {
		t.Fatal("reverse index still resolves after Delete; withdrawn validator could still authorize signatures")
	}
	if cache.Lookup(id) != nil {
		t.Fatal("forward index still resolves after Delete")
	}
}

// TestBLSPubkeyCache_ReverseIndex_ViaEvents pins that the on-chain event
// path (applyLog) also maintains the reverse index. Both production write
// paths — Put for tests/cold-miss, applyLog for NodeLocked / NodeWithdrawn —
// must keep forward and reverse in sync, otherwise custody's authorization
// check becomes a partial view of the Registry depending on which path
// populated a given entry.
func TestBLSPubkeyCache_ReverseIndex_ViaEvents(t *testing.T) {
	cache := NewBLSPubkeyCache(nil, common.Address{}, nil, 0)

	var nodeID [32]byte
	nodeID[0] = 0x42

	data := make([]byte, 6*32)
	// Per-quadrant marker so the resulting pubkey is unique and predictable.
	for i := 0; i < 4; i++ {
		marker := byte(0x20 + i)
		for j := 0; j < 32; j++ {
			data[64+i*32+j] = marker
		}
	}

	lockLog := types.Log{
		Topics: []common.Hash{nodeActivatedEventSig, common.Hash{}, common.BytesToHash(nodeID[:]), common.Hash{}},
		Data:   data,
	}
	cache.applyLog(lockLog)

	pk := cache.Lookup(core.NodeID(nodeID))
	if len(pk) != BLSPubkeyCacheSize {
		t.Fatalf("Lookup after NodeActivated returned %d bytes, want %d", len(pk), BLSPubkeyCacheSize)
	}
	if got, ok := cache.NodeIDForPubkey(pk); !ok || got != core.NodeID(nodeID) {
		t.Fatalf("NodeActivated did not populate reverse index: got %x ok=%v", got, ok)
	}

	releaseLog := types.Log{
		Topics: []common.Hash{nodeReleasedEventSig, common.Hash{}, common.BytesToHash(nodeID[:]), common.Hash{}},
		Data:   []byte{},
	}
	cache.applyLog(releaseLog)

	if _, ok := cache.NodeIDForPubkey(pk); ok {
		t.Fatal("NodeReleased did not evict reverse index entry")
	}
}
