// Package decimal — CBOR adapter methods for decimal.Decimal.
//
// This file attaches MarshalCBOR / UnmarshalCBOR to decimal.Decimal so that
// cbor-gen (which emits `v.MarshalCBOR(cw)` for every field of a struct whose
// type implements cbg.CBORMarshaler / cbg.CBORUnmarshaler) can serialize
// Decimal fields through the canonical tag-4 encoding pinned by ADR-009 §3.
//
// Why not delegate to internal/cborx.Decimal?
//
// internal/cborx.Decimal already wraps decimal.Decimal with a tag-4 codec
// (see internal/cborx/decimal.go), but internal/cborx imports
// internal/decimal — delegating from decimal back to cborx would close an
// import cycle. So this file re-implements the tag-4 byte layout using only
// cbor-gen primitives (cbg.WriteMajorTypeHeader / cbg.CborReadHeader) and
// stdlib. The byte output is byte-identical to cborx.Decimal and is covered
// by internal/cborx's round-trip and golden tests, plus by the round-trip
// tests attached to every clearing/core codec that carries a Decimal field.
//
// Encoding (RFC 8949 §3.4.4):
//
//   - major type 6 (tag) with value 4,
//   - wrapping a definite-length array of exactly two elements,
//   - element 0: exponent (int32 always fits in CBOR's native integer
//     range), shortest-form signed integer,
//   - element 1: mantissa as a tag-2 / tag-3 bignum (RFC 8949 §3.4.3).
//
// Owned by the CBOR encoding migration, Wave 2-core
// (docs/plans/cbor-encoding.md §15.10). Never hand-edited after this
// landing except to track a change in decimal.Decimal's API.
package decimal

import (
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// tag constants mirror internal/cborx (kept local to avoid the import cycle
// with that package; the byte-level output is identical — see the package
// doc).
const (
	tagDecimalFraction uint64 = 4
	tagUnsignedBignum  uint64 = 2
	tagNegativeBignum  uint64 = 3
)

// MarshalCBOR writes d as a canonical RFC 8949 tag-4 decimal fraction.
// The exponent (int32) is written shortest-form; the mantissa is written
// as a tag-2/tag-3 bignum so the sign rides on the tag.
//
// Value receiver — cbor-gen's tuple emitter calls `t.Field.MarshalCBOR(cw)`
// on field values; Decimal is a small by-value type, so a value receiver
// is both idiomatic and sufficient.
func (d Decimal) MarshalCBOR(w io.Writer) error {
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajTag, tagDecimalFraction); err != nil {
		return fmt.Errorf("decimal: MarshalCBOR: write tag: %w", err)
	}
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajArray, 2); err != nil {
		return fmt.Errorf("decimal: MarshalCBOR: write array header: %w", err)
	}

	exp := int64(d.Exponent())
	if exp >= 0 {
		if err := cbg.WriteMajorTypeHeader(w, cbg.MajUnsignedInt, uint64(exp)); err != nil {
			return fmt.Errorf("decimal: MarshalCBOR: write exponent: %w", err)
		}
	} else {
		if err := cbg.WriteMajorTypeHeader(w, cbg.MajNegativeInt, uint64(-exp)-1); err != nil {
			return fmt.Errorf("decimal: MarshalCBOR: write negative exponent: %w", err)
		}
	}

	return writeBignum(w, d.Coefficient())
}

// UnmarshalCBOR decodes a tag-4 two-element array into d. Non-canonical
// encodings (wrong tag, wrong array length, indefinite-length containers,
// over-wide integer headers, non-canonical bignums) are rejected.
func (d *Decimal) UnmarshalCBOR(r io.Reader) error {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("decimal: UnmarshalCBOR: read tag: %w", err)
	}
	if maj != cbg.MajTag {
		return fmt.Errorf("decimal: UnmarshalCBOR: expected tag (major 6), got major %d", maj)
	}
	if val != tagDecimalFraction {
		return fmt.Errorf("decimal: UnmarshalCBOR: expected tag 4, got tag %d", val)
	}

	maj, val, err = cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("decimal: UnmarshalCBOR: read array header: %w", err)
	}
	if maj != cbg.MajArray {
		return fmt.Errorf("decimal: UnmarshalCBOR: expected array (major 4), got major %d", maj)
	}
	if val != 2 {
		return fmt.Errorf("decimal: UnmarshalCBOR: expected 2 elements, got %d", val)
	}

	maj, val, err = cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("decimal: UnmarshalCBOR: read exponent: %w", err)
	}
	var expI int64
	switch maj {
	case cbg.MajUnsignedInt:
		if val > math.MaxInt32 {
			return fmt.Errorf("decimal: UnmarshalCBOR: exponent %d overflows int32", val)
		}
		expI = int64(val)
	case cbg.MajNegativeInt:
		if val > math.MaxInt32 {
			return errors.New("decimal: UnmarshalCBOR: negative exponent overflows int32")
		}
		expI = -int64(val) - 1
	default:
		return fmt.Errorf("decimal: UnmarshalCBOR: exponent must be integer, got major %d", maj)
	}
	if expI < math.MinInt32 || expI > math.MaxInt32 {
		return fmt.Errorf("decimal: UnmarshalCBOR: exponent %d out of int32 range", expI)
	}

	mant, err := readBignum(r)
	if err != nil {
		return fmt.Errorf("decimal: UnmarshalCBOR: mantissa: %w", err)
	}
	if mant == nil {
		return errors.New("decimal: UnmarshalCBOR: mantissa decoded to nil big.Int")
	}

	*d = NewFromBigInt(mant, int32(expI))
	return nil
}

// writeBignum writes a *big.Int as an RFC 8949 tag-2 / tag-3 bignum.
// Non-negative values use tag 2; negative values use tag 3 wrapping the
// magnitude of (-1 - n). Zero is canonical tag-2 + empty byte string.
func writeBignum(w io.Writer, n *big.Int) error {
	if n == nil {
		n = new(big.Int)
	}
	var (
		tag  uint64
		mag  []byte
		sign = n.Sign()
	)
	switch {
	case sign >= 0:
		tag = tagUnsignedBignum
		mag = n.Bytes()
	default:
		tag = tagNegativeBignum
		neg := new(big.Int).Neg(n)
		neg.Sub(neg, big.NewInt(1))
		mag = neg.Bytes()
	}
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajTag, tag); err != nil {
		return fmt.Errorf("decimal: writeBignum: write tag: %w", err)
	}
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajByteString, uint64(len(mag))); err != nil {
		return fmt.Errorf("decimal: writeBignum: write length: %w", err)
	}
	if len(mag) > 0 {
		if _, err := w.Write(mag); err != nil {
			return fmt.Errorf("decimal: writeBignum: write bytes: %w", err)
		}
	}
	return nil
}

// readBignum decodes a tag-2 / tag-3 bignum, enforcing the canonical-zero
// and canonical-leading-byte rules of RFC 8949 §4.2.
func readBignum(r io.Reader) (*big.Int, error) {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return nil, fmt.Errorf("decimal: readBignum: read tag: %w", err)
	}
	if maj != cbg.MajTag {
		return nil, fmt.Errorf("decimal: readBignum: expected tag (major 6), got major %d", maj)
	}
	tag := val

	maj, val, err = cbg.CborReadHeader(r)
	if err != nil {
		return nil, fmt.Errorf("decimal: readBignum: read length: %w", err)
	}
	if maj != cbg.MajByteString {
		return nil, fmt.Errorf("decimal: readBignum: expected byte string (major 2), got major %d", maj)
	}
	length := val

	if length > 1<<20 {
		return nil, fmt.Errorf("decimal: readBignum: refusing length %d (> 1 MiB)", length)
	}

	var buf []byte
	if length > 0 {
		buf = make([]byte, length)
		if _, err := io.ReadFull(r, buf); err != nil {
			return nil, fmt.Errorf("decimal: readBignum: read bytes: %w", err)
		}
		if buf[0] == 0 {
			return nil, errors.New("decimal: readBignum: non-canonical leading zero byte")
		}
	}

	switch tag {
	case tagUnsignedBignum:
		n := new(big.Int)
		if length > 0 {
			n.SetBytes(buf)
		}
		return n, nil
	case tagNegativeBignum:
		if length == 0 {
			return big.NewInt(-1), nil
		}
		mag := new(big.Int).SetBytes(buf)
		n := new(big.Int).Neg(mag)
		n.Sub(n, big.NewInt(1))
		return n, nil
	default:
		return nil, fmt.Errorf("decimal: readBignum: expected tag 2 or 3, got tag %d", tag)
	}
}
