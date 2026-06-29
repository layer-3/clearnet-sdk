import { ClearnetSdkError } from "../../core/errors.js";
import { compareBytes, hexToBytes, reverseBytes } from "./bytes.js";
import type { BitcoinUnspent } from "./types.js";

export interface SelectedUtxos {
  utxos: readonly BitcoinUnspent[];
  feeSats: bigint;
}

export function estimateDepositFeeSats(
  inputCount: number,
  feeRateSatPerVByte: bigint,
): bigint {
  return (97n + 120n * BigInt(inputCount)) * feeRateSatPerVByte;
}

export function selectDepositUtxos(
  available: readonly BitcoinUnspent[],
  amountSats: bigint,
  feeRateSatPerVByte: bigint,
): SelectedUtxos {
  const ordered = [...available].sort(compareUtxoForSelection);
  const selected: BitcoinUnspent[] = [];
  let total = 0n;
  for (const utxo of ordered) {
    selected.push(utxo);
    total += utxo.amountSats;
    const fee = estimateDepositFeeSats(selected.length, feeRateSatPerVByte);
    if (total >= amountSats + fee) {
      return { utxos: selected, feeSats: fee };
    }
  }
  throw new ClearnetSdkError(
    "INSUFFICIENT_FUNDS",
    "insufficient eligible BTC balance",
  );
}

export function compareUtxoForSelection(a: BitcoinUnspent, b: BitcoinUnspent): number {
  if (a.amountSats !== b.amountSats) {
    return a.amountSats > b.amountSats ? -1 : 1;
  }
  const hashCompare = compareInternalTxidBytes(a.txid, b.txid);
  if (hashCompare !== 0) {
    return hashCompare;
  }
  return a.vout - b.vout;
}

export function compareUtxoForInputOrder(a: BitcoinUnspent, b: BitcoinUnspent): number {
  const hashCompare = compareInternalTxidBytes(a.txid, b.txid);
  if (hashCompare !== 0) {
    return hashCompare;
  }
  return a.vout - b.vout;
}

function compareInternalTxidBytes(a: string, b: string): number {
  return compareBytes(internalTxidBytes(a), internalTxidBytes(b));
}

function internalTxidBytes(txid: string): Uint8Array {
  if (!/^[a-fA-F0-9]{64}$/.test(txid)) {
    throw new ClearnetSdkError("INVALID_TX_REF", "Bitcoin txid must be 64 hex characters");
  }
  return reverseBytes(hexToBytes(txid, "txid"));
}
