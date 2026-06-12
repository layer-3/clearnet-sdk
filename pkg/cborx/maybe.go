package cborx

import (
	"fmt"
	"io"
	"math/big"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// cborNullByte is CBOR's null encoding: major type 7, simple value 22
// → 0xf6.
const cborNullByte byte = 0xf6

// MaybeBigInt wraps a nilable *big.Int. CBOR null decodes to a
// MaybeBigInt with V == nil; any other input is delegated to BigInt.
// Useful for optional-amount fields (e.g. a MinAmountOut that may be
// absent rather than zero).
type MaybeBigInt struct {
	V *big.Int
}

// MarshalCBOR writes CBOR null when V == nil, otherwise the BigInt form.
func (m MaybeBigInt) MarshalCBOR(w io.Writer) error {
	if m.V == nil {
		if _, err := w.Write([]byte{cborNullByte}); err != nil {
			return fmt.Errorf("cborx: MaybeBigInt: write null: %w", err)
		}
		return nil
	}
	return BigInt{V: m.V}.MarshalCBOR(w)
}

// UnmarshalCBOR peeks the first byte to distinguish CBOR null from a
// tagged bignum. The cbg.CborReader wrapper provides the unread
// capability we need regardless of the underlying reader type.
func (m *MaybeBigInt) UnmarshalCBOR(r io.Reader) error {
	cr := cbg.NewCborReader(r)

	first, err := cr.ReadByte()
	if err != nil {
		return fmt.Errorf("cborx: MaybeBigInt: peek: %w", err)
	}
	if first == cborNullByte {
		m.V = nil
		return nil
	}
	if err := cr.UnreadByte(); err != nil {
		return fmt.Errorf("cborx: MaybeBigInt: unread: %w", err)
	}

	var inner BigInt
	if err := inner.UnmarshalCBOR(cr); err != nil {
		return err
	}
	m.V = inner.V
	return nil
}
