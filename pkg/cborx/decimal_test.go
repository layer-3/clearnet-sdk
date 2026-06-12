package cborx_test

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
	"github.com/layer-3/clearnet-sdk/pkg/decimal"
)

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	if err != nil {
		t.Fatalf("decimal parse %q: %v", s, err)
	}
	return d
}

func TestDecimal_TagStructure(t *testing.T) {
	// 1.5 = mantissa 15, exponent -1.
	d := mustDec(t, "1.5")
	var buf bytes.Buffer
	if err := (cborx.Decimal{V: d}).MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Expected: tag 4 (0xc4), array length 2 (0x82), exponent -1 (0x20),
	// mantissa BigInt tag 2 length 1 value 0x0f.
	want := "c482" + "20" + "c2410f"
	if hex.EncodeToString(buf.Bytes()) != want {
		t.Fatalf("1.5 encoding = %x, want %s", buf.Bytes(), want)
	}
}

func TestDecimal_RoundTripAtVariousExponents(t *testing.T) {
	cases := []struct {
		mantissa int64
		exp      int32
	}{
		{0, 0},
		{1, 0},
		{-1, 0},
		{123456, 6},
		{123456, -6},
		{1, -18},
		{1, 18},
		{-999999999999, -18},
		{1, -1_000_000}, // far-negative exponent inside int32
	}
	for _, c := range cases {
		d := decimal.NewFromBigInt(big.NewInt(c.mantissa), c.exp)
		var buf bytes.Buffer
		if err := (cborx.Decimal{V: d}).MarshalCBOR(&buf); err != nil {
			t.Fatalf("marshal %d*10^%d: %v", c.mantissa, c.exp, err)
		}

		var got cborx.Decimal
		if err := got.UnmarshalCBOR(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatalf("unmarshal %d*10^%d: %v", c.mantissa, c.exp, err)
		}
		if got.V.Exponent() != c.exp {
			t.Errorf("exp for %d*10^%d: got %d, want %d", c.mantissa, c.exp, got.V.Exponent(), c.exp)
		}
		if got.V.Coefficient().Cmp(big.NewInt(c.mantissa)) != 0 {
			t.Errorf("mantissa for %d*10^%d: got %s", c.mantissa, c.exp, got.V.Coefficient())
		}

		// Idempotence.
		var buf2 bytes.Buffer
		if err := got.MarshalCBOR(&buf2); err != nil {
			t.Fatalf("re-marshal: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
			t.Errorf("idempotence %d*10^%d: %x vs %x", c.mantissa, c.exp, buf.Bytes(), buf2.Bytes())
		}
	}
}

func TestDecimal_LargeMantissaRoundTrip(t *testing.T) {
	// 10^40 * 10^-18 — mantissa doesn't fit in int64.
	mantissa := new(big.Int).Exp(big.NewInt(10), big.NewInt(40), nil)
	d := decimal.NewFromBigInt(mantissa, -18)

	var buf bytes.Buffer
	if err := (cborx.Decimal{V: d}).MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got cborx.Decimal
	if err := got.UnmarshalCBOR(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.V.Coefficient().Cmp(mantissa) != 0 {
		t.Errorf("mantissa = %s, want %s", got.V.Coefficient(), mantissa)
	}
	if got.V.Exponent() != -18 {
		t.Errorf("exp = %d, want -18", got.V.Exponent())
	}
}

func TestDecimal_RejectWrongTag(t *testing.T) {
	// tag 5 instead of 4.
	raw := []byte{0xc5, 0x82, 0x00, 0xc2, 0x40}
	var got cborx.Decimal
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of tag 5")
	}
}

func TestDecimal_RejectWrongArity(t *testing.T) {
	// tag 4, array length 1 instead of 2.
	raw := []byte{0xc4, 0x81, 0x00}
	var got cborx.Decimal
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of 1-element array")
	}
}

func TestDecimal_RejectNonIntegerExponent(t *testing.T) {
	// tag 4, [exponent = text "0", mantissa = 0]. Exponents must be
	// bare signed integers, never strings or floats.
	raw := []byte{0xc4, 0x82, 0x61, '0', 0xc2, 0x40}
	var got cborx.Decimal
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of non-integer exponent")
	}
}

func TestDecimal_RejectExponentOverflow(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
	}{
		{
			name: "positive over int32",
			// tag 4, [2147483648, 0]
			raw: []byte{0xc4, 0x82, 0x1a, 0x80, 0x00, 0x00, 0x00, 0xc2, 0x40},
		},
		{
			name: "negative below int32",
			// tag 4, [-2147483649, 0]. CBOR negative stores -1-n = 2147483648.
			raw: []byte{0xc4, 0x82, 0x3a, 0x80, 0x00, 0x00, 0x00, 0xc2, 0x40},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got cborx.Decimal
			if err := got.UnmarshalCBOR(bytes.NewReader(tt.raw)); err == nil {
				t.Fatalf("expected exponent overflow rejection")
			}
		})
	}
}

func TestDecimal_RejectFloatMantissa(t *testing.T) {
	// tag 4, [exp 0, float 1.0 as half-precision 0xf9 0x3c 0x00].
	raw := []byte{0xc4, 0x82, 0x00, 0xf9, 0x3c, 0x00}
	var got cborx.Decimal
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of float mantissa")
	}
}
