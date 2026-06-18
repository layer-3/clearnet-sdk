import type {
  Account,
  Address,
  Hash,
  PublicClient,
  WalletClient,
} from "viem";

export interface TxRef {
  hash: Hash;
  raw: string;
}

export type DepositStatus = "absent" | "pending" | "confirmed";

export interface SubmitDepositInput {
  asset: string;
  amount: bigint;
  account: string;
  reference?: string;
}

export interface EvmSubmitDepositInput extends SubmitDepositInput {
  asset: Address;
  account: Address;
}

export interface SubmitDepositOptions {
  signal?: AbortSignal;
  receiptTimeoutMs?: number;
  onSubmitted?: (ref: TxRef) => void;
}

export interface VaultDepositor<
  TInput extends SubmitDepositInput = SubmitDepositInput,
> {
  submitDeposit(input: TInput, options?: SubmitDepositOptions): Promise<TxRef>;
  verifyDeposit(
    ref: TxRef,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus>;
}

export interface EvmDepositorConfig {
  publicClient: PublicClient;
  walletClient: WalletClient;
  walletAccount: Account | Address;
  custodyAddress: Address;
  chainId: number;
  receiptTimeoutMs?: number;
}
