package evm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestComputeConfigCommitDigest_GoldenVector pins the Go implementation to
// ConfigGovernor.sol's setConfig digest. The golden was produced by Solidity's
// keccak256(abi.encode(chainid, governor, "setConfig", key, checksum,
// expectedEpoch)) (via `cast`), so a divergence here means the Go digest no
// longer matches what setConfig verifies on chain.
func TestComputeConfigCommitDigest_GoldenVector(t *testing.T) {
	governor := common.HexToAddress("0x0000000000000000000000000000000000AbC123")
	var key, checksum [32]byte
	key[31] = 1
	checksum[31] = 2

	want := common.HexToHash("0xefae1aab464c2b8d70dffb732adf18318077f93e076716b657fbc31f814f481e")
	got := common.Hash(ComputeConfigCommitDigest(31337, governor, key, checksum, 7))
	if got != want {
		t.Fatalf("config commit digest mismatch\nwant %s\ngot  %s", want.Hex(), got.Hex())
	}
}

// TestComputeOperatorRotationDigest_GoldenVector pins the Go implementation to
// ConfigGovernor.sol's updateOperators digest (inner = keccak256(abi.encode(
// newOperators, newThreshold)), outer adds chainid + governor + tag + nonce).
func TestComputeOperatorRotationDigest_GoldenVector(t *testing.T) {
	governor := common.HexToAddress("0x0000000000000000000000000000000000AbC123")
	ops := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}

	want := common.HexToHash("0xf7f135ff7e2ef79c198cd6c4ed29a2e5972fa584ab00e76b0b0ce6ebbfa3287b")
	got := common.Hash(ComputeOperatorRotationDigest(31337, governor, ops, big.NewInt(2), big.NewInt(5)))
	if got != want {
		t.Fatalf("operator rotation digest mismatch\nwant %s\ngot  %s", want.Hex(), got.Hex())
	}
}

// TestConfigDigests_InputsDifferentiate guards against a digest that ignores one
// of its inputs — every field must change the result, and the two operations
// must never collide for the same arguments.
func TestConfigDigests_InputsDifferentiate(t *testing.T) {
	gov := common.HexToAddress("0x000000000000000000000000000000000000beef")
	var key, checksum [32]byte
	key[31] = 0x10
	checksum[31] = 0x20

	d0 := ComputeConfigCommitDigest(1, gov, key, checksum, 0)
	variants := map[string][32]byte{
		"chainID":  ComputeConfigCommitDigest(2, gov, key, checksum, 0),
		"governor": ComputeConfigCommitDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), key, checksum, 0),
		"key":      ComputeConfigCommitDigest(1, gov, [32]byte{0x11}, checksum, 0),
		"checksum": ComputeConfigCommitDigest(1, gov, key, [32]byte{0x21}, 0),
		"epoch":    ComputeConfigCommitDigest(1, gov, key, checksum, 1),
	}
	for name, d := range variants {
		if d == d0 {
			t.Errorf("config commit digest unchanged when %s changed", name)
		}
	}

	ops := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000010"),
		common.HexToAddress("0x0000000000000000000000000000000000000020"),
		common.HexToAddress("0x0000000000000000000000000000000000000030"),
	}
	o0 := ComputeOperatorRotationDigest(1, gov, ops, big.NewInt(2), big.NewInt(0))
	opVariants := map[string][32]byte{
		"chainID":   ComputeOperatorRotationDigest(2, gov, ops, big.NewInt(2), big.NewInt(0)),
		"governor":  ComputeOperatorRotationDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), ops, big.NewInt(2), big.NewInt(0)),
		"threshold": ComputeOperatorRotationDigest(1, gov, ops, big.NewInt(3), big.NewInt(0)),
		"nonce":     ComputeOperatorRotationDigest(1, gov, ops, big.NewInt(2), big.NewInt(1)),
	}
	for name, d := range opVariants {
		if d == o0 {
			t.Errorf("operator rotation digest unchanged when %s changed", name)
		}
	}

	// Cross-operation separation: same chain/governor must not collide.
	if d0 == o0 {
		t.Errorf("setConfig and updateOperators digests collide")
	}
}
