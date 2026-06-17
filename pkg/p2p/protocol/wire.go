package protocol

import (
	"fmt"
	"io"

	cbg "github.com/whyrusleeping/cbor-gen"
)

// TODO(sdk): migrate these handwritten CBOR codecs to cbor-gen (pkg/core/gen)
// once wire_test.go's golden vectors freeze the encoding. They are kept
// handwritten for now so the extraction is a byte-for-byte port of the
// established wire with no generator wiring.
//
// TODO(sdk): consider centralizing every p2p communication structure here —
// the auth challenge/response and receipt ack already live in this file, but
// the receipt bodies (core.BurnReceipt/MintReceipt) and topic event payloads
// (core.FinalizedWithdrawal, []core.Event, …) are defined elsewhere. Gathering
// the wire surface into one place with a consistent naming scheme — as
// erc7824/nitrolite does in pkg/rpc/api.go (structs) + pkg/rpc/methods.go
// (protocol/topic identifiers) — would make the full p2p contract readable at
// a glance. Blocked on deciding whether the shared core.* types should move or
// be re-exported, since they are also consumed off the wire.

// AuthChallenge is sent by the server (entry node) to a connecting peer at the
// start of the auth handshake: 32 random bytes scoped to a single attempt.
//
// Wire encoding: cborx V1 envelope wrapping a 1-tuple (Nonce [32]byte).
type AuthChallenge struct {
	Nonce [32]byte
}

// AuthResponse is the peer's reply after signing the nonce.
//
// Wire encoding: cborx V1 envelope wrapping a 2-tuple (Signature []byte,
// Address string). Operator auth sets Address and signs keccak256(Nonce) with
// the operator secp256k1 key (raw v=0/1 form). Passive auth leaves Address
// empty and signs a domain-separated nonce with the libp2p identity key; the
// Address field is then carried empty.
type AuthResponse struct {
	Signature []byte
	Address   string
}

// ReceiptAck is the server's response to a burn/mint receipt submission.
//
// Accepted is true when the server persisted the receipt or recognized it as a
// duplicate of an already-persisted one (the receipt path is idempotent on the
// natural keys the clearing layer de-dupes by, so retries are safe). Reason
// carries a short diagnostic when Accepted is false; empty otherwise.
//
// Wire encoding: cborx V1 frame wrapping a 2-tuple (Accepted bool, Reason
// string).
type ReceiptAck struct {
	Accepted bool
	Reason   string
}

var lengthBufAuthChallenge = []byte{0x81} // CBOR array, 1 element

// MarshalCBOR writes AuthChallenge as a 1-element CBOR array.
func (t *AuthChallenge) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	cw := cbg.NewCborWriter(w)
	if _, err := cw.Write(lengthBufAuthChallenge); err != nil {
		return err
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, 32); err != nil {
		return err
	}
	_, err := cw.Write(t.Nonce[:])
	return err
}

// UnmarshalCBOR reads AuthChallenge from a 1-element CBOR array.
func (t *AuthChallenge) UnmarshalCBOR(r io.Reader) error {
	*t = AuthChallenge{}
	cr := cbg.NewCborReader(r)
	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajArray || extra != 1 {
		return fmt.Errorf("AuthChallenge: expected 1-element CBOR array, got maj=%d extra=%d", maj, extra)
	}
	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajByteString || extra != 32 {
		return fmt.Errorf("AuthChallenge.Nonce: expected 32-byte string")
	}
	if _, err := io.ReadFull(cr, t.Nonce[:]); err != nil {
		return err
	}
	return nil
}

var lengthBufAuthResponse = []byte{0x82} // CBOR array, 2 elements

// MarshalCBOR writes AuthResponse as a 2-element CBOR array.
func (t *AuthResponse) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	cw := cbg.NewCborWriter(w)
	if _, err := cw.Write(lengthBufAuthResponse); err != nil {
		return err
	}
	if len(t.Signature) > cbg.MaxLength {
		return fmt.Errorf("AuthResponse.Signature too long")
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajByteString, uint64(len(t.Signature))); err != nil {
		return err
	}
	if _, err := cw.Write(t.Signature); err != nil {
		return err
	}
	if len(t.Address) > cbg.MaxLength {
		return fmt.Errorf("AuthResponse.Address too long")
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Address))); err != nil {
		return err
	}
	_, err := cw.WriteString(t.Address)
	return err
}

// UnmarshalCBOR reads AuthResponse from a 2-element CBOR array.
func (t *AuthResponse) UnmarshalCBOR(r io.Reader) error {
	*t = AuthResponse{}
	cr := cbg.NewCborReader(r)
	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajArray || extra != 2 {
		return fmt.Errorf("AuthResponse: expected 2-element CBOR array")
	}
	// Signature (byte string).
	maj, extra, err = cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("AuthResponse.Signature: expected byte string")
	}
	if extra > 1024 {
		return fmt.Errorf("AuthResponse.Signature: implausibly large (%d)", extra)
	}
	t.Signature = make([]byte, extra)
	if _, err := io.ReadFull(cr, t.Signature); err != nil {
		return err
	}
	// Address (text string).
	addr, err := cbg.ReadString(cr)
	if err != nil {
		return fmt.Errorf("AuthResponse.Address: %w", err)
	}
	t.Address = addr
	return nil
}

var lengthBufReceiptAck = []byte{0x82} // CBOR array, 2 elements

// MarshalCBOR writes ReceiptAck as a 2-element CBOR array.
func (t *ReceiptAck) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	cw := cbg.NewCborWriter(w)
	if _, err := cw.Write(lengthBufReceiptAck); err != nil {
		return err
	}
	if err := cbg.WriteBool(cw, t.Accepted); err != nil {
		return err
	}
	if len(t.Reason) > cbg.MaxLength {
		return fmt.Errorf("ReceiptAck.Reason too long (%d)", len(t.Reason))
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.Reason))); err != nil {
		return err
	}
	_, err := cw.WriteString(t.Reason)
	return err
}

// UnmarshalCBOR reads ReceiptAck from a 2-element CBOR array.
func (t *ReceiptAck) UnmarshalCBOR(r io.Reader) error {
	*t = ReceiptAck{}
	cr := cbg.NewCborReader(r)
	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajArray || extra != 2 {
		return fmt.Errorf("ReceiptAck: expected 2-element CBOR array")
	}
	// Accepted (CBOR simple value: 20 = false, 21 = true).
	bmaj, bminor, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	if bmaj != cbg.MajOther {
		return fmt.Errorf("ReceiptAck.Accepted: expected bool, got major %d", bmaj)
	}
	switch bminor {
	case 20:
		t.Accepted = false
	case 21:
		t.Accepted = true
	default:
		return fmt.Errorf("ReceiptAck.Accepted: unexpected minor %d", bminor)
	}
	reason, err := cbg.ReadString(cr)
	if err != nil {
		return fmt.Errorf("ReceiptAck.Reason: %w", err)
	}
	t.Reason = reason
	return nil
}
