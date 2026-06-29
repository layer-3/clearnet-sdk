import { ClearnetSdkError } from "../../core/errors.js";
import type { Bytes32Hex, TxRef } from "../../core/types.js";
import { bytesToHex, hexToBytes, reverseBytes } from "./bytes.js";

const TXID_PATTERN = /^[a-fA-F0-9]{64}$/;
const BYTES32_HEX_PATTERN = /^0x[a-fA-F0-9]{64}$/;

export function txRefFromTxid(txid: string): TxRef {
  const normalized = normalizeTxid(txid);
  const bytes = hexToBytes(normalized, "txid");
  return {
    raw: normalized,
    hash: `0x${bytesToHex(reverseBytes(bytes))}` as Bytes32Hex,
  };
}

export function requireBitcoinTxRef(ref: unknown): TxRef {
  if (!ref || typeof ref !== "object") {
    throw new ClearnetSdkError("INVALID_TX_REF", "ref.raw must be a Bitcoin txid");
  }
  const fields = ref as Partial<TxRef>;
  const raw = normalizeTxid(fields.raw);
  if (typeof fields.hash !== "string" || !BYTES32_HEX_PATTERN.test(fields.hash)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must be a 32-byte hex value",
    );
  }
  const expected = txRefFromTxid(raw);
  if (fields.hash.toLowerCase() !== expected.hash) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must be the byte-reversal of ref.raw",
    );
  }
  return expected;
}

export function normalizeTxid(txid: unknown): string {
  if (typeof txid !== "string" || !TXID_PATTERN.test(txid)) {
    throw new ClearnetSdkError("INVALID_TX_REF", "Bitcoin txid must be 64 hex characters");
  }
  return txid.toLowerCase();
}
