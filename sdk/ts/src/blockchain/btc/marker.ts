import { ClearnetSdkError } from "../../core/errors.js";
import { concatBytes } from "../../core/bytes.js";

/**
 * ADR-023 deposit-attribution marker: a zero-value OP_RETURN output carried
 * alongside every generic-address BTC deposit (ADR §2, §4). TS has no shared
 * code with the Go `marker` package, so this module is a direct, byte-for-byte
 * mirror of it, and is asserted against the same golden vectors
 * (pkg/blockchain/btc/marker/testdata/vectors.json) in test/blockchain/btc/marker.test.ts.
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
