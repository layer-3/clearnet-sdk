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
  // nonceKey selects the upper 192 bits of the 2D nonce; defaults to 0. Each
  // key has its own consecutive sequence, so one depositor address can keep
  // several independent streams. A contract depositing for many users should
  // use one key per user and take the sequence from the user's signed input,
  // never from chain state.
  nonceKey?: bigint;
}

// SubmitDepositResult carries both identifiers a submitted deposit produces.
// txHash is the on-chain transaction/signature hash. depositId is the ID a
// MintReceipt is signed for, as each chain's deposit ID helper builds it; on no
// chain is it the plain txHash.
export interface SubmitDepositResult {
  txHash: string;
  depositId: string;
}

export interface SubmitDepositOptions {
  signal?: AbortSignal;
  receiptTimeoutMs?: number;
  onSubmitted?: (result: SubmitDepositResult) => void;
}

export interface VaultDepositor<
  TInput extends SubmitDepositInput<unknown> = SubmitDepositInput,
> {
  submitDeposit(
    input: TInput,
    options?: SubmitDepositOptions,
  ): Promise<SubmitDepositResult>;
  // chainDepositStatus reports whether the deposit identified by txHash and
  // depositId (as returned by submitDeposit) is present and final on chain.
  // It reports the deposit transaction's on-chain status only and does not
  // apply custody's crediting rules (for example the BTC dust floor and
  // self-deposit guard, or XRPL partial payments and asset support), so
  // "confirmed" does not guarantee the deposit will be credited.
  chainDepositStatus(
    txHash: string,
    depositId: string,
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
