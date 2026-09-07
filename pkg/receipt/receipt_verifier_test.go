package receipt

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/layer-3/clearnet-sdk/pkg/core"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

// stubSignerSource is a controllable SignerSource for tests.
type stubSignerSource struct {
	epoch     uint64
	signers   []common.Address
	threshold int
	loadErr   error
	issuerID  common.Address
}

func (s *stubSignerSource) LoadLatestReceiptSignerState(_ context.Context, issuerID common.Address) (core.ReceiptSignerState, error) {
	if s.loadErr != nil {
		return core.ReceiptSignerState{}, s.loadErr
	}
	if s.issuerID != (common.Address{}) && s.issuerID != issuerID {
		return core.ReceiptSignerState{}, errors.New("wrong issuer")
	}
	out := make([]common.Address, len(s.signers))
	copy(out, s.signers)
	epoch := s.epoch
	if epoch == 0 {
		epoch = 1
	}
	return core.ReceiptSignerState{Epoch: epoch, Signers: out, Threshold: s.threshold}, nil
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
	r.Status = core.WithdrawalExecuted
	r.Proof.SignerEpoch = 1
	return r
}

func signWith(t *testing.T, r *core.BurnReceipt, keys ...*ecdsa.PrivateKey) {
	t.Helper()
	logical := BurnReceiptDigest(r)
	digest := ReceiptAuthorizationDigest(r.Proof.SignerEpoch, logical)
	r.Proof.Signatures = make([][]byte, len(keys))
	for i, k := range keys {
		sig, err := crypto.Sign(digest[:], k)
		if err != nil {
			t.Fatalf("sign[%d]: %v", i, err)
		}
		r.Proof.Signatures[i] = sig
	}
}

func makeMintReceipt() *core.MintReceipt {
	return &core.MintReceipt{
		TxID:     "0xaaa/1",
		Account:  "yellow://ynet/user/0xabc",
		AssetURI: core.AssetURI("yellow://ynet/asset/" + testIssuerID.Hex() + "/evm/1/0xa0b8000000000000000000000000000000000001"),
		Amount:   decimal.NewFromInt(1),
	}
}

func signMintWith(t *testing.T, r *core.MintReceipt, keys ...*ecdsa.PrivateKey) {
	t.Helper()
	r.Proof.SignerEpoch = 1
	logical := MintReceiptDigest(r)
	digest := ReceiptAuthorizationDigest(r.Proof.SignerEpoch, logical)
	r.Proof.Signatures = make([][]byte, len(keys))
	for i, key := range keys {
		sig, err := crypto.Sign(digest[:], key)
		if err != nil {
			t.Fatalf("sign mint[%d]: %v", i, err)
		}
		r.Proof.Signatures[i] = sig
	}
}

func TestPrepareReceiptSignaturesGuards(t *testing.T) {
	a1 := common.HexToAddress("0x0000000000000000000000000000000000000001")
	a2 := common.HexToAddress("0x0000000000000000000000000000000000000002")
	tests := []struct {
		name      string
		signers   []common.Address
		threshold int
		wantErr   bool
	}{
		{name: "zero threshold", signers: []common.Address{a1}, threshold: 0, wantErr: true},
		{name: "threshold equals signer count", signers: []common.Address{a1, a2}, threshold: 2},
		{name: "threshold exceeds signer count", signers: []common.Address{a1, a2}, threshold: 3, wantErr: true},
		{name: "zero signer", signers: []common.Address{a1, {}}, threshold: 1, wantErr: true},
		{name: "dedup collapses below threshold", signers: []common.Address{a1, a1}, threshold: 2, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rv := NewReceiptVerifier(&stubSignerSource{signers: tc.signers, threshold: tc.threshold}, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
			_, err := rv.PrepareBurnReceiptSignatures(context.Background(), makeReceipt(1))
			if (err != nil) != tc.wantErr {
				t.Fatalf("PrepareBurnReceiptSignatures() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestPrepareBurnReceiptSignaturesFreezesRosterAndMatchesVerifier(t *testing.T) {
	k1, k2, k3 := mustGenerateKey(t), mustGenerateKey(t), mustGenerateKey(t)
	source := &stubSignerSource{signers: []common.Address{
		crypto.PubkeyToAddress(k1.PublicKey), crypto.PubkeyToAddress(k2.PublicKey),
	}, threshold: 2}
	rv := NewReceiptVerifier(source, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	r := makeReceipt(0x50)
	prepared, err := rv.PrepareBurnReceiptSignatures(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	signWith(t, r, k1, k2)
	// Mutating the source after preparation must not change candidate rules.
	source.signers = nil
	source.threshold = 0
	for i, sig := range r.Proof.Signatures {
		if _, ok := prepared.ValidateSignature(sig); !ok {
			t.Fatalf("signature %d rejected", i)
		}
	}
	if err := prepared.VerifySignatures(r.Proof.Signatures); err != nil {
		t.Fatal(err)
	}
	forward, err := prepared.QuorumSignatures(r.Proof.Signatures)
	if err != nil {
		t.Fatal(err)
	}
	reverse, err := prepared.QuorumSignatures([][]byte{r.Proof.Signatures[1], r.Proof.Signatures[0]})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(forward, reverse) {
		t.Fatal("receipt quorum assembly depends on arrival order")
	}
	source.signers = []common.Address{crypto.PubkeyToAddress(k2.PublicKey), crypto.PubkeyToAddress(k1.PublicKey)}
	source.threshold = 2
	identical, err := rv.PrepareBurnReceiptSignatures(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.MatchesSigningContext(identical) || !identical.MatchesSigningContext(prepared) {
		t.Fatal("identical signing contexts did not match in both directions")
	}
	source.signers = []common.Address{crypto.PubkeyToAddress(k1.PublicKey), crypto.PubkeyToAddress(k3.PublicKey)}
	source.threshold = 2
	if current, err := rv.PrepareBurnReceiptSignatures(context.Background(), r); err != nil || prepared.MatchesSigningContext(current) {
		t.Fatalf("same-size roster replacement matched prepared snapshot: current=%v err=%v", current, err)
	}
	source.signers = nil
	source.threshold = 0
	if current, err := rv.PrepareBurnReceiptSignatures(context.Background(), r); err == nil || prepared.MatchesSigningContext(current) {
		t.Fatalf("invalid changed signer source unexpectedly matched prepared snapshot: current=%v err=%v", current, err)
	}
}

func newTestBurnVerifier(signers []common.Address, threshold int) *ReceiptVerifier {
	rv := NewReceiptVerifier(nil, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	if err := rv.SetSignersForTest(signers, threshold); err != nil {
		panic(err)
	}
	return rv
}

func TestReceiptVerifier_SetSignersForTestRejectsInvalidInput(t *testing.T) {
	rv := NewReceiptVerifier(nil, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	if err := rv.SetSignersForTest(nil, 1); err == nil {
		t.Fatal("expected invalid signer set error")
	}
	if err := rv.SetSignersForTest([]common.Address{{}}, 1); err == nil {
		t.Fatal("expected zero signer error")
	}
	r := makeReceipt(0x03)
	r.Proof.Signatures = [][]byte{make([]byte, 65)}
	err := rv.VerifyBurnReceipt(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "no signer source") {
		t.Fatalf("Verify after invalid setup = %v, want no signer source", err)
	}
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
	r.Proof.Signatures = [][]byte{make([]byte, 65), make([]byte, 65)}
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
			want:   "load receipt signer state",
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
			r.Proof.Signatures = [][]byte{make([]byte, 65)}
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

func TestReceiptVerifier_RejectsLogicalDigestSignature(t *testing.T) {
	key := mustGenerateKey(t)
	rv := newTestBurnVerifier([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	r := makeReceipt(0x21)
	logical := BurnReceiptDigest(r)
	sig, err := crypto.Sign(logical[:], key)
	if err != nil {
		t.Fatal(err)
	}
	r.Proof.Signatures = [][]byte{sig}

	err = rv.VerifyBurnReceipt(context.Background(), r)
	if verificationCode(err) != ReceiptVerificationInvalidSignatures {
		t.Fatalf("VerifyBurnReceipt() = %v, want invalid_signatures", err)
	}
}

func TestReceiptVerifier_EpochMismatchClassifiesBeforeSignatures(t *testing.T) {
	key := mustGenerateKey(t)
	source := &stubSignerSource{
		epoch:     2,
		signers:   []common.Address{crypto.PubkeyToAddress(key.PublicKey)},
		threshold: 1,
	}
	rv := NewReceiptVerifier(source, stubWithdrawalIssuerResolver{issuerID: testIssuerID})

	stale := makeReceipt(0x22)
	stale.Proof.SignerEpoch = 1
	stale.Proof.Signatures = [][]byte{[]byte("not a signature")}
	if err := rv.VerifyBurnReceipt(context.Background(), stale); verificationCode(err) != ReceiptVerificationStaleEpoch {
		t.Fatalf("stale VerifyBurnReceipt() = %v, want stale_epoch", err)
	}

	future := makeReceipt(0x23)
	future.Proof.SignerEpoch = 3
	future.Proof.Signatures = [][]byte{[]byte("not a signature")}
	if err := rv.VerifyBurnReceipt(context.Background(), future); verificationCode(err) != ReceiptVerificationFutureEpoch {
		t.Fatalf("future VerifyBurnReceipt() = %v, want future_epoch", err)
	}
}

func TestReceiptVerifier_ChangingSignerEpochInvalidatesSignature(t *testing.T) {
	key := mustGenerateKey(t)
	source := &stubSignerSource{
		epoch:     2,
		signers:   []common.Address{crypto.PubkeyToAddress(key.PublicKey)},
		threshold: 1,
	}
	rv := NewReceiptVerifier(source, stubWithdrawalIssuerResolver{issuerID: testIssuerID})
	r := makeReceipt(0x24)
	r.Proof.SignerEpoch = 1
	logical := BurnReceiptDigest(r)
	digest := ReceiptAuthorizationDigest(r.Proof.SignerEpoch, logical)
	sig, err := crypto.Sign(digest[:], key)
	if err != nil {
		t.Fatal(err)
	}
	r.Proof.SignerEpoch = 2
	r.Proof.Signatures = [][]byte{sig}

	if err := rv.VerifyBurnReceipt(context.Background(), r); verificationCode(err) != ReceiptVerificationInvalidSignatures {
		t.Fatalf("VerifyBurnReceipt() = %v, want invalid_signatures", err)
	}
}

func TestReceiptVerifier_VerifyAcceptsEthereumCanonicalV(t *testing.T) {
	key := mustGenerateKey(t)
	rv := newTestBurnVerifier([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)

	r := makeReceipt(0x17)
	signWith(t, r, key)
	r.Proof.Signatures[0][64] += 27

	if err := rv.VerifyBurnReceipt(context.Background(), r); err != nil {
		t.Fatalf("Verify with Ethereum v=27/28: %v", err)
	}
}

func TestReceiptSignatureValidatorCanonicalizesEquivalentEncodings(t *testing.T) {
	key := mustGenerateKey(t)
	rv := newTestBurnVerifier([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	r := makeReceipt(0x18)
	signWith(t, r, key)
	low := append([]byte(nil), r.Proof.Signatures[0]...)
	legacyV := append([]byte(nil), low...)
	legacyV[64] += 27
	high := append([]byte(nil), low...)
	s := new(big.Int).SetBytes(high[32:64])
	s.Sub(crypto.S256().Params().N, s)
	s.FillBytes(high[32:64])
	high[64] ^= 1
	highLegacyV := append([]byte(nil), high...)
	highLegacyV[64] += 27

	validator, err := rv.PrepareBurnReceiptSignatures(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	for name, candidate := range map[string][]byte{
		"low-S wire V": low, "low-S legacy V": legacyV,
		"high-S wire V": high, "high-S legacy V": highLegacyV,
	} {
		t.Run(name, func(t *testing.T) {
			if signer, ok := validator.ValidateSignature(candidate); !ok || signer != crypto.PubkeyToAddress(key.PublicKey) {
				t.Fatalf("ValidateSignature() = %s, %v", signer, ok)
			}
			quorum, err := validator.QuorumSignatures([][]byte{candidate})
			if err != nil {
				t.Fatal(err)
			}
			if len(quorum) != 1 || !reflect.DeepEqual(quorum[0], low) {
				t.Fatalf("canonical quorum = %x, want %x", quorum, low)
			}
		})
	}
}

func TestReceiptVerifier_VerifyAllowsTrailingMalformedCandidates(t *testing.T) {
	// Pool has 5 signers; threshold 3; trailing garbage must not invalidate a
	// receipt once three distinct authorized signatures have been established.
	keys := make([]*ecdsa.PrivateKey, 5)
	addrs := make([]common.Address, 5)
	for i := range keys {
		keys[i] = mustGenerateKey(t)
		addrs[i] = crypto.PubkeyToAddress(keys[i].PublicKey)
	}
	rv := newTestBurnVerifier(addrs, 3)

	r := makeReceipt(0x11)
	signWith(t, r, keys[0], keys[1], keys[2])
	r.Proof.Signatures = append(r.Proof.Signatures, []byte("garbage"), make([]byte, 65))

	if err := rv.VerifyBurnReceipt(context.Background(), r); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestReceiptVerifier_PoisonBeforeValidCandidateDoesNotOccupySlot(t *testing.T) {
	key := mustGenerateKey(t)
	rv := newTestBurnVerifier([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	r := makeReceipt(0x19)
	signWith(t, r, key)
	r.Proof.Signatures = append([][]byte{make([]byte, 65), []byte("garbage")}, r.Proof.Signatures...)
	if err := rv.VerifyBurnReceipt(context.Background(), r); err != nil {
		t.Fatalf("VerifyBurnReceipt() = %v", err)
	}
}

func TestReceiptSignatureValidatorNilAndCandidateLimitGuards(t *testing.T) {
	var nilValidator *ReceiptSignatureValidator
	if err := nilValidator.VerifySignatures(nil); err == nil {
		t.Fatal("nil validator verified signatures")
	}
	if _, err := nilValidator.QuorumSignatures(nil); err == nil {
		t.Fatal("nil validator assembled signatures")
	}
	if nilValidator.Digest() != ([32]byte{}) || nilValidator.Threshold() != 0 ||
		nilValidator.MatchesSigningContext(nil) {
		t.Fatal("nil receiver contract changed")
	}
	key := mustGenerateKey(t)
	rv := newTestBurnVerifier([]common.Address{crypto.PubkeyToAddress(key.PublicKey)}, 1)
	validator, err := rv.PrepareBurnReceiptSignatures(context.Background(), makeReceipt(0x1a))
	if err != nil {
		t.Fatal(err)
	}
	tooMany := make([][]byte, maxCandidatesPerReceiptSigner+1)
	if err := validator.VerifySignatures(tooMany); err == nil {
		t.Fatal("oversized candidate set verified")
	}
	if _, err := validator.QuorumSignatures(tooMany); err == nil {
		t.Fatal("oversized candidate set assembled")
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
	validator, prepareErr := rv.PrepareBurnReceiptSignatures(context.Background(), r)
	if prepareErr != nil {
		t.Fatal(prepareErr)
	}
	if got, ok := validator.ValidateSignature(r.Proof.Signatures[0]); ok || got != (common.Address{}) {
		t.Fatalf("unauthorized signature returned %s, %v", got, ok)
	}

	err := rv.VerifyBurnReceipt(context.Background(), r)
	if err == nil || !strings.Contains(err.Error(), "insufficient distinct signers") ||
		!strings.Contains(err.Error(), "unauthorized=1") {
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
	r.Proof.Signatures = [][]byte{make([]byte, 64)}
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
	r.Proof.Signatures = [][]byte{make([]byte, 65)}
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

func TestPrepareAndVerifyMintReceiptSignatures(t *testing.T) {
	key := mustGenerateKey(t)
	source := &stubSignerSource{
		signers:   []common.Address{crypto.PubkeyToAddress(key.PublicKey)},
		threshold: 1,
		issuerID:  testIssuerID,
	}
	rv := NewReceiptVerifier(source, nil)
	r := makeMintReceipt()
	validator, err := rv.PrepareMintReceiptSignatures(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if validator.LogicalDigest() != MintReceiptDigest(r) || validator.Digest() != ReceiptAuthorizationDigest(validator.SignerEpoch(), MintReceiptDigest(r)) || validator.Threshold() != 1 {
		t.Fatal("prepared MintReceipt digest or threshold mismatch")
	}
	signMintWith(t, r, key)
	if err := rv.VerifyMintReceipt(context.Background(), r); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareMintReceiptSignaturesRejectsInvalidReceipt(t *testing.T) {
	key := mustGenerateKey(t)
	rv := NewReceiptVerifier(&stubSignerSource{
		signers: []common.Address{crypto.PubkeyToAddress(key.PublicKey)}, threshold: 1,
	}, nil)
	tests := []struct {
		name string
		r    *core.MintReceipt
	}{
		{name: "nil", r: nil},
		{name: "non-positive amount", r: func() *core.MintReceipt {
			r := makeMintReceipt()
			r.Amount = decimal.NewFromInt(0)
			return r
		}()},
		{name: "malformed asset URI", r: func() *core.MintReceipt {
			r := makeMintReceipt()
			r.AssetURI = "not-an-asset-uri"
			return r
		}()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := rv.PrepareMintReceiptSignatures(context.Background(), tc.r); err == nil {
				t.Fatal("invalid MintReceipt prepared")
			}
		})
	}
}

func TestReceiptVerifier_DigestIsDeterministic(t *testing.T) {
	r1 := makeReceipt(0x20)
	r2 := makeReceipt(0x20)
	d1 := BurnReceiptDigest(r1)
	d2 := BurnReceiptDigest(r2)
	if d1 != d2 {
		t.Fatalf("digest non-deterministic: %x vs %x", d1, d2)
	}
	// Mutating any field changes the digest.
	r2.TxID = "changed"
	d3 := BurnReceiptDigest(r2)
	if d1 == d3 {
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
	if d1 == d2 {
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
			if MintReceiptDigest(r) == base {
				t.Fatalf("%s mutation did not change digest", tc.name)
			}
		})
	}
}

func TestReceiptDigestIgnoresProof(t *testing.T) {
	mint := makeMintReceipt()
	mintDigest := MintReceiptDigest(mint)
	mint.Proof.SignerEpoch = 99
	mint.Proof.Signatures = [][]byte{{1, 2, 3}}
	if got := MintReceiptDigest(mint); got != mintDigest {
		t.Fatalf("mint logical digest changed with proof: %x vs %x", got, mintDigest)
	}

	burn := makeReceipt(0x40)
	burnDigest := BurnReceiptDigest(burn)
	burn.Proof.SignerEpoch = 99
	burn.Proof.Signatures = [][]byte{{1, 2, 3}}
	if got := BurnReceiptDigest(burn); got != burnDigest {
		t.Fatalf("burn logical digest changed with proof: %x vs %x", got, burnDigest)
	}
}

func TestReceiptLogicalIDIgnoresProofAndPayloadFields(t *testing.T) {
	mint := makeMintReceipt()
	mintID, err := MintReceiptLogicalID(mint)
	if err != nil {
		t.Fatal(err)
	}
	mint.Account = "yellow://ynet/user/changed"
	mint.Amount = decimal.NewFromInt(99)
	mint.Proof = core.ReceiptProof{SignerEpoch: 22, Signatures: [][]byte{{1}}}
	if got, err := MintReceiptLogicalID(mint); err != nil || got != mintID {
		t.Fatalf("mint logical id = %x, %v; want %x", got, err, mintID)
	}

	burn := makeReceipt(0x41)
	burnID, err := BurnReceiptLogicalID(burn)
	if err != nil {
		t.Fatal(err)
	}
	burn.BlockHash = [32]byte{0xaa}
	burn.EntryIndex = 99
	burn.TxID = "changed"
	burn.Status = core.WithdrawalExpired
	burn.Proof = core.ReceiptProof{SignerEpoch: 22, Signatures: [][]byte{{1}}}
	if got, err := BurnReceiptLogicalID(burn); err != nil || got != burnID {
		t.Fatalf("burn logical id = %x, %v; want %x", got, err, burnID)
	}
}

func verificationCode(err error) ReceiptVerificationCode {
	var verificationErr *ReceiptVerificationError
	if errors.As(err, &verificationErr) {
		return verificationErr.Code
	}
	return ""
}
