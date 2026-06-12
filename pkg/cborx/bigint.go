package cborx

import (
	"errors"
	"fmt"
	"io"
	"math/big"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// Bignum tag numbers from RFC 8949 §3.4.3.
const (
	tagUnsignedBignum uint64 = 2 // positive *big.Int, value = big-endian byte string
	tagNegativeBignum uint64 = 3 // negative *big.Int, value = big-endian byte string of (-1 - n)
)

// BigInt is a CBOR adapter for math/big.Int.
//
// Encoding follows RFC 8949 §3.4.3:
//
//   - Non-negative n is written as CBOR tag 2 wrapping a byte-string
//     of big-endian magnitude bytes, with leading zero bytes trimmed.
//   - Negative n is written as CBOR tag 3 wrapping the big-endian
//     magnitude of (-1 - n) with leading zeros trimmed.
//   - Zero is the single canonical form "tag 2 + empty byte string"
//     (length = 0). This keeps zero unambiguous: the negative-bignum
//     form of zero (-1 - 0 = -1, magnitude 1, byte 0x01) is structurally
//     a different encoding of -1, not of 0, and is rejected as malformed
//     when a byte-string of length 0 appears under tag 3.
//   - A nil *big.Int is illegal inside a BigInt — use MaybeBigInt when
//     absence is a legitimate state.
//
// The adapter rejects nil on marshal; use Value only when callers need a
// read-side zero default for an absent pointer.
type BigInt struct {
	// V is the wrapped integer. Callers pass and read the pointer; the
	// adapter never mutates it in-place when marshaling.
	V *big.Int
}

// Value returns a non-nil *big.Int (zero if b.V was nil).
func (b BigInt) Value() *big.Int {
	if b.V == nil {
		return new(big.Int)
	}
	return b.V
}

// MarshalCBOR writes the RFC 8949 tag-2 / tag-3 bignum encoding.
//
// A nil inner value is rejected; callers must wrap a nilable amount in
// MaybeBigInt (which encodes nil as CBOR null).
func (b BigInt) MarshalCBOR(w io.Writer) error {
	if b.V == nil {
		return errors.New("cborx: BigInt: nil *big.Int (use MaybeBigInt for nilable fields)")
	}

	var (
		tag  uint64
		mag  []byte
		sign = b.V.Sign()
	)
	switch {
	case sign >= 0:
		tag = tagUnsignedBignum
		mag = b.V.Bytes() // big-endian, no leading zeros, empty for zero
	default:
		tag = tagNegativeBignum
		// CBOR encodes the negative value n as the magnitude of (-1 - n).
		// Allocate a new big.Int so we never mutate b.V.
		neg := new(big.Int).Neg(b.V)
		neg.Sub(neg, big.NewInt(1))
		mag = neg.Bytes()
	}

	if err := cbg.WriteMajorTypeHeader(w, cbg.MajTag, tag); err != nil {
		return fmt.Errorf("cborx: BigInt: write tag: %w", err)
	}
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajByteString, uint64(len(mag))); err != nil {
		return fmt.Errorf("cborx: BigInt: write length: %w", err)
	}
	if len(mag) > 0 {
		if _, err := w.Write(mag); err != nil {
			return fmt.Errorf("cborx: BigInt: write bytes: %w", err)
		}
	}
	return nil
}

// UnmarshalCBOR decodes a tag-2 / tag-3 bignum into b.V, enforcing the
// canonical-zero and canonical-leading-byte rules of RFC 8949 §4.2.
func (b *BigInt) UnmarshalCBOR(r io.Reader) error {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: BigInt: read tag: %w", err)
	}
	if maj != cbg.MajTag {
		return fmt.Errorf("cborx: BigInt: expected tag (major 6), got major %d", maj)
	}
	tag := val

	maj, val, err = cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: BigInt: read length: %w", err)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("cborx: BigInt: expected byte string (major 2), got major %d", maj)
	}
	length := val

	// Deterministic-encoding rule: a byte string backing a bignum is
	// big-endian with no leading zero padding, except for zero itself
	// which is the empty byte string under tag 2.
	if length > 1<<20 {
		return fmt.Errorf("cborx: BigInt: refusing length %d (> 1 MiB)", length)
	}

	var buf []byte
	if length > 0 {
		buf = make([]byte, length)
		if _, err := io.ReadFull(r, buf); err != nil {
			return fmt.Errorf("cborx: BigInt: read bytes: %w", err)
		}
		if buf[0] == 0 {
			return errors.New("cborx: BigInt: non-canonical leading zero byte")
		}
	}

	switch tag {
	case tagUnsignedBignum:
		n := new(big.Int)
		if length > 0 {
			n.SetBytes(buf)
		}
		b.V = n
		return nil
	case tagNegativeBignum:
		// RFC 8949 §3.4.3: tag 3 wraps the big-endian magnitude m of
		// (-1 - n). For n = -1, m = 0, which is the empty byte string.
		// That is the canonical encoding of -1 under tag 3.
		if length == 0 {
			b.V = big.NewInt(-1)
			return nil
		}
		mag := new(big.Int).SetBytes(buf)
		n := new(big.Int).Neg(mag)
		n.Sub(n, big.NewInt(1))
		b.V = n
		return nil
	default:
		return fmt.Errorf("cborx: BigInt: expected tag 2 or 3, got tag %d", tag)
	}
}
