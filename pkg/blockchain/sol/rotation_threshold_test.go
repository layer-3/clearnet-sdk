package sol

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
)

func rotationThresholdSigners(count int) []string {
	out := make([]string, count)
	for i := range out {
		var raw [32]byte
		binary.BigEndian.PutUint64(raw[24:], uint64(i+1))
		out[i] = solana.PublicKeyFromBytes(raw[:]).String()
	}
	return out
}

func requireSolMajorityThresholdError(t *testing.T, err error, threshold, signerCount int) {
	t.Helper()
	var majorityErr *blockchain.MajorityThresholdError
	if !errors.As(err, &majorityErr) {
		t.Fatalf("error = %v (%T), want *blockchain.MajorityThresholdError", err, err)
	}
	if majorityErr.Threshold != threshold || majorityErr.SignerCount != signerCount {
		t.Fatalf("MajorityThresholdError = %+v, want threshold=%d signerCount=%d", majorityErr, threshold, signerCount)
	}
}

func mustMarshalSolRotation(t *testing.T, signers []string, threshold uint8) []byte {
	t.Helper()
	packed, err := json.Marshal(rotPacked{NewSigners: signers, NewThreshold: threshold, RotationNonce: 0})
	if err != nil {
		t.Fatal(err)
	}
	return packed
}

func TestRotationFinalizerRejectsNonMajorityPackedTarget(t *testing.T) {
	signers := rotationThresholdSigners(3)
	packed := mustMarshalSolRotation(t, signers, 1)
	f := &RotationFinalizer{}

	_, err := f.Pack(context.Background(), [32]byte{}, signers, 1)
	requireSolMajorityThresholdError(t, err, 1, 3)
	err = f.Validate(context.Background(), [32]byte{}, packed, signers, 1)
	requireSolMajorityThresholdError(t, err, 1, 3)
	_, err = f.digestFromPacked(packed)
	requireSolMajorityThresholdError(t, err, 1, 3)
	_, err = f.Sign(context.Background(), packed)
	requireSolMajorityThresholdError(t, err, 1, 3)
	_, err = f.Submit(context.Background(), packed, nil)
	requireSolMajorityThresholdError(t, err, 1, 3)
	if !strings.Contains(err.Error(), "submit target") {
		t.Fatalf("Submit error = %v, want fail-fast submit-target context", err)
	}
	_, _, err = f.VerifyRotation(context.Background(), signers, 1)
	requireSolMajorityThresholdError(t, err, 1, 3)
}

func TestRotationFinalizerValidateRejectsNonMajorityRequest(t *testing.T) {
	signers := rotationThresholdSigners(3)
	packed := mustMarshalSolRotation(t, signers, 2)
	err := (&RotationFinalizer{}).Validate(context.Background(), [32]byte{}, packed, signers, 1)
	requireSolMajorityThresholdError(t, err, 1, 3)
}

func TestRotationFinalizerAcceptsMajorityPackedTargets(t *testing.T) {
	tests := []struct {
		name      string
		count     int
		threshold uint8
	}{
		{name: "two of three", count: 3, threshold: 2},
		{name: "three of four", count: 4, threshold: 3},
		{name: "nine of program maximum", count: maxRotationSigners, threshold: 9},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed := mustMarshalSolRotation(t, rotationThresholdSigners(tt.count), tt.threshold)
			if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err != nil {
				t.Fatalf("digestFromPacked rejected %d-of-%d: %v", tt.threshold, tt.count, err)
			}
		})
	}
}

func TestRotationFinalizerRejectsEvenNonMajorityTarget(t *testing.T) {
	evenSigners := rotationThresholdSigners(4)
	packed := mustMarshalSolRotation(t, evenSigners, 2)
	f := &RotationFinalizer{}

	_, err := f.Pack(context.Background(), [32]byte{}, evenSigners, 2)
	requireSolMajorityThresholdError(t, err, 2, 4)
	err = f.Validate(context.Background(), [32]byte{}, packed, rotationThresholdSigners(3), 2)
	requireSolMajorityThresholdError(t, err, 2, 4)
}

func TestRotationFinalizerRejectsOnChainInvalidSignerSets(t *testing.T) {
	tests := []struct {
		name      string
		signers   []string
		threshold int
		want      string
	}{
		{name: "fewer than three", signers: rotationThresholdSigners(2), threshold: 2, want: "at least 3 signers"},
		{name: "default signer", signers: []string{rotationThresholdSigners(3)[0], rotationThresholdSigners(3)[1], solana.PublicKey{}.String()}, threshold: 2, want: "default signer"},
		{name: "above program maximum", signers: rotationThresholdSigners(maxRotationSigners + 1), threshold: 9, want: "too many signers"},
		{name: "above uint8 cardinality", signers: rotationThresholdSigners(256), threshold: 129, want: "too many signers"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := (&RotationFinalizer{}).Pack(context.Background(), [32]byte{}, tt.signers, tt.threshold); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Pack error = %v, want %q", err, tt.want)
			}
			packed := mustMarshalSolRotation(t, tt.signers, uint8(tt.threshold))
			if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("digestFromPacked error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestRotationFinalizerRejectsInvalidPackedSignerShape(t *testing.T) {
	valid := rotationThresholdSigners(3)
	tests := []struct {
		name    string
		signers []string
		want    string
	}{
		{name: "duplicate", signers: []string{valid[0], valid[1], valid[1]}, want: "duplicate signer"},
		{name: "malformed", signers: []string{valid[0], valid[1], "not-a-public-key"}, want: "neither 32-byte hex nor base58"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packed := mustMarshalSolRotation(t, tt.signers, 2)
			if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("digestFromPacked error = %v, want %q", err, tt.want)
			}
			if err := (&RotationFinalizer{}).Validate(context.Background(), [32]byte{}, packed, valid, 2); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate error = %v, want %q", err, tt.want)
			}
		})
	}
}
