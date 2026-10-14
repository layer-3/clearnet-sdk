// Public gateway client: balances, transfer history, EIP-712 transfers and
// operation recovery. Depends only on viem, so it is safe for browser bundles.
// Import it from "@yellow-org/clearnet-sdk/gateway" to keep chain depositors
// (Solana, XRPL, Bitcoin) out of your bundle.

export { GatewayClient, normalizeUserAccount } from "./client.js";
export type { GatewayClientOptions, PrepareTransferInput, TypedDataSigner } from "./client.js";
export {
  AmountError,
  displayParts,
  formatAmount,
  formatBaseUnits,
  fromBaseUnits,
  normalizeAmountInput,
  parseBaseUnitInteger,
  toBaseUnits,
} from "./amounts.js";
export type { DisplayParts } from "./amounts.js";
export {
  clearnetDomain,
  networkChainId,
  normalizeSignature,
  RecipientError,
  recoverTransferSigner,
  resolveRecipient,
  SignatureError,
  sortTransferAssets,
  TRANSFER_OPERATION_DOMAIN,
  TRANSFER_TYPES,
  transferDigest,
  transferOperationId,
  transferTypedData,
  USER_URI_PREFIX,
  validOperationId,
  VIRTUAL_ROUTER,
} from "./eip712.js";
export type { TransferTypedData } from "./eip712.js";
export { APIError, GATEWAY_ERROR_CODES, GatewayError, SubmissionError } from "./errors.js";
export type { APIErrorInit, GatewayErrorKind } from "./errors.js";
export { networkFingerprint } from "./fingerprint.js";
export {
  applySubmission,
  blockingOperation,
  isFinalOperationState,
  markSubmitting,
  newOperationRecord,
  observeOperation,
  retryDecision,
  staleNonceOperation,
} from "./operations.js";
export type { OperationRecord, OperationState } from "./operations.js";
export {
  CAPABILITY_RECENT_TRANSFERS,
  CAPABILITY_TRANSFER_STATUS,
} from "./types.js";
export type {
  AssetInfo,
  Balance,
  BalanceEntry,
  GatewayContracts,
  GatewayOperationStatus,
  NetworkInfo,
  OperationStatus,
  PoolSummary,
  Receipt,
  SignedTransfer,
  SubmitResult,
  TransferActivity,
  TransferAmount,
  TransferAsset,
  TransferIntent,
  TransferPage,
} from "./types.js";
