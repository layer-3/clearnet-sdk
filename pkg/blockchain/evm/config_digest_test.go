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

func TestComputeConfigRegistryDigests_GoldenVectors(t *testing.T) {
	registry := common.HexToAddress("0x00000000000000000000000000000000000A11CE")
	issuerID := common.HexToAddress("0x0000000000000000000000000000000000000B0B")
	key := common.HexToHash("0x07855b46a623a8ecabac76ed697aa4e13631e3b6718c8a0d342860c13c30d2fc")
	checksum := common.HexToHash("0xb902dd46ff1895e2e3efbe277c6b8857ffc82d78a1f8a424cb69a98d1d793a78")
	nonce := big.NewInt(7)
	threshold := big.NewInt(3)
	keys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000001"),
		common.HexToAddress("0x0000000000000000000000000000000000000002"),
		common.HexToAddress("0x0000000000000000000000000000000000000003"),
	}
	newKeys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000004"),
		common.HexToAddress("0x0000000000000000000000000000000000000005"),
		common.HexToAddress("0x0000000000000000000000000000000000000006"),
	}

	tests := []struct {
		name string
		got  [32]byte
		want common.Hash
	}{
		{
			name: "registration",
			got:  ComputeConfigRegistryRegistrationDigest(31337, registry, keys, threshold),
			want: common.HexToHash("0x5b3d8e9ac53b97a28f70f5231cb17fa5034bd2e77fe1f5e256c66b790e068300"),
		},
		{
			name: "setConfig",
			got:  ComputeConfigRegistrySetConfigDigest(31337, registry, issuerID, key, checksum, nonce),
			want: common.HexToHash("0x4421090980e2549f86f38e4ea769c68c0142f0a1b3cd446971ff12241743c3b0"),
		},
		{
			name: "setConfigWithData",
			got:  ComputeConfigRegistrySetConfigWithDataDigest(31337, registry, issuerID, key, []byte("data"), nonce),
			want: common.HexToHash("0x8be03704cad22b4cdd16d94e0f555ebfc1799059ae2984c5ae706323dde1c560"),
		},
		{
			name: "updateIssuerSettings",
			got:  ComputeConfigRegistryUpdateIssuerSettingsDigest(31337, registry, issuerID, newKeys, threshold, nonce),
			want: common.HexToHash("0x7ac8089411d6fdf6a9de709a227cde395af11729cb64aff0722dce7878941d42"),
		},
	}
	for _, tt := range tests {
		if common.Hash(tt.got) != tt.want {
			t.Fatalf("%s digest mismatch\nwant %s\ngot  %s", tt.name, tt.want.Hex(), common.Hash(tt.got).Hex())
		}
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

func TestConfigRegistryDigests_InputsDifferentiate(t *testing.T) {
	registry := common.HexToAddress("0x000000000000000000000000000000000000beef")
	issuerID := common.HexToAddress("0x000000000000000000000000000000000000b0b0")
	keys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000010"),
		common.HexToAddress("0x0000000000000000000000000000000000000020"),
		common.HexToAddress("0x0000000000000000000000000000000000000030"),
	}
	otherKeys := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000010"),
		common.HexToAddress("0x0000000000000000000000000000000000000020"),
		common.HexToAddress("0x0000000000000000000000000000000000000040"),
	}
	threshold := big.NewInt(2)
	nonce := big.NewInt(0)
	var key, checksum [32]byte
	key[31] = 0x10
	checksum[31] = 0x20

	reg0 := ComputeConfigRegistryRegistrationDigest(1, registry, keys, threshold)
	regVariants := map[string][32]byte{
		"chainID":   ComputeConfigRegistryRegistrationDigest(2, registry, keys, threshold),
		"registry":  ComputeConfigRegistryRegistrationDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), keys, threshold),
		"keys":      ComputeConfigRegistryRegistrationDigest(1, registry, otherKeys, threshold),
		"threshold": ComputeConfigRegistryRegistrationDigest(1, registry, keys, big.NewInt(3)),
	}
	for name, d := range regVariants {
		if d == reg0 {
			t.Errorf("registry registration digest unchanged when %s changed", name)
		}
	}

	set0 := ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, key, checksum, nonce)
	setVariants := map[string][32]byte{
		"chainID":  ComputeConfigRegistrySetConfigDigest(2, registry, issuerID, key, checksum, nonce),
		"registry": ComputeConfigRegistrySetConfigDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), issuerID, key, checksum, nonce),
		"issuerID": ComputeConfigRegistrySetConfigDigest(1, registry, common.HexToAddress("0x000000000000000000000000000000000000b0b1"), key, checksum, nonce),
		"key":      ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, [32]byte{0x11}, checksum, nonce),
		"checksum": ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, key, [32]byte{0x21}, nonce),
		"nonce":    ComputeConfigRegistrySetConfigDigest(1, registry, issuerID, key, checksum, big.NewInt(1)),
	}
	for name, d := range setVariants {
		if d == set0 {
			t.Errorf("registry setConfig digest unchanged when %s changed", name)
		}
	}

	data0 := ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, key, []byte("data"), nonce)
	dataVariants := map[string][32]byte{
		"chainID":  ComputeConfigRegistrySetConfigWithDataDigest(2, registry, issuerID, key, []byte("data"), nonce),
		"registry": ComputeConfigRegistrySetConfigWithDataDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), issuerID, key, []byte("data"), nonce),
		"issuerID": ComputeConfigRegistrySetConfigWithDataDigest(1, registry, common.HexToAddress("0x000000000000000000000000000000000000b0b1"), key, []byte("data"), nonce),
		"key":      ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, [32]byte{0x11}, []byte("data"), nonce),
		"data":     ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, key, []byte("other"), nonce),
		"nonce":    ComputeConfigRegistrySetConfigWithDataDigest(1, registry, issuerID, key, []byte("data"), big.NewInt(1)),
	}
	for name, d := range dataVariants {
		if d == data0 {
			t.Errorf("registry setConfigWithData digest unchanged when %s changed", name)
		}
	}

	update0 := ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, otherKeys, threshold, nonce)
	updateVariants := map[string][32]byte{
		"chainID":      ComputeConfigRegistryUpdateIssuerSettingsDigest(2, registry, issuerID, otherKeys, threshold, nonce),
		"registry":     ComputeConfigRegistryUpdateIssuerSettingsDigest(1, common.HexToAddress("0x000000000000000000000000000000000000bee0"), issuerID, otherKeys, threshold, nonce),
		"issuerID":     ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, common.HexToAddress("0x000000000000000000000000000000000000b0b1"), otherKeys, threshold, nonce),
		"newKeys":      ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, keys, threshold, nonce),
		"newThreshold": ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, otherKeys, big.NewInt(3), nonce),
		"nonce":        ComputeConfigRegistryUpdateIssuerSettingsDigest(1, registry, issuerID, otherKeys, threshold, big.NewInt(1)),
	}
	for name, d := range updateVariants {
		if d == update0 {
			t.Errorf("registry updateIssuerSettings digest unchanged when %s changed", name)
		}
	}

	if set0 == data0 {
		t.Errorf("registry setConfig and setConfigWithData digests collide")
	}
	if reg0 == update0 {
		t.Errorf("registry registerIssuer and updateIssuerSettings digests collide")
	}
}
