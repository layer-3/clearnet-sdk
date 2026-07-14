package btc

import (
	"context"
	"fmt"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
)

const nativeAssetAddress = "0"

type AssetResolver struct{}

var _ blockchain.AssetResolver = AssetResolver{}

func NewAssetResolver() AssetResolver {
	return AssetResolver{}
}

func (AssetResolver) ValidateAssetAddress(_ context.Context, assetAddress string) error {
	if assetAddress != nativeAssetAddress {
		return fmt.Errorf("btc: only native BTC asset address %q is supported, got %q", nativeAssetAddress, assetAddress)
	}
	return nil
}

func (AssetResolver) AssetDecimals(ctx context.Context, assetAddress string) (uint8, error) {
	if err := (AssetResolver{}).ValidateAssetAddress(ctx, assetAddress); err != nil {
		return 0, err
	}
	return 8, nil
}
