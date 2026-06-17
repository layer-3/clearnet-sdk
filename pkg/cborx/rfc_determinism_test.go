package cborx_test

import (
	"bytes"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

// Hand-assembled wire bytes from RFC 8949 §4.2 violations. Every entry
// below is a byte string a malicious or buggy peer could send that
// MUST be rejected by our adapter pipeline.
var determinismRejects = map[string][]byte{
	// BigInt: tag 2 with byte-string length encoded 2-byte-wide for a
	// value that fits in 1 byte. cbor-gen's reader catches
	// "lval 25 with value <= MaxUint8".
	"bigint_non_shortest_len_2byte": {0xc2, 0x59, 0x00, 0x01, 0x01},
	// BigInt: tag 2 encoded with a one-byte payload instead of the direct
	// additional-info form. Canonical tag 2 is 0xc2, not 0xd8 0x02.
	"bigint_non_shortest_tag": {0xd8, 0x02, 0x40},
	// BigInt: tag 2 with byte-string length encoded 4-byte-wide for a
	// value that fits in 2 bytes.
	"bigint_non_shortest_len_4byte": {0xc2, 0x5a, 0x00, 0x00, 0x01, 0x00, 0x01, 0x02},
	// BigInt: tag 2 indefinite-length byte string.
	"bigint_indefinite_bytestring": {0xc2, 0x5f, 0x41, 0x01, 0xff},
	// Time: half-precision NaN (0xf9 0x7e 0x00).
	"time_float_nan": {0xf9, 0x7e, 0x00},
	// Time: half-precision +Inf (0xf9 0x7c 0x00).
	"time_float_inf": {0xf9, 0x7c, 0x00},
	// Time: single-precision negative zero (0xfa 0x80 0x00 0x00 0x00).
	"time_float_neg_zero": {0xfa, 0x80, 0x00, 0x00, 0x00},
	// Decimal: tag 4, array-length encoded 2-byte-wide instead of 1-byte.
	"decimal_non_shortest_array": {0xc4, 0x98, 0x02, 0x00, 0xc2, 0x40},
	// Decimal: tag 4 encoded with a one-byte payload instead of direct tag 4.
	"decimal_non_shortest_tag": {0xd8, 0x04, 0x82, 0x00, 0xc2, 0x40},
	// Hash32: indefinite-length byte string of 32 bytes total.
	"hash32_indefinite": append([]byte{0x5f, 0x58, 0x20}, append(bytes.Repeat([]byte{0}, 32), 0xff)...),
	// Hash32: fixed byte-string length 32 encoded 2-byte-wide instead of
	// the shortest one-byte length payload.
	"hash32_non_shortest_len": append([]byte{0x59, 0x00, 0x20}, bytes.Repeat([]byte{0}, 32)...),
	// Addr20: length 20 fits directly in the additional-info bits and must
	// not be encoded using an extra one-byte payload.
	"addr20_non_shortest_len": append([]byte{0x58, 0x14}, bytes.Repeat([]byte{0}, 20)...),
}

func TestDeterminismRejects_BigInt(t *testing.T) {
	for name, raw := range map[string][]byte{
		"bigint_non_shortest_len_2byte": determinismRejects["bigint_non_shortest_len_2byte"],
		"bigint_non_shortest_tag":       determinismRejects["bigint_non_shortest_tag"],
		"bigint_non_shortest_len_4byte": determinismRejects["bigint_non_shortest_len_4byte"],
		"bigint_indefinite_bytestring":  determinismRejects["bigint_indefinite_bytestring"],
	} {
		var got cborx.BigInt
		if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
			t.Errorf("%s: expected rejection, decoded %s", name, got.V)
		}
	}
}

func TestDeterminismRejects_Time(t *testing.T) {
	for name, raw := range map[string][]byte{
		"time_float_nan":      determinismRejects["time_float_nan"],
		"time_float_inf":      determinismRejects["time_float_inf"],
		"time_float_neg_zero": determinismRejects["time_float_neg_zero"],
	} {
		var got cborx.Time
		if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
			t.Errorf("%s: expected rejection of float, decoded %v", name, got.V)
		}
	}
}

func TestDeterminismRejects_Decimal(t *testing.T) {
	for name, raw := range map[string][]byte{
		"decimal_non_shortest_array": determinismRejects["decimal_non_shortest_array"],
		"decimal_non_shortest_tag":   determinismRejects["decimal_non_shortest_tag"],
	} {
		var got cborx.Decimal
		if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestDeterminismRejects_Hash32(t *testing.T) {
	for name, raw := range map[string][]byte{
		"hash32_indefinite":       determinismRejects["hash32_indefinite"],
		"hash32_non_shortest_len": determinismRejects["hash32_non_shortest_len"],
	} {
		var got cborx.Hash32
		if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
}

func TestDeterminismRejects_Addr20(t *testing.T) {
	var got cborx.Addr20
	if err := got.UnmarshalCBOR(bytes.NewReader(determinismRejects["addr20_non_shortest_len"])); err == nil {
		t.Errorf("addr20_non_shortest_len: expected rejection")
	}
}
