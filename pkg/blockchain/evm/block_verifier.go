package evm

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/layer-3/clearnet-sdk/pkg/bls"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/log"
)

// BlockVerifier authorizes block-attached validator pubkeys against the
// on-chain registry (via BLSPubkeyCache) and runs the aggregate BLS check
// from pkg/bls. The cache lifecycle (Backfill + Watch + NodeActivated/
// NodeReleased events + confirmation gate) is owned by BLSPubkeyCache
// upstream.
type BlockVerifier struct {
	cache                *BLSPubkeyCache
	registryAddr         common.Address
	logger               log.Logger
	requireNonEmptyCache bool
}

// BlockVerifierOption configures optional BlockVerifier behaviour.
type BlockVerifierOption func(*BlockVerifier)

// WithRequireNonEmptyCache makes Start fail when Backfill completes leaving
// the cache empty. OFF by default: an empty registry is a legitimate state
// for some deployments, so refusing to start is an issuer's operational
// policy rather than behaviour every issuer inherits.
func WithRequireNonEmptyCache() BlockVerifierOption {
	return func(v *BlockVerifier) {
		v.requireNonEmptyCache = true
	}
}

// NewBlockVerifier prepares the pubkey cache against a live client.
// Call Start(ctx) before VerifyBlockSignature.
func NewBlockVerifier(client *ethclient.Client, registryAddr common.Address, confirmations uint64, logger log.Logger, opts ...BlockVerifierOption) (*BlockVerifier, error) {
	if client == nil {
		return nil, errors.New("bls: anchor chain client required")
	}
	if registryAddr == (common.Address{}) {
		return nil, errors.New("bls: registry address required")
	}
	if logger == nil {
		logger = log.NewNoopLogger()
	}
	reg, err := NewClearnetRegistry(registryAddr, client)
	if err != nil {
		return nil, fmt.Errorf("bls: bind registry: %w", err)
	}
	cache := NewBLSPubkeyCache(client, registryAddr, reg, confirmations)
	v := &BlockVerifier{
		cache:        cache,
		registryAddr: registryAddr,
		logger:       logger.WithKV("component", "bls-verify"),
	}
	for _, opt := range opts {
		opt(v)
	}
	return v, nil
}

// Start seeds the cache (synchronous Backfill so callers fail fast on an
// unreachable registry) and launches Watch in the background.
func (v *BlockVerifier) Start(ctx context.Context) error {
	if err := v.cache.Backfill(ctx); err != nil {
		return fmt.Errorf("bls: initial backfill: %w", err)
	}
	if v.requireNonEmptyCache && v.cache.Size() == 0 {
		return fmt.Errorf("bls: pubkey cache empty after backfill for registry %s (no nodes registered on chain, or the registry address is misconfigured)", v.registryAddr)
	}
	go func() {
		if err := v.cache.Watch(ctx); err != nil {
			v.logger.Error("bls pubkey cache watch exited", "error", err)
		}
	}()
	v.logger.Info("bls pubkey cache seeded", "size", v.cache.Size(), "watermark", v.cache.Watermark())
	return nil
}

// Watermark returns the highest anchor-chain block fully processed by the
// registry cache.
func (v *BlockVerifier) Watermark() uint64 { return v.cache.Watermark() }

// VerifyBlockSignature authorizes the block's embedded validator pubkeys
// against the on-chain registry, then verifies the aggregate BLS threshold
// signature.
func (v *BlockVerifier) VerifyBlockSignature(block *core.Block) error {
	if block == nil {
		return errors.New("bls: nil block")
	}
	if err := v.authorizeValidators(block.Attestation.Validators); err != nil {
		return err
	}
	return bls.VerifyBlock(block, block.Attestation.Validators)
}

// authorizeValidators cross-checks every pubkey in `entries` against the
// on-chain NodeRegistry via the BLSPubkeyCache. The BLS pairing check alone
// proves only that the embedded pubkeys produced the signature — without
// this membership gate, a sealer could insert self-generated keys, sign with
// them, and forge a valid-looking block.
func (v *BlockVerifier) authorizeValidators(entries [][]byte) error {
	seen := make(map[string]int, len(entries))
	for i, pk := range entries {
		if len(pk) != BLSPubkeyCacheSize {
			return fmt.Errorf("bls: validator[%d] has wrong length %d (want %d)", i, len(pk), BLSPubkeyCacheSize)
		}
		key := string(pk)
		if first, duplicate := seen[key]; duplicate {
			return fmt.Errorf("bls: duplicate validator pubkey at indices %d and %d", first, i)
		}
		seen[key] = i
		if _, ok := v.cache.NodeIDForPubkey(pk); !ok {
			return fmt.Errorf("bls: validator[%d] pubkey not registered on chain", i)
		}
	}
	return nil
}
