package sol

import (
	"context"
	"fmt"
	"sync"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/token"
	"github.com/gagliardetto/solana-go/rpc"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
)

const nativeAssetAddress = "0"

type AssetResolver struct {
	client     *rpc.Client
	commitment rpc.CommitmentType

	mu    sync.RWMutex
	cache map[string]uint8
}

var _ blockchain.AssetResolver = (*AssetResolver)(nil)

func NewAssetResolver(rpcURL string, commitment rpc.CommitmentType) *AssetResolver {
	if commitment == "" {
		commitment = rpc.CommitmentFinalized
	}
	return &AssetResolver{
		client:     rpc.New(rpcURL),
		commitment: commitment,
		cache:      make(map[string]uint8),
	}
}

func (r *AssetResolver) ValidateAssetAddress(_ context.Context, assetAddress string) error {
	if assetAddress == nativeAssetAddress {
		return nil
	}
	if _, err := solana.PublicKeyFromBase58(assetAddress); err != nil {
		return fmt.Errorf("sol: asset address %q is not a base58 mint: %w", assetAddress, err)
	}
	return nil
}

func (r *AssetResolver) AssetDecimals(ctx context.Context, assetAddress string) (uint8, error) {
	if err := r.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return 0, err
	}
	if assetAddress == nativeAssetAddress {
		return 9, nil
	}
	r.mu.RLock()
	if decimals, ok := r.cache[assetAddress]; ok {
		r.mu.RUnlock()
		return decimals, nil
	}
	r.mu.RUnlock()

	mint, _ := solana.PublicKeyFromBase58(assetAddress)
	info, err := r.client.GetAccountInfoWithOpts(ctx, mint, &rpc.GetAccountInfoOpts{Commitment: r.commitment})
	if err != nil {
		return 0, fmt.Errorf("sol: read mint %s: %w", assetAddress, err)
	}
	if info == nil || info.Value == nil {
		return 0, fmt.Errorf("sol: mint %s not found", assetAddress)
	}
	data := info.Value.Data.GetBinary()
	if len(data) < token.MINT_SIZE {
		return 0, fmt.Errorf("sol: mint %s account too short: %d", assetAddress, len(data))
	}
	// TODO: Verify the mint account owner is the SPL Token program before
	// trusting mint account data.
	decimals := data[44]
	r.mu.Lock()
	r.cache[assetAddress] = decimals
	r.mu.Unlock()
	return decimals, nil
}
