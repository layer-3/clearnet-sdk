package bls

import (
	"encoding/hex"
	"errors"
	"math/big"
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// ---------------------------------------------------------------------------
// Key Generation
// ---------------------------------------------------------------------------

func TestGenerateKeyPair(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}
	if kp.Secret.IsZero() {
		t.Fatal("secret key should not be zero")
	}
	// Public keys must be on curve.
	if !kp.PublicG1.IsOnCurve() {
		t.Fatal("G1 public key not on curve")
	}
	if !kp.PublicG2.IsOnCurve() {
		t.Fatal("G2 public key not on curve")
	}
}

func TestGenerateKeyPairUnique(t *testing.T) {
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()
	if kp1.Secret.Equal(&kp2.Secret) {
		t.Fatal("two random key pairs should have different secrets")
	}
}

func TestKeyPairFromSeedDeterministic(t *testing.T) {
	seed := []byte("test-seed-12345")
	kp1 := KeyPairFromSeed(seed)
	kp2 := KeyPairFromSeed(seed)
	if !kp1.Secret.Equal(&kp2.Secret) {
		t.Fatal("same seed should produce same secret")
	}
	if kp1.PublicG1 != kp2.PublicG1 {
		t.Fatal("same seed should produce same G1 pubkey")
	}
}

func TestKeyPairFromSeedDifferent(t *testing.T) {
	kp1 := KeyPairFromSeed([]byte("seed-a"))
	kp2 := KeyPairFromSeed([]byte("seed-b"))
	if kp1.Secret.Equal(&kp2.Secret) {
		t.Fatal("different seeds should produce different secrets")
	}
}

// ---------------------------------------------------------------------------
// HashToG1
// ---------------------------------------------------------------------------

func TestHashToG1OnCurve(t *testing.T) {
	msg := crypto.Keccak256Hash([]byte("test message"))
	pt, err := HashToG1(msg)
	if err != nil {
		t.Fatalf("HashToG1: %v", err)
	}
	if !pt.IsOnCurve() {
		t.Fatal("hashed point not on curve")
	}
}

func TestHashToG1Deterministic(t *testing.T) {
	msg := crypto.Keccak256Hash([]byte("deterministic"))
	p1, _ := HashToG1(msg)
	p2, _ := HashToG1(msg)
	if p1 != p2 {
		t.Fatal("HashToG1 should be deterministic")
	}
}

func TestHashToG1DifferentMessages(t *testing.T) {
	m1 := crypto.Keccak256Hash([]byte("message-1"))
	m2 := crypto.Keccak256Hash([]byte("message-2"))
	p1, _ := HashToG1(m1)
	p2, _ := HashToG1(m2)
	if p1 == p2 {
		t.Fatal("different messages should hash to different points")
	}
}

// ---------------------------------------------------------------------------
// Sign / Verify
// ---------------------------------------------------------------------------

func TestSignAndVerify(t *testing.T) {
	kp, _ := GenerateKeyPair()
	msg := crypto.Keccak256Hash([]byte("hello world"))

	sig, err := Sign(&kp.Secret, msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !sig.IsOnCurve() {
		t.Fatal("signature point not on curve")
	}

	ok, err := Verify(sig, kp.PublicG2, msg)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !ok {
		t.Fatal("valid signature should verify")
	}
}

func TestVerifyWrongMessage(t *testing.T) {
	kp, _ := GenerateKeyPair()
	msg := crypto.Keccak256Hash([]byte("correct"))
	wrong := crypto.Keccak256Hash([]byte("wrong"))

	sig, _ := Sign(&kp.Secret, msg)
	ok, err := Verify(sig, kp.PublicG2, wrong)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Fatal("signature should not verify with wrong message")
	}
}

func TestVerifyWrongKey(t *testing.T) {
	kp1, _ := GenerateKeyPair()
	kp2, _ := GenerateKeyPair()
	msg := crypto.Keccak256Hash([]byte("test"))

	sig, _ := Sign(&kp1.Secret, msg)
	ok, err := Verify(sig, kp2.PublicG2, msg)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if ok {
		t.Fatal("signature should not verify with wrong public key")
	}
}

// ---------------------------------------------------------------------------
// Aggregation
// ---------------------------------------------------------------------------

func TestAggregateG1SinglePoint(t *testing.T) {
	kp, _ := GenerateKeyPair()
	msg := crypto.Keccak256Hash([]byte("aggregate-test"))
	sig, _ := Sign(&kp.Secret, msg)

	agg, err := AggregateG1([]bn254.G1Affine{sig})
	if err != nil {
		t.Fatalf("AggregateG1: %v", err)
	}
	if agg != sig {
		t.Fatal("aggregating a single signature should return the same signature")
	}
}

func TestAggregateAndVerifyMultipleSigners(t *testing.T) {
	msg := crypto.Keccak256Hash([]byte("multi-signer"))
	n := 5

	keys := make([]*KeyPair, n)
	sigs := make([]bn254.G1Affine, n)
	g2Pubs := make([]bn254.G2Affine, n)

	for i := 0; i < n; i++ {
		kp, _ := GenerateKeyPair()
		keys[i] = kp
		sig, _ := Sign(&kp.Secret, msg)
		sigs[i] = sig
		g2Pubs[i] = kp.PublicG2
	}

	aggSig, err := AggregateG1(sigs)
	if err != nil {
		t.Fatalf("AggregateG1: %v", err)
	}
	aggPub, err := AggregateG2(g2Pubs)
	if err != nil {
		t.Fatalf("AggregateG2: %v", err)
	}

	ok, verr := Verify(aggSig, aggPub, msg)
	if verr != nil {
		t.Fatalf("Verify: %v", verr)
	}
	if !ok {
		t.Fatal("aggregated signature should verify against aggregated public key")
	}
}

func TestAggregateG1Empty(t *testing.T) {
	_, err := AggregateG1(nil)
	if err == nil {
		t.Fatal("expected error for empty G1 aggregation")
	}
	if !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("expected ErrEmptyAggregation, got %v", err)
	}
}

func TestAggregateG2Empty(t *testing.T) {
	_, err := AggregateG2(nil)
	if err == nil {
		t.Fatal("expected error for empty G2 aggregation")
	}
	if !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("expected ErrEmptyAggregation, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Serialization
// ---------------------------------------------------------------------------

func TestSerializeDeserializeG1(t *testing.T) {
	kp, _ := GenerateKeyPair()
	msg := crypto.Keccak256Hash([]byte("serialize-test"))
	sig, _ := Sign(&kp.Secret, msg)

	serialized := SerializeG1(sig)
	if len(serialized) != 64 {
		t.Fatalf("expected 64 bytes, got %d", len(serialized))
	}

	deserialized, err := DeserializeG1(serialized)
	if err != nil {
		t.Fatalf("DeserializeG1: %v", err)
	}
	if deserialized != sig {
		t.Fatal("round-trip serialization should preserve the point")
	}
}

func TestDeserializeG1InvalidLength(t *testing.T) {
	_, err := DeserializeG1([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for invalid length")
	}
}

func TestG1ToCoords(t *testing.T) {
	kp, _ := GenerateKeyPair()
	coords := G1ToCoords(kp.PublicG1)
	if coords[0] == nil || coords[1] == nil {
		t.Fatal("coordinates should not be nil")
	}
	if coords[0].Sign() == 0 && coords[1].Sign() == 0 {
		t.Fatal("at least one coordinate should be non-zero for a valid key")
	}
}

func TestG2ToCoords(t *testing.T) {
	kp, _ := GenerateKeyPair()
	coords := G2ToCoords(kp.PublicG2)
	for i, c := range coords {
		if c == nil {
			t.Fatalf("G2 coordinate[%d] should not be nil", i)
		}
	}
}

// ---------------------------------------------------------------------------
// ComputePopHash
// ---------------------------------------------------------------------------

func TestComputePopHash(t *testing.T) {
	addr := common.HexToAddress("0xAbCdEf0123456789AbCdEf0123456789AbCdEf01")
	h := ComputePopHash(addr)
	expected := crypto.Keccak256Hash(addr.Bytes())
	if h != expected {
		t.Fatalf("ComputePopHash mismatch: got %x, want %x", h, expected)
	}
}

// TestComputePopHash_GoldenVectors pins B8's on-chain byte layout
// against literal expected hashes. Spec: ADR-008 §WS-1 +
// contracts/evm/src/Registry.sol:531 compute
// keccak256(abi.encodePacked(msg.sender)) — which for a single address
// is 20 raw address bytes, no 32-byte left-padding. A mutation from
// addr.Bytes() to abi.encode(addr) (32-byte padded), to
// []byte(addr.Hex()) (ASCII), or to keccak over addr+salt, all produce
// different 32-byte outputs that this test catches.
func TestComputePopHash_GoldenVectors(t *testing.T) {
	cases := []struct {
		addr   string
		expect string // hex, no 0x
	}{
		// keccak256 of 20 raw bytes 0x00…
		{"0x0000000000000000000000000000000000000000",
			"5380c7b7ae81a58eb98d9c78de4a1fd7fd9535fc953ed2be602daaa41767312a"},
		// keccak256 of 20 raw bytes 0x1234…5678
		{"0x1234567890abcdef1234567890abcdef12345678",
			"5f6174255b44b7ca652c5289d2546de65e4394eb6aa52a40045e01237736d023"},
		// keccak256 of the existing test address, as a literal this time.
		{"0xAbCdEf0123456789AbCdEf0123456789AbCdEf01",
			"90a01d80e0e12c0b6acd5e4a69eec3dafeb108be3340405e3264b567330b5ba0"},
	}
	for _, tc := range cases {
		addr := common.HexToAddress(tc.addr)
		got := ComputePopHash(addr)
		wantBytes, err := hex.DecodeString(tc.expect)
		if err != nil {
			t.Fatalf("bad fixture hex %s: %v", tc.expect, err)
		}
		var want [32]byte
		copy(want[:], wantBytes)
		if got != want {
			t.Fatalf("ComputePopHash(%s) layout drift:\n  got  %x\n  want %x\n"+
				"This indicates an abi.encode vs abi.encodePacked divergence from "+
				"Registry.sol:531 — coordinate any change with the on-chain verifier.",
				tc.addr, got, want)
		}
	}
}

func TestComputePopHashDifferentAddresses(t *testing.T) {
	h1 := ComputePopHash(common.HexToAddress("0x0000000000000000000000000000000000000001"))
	h2 := ComputePopHash(common.HexToAddress("0x0000000000000000000000000000000000000002"))
	if h1 == h2 {
		t.Fatal("different addresses should produce different PoP hashes")
	}
}

// ---------------------------------------------------------------------------
// EncodeSignatureForContract — round-trip
// ---------------------------------------------------------------------------

func TestEncodeSignatureForContractRoundTrip(t *testing.T) {
	kp, _ := GenerateKeyPair()
	msg := crypto.Keccak256Hash([]byte("contract-encoding"))
	sig, _ := Sign(&kp.Secret, msg)

	bitmask := big.NewInt(0xFF)
	encoded, err := EncodeSignatureForContract(bitmask, sig, kp.PublicG2)
	if err != nil {
		t.Fatalf("EncodeSignatureForContract: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("encoded signature should not be empty")
	}
	// 7 * 32 bytes = 224 bytes (1 uint256 + 2 uint256 + 4 uint256)
	if len(encoded) != 7*32 {
		t.Fatalf("expected %d bytes, got %d", 7*32, len(encoded))
	}
}

// ---------------------------------------------------------------------------
// End-to-end: sign, aggregate, verify — simulating a cluster
// ---------------------------------------------------------------------------

func TestClusterSignatureWorkflow(t *testing.T) {
	const clusterSize = 8
	msg := crypto.Keccak256Hash([]byte("cluster-test-withdrawal"))

	keys := make([]*KeyPair, clusterSize)
	sigs := make([]bn254.G1Affine, clusterSize)
	g2Pubs := make([]bn254.G2Affine, clusterSize)

	for i := 0; i < clusterSize; i++ {
		kp, _ := GenerateKeyPair()
		keys[i] = kp
		sig, err := Sign(&kp.Secret, msg)
		if err != nil {
			t.Fatalf("Sign[%d]: %v", i, err)
		}
		sigs[i] = sig
		g2Pubs[i] = kp.PublicG2

		// Each individual signature should verify.
		ok, err := Verify(sig, kp.PublicG2, msg)
		if err != nil {
			t.Fatalf("individual Verify[%d]: %v", i, err)
		}
		if !ok {
			t.Fatalf("individual signature[%d] failed to verify", i)
		}
	}

	// Aggregate all signatures and public keys.
	aggSig, err := AggregateG1(sigs)
	if err != nil {
		t.Fatalf("AggregateG1: %v", err)
	}
	aggPub, err := AggregateG2(g2Pubs)
	if err != nil {
		t.Fatalf("AggregateG2: %v", err)
	}

	ok, err := Verify(aggSig, aggPub, msg)
	if err != nil {
		t.Fatalf("aggregated Verify: %v", err)
	}
	if !ok {
		t.Fatal("aggregated cluster signature should verify")
	}

	// Verify with subset (threshold = 2/3 + 1 = 6).
	threshold := (clusterSize*2)/3 + 1
	subSigs, err := AggregateG1(sigs[:threshold])
	if err != nil {
		t.Fatalf("AggregateG1 subset: %v", err)
	}
	subPubs, err := AggregateG2(g2Pubs[:threshold])
	if err != nil {
		t.Fatalf("AggregateG2 subset: %v", err)
	}

	ok, err = Verify(subSigs, subPubs, msg)
	if err != nil {
		t.Fatalf("threshold Verify: %v", err)
	}
	if !ok {
		t.Fatal("threshold subset signature should verify")
	}
}
