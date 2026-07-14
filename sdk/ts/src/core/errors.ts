export type ClearnetSdkErrorCode =
  | "INVALID_INPUT"
  | "INVALID_ADDRESS"
  | "INVALID_AMOUNT"
  | "INVALID_CONFIRMATIONS"
  | "INVALID_REFERENCE"
  | "INVALID_TX_ID"
  | "MISSING_WALLET_ACCOUNT"
  | "INSUFFICIENT_FUNDS"
  | "CHAIN_MISMATCH"
  | "TX_REVERTED"
  | "RECEIPT_TIMEOUT"
  | "RPC_ERROR";

interface ClearnetSdkErrorOptions {
  txID?: string;
  cause?: unknown;
}

export class ClearnetSdkError extends Error {
  readonly code: ClearnetSdkErrorCode;
  readonly txID?: string;
  override cause?: unknown;

  constructor(
    code: ClearnetSdkErrorCode,
    message: string,
    options: ClearnetSdkErrorOptions = {},
  ) {
    super(message, causeOptions(options.cause));
    this.name = "ClearnetSdkError";
    this.code = code;
    if (options.txID !== undefined) {
      this.txID = options.txID;
    }
    if (options.cause !== undefined) {
      this.cause = options.cause;
    }
  }
}

function causeOptions(cause: unknown): ErrorOptions | undefined {
  return cause === undefined ? undefined : { cause };
}
