package evm

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
)

// fakeConfigReader is a backfill-only ConfigWatcherReader (client nil, so Watch
// is a no-op). FilterConfigSet is never exercised by these tests.
type fakeConfigReader struct {
	checksums map[[32]byte][32]byte
	epochs    map[[32]byte]uint64
}

func (f fakeConfigReader) LatestConfigChecksum(_ *bind.CallOpts, key [32]byte) ([32]byte, error) {
	return f.checksums[key], nil
}

func (f fakeConfigReader) ConfigEpoch(_ *bind.CallOpts, key [32]byte) (uint64, error) {
	return f.epochs[key], nil
}

func (f fakeConfigReader) FilterConfigSet(_ *bind.FilterOpts, _ [][32]byte, _ []common.Address) (*ConfigConfigSetIterator, error) {
	return nil, nil
}

func TestConfigWatcher_BackfillAndLookup(t *testing.T) {
	var kConfig, kSigners [32]byte
	kConfig[0] = 0x01
	kSigners[0] = 0x02

	var csum [32]byte
	csum[31] = 0xAB

	reg := fakeConfigReader{
		checksums: map[[32]byte][32]byte{kConfig: csum},        // kSigners not committed yet
		epochs:    map[[32]byte]uint64{kConfig: 3, kSigners: 0}, // genesis for signers
	}
	w, err := NewConfigWatcher(nil, reg, [][32]byte{kConfig, kSigners}, 12)
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	if err := w.Backfill(context.Background()); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	st, ok := w.Lookup(kConfig)
	if !ok || st.Epoch != 3 || st.Checksum != csum {
		t.Fatalf("kConfig lookup: ok=%v epoch=%d checksum=%x", ok, st.Epoch, st.Checksum)
	}
	if c, ok := w.LatestChecksum(kConfig); !ok || c != csum {
		t.Fatalf("kConfig LatestChecksum: ok=%v c=%x", ok, c)
	}

	// kSigners is known (watched) but has a zero checksum (genesis) — Lookup
	// reports it, but LatestChecksum treats zero as not-ready so the signer
	// source doesn't load an empty set.
	st, ok = w.Lookup(kSigners)
	if !ok || st.Epoch != 0 || st.Checksum != ([32]byte{}) {
		t.Fatalf("kSigners lookup: ok=%v epoch=%d checksum=%x", ok, st.Epoch, st.Checksum)
	}
	if _, ok := w.LatestChecksum(kSigners); ok {
		t.Fatalf("kSigners LatestChecksum should be not-ready at genesis")
	}

	// Unwatched key is unknown.
	var other [32]byte
	other[0] = 0xFF
	if _, ok := w.Lookup(other); ok {
		t.Fatalf("unwatched key should be unknown")
	}
}

// TestConfigWatcher_RejectsEmptyKeys guards against the eth_getLogs wildcard:
// an empty key set must fail at construction rather than match every event.
func TestConfigWatcher_RejectsEmptyKeys(t *testing.T) {
	if _, err := NewConfigWatcher(nil, fakeConfigReader{}, nil, 0); err == nil {
		t.Fatal("expected error for empty key set")
	}
}

// TestConfigWatcher_OnChangeWiring is a light check that SetOnChange stores the
// callback (full event-driven coverage needs a chain and lives in integration).
func TestConfigWatcher_OnChangeWiring(t *testing.T) {
	w, err := NewConfigWatcher(nil, fakeConfigReader{}, [][32]byte{{0x01}}, 0)
	if err != nil {
		t.Fatalf("new watcher: %v", err)
	}
	called := false
	w.SetOnChange(func(_ [32]byte, _ ConfigState) { called = true })
	if w.onChange == nil {
		t.Fatal("onChange not set")
	}
	w.onChange([32]byte{}, ConfigState{})
	if !called {
		t.Fatal("callback not invoked")
	}
}
