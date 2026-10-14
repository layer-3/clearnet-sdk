package wallet

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/tyler-smith/go-bip39"
)

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
