// Package cborx provides shared primitive-adapter codecs for the
// canonical deterministic CBOR (RFC 8949 §4.2) encoding used across the
// Yellow Network protocol.
//
// This is the foundation package for the canonical CBOR rules in
// docs/specs/cbor.md and ADR-009 (docs/decisions/009-cbor-encoding.md §3 —
// the adapter table, §5 — the version byte envelope).
//
// # Scope
//
// cborx does NOT generate per-type struct codecs. It only wraps
// whyrusleeping/cbor-gen primitives with adapters so that generated
// codecs (owned by later waves under each package's gen/ subfolder)
// can delegate the encoding of complex primitive types — big integers,
// fixed-point decimals, fixed-length hashes and addresses, and timestamps
// — to one canonical implementation that satisfies RFC 8949 §4.2
// deterministic-encoding rules.
//
// # Adapter set (ADR-009 §3)
//
//   - BigInt        — *big.Int as RFC 8949 tag 2 (unsigned bignum) or
//     tag 3 (negative bignum); zero is tag 2 with empty
//     byte string; nil is invalid inside a plain BigInt.
//   - MaybeBigInt   — nilable *big.Int; nil encodes as CBOR null.
//   - Decimal       — internal/decimal.Decimal as RFC 8949 tag 4
//     (decimal fraction) = [exponent int32, mantissa bigint].
//   - Hash32        — [32]byte as major type 2, definite length 32.
//   - Addr20        — common.Address / [20]byte as major type 2,
//     definite length 20.
//   - Time          — time.Time as major type 0/1, Unix nanoseconds int64.
//
// # Envelope (ADR-009 §5)
//
// Every CBOR frame on the wire or in a BLOB is prefixed with one byte
// (the global schema-family version). WriteEnvelope / ReadEnvelope
// handle the prefix around a cbor-gen CBORMarshaler / CBORUnmarshaler
// body. V1 (0x01) is the only version emitted by this migration;
// ReadEnvelope rejects 0x00 (reserved) and anything above V1 with a
// typed ErrUnsupportedVersion so callers can distinguish framing errors
// from payload errors.
//
// # Determinism (RFC 8949 §4.2)
//
// Adapters enforce — and tests prove — the deterministic-encoding rules in
// docs/specs/cbor.md §2:
//
//  1. Integers in shortest form (no over-wide length encoding).
//  2. Definite-length containers only; indefinite-length input is rejected.
//  3. Map keys sorted lexicographically by their CBOR-encoded bytes
//     (RFC 8949 §4.2.1). Adapters themselves hold no maps; the envelope
//     helpers preserve sort order produced by cbor-gen.
//  4. No floating-point types at all. NaN, infinities, and negative
//     zero are never encoded and are rejected on decode.
//  5. Tagged integers only where declared (tag 2/3 for bignums, tag 4
//     for decimals).
//
// # cbor-gen mode
//
// ADR-009 §2 mandates **struct-as-array (positional, no keys on wire)**
// for every generated type in the network. Wave 2's per-package
// gen/main.go must invoke cbor-gen with the tuple-encoding entry point
// (typegen.WriteTupleEncodersToFile) rather than the map entry point.
// Fields in generated structs may only be appended at the end; insert,
// reorder, rename, or remove requires a version-byte bump and a re-seed
// of every CBOR-encoded BLOB (ADR-009 §5).
//
// # Imports
//
// This package imports whyrusleeping/cbor-gen and fxamacker/cbor/v2.
// internal/ is stdlib-only under the repo's architecture rules; the
// relevant import lines carry the `layer-guard: allow` escape hatch
// because ADR-009 and the migration plan explicitly sanction cborx as
// the single package that wraps these libraries.
package cborx
