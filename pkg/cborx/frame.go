package cborx

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// Frame size caps per docs/specs/cbor.md §5.2 / ADR-009 §8. Callers choose which cap
// applies to their stream: backfill (blocks/logs) is bulk, every other
// libp2p stream, the clearnet TCP listener, and every gossipsub
// message is control.
const (
	// MaxBulkFrame bounds a single length-prefixed frame for
	// block / log backfill streams (1 MB).
	MaxBulkFrame uint64 = 1 << 20

	// MaxControlFrame bounds a single length-prefixed frame for every
	// non-bulk libp2p stream, the clearnet TCP listener, and every
	// gossipsub message (64 KB).
	MaxControlFrame uint64 = 64 << 10
)

// ErrFrameTooLarge reports a declared frame length above the caller's
// cap. Callers should close the stream when this is returned — the
// remote is either mis-framed or attempting a DoS.
var ErrFrameTooLarge = errors.New("cborx: frame exceeds size cap")

// ErrFrameTrailingBytes reports a frame whose declared length contains bytes
// after the envelope body has decoded. ADR-009 requires strict decode at the
// frame boundary rather than silently accepting schema drift or concatenation.
var ErrFrameTrailingBytes = errors.New("cborx: trailing bytes in frame")

// WriteFrame writes a length-prefixed envelope to w:
//
//	[uvarint body-length][1 byte version][CBOR body]
//
// where body-length covers the version byte + CBOR body (i.e. the
// envelope that WriteEnvelope would produce). Used on streams that
// carry more than one frame per connection: the clearnet TCP listener
// (docs/specs/cbor.md §5.2), libp2p backfill protocols, and anywhere a caller wants
// explicit per-frame bounds before allocation.
//
// Single-message libp2p request/response streams use this framing so
// receivers get an explicit message boundary without waiting for stream EOF.
func WriteFrame(w io.Writer, v Version, body cbg.CBORMarshaler) error {
	var buf bytes.Buffer
	if err := WriteEnvelope(&buf, v, body); err != nil {
		return err
	}
	var hdr [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(hdr[:], uint64(buf.Len()))
	if _, err := w.Write(hdr[:n]); err != nil {
		return fmt.Errorf("cborx: write frame length: %w", err)
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("cborx: write frame body: %w", err)
	}
	return nil
}

// ReadFrame reads a length-prefixed envelope produced by WriteFrame.
// max is the largest body length this call will tolerate before
// returning ErrFrameTooLarge; callers pass MaxBulkFrame or
// MaxControlFrame depending on the stream.
//
// On success *v carries the observed version byte and body is filled
// via UnmarshalCBOR. A clean EOF before any bytes were read is
// reported as io.EOF (not wrapped) so callers can terminate backfill
// loops without special-casing ErrEmptyEnvelope.
func ReadFrame(r io.Reader, max uint64, v *Version, body cbg.CBORUnmarshaler) error {
	if v == nil {
		return errors.New("cborx: ReadFrame: nil *Version")
	}
	if body == nil {
		return errors.New("cborx: ReadFrame: nil body")
	}

	br := asByteReader(r)
	length, err := binary.ReadUvarint(br)
	if err != nil {
		if err == io.EOF {
			return io.EOF
		}
		return fmt.Errorf("cborx: read frame length: %w", err)
	}
	if length == 0 {
		return fmt.Errorf("%w", ErrEmptyEnvelope)
	}
	if length > max {
		return fmt.Errorf("%w: %d > %d", ErrFrameTooLarge, length, max)
	}

	// Bound the envelope reader to exactly `length` bytes so a mis-sized
	// frame can't bleed into the next frame's length prefix.
	lr := io.LimitReader(br, int64(length))
	if err := ReadEnvelope(lr, v, body); err != nil {
		return err
	}
	var extra [1]byte
	if n, err := lr.Read(extra[:]); n > 0 || err == nil {
		return fmt.Errorf("%w", ErrFrameTrailingBytes)
	} else if err != io.EOF {
		return fmt.Errorf("cborx: check frame trailing bytes: %w", err)
	}
	return nil
}

// byteReaderAdapter wraps an io.Reader as an io.ByteReader for
// binary.ReadUvarint. Reads one byte at a time — acceptable for the
// varint header (at most 10 bytes).
type byteReaderAdapter struct{ r io.Reader }

func (b byteReaderAdapter) ReadByte() (byte, error) {
	var buf [1]byte
	if _, err := io.ReadFull(b.r, buf[:]); err != nil {
		return 0, err
	}
	return buf[0], nil
}

// Read satisfies io.Reader so downstream callers (ReadFrame's
// io.LimitReader) can share one adapter across header + body.
func (b byteReaderAdapter) Read(p []byte) (int, error) { return b.r.Read(p) }

// asByteReader returns r itself when it already implements io.ByteReader,
// or wraps it in a one-byte-at-a-time adapter otherwise.
func asByteReader(r io.Reader) interface {
	io.Reader
	io.ByteReader
} {
	if br, ok := r.(interface {
		io.Reader
		io.ByteReader
	}); ok {
		return br
	}
	return byteReaderAdapter{r: r}
}
