import { sha256 } from "@noble/hashes/sha2.js";

import { bytesToHex } from "../../core/bytes.js";
import { ClearnetSdkError } from "../../core/errors.js";
import { parseDepositIdIndex } from "../../core/depositIdIndex.js";
import { requireTxID } from "./validation.js";

const UINT64_MAX = (1n << 64n) - 1n;

/**
 * Computes the Solana deposit ID: "0x" + lowercase hex of
 * sha256(the 64 signature bytes), ":", and index in decimal. index is the
 * Deposited event's zero-based position among the transaction's custody event
 * CPIs (Deposited and Executed); it is 0 for a transaction submitDeposit
 * builds. MintReceipts are signed over it, so it must stay byte-identical to
 * the Go implementation; shared vectors in testdata/deposit_id_vectors.json at
 * the repository root.
 */
export function solanaDepositId(signature: string, index: bigint | number): string {
  const prefix = depositIdPrefix(requireTxID(signature));
  return `${prefix}:${requireEventIndex(index).toString()}`;
}

/**
 * Returns the event index of depositId, which must be exactly
 * solanaDepositId(signature, index) for some index. Throws INVALID_DEPOSIT_ID
 * otherwise.
 */
export function parseSolanaDepositId(signature: string, depositId: unknown): bigint {
  const prefix = depositIdPrefix(requireTxID(signature));
  if (typeof depositId !== "string" || !depositId.includes(":")) {
    throw new ClearnetSdkError(
      "INVALID_DEPOSIT_ID",
      "Solana deposit ID must be 0x<signature hash>:<index>",
    );
  }
  const index = parseDepositIdIndex(depositId, prefix, UINT64_MAX);
  if (index === undefined) {
    throw new ClearnetSdkError(
      "INVALID_DEPOSIT_ID",
      "Solana deposit ID does not match the signature",
    );
  }
  return index;
}

function depositIdPrefix(signature: Uint8Array): string {
  return `0x${bytesToHex(sha256(signature))}`;
}

function requireEventIndex(index: bigint | number): bigint {
  const value =
    typeof index === "bigint"
      ? index
      : typeof index === "number" && Number.isSafeInteger(index)
        ? BigInt(index)
        : -1n;
  if (value < 0n || value > UINT64_MAX) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "Solana event index must be an integer in [0, 2^64)",
    );
  }
  return value;
}
