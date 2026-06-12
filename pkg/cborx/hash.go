package cborx

import (
	"fmt"
	"io"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// Hash32Len / Addr20Len fix the declared definite lengths for the
// hash / address adapters. The length is part of the encoding
// contract, not a bound — a decoder that reads a different length
// rejects.
const (
	Hash32Len = 32
	Addr20Len = 20
)

// Hash32 wraps a 32-byte digest (keccak256 output, block hash, entry
// hash, state root, etc.). Encoded as CBOR major type 2 with a definite
// length of 32.
type Hash32 struct {
	V [Hash32Len]byte
}

// MarshalCBOR writes the 32-byte byte string with the canonical
// shortest-form length (a single byte in this case).
func (h Hash32) MarshalCBOR(w io.Writer) error {
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajByteString, Hash32Len); err != nil {
		return fmt.Errorf("cborx: Hash32: write header: %w", err)
	}
	if _, err := w.Write(h.V[:]); err != nil {
		return fmt.Errorf("cborx: Hash32: write body: %w", err)
	}
	return nil
}

// UnmarshalCBOR reads exactly 32 bytes into h.V; any other length is
// a hard reject.
func (h *Hash32) UnmarshalCBOR(r io.Reader) error {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: Hash32: read header: %w", err)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("cborx: Hash32: expected byte string (major 2), got major %d", maj)
	}
	if val != Hash32Len {
		return fmt.Errorf("cborx: Hash32: expected length %d, got %d", Hash32Len, val)
	}
	if _, err := io.ReadFull(r, h.V[:]); err != nil {
		return fmt.Errorf("cborx: Hash32: read body: %w", err)
	}
	return nil
}

// Addr20 wraps a 20-byte Ethereum-style address as a bare byte array
// so internal/cborx can stay free of outer-layer dependencies. Wave 2
// generated codecs pass a go-ethereum common.Address in via an explicit
// [20]byte conversion (Address is a named [20]byte type).
//
// Encoded as CBOR major type 2 with a definite length of 20. Round-trips
// bit-identically to the ABI encoding of a bare 20-byte address slice
// so later boundaries (e.g. custody/evm/) can splice the bytes without
// extra conversion.
type Addr20 struct {
	V [Addr20Len]byte
}

// MarshalCBOR writes the 20-byte byte string.
func (a Addr20) MarshalCBOR(w io.Writer) error {
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajByteString, Addr20Len); err != nil {
		return fmt.Errorf("cborx: Addr20: write header: %w", err)
	}
	if _, err := w.Write(a.V[:]); err != nil {
		return fmt.Errorf("cborx: Addr20: write body: %w", err)
	}
	return nil
}

// UnmarshalCBOR reads exactly 20 bytes into a.V; any other length is
// a hard reject.
func (a *Addr20) UnmarshalCBOR(r io.Reader) error {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: Addr20: read header: %w", err)
	}
	if maj != cbg.MajByteString {
		return fmt.Errorf("cborx: Addr20: expected byte string (major 2), got major %d", maj)
	}
	if val != Addr20Len {
		return fmt.Errorf("cborx: Addr20: expected length %d, got %d", Addr20Len, val)
	}
	if _, err := io.ReadFull(r, a.V[:]); err != nil {
		return fmt.Errorf("cborx: Addr20: read body: %w", err)
	}
	return nil
}
