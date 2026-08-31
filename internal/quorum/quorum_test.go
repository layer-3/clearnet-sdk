package quorum

import (
	"bytes"
	"errors"
	"testing"
)

func TestSnapshotFreezesSigningContext(t *testing.T) {
	digest := [32]byte{1}
	signers := []byte{3, 1, 2}
	snapshot, err := NewSnapshot(digest, signers, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	signers[0] = 9

	same, err := NewSnapshot(digest, []byte{2, 3, 1}, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	differentDigest, err := NewSnapshot([32]byte{2}, []byte{1, 2, 3}, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}

	if !snapshot.Authorized(3) || snapshot.Authorized(9) {
		t.Fatal("snapshot roster changed after caller mutated its signer slice")
	}
	if !snapshot.Matches(same) || snapshot.Matches(differentDigest) {
		t.Fatal("snapshot context comparison did not bind digest, roster, and threshold")
	}
}

func TestSnapshotConstructorGuards(t *testing.T) {
	tests := []struct {
		name      string
		signers   []byte
		threshold int
		want      any
	}{
		{name: "threshold range", signers: []byte{1}, threshold: 0, want: &ThresholdRangeError{}},
		{name: "zero signer", signers: []byte{1, 0}, threshold: 1, want: ErrZeroSigner},
		{name: "distinct signer collapse", signers: []byte{1, 1}, threshold: 2, want: &DistinctSignerError{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSnapshot([32]byte{1}, tc.signers, tc.threshold, func(signer byte) bool { return signer == 0 })
			switch want := tc.want.(type) {
			case *ThresholdRangeError:
				var got *ThresholdRangeError
				if !errors.As(err, &got) {
					t.Fatalf("NewSnapshot() error = %v, want ThresholdRangeError", err)
				}
			case *DistinctSignerError:
				var got *DistinctSignerError
				if !errors.As(err, &got) {
					t.Fatalf("NewSnapshot() error = %v, want DistinctSignerError", err)
				}
			case error:
				if !errors.Is(err, want) {
					t.Fatalf("NewSnapshot() error = %v, want %v", err, want)
				}
			}
		})
	}
}

func TestValidateCandidateAuthorizesBeforeDeferredVerification(t *testing.T) {
	snapshot, err := NewSnapshot([32]byte{1}, []byte{1}, 1, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	decode := func(candidate []byte) (byte, []byte, bool) { return candidate[0], candidate[1:], true }
	verifyCalls := 0
	verify := func(_ byte, payload []byte) bool {
		verifyCalls++
		return bytes.Equal(payload, []byte{7})
	}

	if _, _, result := snapshot.ValidateCandidate([]byte{9, 7}, decode, verify); result != Unauthorized {
		t.Fatalf("unauthorized candidate result = %v", result)
	}
	if verifyCalls != 0 {
		t.Fatal("cryptographic verifier ran for an unauthorized signer")
	}
	if signer, _, result := snapshot.ValidateCandidate([]byte{1, 7}, decode, verify); result != Authorized || signer != 1 {
		t.Fatalf("authorized candidate = signer %d, result %v", signer, result)
	}
	if verifyCalls != 1 {
		t.Fatalf("cryptographic verifier calls = %d, want 1", verifyCalls)
	}
}

func TestHasQuorumStopsAtThreshold(t *testing.T) {
	snapshot, err := NewSnapshot([32]byte{1}, []byte{1, 2, 3}, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	decodeCalls := 0
	decode := func(candidate []byte) (byte, []byte, bool) {
		decodeCalls++
		return candidate[0], nil, true
	}

	if err := snapshot.HasQuorum([][]byte{{1}, {2}, {3}}, decode, nil); err != nil {
		t.Fatal(err)
	}
	if decodeCalls != 2 {
		t.Fatalf("decoder calls = %d, want 2; verification scanned beyond threshold", decodeCalls)
	}
}

func TestHasQuorumPreservesFailureDiagnostics(t *testing.T) {
	snapshot, err := NewSnapshot([32]byte{1}, []byte{1, 2}, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	decode := func(candidate []byte) (byte, []byte, bool) {
		if len(candidate) != 1 {
			return 0, nil, false
		}
		return candidate[0], nil, true
	}
	err = snapshot.HasQuorum([][]byte{{1}, {1}, {9}, {}}, decode, nil)
	var below *BelowThresholdError
	if !errors.As(err, &below) {
		t.Fatalf("HasQuorum() error = %v, want BelowThresholdError", err)
	}
	if below.Accepted != 1 || below.Stats != (Stats{Invalid: 1, Unauthorized: 1, Duplicate: 1}) {
		t.Fatalf("unexpected rejection accounting: %+v", below)
	}
}

func TestAssembleUsesOneDeterministicDuplicateRule(t *testing.T) {
	snapshot, err := NewSnapshot([32]byte{1}, []byte{1, 2, 3}, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	decode := func(candidate []byte) (byte, []byte, bool) {
		if len(candidate) != 2 {
			return 0, nil, false
		}
		return candidate[0], candidate[1:], true
	}
	less := func(a, b byte) bool { return a < b }
	candidates := [][]byte{{9}, {7, 1}, {2, 9}, {1, 8}, {1, 3}, {3, 1}}

	forward, err := snapshot.Assemble(candidates, decode, nil, less)
	if err != nil {
		t.Fatal(err)
	}
	for i, j := 0, len(candidates)-1; i < j; i, j = i+1, j-1 {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	}
	reverse, err := snapshot.Assemble(candidates, decode, nil, less)
	if err != nil {
		t.Fatal(err)
	}

	if len(forward) != 2 || forward[0].Signer != 1 || forward[1].Signer != 2 {
		t.Fatalf("unexpected sorted, trimmed quorum: %+v", forward)
	}
	if !bytes.Equal(forward[0].Payload, []byte{3}) {
		t.Fatalf("duplicate tie-break kept %x, want lexicographically smallest payload 03", forward[0].Payload)
	}
	for i := range forward {
		if forward[i].Signer != reverse[i].Signer || !bytes.Equal(forward[i].Payload, reverse[i].Payload) {
			t.Fatalf("candidate order changed quorum: forward=%+v reverse=%+v", forward, reverse)
		}
	}
}

func TestAssembleReportsSharedRejectionTallies(t *testing.T) {
	snapshot, err := NewSnapshot([32]byte{1}, []byte{1, 2}, 2, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	decode := func(candidate []byte) (byte, []byte, bool) {
		if len(candidate) != 2 {
			return 0, nil, false
		}
		return candidate[0], candidate[1:], true
	}
	_, err = snapshot.Assemble([][]byte{{1, 3}, {1, 4}, {9, 1}, {0}}, decode, nil, func(a, b byte) bool { return a < b })
	var below *BelowThresholdError
	if !errors.As(err, &below) {
		t.Fatalf("Assemble() error = %v, want BelowThresholdError", err)
	}
	if below.Accepted != 1 || below.Stats != (Stats{Invalid: 1, Unauthorized: 1, Duplicate: 1}) {
		t.Fatalf("unexpected rejection accounting: %+v", below)
	}
}

func TestAssembleEnforcesCandidateCap(t *testing.T) {
	snapshot, err := NewSnapshot([32]byte{1}, []byte{1}, 1, func(signer byte) bool { return signer == 0 })
	if err != nil {
		t.Fatal(err)
	}
	_, err = snapshot.Assemble(make([][]byte, MaxCandidatesPerSigner+1), nil, nil, func(a, b byte) bool { return a < b })
	var limit *CandidateLimitError
	if !errors.As(err, &limit) || limit.Candidates != MaxCandidatesPerSigner+1 || limit.Signers != 1 {
		t.Fatalf("Assemble() error = %v, want candidate limit details", err)
	}
}
