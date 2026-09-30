export type ClearnetSdkErrorCode =
  | "INVALID_INPUT"
  | "INVALID_ADDRESS"
  | "INVALID_AMOUNT"
  | "INVALID_CONFIRMATIONS"
  | "INVALID_REFERENCE"
  | "INVALID_TX_ID"
  | "INVALID_DEPOSIT_ID"
  | "MISSING_WALLET_ACCOUNT"
  | "INSUFFICIENT_FUNDS"
  | "CHAIN_MISMATCH"
  | "TX_REVERTED"
  | "RECEIPT_TIMEOUT"
  | "STALE_NONCE"
  | "DEPOSIT_EVENT_NOT_FOUND"
  | "RPC_ERROR";

// DepositStep names which half of an EVM deposit a ClearnetSdkError failed in.
// Not meaningful on chains where submitDeposit is a single transaction.
export type DepositStep = "approve" | "deposit";

interface ClearnetSdkErrorOptions {
  // txHash is the on-chain transaction/signature hash this error concerns,
  // when known. On EVM it differs from depositId, hence separate fields.
  txHash?: string;
  // depositId is the deposit ID this error concerns, when known before the
  // failure (the nonce is read before an EVM approve/submit step).
  depositId?: string;
  // step is which half of an EVM deposit failed ("approve" or "deposit").
  step?: DepositStep;
  // nonceKey and nonce are the EVM deposit's 2D nonce key and full nonce
  // (key << 64 | sequence), set with depositId. Without a txHash, compare nonce
  // against a fresh getNonce(depositor, nonceKey) to see whether it landed.
  nonceKey?: bigint;
  nonce?: bigint;
  cause?: unknown;
}

export class ClearnetSdkError extends Error {
  readonly code: ClearnetSdkErrorCode;
  readonly txHash?: string;
  readonly depositId?: string;
  readonly step?: DepositStep;
  readonly nonceKey?: bigint;
  readonly nonce?: bigint;
  override cause?: unknown;

  constructor(
    code: ClearnetSdkErrorCode,
    message: string,
    options: ClearnetSdkErrorOptions = {},
  ) {
    super(message, causeOptions(options.cause));
    this.name = "ClearnetSdkError";
    this.code = code;
    if (options.txHash !== undefined) {
      this.txHash = options.txHash;
    }
    if (options.depositId !== undefined) {
      this.depositId = options.depositId;
    }
    if (options.step !== undefined) {
      this.step = options.step;
    }
    if (options.nonceKey !== undefined) {
      this.nonceKey = options.nonceKey;
    }
    if (options.nonce !== undefined) {
      this.nonce = options.nonce;
    }
    if (options.cause !== undefined) {
      this.cause = options.cause;
    }
  }
}

function causeOptions(cause: unknown): ErrorOptions | undefined {
  return cause === undefined ? undefined : { cause };
}
