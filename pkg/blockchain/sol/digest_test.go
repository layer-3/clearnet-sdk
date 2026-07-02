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
		name     string
		mint     solana.PublicKey
		deadline int64
		want     string
	}{
		{
			name:     "spl mint",
			mint:     mint,
			deadline: 1700000000,
			want:     "c850ef8d47806dc2ac72e968f7c6d98762b54ef112bca814b49acc482bd8eb62",
		},
		{
			name:     "native mint",
			mint:     solana.PublicKey{},
			deadline: 1700000000,
			want:     "4ed8353197f5ee4c66c9d5f337b5f9feb6dfb085280b16d452d6b201e9a6fa9d",
		},
		{
			name:     "boundary deadline",
			mint:     mint,
			deadline: 1<<63 - 1, // i64::MAX
			want:     "876abec0a980181a365dd2ae90b44f56c51d6a22ed5128ff03f55aa6a04a55f5",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := WithdrawDigest(7, pid, vault, to, tc.mint, amount, wid, tc.deadline)
			if got := hex.EncodeToString(d[:]); got != tc.want {
				t.Fatalf("digest mismatch:\n got  %s\n want %s", got, tc.want)
			}
		})
	}
}
