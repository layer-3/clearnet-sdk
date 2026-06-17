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

	if _, err := packedFromOp(&core.WithdrawalOp{Recipient: addr, L1Asset: asset, Amount: decimal.NewFromInt(1)}, wid); err != nil {
		t.Fatalf("valid op rejected: %v", err)
	}
	if _, err := packedFromOp(&core.WithdrawalOp{Recipient: "not-an-address", L1Asset: asset, Amount: decimal.NewFromInt(1)}, wid); err == nil {
		t.Error("malformed recipient accepted")
	}
	if _, err := packedFromOp(&core.WithdrawalOp{Recipient: addr, L1Asset: "0xzz", Amount: decimal.NewFromInt(1)}, wid); err == nil {
		t.Error("malformed asset accepted")
	}
}
