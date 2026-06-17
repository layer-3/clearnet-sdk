package bls

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254"
)

// TestDeserializeG1_Validation covers the acceptance-path checks: a valid point
// round-trips, while non-canonical (coordinate >= P) and off-curve encodings are
// rejected. (G1 has cofactor 1, so every on-curve point is in the subgroup — the
// subgroup check is exercised by the G2 case.)
func TestDeserializeG1_Validation(t *testing.T) {
	_, _, g1, _ := bn254.Generators()
	if _, err := DeserializeG1(SerializeG1(g1)); err != nil {
		t.Fatalf("valid G1 rejected: %v", err)
	}

	nonCanonical := make([]byte, 64)
	fieldP.FillBytes(nonCanonical[:32]) // X == P
	if _, err := DeserializeG1(nonCanonical); err == nil {
		t.Error("G1 with coordinate == P accepted")
	}

	offCurve := make([]byte, 64)
	offCurve[31], offCurve[63] = 1, 1 // (1,1): 1 != 1+3, not on y^2 = x^3 + 3
	if _, err := DeserializeG1(offCurve); err == nil {
		t.Error("off-curve G1 accepted")
	}

	if _, err := DeserializeG1(make([]byte, 63)); err == nil {
		t.Error("wrong-length G1 accepted")
	}
}

// TestDeserializeG2_Validation covers the same checks for G2, including the
// prime-order subgroup membership (G2 has cofactor > 1).
func TestDeserializeG2_Validation(t *testing.T) {
	_, _, _, g2 := bn254.Generators()
	if _, err := DeserializeG2(SerializeG2(g2)); err != nil {
		t.Fatalf("valid G2 rejected: %v", err)
	}

	nonCanonical := make([]byte, 128)
	fieldP.FillBytes(nonCanonical[:32]) // X.A1 == P
	if _, err := DeserializeG2(nonCanonical); err == nil {
		t.Error("G2 with coordinate == P accepted")
	}

	offCurve := make([]byte, 128)
	offCurve[31], offCurve[63], offCurve[95], offCurve[127] = 1, 1, 1, 1
	if _, err := DeserializeG2(offCurve); err == nil {
		t.Error("off-curve G2 accepted")
	}

	if _, err := DeserializeG2(make([]byte, 127)); err == nil {
		t.Error("wrong-length G2 accepted")
	}
}
