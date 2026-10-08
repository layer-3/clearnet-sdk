import { ClearnetSdkError } from "../../core/errors.js";
import { concatBytes } from "../../core/bytes.js";

/**
 * ADR-023 deposit-attribution marker: a zero-value OP_RETURN output carried
 * alongside every generic-address BTC deposit (ADR §2, §4). TS has no shared
 * code with the Go `marker` package, so this module is a direct, byte-for-byte
 * mirror of it, and is asserted against the same golden vectors
 * (testdata/btc/marker_vectors.json) in test/blockchain/btc/marker.test.ts.
 * Magic, version bytes and the generic deposit tag preimage are compile-time
 * constants here too - never config, never caller-supplied.
 */

const TEXT_ENCODER = new TextEncoder();

/** 4-byte prefix every marker payload begins with. */
export const MARKER_MAGIC = TEXT_ENCODER.encode("YNET");

/** Marker payload versions. An unknown version is a hard construction failure. */
export const MARKER_VERSION_1 = 1; // "YNET" || 0x01 || address(20)             = 25 bytes
export const MARKER_VERSION_2 = 2; // "YNET" || 0x02 || address(20) || ref(32)  = 57 bytes

export type BitcoinMarkerVersion = typeof MARKER_VERSION_1 | typeof MARKER_VERSION_2;

/**
 * ADR-023's single generic deposit tag. Every BTC deposit pays the one P2WSH
 * address derived from this tag. Pinned as a literal, mirroring
 * `pkg/blockchain/btc/marker/tag.go`, so a typo in the preimage cannot
 * silently change the tag.
 */
export const GENERIC_DEPOSIT_TAG_PREIMAGE = "yellow/btc/generic-deposit/v1";
export const GENERIC_DEPOSIT_TAG_HEX =
  "791019971cfb071da60191897acda17879f6866754c6bdbee1a312a4ee33a8ac";

const ADDRESS_LEN = 20;
const REFERENCE_LEN = 32;

/** Opcode every marker scriptPubKey begins with. */
const OP_RETURN = 0x6a;

export interface BitcoinMarker {
  version: BitcoinMarkerVersion;
  /** 20-byte Clearnet account address. There is no account whose identifier is all-zero. */
  address: Uint8Array;
  /** 32-byte opaque reference. Version 1 carries none; version 2 requires a non-zero value. */
  reference?: Uint8Array | undefined;
}

/**
 * Serialises marker to its wire payload (25 or 57 bytes). Refuses an unknown
 * version, an all-zero address, and - for version 2 - a missing or all-zero
 * reference. The writer enforces exactly the rules the Go reader does, so a
 * construction bug cannot emit a marker the format would reject.
 */
export function encodeMarkerPayload(marker: BitcoinMarker): Uint8Array {
  switch (marker.version) {
    case MARKER_VERSION_1:
      return concatBytes(
        MARKER_MAGIC,
        Uint8Array.of(MARKER_VERSION_1),
        requireNonZeroAddress(marker.address),
      );
    case MARKER_VERSION_2:
      return concatBytes(
        MARKER_MAGIC,
        Uint8Array.of(MARKER_VERSION_2),
        requireNonZeroAddress(marker.address),
        requireNonZeroReference(marker.reference),
      );
    default:
      throw new ClearnetSdkError(
        "INVALID_INPUT",
        `btc: unknown marker version ${String(marker.version)}`,
      );
  }
}

/**
 * Serialises marker to a complete zero-value OP_RETURN scriptPubKey:
 * OP_RETURN <canonical direct push of len(payload)> <payload> - 27 bytes for
 * version 1, 59 for version 2. Every marker payload is well under 76 bytes,
 * so the push opcode is always the canonical direct form (opcode == payload
 * length): the only push encoding `marker.go::DecodeScript` accepts.
 * OP_PUSHDATA1/2/4 are rejected on the way back in, so this writer must never
 * emit one.
 */
export function encodeMarkerScript(marker: BitcoinMarker): Uint8Array {
  const payload = encodeMarkerPayload(marker);
  return concatBytes(Uint8Array.of(OP_RETURN, payload.length), payload);
}

/** True if every byte of bytes is 0x00. Exported so callers can pick a marker version from a reference. */
export function isZeroBytes(bytes: Uint8Array): boolean {
  return bytes.every((byte) => byte === 0);
}

function requireNonZeroAddress(address: Uint8Array): Uint8Array {
  if (address.length !== ADDRESS_LEN) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "btc: marker address must be a 20-byte address",
    );
  }
  if (isZeroBytes(address)) {
    throw new ClearnetSdkError("INVALID_ADDRESS", "btc: marker address is all-zero");
  }
  return address;
}

function requireNonZeroReference(reference: Uint8Array | undefined): Uint8Array {
  if (reference === undefined || reference.length !== REFERENCE_LEN) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "btc: version 0x02 marker requires a 32-byte reference",
    );
  }
  if (isZeroBytes(reference)) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "btc: version 0x02 marker reference is all-zero",
    );
  }
  return reference;
}

/**
 * Error codes of the marker reader, named as in
 * testdata/btc/marker_vectors.json. "not_marker" means the
 * scriptPubKey is not a marker candidate; every other code from
 * decodeMarkerScript means it is an invalid candidate, which leaves the whole
 * transaction unattributed. "no_marker" and "multiple_markers" come from
 * scanMarkerOutputs only.
 */
export type BitcoinMarkerErrorCode =
  | "not_marker"
  | "multiple_pushes"
  | "non_canonical_push"
  | "unknown_version"
  | "bad_length"
  | "zero_address"
  | "zero_reference"
  | "no_marker"
  | "multiple_markers";

export type BitcoinMarkerResult =
  | { ok: true; marker: BitcoinMarker }
  | { ok: false; error: BitcoinMarkerErrorCode };

const PAYLOAD_LEN_V1 = MARKER_MAGIC.length + 1 + ADDRESS_LEN;
const PAYLOAD_LEN_V2 = PAYLOAD_LEN_V1 + REFERENCE_LEN;
const OP_PUSHDATA1 = 0x4c;
const OP_PUSHDATA2 = 0x4d;
const OP_PUSHDATA4 = 0x4e;

/** Parses a bare marker payload (no script wrapper). Mirrors marker.go::DecodePayload. */
export function decodeMarkerPayload(payload: Uint8Array): BitcoinMarkerResult {
  if (!hasMagic(payload)) {
    return { ok: false, error: "not_marker" };
  }
  if (payload.length < MARKER_MAGIC.length + 1) {
    return { ok: false, error: "bad_length" };
  }
  const version = payload[MARKER_MAGIC.length];
  let wantLen: number;
  switch (version) {
    case MARKER_VERSION_1:
      wantLen = PAYLOAD_LEN_V1;
      break;
    case MARKER_VERSION_2:
      wantLen = PAYLOAD_LEN_V2;
      break;
    default:
      return { ok: false, error: "unknown_version" };
  }
  if (payload.length !== wantLen) {
    return { ok: false, error: "bad_length" };
  }
  const address = payload.slice(MARKER_MAGIC.length + 1, PAYLOAD_LEN_V1);
  if (isZeroBytes(address)) {
    return { ok: false, error: "zero_address" };
  }
  if (version === MARKER_VERSION_1) {
    return { ok: true, marker: { version: MARKER_VERSION_1, address } };
  }
  const reference = payload.slice(PAYLOAD_LEN_V1);
  if (isZeroBytes(reference)) {
    return { ok: false, error: "zero_reference" };
  }
  return { ok: true, marker: { version: MARKER_VERSION_2, address, reference } };
}

/**
 * Parses one scriptPubKey. Mirrors marker.go::DecodeScript: only the
 * canonical direct push is accepted, but an OP_PUSHDATA1/2/4 push whose
 * payload begins with the magic is still a candidate (non_canonical_push).
 */
export function decodeMarkerScript(script: Uint8Array): BitcoinMarkerResult {
  if (script.length === 0 || script[0] !== OP_RETURN) {
    return { ok: false, error: "not_marker" };
  }
  const push = decodePush(script.subarray(1));
  if (push === undefined || !hasMagic(push.payload)) {
    return { ok: false, error: "not_marker" };
  }
  if (1 + push.consumed !== script.length) {
    return { ok: false, error: "multiple_pushes" };
  }
  if (!push.canonical) {
    return { ok: false, error: "non_canonical_push" };
  }
  return decodeMarkerPayload(push.payload);
}

/**
 * Applies the attribution rule to every scriptPubKey of one transaction:
 * exactly one marker candidate, and it must be valid. Mirrors
 * marker.go::ScanOutputs. An error means the transaction is not attributed,
 * not that it can be ignored: a funded deposit output with an error here is
 * an unattributed inflow.
 */
export function scanMarkerOutputs(
  scriptPubKeys: readonly Uint8Array[],
): BitcoinMarkerResult {
  const candidates: BitcoinMarkerResult[] = [];
  for (const script of scriptPubKeys) {
    const result = decodeMarkerScript(script);
    if (!result.ok && result.error === "not_marker") {
      continue;
    }
    candidates.push(result);
  }
  if (candidates.length === 0) {
    return { ok: false, error: "no_marker" };
  }
  if (candidates.length > 1) {
    return { ok: false, error: "multiple_markers" };
  }
  return candidates[0] as BitcoinMarkerResult;
}

function hasMagic(payload: Uint8Array): boolean {
  if (payload.length < MARKER_MAGIC.length) {
    return false;
  }
  return MARKER_MAGIC.every((byte, index) => payload[index] === byte);
}

interface DecodedPush {
  payload: Uint8Array;
  // True only for the direct-push form (opcode == payload length).
  canonical: boolean;
  // Bytes consumed, including the opcode and any length prefix.
  consumed: number;
}

function decodePush(s: Uint8Array): DecodedPush | undefined {
  const op = s[0];
  if (op === undefined) {
    return undefined;
  }
  let headerLen: number;
  let length: number;
  if (op >= 0x01 && op <= 0x4b) {
    headerLen = 1;
    length = op;
  } else if (op === OP_PUSHDATA1) {
    if (s.length < 2) {
      return undefined;
    }
    headerLen = 2;
    length = s[1] ?? 0;
  } else if (op === OP_PUSHDATA2) {
    if (s.length < 3) {
      return undefined;
    }
    headerLen = 3;
    length = new DataView(s.buffer, s.byteOffset + 1, 2).getUint16(0, true);
  } else if (op === OP_PUSHDATA4) {
    if (s.length < 5) {
      return undefined;
    }
    headerLen = 5;
    length = new DataView(s.buffer, s.byteOffset + 1, 4).getUint32(0, true);
  } else {
    return undefined;
  }
  if (s.length < headerLen + length) {
    return undefined;
  }
  return {
    payload: s.subarray(headerLen, headerLen + length),
    canonical: headerLen === 1,
    consumed: headerLen + length,
  };
}
