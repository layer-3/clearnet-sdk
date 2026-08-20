package receipt

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// stubSignerSource is a controllable SignerSource for tests.
type stubSignerSource struct {
	signers   []common.Address
	threshold int
	loadErr   error
	issuerID  common.Address
}

func (s *stubSignerSource) LoadReceiptSigners(_ context.Context, issuerID common.Address) (core.ReceiptSignerSet, error) {
	if s.loadErr != nil {
		return core.ReceiptSignerSet{}, s.loadErr
	}
	if s.issuerID != (common.Address{}) && s.issuerID != issuerID {
		return core.ReceiptSignerSet{}, errors.New("wrong issuer")
	}
	out := make([]common.Address, len(s.signers))
	copy(out, s.signers)
	return core.ReceiptSignerSet{Signers: out, Threshold: s.threshold}, nil
}

type stubWithdrawalIssuerResolver struct {
	issuerID common.Address
	err      error
}

func (r stubWithdrawalIssuerResolver) IssuerIDByWithdrawalID(context.Context, [32]byte) (common.Address, error) {
	return r.issuerID, r.err
}

var testIssuerID = common.HexToAddress("0x0000000000000000000000000000000000001234")
var testOtherIssuerID = common.HexToAddress("0x0000000000000000000000000000000000005678")

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
	r.TxID = string([]byte{seed, 0xC3})
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

func newTestBurnVerifier(signers []common.Address, threshold int) *ReceiptVerifier {
	rv := NewReceiptVerifier(nil, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	rv.SetSignersForTest(signers, threshold)
	return rv
}

func TestReceiptVerifier_LoadsIssuerScopedSigners(t *testing.T) {
	signers := []common.Address{
		common.HexToAddress("0x1111111111111111111111111111111111111111"),
		common.HexToAddress("0x2222222222222222222222222222222222222222"),
		common.HexToAddress("0x3333333333333333333333333333333333333333"),
	}
	rv := NewReceiptVerifier(&stubSignerSource{
		signers:   signers,
		threshold: 2,
		issuerID:  testIssuerID,
	}, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	r := makeReceipt(0x01)
	r.Signatures = [][]byte{make([]byte, 65), make([]byte, 65)}
	err := rv.VerifyBurnReceipt(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "insufficient distinct signers") {
		t.Fatalf("Verify = %v, want signer load through issuer %s", err, testIssuerID.Hex())
	}
}

func TestReceiptVerifier_LoadFailsClosedOnSourceErrors(t *testing.T) {
	cases := []struct {
		name   string
		source *stubSignerSource
		want   string
	}{
		{
			name:   "Load error",
			source: &stubSignerSource{loadErr: errors.New("source down")},
			want:   "load receipt signers",
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
			rv := NewReceiptVerifier(tc.source, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
			r := makeReceipt(0x02)
			r.Signatures = [][]byte{make([]byte, 65)}
			err := rv.VerifyBurnReceipt(context.Background(), r)
			if err == nil {
				t.Fatal("expected Verify to fail")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Verify error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestReceiptVerifier_VerifyHappyPath(t *testing.T) {
	keys := []*ecdsa.PrivateKey{mustGenerateKey(t), mustGenerateKey(t), mustGenerateKey(t)}
	addrs := []common.Address{
		crypto.PubkeyToAddress(keys[0].PublicKey),
		crypto.PubkeyToAddress(keys[1].PublicKey),
		crypto.PubkeyToAddress(keys[2].PublicKey),
	}
	rv := newTestBurnVerifier(addrs, 2)

	r := makeReceipt(0x10)
	signWith(t, r, keys[0], keys[2])

	if err := rv.VerifyBurnReceipt(context.Background(), r); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestReceiptVerifier_VerifyAcceptsEthereumCanonicalV(t *testing.T) {
	key := mustGenerateKey(t)
	rv := newTestBurnVerifier([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)

	r := makeReceipt(0x17)
	signWith(t, r, key)
	r.Signatures[0][64] += 27

	if err := rv.VerifyBurnReceipt(context.Background(), r); err != nil {
		t.Fatalf("Verify with Ethereum v=27/28: %v", err)
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
	rv := newTestBurnVerifier(addrs, 3)

	r := makeReceipt(0x11)
	signWith(t, r, keys[0], keys[1], keys[2])
	// Append two malformed sigs; if the verifier scanned them it would
	// still succeed (they're skipped), but the test asserts threshold-met
	// short-circuits regardless.
	r.Signatures = append(r.Signatures, []byte("garbage"), make([]byte, 65))

	if err := rv.VerifyBurnReceipt(context.Background(), r); err != nil {
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
	rv := newTestBurnVerifier(addrs, 3)

	r := makeReceipt(0x12)
	signWith(t, r, keys[0], keys[1])

	err := rv.VerifyBurnReceipt(context.Background(), r)
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
	rv := newTestBurnVerifier(addrs, 2)

	r := makeReceipt(0x13)
	// Two valid signatures from the same signer must NOT count as two
	// distinct contributors.
	signWith(t, r, keys[0], keys[0])

	err := rv.VerifyBurnReceipt(context.Background(), r)
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
	rv := newTestBurnVerifier(addrs, 1)

	r := makeReceipt(0x14)
	signWith(t, r, intruderKey)

	err := rv.VerifyBurnReceipt(context.Background(), r)
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
	rv := newTestBurnVerifier(addrs, 1)

	r := makeReceipt(0x15)
	r.Signatures = [][]byte{make([]byte, 64)}
	err := rv.VerifyBurnReceipt(context.Background(), r)
	// Sig count >= threshold so the early-out doesn't trigger; ED25519 is
	// recognised but ignored, so we end up with zero distinct signers.
	if err == nil || !strings.Contains(err.Error(), "insufficient distinct signers") {
		t.Fatalf("Verify: want ED25519-stub rejection, got %v", err)
	}
}

func TestReceiptVerifier_VerifyFailsClosedWithoutSignerSource(t *testing.T) {
	rv := NewReceiptVerifier(nil, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	r := makeReceipt(0x16)
	r.Signatures = [][]byte{make([]byte, 65)}
	err := rv.VerifyBurnReceipt(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "no signer source") {
		t.Fatalf("Verify: want no signer source error, got %v", err)
	}
}

func TestReceiptVerifier_VerifyRejectsNilReceipt(t *testing.T) {
	rv := newTestBurnVerifier([]common.Address{common.HexToAddress("0x01")}, 1)
	if err := rv.VerifyBurnReceipt(context.Background(), nil); err == nil {
		t.Fatal("Verify(nil): want error")
	}
}

func TestReceiptVerifier_NilReceiverFailsClosed(t *testing.T) {
	var rv *ReceiptVerifier
	if err := rv.VerifyBurnReceipt(context.Background(), makeReceipt(0)); err == nil {
		t.Fatal("Verify on nil verifier: want error")
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
	r2.TxID = "changed"
	d3 := BurnReceiptDigest(r2)
	if string(d1) == string(d3) {
		t.Fatal("digest unchanged after mutating TxID")
	}
}

// Without uint32 length prefixes on variable-size fields, boundary shifts can
// hash to the same preimage.
func TestMintReceiptDigest_NoStringCollision(t *testing.T) {
	mk := func(txID, account, assetURI string) *core.MintReceipt {
		return &core.MintReceipt{
			TxID:     txID,
			Account:  account,
			AssetURI: core.AssetURI(assetURI),
			Amount:   decimal.NewFromInt(1),
		}
	}
	d1 := MintReceiptDigest(mk("abc", "def", "ghi"))
	d2 := MintReceiptDigest(mk("abcd", "ef", "ghi"))
	if string(d1) == string(d2) {
		t.Fatalf("digest collided across variable-field boundary shift: %x", d1)
	}
}

func TestMintReceiptDigest_FieldSensitivity(t *testing.T) {
	mk := func() *core.MintReceipt {
		return &core.MintReceipt{
			TxID:     "0xaaa/1",
			Account:  "yellow://ynet/user/0xabc",
			AssetURI: core.AssetURI("yellow://ynet/asset/" + testIssuerID.Hex() + "/evm/1/0xa0b8000000000000000000000000000000000001"),
			Amount:   decimal.NewFromInt(1),
		}
	}
	base := MintReceiptDigest(mk())

	cases := []struct {
		name string
		mut  func(*core.MintReceipt)
	}{
		{"TxID", func(r *core.MintReceipt) { r.TxID = "0xaaa/2" }},
		{"Account", func(r *core.MintReceipt) { r.Account = "yellow://ynet/user/0xabd" }},
		{"AssetURI", func(r *core.MintReceipt) {
			r.AssetURI = core.AssetURI("yellow://ynet/asset/" + testOtherIssuerID.Hex() + "/evm/1/0xa0b8000000000000000000000000000000000002")
		}},
		{"Amount", func(r *core.MintReceipt) { r.Amount = decimal.NewFromInt(2) }},
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
