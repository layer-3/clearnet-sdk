package protocol

import (
	"fmt"
	"io"

	"github.com/ipfs/go-cid"
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
// Wire encoding: cborx V1 envelope wrapping a 3-tuple (Signature []byte,
// Address string, IssuerID string). Operator auth sets Address and IssuerID and
// signs a domain-separated digest over Nonce + IssuerID. Passive auth leaves
// Address and IssuerID empty and signs a domain-separated nonce with the libp2p
// identity key.
type AuthResponse struct {
	Signature []byte
	Address   string
	IssuerID  string
}

type ReceiptAckCode string

const (
	ReceiptAckAccepted               ReceiptAckCode = "accepted"
	ReceiptAckAlreadyAccepted        ReceiptAckCode = "already_accepted"
	ReceiptAckStaleEpoch             ReceiptAckCode = "stale_epoch"
	ReceiptAckFutureEpoch            ReceiptAckCode = "future_epoch"
	ReceiptAckSignerStateUnavailable ReceiptAckCode = "signer_state_unavailable"
	ReceiptAckTemporaryFailure       ReceiptAckCode = "temporary_failure"
	ReceiptAckRejected               ReceiptAckCode = "rejected"
	ReceiptAckCorrupt                ReceiptAckCode = "corrupt"
)

// ReceiptAck is the server's response to a custody-to-clearnet receipt ingress
// submission. Reason is diagnostic only and must be non-empty for rejected,
// corrupt, and temporary_failure.
//
// Wire encoding: cborx V1 frame wrapping a 2-tuple (Code string, Reason string).
type ReceiptAck struct {
	Code   ReceiptAckCode
	Reason string
}

func (t ReceiptAck) Validate() error {
	switch t.Code {
	case ReceiptAckAccepted, ReceiptAckAlreadyAccepted, ReceiptAckStaleEpoch, ReceiptAckFutureEpoch, ReceiptAckSignerStateUnavailable:
		return nil
	case ReceiptAckTemporaryFailure, ReceiptAckRejected, ReceiptAckCorrupt:
		if t.Reason == "" {
			return fmt.Errorf("ReceiptAck.%s requires reason", t.Code)
		}
		return nil
	case "":
		return fmt.Errorf("ReceiptAck.Code is empty")
	default:
		return fmt.Errorf("ReceiptAck.Code unknown: %s", t.Code)
	}
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

var lengthBufAuthResponse = []byte{0x83} // CBOR array, 3 elements

// MarshalCBOR writes AuthResponse as a 3-element CBOR array.
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
	if _, err := cw.WriteString(t.Address); err != nil {
		return err
	}
	if len(t.IssuerID) > cbg.MaxLength {
		return fmt.Errorf("AuthResponse.IssuerID too long")
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(t.IssuerID))); err != nil {
		return err
	}
	_, err := cw.WriteString(t.IssuerID)
	return err
}

// UnmarshalCBOR reads AuthResponse from a 3-element CBOR array.
func (t *AuthResponse) UnmarshalCBOR(r io.Reader) error {
	*t = AuthResponse{}
	cr := cbg.NewCborReader(r)
	maj, extra, err := cr.ReadHeader()
	if err != nil {
		return err
	}
	if maj != cbg.MajArray || extra != 3 {
		return fmt.Errorf("AuthResponse: expected 3-element CBOR array")
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
	issuerID, err := cbg.ReadString(cr)
	if err != nil {
		return fmt.Errorf("AuthResponse.IssuerID: %w", err)
	}
	t.IssuerID = issuerID
	return nil
}

var lengthBufReceiptAck = []byte{0x82} // CBOR array, 2 elements

// MarshalCBOR writes ReceiptAck as a 2-element CBOR array.
func (t *ReceiptAck) MarshalCBOR(w io.Writer) error {
	if t == nil {
		_, err := w.Write(cbg.CborNull)
		return err
	}
	if err := t.Validate(); err != nil {
		return err
	}
	cw := cbg.NewCborWriter(w)
	if _, err := cw.Write(lengthBufReceiptAck); err != nil {
		return err
	}
	code := string(t.Code)
	if len(code) > cbg.MaxLength {
		return fmt.Errorf("ReceiptAck.Code too long (%d)", len(code))
	}
	if err := cw.WriteMajorTypeHeader(cbg.MajTextString, uint64(len(code))); err != nil {
		return err
	}
	if _, err := cw.WriteString(code); err != nil {
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
	// Accept >= 2 elements and skip any trailing fields for forward-compatible
	// readers. Only the first two fields are part of this contract.
	if maj != cbg.MajArray {
		return fmt.Errorf("ReceiptAck: expected CBOR array, got major %d", maj)
	}
	if extra < 2 {
		return fmt.Errorf("ReceiptAck: expected >=2 elements, got %d", extra)
	}
	code, err := cbg.ReadString(cr)
	if err != nil {
		return fmt.Errorf("ReceiptAck.Code: %w", err)
	}
	t.Code = ReceiptAckCode(code)
	reason, err := cbg.ReadString(cr)
	if err != nil {
		return fmt.Errorf("ReceiptAck.Reason: %w", err)
	}
	t.Reason = reason
	if !isKnownReceiptAckCode(t.Code) {
		if t.Reason == "" {
			t.Reason = fmt.Sprintf("unsupported receipt ACK code: %s", code)
		}
		t.Code = ReceiptAckTemporaryFailure
	}
	// Skip any trailing elements (ScanForLinks walks exactly one CBOR item per
	// call, recursing into arrays/maps); the CID sink is a no-op.
	noLink := func(cid.Cid) {}
	for i := uint64(2); i < extra; i++ {
		if err := cbg.ScanForLinks(cr, noLink); err != nil {
			return fmt.Errorf("ReceiptAck: skip trailing element %d: %w", i, err)
		}
	}
	return t.Validate()
}

func isKnownReceiptAckCode(code ReceiptAckCode) bool {
	switch code {
	case ReceiptAckAccepted, ReceiptAckAlreadyAccepted, ReceiptAckStaleEpoch, ReceiptAckFutureEpoch,
		ReceiptAckSignerStateUnavailable, ReceiptAckTemporaryFailure, ReceiptAckRejected, ReceiptAckCorrupt:
		return true
	default:
		return false
	}
}
