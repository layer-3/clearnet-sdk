package protocol

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

// Golden CBOR vectors freeze the wire bytes. Both clearnet and custody must
// encode these structs identically; a mismatch means the wire forked.
func TestWireGoldens(t *testing.T) {
	var nonce [32]byte
	for i := range nonce {
		nonce[i] = byte(i)
	}

	tests := []struct {
		name    string
		marshal func() ([]byte, error)
		wantHex string
	}{
		{
			name: "AuthChallenge",
			marshal: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&AuthChallenge{Nonce: nonce}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 81 = array(1); 5820 = byte string len 32; then the nonce bytes.
			wantHex: "815820" + "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f",
		},
		{
			name: "AuthResponse",
			marshal: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&AuthResponse{
					Signature: []byte{0xde, 0xad, 0xbe, 0xef},
					Address:   "0x" + strings.Repeat("1", 40),
				}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 82 = array(2); 44 deadbeef = bstr len 4; 782a = tstr len 42; 3078 = "0x"; then 40×'1'.
			wantHex: "8244deadbeef782a3078" + strings.Repeat("31", 40),
		},
		{
			name: "ReceiptAck true/ok",
			marshal: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&ReceiptAck{Accepted: true, Reason: "ok"}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 82 = array(2); f5 = true; 626f6b = tstr "ok".
			wantHex: "82f5626f6b",
		},
		{
			name: "ReceiptAck false/empty",
			marshal: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&ReceiptAck{Accepted: false, Reason: ""}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 82 = array(2); f4 = false; 60 = tstr len 0.
			wantHex: "82f460",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.marshal()
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			if gotHex := hex.EncodeToString(got); gotHex != tc.wantHex {
				t.Errorf("bytes = %s\n want = %s", gotHex, tc.wantHex)
			}
		})
	}
}

func TestWireRoundTrip(t *testing.T) {
	var nonce [32]byte
	nonce[0], nonce[31] = 0x11, 0x22

	t.Run("AuthChallenge", func(t *testing.T) {
		in := &AuthChallenge{Nonce: nonce}
		var buf bytes.Buffer
		if err := in.MarshalCBOR(&buf); err != nil {
			t.Fatal(err)
		}
		var out AuthChallenge
		if err := out.UnmarshalCBOR(&buf); err != nil {
			t.Fatal(err)
		}
		if out.Nonce != in.Nonce {
			t.Errorf("nonce mismatch")
		}
	})

	t.Run("AuthResponse", func(t *testing.T) {
		in := &AuthResponse{Signature: bytes.Repeat([]byte{0x7}, 65), Address: "0xDeadBeef"}
		var buf bytes.Buffer
		if err := in.MarshalCBOR(&buf); err != nil {
			t.Fatal(err)
		}
		var out AuthResponse
		if err := out.UnmarshalCBOR(&buf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.Signature, in.Signature) || out.Address != in.Address {
			t.Errorf("round-trip mismatch: %+v", out)
		}
	})

	t.Run("ReceiptAck wider ack", func(t *testing.T) {
		// A real clearnode emits a 6-element ack; the reader must accept it and
		// take only the first two fields (Accepted, Reason), skipping the rest.
		// 86 = array(6); f5 = true; 626f6b = "ok"; then 4 trailing ints 0..3.
		wire, err := hex.DecodeString("86f5626f6b00010203")
		if err != nil {
			t.Fatal(err)
		}
		var out ReceiptAck
		if err := out.UnmarshalCBOR(bytes.NewReader(wire)); err != nil {
			t.Fatalf("decode 6-element ack: %v", err)
		}
		if !out.Accepted || out.Reason != "ok" {
			t.Errorf("got %+v, want {Accepted:true Reason:ok}", out)
		}
	})

	t.Run("ReceiptAck", func(t *testing.T) {
		for _, in := range []*ReceiptAck{
			{Accepted: true, Reason: ""},
			{Accepted: false, Reason: "rejected: bad signature"},
		} {
			var buf bytes.Buffer
			if err := in.MarshalCBOR(&buf); err != nil {
				t.Fatal(err)
			}
			var out ReceiptAck
			if err := out.UnmarshalCBOR(&buf); err != nil {
				t.Fatal(err)
			}
			if out != *in {
				t.Errorf("round-trip mismatch: got %+v want %+v", out, *in)
			}
		}
	})
}
