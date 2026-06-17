package cborx_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

func TestHash32_EncodesWithDefiniteLength32(t *testing.T) {
	var h cborx.Hash32
	for i := range h.V {
		h.V[i] = byte(i)
	}
	var buf bytes.Buffer
	if err := h.MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// major type 2 with length 32 = 0x58 0x20, then the 32 bytes.
	wantPrefix := "5820"
	got := hex.EncodeToString(buf.Bytes())
	if !strings.HasPrefix(got, wantPrefix) {
		t.Fatalf("prefix = %s, want %s", got[:len(wantPrefix)], wantPrefix)
	}
	if len(buf.Bytes()) != 2+32 {
		t.Fatalf("total length = %d, want 34", len(buf.Bytes()))
	}
}

func TestHash32_RoundTrip(t *testing.T) {
	var h cborx.Hash32
	for i := range h.V {
		h.V[i] = byte(0xA0 ^ i)
	}
	var buf bytes.Buffer
	if err := h.MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got cborx.Hash32
	if err := got.UnmarshalCBOR(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.V != h.V {
		t.Fatalf("round-trip mismatch: %x vs %x", got.V, h.V)
	}

	var buf2 bytes.Buffer
	if err := got.MarshalCBOR(&buf2); err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
		t.Fatalf("idempotence: %x vs %x", buf.Bytes(), buf2.Bytes())
	}
}

func TestHash32_RejectWrongLength(t *testing.T) {
	cases := [][]byte{
		// length 0: 0x40
		{0x40},
		// length 31: 0x58 0x1f + 31 zero bytes
		append([]byte{0x58, 0x1f}, bytes.Repeat([]byte{0}, 31)...),
		// length 33: 0x58 0x21 + 33 zero bytes
		append([]byte{0x58, 0x21}, bytes.Repeat([]byte{0}, 33)...),
	}
	for i, c := range cases {
		var got cborx.Hash32
		if err := got.UnmarshalCBOR(bytes.NewReader(c)); err == nil {
			t.Errorf("case %d: expected rejection", i)
		}
	}
}

func TestHash32_RejectWrongMajor(t *testing.T) {
	// Text string (major 3) of length 32 — wrong major.
	raw := append([]byte{0x78, 0x20}, bytes.Repeat([]byte{'a'}, 32)...)
	var got cborx.Hash32
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of major 3")
	}
}

func TestAddr20_EncodesWithDefiniteLength20(t *testing.T) {
	var a cborx.Addr20
	for i := range a.V {
		a.V[i] = byte(i + 1)
	}
	var buf bytes.Buffer
	if err := a.MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// major 2, length 20 = 0x54.
	if buf.Bytes()[0] != 0x54 {
		t.Fatalf("first byte = 0x%02x, want 0x54", buf.Bytes()[0])
	}
	if len(buf.Bytes()) != 1+20 {
		t.Fatalf("total length = %d, want 21", len(buf.Bytes()))
	}
}

func TestAddr20_RoundTrip(t *testing.T) {
	var a cborx.Addr20
	for i := range a.V {
		a.V[i] = byte(i)
	}
	var buf bytes.Buffer
	if err := a.MarshalCBOR(&buf); err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got cborx.Addr20
	if err := got.UnmarshalCBOR(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.V != a.V {
		t.Fatalf("round-trip mismatch: %x vs %x", got.V, a.V)
	}

	var buf2 bytes.Buffer
	if err := got.MarshalCBOR(&buf2); err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), buf2.Bytes()) {
		t.Fatalf("idempotence: %x vs %x", buf.Bytes(), buf2.Bytes())
	}
}

func TestAddr20_RejectWrongLength(t *testing.T) {
	cases := [][]byte{
		{0x40}, // 0 bytes
		append([]byte{0x53}, bytes.Repeat([]byte{0}, 19)...), // 19 bytes
		append([]byte{0x55}, bytes.Repeat([]byte{0}, 21)...), // 21 bytes
	}
	for i, c := range cases {
		var got cborx.Addr20
		if err := got.UnmarshalCBOR(bytes.NewReader(c)); err == nil {
			t.Errorf("case %d: expected rejection", i)
		}
	}
}

func TestAddr20_RejectWrongMajor(t *testing.T) {
	// Text string (major 3) of length 20 — wrong major.
	raw := append([]byte{0x74}, bytes.Repeat([]byte{'a'}, 20)...)
	var got cborx.Addr20
	if err := got.UnmarshalCBOR(bytes.NewReader(raw)); err == nil {
		t.Fatalf("expected rejection of major 3")
	}
}
