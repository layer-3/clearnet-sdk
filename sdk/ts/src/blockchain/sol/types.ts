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

export interface SolanaSubmitDepositInput extends SubmitDepositInput<string> {
  asset: SolanaAsset;
  amount: string;
  destination: SolanaDepositDestination;
}

export interface SolanaSigner {
  publicKey: string;
  /**
   * Signs and submits a @solana/web3.js v1 Transaction.
   *
   * Implementations must set transaction.recentBlockhash, usually from
   * getLatestBlockhash, before signing.
   */
  signAndSend(transaction: Transaction): Promise<string>;
}

export interface SolanaDepositorConfig {
  rpcUrl: string;
  signer: SolanaSigner;
  programId?: string;
  commitment?: SolanaCommitment;
  receiptTimeoutMs?: number;
}
