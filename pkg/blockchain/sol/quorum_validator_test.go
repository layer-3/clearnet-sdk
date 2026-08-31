package sol

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go"
)

func TestNewSignatureValidatorGuards(t *testing.T) {
	p1, _, _ := ed25519.GenerateKey(rand.Reader)
	p2, _, _ := ed25519.GenerateKey(rand.Reader)
	var a1, a2 solana.PublicKey
	copy(a1[:], p1)
	copy(a2[:], p2)
	tests := []struct {
		name      string
		signers   []solana.PublicKey
		threshold int
		wantErr   bool
	}{
		{name: "zero threshold", signers: []solana.PublicKey{a1}, threshold: 0, wantErr: true},
		{name: "threshold equals signer count", signers: []solana.PublicKey{a1, a2}, threshold: 2},
		{name: "threshold exceeds signer count", signers: []solana.PublicKey{a1, a2}, threshold: 3, wantErr: true},
		{name: "zero signer", signers: []solana.PublicKey{a1, {}}, threshold: 1, wantErr: true},
		{name: "dedup collapses below threshold", signers: []solana.PublicKey{a1, a1}, threshold: 2, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSignatureValidator([32]byte{1}, tc.signers, tc.threshold)
			if (err != nil) != tc.wantErr {
				t.Fatalf("NewSignatureValidator() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestSignatureValidatorRejectsInvalidShareBeforeSignerDedup(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var signer solana.PublicKey
	copy(signer[:], pub)
	digest := [32]byte{1, 2, 3}
	valid := append(append([]byte(nil), pub...), ed25519.Sign(priv, digest[:])...)
	poison := append([]byte(nil), valid...)
	poison[len(poison)-1] ^= 1
	v, err := NewSignatureValidator(digest, []solana.PublicKey{signer}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := v.ValidateShare(poison); ok {
		t.Fatal("invalid Ed25519 share accepted")
	}
	if got, ok := v.ValidateShare(valid); !ok || got != signer {
		t.Fatal("valid share rejected")
	}
	pubkeys, sigs, err := v.AssembleShares([][]byte{poison, valid})
	if err != nil {
		t.Fatal(err)
	}
	if len(pubkeys) != 1 || len(sigs) != 1 {
		t.Fatalf("assembled %d/%d", len(pubkeys), len(sigs))
	}
}

func TestSignatureValidatorRejectsWrongDigestAndUnauthorizedClaim(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	var signer solana.PublicKey
	copy(signer[:], pub)
	v, err := NewSignatureValidator([32]byte{9}, []solana.PublicKey{signer}, 1)
	if err != nil {
		t.Fatal(err)
	}
	share := append(append([]byte(nil), pub...), ed25519.Sign(priv, []byte("wrong"))...)
	if _, ok := v.ValidateShare(share); ok {
		t.Fatal("wrong-digest share accepted")
	}
	other, _, _ := ed25519.GenerateKey(rand.Reader)
	copy(share[:32], other)
	if got, ok := v.ValidateShare(share); ok || got != (solana.PublicKey{}) {
		t.Fatalf("unauthorized claimed pubkey returned %s, %v", got, ok)
	}
}

func TestSignatureValidatorPoisonBelowThresholdReturnsDiagnosticError(t *testing.T) {
	p1, k1, _ := ed25519.GenerateKey(rand.Reader)
	p2, _, _ := ed25519.GenerateKey(rand.Reader)
	var a1, a2 solana.PublicKey
	copy(a1[:], p1)
	copy(a2[:], p2)
	digest := [32]byte{4}
	valid := append(append([]byte(nil), p1...), ed25519.Sign(k1, digest[:])...)
	v, _ := NewSignatureValidator(digest, []solana.PublicKey{a1, a2}, 2)
	pubkeys, sigs, err := v.AssembleShares([][]byte{valid, make([]byte, shareLen)})
	if err == nil || pubkeys != nil || sigs != nil {
		t.Fatalf("AssembleShares() = %x/%x, %v; want nil results and error", pubkeys, sigs, err)
	}
	if !strings.Contains(err.Error(), "only 1 of 2") || !strings.Contains(err.Error(), "unauthorized=1") {
		t.Fatalf("diagnostic error = %v", err)
	}
}

func TestSignatureValidatorRejectsWrongShareLengths(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	var signer solana.PublicKey
	copy(signer[:], pub)
	v, _ := NewSignatureValidator([32]byte{1}, []solana.PublicKey{signer}, 1)
	for _, length := range []int{0, shareLen - 1, shareLen + 1} {
		if _, ok := v.ValidateShare(make([]byte, length)); ok {
			t.Fatalf("accepted share length %d", length)
		}
	}
}

func TestSignatureValidatorSnapshotEqualityIncludesRosterThresholdAndDigest(t *testing.T) {
	p1, _, _ := ed25519.GenerateKey(rand.Reader)
	p2, _, _ := ed25519.GenerateKey(rand.Reader)
	p3, _, _ := ed25519.GenerateKey(rand.Reader)
	var a1, a2, a3 solana.PublicKey
	copy(a1[:], p1)
	copy(a2[:], p2)
	copy(a3[:], p3)
	base, _ := NewSignatureValidator([32]byte{1}, []solana.PublicKey{a1, a2}, 1)
	reordered, _ := NewSignatureValidator([32]byte{1}, []solana.PublicKey{a2, a1}, 1)
	changedThreshold, _ := NewSignatureValidator([32]byte{1}, []solana.PublicKey{a1, a2}, 2)
	changedDigest, _ := NewSignatureValidator([32]byte{2}, []solana.PublicKey{a1, a2}, 1)
	changedSameSizeRoster, _ := NewSignatureValidator([32]byte{1}, []solana.PublicKey{a1, a3}, 1)
	if !base.MatchesSigningContext(reordered) {
		t.Fatal("equivalent signer set did not match")
	}
	if base.MatchesSigningContext(changedThreshold) || base.MatchesSigningContext(changedDigest) ||
		base.MatchesSigningContext(changedSameSizeRoster) {
		t.Fatal("changed quorum or digest matched snapshot")
	}
}

func TestSignatureValidatorNilAndCandidateLimitGuards(t *testing.T) {
	var nilValidator *SignatureValidator
	if _, _, err := nilValidator.AssembleShares(nil); err == nil {
		t.Fatal("nil validator assembled shares")
	}
	if nilValidator.Digest() != ([32]byte{}) || nilValidator.Threshold() != 0 ||
		nilValidator.MatchesSigningContext(nil) {
		t.Fatal("nil receiver contract changed")
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	var signer solana.PublicKey
	copy(signer[:], pub)
	v, _ := NewSignatureValidator([32]byte{1}, []solana.PublicKey{signer}, 1)
	if _, _, err := v.AssembleShares(make([][]byte, maxCandidatesPerAuthorizedSigner+1)); err == nil {
		t.Fatal("oversized candidate set accepted")
	}
}

func TestSignatureValidatorAssemblyIsArrivalOrderIndependent(t *testing.T) {
	p1, k1, _ := ed25519.GenerateKey(rand.Reader)
	p2, k2, _ := ed25519.GenerateKey(rand.Reader)
	var a1, a2 solana.PublicKey
	copy(a1[:], p1)
	copy(a2[:], p2)
	digest := [32]byte{7}
	s1 := append(append([]byte(nil), p1...), ed25519.Sign(k1, digest[:])...)
	s2 := append(append([]byte(nil), p2...), ed25519.Sign(k2, digest[:])...)
	v, _ := NewSignatureValidator(digest, []solana.PublicKey{a1, a2}, 2)
	pk1, sig1, err := v.AssembleShares([][]byte{s1, s2})
	if err != nil {
		t.Fatal(err)
	}
	pk2, sig2, err := v.AssembleShares([][]byte{s2, s1})
	if err != nil {
		t.Fatal(err)
	}
	if len(pk1) != 2 || !bytes.Equal(pk1[0], pk2[0]) || !bytes.Equal(pk1[1], pk2[1]) ||
		!bytes.Equal(sig1[0], sig2[0]) || !bytes.Equal(sig1[1], sig2[1]) {
		t.Fatal("Solana quorum assembly depends on arrival order")
	}
}
