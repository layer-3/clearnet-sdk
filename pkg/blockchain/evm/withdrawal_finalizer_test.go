package evm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

type testAssetResolver struct{}

func (testAssetResolver) ValidateAssetAddress(_ context.Context, assetAddress string) error {
	if assetAddress == nativeAssetAddress {
		return nil
	}
	if !common.IsHexAddress(assetAddress) || common.HexToAddress(assetAddress) == (common.Address{}) {
		return fmt.Errorf("bad asset")
	}
	return nil
}

func (testAssetResolver) AssetDecimals(context.Context, string) (uint8, error) {
	return 18, nil
}

var _ blockchain.AssetResolver = testAssetResolver{}

// TestPackedFromOp_RejectsMalformedAddress guards M3: common.HexToAddress
// silently zero-fills a malformed address, so packedFromOp must reject a
// recipient or asset that is not a well-formed hex address instead of signing a
// withdrawal to the wrong destination.
func TestPackedFromOp_RejectsMalformedAddress(t *testing.T) {
	var wid [32]byte
	addr := "0x" + strings.Repeat("ab", 20)
	assetURI := core.AssetURI("yellow://ynet/asset/custody/evm/1/0")
	f := &WithdrawalFinalizer{chainID: 1, assets: testAssetResolver{}}

	if _, err := f.packedFromOp(context.Background(), &core.WithdrawalOp{Recipient: addr, AssetURI: assetURI, Amount: decimal.NewFromInt(1)}, wid, 0); err != nil {
		t.Fatalf("valid op rejected: %v", err)
	}
	if _, err := f.packedFromOp(context.Background(), &core.WithdrawalOp{Recipient: "not-an-address", AssetURI: assetURI, Amount: decimal.NewFromInt(1)}, wid, 0); err == nil {
		t.Error("malformed recipient accepted")
	}
	if _, err := f.packedFromOp(context.Background(), &core.WithdrawalOp{Recipient: addr, AssetURI: "yellow://ynet/asset/custody/evm/1/0xzz", Amount: decimal.NewFromInt(1)}, wid, 0); err == nil {
		t.Error("malformed asset accepted")
	}
}
