package evm

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
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

func TestWithdrawalFinalizerSignUsesAuthorizer(t *testing.T) {
	ctx := context.Background()
	authorizerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	authorizer := sign.NewKeySignerFromECDSA(authorizerKey)
	payer := sign.NewKeySignerFromECDSA(payerKey)
	authorizerAddr, err := sign.EthAddress(authorizer)
	if err != nil {
		t.Fatal(err)
	}
	payerAddr, err := sign.EthAddress(payer)
	if err != nil {
		t.Fatal(err)
	}
	if authorizerAddr == payerAddr {
		t.Fatal("test generated identical authorizer and payer")
	}

	f := &WithdrawalFinalizer{
		chainID:        1,
		vaultAddr:      common.HexToAddress("0x000000000000000000000000000000000000c057"),
		authorizer:     authorizer,
		authorizerAddr: authorizerAddr,
	}
	p := evmPacked{
		To:           common.HexToAddress("0x000000000000000000000000000000000000b0b0").Hex(),
		Asset:        common.Address{}.Hex(),
		Amount:       "1",
		WithdrawalID: strings.Repeat("11", 32),
		Deadline:     123,
	}
	packed := []byte(fmt.Sprintf(`{"to":%q,"asset":%q,"amount":%q,"withdrawalId":%q,"deadline":%d}`, p.To, p.Asset, p.Amount, p.WithdrawalID, p.Deadline))
	sig, err := f.Sign(ctx, packed)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := f.digest(p)
	if err != nil {
		t.Fatal(err)
	}
	pub, err := crypto.SigToPub(digest[:], sig)
	if err != nil {
		t.Fatal(err)
	}
	recovered := crypto.PubkeyToAddress(*pub)
	if recovered != authorizerAddr {
		t.Fatalf("signature recovered to %s, want authorizer %s", recovered, authorizerAddr)
	}
	if recovered == payerAddr {
		t.Fatalf("signature recovered to payer %s", payerAddr)
	}
}

func TestSignerTransactorFromUsesPayer(t *testing.T) {
	payerKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payer := sign.NewKeySignerFromECDSA(payerKey)
	want, err := sign.EthAddress(payer)
	if err != nil {
		t.Fatal(err)
	}
	txr, err := NewSignerTransactor(context.Background(), nil, payer)
	if err != nil {
		t.Fatal(err)
	}
	if txr.From() != want {
		t.Fatalf("From = %s, want %s", txr.From(), want)
	}
}
