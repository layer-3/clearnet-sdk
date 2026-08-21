package evm

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

type fakeConfigRegistryEventStore struct {
	events  []core.ConfigRegistryEvent
	cursors []core.ConfigRegistryCursor

	eventErr  error
	cursorErr error
}

func (s *fakeConfigRegistryEventStore) StoreConfigRegistryEvent(_ context.Context, ev core.ConfigRegistryEvent) error {
	if s.eventErr != nil {
		return s.eventErr
	}
	s.events = append(s.events, ev)
	return nil
}

func (s *fakeConfigRegistryEventStore) StoreConfigRegistryCursor(_ context.Context, cursor core.ConfigRegistryCursor) error {
	if s.cursorErr != nil {
		return s.cursorErr
	}
	s.cursors = append(s.cursors, cursor)
	return nil
}

func TestConfigRegistryEventForwarder_RejectsNilStore(t *testing.T) {
	if _, err := NewConfigRegistryEventForwarder(nil, nil, nil); err == nil {
		t.Fatal("expected nil store error")
	}
}

func TestConfigRegistryEventForwarder_EmptyFiltersForwardAll(t *testing.T) {
	store := &fakeConfigRegistryEventStore{}
	f, err := NewConfigRegistryEventForwarder(nil, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	ev := sampleConfigRegistryEvent()
	if err := f.HandleConfigRegistryEvent(context.Background(), ev); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(store.events) != 1 || store.events[0].IssuerID != ev.IssuerID || store.events[0].Key != ev.Key {
		t.Fatalf("events: %+v", store.events)
	}
	if len(store.cursors) != 0 {
		t.Fatalf("unexpected cursor store: %+v", store.cursors)
	}
}

func TestConfigRegistryEventForwarder_FiltersIssuerAndKey(t *testing.T) {
	ev := sampleConfigRegistryEvent()
	store := &fakeConfigRegistryEventStore{}
	f, err := NewConfigRegistryEventForwarder([]common.Address{ev.IssuerID}, [][32]byte{ev.Key}, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.HandleConfigRegistryEvent(context.Background(), ev); err != nil {
		t.Fatalf("handle matching: %v", err)
	}
	if len(store.events) != 1 {
		t.Fatalf("matching event was not stored")
	}

	wrongIssuer := ev
	wrongIssuer.IssuerID = common.HexToAddress("0x0000000000000000000000000000000000009999")
	if err := f.HandleConfigRegistryEvent(context.Background(), wrongIssuer); err != nil {
		t.Fatalf("handle wrong issuer: %v", err)
	}
	wrongKey := ev
	wrongKey.Key[0] ^= 0xFF
	if err := f.HandleConfigRegistryEvent(context.Background(), wrongKey); err != nil {
		t.Fatalf("handle wrong key: %v", err)
	}

	if len(store.events) != 1 {
		t.Fatalf("filtered events should not be stored as events: %d", len(store.events))
	}
	if len(store.cursors) != 2 {
		t.Fatalf("filtered events should store cursors: %d", len(store.cursors))
	}
	if store.cursors[0].BlockNumber != wrongIssuer.BlockNumber || store.cursors[0].LogIndex != wrongIssuer.LogIndex {
		t.Fatalf("wrong issuer cursor mismatch: %+v", store.cursors[0])
	}
}

func TestConfigRegistryEventForwarder_FilterWildcardsByDimension(t *testing.T) {
	ev := sampleConfigRegistryEvent()

	issuerOnlyStore := &fakeConfigRegistryEventStore{}
	issuerOnly, err := NewConfigRegistryEventForwarder([]common.Address{ev.IssuerID}, nil, issuerOnlyStore)
	if err != nil {
		t.Fatal(err)
	}
	if err := issuerOnly.HandleConfigRegistryEvent(context.Background(), ev); err != nil {
		t.Fatalf("issuer-only handle: %v", err)
	}
	if len(issuerOnlyStore.events) != 1 {
		t.Fatal("issuer-only wildcard key filter did not forward")
	}

	keyOnlyStore := &fakeConfigRegistryEventStore{}
	keyOnly, err := NewConfigRegistryEventForwarder(nil, [][32]byte{ev.Key}, keyOnlyStore)
	if err != nil {
		t.Fatal(err)
	}
	if err := keyOnly.HandleConfigRegistryEvent(context.Background(), ev); err != nil {
		t.Fatalf("key-only handle: %v", err)
	}
	if len(keyOnlyStore.events) != 1 {
		t.Fatal("key-only wildcard issuer filter did not forward")
	}
}

func TestConfigRegistryEventForwarder_PropagatesStoreErrors(t *testing.T) {
	ev := sampleConfigRegistryEvent()
	eventErr := fmt.Errorf("event store failed")
	store := &fakeConfigRegistryEventStore{eventErr: eventErr}
	f, err := NewConfigRegistryEventForwarder(nil, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.HandleConfigRegistryEvent(context.Background(), ev); err != eventErr {
		t.Fatalf("event error: got %v want %v", err, eventErr)
	}

	cursorErr := fmt.Errorf("cursor store failed")
	store = &fakeConfigRegistryEventStore{cursorErr: cursorErr}
	f, err = NewConfigRegistryEventForwarder([]common.Address{common.HexToAddress("0x9999")}, nil, store)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.HandleConfigRegistryEvent(context.Background(), ev); err != cursorErr {
		t.Fatalf("cursor error: got %v want %v", err, cursorErr)
	}
}

func sampleConfigRegistryEvent() core.ConfigRegistryEvent {
	var key, checksum [32]byte
	key[31] = 0xA
	checksum[31] = 0xB
	return core.ConfigRegistryEvent{
		Registry:    common.HexToAddress("0x000000000000000000000000000000000000beef"),
		IssuerID:    common.HexToAddress("0x0000000000000000000000000000000000000001"),
		Key:         key,
		Checksum:    checksum,
		BlockNumber: 100,
		LogIndex:    7,
		TxHash:      common.HexToHash("0x1234"),
	}
}
