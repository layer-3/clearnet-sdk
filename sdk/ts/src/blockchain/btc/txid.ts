import { ClearnetSdkError } from "../../core/errors.js";

const TXID_PATTERN = /^[a-fA-F0-9]{64}$/;

export function txIDFromTxid(txid: string): string {
  return normalizeTxid(txid);
}

export function requireBitcoinTxID(txID: unknown): string {
  return normalizeTxid(txID);
}

export function normalizeTxid(txid: unknown): string {
  if (typeof txid !== "string" || !TXID_PATTERN.test(txid)) {
    throw new ClearnetSdkError("INVALID_TX_ID", "Bitcoin txid must be 64 hex characters");
  }
  return txid.toLowerCase();
}
