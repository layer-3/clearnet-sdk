package wallet

import (
	"strconv"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

// TestDerivationPathMatchesIndexes keeps the documented path and the indexes
// DeriveKey actually walks in step.
func TestDerivationPathMatchesIndexes(t *testing.T) {
	segments := strings.Split(strings.TrimPrefix(DerivationPath, "m/"), "/")
	if len(segments) != len(derivationIndexes) {
		t.Fatalf("path %q has %d segments, %d indexes", DerivationPath, len(segments), len(derivationIndexes))
	}
	for i, segment := range segments {
		hardened := strings.HasSuffix(segment, "'")
		number, err := strconv.ParseUint(strings.TrimSuffix(segment, "'"), 10, 32)
		if err != nil {
			t.Fatalf("segment %q: %v", segment, err)
		}
		index := uint32(number)
		if hardened {
			index |= hdkeychain.HardenedKeyStart
		}
		if index != derivationIndexes[i] {
			t.Fatalf("segment %d = %d, DeriveKey walks %d", i, index, derivationIndexes[i])
		}
	}
}

func TestDerivationGoldens(t *testing.T) {
	for _, tt := range []struct{ words, address string }{
		{strings.Repeat("abandon ", 11) + "about", "0x9858EfFD232B4033E47d90003D41EC34EcaEda94"},
		{"test test test test test test test test test test test junk", "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"},
	} {
		key, err := DeriveKey(tt.words)
		if err != nil {
			t.Fatal(err)
		}
		if got := crypto.PubkeyToAddress(key.PublicKey).Hex(); got != tt.address {
			t.Fatalf("got %s want %s", got, tt.address)
		}
		key.D.SetInt64(0)
	}
	if _, err := DeriveKey(strings.Repeat("abandon ", 12)); err == nil {
		t.Fatal("invalid checksum accepted")
	}
}
func TestNewMnemonic(t *testing.T) {
	first, err := NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewMnemonic()
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Fields(first)) != 24 || !bip39.IsMnemonicValid(first) || first == second {
		t.Fatal("recovery words are not random valid 24-word mnemonics")
	}
}
