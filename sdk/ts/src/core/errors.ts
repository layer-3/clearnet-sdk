import type { TxRef } from "./types.js";

export type ClearnetSdkErrorCode =
  | "INVALID_ADDRESS"
  | "INVALID_AMOUNT"
  | "INVALID_CONFIRMATIONS"
  | "INVALID_REFERENCE"
  | "INVALID_TX_REF"
  | "MISSING_WALLET_ACCOUNT"
  | "CHAIN_MISMATCH"
  | "TX_REVERTED"
  | "RECEIPT_TIMEOUT"
  | "RPC_ERROR";

interface ClearnetSdkErrorOptions {
  txRef?: TxRef;
  cause?: unknown;
}

export class ClearnetSdkError extends Error {
  readonly code: ClearnetSdkErrorCode;
  readonly txRef?: TxRef;
  override cause?: unknown;

  constructor(
    code: ClearnetSdkErrorCode,
    message: string,
    options: ClearnetSdkErrorOptions = {},
  ) {
    super(message, causeOptions(options.cause));
    this.name = "ClearnetSdkError";
    this.code = code;
    if (options.txRef !== undefined) {
      this.txRef = options.txRef;
    }
    if (options.cause !== undefined) {
      this.cause = options.cause;
    }
  }
}

function causeOptions(cause: unknown): ErrorOptions | undefined {
  return cause === undefined ? undefined : { cause };
}
