package cborx

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// ErrEnvelopeTrailingBytes reports an envelope decoded from a bounded
// reader (byte slice or io.LimitReader) that contains bytes after the
// canonical body. Canonical CBOR (ADR-009 §1, RFC 8949 §4.2) requires
// exactly one logical value per encoded byte string; trailing bytes
// would admit multiple wire encodings for the same logical value.
var ErrEnvelopeTrailingBytes = errors.New("cborx: trailing bytes after envelope body")

// WriteEnvelope writes the one-byte schema-family version followed by
// the canonical CBOR body produced by body.MarshalCBOR. It is the sole
// sanctioned way to stamp a frame before it hits a stream or a BLOB
// (ADR-009 §5).
//
// v MUST be V1 for this migration; any other value returns a wrapped
// ErrUnsupportedVersion or ErrReservedVersion so callers cannot
// accidentally write a frame with a version this build doesn't know
// how to read back.
func WriteEnvelope(w io.Writer, v Version, body cbg.CBORMarshaler) error {
	if v == VersionReserved {
		return fmt.Errorf("%w: cannot write 0x00", ErrReservedVersion)
	}
	if v != V1 {
		return fmt.Errorf("%w: write refused for v=0x%02x (this build writes V1 only)", ErrUnsupportedVersion, uint8(v))
	}
	if body == nil {
		return errors.New("cborx: WriteEnvelope: nil body")
	}
	if _, err := w.Write([]byte{byte(v)}); err != nil {
		return fmt.Errorf("cborx: write version byte: %w", err)
	}
	if err := body.MarshalCBOR(w); err != nil {
		return fmt.Errorf("cborx: marshal body: %w", err)
	}
	return nil
}

// ReadEnvelope reads the one-byte schema-family version, rejects
// reserved / unsupported values with a typed error, and decodes the
// remainder of the stream into body via body.UnmarshalCBOR.
//
// On return:
//   - *v is set to the observed version byte on success, or the
//     rejected byte when the error is one of ErrReservedVersion or
//     ErrUnsupportedVersion (so callers can log the raw byte).
//   - A body-level decode failure is returned unwrapped from either
//     version sentinel, which lets callers distinguish framing errors
//     from payload errors via errors.Is.
func ReadEnvelope(r io.Reader, v *Version, body cbg.CBORUnmarshaler) error {
	if v == nil {
		return errors.New("cborx: ReadEnvelope: nil *Version")
	}
	if body == nil {
		return errors.New("cborx: ReadEnvelope: nil body")
	}

	var buf [1]byte
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		if (err == io.EOF || err == io.ErrUnexpectedEOF) && n == 0 {
			return fmt.Errorf("%w", ErrEmptyEnvelope)
		}
		return fmt.Errorf("cborx: read version byte: %w", err)
	}

	ver := Version(buf[0])
	*v = ver

	switch {
	case ver == VersionReserved:
		return fmt.Errorf("%w", ErrReservedVersion)
	case ver == V1:
		// ok
	case ver > V1:
		return fmt.Errorf("%w: 0x%02x", ErrUnsupportedVersion, uint8(ver))
	}

	if err := body.UnmarshalCBOR(r); err != nil {
		return fmt.Errorf("cborx: unmarshal body: %w", err)
	}
	return nil
}

// ReadEnvelopeStrict reads an envelope and additionally requires that
// the underlying reader is exhausted after the body decodes. Use this
// at every byte-slice and bounded-stream (io.LimitReader) boundary to
// reject non-canonical inputs that would otherwise round-trip to a
// shorter canonical encoding.
//
// Mirrors the trailing-byte check in ReadFrame (see frame.go). Stream-
// based callers that intentionally pipeline multiple envelopes on one
// connection should keep using ReadEnvelope; ReadEnvelopeStrict is for
// the canonical-single-value boundary.
func ReadEnvelopeStrict(r io.Reader, v *Version, body cbg.CBORUnmarshaler) error {
	if err := ReadEnvelope(r, v, body); err != nil {
		return err
	}
	var extra [1]byte
	n, err := r.Read(extra[:])
	if n > 0 {
		return fmt.Errorf("%w", ErrEnvelopeTrailingBytes)
	}
	if err == nil || err == io.EOF {
		return nil
	}
	return fmt.Errorf("cborx: check envelope trailing bytes: %w", err)
}

// UnmarshalExact decodes a single canonical CBOR value from data into
// body and rejects any trailing bytes. Use for raw-CBOR boundaries that
// do not carry a version envelope (e.g. BlockEntry.Payload).
func UnmarshalExact(data []byte, body cbg.CBORUnmarshaler) error {
	if body == nil {
		return errors.New("cborx: UnmarshalExact: nil body")
	}
	r := bytes.NewReader(data)
	if err := body.UnmarshalCBOR(r); err != nil {
		return fmt.Errorf("cborx: unmarshal body: %w", err)
	}
	if r.Len() != 0 {
		return fmt.Errorf("%w: %d bytes remaining", ErrEnvelopeTrailingBytes, r.Len())
	}
	return nil
}
