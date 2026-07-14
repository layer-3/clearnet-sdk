import type {
  Account,
  Address,
  PublicClient,
  WalletClient,
} from "viem";

export type Bytes32Hex = `0x${string}`;

export type DepositStatus = "absent" | "pending" | "confirmed";

export interface DepositDestination {
  account: string;
  ref?: Bytes32Hex;
}

export interface EvmDepositDestination extends DepositDestination {
  account: Address;
}

export interface SubmitDepositInput<TAmount = string> {
  asset: string;
  amount: TAmount;
  destination: DepositDestination;
}

export interface EvmSubmitDepositInput extends SubmitDepositInput<string> {
  asset: Address | "";
  destination: EvmDepositDestination;
}

export interface SubmitDepositOptions {
  signal?: AbortSignal;
  receiptTimeoutMs?: number;
  onSubmitted?: (txID: string) => void;
}

export interface VaultDepositor<
  TInput extends SubmitDepositInput<unknown> = SubmitDepositInput,
> {
  submitDeposit(input: TInput, options?: SubmitDepositOptions): Promise<string>;
  verifyDeposit(
    txID: string,
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
  nativeDecimals?: number;
}
