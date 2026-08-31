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

func (v *SignatureValidator) Digest() [32]byte {
	if v == nil {
		return [32]byte{}
	}
	return v.digest
}

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
// prepared digest before returning the canonical signer identity.
func (v *SignatureValidator) ValidateShare(share []byte) (solana.PublicKey, bool) {
	if v == nil || len(share) != shareLen {
		return solana.PublicKey{}, false
	}
	var pub solana.PublicKey
	copy(pub[:], share[:32])
	if _, ok := v.authorized[pub]; !ok {
		return solana.PublicKey{}, false
	}
	if !ed25519.Verify(ed25519.PublicKey(pub[:]), v.digest[:], share[32:]) {
		return solana.PublicKey{}, false
	}
	return pub, true
}

// AssembleShares filters invalid and duplicate shares, then returns a sorted,
// deterministic quorum for the Ed25519 precompile.
func (v *SignatureValidator) AssembleShares(shares [][]byte) (pubkeys, sigs [][]byte, err error) {
	if v == nil || v.threshold <= 0 {
		return nil, nil, fmt.Errorf("sol: signature validator not configured")
	}
	bySigner := make(map[solana.PublicKey][]byte, v.threshold)
	for _, share := range shares {
		pub, ok := v.ValidateShare(share)
		if !ok {
			continue
		}
		sig := share[32:]
		if kept, exists := bySigner[pub]; !exists || bytes.Compare(sig, kept) < 0 {
			bySigner[pub] = append([]byte(nil), sig...)
		}
	}
	if len(bySigner) < v.threshold {
		return nil, nil, fmt.Errorf("sol: only %d of %d authorized shares", len(bySigner), v.threshold)
	}
	pubs := make([]solana.PublicKey, 0, len(bySigner))
	for pub := range bySigner {
		pubs = append(pubs, pub)
	}
	sort.Slice(pubs, func(i, j int) bool { return bytes.Compare(pubs[i][:], pubs[j][:]) < 0 })
	for _, pub := range pubs[:v.threshold] {
		pubkeys = append(pubkeys, append([]byte(nil), pub[:]...))
		sigs = append(sigs, bySigner[pub])
	}
	return pubkeys, sigs, nil
}
