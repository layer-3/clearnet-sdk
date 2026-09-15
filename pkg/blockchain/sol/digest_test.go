package sol

import (
	"encoding/hex"
	"testing"

	"github.com/gagliardetto/solana-go"
)

// TestDigestVectors pins the withdrawal digest to byte-exact cross-implementation
// vectors. The same inputs and expected hex are asserted by the Anchor program's
// `withdraw_vector` test (chains/sol/contract/programs/custody/src/digest.rs).
// Any divergence between this off-chain digest and the on-chain one would make
// every Solana withdrawal fail Ed25519 verification, so the two MUST stay
// identical — this test is the guard.
func TestDigestVectors(t *testing.T) {
	pk := func(f func(int) byte) solana.PublicKey {
		var b [32]byte
		for i := 0; i < 32; i++ {
			b[i] = f(i)
		}
		return solana.PublicKeyFromBytes(b[:])
	}
	pid := pk(func(i int) byte { return byte(i + 1) })
	vault := pk(func(i int) byte { return byte(i + 100) })
	to := pk(func(i int) byte { return byte(200 - i) })
	mint := pk(func(i int) byte { return byte(i * 3) })
	var wid [32]byte
	for i := 0; i < 32; i++ {
		wid[i] = byte(i)
	}
	const amount = uint64(1234567890)

	cases := []struct {
		name        string
		mint        solana.PublicKey
		finalizedAt int64
		signerNonce uint64
		want        string
	}{
		{
			name:        "spl mint",
			mint:        mint,
			finalizedAt: 1700000000,
			signerNonce: 42,
			want:        "0c7d0c7b03b4118b299d22da2ad628fe2ddbb5121926382a6cff9f4a6f537704",
		},
		{
			name:        "native mint",
			mint:        solana.PublicKey{},
			finalizedAt: 1700000000,
			signerNonce: 42,
			want:        "cc911b736aac34c321b28735bfb61134b78c9c943a732c2810e903308011de8f",
		},
		{
			name:        "boundary values",
			mint:        mint,
			finalizedAt: 1<<63 - 1,
			signerNonce: 1<<64 - 1,
			want:        "34aafc5e18eb181dc6b20d2038d2d7e3d240c9e6223578c0d0cabd8b56c60da3",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := WithdrawDigest(7, pid, vault, to, tc.mint, amount, wid, tc.finalizedAt, tc.signerNonce)
			if got := hex.EncodeToString(d[:]); got != tc.want {
				t.Fatalf("digest mismatch:\n got  %s\n want %s", got, tc.want)
			}
		})
	}
}
