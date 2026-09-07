import type {
  DepositDestination,
  SubmitDepositInput,
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

/**
 * One output of the exact set prepareDepositPsbt built, as prepared - in
 * order, script bytes and amount both required. submitSignedDepositPsbt
 * compares a wallet-signed PSBT's finalized outputs against this set and
 * rejects any mismatch, which is what detects a stripped or altered
 * deposit-attribution marker (ADR-023 §3) before broadcast. Wallet
 * finalization can change scriptSig/witness data and even the txid
 * (nested-SegWit in particular), but must never change the output set itself.
 */
export interface BitcoinExpectedDepositOutput {
  /** Output scriptPubKey, as lowercase hex. */
  script: string;
  /** Output value in satoshis. */
  amount: bigint;
}

export interface BitcoinPreparedDepositPsbt {
  psbtHex: string;
  inputIndexesToSign: readonly number[];
  /**
   * Transaction ID for the unsigned PSBT transaction shape. Wallet finalization
   * can change the final txid, especially for nested-SegWit inputs, so callers
   * must use the txID returned by submitSignedDepositPsbt for verification.
   */
  unsignedTxID: string;
  fundingAddress: string;
  depositAddress: string;
  feeSats: bigint;
  selectedUtxos: readonly BitcoinUnspent[];
  /* The exact output set (order, script, amount) this PSBT was built with. */
  expectedOutputs: readonly BitcoinExpectedDepositOutput[];
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
