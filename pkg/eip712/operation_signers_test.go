package eip712_test

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func TestOperationSignersMatchSDKVerifiers(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := sign.NewKeySignerFromECDSA(key)
	defer signer.Close()
	from := crypto.PubkeyToAddress(key.PublicKey)
	domain := eip712.NetworkChainID("YDEV")
	ctx := context.Background()

	swapSig, err := eip712.SignSwap(ctx, signer, domain, "ETH", "YELLOW", big.NewInt(100), big.NewInt(90), big.NewInt(0), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(swapSig) != 65 || swapSig[64] < 27 || swapSig[64] > 28 {
		t.Fatalf("unexpected swap signature V: %x", swapSig)
	}
	recovered, err := eip712.RecoverSwap(domain, "ETH", "YELLOW", big.NewInt(100), big.NewInt(90), big.NewInt(0), 7, swapSig)
	if err != nil || recovered != from {
		t.Fatalf("swap recovered=%s, err=%v", recovered, err)
	}

	recipient := common.HexToAddress("0x00000000000000000000000000000000000000bb")
	withdrawSig, err := eip712.SignWithdrawal(ctx, signer, domain, "YELLOW", big.NewInt(500), 11155111, recipient, big.NewInt(20), 8)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err = eip712.RecoverWithdrawal(domain, "YELLOW", big.NewInt(500), 11155111, recipient, big.NewInt(20), 8, withdrawSig)
	if err != nil || recovered != from {
		t.Fatalf("withdrawal recovered=%s, err=%v", recovered, err)
	}
	if _, err := eip712.SignWithdrawal(ctx, signer, domain, "YELLOW", big.NewInt(500), 0, recipient, big.NewInt(20), 8); err == nil {
		t.Fatal("zero withdrawal destination chain must be rejected")
	}
}
