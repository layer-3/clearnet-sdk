package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestComputeRotationDigest_GoldenVector pins the Go implementation to the
// Solidity contract. The golden digest was produced by Custody.sol's exact
// keccak256(abi.encode(...)) for these inputs, so a divergence here means the Go
// digest no longer matches what updateSigners will verify on chain.
func TestComputeRotationDigest_GoldenVector(t *testing.T) {
	chainID := uint64(31337)
	vault := common.HexToAddress("0x0000000000000000000000000000000000AbC123")
	signers := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	threshold := big.NewInt(2)
	nonce := big.NewInt(7)

	want := common.HexToHash("0x96fbf07188f867ef8a2f43996500d7926c85d189dcf8951a03808d864853a6cd")
	got := common.Hash(ComputeRotationDigest(chainID, vault, signers, threshold, nonce))
	if got != want {
		t.Fatalf("rotation digest mismatch\nwant %s\ngot  %s", want.Hex(), got.Hex())
	}
}

// TestComputeRotationDigest_InputsDifferentiate guards against a bug where the
// digest ignores one of its inputs (e.g. dropping signerNonce) that the golden
// alone could miss — every field must change the digest.
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
