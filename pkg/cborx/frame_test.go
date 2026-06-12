package cborx_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

func TestFrameRoundTrip(t *testing.T) {
	body := cborx.BigInt{V: big.NewInt(42)}
	var buf bytes.Buffer
	if err := cborx.WriteFrame(&buf, cborx.V1, body); err != nil {
		t.Fatalf("WriteFrame: %v", err)
	}

	var got cborx.BigInt
	var ver cborx.Version
	if err := cborx.ReadFrame(&buf, cborx.MaxControlFrame, &ver, &got); err != nil {
		t.Fatalf("ReadFrame: %v", err)
	}
	if ver != cborx.V1 {
		t.Fatalf("version = 0x%02x, want V1", uint8(ver))
	}
	if got.V.Cmp(body.V) != 0 {
		t.Fatalf("round-trip value = %s, want %s", got.V, body.V)
	}
}

func TestFrameRejectsOversizeBeforeBodyAllocation(t *testing.T) {
	var hdr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(hdr[:], cborx.MaxControlFrame+1)
	buf := bytes.NewReader(hdr[:n])

	var got cborx.BigInt
	var ver cborx.Version
	err := cborx.ReadFrame(buf, cborx.MaxControlFrame, &ver, &got)
	if !errors.Is(err, cborx.ErrFrameTooLarge) {
		t.Fatalf("expected ErrFrameTooLarge, got %v", err)
	}
}

func TestFrameCleanEOF(t *testing.T) {
	var got cborx.BigInt
	var ver cborx.Version
	err := cborx.ReadFrame(bytes.NewReader(nil), cborx.MaxControlFrame, &ver, &got)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("empty stream = %v, want io.EOF", err)
	}
}

func TestFrameRejectsTrailingBytesInsideDeclaredFrame(t *testing.T) {
	var body bytes.Buffer
	if err := cborx.WriteEnvelope(&body, cborx.V1, cborx.BigInt{V: big.NewInt(42)}); err != nil {
		t.Fatalf("WriteEnvelope: %v", err)
	}
	body.WriteByte(0xff)

	var frame bytes.Buffer
	var hdr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(hdr[:], uint64(body.Len()))
	frame.Write(hdr[:n])
	frame.Write(body.Bytes())

	var got cborx.BigInt
	var ver cborx.Version
	err := cborx.ReadFrame(&frame, cborx.MaxControlFrame, &ver, &got)
	if !errors.Is(err, cborx.ErrFrameTrailingBytes) {
		t.Fatalf("expected ErrFrameTrailingBytes, got %v", err)
	}
}
