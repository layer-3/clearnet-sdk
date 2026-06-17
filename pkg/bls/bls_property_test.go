package bls

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"

	bn254 "github.com/consensys/gnark-crypto/ecc/bn254"
)

func randBytes32Det(rng *rand.Rand) [32]byte {
	var b [32]byte
	rng.Read(b[:])
	return b
}

// TestProperty_BLS_SignVerifyRoundTrip asserts invariant B1: for any
// keypair and any 32-byte message, Sign + Verify with the matching
// public G2 key returns true, and with a wrong key returns false.
//
// Mutation-check 2026-04-18: negated the pairing check in Verify
// (swapped the sign of negHm) — test failed as signatures no longer
// verified; restored.
func TestProperty_BLS_SignVerifyRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x4_0B01))
	for trial := 0; trial < 20; trial++ {
		// BLS operations are slow — trial count kept modest.
		seed := randBytes32Det(rng)
		kp := KeyPairFromSeed(seed[:])
		msg := randBytes32Det(rng)

		sig, err := Sign(&kp.Secret, msg)
		if err != nil {
			t.Fatalf("trial=%d: sign: %v", trial, err)
		}

		ok, err := Verify(sig, kp.PublicG2, msg)
		if err != nil {
			t.Fatalf("trial=%d: verify: %v", trial, err)
		}
		if !ok {
			t.Fatalf("trial=%d: valid signature did not verify", trial)
		}

		// Different message must not verify under same sig + key.
		other := randBytes32Det(rng)
		if other == msg {
			other[0] ^= 1
		}
		ok, err = Verify(sig, kp.PublicG2, other)
		if err != nil {
			t.Fatalf("trial=%d: verify wrong msg: %v", trial, err)
		}
		if ok {
			t.Fatalf("trial=%d: signature verified against wrong message", trial)
		}

		// Different key must not verify under same sig + msg.
		otherSeed := randBytes32Det(rng)
		if otherSeed == seed {
			otherSeed[0] ^= 1
		}
		otherKp := KeyPairFromSeed(otherSeed[:])
		ok, err = Verify(sig, otherKp.PublicG2, msg)
		if err != nil {
			t.Fatalf("trial=%d: verify wrong key: %v", trial, err)
		}
		if ok {
			t.Fatalf("trial=%d: signature verified against wrong public key", trial)
		}
	}
}

// TestProperty_BLS_KeyPairFromSeed_Deterministic asserts invariant B2:
// same seed → same keypair (scalar, G1, G2 all equal).
func TestProperty_BLS_KeyPairFromSeed_Deterministic(t *testing.T) {
	rng := rand.New(rand.NewSource(0x4_0B02))
	for trial := 0; trial < 50; trial++ {
		n := 1 + rng.Intn(128)
		seed := make([]byte, n)
		rng.Read(seed)

		a := KeyPairFromSeed(seed)
		b := KeyPairFromSeed(seed)

		if !a.Secret.Equal(&b.Secret) {
			t.Fatalf("trial=%d: secret scalars differ", trial)
		}
		if !a.PublicG1.Equal(&b.PublicG1) {
			t.Fatalf("trial=%d: G1 pubkeys differ", trial)
		}
		if !a.PublicG2.Equal(&b.PublicG2) {
			t.Fatalf("trial=%d: G2 pubkeys differ", trial)
		}
	}
}

// TestProperty_BLS_KeyPairFromSeed_Sensitivity asserts B3: different
// seeds produce different keypairs. Cross-check: serialised G1 points
// are distinct across random seeds.
func TestProperty_BLS_KeyPairFromSeed_Sensitivity(t *testing.T) {
	rng := rand.New(rand.NewSource(0x4_0B03))
	seenG1 := map[[64]byte]int{} // G1 is 64 bytes uncompressed via SerializeG1
	for trial := 0; trial < 100; trial++ {
		var seed [32]byte
		rng.Read(seed[:])
		kp := KeyPairFromSeed(seed[:])
		raw := SerializeG1(kp.PublicG1)
		var key [64]byte
		if len(raw) != 64 {
			// If serialization changes shape, fail loudly rather than truncate.
			t.Fatalf("trial=%d: unexpected G1 serialisation length %d", trial, len(raw))
		}
		copy(key[:], raw)
		if prior, ok := seenG1[key]; ok && prior != trial {
			t.Fatalf("trial=%d: seed collision — same G1 as trial %d", trial, prior)
		}
		seenG1[key] = trial
	}
}

// TestProperty_BLS_AggregateEmpty_Rejects asserts invariant B4: both
// AggregateG1 and AggregateG2 reject empty input with
// ErrEmptyAggregation. Also tests nil input (separate from empty slice).
func TestProperty_BLS_AggregateEmpty_Rejects(t *testing.T) {
	// G1
	if _, err := AggregateG1(nil); !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("AggregateG1(nil): got %v, want ErrEmptyAggregation", err)
	}
	if _, err := AggregateG1([]bn254.G1Affine{}); !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("AggregateG1([]): got %v, want ErrEmptyAggregation", err)
	}
	// G2
	if _, err := AggregateG2(nil); !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("AggregateG2(nil): got %v, want ErrEmptyAggregation", err)
	}
	if _, err := AggregateG2([]bn254.G2Affine{}); !errors.Is(err, ErrEmptyAggregation) {
		t.Fatalf("AggregateG2([]): got %v, want ErrEmptyAggregation", err)
	}

	// Single-point aggregations succeed (not empty).
	rng := rand.New(rand.NewSource(0x4_0B04))
	var seed [32]byte
	rng.Read(seed[:])
	kp := KeyPairFromSeed(seed[:])
	if _, err := AggregateG1([]bn254.G1Affine{kp.PublicG1}); err != nil {
		t.Fatalf("single-point G1 aggregate: %v", err)
	}
	if _, err := AggregateG2([]bn254.G2Affine{kp.PublicG2}); err != nil {
		t.Fatalf("single-point G2 aggregate: %v", err)
	}
}

// TestProperty_BLS_SerializeG1G2_RoundTrip asserts invariant B7:
// DeserializeG1(SerializeG1(p)) == p across random G1 points derived
// from random scalars. Same for G2.
func TestProperty_BLS_SerializeG1G2_RoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(0x4_0B07))
	for trial := 0; trial < 50; trial++ {
		var seed [32]byte
		rng.Read(seed[:])
		kp := KeyPairFromSeed(seed[:])

		// G1 round-trip
		raw := SerializeG1(kp.PublicG1)
		back, err := DeserializeG1(raw)
		if err != nil {
			t.Fatalf("trial=%d: DeserializeG1: %v", trial, err)
		}
		if !back.Equal(&kp.PublicG1) {
			t.Fatalf("trial=%d: G1 round-trip mismatch", trial)
		}

		// Re-serialise and check byte-equality.
		raw2 := SerializeG1(back)
		if !bytes.Equal(raw, raw2) {
			t.Fatalf("trial=%d: G1 serialisation not idempotent", trial)
		}

		// G2 round-trip
		rawG2 := SerializeG2(kp.PublicG2)
		backG2, err := DeserializeG2(rawG2)
		if err != nil {
			t.Fatalf("trial=%d: DeserializeG2: %v", trial, err)
		}
		if !backG2.Equal(&kp.PublicG2) {
			t.Fatalf("trial=%d: G2 round-trip mismatch", trial)
		}
		rawG2_2 := SerializeG2(backG2)
		if !bytes.Equal(rawG2, rawG2_2) {
			t.Fatalf("trial=%d: G2 serialisation not idempotent", trial)
		}
	}
}
