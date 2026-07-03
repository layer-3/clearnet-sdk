package sol

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"

	"github.com/gagliardetto/solana-go"

	"github.com/layer-3/clearnet-sdk/pkg/sign"
)

func mustEd25519Signer(t *testing.T) sign.Signer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen ed25519: %v", err)
	}
	s, err := sign.NewKeySignerFromEd25519(priv)
	if err != nil {
		t.Fatalf("ed25519 signer: %v", err)
	}
	return s
}

// TestBuildExecuteIx_SPLAccounts pins the account shape of the execute
// instruction: native SOL uses the program's base account list, while an SPL
// withdrawal appends the token remaining-accounts (token program, vault ATA,
// recipient ATA) the program expects. Devnet-free — buildExecuteIx is pure given
// the keys.
func TestBuildExecuteIx_SPLAccounts(t *testing.T) {
	programID := solana.MustPublicKeyFromBase58("98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg")
	f, err := NewWithdrawalFinalizer("http://127.0.0.1:8899", programID, mustEd25519Signer(t), mustEd25519Signer(t), Config{ChainID: 1})
	if err != nil {
		t.Fatalf("NewWithdrawalFinalizer: %v", err)
	}

	to := solana.NewWallet().PublicKey()
	mint := solana.NewWallet().PublicKey()
	var wid [32]byte
	wid[0], wid[31] = 0x5A, 0x01

	// Native: base account list, no token accounts.
	nativeIx, err := f.buildExecuteIx(to, solana.PublicKey{}, 100, wid, 2, 1700000000)
	if err != nil {
		t.Fatalf("native buildExecuteIx: %v", err)
	}
	base := len(nativeIx.Accounts())

	// SPL: base + 3 token remaining-accounts.
	splIx, err := f.buildExecuteIx(to, mint, 100, wid, 3, 1700000000)
	if err != nil {
		t.Fatalf("spl buildExecuteIx: %v", err)
	}
	splAccts := splIx.Accounts()
	if len(splAccts) != base+3 {
		t.Fatalf("SPL accounts = %d, want base+3 (%d)", len(splAccts), base+3)
	}

	vaultATA, _, err := solana.FindAssociatedTokenAddress(f.vaultPDA, mint)
	if err != nil {
		t.Fatal(err)
	}
	recipientATA, _, err := solana.FindAssociatedTokenAddress(to, mint)
	if err != nil {
		t.Fatal(err)
	}
	want := []solana.PublicKey{solana.TokenProgramID, vaultATA, recipientATA}
	for i, w := range want {
		if got := splAccts[base+i].PublicKey; got != w {
			t.Errorf("SPL remaining account %d = %s, want %s", i, got, w)
		}
	}
}
