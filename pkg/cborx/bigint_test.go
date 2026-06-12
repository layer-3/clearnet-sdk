package cborx_test

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

// encodeBigInt is the shared helper every table-driven test goes
// through. Returns the byte output of a BigInt{V: n}.MarshalCBOR.
func encodeBigInt(t *testing.T, n *big.Int) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := (cborx.BigInt{V: n}).MarshalCBOR(&buf); err != nil {
		t.Fatalf("MarshalCBOR(%s): %v", n, err)
	}
	return buf.Bytes()
}

func TestBigInt_ZeroCanonicalForm(t *testing.T) {
	want := []byte{0xc2, 0x40} // tag 2, byte string length 0
	got := encodeBigInt(t, big.NewInt(0))
	if !bytes.Equal(got, want) {
		t.Fatalf("zero encoding = %x, want %x", got, want)
	}
	// Default-constructed big.Int is also zero.
	got2 := encodeBigInt(t, new(big.Int))
	if !bytes.Equal(got2, want) {
		t.Fatalf("new(big.Int) encoding = %x, want %x", got2, want)
	}
}

func TestBigInt_ShortestForm(t *testing.T) {
	cases := []struct {
		n    *big.Int
		want string // hex
	}{
		// Positive tag 2 with big-endian magnitude.
		{big.NewInt(1), "c241" + "01"},
		{big.NewInt(23), "c241" + "17"},
		{big.NewInt(255), "c241" + "ff"},
		{big.NewInt(256), "c242" + "0100"},
		{big.NewInt(65535), "c242" + "ffff"},
		{big.NewInt(65536), "c243" + "010000"},
		{big.NewInt(1<<32 - 1), "c244" + "ffffffff"},
		{big.NewInt(1 << 32), "c245" + "0100000000"},
		{big.NewInt(int64(1<<63 - 1)), "c248" + "7fffffffffffffff"},
		{new(big.Int).SetUint64(^uint64(0)), "c248" + "ffffffffffffffff"},
		// Negative tag 3: magnitude = -1 - n.
		{big.NewInt(-1), "c340"},          // mag 0, empty byte string
		{big.NewInt(-2), "c341" + "01"},   // mag 1
		{big.NewInt(-24), "c341" + "17"},  // mag 23
		{big.NewInt(-25), "c341" + "18"},  // mag 24
		{big.NewInt(-256), "c341" + "ff"}, // mag 255
		{big.NewInt(-257), "c342" + "0100"},
	}
	for _, c := range cases {
		got := encodeBigInt(t, c.n)
		if hex.EncodeToString(got) != c.want {
			t.Errorf("encode(%s) = %x, want %s", c.n, got, c.want)
		}
	}
}

func TestBigInt_LargeBignum(t *testing.T) {
	// 2^128 is canonical tag-2 with 17 big-endian bytes (0x01 + 16 zeros).
	n := new(big.Int).Lsh(big.NewInt(1), 128)
	got := encodeBigInt(t, n)
	wantHex := "c251" + "0100000000000000000000000000000000"
	if hex.EncodeToString(got) != wantHex {
		t.Fatalf("2^128 encoded = %x, want %s", got, wantHex)
	}
}

func TestBigInt_MarshalDoesNotMutateNegativeInput(t *testing.T) {
	n := big.NewInt(-257)
	want := new(big.Int).Set(n)

	_ = encodeBigInt(t, n)

	if n.Cmp(want) != 0 {
		t.Fatalf("MarshalCBOR mutated input: got %s, want %s", n, want)
	}
}

func TestBigInt_RoundTrip(t *testing.T) {
	cases := []*big.Int{
		big.NewInt(0),
		big.NewInt(1),
		big.NewInt(-1),
		big.NewInt(23),
		big.NewInt(24),
		big.NewInt(255),
		big.NewInt(256),
		big.NewInt(1<<32 - 1),
		big.NewInt(1 << 32),
		big.NewInt(int64(1<<63 - 1)),
		new(big.Int).SetUint64(^uint64(0)),
		new(big.Int).Lsh(big.NewInt(1), 128),
		new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 128)),
		big.NewInt(-24),
		big.NewInt(-25),
	}
	for _, n := range cases {
		bz := encodeBigInt(t, n)
		var got cborx.BigInt
		if err := got.UnmarshalCBOR(bytes.NewReader(bz)); err != nil {
			t.Fatalf("UnmarshalCBOR(%s): %v", n, err)
		}
		if got.V.Cmp(n) != 0 {
			t.Errorf("round-trip: got %s, want %s", got.V, n)
		}

		// Idempotence: re-encode and compare bytes.
		var buf2 bytes.Buffer
		if err := got.MarshalCBOR(&buf2); err != nil {
			t.Fatalf("re-marshal: %v", err)
		}
		if !bytes.Equal(buf2.Bytes(), bz) {
			t.Errorf("idempotence for %s: %x vs %x", n, buf2.Bytes(), bz)
		}
	}
}

func TestBigInt_RejectNil(t *testing.T) {
	var buf bytes.Buffer
	err := (cborx.BigInt{V: nil}).MarshalCBOR(&buf)
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("expected nil-rejection error, got %v", err)
	}
}

func TestBigInt_RejectNonCanonicalLeadingZero(t *testing.T) {
	// tag 2, length 1, byte 0x00 — value decodes to 0, but canonical
	// zero is the empty byte string. Must reject.
	raw := []byte{0xc2, 0x41, 0x00}
	var got cborx.BigInt
	err := got.UnmarshalCBOR(bytes.NewReader(raw))
	if err == nil {
		t.Fatalf("expected rejection of tag-2 with leading zero, decoded %v", got.V)
	}
}

func TestBigInt_RejectNegativeNonCanonicalLeadingZero(t *testing.T) {
	// tag 3, length 1, byte 0x00 — value decodes to -1, but canonical
	// -1 is tag 3 with an empty byte string. Must reject.
	raw := []byte{0xc3, 0x41, 0x00}
	var got cborx.BigInt
	err := got.UnmarshalCBOR(bytes.NewReader(raw))
	if err == nil {
		t.Fatalf("expected rejection of tag-3 with leading zero, decoded %v", got.V)
	}
}

func TestBigInt_RejectWrongTag(t *testing.T) {
	// tag 5 is reserved; bignum decoder must reject.
	raw := []byte{0xc5, 0x40}
	var got cborx.BigInt
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of tag 5")
	}
}

func TestBigInt_RejectIndefiniteLengthByteString(t *testing.T) {
	// tag 2 + indefinite-length byte string (0x5f ... 0xff).
	// cbor-gen's reader rejects indefinite by hitting the default
	// "invalid header" branch; we surface that as a decode error.
	raw := []byte{0xc2, 0x5f, 0x41, 0x01, 0xff}
	var got cborx.BigInt
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of indefinite byte string")
	}
}

func TestBigInt_NonShortestHeaderRejected(t *testing.T) {
	// tag 2, byte-string length encoded as 2-byte value 0x0001 (low = 25).
	// Canonical would be length 1 (low = 0x41). cbor-gen's header reader
	// rejects this as non-canonical on decode.
	raw := []byte{0xc2, 0x59, 0x00, 0x01, 0x42}
	var got cborx.BigInt
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of non-shortest length header")
	}
}

func TestBigInt_RejectsOversizeMagnitudeBeforeAllocation(t *testing.T) {
	// tag 2, byte-string length = 1 MiB + 1, no body. The decoder must
	// reject on the declared length before allocating or attempting ReadFull.
	raw := []byte{0xc2, 0x5a, 0x00, 0x10, 0x00, 0x01}
	var got cborx.BigInt
	err := got.UnmarshalCBOR(bytes.NewReader(raw))
	if err == nil || !strings.Contains(err.Error(), "1 MiB") {
		t.Fatalf("expected oversize magnitude rejection, got %v", err)
	}
}
