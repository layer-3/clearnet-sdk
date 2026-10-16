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

func TestTransferOperationIDGolden(t *testing.T) {
	id, err := eip712.TransferOperationID(
		eip712.NetworkChainID("YDEV"),
		common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		"0x00000000000000000000000000000000000000bb",
		[]eip712.TransferAsset{{Asset: "YELLOW", Amount: big.NewInt(1000)}},
		big.NewInt(100), 7,
	)
	if err != nil {
		t.Fatal(err)
	}
	const want = "0x6584fcefdda0d7cd8a6e664361851647fcc942bec3d39ae0fc0329646451165f"
	if id != want {
		t.Fatalf("SDK operation ID = %s, want %s", id, want)
	}
}

func TestTransferURIRecipientGolden(t *testing.T) {
	const recipient = "yellow://ynet/user/0x00000000000000000000000000000000000000bb"
	id, err := eip712.TransferOperationID(
		eip712.NetworkChainID("YDEV"),
		common.HexToAddress("0x00000000000000000000000000000000000000aa"),
		recipient,
		[]eip712.TransferAsset{{Asset: "YELLOW", Amount: big.NewInt(1000)}},
		big.NewInt(100), 7,
	)
	if err != nil {
		t.Fatal(err)
	}
	const want = "0x01440c9c01d11dbc51566065cb6d27d89647992df4ea4a6d98ee6c865ad6bfb9"
	if id != want {
		t.Fatalf("URI operation ID = %s, want %s", id, want)
	}
	to, err := eip712.TransferRecipientAddress(recipient)
	if err != nil || to == common.HexToAddress("0x00000000000000000000000000000000000000bb") {
		t.Fatalf("URI to slot = %s, %v, want derived address", to, err)
	}
}

func TestSignTransferURIRecovers(t *testing.T) {
	private, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	signer := sign.NewKeySignerFromECDSA(private)
	defer signer.Close()
	to, err := eip712.TransferRecipientAddress("yellow://ynet/user/0x00000000000000000000000000000000000000bb")
	if err != nil {
		t.Fatal(err)
	}
	assets := []eip712.TransferAsset{{Asset: "YELLOW", Amount: big.NewInt(1000)}}
	chainID := eip712.NetworkChainID("YDEV")
	fee := big.NewInt(100)
	sig, id, err := eip712.SignTransfer(context.Background(), signer, chainID,
		"yellow://ynet/user/0x00000000000000000000000000000000000000bb", assets, fee, 7)
	if err != nil {
		t.Fatal(err)
	}
	address, err := sign.EthAddress(signer)
	if err != nil {
		t.Fatal(err)
	}
	recovered, err := eip712.RecoverTransfer(chainID, to, assets, fee, 7, sig)
	if err != nil || recovered != address {
		t.Fatalf("recovered %s, want %s: %v", recovered, address, err)
	}
	if !eip712.ValidTransferOperationID(id) {
		t.Fatalf("invalid operation ID %q", id)
	}
}
