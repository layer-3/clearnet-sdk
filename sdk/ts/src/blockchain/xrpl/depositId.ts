import { requireTxID } from "./validation.js";

/**
 * Computes the XRPL deposit ID: the transaction hash lower-cased. MintReceipts
 * are signed over it, so it must stay byte-identical to the Go implementation;
 * shared vectors in testdata/deposit_id_vectors.json at the repository root.
 */
export function xrplDepositId(txHash: string): string {
  return requireTxID(txHash).toLowerCase();
}
