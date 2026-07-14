import { Buffer } from "buffer";

import { isValidClassicAddress } from "xrpl";

import { ClearnetSdkError } from "../../core/errors.js";
import type { Bytes32Hex, TxRef } from "../../core/types.js";
import {
  BYTES32_HEX_PATTERN,
  normalizeMinConfirmations,
} from "../../core/validation.js";
import { UINT64_MAX, XRPL_NATIVE_ASSET } from "./constants.js";
import type { XrplDepositDestination, XrplSigner } from "./types.js";

const HASH_PATTERN = /^[a-fA-F0-9]{64}$/;
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
      "destination is required and must be an object",
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
  issuedAssetDecimals: ReadonlyMap<string, number>,
): ResolvedXrplAmount {
  if (isNativeAsset(asset)) {
    return { kind: "native", amount: decimalToBaseUnits(amount, 6).toString() };
  }
  const issued = requireIssuedAsset(asset);
  const decimals = issuedAssetDecimals.get(issued.key);
  if (decimals === undefined) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      `issued XRPL decimals are not configured for ${issued.key}`,
    );
  }
  const value = baseUnitsToDecimal(decimalToBaseUnits(amount, decimals), decimals);
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

export { normalizeMinConfirmations };

function isNativeAsset(asset: unknown): boolean {
  if (asset === "" || asset === undefined) {
    return true;
  }
  return false;
}

function requireIssuedAsset(asset: unknown): {
  key: string;
  currency: string;
  issuer: string;
} {
  if (typeof asset !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "issued XRPL asset must be CUR.rIssuer",
    );
  }
  const trimmed = asset.trim();
  const dot = trimmed.indexOf(".");
  if (
    dot <= 0 ||
    dot !== trimmed.lastIndexOf(".") ||
    dot === trimmed.length - 1 ||
    trimmed.includes(":")
  ) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "issued XRPL asset must be CUR.rIssuer",
    );
  }
  const currency = trimmed.slice(0, dot);
  const issuer = trimmed.slice(dot + 1);
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
  return {
    key: trimmed,
    currency,
    issuer: requireClassicAddress(issuer, "asset issuer"),
  };
}

export function normalizeIssuedAssetDecimals(
  decimals: unknown,
): ReadonlyMap<string, number> {
  if (decimals === undefined) {
    return new Map();
  }
  if (!decimals || typeof decimals !== "object" || Array.isArray(decimals)) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "issuedAssetDecimals must be an object",
    );
  }
  const out = new Map<string, number>();
  for (const [asset, value] of Object.entries(decimals as Record<string, unknown>)) {
    requireIssuedAsset(asset);
    if (
      typeof value !== "number" ||
      !Number.isInteger(value) ||
      value < 0 ||
      value > 255
    ) {
      throw new ClearnetSdkError(
        "INVALID_AMOUNT",
        `issued XRPL decimals for ${asset} must be an integer from 0 to 255`,
      );
    }
    out.set(asset, value);
  }
  return out;
}

function decimalToBaseUnits(amount: unknown, decimals: number): bigint {
  if (typeof amount !== "string" || !/^[0-9]+(?:\.[0-9]+)?$/.test(amount)) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "XRPL amount must be a positive decimal string",
    );
  }
  const [whole, fractional = ""] = amount.split(".");
  if (fractional.length > decimals) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      `XRPL amount has more than ${decimals} decimal places`,
    );
  }
  const padded = fractional.padEnd(decimals, "0");
  const base = BigInt(`${whole}${padded}`.replace(/^0+(?=\d)/, ""));
  if (base <= 0n) {
    throw new ClearnetSdkError("INVALID_AMOUNT", "XRPL amount must be positive");
  }
  if (decimals === 6 && base > UINT64_MAX) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "native XRP amount must fit in uint64 drops",
    );
  }
  return base;
}

function baseUnitsToDecimal(baseUnits: bigint, decimals: number): string {
  if (decimals === 0) {
    return baseUnits.toString();
  }
  const raw = baseUnits.toString().padStart(decimals + 1, "0");
  const whole = raw.slice(0, -decimals);
  const fractional = raw.slice(-decimals).replace(/0+$/, "");
  return fractional === "" ? whole : `${whole}.${fractional}`;
}
