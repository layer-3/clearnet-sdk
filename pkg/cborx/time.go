package cborx

import (
	"fmt"
	"io"
	"math"
	"time"

	cbg "github.com/whyrusleeping/cbor-gen" // layer-guard: allow
)

// Time is a CBOR adapter for time.Time. Encoding is Unix nanoseconds
// as a signed CBOR integer:
//
//   - positive (or zero) → major type 0 with shortest-form width.
//   - negative (pre-epoch)  → major type 1 with shortest-form width.
//
// This is the same encoding used by cbor-gen's CborTime helper, chosen
// over RFC 8949 tag 0/1 because:
//
//   - Our timestamps are already Unix-normalized (no sub-second-range
//     tricks, no text form).
//   - Positional struct-as-array codegen is friendlier to bare integers
//     than to tagged wrappers.
//   - int64 nanoseconds give a range of ±292 years around 1970
//     (roughly 1678-09-21..2262-04-11), which comfortably covers every
//     network timestamp. Times outside that range fail closed rather than
//     relying on time.Time.UnixNano(), which silently overflows.
type Time struct {
	V time.Time
}

var (
	minUnixNanoTime = time.Unix(0, math.MinInt64).UTC()
	maxUnixNanoTime = time.Unix(0, math.MaxInt64).UTC()
)

// MarshalCBOR writes the Unix-nanosecond integer.
func (t Time) MarshalCBOR(w io.Writer) error {
	if t.V.Before(minUnixNanoTime) || t.V.After(maxUnixNanoTime) {
		return fmt.Errorf("cborx: Time: %s outside int64 Unix-nanosecond range", t.V)
	}
	nsecs := t.V.UnixNano()
	if nsecs >= 0 {
		if err := cbg.WriteMajorTypeHeader(w, cbg.MajUnsignedInt, uint64(nsecs)); err != nil {
			return fmt.Errorf("cborx: Time: write positive: %w", err)
		}
		return nil
	}
	if err := cbg.WriteMajorTypeHeader(w, cbg.MajNegativeInt, uint64(-nsecs)-1); err != nil {
		return fmt.Errorf("cborx: Time: write negative: %w", err)
	}
	return nil
}

// UnmarshalCBOR decodes a shortest-form signed int as Unix nanoseconds.
// A major-type mismatch (e.g. float, tagged value, byte string) is a
// hard reject — ADR-009 §3 forbids float-encoded timestamps anywhere
// in the protocol.
func (t *Time) UnmarshalCBOR(r io.Reader) error {
	maj, val, err := cbg.CborReadHeader(r)
	if err != nil {
		return fmt.Errorf("cborx: Time: read header: %w", err)
	}
	var nsecs int64
	switch maj {
	case cbg.MajUnsignedInt:
		if val > math.MaxInt64 {
			return fmt.Errorf("cborx: Time: positive ns %d overflows int64", val)
		}
		nsecs = int64(val)
	case cbg.MajNegativeInt:
		if val > math.MaxInt64 {
			return fmt.Errorf("cborx: Time: negative ns overflows int64")
		}
		nsecs = -int64(val) - 1
	default:
		return fmt.Errorf("cborx: Time: expected integer, got major %d", maj)
	}
	t.V = time.Unix(0, nsecs).UTC()
	return nil
}
