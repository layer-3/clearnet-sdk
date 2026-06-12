package core

// Block-entry payload encoding. Per ADR-009 §4, `BlockEntry.Payload`
// bytes are canonical CBOR (cbor-gen tuple codec of the op struct) —
// no version-byte envelope, since BlockEntry.Payload is a nested byte
// string inside the already-enveloped Block.

import (
	"bytes"
	"fmt"
	"io"

	"github.com/layer-3/clearnet-sdk/pkg/cborx"
)

// cborPayloadMarshaler / cborPayloadUnmarshaler narrow the interface
// exactly to what cbor-gen's tuple emitter produces. Kept file-local so
// they do not leak into the exported surface.
type (
	cborPayloadMarshaler interface {
		MarshalCBOR(w io.Writer) error
	}
	cborPayloadUnmarshaler interface {
		UnmarshalCBOR(r io.Reader) error
	}
)

// marshalPayload encodes a codec target into a fresh byte slice using
// the generated CBOR tuple codec. The signature mirrors the old
// hand-rolled `op.Encode() []byte` so call-sites need no update.
func marshalPayload(op cborPayloadMarshaler) []byte {
	var buf bytes.Buffer
	if err := op.MarshalCBOR(&buf); err != nil {
		// Writing to bytes.Buffer cannot fail; a non-nil error means the
		// codec itself refused to serialize a required field — a
		// programmer error we surface loudly.
		panic(fmt.Errorf("core: payload MarshalCBOR: %w", err))
	}
	return buf.Bytes()
}

// unmarshalPayload decodes payload bytes into the given op using the
// generated CBOR tuple codec. Returns a wrapped error on any decode
// failure (truncated input, wrong type, non-canonical bytes).
//
// ADR-009 §1 requires a single canonical CBOR value per encoded byte
// string; cborx.UnmarshalExact rejects trailing bytes so a non-canonical
// payload cannot round-trip to a shorter canonical encoding.
func unmarshalPayload(payload []byte, op cborPayloadUnmarshaler) error {
	if err := cborx.UnmarshalExact(payload, op); err != nil {
		return fmt.Errorf("payload UnmarshalCBOR: %w", err)
	}
	return nil
}

// DecodePayload decodes a BlockEntry's binary payload into the typed
// operation struct based on entry.Type. Returns one of:
//
//	*TransferOp, *SwapOp, *WithdrawalOp, *RepegOp,
//	*SessionCloseOp, *SessionChallengeOp
//
// Returns an error for unknown entry types or malformed payloads.
func DecodePayload(entry BlockEntry) (interface{}, error) {
	switch entry.Type {
	case OpTransfer:
		op := &TransferOp{}
		if err := op.Decode(entry.Payload); err != nil {
			return nil, fmt.Errorf("decode transfer payload: %w", err)
		}
		return op, nil
	case OpSwap:
		op := &SwapOp{}
		if err := op.Decode(entry.Payload); err != nil {
			return nil, fmt.Errorf("decode swap payload: %w", err)
		}
		return op, nil
	case OpWithdrawal:
		op := &WithdrawalOp{}
		if err := op.Decode(entry.Payload); err != nil {
			return nil, fmt.Errorf("decode withdrawal payload: %w", err)
		}
		return op, nil
	case OpRepeg:
		op := &RepegOp{}
		if err := op.Decode(entry.Payload); err != nil {
			return nil, fmt.Errorf("decode repeg payload: %w", err)
		}
		return op, nil
	case OpBurn:
		v := &BurnReceipt{}
		if err := unmarshalPayload(entry.Payload, v); err != nil {
			return nil, fmt.Errorf("decode burn receipt payload: %w", err)
		}
		return v, nil
	case OpMint:
		v := &MintReceipt{}
		if err := unmarshalPayload(entry.Payload, v); err != nil {
			return nil, fmt.Errorf("decode mint receipt payload: %w", err)
		}
		return v, nil
	case OpSessionClose:
		op := &SessionCloseOp{}
		if err := op.Decode(entry.Payload); err != nil {
			return nil, fmt.Errorf("decode session close payload: %w", err)
		}
		return op, nil
	case OpSessionChallenge:
		op := &SessionChallengeOp{}
		if err := op.Decode(entry.Payload); err != nil {
			return nil, fmt.Errorf("decode session challenge payload: %w", err)
		}
		return op, nil
	default:
		return nil, fmt.Errorf("unknown entry type: %d", entry.Type)
	}
}

// ---------------------------------------------------------------------------
// TransferOp
// ---------------------------------------------------------------------------

// Encode serializes the TransferOp into a block entry payload
// (canonical CBOR, ADR-009 §4).
func (op *TransferOp) Encode() []byte { return marshalPayload(op) }

// Decode deserializes a TransferOp from a block entry payload.
func (op *TransferOp) Decode(payload []byte) error { return unmarshalPayload(payload, op) }

// ---------------------------------------------------------------------------
// SwapOp
// ---------------------------------------------------------------------------

// Encode serializes the SwapOp into a block entry payload.
func (op *SwapOp) Encode() []byte { return marshalPayload(op) }

// Decode deserializes a SwapOp from a block entry payload.
func (op *SwapOp) Decode(payload []byte) error { return unmarshalPayload(payload, op) }

// ---------------------------------------------------------------------------
// WithdrawalOp
// ---------------------------------------------------------------------------

// Encode serializes the WithdrawalOp into a block entry payload.
func (op *WithdrawalOp) Encode() []byte { return marshalPayload(op) }

// Decode deserializes a WithdrawalOp from a block entry payload.
func (op *WithdrawalOp) Decode(payload []byte) error { return unmarshalPayload(payload, op) }

// ---------------------------------------------------------------------------
// RepegOp
// ---------------------------------------------------------------------------

// Encode serializes the RepegOp into a block entry payload.
func (op *RepegOp) Encode() []byte { return marshalPayload(op) }

// Decode deserializes a RepegOp from a block entry payload.
func (op *RepegOp) Decode(payload []byte) error { return unmarshalPayload(payload, op) }

// ---------------------------------------------------------------------------
// MintReceipt / BurnReceipt)
// ---------------------------------------------------------------------------

// EncodePayload serializes a MintReceipt into a BlockEntry payload.
func (v *MintReceipt) EncodePayload() []byte { return marshalPayload(v) }

// EncodePayload serializes a BurnReceipt into a BlockEntry payload.
func (v *BurnReceipt) EncodePayload() []byte { return marshalPayload(v) }

// ---------------------------------------------------------------------------
// SessionCloseOp
// ---------------------------------------------------------------------------

// Encode serializes the SessionCloseOp into a block entry payload.
func (op *SessionCloseOp) Encode() []byte { return marshalPayload(op) }

// Decode deserializes a SessionCloseOp from a block entry payload.
func (op *SessionCloseOp) Decode(payload []byte) error { return unmarshalPayload(payload, op) }

// EncodeSessionCloseOp serializes a SessionCloseOp.
func EncodeSessionCloseOp(op *SessionCloseOp) []byte { return op.Encode() }

// DecodeSessionCloseOp deserializes a SessionCloseOp.
func DecodeSessionCloseOp(data []byte) (*SessionCloseOp, error) {
	op := &SessionCloseOp{}
	if err := op.Decode(data); err != nil {
		return nil, err
	}
	return op, nil
}

// ---------------------------------------------------------------------------
// SessionChallengeOp
// ---------------------------------------------------------------------------

// Encode serializes the SessionChallengeOp into a block entry payload.
func (op *SessionChallengeOp) Encode() []byte { return marshalPayload(op) }

// Decode deserializes a SessionChallengeOp from a block entry payload.
func (op *SessionChallengeOp) Decode(payload []byte) error { return unmarshalPayload(payload, op) }

// EncodeSessionChallengeOp serializes a SessionChallengeOp.
func EncodeSessionChallengeOp(op *SessionChallengeOp) []byte { return op.Encode() }

// DecodeSessionChallengeOp deserializes a SessionChallengeOp.
func DecodeSessionChallengeOp(data []byte) (*SessionChallengeOp, error) {
	op := &SessionChallengeOp{}
	if err := op.Decode(data); err != nil {
		return nil, err
	}
	return op, nil
}
