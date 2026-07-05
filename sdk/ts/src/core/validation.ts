import { ClearnetSdkError } from "./errors.js";

export const BYTES32_HEX_PATTERN = /^0x[a-fA-F0-9]{64}$/;
export const ZERO_BYTES32_PATTERN = /^0x0{64}$/i;

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

export function normalizeReceiptTimeoutMs(value: number): number {
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new ClearnetSdkError(
      "RECEIPT_TIMEOUT",
      "receiptTimeoutMs must be a positive safe integer",
    );
  }
  return value;
}
