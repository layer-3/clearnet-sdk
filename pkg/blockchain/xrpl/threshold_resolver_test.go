package xrpl

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

// TestFeeQuorumResolver covers the live-quorum fee hook: with no resolver the
// fee autofill uses the static construction-time threshold; with one set it uses
// the resolved live SignerQuorum (so a quorum-raising rotation pays a correctly
// sized fee without a fleet restart); and a resolver error propagates.
func TestFeeQuorumResolver(t *testing.T) {
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	signer := sign.NewKeySignerFromECDSA(k)

	f, err := NewWithdrawalFinalizer("http://127.0.0.1:1", "rVaultAddressNotARealAccount11111111", 5, signer, nil, NewAssetResolver(AssetResolverConfig{}))
	if err != nil {
		t.Fatalf("NewWithdrawalFinalizer: %v", err)
	}
	ctx := context.Background()

	// No resolver → static threshold.
	if q, err := f.feeQuorum(ctx); err != nil || q != 5 {
		t.Fatalf("default feeQuorum = (%d, %v), want (5, nil)", q, err)
	}

	// Resolver set → live quorum.
	f.SetThresholdResolver(func(context.Context) (int, error) { return 9, nil })
	if q, err := f.feeQuorum(ctx); err != nil || q != 9 {
		t.Fatalf("resolved feeQuorum = (%d, %v), want (9, nil)", q, err)
	}

	// Resolver error propagates.
	f.SetThresholdResolver(func(context.Context) (int, error) { return 0, errors.New("boom") })
	if _, err := f.feeQuorum(ctx); err == nil {
		t.Fatal("resolver error was swallowed")
	}
}
