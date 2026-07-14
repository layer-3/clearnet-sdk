package evm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
)

const nativeAssetAddress = "0"

type AssetResolverConfig struct {
	NativeDecimals *uint8
}

type AssetResolver struct {
	client         bind.ContractBackend
	nativeDecimals uint8

	mu    sync.RWMutex
	cache map[string]uint8
}

var _ blockchain.AssetResolver = (*AssetResolver)(nil)

func NewAssetResolver(client bind.ContractBackend, cfg AssetResolverConfig) *AssetResolver {
	if client == nil {
		panic("evm: asset resolver client is required")
	}
	nativeDecimals := uint8(18)
	if cfg.NativeDecimals != nil {
		nativeDecimals = *cfg.NativeDecimals
	}
	return &AssetResolver{
		client:         client,
		nativeDecimals: nativeDecimals,
		cache:          make(map[string]uint8),
	}
}

func (r *AssetResolver) ValidateAssetAddress(_ context.Context, assetAddress string) error {
	if assetAddress == nativeAssetAddress {
		return nil
	}
	if !common.IsHexAddress(assetAddress) {
		return fmt.Errorf("evm: asset address %q is not a valid hex address", assetAddress)
	}
	if common.HexToAddress(assetAddress) == (common.Address{}) {
		return fmt.Errorf("evm: zero hex address is not a valid asset address; use %q for native ETH", nativeAssetAddress)
	}
	return nil
}

func (r *AssetResolver) AssetDecimals(ctx context.Context, assetAddress string) (uint8, error) {
	if err := r.ValidateAssetAddress(ctx, assetAddress); err != nil {
		return 0, err
	}
	if assetAddress == nativeAssetAddress {
		return r.nativeDecimals, nil
	}
	key := strings.ToLower(common.HexToAddress(assetAddress).Hex())
	r.mu.RLock()
	if decimals, ok := r.cache[key]; ok {
		r.mu.RUnlock()
		return decimals, nil
	}
	r.mu.RUnlock()

	token, err := NewMockERC20(common.HexToAddress(assetAddress), r.client)
	if err != nil {
		return 0, fmt.Errorf("evm: bind token %s: %w", common.HexToAddress(assetAddress).Hex(), err)
	}
	decimals, err := token.Decimals(&bind.CallOpts{Context: ctx})
	if err != nil {
		return 0, fmt.Errorf("evm: token decimals %s: %w", common.HexToAddress(assetAddress).Hex(), err)
	}
	r.mu.Lock()
	r.cache[key] = decimals
	r.mu.Unlock()
	return decimals, nil
}
