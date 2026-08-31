package sol

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"sort"

	"github.com/gagliardetto/solana-go"
)

// SignatureValidator is an immutable Solana digest, signer-set, and threshold
// snapshot for validation-first quorum collection.
type SignatureValidator struct {
	digest     [32]byte
	authorized map[solana.PublicKey]struct{}
	threshold  int
}

const maxCandidatesPerAuthorizedSigner = 4

type shareValidation uint8

const (
	shareInvalid shareValidation = iota
	shareUnauthorized
	shareAuthorized
)

// NewSignatureValidator freezes the exact digest, authorized signer set, and
// threshold for one Solana ceremony. Candidate shares use the 96-byte wire form
// public-key(32) || Ed25519-signature(64).
func NewSignatureValidator(digest [32]byte, signers []solana.PublicKey, threshold int) (*SignatureValidator, error) {
	if threshold <= 0 || threshold > len(signers) {
		return nil, fmt.Errorf("sol: threshold %d out of range for %d signers", threshold, len(signers))
	}
	authorized := make(map[solana.PublicKey]struct{}, len(signers))
	for _, signer := range signers {
		if signer.IsZero() {
			return nil, fmt.Errorf("sol: zero authorized signer")
		}
		authorized[signer] = struct{}{}
	}
	if threshold > len(authorized) {
		return nil, fmt.Errorf("sol: threshold %d exceeds %d distinct signers", threshold, len(authorized))
	}
	return &SignatureValidator{digest: digest, authorized: authorized, threshold: threshold}, nil
}

// Digest returns the prepared operation digest. A nil validator returns zero.
func (v *SignatureValidator) Digest() [32]byte {
	if v == nil {
		return [32]byte{}
	}
	return v.digest
}

// Threshold returns the prepared quorum threshold. A nil validator returns zero.
func (v *SignatureValidator) Threshold() int {
	if v == nil {
		return 0
	}
	return v.threshold
}

// MatchesSigningContext reports whether two validators bind the same digest
// and exact authorized quorum.
func (v *SignatureValidator) MatchesSigningContext(other *SignatureValidator) bool {
	if v == nil || other == nil || v.digest != other.digest ||
		v.threshold != other.threshold || len(v.authorized) != len(other.authorized) {
		return false
	}
	for signer := range v.authorized {
		if _, ok := other.authorized[signer]; !ok {
			return false
		}
	}
	return true
}

// ValidateShare verifies the claimed public key and Ed25519 signature over the
// prepared digest before returning the canonical signer identity. Every
// rejected candidate, including an unauthorized claim, returns zero,false.
func (v *SignatureValidator) ValidateShare(share []byte) (solana.PublicKey, bool) {
	pub, result := v.validateShare(share)
	if result != shareAuthorized {
		return solana.PublicKey{}, false
	}
	return pub, true
}

func (v *SignatureValidator) validateShare(share []byte) (solana.PublicKey, shareValidation) {
	if v == nil || len(share) != shareLen {
		return solana.PublicKey{}, shareInvalid
	}
	var pub solana.PublicKey
	copy(pub[:], share[:32])
	if _, ok := v.authorized[pub]; !ok {
		return pub, shareUnauthorized
	}
	if !ed25519.Verify(ed25519.PublicKey(pub[:]), v.digest[:], share[32:]) {
		return solana.PublicKey{}, shareInvalid
	}
	return pub, shareAuthorized
}

// AssembleShares filters invalid and duplicate shares, then returns a sorted,
// deterministic quorum for the Ed25519 precompile.
func (v *SignatureValidator) AssembleShares(shares [][]byte) (pubkeys, sigs [][]byte, err error) {
	if v == nil || v.threshold <= 0 {
		return nil, nil, fmt.Errorf("sol: signature validator not configured")
	}
	if len(shares) > maxCandidatesPerAuthorizedSigner*len(v.authorized) {
		return nil, nil, fmt.Errorf("sol: too many share candidates: %d for %d authorized signers", len(shares), len(v.authorized))
	}
	bySigner := make(map[solana.PublicKey][]byte, v.threshold)
	var invalid, unauthorized, duplicates int
	for _, share := range shares {
		if len(share) != shareLen {
			invalid++
			continue
		}
		var claimed solana.PublicKey
		copy(claimed[:], share[:32])
		if _, accepted := bySigner[claimed]; accepted {
			duplicates++
			continue
		}
		pub, result := v.validateShare(share)
		switch result {
		case shareInvalid:
			invalid++
			continue
		case shareUnauthorized:
			unauthorized++
			continue
		}
		bySigner[pub] = append([]byte(nil), share[32:]...)
	}
	if len(bySigner) < v.threshold {
		return nil, nil, fmt.Errorf("sol: only %d of %d authorized shares (invalid=%d unauthorized=%d duplicate=%d)",
			len(bySigner), v.threshold, invalid, unauthorized, duplicates)
	}
	pubs := make([]solana.PublicKey, 0, len(bySigner))
	for pub := range bySigner {
		pubs = append(pubs, pub)
	}
	sort.Slice(pubs, func(i, j int) bool { return bytes.Compare(pubs[i][:], pubs[j][:]) < 0 })
	pubkeys = make([][]byte, 0, v.threshold)
	sigs = make([][]byte, 0, v.threshold)
	for _, pub := range pubs[:v.threshold] {
		pubkeys = append(pubkeys, append([]byte(nil), pub[:]...))
		sigs = append(sigs, bySigner[pub])
	}
	return pubkeys, sigs, nil
}
