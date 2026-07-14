import type {
  DepositDestination,
  SubmitDepositInput,
  TxRef,
} from "../../core/types.js";

export type BitcoinNetwork = "mainnet" | "testnet" | "signet" | "regtest";

export type BitcoinAsset = string;

export interface BitcoinDepositDestination extends DepositDestination {
  account: string;
}

export interface BitcoinSubmitDepositInput extends SubmitDepositInput<string> {
  asset: BitcoinAsset;
  destination: BitcoinDepositDestination;
}

export interface BitcoinSigner {
  readonly algorithm: "secp256k1";
  getPublicKeyCompressed(): Uint8Array | Promise<Uint8Array>;
  signDigest32(digest: Uint8Array): Uint8Array | Promise<Uint8Array>;
}

export interface BitcoinRpc {
  listUnspent(
    minConfirmations: number,
    addresses: readonly string[],
  ): Promise<readonly BitcoinUnspent[]>;
  estimateSmartFeeSatPerVByte(
    confirmationTarget: number,
    fallbackRate: bigint,
  ): Promise<bigint>;
  sendRawTransaction(hexTx: string): Promise<string>;
  getRawTransaction(txid: string): Promise<BitcoinRawTransaction | null>;
}

export interface BitcoinUnspent {
  txid: string;
  vout: number;
  amountSats: bigint;
  confirmations: number;
  scriptPubKey: string;
}

export interface BitcoinRawTransaction {
  txid: string;
  confirmations: number;
}

export interface BitcoinDepositorConfig {
  network: BitcoinNetwork;
  rpc: BitcoinRpc;
  signer?: BitcoinSigner | undefined;
  vaultPubkeys: readonly (Uint8Array | string)[];
  threshold: number;
  minFundingConfirmations?: bigint | number;
  feeTargetBlocks?: bigint | number;
  fallbackFeeRateSatPerVByte?: bigint | number;
  dustThresholdSats?: bigint | number;
}

export interface BitcoinCoreRpcClientConfig {
  url: string;
  username?: string;
  password?: string;
  wallet?: string;
  fetch?: typeof fetch;
}

export class BitcoinRpcError extends Error {
  readonly code: number;

  constructor(code: number, message: string) {
    super(message);
    this.name = "BitcoinRpcError";
    this.code = code;
  }
}

export interface BitcoinPsbtSignerInfo {
  publicKey: Uint8Array | string;
  address?: string;
  addressType?: BitcoinWalletAddressType;
}

export type BitcoinWalletAddressType = "p2wpkh" | "p2sh";

export interface BitcoinPreparedDepositPsbt {
  psbtHex: string;
  inputIndexesToSign: readonly number[];
  /**
   * TxRef for the unsigned PSBT transaction shape. Wallet finalization can
   * change the final txid, especially for nested-SegWit inputs, so callers must
   * use the TxRef returned by submitSignedDepositPsbt for verification.
   */
  unsignedRef: TxRef;
  fundingAddress: string;
  depositAddress: string;
  feeSats: bigint;
  selectedUtxos: readonly BitcoinUnspent[];
}

export interface NormalizedBitcoinConfig {
  network: BitcoinNetwork;
  rpc: BitcoinRpc;
  signer: BitcoinSigner | undefined;
  vaultPubkeys: readonly Uint8Array[];
  threshold: number;
  minFundingConfirmations: number;
  feeTargetBlocks: number;
  fallbackFeeRateSatPerVByte: bigint;
  dustThresholdSats: bigint;
}
