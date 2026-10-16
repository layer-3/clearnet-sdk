// Wire types of the Clearnet public gateway. Mirrors pkg/client/types.go.

export const CAPABILITY_TRANSFER_STATUS = "transfer-operation-status-v1";
export const CAPABILITY_RECENT_TRANSFERS = "recent-transfers-v1";

export interface GatewayContracts {
  custody: string;
  registry: string;
  token: string;
  faucet: string;
}

export interface AssetInfo {
  symbol: string;
  chainId: number;
  address: string;
  decimals: number;
  native: boolean;
}

export interface NetworkInfo {
  chainId: number;
  networkId: string;
  rpcUrl?: string;
  blockExplorer?: string;
  l1ChainId?: number;
  confirmations?: number;
  contracts?: Partial<GatewayContracts>;
  assets: AssetInfo[];
  capabilities: string[];
  virtualRouter: string;
  version?: string;
}

export interface BalanceEntry {
  asset: string;
  symbol: string;
  l1Balance?: string;
  clearnetBalance: string;
  pendingWithdrawal?: string;
  feeRevenue?: string;
  usdValue?: string;
}

export interface Balance {
  address: string;
  accountType?: string;
  nonce: number;
  balances: BalanceEntry[];
  codes?: Record<string, Record<string, string>>;
  totalUsdValue?: string;
}

/** One asset leg in integer base units. */
export interface TransferAsset {
  asset: string;
  amount: string;
}

/** One asset leg in human decimal units, as typed by the user. */
export interface TransferAmount {
  asset: string;
  amount: string;
}

export interface TransferIntent {
  /** Metadata used for conversion; a local guard, outside the EIP-712 payload. */
  catalogueFingerprint: string;
  networkId: string;
  chainId: number;
  from: `0x${string}`;
  /** Routed recipient. A canonical yellow:// URI signs its derived EIP-712 address slot. */
  to: `0x${string}` | `yellow://${string}`;
  assets: TransferAsset[];
  /** Integer YELLOW base units. "0" authorizes no fee. */
  maxFee: string;
  nonce: number;
}

/**
 * A spend authorization. Persist it before submitting; never rebuild it at a
 * fresh nonce after an uncertain outcome.
 */
export interface SignedTransfer {
  operationId: `0x${string}`;
  intent: TransferIntent;
  signature: `0x${string}`;
}

export interface Receipt {
  blockHash: string;
  entryIndex: number;
  sealedAt: number;
  stateRoot: string;
  entryHash?: string;
  /** Must match the signed intent; missing or mismatched IDs are not seals. */
  operationId: string;
}

export type GatewayOperationStatus = "pending" | "sealed" | "rejected" | "unknown";

export interface OperationStatus {
  operationId: string;
  kind: string;
  status: GatewayOperationStatus;
  reason?: string;
  /** Unix seconds; end of the retained routing wait window. Allows identical
   * replay only; it does not expire the signature or prove non-execution. */
  pendingUntil?: number;
  receipt?: Receipt;
}

export interface SubmitResult {
  operationId: `0x${string}`;
  status: "sealed" | "pending";
  receipt?: Receipt;
}

export interface TransferActivity {
  operationId?: string;
  blockHash: string;
  entryHash: string;
  entryIndex: number;
  sealedAt: number;
  from: string;
  to: string;
  direction: string;
  delivery?: string;
  nonce: number;
  assets: TransferAsset[];
}

export interface TransferPage {
  items: TransferActivity[];
  nextCursor?: string | null;
  retentionDays: number;
}

export interface PoolSummary {
  poolId: string;
  tokenA: string;
  tokenB: string;
  /** tokenB base units per tokenA base unit, multiplied by 10^18. */
  spotPrice: string;
  /** Effective integer YELLOW base units per tokenA base unit for transfer fees.
   * Missing on older gateways; do not substitute spot price for EMA. */
  transferPrice?: string;
}
