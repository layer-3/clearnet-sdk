package sol

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gagliardetto/solana-go"
)

func rotationThresholdSigners(count int) []string {
	out := make([]string, count)
	for i := range out {
		var raw [32]byte
		raw[31] = byte(i + 1)
		out[i] = solana.PublicKeyFromBytes(raw[:]).String()
	}
	return out
}

func TestRotationFinalizerRejectsNonMajorityPackedTarget(t *testing.T) {
	packed, err := json.Marshal(rotPacked{
		NewSigners:   rotationThresholdSigners(3),
		NewThreshold: 1,
		SignerNonce:  0,
	})
	if err != nil {
		t.Fatal(err)
	}

	f := &RotationFinalizer{}
	if _, err := f.Pack(context.Background(), [32]byte{}, rotationThresholdSigners(3), 1); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Pack error = %v, want strict-majority rejection", err)
	}
	if err := f.Validate(context.Background(), [32]byte{}, packed, rotationThresholdSigners(3), 1); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Validate error = %v, want strict-majority rejection", err)
	}
	if _, err := f.digestFromPacked(packed); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("digestFromPacked error = %v, want strict-majority rejection", err)
	}
	if _, err := f.Sign(context.Background(), packed); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Sign error = %v, want strict-majority rejection", err)
	}
	if _, err := f.Submit(context.Background(), packed, nil); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Submit error = %v, want strict-majority rejection", err)
	}
	if _, _, err := f.VerifyRotation(context.Background(), rotationThresholdSigners(3), 1); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("VerifyRotation error = %v, want strict-majority rejection", err)
	}
}

func TestRotationFinalizerAcceptsMajorityPackedTarget(t *testing.T) {
	packed, err := json.Marshal(rotPacked{
		NewSigners:   rotationThresholdSigners(3),
		NewThreshold: 2,
		SignerNonce:  0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err != nil {
		t.Fatalf("digestFromPacked rejected 2-of-3: %v", err)
	}
}
