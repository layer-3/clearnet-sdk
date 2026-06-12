package cborx

import "errors"

// Version is the one-byte schema-family prefix carried by every CBOR
// frame on the wire or in a BLOB. ADR-009 §5.
//
// Allocation:
//
//	0x00         reserved — never written; rejected on decode.
//	0x01         V1 — the initial CBOR migration shipped by this repo.
//	0x02..0x7F   reserved for future CBOR schema-family versions.
//	0x80..0xFF   reserved for future non-CBOR codecs (e.g. a later
//	             migration away from CBOR).
//
// A wire-incompatible change bumps the byte globally; there is no
// per-codec version and no backwards-compatible reader.
type Version uint8

const (
	// V1 is the CBOR schema-family version shipped by ADR-009.
	V1 Version = 0x01

	// VersionReserved is the guard value; readers reject it explicitly
	// so a zero-filled buffer can never be mistaken for a valid frame.
	VersionReserved Version = 0x00
)

// ErrReservedVersion reports a 0x00 version byte on decode.
var ErrReservedVersion = errors.New("cborx: reserved version byte 0x00")

// ErrUnsupportedVersion reports a version byte above V1 (or otherwise
// unknown to this binary). Callers can distinguish version-envelope
// errors from payload errors by unwrapping against this sentinel.
var ErrUnsupportedVersion = errors.New("cborx: unsupported schema version")

// ErrEmptyEnvelope reports that the version byte could not be read.
var ErrEmptyEnvelope = errors.New("cborx: empty envelope")
