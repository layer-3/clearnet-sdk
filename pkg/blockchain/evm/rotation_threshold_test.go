package evm

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRotationFinalizerRejectsNonMajorityPackedTarget(t *testing.T) {
	packed, err := json.Marshal(evmRotPacked{
		NewSigners: []string{
			"0x0000000000000000000000000000000000000001",
			"0x0000000000000000000000000000000000000002",
			"0x0000000000000000000000000000000000000003",
		},
		NewThreshold: 1,
		SignerNonce:  "0",
	})
	if err != nil {
		t.Fatal(err)
	}

	f := &RotationFinalizer{}
	if _, err := f.Pack(context.Background(), [32]byte{}, []string{
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
		"0x0000000000000000000000000000000000000003",
	}, 1); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Pack error = %v, want strict-majority rejection", err)
	}
	if err := f.Validate(context.Background(), [32]byte{}, packed, []string{
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
		"0x0000000000000000000000000000000000000003",
	}, 1); err == nil || !strings.Contains(err.Error(), "strict majority") {
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
	if _, _, err := f.VerifyRotation(context.Background(), []string{
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
		"0x0000000000000000000000000000000000000003",
	}, 1); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("VerifyRotation error = %v, want strict-majority rejection", err)
	}
}

func TestRotationFinalizerAcceptsMajorityPackedTarget(t *testing.T) {
	packed, err := json.Marshal(evmRotPacked{
		NewSigners: []string{
			"0x0000000000000000000000000000000000000001",
			"0x0000000000000000000000000000000000000002",
			"0x0000000000000000000000000000000000000003",
		},
		NewThreshold: 2,
		SignerNonce:  "0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (&RotationFinalizer{}).digestFromPacked(packed); err != nil {
		t.Fatalf("digestFromPacked rejected 2-of-3: %v", err)
	}
}

func TestRotationFinalizerRejectsEvenNonMajorityTarget(t *testing.T) {
	evenSigners := []string{
		"0x0000000000000000000000000000000000000001",
		"0x0000000000000000000000000000000000000002",
		"0x0000000000000000000000000000000000000003",
		"0x0000000000000000000000000000000000000004",
	}
	packed, err := json.Marshal(evmRotPacked{
		NewSigners:   evenSigners,
		NewThreshold: 2,
		SignerNonce:  "0",
	})
	if err != nil {
		t.Fatal(err)
	}

	f := &RotationFinalizer{}
	if _, err := f.Pack(context.Background(), [32]byte{}, evenSigners, 2); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Pack error = %v, want strict-majority rejection", err)
	}
	if err := f.Validate(context.Background(), [32]byte{}, packed, evenSigners[:3], 2); err == nil || !strings.Contains(err.Error(), "strict majority") {
		t.Fatalf("Validate packed 2-of-4 error = %v, want strict-majority rejection", err)
	}
}
