package evm

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/log"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// emptyRegistryStub is a BLSPubkeyCacheRegistry that reports zero nodes, so
// Backfill takes the "totalNodes() == 0" early-return path and completes
// with a nil error while leaving the cache empty.
type emptyRegistryStub struct{}

func (emptyRegistryStub) TotalNodes(opts *bind.CallOpts) (*big.Int, error) {
	return big.NewInt(0), nil
}

func (emptyRegistryStub) GetNodeIds(opts *bind.CallOpts, offset, limit *big.Int) ([][32]byte, error) {
	return nil, nil
}

func (emptyRegistryStub) GetNodes(opts *bind.CallOpts, offset, limit *big.Int) ([]NodeRecord, error) {
	return nil, nil
}

func (emptyRegistryStub) GetNodeById(opts *bind.CallOpts, nodeId [32]byte) (NodeRecord, error) {
	return NodeRecord{}, nil
}

func newTestCache() *BLSPubkeyCache {
	return NewBLSPubkeyCache(nil, common.Address{}, nil, 0)
}

func newTestBlockVerifier(cache *BLSPubkeyCache, opts ...BlockVerifierOption) *BlockVerifier {
	v := &BlockVerifier{
		cache:  cache,
		logger: log.NewNoopLogger(),
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

// makePubkey returns a deterministic 128-byte pubkey for tests. The bytes are
// not cryptographically valid G2 points — authorizeValidators only cares
// about length + cache membership, so the test fixture stays simple.
func makePubkey(seed byte) []byte {
	pk := make([]byte, BLSPubkeyCacheSize)
	for i := range pk {
		pk[i] = seed ^ byte(i)
	}
	return pk
}

func TestNewBlockVerifier_NilLoggerUsesNoop(t *testing.T) {
	v, err := NewBlockVerifier(new(ethclient.Client), common.HexToAddress("0x1"), 0, nil)
	if err != nil {
		t.Fatalf("NewBlockVerifier: %v", err)
	}
	if _, ok := v.logger.(log.NoopLogger); !ok {
		t.Fatalf("logger type = %T, want log.NoopLogger", v.logger)
	}
}

// TestAuthorizeValidators_RegisteredPubkeyAccepted is the canonical
// production case: SigningCoordinator wrote a 128-byte G2 pubkey into
// Attestation.Validators, and that pubkey is currently locked on chain.
func TestAuthorizeValidators_RegisteredPubkeyAccepted(t *testing.T) {
	cache := newTestCache()
	pk := makePubkey(0x10)
	cache.Put(core.NodeID{0x01}, pk)

	v := newTestBlockVerifier(cache)
	if err := v.authorizeValidators([][]byte{pk}); err != nil {
		t.Fatalf("registered pubkey was rejected: %v", err)
	}
}

// TestAuthorizeValidators_DuplicateRegisteredPubkeyRejected pins ISS-054 at
// the registry boundary: a genuinely registered key still represents one
// validator and cannot occupy several roster positions.
func TestAuthorizeValidators_DuplicateRegisteredPubkeyRejected(t *testing.T) {
	cache := newTestCache()
	pk := makePubkey(0x15)
	cache.Put(core.NodeID{0x01}, pk)

	v := newTestBlockVerifier(cache)
	duplicate := append([]byte(nil), pk...)
	err := v.authorizeValidators([][]byte{pk, duplicate})
	if err == nil {
		t.Fatal("expected rejection when one registered pubkey occupies two validator positions")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("duplicate validator pubkey at indices 0 and 1")) {
		t.Fatalf("expected duplicate indices in error, got %v", err)
	}
}

// TestAuthorizeValidators_UnregisteredPubkeyRejected pins the security
// invariant: a pubkey the Registry has never seen must be rejected, even
// though it's well-formed 128 bytes.
func TestAuthorizeValidators_UnregisteredPubkeyRejected(t *testing.T) {
	v := newTestBlockVerifier(newTestCache())

	rogue := makePubkey(0x99)
	err := v.authorizeValidators([][]byte{rogue})
	if err == nil {
		t.Fatal("expected rejection: unregistered 128-byte pubkey passed through")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("not registered on chain")) {
		t.Fatalf("expected 'not registered on chain' error, got %v", err)
	}
}

// TestAuthorizeValidators_MixedRegisteredAndRogue ensures one rogue entry in
// the middle of an otherwise-valid set fails the whole batch — partial
// authorization must not be possible.
func TestAuthorizeValidators_MixedRegisteredAndRogue(t *testing.T) {
	cache := newTestCache()
	good := makePubkey(0x20)
	cache.Put(core.NodeID{0x02}, good)

	rogue := makePubkey(0x99)

	v := newTestBlockVerifier(cache)
	if err := v.authorizeValidators([][]byte{good, rogue, good}); err == nil {
		t.Fatal("expected rejection when any entry is unregistered")
	}
}

// TestAuthorizeValidators_RejectsWithdrawnPubkey simulates a validator that
// was once registered but has since been withdrawn (NodeWithdrawn event
// applied).
func TestAuthorizeValidators_RejectsWithdrawnPubkey(t *testing.T) {
	cache := newTestCache()
	id := core.NodeID{0x04}
	pk := makePubkey(0x40)
	cache.Put(id, pk)
	cache.Delete(id) // simulate NodeWithdrawn

	v := newTestBlockVerifier(cache)
	if err := v.authorizeValidators([][]byte{pk}); err == nil {
		t.Fatal("withdrawn validator pubkey was still accepted")
	}
}

// TestStart_EmptyRegistryFailsFast pins the fix for a silent outage: a
// registry that reports zero nodes (misconfigured registry address, or a
// genuinely empty registry) makes Backfill return nil while leaving the
// cache empty.
func TestStart_EmptyRegistryFailsFast(t *testing.T) {
	cache := NewBLSPubkeyCache(nil, common.Address{}, emptyRegistryStub{}, 0)
	v := newTestBlockVerifier(cache, WithRequireNonEmptyCache())

	err := v.Start(context.Background())
	if err == nil {
		t.Fatal("expected Start to fail fast on an empty pubkey cache")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("pubkey cache empty")) {
		t.Fatalf("expected 'pubkey cache empty' error, got %v", err)
	}
}

// TestAuthorizeValidators_RejectsNon128Length covers every wire shape that
// isn't the canonical 128-byte G2 pubkey — including the now-removed 32-byte
func TestStart_EmptyRegistryAllowedByDefault(t *testing.T) {
	cache := NewBLSPubkeyCache(nil, common.Address{}, emptyRegistryStub{}, 0)
	v := newTestBlockVerifier(cache)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := v.Start(ctx); err != nil {
		t.Fatalf("Start must succeed on an empty registry without WithRequireNonEmptyCache, got: %v", err)
	}
	if v.cache.Size() != 0 {
		t.Fatalf("cache size = %d, want 0", v.cache.Size())
	}
}

// NodeID legacy form. Only the serialized G2 pubkey ADR-008 §11 mandates is
// accepted.
func TestAuthorizeValidators_RejectsNon128Length(t *testing.T) {
	v := newTestBlockVerifier(newTestCache())

	for _, size := range []int{0, 32, 64, 127, 129, 256} {
		err := v.authorizeValidators([][]byte{make([]byte, size)})
		if err == nil {
			t.Fatalf("expected rejection of %d-byte validator entry", size)
		}
		if !bytes.Contains([]byte(err.Error()), []byte("wrong length")) {
			t.Fatalf("size=%d: expected 'wrong length' error, got %v", size, err)
		}
	}
}
