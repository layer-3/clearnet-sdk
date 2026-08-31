// Package quorum contains the protocol-independent mechanics shared by
// prepared signature validators. Cryptographic recovery and wire encoding stay
// in the chain packages that define them.
package quorum

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

const MaxCandidatesPerSigner = 4

var (
	ErrNotConfigured = errors.New("quorum snapshot not configured")
	ErrZeroSigner    = errors.New("zero authorized signer")
)

type ThresholdRangeError struct {
	Threshold int
	Signers   int
}

func (e *ThresholdRangeError) Error() string {
	return fmt.Sprintf("threshold %d out of range for %d signers", e.Threshold, e.Signers)
}

type DistinctSignerError struct {
	Threshold int
	Signers   int
}

func (e *DistinctSignerError) Error() string {
	return fmt.Sprintf("threshold %d exceeds %d distinct signers", e.Threshold, e.Signers)
}

type CandidateLimitError struct {
	Candidates int
	Signers    int
}

func (e *CandidateLimitError) Error() string {
	return fmt.Sprintf("too many candidates: %d for %d authorized signers", e.Candidates, e.Signers)
}

type Stats struct {
	Invalid      int
	Unauthorized int
	Duplicate    int
}

type BelowThresholdError struct {
	Accepted  int
	Threshold int
	Stats     Stats
}

func (e *BelowThresholdError) Error() string {
	return fmt.Sprintf("only %d of %d authorized contributions (invalid=%d unauthorized=%d duplicate=%d)",
		e.Accepted, e.Threshold, e.Stats.Invalid, e.Stats.Unauthorized, e.Stats.Duplicate)
}

type Validation uint8

const (
	Invalid Validation = iota
	Unauthorized
	Authorized
)

// Decoder parses a candidate and recovers or extracts its canonical signer.
// The returned payload is the contribution used for duplicate selection and
// final encoding. It may already be canonicalized by the protocol package.
type Decoder[K comparable] func([]byte) (K, []byte, bool)

// Verifier performs protocol-specific cryptographic verification that is
// cheaper to defer until after the shared roster check. A nil Verifier means
// Decoder already completed all cryptographic validation.
type Verifier[K comparable] func(K, []byte) bool

type Entry[K comparable] struct {
	Signer  K
	Payload []byte
}

// Snapshot freezes one digest, authorized signer set, and threshold. K must be
// the canonical cryptographic signer identity, never a transport identity.
type Snapshot[K comparable] struct {
	digest     [32]byte
	authorized map[K]struct{}
	threshold  int
}

func NewSnapshot[K comparable](digest [32]byte, signers []K, threshold int, isZero func(K) bool) (*Snapshot[K], error) {
	if threshold <= 0 || threshold > len(signers) {
		return nil, &ThresholdRangeError{Threshold: threshold, Signers: len(signers)}
	}
	authorized := make(map[K]struct{}, len(signers))
	for _, signer := range signers {
		if isZero(signer) {
			return nil, ErrZeroSigner
		}
		authorized[signer] = struct{}{}
	}
	if threshold > len(authorized) {
		return nil, &DistinctSignerError{Threshold: threshold, Signers: len(authorized)}
	}
	return &Snapshot[K]{digest: digest, authorized: authorized, threshold: threshold}, nil
}

func (s *Snapshot[K]) Digest() [32]byte {
	if s == nil {
		return [32]byte{}
	}
	return s.digest
}

func (s *Snapshot[K]) Threshold() int {
	if s == nil {
		return 0
	}
	return s.threshold
}

func (s *Snapshot[K]) Authorized(signer K) bool {
	if s == nil {
		return false
	}
	_, ok := s.authorized[signer]
	return ok
}

func (s *Snapshot[K]) Matches(other *Snapshot[K]) bool {
	if s == nil || other == nil || s.digest != other.digest ||
		s.threshold != other.threshold || len(s.authorized) != len(other.authorized) {
		return false
	}
	for signer := range s.authorized {
		if !other.Authorized(signer) {
			return false
		}
	}
	return true
}

// ValidateCandidate applies the common decode, authorize, then verify order.
func (s *Snapshot[K]) ValidateCandidate(candidate []byte, decode Decoder[K], verify Verifier[K]) (K, []byte, Validation) {
	var zero K
	if s == nil || decode == nil {
		return zero, nil, Invalid
	}
	signer, payload, ok := decode(candidate)
	if !ok {
		return zero, nil, Invalid
	}
	if !s.Authorized(signer) {
		return signer, nil, Unauthorized
	}
	if verify != nil && !verify(signer, payload) {
		return zero, nil, Invalid
	}
	return signer, payload, Authorized
}

func (s *Snapshot[K]) checkCandidates(count int) error {
	if s == nil || s.threshold <= 0 {
		return ErrNotConfigured
	}
	if count > MaxCandidatesPerSigner*len(s.authorized) {
		return &CandidateLimitError{Candidates: count, Signers: len(s.authorized)}
	}
	return nil
}

// HasQuorum validates candidates only until it finds threshold distinct
// authorized signers. Unlike Assemble, it does not choose or order a
// deterministic subset because callers that only verify quorum do not consume
// that output.
func (s *Snapshot[K]) HasQuorum(candidates [][]byte, decode Decoder[K], verify Verifier[K]) error {
	if err := s.checkCandidates(len(candidates)); err != nil {
		return err
	}

	seen := make(map[K]struct{}, s.threshold)
	var stats Stats
	for _, candidate := range candidates {
		signer, _, result := s.ValidateCandidate(candidate, decode, verify)
		switch result {
		case Invalid:
			stats.Invalid++
			continue
		case Unauthorized:
			stats.Unauthorized++
			continue
		case Authorized:
		default:
			stats.Invalid++
			continue
		}
		if _, duplicate := seen[signer]; duplicate {
			stats.Duplicate++
			continue
		}
		seen[signer] = struct{}{}
		if len(seen) == s.threshold {
			return nil
		}
	}
	return &BelowThresholdError{Accepted: len(seen), Threshold: s.threshold, Stats: stats}
}

// Assemble validates the entire bounded candidate set, deduplicates by
// cryptographic signer, retains the lexicographically smallest valid payload
// for each signer, orders by signer, and returns exactly threshold entries.
func (s *Snapshot[K]) Assemble(
	candidates [][]byte,
	decode Decoder[K],
	verify Verifier[K],
	lessSigner func(K, K) bool,
) ([]Entry[K], error) {
	if err := s.checkCandidates(len(candidates)); err != nil {
		return nil, err
	}

	bySigner := make(map[K][]byte, s.threshold)
	var stats Stats
	for _, candidate := range candidates {
		signer, payload, result := s.ValidateCandidate(candidate, decode, verify)
		switch result {
		case Invalid:
			stats.Invalid++
			continue
		case Unauthorized:
			stats.Unauthorized++
			continue
		case Authorized:
		default:
			stats.Invalid++
			continue
		}
		if kept, exists := bySigner[signer]; exists {
			stats.Duplicate++
			if bytes.Compare(payload, kept) < 0 {
				bySigner[signer] = append([]byte(nil), payload...)
			}
		} else {
			bySigner[signer] = append([]byte(nil), payload...)
		}
	}
	if len(bySigner) < s.threshold {
		return nil, &BelowThresholdError{Accepted: len(bySigner), Threshold: s.threshold, Stats: stats}
	}

	signers := make([]K, 0, len(bySigner))
	for signer := range bySigner {
		signers = append(signers, signer)
	}
	sort.Slice(signers, func(i, j int) bool { return lessSigner(signers[i], signers[j]) })
	entries := make([]Entry[K], s.threshold)
	for i, signer := range signers[:s.threshold] {
		entries[i] = Entry[K]{Signer: signer, Payload: bySigner[signer]}
	}
	return entries, nil
}
