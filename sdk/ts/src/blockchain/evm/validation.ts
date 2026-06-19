import { isAddress, zeroHash } from "viem";
import type { Account, Address, Hash } from "viem";

import { ClearnetSdkError } from "../../core/errors.js";
import type { TxRef } from "../../core/types.js";

const UINT256_MAX = (1n << 256n) - 1n;
const HASH_PATTERN = /^0x[a-fA-F0-9]{64}$/;

export interface ValidatedDepositDestination {
  account: Address;
  ref: Hash;
}

export function requireAddress(value: unknown, field: string): Address {
  if (typeof value !== "string" || !isAddress(value)) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `${field} must be a valid EVM address`,
    );
  }
  return value;
}

export function requireWalletAccount(account: Account | Address): Account | Address {
  if (typeof account === "string") {
    return requireAddress(account, "walletAccount");
  }
  if (!account || typeof account !== "object") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "walletAccount is required",
    );
  }
  requireAddress(account.address, "walletAccount.address");
  return account;
}

export function walletAccountAddress(account: Account | Address): Address {
  return typeof account === "string" ? account : account.address;
}

export function requireAmount(amount: unknown): bigint {
  if (typeof amount !== "bigint") {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must be a bigint in base units",
    );
  }
  if (amount <= 0n) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must be greater than zero",
    );
  }
  if (amount > UINT256_MAX) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must fit in uint256",
    );
  }
  return amount;
}

export function requireDepositDestination(
  destination: unknown,
): ValidatedDepositDestination {
  if (!destination || typeof destination !== "object") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a valid EVM address",
    );
  }
  const fields = destination as Record<"account" | "ref", unknown>;
  return {
    account: requireAddress(fields.account, "destination.account"),
    ref: requireReference(fields.ref),
  };
}

export function requireChainId(chainId: number): number {
  if (!Number.isSafeInteger(chainId) || chainId <= 0) {
    throw new ClearnetSdkError(
      "CHAIN_MISMATCH",
      "chainId must be a positive safe integer",
    );
  }
  return chainId;
}

export function requireReference(reference: unknown): Hash {
  if (reference === undefined || reference === "") {
    return zeroHash;
  }
  if (typeof reference !== "string" || !HASH_PATTERN.test(reference)) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "destination.ref must be a 32-byte hex value",
    );
  }
  return reference as Hash;
}

export function requireTxRef(ref: unknown): Hash {
  if (!ref || typeof ref !== "object" || !("hash" in ref)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must be a 32-byte EVM transaction hash",
    );
  }
  const hash = (ref as Record<"hash", unknown>).hash;
  if (typeof hash !== "string" || !HASH_PATTERN.test(hash)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must be a 32-byte EVM transaction hash",
    );
  }
  return hash as Hash;
}

export function txRef(hash: Hash): TxRef {
  return { hash, raw: hash };
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

export function isTransactionNotFound(error: unknown): boolean {
  const name = getErrorField(error, "name");
  if (
    name === "TransactionNotFoundError" ||
    name === "TransactionReceiptNotFoundError"
  ) {
    return true;
  }
  const message = getErrorField(error, "message").toLowerCase();
  return message.includes("not found") && message.includes("transaction");
}

function getErrorField(error: unknown, field: "name" | "message"): string {
  if (error && typeof error === "object" && field in error) {
    const value = (error as Record<typeof field, unknown>)[field];
    return typeof value === "string" ? value : "";
  }
  return "";
}
