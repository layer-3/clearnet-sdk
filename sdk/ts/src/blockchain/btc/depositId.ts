import { ClearnetSdkError } from "../../core/errors.js";
import { parseDepositIdIndex } from "../../core/depositIdIndex.js";
import { requireTxidHex } from "./txid.js";

const UINT32_MAX = 0xffffffff;

/**
 * Computes the Bitcoin deposit ID for output vout of transaction txid:
 * "<txid>:<vout>". txid is expected as lowercase hex, as Bitcoin Core reports
 * it; its case is kept as given. MintReceipts are signed over it, so it must
 * stay byte-identical to the Go implementation; shared vectors in
 * testdata/deposit_id_vectors.json at the repository root.
 */
export function bitcoinDepositId(txid: string, vout: number): string {
  const normalized = requireTxidHex(txid);
  if (!Number.isSafeInteger(vout) || vout < 0 || vout > UINT32_MAX) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "Bitcoin output index must be an integer in [0, 2^32)",
    );
  }
  return `${normalized}:${vout}`;
}

/**
 * Returns the output index of depositId, which must be exactly
 * bitcoinDepositId(txid, vout) for some vout. Throws INVALID_DEPOSIT_ID
 * otherwise.
 */
export function parseBitcoinDepositId(txid: string, depositId: unknown): number {
  const normalized = requireTxidHex(txid);
  if (typeof depositId !== "string" || !depositId.startsWith(`${normalized}:`)) {
    throw new ClearnetSdkError(
      "INVALID_DEPOSIT_ID",
      "Bitcoin deposit ID must be <txid>:<vout> for the given txid",
    );
  }
  const vout = parseDepositIdIndex(depositId, normalized, BigInt(UINT32_MAX));
  if (vout === undefined) {
    throw new ClearnetSdkError(
      "INVALID_DEPOSIT_ID",
      "Bitcoin deposit ID has an invalid output index",
    );
  }
  return Number(vout);
}
