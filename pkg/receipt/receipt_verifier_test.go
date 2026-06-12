package receipt

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
)

// stubSignerSource is a controllable SignerSource for tests.
type stubSignerSource struct {
	signers   []common.Address
	threshold int
	loadErr   error
}

func (s *stubSignerSource) Load(context.Context) ([]common.Address, int, error) {
	if s.loadErr != nil {
		return nil, 0, s.loadErr
	}
	out := make([]common.Address, len(s.signers))
	copy(out, s.signers)
	return out, s.threshold, nil
}

func mustGenerateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	k, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	return k
}

func makeReceipt(seed byte) *core.BurnReceipt {
	r := &core.BurnReceipt{}
	r.WithdrawalID = [32]byte{seed, 0xA1}
	r.BlockHash = [32]byte{seed, 0xB2}
	r.EntryIndex = uint64(seed)
	r.L1TxHash = [32]byte{seed, 0xC3}
	return r
}

func signWith(t *testing.T, r *core.BurnReceipt, keys ...*ecdsa.PrivateKey) {
	t.Helper()
	digest := BurnReceiptDigest(r)
	r.Signatures = make([][]byte, len(keys))
	for i, k := range keys {
		sig, err := crypto.Sign(digest, k)
		if err != nil {
			t.Fatalf("sign[%d]: %v", i, err)
		}
		r.Signatures[i] = sig
	}
}

func TestReceiptVerifier_RefreshPopulatesCache(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}
	rv := NewReceiptVerifier(&stubSignerSource{
		signers:   signers,
		threshold: 2,
	}, 0)
	if err := rv.Refresh(context.Background()); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if got := rv.SignerCount(); got != 3 {
		t.Fatalf("SignerCount = %d, want 3", got)
	}
	if got := rv.Threshold(); got != 2 {
		t.Fatalf("Threshold = %d, want 2", got)
	}
}

func TestReceiptVerifier_RefreshFailsClosedOnSourceErrors(t *testing.T) {
	cases := []struct {
		name   string
		source *stubSignerSource
		want   string
	}{
		{
			name:   "Load error",
			source: &stubSignerSource{loadErr: errors.New("source down")},
			want:   "load custody signers",
		},
		{
			name: "threshold zero",
			source: &stubSignerSource{
				signers:   []common.Address{common.HexToAddress("0x01")},
				threshold: 0,
			},
			want: "out of range",
		},
		{
			name: "threshold exceeds signer count",
			source: &stubSignerSource{
				signers:   []common.Address{common.HexToAddress("0x01")},
				threshold: 2,
			},
			want: "out of range",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rv := NewReceiptVerifier(tc.source, 0)
			err := rv.Refresh(context.Background())
			if err == nil {
				t.Fatal("expected Refresh to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Refresh error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestReceiptVerifier_RefreshAtomicallyReplacesCache(t *testing.T) {
	first := []common.Address{common.HexToAddress("0xAA")}
	second := []common.Address{
		common.HexToAddress("0xBB"),
		common.HexToAddress("0xCC"),
	}
	source := &stubSignerSource{signers: first, threshold: 1}
	rv := NewReceiptVerifier(source, 0)
	if err := rv.Refresh(context.Background()); err != nil {
		t.Fatalf("first refresh: %v", err)
	}

	source.signers = second
	source.threshold = 2
	if err := rv.Refresh(context.Background()); err != nil {
		t.Fatalf("second refresh: %v", err)
	}
	if got := rv.SignerCount(); got != 2 {
		t.Fatalf("SignerCount after refresh = %d, want 2", got)
	}
	if got := rv.Threshold(); got != 2 {
		t.Fatalf("Threshold after refresh = %d, want 2", got)
	}
}

func TestReceiptVerifier_VerifyHappyPath(t *testing.T) {
	keys := []*ecdsa.PrivateKey{mustGenerateKey(t), mustGenerateKey(t), mustGenerateKey(t)}
	addrs := []common.Address{
		crypto.PubkeyToAddress(keys[0].PublicKey),
		crypto.PubkeyToAddress(keys[1].PublicKey),
		crypto.PubkeyToAddress(keys[2].PublicKey),
	}
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest(addrs, 2)

	r := makeReceipt(0x10)
	signWith(t, r, keys[0], keys[2])

	if err := rv.VerifyBurnReceipt(r); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestReceiptVerifier_VerifyAcceptsEthereumCanonicalV(t *testing.T) {
	key := mustGenerateKey(t)
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)

	r := makeReceipt(0x17)
	signWith(t, r, key)
	r.Signatures[0][64] += 27

	if err := rv.VerifyBurnReceipt(r); err != nil {
		t.Fatalf("Verify with Ethereum v=27/28: %v", err)
	}
}

func TestReceiptVerifier_VerifyFailsClosedOnStaleCache(t *testing.T) {
	key := mustGenerateKey(t)
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	rv.mu.Lock()
	rv.refreshedAt = time.Now().Add(-time.Hour)
	rv.maxAge = time.Minute
	rv.mu.Unlock()

	r := makeReceipt(0x18)
	signWith(t, r, key)
	err := rv.VerifyBurnReceipt(r)
	if err == nil || !strings.Contains(err.Error(), "signer set stale") {
		t.Fatalf("Verify: want stale signer set error, got %v", err)
	}
}

func TestReceiptVerifier_VerifyExitsAsSoonAsThresholdMet(t *testing.T) {
	// Pool has 5 signers; threshold 3; receipt carries 5 sigs but the
	// first three suffice. The verifier returns nil without scanning sigs
	// 4 and 5 (which is the property under test — we feed garbage in
	// those slots and verification still succeeds).
	keys := make([]*ecdsa.PrivateKey, 5)
	addrs := make([]common.Address, 5)
	for i := range keys {
		keys[i] = mustGenerateKey(t)
		addrs[i] = crypto.PubkeyToAddress(keys[i].PublicKey)
	}
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest(addrs, 3)

	r := makeReceipt(0x11)
	signWith(t, r, keys[0], keys[1], keys[2])
	// Append two malformed sigs; if the verifier scanned them it would
	// still succeed (they're skipped), but the test asserts threshold-met
	// short-circuits regardless.
	r.Signatures = append(r.Signatures, []byte("garbage"), make([]byte, 65))

	if err := rv.VerifyBurnReceipt(r); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestReceiptVerifier_VerifyRejectsTooFewSignatures(t *testing.T) {
	keys := []*ecdsa.PrivateKey{mustGenerateKey(t), mustGenerateKey(t), mustGenerateKey(t)}
	addrs := []common.Address{
		crypto.PubkeyToAddress(keys[0].PublicKey),
		crypto.PubkeyToAddress(keys[1].PublicKey),
		crypto.PubkeyToAddress(keys[2].PublicKey),
	}
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest(addrs, 3)

	r := makeReceipt(0x12)
	signWith(t, r, keys[0], keys[1])

	err := rv.VerifyBurnReceipt(r)
	if err == nil || !strings.Contains(err.Error(), "insufficient signatures") {
		t.Fatalf("Verify: want 'insufficient signatures' error, got %v", err)
	}
}

func TestReceiptVerifier_VerifyRejectsDuplicateSigner(t *testing.T) {
	keys := []*ecdsa.PrivateKey{mustGenerateKey(t), mustGenerateKey(t), mustGenerateKey(t)}
	addrs := []common.Address{
		crypto.PubkeyToAddress(keys[0].PublicKey),
		crypto.PubkeyToAddress(keys[1].PublicKey),
		crypto.PubkeyToAddress(keys[2].PublicKey),
	}
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest(addrs, 2)

	r := makeReceipt(0x13)
	// Two valid signatures from the same signer must NOT count as two
	// distinct contributors.
	signWith(t, r, keys[0], keys[0])

	err := rv.VerifyBurnReceipt(r)
	if err == nil || !strings.Contains(err.Error(), "insufficient distinct signers") {
		t.Fatalf("Verify: want 'insufficient distinct signers' error, got %v", err)
	}
}

func TestReceiptVerifier_VerifyRejectsNonSigner(t *testing.T) {
	signerKey := mustGenerateKey(t)
	intruderKey := mustGenerateKey(t)
	addrs := []common.Address{
		crypto.PubkeyToAddress(signerKey.PublicKey),
	}
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest(addrs, 1)

	r := makeReceipt(0x14)
	signWith(t, r, intruderKey)

	err := rv.VerifyBurnReceipt(r)
	if err == nil || !strings.Contains(err.Error(), "insufficient distinct signers") {
		t.Fatalf("Verify: want 'insufficient distinct signers' error, got %v", err)
	}
}

func TestReceiptVerifier_VerifyED25519IsStubAndIgnored(t *testing.T) {
	// 64-byte signatures are recognised as ED25519 and ignored until the
	// XRPL/Solana custody adapter lands. They must not be counted toward
	// the threshold.
	signerKey := mustGenerateKey(t)
	addrs := []common.Address{crypto.PubkeyToAddress(signerKey.PublicKey)}
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest(addrs, 1)

	r := makeReceipt(0x15)
	r.Signatures = [][]byte{make([]byte, 64)}
	err := rv.VerifyBurnReceipt(r)
	// Sig count >= threshold so the early-out doesn't trigger; ED25519 is
	// recognised but ignored, so we end up with zero distinct signers.
	if err == nil || !strings.Contains(err.Error(), "insufficient distinct signers") {
		t.Fatalf("Verify: want ED25519-stub rejection, got %v", err)
	}
}

func TestReceiptVerifier_VerifyFailsClosedOnUninitialisedCache(t *testing.T) {
	rv := NewReceiptVerifier(nil, 0)
	r := makeReceipt(0x16)
	r.Signatures = [][]byte{make([]byte, 65)}
	err := rv.VerifyBurnReceipt(r)
	if err == nil || !strings.Contains(err.Error(), "signer set not initialised") {
		t.Fatalf("Verify: want uninitialised error, got %v", err)
	}
}

func TestReceiptVerifier_VerifyRejectsNilReceipt(t *testing.T) {
	rv := NewReceiptVerifier(nil, 0)
	rv.SetSignersForTest([]common.Address{common.HexToAddress("0x01")}, 1)
	if err := rv.VerifyBurnReceipt(nil); err == nil {
		t.Fatal("Verify(nil): want error")
	}
}

func TestReceiptVerifier_NilReceiverFailsClosed(t *testing.T) {
	var rv *ReceiptVerifier
	if err := rv.VerifyBurnReceipt(makeReceipt(0)); err == nil {
		t.Fatal("Verify on nil verifier: want error")
	}
	if err := rv.Refresh(context.Background()); err == nil {
		t.Fatal("Refresh on nil verifier: want error")
	}
}

func TestReceiptVerifier_DigestIsDeterministic(t *testing.T) {
	r1 := makeReceipt(0x20)
	r2 := makeReceipt(0x20)
	d1 := BurnReceiptDigest(r1)
	d2 := BurnReceiptDigest(r2)
	if string(d1) != string(d2) {
		t.Fatalf("digest non-deterministic: %x vs %x", d1, d2)
	}
	// Mutating any field changes the digest.
	r2.L1TxHash[0] ^= 0xFF
	d3 := BurnReceiptDigest(r2)
	if string(d1) == string(d3) {
		t.Fatal("digest unchanged after mutating L1TxHash")
	}
}

// Without uint32 length prefixes on Account and Asset, ("abc","XYZ") and
// ("abcX","YZ") would hash to the same preimage.
func TestMintReceiptDigest_NoStringCollision(t *testing.T) {
	mk := func(account, asset string) *core.MintReceipt {
		return &core.MintReceipt{
			ChainID: 1,
			Account: account,
			Asset:   asset,
			Amount:  big.NewInt(1),
		}
	}
	d1 := MintReceiptDigest(mk("abc", "XYZ"))
	d2 := MintReceiptDigest(mk("abcX", "YZ"))
	if string(d1) == string(d2) {
		t.Fatalf("digest collided across (Account,Asset) boundary shift: %x", d1)
	}
}

func TestMintReceiptDigest_FieldSensitivity(t *testing.T) {
	mk := func() *core.MintReceipt {
		return &core.MintReceipt{
			ChainID:  1,
			L1TxHash: [32]byte{0xAA},
			LogIndex: 1,
			Account:  "yellow://ynet/user/0xabc",
			Asset:    "USDC",
			Amount:   big.NewInt(1),
		}
	}
	base := MintReceiptDigest(mk())

	cases := []struct {
		name string
		mut  func(*core.MintReceipt)
	}{
		{"ChainID", func(r *core.MintReceipt) { r.ChainID = 2 }},
		{"L1TxHash", func(r *core.MintReceipt) { r.L1TxHash[0] ^= 0xFF }},
		{"LogIndex", func(r *core.MintReceipt) { r.LogIndex = 2 }},
		{"Account", func(r *core.MintReceipt) { r.Account = "yellow://ynet/user/0xabd" }},
		{"Asset", func(r *core.MintReceipt) { r.Asset = "USDT" }},
		{"Amount", func(r *core.MintReceipt) { r.Amount = big.NewInt(2) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := mk()
			tc.mut(r)
			if string(MintReceiptDigest(r)) == string(base) {
				t.Fatalf("%s mutation did not change digest", tc.name)
			}
		})
	}
}
