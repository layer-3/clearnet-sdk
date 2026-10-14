package eip712_test

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/layer-3/clearnet-sdk/pkg/eip712"
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
