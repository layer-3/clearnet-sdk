import { encodeAbiParameters, keccak256 } from "viem";
import type { Address, Hex } from "viem";

import { ClearnetSdkError } from "../../core/errors.js";

// 2D nonce: upper 192 bits are a caller-chosen key, lower 64 a consecutive
// per-(depositor, key) sequence (ERC-4337 EntryPoint layout).
const NONCE_SEQUENCE_BITS = 64n;
const UINT64_MASK = (1n << NONCE_SEQUENCE_BITS) - 1n;
const MAX_NONCE_KEY = (1n << 192n) - 1n;

const DEPOSIT_ID_PATTERN = /^0x[0-9a-f]{64}$/;

/**
 * Computes the EVM deposit ID (ISS-068; ADR-018 §Transaction IDs): "0x" +
 * lowercase hex of keccak256(abi.encode(chainid, vault, depositor, nonce)).
 * MintReceipts are signed over it, so it must stay byte-identical to the Go
 * implementation; shared vectors in testdata/deposit_id_vectors.json at the
 * repository root.
 */
export function depositId(
  chainId: bigint,
  vault: Address,
  depositor: Address,
  nonce: bigint,
): Hex {
  const encoded = encodeAbiParameters(
    [
      { type: "uint256" },
      { type: "address" },
      { type: "address" },
      { type: "uint256" },
    ],
    [chainId, vault, depositor, nonce],
  );
  return keccak256(encoded);
}

/**
 * Validates that value has exactly the depositId shape ("0x" + 64 lowercase
 * hex characters). Rejects a "txHash/logIndex" shape. It cannot tell a deposit
 * ID from a transaction hash (same shape), so callers keep them in separate
 * fields (txHash, depositId).
 */
export function requireDepositId(value: unknown): Hex {
  if (typeof value !== "string" || !DEPOSIT_ID_PATTERN.test(value)) {
    throw new ClearnetSdkError(
      "INVALID_DEPOSIT_ID",
      'deposit ID must be "0x" followed by 64 lowercase hex characters',
    );
  }
  return value as Hex;
}

/**
 * Returns key, or 0n when it is undefined. Throws INVALID_INPUT unless key is a
 * bigint in [0, 2^192).
 */
export function requireNonceKey(key: unknown): bigint {
  if (key === undefined) {
    return 0n;
  }
  if (typeof key !== "bigint" || key < 0n || key > MAX_NONCE_KEY) {
    throw new ClearnetSdkError("INVALID_INPUT", "nonce key must be a bigint in [0, 2^192)");
  }
  return key;
}

/**
 * Packs key and sequence into the single uint256 nonce Custody.deposit
 * expects: key << 64 | sequence. Throws INVALID_INPUT unless
 * 0 <= key < 2^192 and 0 <= sequence < 2^64, so neither field can spill into
 * the other.
 */
export function composeNonce(key: bigint, sequence: bigint): bigint {
  if (key < 0n || key > MAX_NONCE_KEY) {
    throw new ClearnetSdkError("INVALID_INPUT", "nonce key must be in [0, 2^192)");
  }
  if (sequence < 0n || sequence > UINT64_MASK) {
    throw new ClearnetSdkError("INVALID_INPUT", "nonce sequence must be in [0, 2^64)");
  }
  return (key << NONCE_SEQUENCE_BITS) | sequence;
}

/** Reverses composeNonce: key is the upper 192 bits, sequence the lower 64. */
export function splitNonce(nonce: bigint): { key: bigint; sequence: bigint } {
  return { key: nonce >> NONCE_SEQUENCE_BITS, sequence: nonce & UINT64_MASK };
}
