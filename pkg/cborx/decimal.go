package cborx

import (
	"errors"
	"fmt"
	"io"
	"math"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow

	"github.com/layer-3/clearnet-sdk/pkg/decimal" // layer-guard: allow
)

// tagDecimalFraction is RFC 8949 §3.4.4: major type 6, tag 4, wrapping
// a two-element array [exponent, mantissa].
const tagDecimalFraction uint64 = 4

// Decimal is a CBOR adapter for internal/decimal.Decimal (the project's
// vendored fixed-point decimal; API: Exponent() int32, Coefficient()
// *big.Int, NewFromBigInt).
//
// Encoding follows RFC 8949 §3.4.4:
//
//   - major type 6 (tag) with value 4,
//   - wrapping a definite-length array of exactly two elements,
//   - element 0: exponent, written as the shortest-form
//     MajUnsignedInt / MajNegativeInt (docs/specs/cbor.md §2 disallows
//     wider encodings),
//   - element 1: mantissa, written as a BigInt (RFC 8949 tag 2/3)
//     so the sign is carried without an extra prefix.
//
// internal/decimal exposes only int32 exponents, so the exponent fits
// comfortably into CBOR's native integer range and no bignum is ever
// needed for it.
type Decimal struct {
	V decimal.Decimal
}

// MarshalCBOR writes the tag-4 two-element array.
func (d Decimal) MarshalCBOR(w io.Writer) error {
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajTag, tagDecimalFraction); err != nil {
		return fmt.Errorf("cborx: Decimal: write tag: %w", err)
	}
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajArray, 2); err != nil {
		return fmt.Errorf("cborx: Decimal: write array header: %w", err)
	}

	exp := int64(d.V.Exponent())
	if exp >= 0 {
		if err := cbg.WriteMajorTypeHeader(w, cbg.MajUnsignedInt, uint64(exp)); err != nil {
			return fmt.Errorf("cborx: Decimal: write exponent: %w", err)
		}
	} else {
		// Canonical negative int: value = -1 - n, stored as uint64.
		if err := cbg.WriteMajorTypeHeader(w, cbg.MajNegativeInt, uint64(-exp)-1); err != nil {
			return fmt.Errorf("cborx: Decimal: write negative exponent: %w", err)
		}
	}

	mantissa := d.V.Coefficient()
	if err := (BigInt{V: mantissa}).MarshalCBOR(w); err != nil {
		return fmt.Errorf("cborx: Decimal: write mantissa: %w", err)
	}
	return nil
}

// UnmarshalCBOR decodes the tag-4 two-element array.
func (d *Decimal) UnmarshalCBOR(r io.Reader) error {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: Decimal: read tag: %w", err)
	}
	if maj != cbg.MajTag {
		return fmt.Errorf("cborx: Decimal: expected tag (major 6), got major %d", maj)
	}
	if val != tagDecimalFraction {
		return fmt.Errorf("cborx: Decimal: expected tag 4, got tag %d", val)
	}

	maj, val, err = cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: Decimal: read array header: %w", err)
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("cborx: Decimal: expected array (major 4), got major %d", maj)
	}
	if val != 2 {
		return fmt.Errorf("cborx: Decimal: expected 2 elements, got %d", val)
	}

	var expI int64
	maj, val, err = cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: Decimal: read exponent: %w", err)
	}
	switch maj {
	case cbg.MajUnsignedInt:
		if val > math.MaxInt32 {
			return fmt.Errorf("cborx: Decimal: exponent %d overflows int32", val)
		}
		expI = int64(val)
	case cbg.MajNegativeInt:
		// CBOR negative = -1 - val; int32 range check.
		if val > math.MaxInt32 {
			return fmt.Errorf("cborx: Decimal: negative exponent overflows int32")
		}
		expI = -int64(val) - 1
	default:
		return fmt.Errorf("cborx: Decimal: exponent must be integer, got major %d", maj)
	}
	if expI < math.MinInt32 || expI > math.MaxInt32 {
		return fmt.Errorf("cborx: Decimal: exponent %d out of int32 range", expI)
	}

	var mantissa BigInt
	if err := mantissa.UnmarshalCBOR(r); err != nil {
		return fmt.Errorf("cborx: Decimal: mantissa: %w", err)
	}
	if mantissa.V == nil {
		return errors.New("cborx: Decimal: mantissa decoded to nil big.Int")
	}

	d.V = decimal.NewFromBigInt(mantissa.V, int32(expI))
	return nil
}
