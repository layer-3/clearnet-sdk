package evm

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/blockchain"
)

var rotationThresholdAddresses = []string{
	"0x0000000000000000000000000000000000000001",
	"0x0000000000000000000000000000000000000002",
	"0x0000000000000000000000000000000000000003",
}

func requireEVMMajorityThresholdError(t *testing.T, err error, threshold, signerCount int) {
	t.Helper()
	var majorityErr *blockchain.MajorityThresholdError
	if !errors.As(err, &majorityErr) {
		t.Fatalf("error = %v (%T), want *blockchain.MajorityThresholdError", err, err)
	}
	if majorityErr.Threshold != threshold || majorityErr.SignerCount != signerCount {
		t.Fatalf("MajorityThresholdError = %+v, want threshold=%d signerCount=%d", majorityErr, threshold, signerCount)
	}
}

func mustMarshalEVMRotation(t *testing.T, signers []string, threshold int) []byte {
	t.Helper()
	packed, err := json.Marshal(evmRotPacked{NewSigners: signers, NewThreshold: threshold, RotationNonce: "0"})
	if err != nil {
		t.Fatal(err)
	}
	return packed
}

func TestRotationFinalizerRejectsNonMajorityPackedTarget(t *testing.T) {
	packed := mustMarshalEVMRotation(t, rotationThresholdAddresses, 1)
	f := &RotationFinalizer{}

	_, err := f.Pack(context.Background(), [32]byte{}, rotationThresholdAddresses, 1)
	requireEVMMajorityThresholdError(t, err, 1, 3)
	err = f.Validate(context.Background(), [32]byte{}, packed, rotationThresholdAddresses, 1)
	requireEVMMajorityThresholdError(t, err, 1, 3)
	_, err = f.digestFromPacked(packed)
	requireEVMMajorityThresholdError(t, err, 1, 3)
	_, err = f.Sign(context.Background(), packed)
	requireEVMMajorityThresholdError(t, err, 1, 3)
	_, err = f.Submit(context.Background(), packed, nil)
	requireEVMMajorityThresholdError(t, err, 1, 3)
	if !strings.Contains(err.Error(), "submit target") {
		t.Fatalf("Submit error = %v, want fail-fast submit-target context", err)
	}
	_, _, err = f.VerifyRotation(context.Background(), rotationThresholdAddresses, 1)
	requireEVMMajorityThresholdError(t, err, 1, 3)
}

func TestRotationFinalizerValidateRejectsNonMajorityRequest(t *testing.T) {
	packed := mustMarshalEVMRotation(t, rotationThresholdAddresses, 2)
	err := (&RotationFinalizer{}).Validate(context.Background(), [32]byte{}, packed, rotationThresholdAddresses, 1)
	requireEVMMajorityThresholdError(t, err, 1, 3)
}

func TestRotationFinalizerAcceptsMajorityPackedTarget(t *testing.T) {
	packed := mustMarshalEVMRotation(t, rotationThresholdAddresses, 2)
	if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err != nil {
		t.Fatalf("digestFromPacked rejected 2-of-3: %v", err)
	}
}

func TestRotationFinalizerRejectsEvenNonMajorityTarget(t *testing.T) {
	evenSigners := append(append([]string(nil), rotationThresholdAddresses...), "0x0000000000000000000000000000000000000004")
	packed := mustMarshalEVMRotation(t, evenSigners, 2)
	f := &RotationFinalizer{}

	_, err := f.Pack(context.Background(), [32]byte{}, evenSigners, 2)
	requireEVMMajorityThresholdError(t, err, 2, 4)
	err = f.Validate(context.Background(), [32]byte{}, packed, rotationThresholdAddresses, 2)
	requireEVMMajorityThresholdError(t, err, 2, 4)
}

func TestRotationFinalizerRejectsOnChainInvalidSignerSets(t *testing.T) {
	tests := []struct {
		name    string
		signers []string
		want    string
	}{
		{name: "fewer than three", signers: rotationThresholdAddresses[:2], want: "at least 3 signers"},
		{name: "zero address", signers: []string{rotationThresholdAddresses[0], rotationThresholdAddresses[1], "0x0000000000000000000000000000000000000000"}, want: "zero signer address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := (&RotationFinalizer{}).Pack(context.Background(), [32]byte{}, tt.signers, 2); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Pack error = %v, want %q", err, tt.want)
			}
			packed := mustMarshalEVMRotation(t, tt.signers, 2)
			if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("digestFromPacked error = %v, want %q", err, tt.want)
			}
		})
	}
}
