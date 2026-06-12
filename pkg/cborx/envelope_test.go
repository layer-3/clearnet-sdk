package cborx_test

import (
	"bytes"
	"errors"
	"io"
	"math/big"
	"testing"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

func TestEnvelope_RoundTripV1(t *testing.T) {
	body := cborx.BigInt{V: big.NewInt(1234567890)}
	var buf bytes.Buffer
	if err := cborx.WriteEnvelope(&buf, cborx.V1, body); err != nil {
		t.Fatalf("WriteEnvelope: %v", err)
	}
	if buf.Bytes()[0] != byte(cborx.V1) {
		t.Fatalf("first byte = 0x%02x, want 0x%02x", buf.Bytes()[0], byte(cborx.V1))
	}

	var got cborx.BigInt
	var ver cborx.Version
	if err := cborx.ReadEnvelope(&buf, &ver, &got); err != nil {
		t.Fatalf("ReadEnvelope: %v", err)
	}
	if ver != cborx.V1 {
		t.Fatalf("version = 0x%02x, want V1", uint8(ver))
	}
	if got.V.Cmp(body.V) != 0 {
		t.Fatalf("round-trip value = %s, want %s", got.V, body.V)
	}
}

func TestEnvelope_RejectReservedByte(t *testing.T) {
	buf := bytes.NewReader([]byte{0x00, 0xc2, 0x40}) // 0x00 then a valid tag-2 zero
	var got cborx.BigInt
	var ver cborx.Version
	err := cborx.ReadEnvelope(buf, &ver, &got)
	if err == nil {
		t.Fatalf("expected error reading reserved 0x00")
	}
	if !errors.Is(err, cborx.ErrReservedVersion) {
		t.Fatalf("expected ErrReservedVersion, got %v", err)
	}
	if ver != cborx.VersionReserved {
		t.Fatalf("ver should record the rejected byte (0x00), got 0x%02x", uint8(ver))
	}
}

func TestEnvelope_RejectFutureVersion(t *testing.T) {
	for _, b := range []byte{0x02, 0x03, 0x7f, 0x80, 0xff} {
		buf := bytes.NewReader([]byte{b, 0xc2, 0x40})
		var got cborx.BigInt
		var ver cborx.Version
		err := cborx.ReadEnvelope(buf, &ver, &got)
		if err == nil {
			t.Fatalf("expected error reading 0x%02x", b)
		}
		if !errors.Is(err, cborx.ErrUnsupportedVersion) {
			t.Fatalf("for 0x%02x: expected ErrUnsupportedVersion, got %v", b, err)
		}
		if uint8(ver) != b {
			t.Fatalf("for 0x%02x: ver = 0x%02x, want 0x%02x", b, uint8(ver), b)
		}
	}
}

func TestEnvelope_WriteRejectsReservedAndFuture(t *testing.T) {
	body := cborx.BigInt{V: big.NewInt(0)}

	var buf bytes.Buffer
	if err := cborx.WriteEnvelope(&buf, cborx.VersionReserved, body); !errors.Is(err, cborx.ErrReservedVersion) {
		t.Fatalf("WriteEnvelope(0x00): expected ErrReservedVersion, got %v", err)
	}
	for _, v := range []cborx.Version{0x02, 0x7f, 0x80, 0xff} {
		var tmp bytes.Buffer
		if err := cborx.WriteEnvelope(&tmp, v, body); !errors.Is(err, cborx.ErrUnsupportedVersion) {
			t.Fatalf("WriteEnvelope(0x%02x): expected ErrUnsupportedVersion, got %v", uint8(v), err)
		}
	}
}

func TestEnvelope_RejectNilArguments(t *testing.T) {
	var buf bytes.Buffer
	if err := cborx.WriteEnvelope(&buf, cborx.V1, nil); err == nil {
		t.Fatal("WriteEnvelope nil body: got nil error")
	}

	var got cborx.BigInt
	if err := cborx.ReadEnvelope(bytes.NewReader([]byte{0x01, 0xc2, 0x40}), nil, &got); err == nil {
		t.Fatal("ReadEnvelope nil version pointer: got nil error")
	}
	var ver cborx.Version
	if err := cborx.ReadEnvelope(bytes.NewReader([]byte{0x01, 0xc2, 0x40}), &ver, nil); err == nil {
		t.Fatal("ReadEnvelope nil body: got nil error")
	}
}

func TestEnvelope_EmptyInput(t *testing.T) {
	var got cborx.BigInt
	var ver cborx.Version
	err := cborx.ReadEnvelope(bytes.NewReader(nil), &ver, &got)
	if !errors.Is(err, cborx.ErrEmptyEnvelope) {
		t.Fatalf("empty input: expected ErrEmptyEnvelope, got %v", err)
	}
}

// brokenBody is a CBORUnmarshaler that always fails, used to prove
// body errors are distinct from version-byte errors.
type brokenBody struct{}

func (brokenBody) UnmarshalCBOR(r io.Reader) error { return errors.New("body broken") }

func TestEnvelope_BodyErrorDistinguishable(t *testing.T) {
	// Valid V1 envelope followed by bytes the broken body will reject.
	buf := bytes.NewReader([]byte{0x01, 0x00})
	var ver cborx.Version
	err := cborx.ReadEnvelope(buf, &ver, brokenBody{})
	if err == nil {
		t.Fatal("expected body error")
	}
	if errors.Is(err, cborx.ErrReservedVersion) || errors.Is(err, cborx.ErrUnsupportedVersion) || errors.Is(err, cborx.ErrEmptyEnvelope) {
		t.Fatalf("body error should not match version sentinels; got %v", err)
	}
	if ver != cborx.V1 {
		t.Fatalf("ver should be V1 once the version byte parses; got 0x%02x", uint8(ver))
	}
}
