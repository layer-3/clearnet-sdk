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
					IssuerID:  "0x" + strings.Repeat("2", 40),
				}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 83 = array(3); 44 deadbeef = bstr len 4; 782a = tstr len 42; then address and issuer id.
			wantHex: "8344deadbeef782a3078" + strings.Repeat("31", 40) + "782a3078" + strings.Repeat("32", 40),
		},
		{
			name: "ReceiptAck accepted",
			marshal: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&ReceiptAck{Code: ReceiptAckAccepted}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 82 = array(2); 686163636570746564 = tstr "accepted"; 60 = empty reason.
			wantHex: "8268616363657074656460",
		},
		{
			name: "ReceiptAck corrupt",
			marshal: func() ([]byte, error) {
				var buf bytes.Buffer
				err := (&ReceiptAck{Code: ReceiptAckCorrupt, Reason: "bad"}).MarshalCBOR(&buf)
				return buf.Bytes(), err
			},
			// 82 = array(2); 67636f7272757074 = tstr "corrupt"; 63626164 = "bad".
			wantHex: "8267636f727275707463626164",
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
		in := &AuthResponse{Signature: bytes.Repeat([]byte{0x7}, 65), Address: "0xDeadBeef", IssuerID: "0xIssuer"}
		var buf bytes.Buffer
		if err := in.MarshalCBOR(&buf); err != nil {
			t.Fatal(err)
		}
		var out AuthResponse
		if err := out.UnmarshalCBOR(&buf); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out.Signature, in.Signature) || out.Address != in.Address || out.IssuerID != in.IssuerID {
			t.Errorf("round-trip mismatch: %+v", out)
		}
	})

	t.Run("ReceiptAck wider ack", func(t *testing.T) {
		// The reader accepts wider future ACKs and takes only Code and Reason.
		// 86 = array(6); "accepted"; ""; then 4 trailing ints 0..3.
		wire, err := hex.DecodeString("866861636365707465646000010203")
		if err != nil {
			t.Fatal(err)
		}
		var out ReceiptAck
		if err := out.UnmarshalCBOR(bytes.NewReader(wire)); err != nil {
			t.Fatalf("decode 6-element ack: %v", err)
		}
		if out.Code != ReceiptAckAccepted || out.Reason != "" {
			t.Errorf("got %+v, want accepted", out)
		}
	})

	t.Run("ReceiptAck", func(t *testing.T) {
		for _, in := range []*ReceiptAck{
			{Code: ReceiptAckAccepted},
			{Code: ReceiptAckRejected, Reason: "bad signature"},
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

func TestReceiptAckValidate(t *testing.T) {
	for _, ack := range []ReceiptAck{
		{Code: ReceiptAckAccepted},
		{Code: ReceiptAckAlreadyAccepted},
		{Code: ReceiptAckStaleEpoch},
		{Code: ReceiptAckFutureEpoch},
		{Code: ReceiptAckSignerStateUnavailable},
		{Code: ReceiptAckTemporaryFailure, Reason: "retry"},
		{Code: ReceiptAckRejected, Reason: "semantic"},
		{Code: ReceiptAckCorrupt, Reason: "bad signature"},
	} {
		if err := ack.Validate(); err != nil {
			t.Fatalf("%+v failed validation: %v", ack, err)
		}
	}
	for _, ack := range []ReceiptAck{
		{},
		{Code: ReceiptAckCode("bogus")},
		{Code: ReceiptAckTemporaryFailure},
		{Code: ReceiptAckRejected},
		{Code: ReceiptAckCorrupt},
	} {
		if err := ack.Validate(); err == nil {
			t.Fatalf("%+v unexpectedly validated", ack)
		}
	}
}
