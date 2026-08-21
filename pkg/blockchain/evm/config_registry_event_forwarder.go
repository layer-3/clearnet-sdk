package evm

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// ConfigRegistryEventStore is the persistence boundary behind
// ConfigRegistryEventForwarder. StoreConfigRegistryEvent should persist the
// event and its cursor atomically; StoreConfigRegistryCursor advances over
// filtered-out events that should not be replayed after restart.
type ConfigRegistryEventStore interface {
	StoreConfigRegistryEvent(ctx context.Context, ev core.ConfigRegistryEvent) error
	StoreConfigRegistryCursor(ctx context.Context, cursor core.ConfigRegistryCursor) error
}

// ConfigRegistryEventForwarder filters watcher events by issuer and key, then
// forwards relevant events to a store. Empty filter slices are wildcards.
type ConfigRegistryEventForwarder struct {
	issuers map[common.Address]struct{}
	keys    map[[32]byte]struct{}
	store   ConfigRegistryEventStore
}

var _ ConfigRegistryEventHandler = (*ConfigRegistryEventForwarder)(nil)

func NewConfigRegistryEventForwarder(issuers []common.Address, keys [][32]byte, store ConfigRegistryEventStore) (*ConfigRegistryEventForwarder, error) {
	if store == nil {
		return nil, fmt.Errorf("config registry event forwarder: nil store")
	}
	f := &ConfigRegistryEventForwarder{store: store}
	if len(issuers) > 0 {
		f.issuers = make(map[common.Address]struct{}, len(issuers))
		for _, issuer := range issuers {
			f.issuers[issuer] = struct{}{}
		}
	}
	if len(keys) > 0 {
		f.keys = make(map[[32]byte]struct{}, len(keys))
		for _, key := range keys {
			f.keys[key] = struct{}{}
		}
	}
	return f, nil
}

func (f *ConfigRegistryEventForwarder) HandleConfigRegistryEvent(ctx context.Context, ev core.ConfigRegistryEvent) error {
	if f.matches(ev) {
		return f.store.StoreConfigRegistryEvent(ctx, ev)
	}
	return f.store.StoreConfigRegistryCursor(ctx, core.ConfigRegistryCursor{
		Registry:    ev.Registry,
		BlockNumber: ev.BlockNumber,
		LogIndex:    ev.LogIndex,
		TxHash:      ev.TxHash,
	})
}

func (f *ConfigRegistryEventForwarder) matches(ev core.ConfigRegistryEvent) bool {
	if len(f.issuers) > 0 {
		if _, ok := f.issuers[ev.IssuerID]; !ok {
			return false
		}
	}
	if len(f.keys) > 0 {
		if _, ok := f.keys[ev.Key]; !ok {
			return false
		}
	}
	return true
}
