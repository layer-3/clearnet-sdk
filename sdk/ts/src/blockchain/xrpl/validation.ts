import { Buffer } from "buffer";

import { isValidClassicAddress } from "xrpl";

import { ClearnetSdkError } from "../../core/errors.js";
import type { Bytes32Hex, TxRef } from "../../core/types.js";
import { UINT64_MAX, XRPL_NATIVE_ASSET } from "./constants.js";
import type { XrplDepositDestination, XrplSigner } from "./types.js";

const BYTES32_HEX_PATTERN = /^0x[a-fA-F0-9]{64}$/;
const HASH_PATTERN = /^[a-fA-F0-9]{64}$/;
const DECIMAL_PATTERN = /^(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$/;
const STANDARD_CURRENCY_PATTERN = /^[A-Za-z0-9]{3}$/;
const HEX_CURRENCY_PATTERN = /^[a-fA-F0-9]{40}$/;

export type ResolvedXrplAmount =
  | { kind: "native"; amount: string }
  | {
      kind: "issued";
      amount: { currency: string; issuer: string; value: string };
    };

export function requireRpcUrl(rpcUrl: unknown): string {
  if (typeof rpcUrl !== "string" || rpcUrl.trim() === "") {
    throw new ClearnetSdkError("RPC_ERROR", "rpcUrl is required");
  }
  let url: URL;
  try {
    url = new URL(rpcUrl);
  } catch (error) {
    throw new ClearnetSdkError(
      "RPC_ERROR",
      "rpcUrl must be a valid XRPL WebSocket URL",
      { cause: error },
    );
  }
  if (url.protocol !== "ws:" && url.protocol !== "wss:") {
    throw new ClearnetSdkError(
      "RPC_ERROR",
      "rpcUrl must use ws: or wss: for xrpl.js",
    );
  }
  return rpcUrl;
}

export function requireClassicAddress(value: unknown, field: string): string {
  if (typeof value !== "string" || !isValidClassicAddress(value)) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `${field} must be a valid XRPL classic address`,
    );
  }
  return value;
}

export function requireSigner(signer: unknown): XrplSigner {
  if (!signer || typeof signer !== "object") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "XRPL signer is required",
    );
  }
  const candidate = signer as Partial<XrplSigner>;
  requireClassicAddress(candidate.classicAddress, "signer.classicAddress");
  if (typeof candidate.sign !== "function") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "XRPL signer.sign is required",
    );
  }
  return candidate as XrplSigner;
}

export function requireDepositDestination(
  destination: unknown,
): XrplDepositDestination {
  if (!destination || typeof destination !== "object") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  return destination as XrplDepositDestination;
}

export function requireClearnetAccount(account: unknown): Uint8Array {
  if (typeof account !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  const trimmed = account.trim();
  const hex = trimmed.toLowerCase().replace(/^0x/, "");
  if (!/^[a-f0-9]+$/.test(hex) || hex.length !== 40) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  return Uint8Array.from(Buffer.from(hex, "hex"));
}

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
  return Uint8Array.from(Buffer.from(reference.slice(2), "hex"));
}

export function resolveAmount(
  asset: unknown,
  amount: unknown,
): ResolvedXrplAmount {
  if (isNativeAsset(asset)) {
    return { kind: "native", amount: requireNativeAmount(amount).toString() };
  }
  const issued = requireIssuedAsset(asset);
  const value = requireIssuedAmount(amount);
  return {
    kind: "issued",
    amount: { currency: issued.currency, issuer: issued.issuer, value },
  };
}

export function normalizeFeeDrops(value: bigint | number): bigint {
  if (typeof value === "bigint") {
    if (value <= 0n || value > UINT64_MAX) {
      throw new ClearnetSdkError(
        "INVALID_AMOUNT",
        "maxFeeDrops must be a positive uint64 value",
      );
    }
    return value;
  }
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "maxFeeDrops must be a positive safe integer",
    );
  }
  return BigInt(value);
}

export function normalizeTxHash(hash: string): TxRef {
  if (!HASH_PATTERN.test(hash)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "XRPL transaction hash must be 64 hex characters",
    );
  }
  const raw = hash.toUpperCase();
  return { hash: `0x${raw.toLowerCase()}` as Bytes32Hex, raw };
}

export function requireTxRef(ref: unknown): TxRef {
  if (!ref || typeof ref !== "object") {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.raw must be an XRPL transaction hash",
    );
  }
  const fields = ref as Partial<TxRef>;
  if (typeof fields.raw !== "string" || !HASH_PATTERN.test(fields.raw)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.raw must be an XRPL transaction hash",
    );
  }
  if (typeof fields.hash !== "string" || !BYTES32_HEX_PATTERN.test(fields.hash)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must be a 32-byte hex value",
    );
  }
  const normalized = normalizeTxHash(fields.raw);
  if (normalized.hash.toLowerCase() !== fields.hash.toLowerCase()) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must match ref.raw",
    );
  }
  return normalized;
}

export function normalizeMinConfirmations(value: bigint | number): bigint {
  if (typeof value === "bigint") {
    if (value < 0n) {
      throw new ClearnetSdkError(
        "INVALID_CONFIRMATIONS",
        "minConfirmations must be non-negative",
      );
    }
    return value;
  }
  if (!Number.isSafeInteger(value) || value < 0) {
    throw new ClearnetSdkError(
      "INVALID_CONFIRMATIONS",
      "minConfirmations must be a non-negative safe integer",
    );
  }
  return BigInt(value);
}

function isNativeAsset(asset: unknown): boolean {
  if (asset === "" || asset === undefined) {
    return true;
  }
  return typeof asset === "string" && asset.trim().toUpperCase() === XRPL_NATIVE_ASSET;
}

function requireNativeAmount(amount: unknown): bigint {
  if (typeof amount !== "bigint") {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "native XRP amount must be a bigint in drops",
    );
  }
  if (amount <= 0n || amount > UINT64_MAX) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "native XRP amount must be a positive uint64 drops value",
    );
  }
  return amount;
}

function requireIssuedAsset(asset: unknown): { currency: string; issuer: string } {
  if (typeof asset !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "issued XRPL asset must be CUR.rIssuer or CUR:rIssuer",
    );
  }
  const trimmed = asset.trim();
  const dot = trimmed.indexOf(".");
  const colon = trimmed.indexOf(":");
  const separator =
    dot > 0 && (colon < 0 || dot < colon) ? dot : colon > 0 ? colon : -1;
  if (separator <= 0 || separator === trimmed.length - 1) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "issued XRPL asset must be CUR.rIssuer or CUR:rIssuer",
    );
  }
  const currency = trimmed.slice(0, separator);
  const issuer = trimmed.slice(separator + 1);
  if (
    currency.toUpperCase() === XRPL_NATIVE_ASSET ||
    (!STANDARD_CURRENCY_PATTERN.test(currency) &&
      !HEX_CURRENCY_PATTERN.test(currency))
  ) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "issued XRPL currency must be a 3-character code or 20-byte hex code",
    );
  }
  return { currency, issuer: requireClassicAddress(issuer, "asset issuer") };
}

function requireIssuedAmount(amount: unknown): string {
  if (typeof amount !== "string") {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "issued XRPL amount must be a decimal string",
    );
  }
  const trimmed = amount.trim();
  if (
    !DECIMAL_PATTERN.test(trimmed) ||
    trimmed.replace(".", "").replace(/0/g, "") === ""
  ) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "issued XRPL amount must be a positive decimal string",
    );
  }
  return trimmed;
}
