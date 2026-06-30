import { ClearnetSdkError } from "../../core/errors.js";
import type { SubmitDepositOptions } from "../../core/types.js";
import {
  BYTES32_HEX_PATTERN,
  ZERO_BYTES32_PATTERN,
} from "../../core/validation.js";
import {
  BITCOIN_DEFAULT_FALLBACK_FEE_RATE_SAT_PER_VBYTE,
  BITCOIN_DEFAULT_FEE_TARGET_BLOCKS,
  BITCOIN_DEFAULT_MIN_FUNDING_CONFIRMATIONS,
  BITCOIN_DUST_THRESHOLD_SATS,
  BITCOIN_MAX_SIGNED_SATS,
  BITCOIN_NATIVE_ASSET,
} from "./constants.js";
import { normalizeVaultPubkeys, requireCompressedPublicKey } from "./address.js";
import type {
  BitcoinDepositDestination,
  BitcoinDepositorConfig,
  BitcoinNetwork,
  BitcoinSigner,
  NormalizedBitcoinConfig,
} from "./types.js";

export function normalizeConfig(
  config: BitcoinDepositorConfig,
): NormalizedBitcoinConfig {
  const network = requireNetwork(config.network);
  const rpc = requireRpc(config.rpc);
  const signer =
    config.signer === undefined ? undefined : requireConfiguredSigner(config.signer);
  const vaultPubkeys = normalizeVaultPubkeys(config.vaultPubkeys);
  const threshold = requireThreshold(config.threshold, vaultPubkeys.length);
  return {
    network,
    rpc,
    signer,
    vaultPubkeys,
    threshold,
    minFundingConfirmations: Number(
      normalizeNonNegativeIntegerLike(
        config.minFundingConfirmations,
        BITCOIN_DEFAULT_MIN_FUNDING_CONFIRMATIONS,
        "minFundingConfirmations",
      ),
    ),
    feeTargetBlocks: Number(
      normalizePositiveIntegerLike(
        config.feeTargetBlocks,
        BITCOIN_DEFAULT_FEE_TARGET_BLOCKS,
        "feeTargetBlocks",
      ),
    ),
    fallbackFeeRateSatPerVByte: normalizePositiveIntegerLike(
      config.fallbackFeeRateSatPerVByte,
      BITCOIN_DEFAULT_FALLBACK_FEE_RATE_SAT_PER_VBYTE,
      "fallbackFeeRateSatPerVByte",
    ),
    dustThresholdSats: normalizePositiveIntegerLike(
      config.dustThresholdSats,
      BITCOIN_DUST_THRESHOLD_SATS,
      "dustThresholdSats",
    ),
  };
}

export function requireDepositDestination(
  destination: unknown,
): BitcoinDepositDestination {
  if (!destination || typeof destination !== "object") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination is required and must be an object",
    );
  }
  const fields = destination as Partial<BitcoinDepositDestination>;
  if (typeof fields.account !== "string" || fields.account.length === 0) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a non-empty string",
    );
  }
  return destination as BitcoinDepositDestination;
}

export function requireReference(reference: unknown): void {
  if (reference === undefined || reference === "") {
    return;
  }
  if (typeof reference !== "string" || !BYTES32_HEX_PATTERN.test(reference)) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "destination.ref must be a 32-byte hex value",
    );
  }
  if (!ZERO_BYTES32_PATTERN.test(reference)) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "Bitcoin deposits do not support non-zero destination.ref",
    );
  }
}

export function requireBitcoinAmount(amount: unknown): bigint {
  if (typeof amount !== "bigint") {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must be a bigint in satoshis",
    );
  }
  if (amount <= 0n) {
    throw new ClearnetSdkError("INVALID_AMOUNT", "amount must be greater than zero");
  }
  if (amount > BITCOIN_MAX_SIGNED_SATS) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must fit in signed 64-bit satoshis",
    );
  }
  return amount;
}

export function requireBitcoinAsset(asset: unknown): string {
  if (asset === undefined || asset === "") {
    return BITCOIN_NATIVE_ASSET;
  }
  if (typeof asset !== "string") {
    throw new ClearnetSdkError("INVALID_INPUT", "asset must be BTC");
  }
  const normalized = asset.trim().toUpperCase();
  if (normalized !== BITCOIN_NATIVE_ASSET) {
    throw new ClearnetSdkError("INVALID_INPUT", "asset must be BTC");
  }
  return normalized;
}

export function requireSubmitDepositOptions(options: unknown): SubmitDepositOptions {
  if (options === undefined) {
    return {};
  }
  if (options === null || typeof options !== "object") {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "submit options must be an object",
    );
  }
  const candidate = options as Partial<SubmitDepositOptions>;
  if (
    candidate.onSubmitted !== undefined &&
    typeof candidate.onSubmitted !== "function"
  ) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "submit options.onSubmitted must be a function",
    );
  }
  return options;
}

function requireNetwork(network: unknown): BitcoinNetwork {
  if (
    network !== "mainnet" &&
    network !== "testnet" &&
    network !== "signet" &&
    network !== "regtest"
  ) {
    throw new ClearnetSdkError("CHAIN_MISMATCH", "network must be a Bitcoin network");
  }
  return network;
}

function requireRpc(rpc: unknown): NormalizedBitcoinConfig["rpc"] {
  if (!rpc || typeof rpc !== "object") {
    throw new ClearnetSdkError("RPC_ERROR", "Bitcoin rpc is required");
  }
  const candidate = rpc as Record<string, unknown>;
  for (const method of [
    "listUnspent",
    "estimateSmartFeeSatPerVByte",
    "sendRawTransaction",
    "getRawTransaction",
  ]) {
    if (typeof candidate[method] !== "function") {
      throw new ClearnetSdkError("RPC_ERROR", `Bitcoin rpc.${method} is required`);
    }
  }
  return rpc as NormalizedBitcoinConfig["rpc"];
}

export function requireConfiguredSigner(signer: unknown): BitcoinSigner {
  if (!signer || typeof signer !== "object") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "Bitcoin signer is required",
    );
  }
  const candidate = signer as Partial<BitcoinSigner>;
  if (candidate.algorithm !== "secp256k1") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "Bitcoin signer.algorithm must be secp256k1",
    );
  }
  if (typeof candidate.getPublicKeyCompressed !== "function") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "Bitcoin signer.getPublicKeyCompressed is required",
    );
  }
  if (typeof candidate.signDigest32 !== "function") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "Bitcoin signer.signDigest32 is required",
    );
  }
  return candidate as BitcoinSigner;
}

function requireThreshold(threshold: unknown, keyCount: number): number {
  if (!Number.isSafeInteger(threshold)) {
    throw new ClearnetSdkError("INVALID_INPUT", "threshold must be an integer");
  }
  const value = Number(threshold);
  if (value < 1 || value > keyCount) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "threshold must be between 1 and vaultPubkeys length",
    );
  }
  return value;
}

function normalizePositiveIntegerLike(
  value: bigint | number | undefined,
  fallback: bigint,
  field: string,
): bigint {
  const normalized = normalizeNonNegativeIntegerLike(value, fallback, field);
  if (normalized === 0n) {
    throw new ClearnetSdkError("INVALID_INPUT", `${field} must be a positive safe integer`);
  }
  return normalized;
}

function normalizeNonNegativeIntegerLike(
  value: bigint | number | undefined,
  fallback: bigint,
  field: string,
): bigint {
  if (value === undefined) {
    return fallback;
  }
  if (typeof value === "bigint") {
    if (value < 0n || value > BigInt(Number.MAX_SAFE_INTEGER)) {
      throw new ClearnetSdkError("INVALID_INPUT", `${field} must be a non-negative safe integer`);
    }
    return value;
  }
  if (!Number.isSafeInteger(value) || value < 0) {
    throw new ClearnetSdkError("INVALID_INPUT", `${field} must be a non-negative safe integer`);
  }
  return BigInt(value);
}

export async function signerPublicKey(signer: BitcoinSigner): Promise<Uint8Array> {
  return requireCompressedPublicKey(
    await signer.getPublicKeyCompressed(),
    "signer.publicKey",
  );
}
