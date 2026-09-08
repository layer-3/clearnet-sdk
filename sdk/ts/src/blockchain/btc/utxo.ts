import { ClearnetSdkError } from "../../core/errors.js";
import { compareBytes, hexToBytes, reverseBytes } from "../../core/bytes.js";
import type { BitcoinUnspent, BitcoinWalletAddressType } from "./types.js";

export interface SelectedUtxos {
  utxos: readonly BitcoinUnspent[];
  feeSats: bigint;
}

export function estimateDepositFeeSats(
  inputCount: number,
  feeRateSatPerVByte: bigint,
  addressType: BitcoinWalletAddressType = "p2wpkh",
  hasReference = false,
): bigint {
  return (
    DEPOSIT_BASE_VBYTES +
    markerOutputVbytes(hasReference) +
    inputVbytes(addressType) * BigInt(inputCount)
  ) * feeRateSatPerVByte;
}

export function selectDepositUtxos(
  available: readonly BitcoinUnspent[],
  amountSats: bigint,
  feeRateSatPerVByte: bigint,
  addressType: BitcoinWalletAddressType = "p2wpkh",
  hasReference = false,
): SelectedUtxos {
  const ordered = [...available].sort(compareUtxoForSelection);
  const selected: BitcoinUnspent[] = [];
  let total = 0n;
  for (const utxo of ordered) {
    selected.push(utxo);
    total += utxo.amountSats;
    const fee = estimateDepositFeeSats(
      selected.length,
      feeRateSatPerVByte,
      addressType,
      hasReference,
    );
    if (total >= amountSats + fee) {
      return { utxos: selected, feeSats: fee };
    }
  }
  throw new ClearnetSdkError(
    "INSUFFICIENT_FUNDS",
    "insufficient eligible BTC balance",
  );
}

// Conservative final transaction sizing for one P2WSH deposit output plus one
// P2WPKH change output. The constants intentionally cover finalized
// max-length DER signatures rather than the smaller unsigned PSBT shape.
const DEPOSIT_BASE_VBYTES = 85n;
const P2WPKH_INPUT_VBYTES = 68n;
const P2SH_P2WPKH_INPUT_VBYTES = 92n;

// The ADR-023 zero-value OP_RETURN marker output every deposit now carries.
// An output's vbyte cost is its serialized byte length (8-byte value +
// a 1-byte varint script length + the script, since every marker script stays
// well under the 253-byte varint boundary): 8 + 1 + 27 = 36 for a 0x01
// marker (no reference), 8 + 1 + 59 = 68 for a 0x02 marker (with reference).
// Under-estimating here produces a below-relay-rate transaction, not a wrong marker.
const MARKER_OUTPUT_V1_VBYTES = 36n;
const MARKER_OUTPUT_V2_VBYTES = 68n;

function markerOutputVbytes(hasReference: boolean): bigint {
  return hasReference ? MARKER_OUTPUT_V2_VBYTES : MARKER_OUTPUT_V1_VBYTES;
}

function inputVbytes(addressType: BitcoinWalletAddressType): bigint {
  return addressType === "p2sh" ? P2SH_P2WPKH_INPUT_VBYTES : P2WPKH_INPUT_VBYTES;
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
    throw new ClearnetSdkError("INVALID_TX_ID", "Bitcoin txid must be 64 hex characters");
  }
  return reverseBytes(hexToBytes(txid, "txid"));
}
