package sol

import (
	"bytes"
	"crypto/ed25519"
	"errors"
	"fmt"

	"github.com/gagliardetto/solana-go"
	internalquorum "github.com/layer-3/clearnet-sdk/internal/quorum"
)

// SignatureValidator is an immutable Solana digest, signer-set, and threshold
// snapshot for validation-first quorum collection.
type SignatureValidator struct {
	snapshot *internalquorum.Snapshot[solana.PublicKey]
}

const maxCandidatesPerAuthorizedSigner = internalquorum.MaxCandidatesPerSigner

// NewSignatureValidator freezes the exact digest, authorized signer set, and
// threshold for one Solana ceremony. Candidate shares use the 96-byte wire form
// public-key(32) || Ed25519-signature(64).
func NewSignatureValidator(digest [32]byte, signers []solana.PublicKey, threshold int) (*SignatureValidator, error) {
	snapshot, err := internalquorum.NewSnapshot(digest, signers, threshold, func(signer solana.PublicKey) bool {
		return signer.IsZero()
	})
	if err != nil {
		return nil, fmt.Errorf("sol: %w", err)
	}
	return &SignatureValidator{snapshot: snapshot}, nil
}

// Digest returns the prepared operation digest. A nil validator returns zero.
func (v *SignatureValidator) Digest() [32]byte {
	if v == nil {
		return [32]byte{}
	}
	return v.snapshot.Digest()
}

// Threshold returns the prepared quorum threshold. A nil validator returns zero.
func (v *SignatureValidator) Threshold() int {
	if v == nil {
		return 0
	}
	return v.snapshot.Threshold()
}

// MatchesSigningContext reports whether two validators bind the same digest
// and exact authorized quorum.
func (v *SignatureValidator) MatchesSigningContext(other *SignatureValidator) bool {
	return v != nil && other != nil && v.snapshot.Matches(other.snapshot)
}

// ValidateShare verifies the claimed public key and Ed25519 signature over the
// prepared digest before returning the canonical signer identity. Every
// rejected candidate, including an unauthorized claim, returns zero,false.
func (v *SignatureValidator) ValidateShare(share []byte) (solana.PublicKey, bool) {
	if v == nil {
		return solana.PublicKey{}, false
	}
	pub, _, result := v.snapshot.ValidateCandidate(share, v.decodeShare, v.verifyShare)
	if result != internalquorum.Authorized {
		return solana.PublicKey{}, false
	}
	return pub, true
}

func (v *SignatureValidator) decodeShare(share []byte) (solana.PublicKey, []byte, bool) {
	if v == nil || len(share) != shareLen {
		return solana.PublicKey{}, nil, false
	}
	var pub solana.PublicKey
	copy(pub[:], share[:32])
	return pub, share[32:], true
}

func (v *SignatureValidator) verifyShare(pub solana.PublicKey, signature []byte) bool {
	digest := v.snapshot.Digest()
	return ed25519.Verify(ed25519.PublicKey(pub[:]), digest[:], signature)
}

// AssembleShares filters invalid and duplicate shares, then returns a sorted,
// deterministic quorum for the Ed25519 precompile.
func (v *SignatureValidator) AssembleShares(shares [][]byte) (pubkeys, sigs [][]byte, err error) {
	var snapshot *internalquorum.Snapshot[solana.PublicKey]
	if v != nil {
		snapshot = v.snapshot
	}
	entries, assembleErr := snapshot.Assemble(shares, v.decodeShare, v.verifyShare, func(a, b solana.PublicKey) bool {
		return bytes.Compare(a[:], b[:]) < 0
	})
	if assembleErr != nil {
		var limit *internalquorum.CandidateLimitError
		var below *internalquorum.BelowThresholdError
		switch {
		case errors.Is(assembleErr, internalquorum.ErrNotConfigured):
			return nil, nil, fmt.Errorf("sol: signature validator not configured")
		case errors.As(assembleErr, &limit):
			return nil, nil, fmt.Errorf("sol: too many share candidates: %d for %d authorized signers", limit.Candidates, limit.Signers)
		case errors.As(assembleErr, &below):
			return nil, nil, fmt.Errorf("sol: only %d of %d authorized shares (invalid=%d unauthorized=%d duplicate=%d)",
				below.Accepted, below.Threshold, below.Stats.Invalid, below.Stats.Unauthorized, below.Stats.Duplicate)
		default:
			return nil, nil, assembleErr
		}
	}
	pubkeys = make([][]byte, len(entries))
	sigs = make([][]byte, len(entries))
	for i, entry := range entries {
		pubkeys[i] = append([]byte(nil), entry.Signer[:]...)
		sigs[i] = entry.Payload
	}
	return pubkeys, sigs, nil
}
