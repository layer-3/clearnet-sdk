import type { Transaction } from "@solana/web3.js";

import type {
  Bytes32Hex,
  DepositDestination,
  SubmitDepositInput,
} from "../../core/types.js";

export type SolanaAsset = string;

export type SolanaCommitment = "processed" | "confirmed" | "finalized";

export interface SolanaDepositDestination extends DepositDestination {
  account: string;
  ref?: Bytes32Hex;
}

export interface SolanaSubmitDepositInput extends SubmitDepositInput {
  asset: SolanaAsset;
  amount: bigint;
  destination: SolanaDepositDestination;
}

export interface SolanaSigner {
  publicKey: string;
  signAndSend(transaction: Transaction): Promise<string>;
}

export interface SolanaDepositorConfig {
  rpcUrl: string;
  signer: SolanaSigner;
  programId?: string;
  commitment?: SolanaCommitment;
  receiptTimeoutMs?: number;
}
