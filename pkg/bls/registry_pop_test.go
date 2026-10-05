package bls

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestRegistryPossessionABIParityAndReplayBinding(t *testing.T) {
	kp := KeyPairFromSeed([]byte("Registry PoP vector"))
	chain := big.NewInt(31337)
	registry, operator := common.HexToAddress("0x1234"), common.HexToAddress("0xabcd")
	g1, g2 := G1ToCoords(kp.PublicG1), G2ToCoords(kp.PublicG2)
	got, err := RegistryProofOfPossessionDigest(chain, registry, operator, g1, g2)
	if err != nil {
		t.Fatal(err)
	}
	// Independent ABI layout: static arrays expand inline, no offsets/lengths.
	domain := crypto.Keccak256([]byte("CLEARNET_REGISTRY_BLS_POP_V1"))
	preimage := append([]byte(nil), domain...)
	for _, n := range append([]*big.Int{chain, new(big.Int).SetBytes(registry[:]), new(big.Int).SetBytes(operator[:])}, append(g1[:], g2[:]...)...) {
		preimage = append(preimage, n.FillBytes(make([]byte, 32))...)
	}
	if len(preimage) != 320 || got != crypto.Keccak256Hash(preimage) {
		t.Fatal("Solidity ABI preimage drift")
	}
	sig, err := Sign(&kp.Secret, got)
	if err != nil {
		t.Fatal(err)
	}
	for _, tuple := range []struct {
		chain              *big.Int
		registry, operator common.Address
	}{
		{big.NewInt(31338), registry, operator}, {chain, common.HexToAddress("0x5678"), operator}, {chain, registry, common.HexToAddress("0xefab")},
	} {
		digest, err := RegistryProofOfPossessionDigest(tuple.chain, tuple.registry, tuple.operator, g1, g2)
		if err != nil || digest == got {
			t.Fatalf("missing domain binding: %v", err)
		}
		if valid, err := Verify(sig, kp.PublicG2, digest); err == nil && valid {
			t.Fatal("replayed proof verified")
		}
	}
}
