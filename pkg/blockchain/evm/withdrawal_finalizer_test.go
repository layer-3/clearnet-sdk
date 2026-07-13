package evm

import (
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// TestPackedFromOp_RejectsMalformedAddress guards M3: common.HexToAddress
// silently zero-fills a malformed address, so packedFromOp must reject a
// recipient or asset that is not a well-formed hex address instead of signing a
// withdrawal to the wrong destination.
func TestPackedFromOp_RejectsMalformedAddress(t *testing.T) {
	var wid [32]byte
	addr := "0x" + strings.Repeat("ab", 20)
	asset := "0x" + strings.Repeat("cd", 20)
	assetURI := core.AssetURI("yellow://ynet/asset/custody/evm/1/" + asset)

	if _, err := packedFromOp(&core.WithdrawalOp{Recipient: addr, AssetURI: assetURI, Amount: decimal.NewFromInt(1)}, wid, 0); err != nil {
		t.Fatalf("valid op rejected: %v", err)
	}
	if _, err := packedFromOp(&core.WithdrawalOp{Recipient: "not-an-address", AssetURI: assetURI, Amount: decimal.NewFromInt(1)}, wid, 0); err == nil {
		t.Error("malformed recipient accepted")
	}
	if _, err := packedFromOp(&core.WithdrawalOp{Recipient: addr, AssetURI: "yellow://ynet/asset/custody/evm/1/0xzz", Amount: decimal.NewFromInt(1)}, wid, 0); err == nil {
		t.Error("malformed asset accepted")
	}
}
