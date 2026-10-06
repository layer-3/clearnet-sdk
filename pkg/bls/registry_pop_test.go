package bls

import (
	"math/big"
	"strings"
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

// Produced and checked with clearnet/contracts/evm/test/RegistryPoPVector.t.sol,
// using Registry's BLS.hashToG1 and pairing precompile. Secret scalar is 1.
func TestRegistryProofOfPossessionSolidityVector(t *testing.T) {
	kp, err := UnmarshalHexKeyPair(strings.Repeat("0", 63) + "1")
	if err != nil {
		t.Fatal(err)
	}
	g1, g2 := G1ToCoords(kp.PublicG1), G2ToCoords(kp.PublicG2)
	digest, err := RegistryProofOfPossessionDigest(big.NewInt(31337), common.HexToAddress("0x1234"), common.HexToAddress("0xabcd"), g1, g2)
	if err != nil {
		t.Fatal(err)
	}
	if digest != common.HexToHash("0xa5045efd3de10fa5fe9bf8b7c02a8071fffa2dea06240c6240c85d75d17e60d0") {
		t.Fatal("Solidity digest mismatch")
	}
	signature, err := Sign(&kp.Secret, digest)
	if err != nil {
		t.Fatal(err)
	}
	coords := G1ToCoords(signature)
	if coords[0].String() != "8974614380519981623452246740517506603510422484796169621586027206287431264507" ||
		coords[1].String() != "10576346420417623748446328620750859518371172120710216662612533085755919458212" {
		t.Fatal("Solidity signature/hash-to-curve mismatch")
	}
	if valid, err := Verify(signature, kp.PublicG2, digest); err != nil || !valid {
		t.Fatalf("Solidity proof invalid: %v", err)
	}
}

func TestRegistryProofDigestRejectsInvalidInputs(t *testing.T) {
	kp := KeyPairFromSeed([]byte("Registry PoP vector"))
	g1, g2 := G1ToCoords(kp.PublicG1), G2ToCoords(kp.PublicG2)
	registry, operator := common.HexToAddress("0x1234"), common.HexToAddress("0xabcd")
	for _, chain := range []*big.Int{nil, big.NewInt(-1), new(big.Int), new(big.Int).Lsh(big.NewInt(1), 256)} {
		if _, err := RegistryProofOfPossessionDigest(chain, registry, operator, g1, g2); err == nil {
			t.Fatal("invalid chain accepted")
		}
	}
	for _, bad := range []*big.Int{nil, big.NewInt(-1), new(big.Int).Set(fieldP), new(big.Int).Add(fieldP, big.NewInt(1))} {
		for i := 0; i < 6; i++ {
			x, y := g1, g2
			if i < 2 {
				x[i] = bad
			} else {
				y[i-2] = bad
			}
			if _, err := RegistryProofOfPossessionDigest(big.NewInt(31337), registry, operator, x, y); err == nil {
				t.Fatal("invalid coordinate accepted")
			}
		}
	}
}
