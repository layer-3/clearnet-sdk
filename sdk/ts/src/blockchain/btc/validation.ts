import { ClearnetSdkError } from "../../core/errors.js";
import { hexToBytes } from "../../core/bytes.js";
import type { SubmitDepositOptions } from "../../core/types.js";
import { BYTES32_HEX_PATTERN } from "../../core/validation.js";
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
  BitcoinExpectedDepositOutput,
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

/**
 * Parses destination.account into the raw 20-byte address ADR-023 carries in
 * the deposit marker, matching the other three chains. Accepts a bare hex
 * address or a yellow://.../user/<hex> URI's last segment, tolerating
 * surrounding whitespace, so it accepts exactly what the Go SDK's BTC
 * parseClearnetAccount does.
 */
export function requireClearnetAccount(account: unknown): Uint8Array {
  if (typeof account !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  const trimmed = account.trim();
  const segment = trimmed.slice(trimmed.lastIndexOf("/") + 1);
  const hex = segment.toLowerCase().replace(/^0x/, "");
  if (!/^[a-f0-9]+$/.test(hex) || hex.length !== 40) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  return hexToBytes(hex, "destination.account");
}

/**
 * Parses destination.ref into 32 bytes, zero-filled when omitted.The caller
 * selects the marker version from whether the returned bytes are all-zero.
 */
export function requireReference(reference: unknown): Uint8Array {
  if (reference === undefined || reference === "") {
    return new Uint8Array(32);
  }
  if (typeof reference !== "string" || !BYTES32_HEX_PATTERN.test(reference)) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "destination.ref must be a 32-byte hex value",
    );
  }
  return hexToBytes(reference.slice(2), "destination.ref");
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
  throw new ClearnetSdkError("INVALID_INPUT", 'asset must be "" for native BTC');
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

/**
 * Validates the expectedOutputs a caller passes to submitSignedDepositPsbt.
 * Required, not optional: prepareDepositPsbt's expectedOutputs must round-trip
 * back unmodified, and there is no default that would let a caller skip the
 * check by omission - an absent, empty, or malformed array is rejected here
 * rather than silently disabling output verification.
 */
export function requireExpectedDepositOutputs(
  outputs: unknown,
): readonly BitcoinExpectedDepositOutput[] {
  if (!Array.isArray(outputs) || outputs.length === 0) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "expectedOutputs is required and must be the non-empty array returned by prepareDepositPsbt",
    );
  }
  return outputs.map((output, index) => {
    if (!output || typeof output !== "object") {
      throw new ClearnetSdkError(
        "INVALID_INPUT",
        `expectedOutputs[${index}] must be an object`,
      );
    }
    const candidate = output as Partial<BitcoinExpectedDepositOutput>;
    if (
      typeof candidate.script !== "string" ||
      !/^[a-f0-9]+$/i.test(candidate.script) ||
      candidate.script.length % 2 !== 0
    ) {
      throw new ClearnetSdkError(
        "INVALID_INPUT",
        `expectedOutputs[${index}].script must be an even-length hex string`,
      );
    }
    if (typeof candidate.amount !== "bigint" || candidate.amount < 0n) {
      throw new ClearnetSdkError(
        "INVALID_INPUT",
        `expectedOutputs[${index}].amount must be a non-negative bigint`,
      );
    }
    return { script: candidate.script.toLowerCase(), amount: candidate.amount };
  });
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
