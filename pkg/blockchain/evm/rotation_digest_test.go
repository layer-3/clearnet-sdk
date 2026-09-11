package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// Every rotation field remains bound by the EIP-712 authorization.
func TestComputeRotationDigest_InputsDifferentiate(t *testing.T) {
	base := struct {
		chainID   uint64
		vault     common.Address
		signers   []common.Address
		threshold *big.Int
		nonce     *big.Int
	}{
		chainID: 1,
		vault:   common.HexToAddress("0x000000000000000000000000000000000000beef"),
		signers: []common.Address{
			common.HexToAddress("0x0000000000000000000000000000000000000010"),
			common.HexToAddress("0x0000000000000000000000000000000000000020"),
			common.HexToAddress("0x0000000000000000000000000000000000000030"),
		},
		threshold: big.NewInt(2),
		nonce:     big.NewInt(0),
	}
	d0 := ComputeRotationDigest(base.chainID, base.vault, base.signers, base.threshold, base.nonce)

	variants := map[string][32]byte{
		"chainID":   ComputeRotationDigest(base.chainID+1, base.vault, base.signers, base.threshold, base.nonce),
		"vault":     ComputeRotationDigest(base.chainID, common.HexToAddress("0x000000000000000000000000000000000000bee0"), base.signers, base.threshold, base.nonce),
		"threshold": ComputeRotationDigest(base.chainID, base.vault, base.signers, big.NewInt(3), base.nonce),
		"nonce":     ComputeRotationDigest(base.chainID, base.vault, base.signers, base.threshold, big.NewInt(1)),
	}
	signersChanged := make([]common.Address, len(base.signers))
	copy(signersChanged, base.signers)
	signersChanged[0] = common.HexToAddress("0x0000000000000000000000000000000000000099")
	variants["signers"] = ComputeRotationDigest(base.chainID, base.vault, signersChanged, base.threshold, base.nonce)

	for name, d := range variants {
		if d == d0 {
			t.Errorf("digest unchanged when %s changed — input not committed", name)
		}
	}
}
