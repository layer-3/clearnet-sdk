import type { Payment } from "xrpl";

import type {
  Bytes32Hex,
  DepositDestination,
  SubmitDepositInput,
} from "../../core/types.js";

export type XrplAmount = bigint | string;
export type XrplAsset = string;
export type XrplPreparedPayment = Payment;

export interface XrplDepositDestination extends DepositDestination {
  account: string;
  ref?: Bytes32Hex;
}

export interface XrplNativeDepositInput extends SubmitDepositInput<bigint> {
  asset: "" | "XRP";
  amount: bigint;
  destination: XrplDepositDestination;
}

export interface XrplIssuedDepositInput extends SubmitDepositInput<string> {
  asset: `${string}.${string}` | `${string}:${string}`;
  amount: string;
  destination: XrplDepositDestination;
}

export type XrplSubmitDepositInput =
  | XrplNativeDepositInput
  | XrplIssuedDepositInput;

export interface XrplSignedTransaction {
  txBlob: string;
  hash: string;
}

export interface XrplSigner {
  readonly classicAddress: string;
  sign(payment: XrplPreparedPayment): Promise<XrplSignedTransaction>;
}

export interface XrplDepositorConfig {
  rpcUrl: string;
  vaultAddress: string;
  signer: XrplSigner;
  maxFeeDrops?: bigint | number;
}
