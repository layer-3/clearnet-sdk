package cborx_test

import (
	"bytes"
	"encoding/hex"
	"math"
	"testing"
	"time"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

func TestTime_ZeroIsPositiveIntZero(t *testing.T) {
	tm := cborx.Time{V: time.Unix(0, 0).UTC()}
	var buf bytes.Buffer
	if err := tm.MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if hex.EncodeToString(buf.Bytes()) != "00" {
		t.Fatalf("zero time = %x, want 00", buf.Bytes())
	}
}

func TestTime_RoundTrip(t *testing.T) {
	cases := []time.Time{
		time.Unix(0, 0).UTC(),                         // epoch
		time.Unix(1_700_000_000, 123_456_789).UTC(),   // ~2023
		time.Unix(0, -1).UTC(),                        // 1ns before epoch
		time.Unix(-1_000_000, 0).UTC(),                // pre-1970
		time.Unix(7_000_000_000, 999_999_999).UTC(),   // year ~2191 (within int64 ns)
		time.Unix(-7_000_000_000, -999_999_999).UTC(), // year ~1748
		time.Unix(0, math.MaxInt64).UTC(),             // highest representable Unix ns
		time.Unix(0, math.MinInt64).UTC(),             // lowest representable Unix ns
	}
	for _, tm := range cases {
		var buf bytes.Buffer
		if err := (cborx.Time{V: tm}).MarshalCBOR(&buf); err != nil {
			t.Fatalf("marshal %s: %v", tm, err)
		}

		var got cborx.Time
		if err := got.UnmarshalCBOR(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatalf("unmarshal %s: %v", tm, err)
		}
		if !got.V.Equal(tm) {
			t.Errorf("round-trip %s: got %s", tm, got.V)
		}

		// Idempotence.
		var buf2 bytes.Buffer
		if err := got.MarshalCBOR(&buf2); err != nil {
			t.Fatalf("re-marshal: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
			t.Errorf("idempotence %s: %x vs %x", tm, buf.Bytes(), buf2.Bytes())
		}
	}
}

func TestTime_RejectMarshalOutsideUnixNanoRange(t *testing.T) {
	cases := map[string]time.Time{
		"before_min": time.Unix(0, math.MinInt64).Add(-time.Nanosecond).UTC(),
		"after_max":  time.Unix(0, math.MaxInt64).Add(time.Nanosecond).UTC(),
	}
	for name, tm := range cases {
		var buf bytes.Buffer
		if err := (cborx.Time{V: tm}).MarshalCBOR(&buf); err == nil {
			t.Fatalf("%s: expected marshal rejection for %s", name, tm)
		}
	}
}

func TestTime_RejectFloat(t *testing.T) {
	// half-precision 1.0 = 0xf9 0x3c 0x00
	raw := []byte{0xf9, 0x3c, 0x00}
	var got cborx.Time
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of float")
	}
}

func TestTime_RejectTag(t *testing.T) {
	// tag 0 (text date-time) wrapping "1970-01-01T00:00:00Z" — RFC 8949
	// allows this, but the cborx.Time adapter is bare-int only per
	// ADR-009 §3 and docs/specs/cbor.md §2.
	raw := []byte{0xc0, 0x74, '1', '9', '7', '0', '-', '0', '1', '-', '0', '1',
		'T', '0', '0', ':', '0', '0', ':', '0', '0', 'Z'}
	var got cborx.Time
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of tag-0 time string")
	}
}
