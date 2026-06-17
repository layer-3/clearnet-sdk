package cborx_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

func TestMaybeBigInt_NilEncodesAsNull(t *testing.T) {
	var buf bytes.Buffer
	if err := (cborx.MaybeBigInt{V: nil}).MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := []byte{0xf6} // CBOR null
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("nil MaybeBigInt = %x, want %x", buf.Bytes(), want)
	}
}

func TestMaybeBigInt_NullDecodesToNil(t *testing.T) {
	m := cborx.MaybeBigInt{V: big.NewInt(42)} // pre-populated, expect override
	if err := m.UnmarshalCBOR(bytes.NewReader([]byte{0xf6})); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if m.V != nil {
		t.Fatalf("expected nil V, got %v", m.V)
	}
}

func TestMaybeBigInt_ValueRoundTrips(t *testing.T) {
	for _, n := range []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(-1),
		big.NewInt(1 << 62),
		new(big.Int).Lsh(big.NewInt(1), 100),
	} {
		var buf bytes.Buffer
		if err := (cborx.MaybeBigInt{V: n}).MarshalCBOR(&buf); err != nil {
			t.Fatalf("marshal %s: %v", n, err)
		}
		var got cborx.MaybeBigInt
		if err := got.UnmarshalCBOR(bytes.NewReader(buf.Bytes())); err != nil {
			t.Fatalf("unmarshal %s: %v", n, err)
		}
		if got.V == nil || got.V.Cmp(n) != 0 {
			t.Fatalf("round-trip %s: got %v", n, got.V)
		}

		// Idempotence.
		var buf2 bytes.Buffer
		if err := got.MarshalCBOR(&buf2); err != nil {
			t.Fatalf("re-marshal: %v", err)
		}
		if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
			t.Fatalf("idempotence %s: %x vs %x", n, buf.Bytes(), buf2.Bytes())
		}
	}
}

func TestMaybeBigInt_RejectMalformedNonNull(t *testing.T) {
	// Anything other than CBOR null must be a canonical BigInt tag 2/3 value.
	// CBOR false is a valid simple value, but it is not a MaybeBigInt encoding.
	var got cborx.MaybeBigInt
	if err := got.UnmarshalCBOR(bytes.NewReader([]byte{0xf4})); err == nil {
		t.Fatal("expected malformed non-null MaybeBigInt to reject")
	}
}

func TestMaybeBigInt_NilIdempotence(t *testing.T) {
	var buf bytes.Buffer
	_ = (cborx.MaybeBigInt{V: nil}).MarshalCBOR(&buf)

	var got cborx.MaybeBigInt
	_ = got.UnmarshalCBOR(bytes.NewReader(buf.Bytes()))

	var buf2 bytes.Buffer
	_ = got.MarshalCBOR(&buf2)
	if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
		t.Fatalf("nil idempotence: %x vs %x", buf.Bytes(), buf2.Bytes())
	}
}
