package btc

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

// validCompressedPubkey generates a real compressed secp256k1 public key for
// use as a well-formed fixture.
func validCompressedPubkey(t *testing.T) []byte {
	t.Helper()
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("gen key: %v", err)
	}
	pk := crypto.CompressPubkey(&k.PublicKey)
	if len(pk) != 33 {
		t.Fatalf("generated pubkey is %d bytes, want 33", len(pk))
	}
	return pk
}

func TestSortedPubkeys_ValidCompressedKey(t *testing.T) {
	pk := validCompressedPubkey(t)
	sorted, err := sortedPubkeys(1, [][]byte{pk})
	if err != nil {
		t.Fatalf("sortedPubkeys: unexpected error: %v", err)
	}
	if len(sorted) != 1 {
		t.Fatalf("sortedPubkeys: got %d keys, want 1", len(sorted))
	}
}

func TestSortedPubkeys_RejectsOffCurveGarbage(t *testing.T) {
	valid := validCompressedPubkey(t)
	garbage := make([]byte, 33)
	garbage[0] = 0x02 // valid prefix byte, but the remaining bytes are not a point on the curve
	for i := 1; i < 33; i++ {
		garbage[i] = byte(i)
	}

	_, err := sortedPubkeys(2, [][]byte{valid, garbage})
	if err == nil {
		t.Fatal("sortedPubkeys: expected error for off-curve garbage pubkey, got nil")
	}
}

func TestSortedPubkeys_RejectsWrongLength(t *testing.T) {
	_, err := sortedPubkeys(1, [][]byte{make([]byte, 32)})
	if err == nil {
		t.Fatal("sortedPubkeys: expected error for wrong-length pubkey, got nil")
	}
}
