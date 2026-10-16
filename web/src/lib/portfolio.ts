import { fromBaseUnits, type AssetInfo, type Balance, type PoolSummary, type TransferAsset } from "@yellow-org/clearnet-sdk/gateway";

export const YELLOW = "YELLOW";
/** An account needs this much available YELLOW before it can send (AccountActivationCost). */
export const ACTIVATION_YELLOW = "10";
/** Each leg pays 0.10% of its YELLOW value, minimum 0.0001 YELLOW per leg. */
export const FEE_BPS = 10n;

export interface Holding {
  asset: AssetInfo;
  available: bigint;
  pendingWithdrawal: bigint;
  /** Value in YELLOW base units, or null when there is no price. */
  yellowValue: bigint | null;
}

export interface Portfolio {
  holdings: Holding[];
  yellow: Holding | undefined;
  /** Sum of every holding in YELLOW base units; null if a non-zero holding has no price. */
  totalYellow: bigint | null;
  activated: boolean;
  nonce: number;
}

function parseDecimal(value: string): { int: bigint; scale: bigint } | null {
  if (!/^\d+(\.\d+)?$/.test(value)) return null;
  const [w, f = ""] = value.split(".");
  return { int: BigInt(w! + f), scale: 10n ** BigInt(f.length) };
}

interface PriceRatio { numerator: bigint; denominator: bigint }
const E18 = 10n ** 18n;

function uint256(raw: string | undefined): bigint | null {
  if (typeof raw !== "string" || raw.length > 78 || !/^\d+$/.test(raw)) return null;
  const n = BigInt(raw);
  return n > 0n && n < 2n ** 256n ? n : null;
}

/** Exact YELLOW base units per asset base unit; gateway spotPrice is scaled by 1e18. */
export function priceMap(pools: PoolSummary[] | undefined): Map<string, PriceRatio> {
  const prices = new Map<string, PriceRatio>([[YELLOW, { numerator: 1n, denominator: 1n }]]);
  for (const pool of pools ?? []) {
    const price = uint256(pool.spotPrice);
    if (price === null) continue;
    if (pool.tokenB === YELLOW && pool.tokenA !== YELLOW) {
      prices.set(pool.tokenA, { numerator: price, denominator: E18 });
    } else if (pool.tokenA === YELLOW && pool.tokenB !== YELLOW) {
      prices.set(pool.tokenB, { numerator: E18, denominator: price });
    }
  }
  return prices;
}

export function toYellow(amount: bigint, asset: AssetInfo, prices: Map<string, PriceRatio>): bigint | null {
  const price = prices.get(asset.symbol);
  return price ? amount * price.numerator / price.denominator : null;
}

export function buildPortfolio(assets: AssetInfo[], balance: Balance, pools: PoolSummary[] | undefined): Portfolio {
  const prices = priceMap(pools);
  const yellowDecimals = assets.find((a) => a.symbol === YELLOW)?.decimals ?? 18;
  const bySymbol = new Map(balance.balances.map((b) => [b.symbol || b.asset, b]));
  const holdings: Holding[] = assets.map((asset) => {
    const entry = bySymbol.get(asset.symbol);
    const available = safeBig(entry?.clearnetBalance);
    return {
      asset,
      available,
      pendingWithdrawal: safeBig(entry?.pendingWithdrawal),
      yellowValue: toYellow(available, asset, prices),
    };
  });
  holdings.sort((a, b) => {
    if (a.asset.symbol === YELLOW) return -1;
    if (b.asset.symbol === YELLOW) return 1;
    return Number((b.yellowValue ?? 0n) - (a.yellowValue ?? 0n)) || a.asset.symbol.localeCompare(b.asset.symbol);
  });
  let totalYellow: bigint | null = 0n;
  for (const h of holdings) {
    if (h.yellowValue === null) {
      if (h.available > 0n) totalYellow = null;
    } else if (totalYellow !== null) totalYellow += h.yellowValue;
  }
  const yellow = holdings.find((h) => h.asset.symbol === YELLOW);
  const activation = 10n * 10n ** BigInt(yellowDecimals);
  return { holdings, yellow, totalYellow, activated: (yellow?.available ?? 0n) >= activation, nonce: balance.nonce };
}

function safeBig(v: string | undefined): bigint {
  try {
    return v && /^\d+$/.test(v) ? BigInt(v) : 0n;
  } catch {
    return 0n;
  }
}

export interface FeeEstimate {
  /** Estimated fee in YELLOW base units, or null if a leg has no price. */
  fee: bigint | null;
  /** Suggested ceiling in YELLOW, or null when the fee cannot be estimated. */
  suggestedCap: string | null;
}

/**
 * Client-side estimate of the network fee, from the protocol rule (fixed 10 bps
 * of each leg's YELLOW value at the gateway's effective EMA price, minimum
 * 0.0001 YELLOW per leg). Older gateways without that price require an explicit
 * cap. The signed cap protects
 * the user if the real fee differs: the network rejects, nothing is charged.
 */
export function estimateFee(legs: TransferAsset[], assets: AssetInfo[], pools: PoolSummary[] | undefined): FeeEstimate {
  const yellowDecimals = assets.find((a) => a.symbol === YELLOW)?.decimals ?? 18;
  const min = 10n ** BigInt(Math.max(0, yellowDecimals - 4));
  const prices = new Map<string, bigint>([[YELLOW, 1n]]);
  for (const pool of pools ?? []) {
    const price = uint256(pool.transferPrice);
    if (pool.tokenB === YELLOW && pool.tokenA !== YELLOW && price !== null) prices.set(pool.tokenA, price);
  }
  let fee = 0n;
  for (const leg of legs) {
    const asset = assets.find((a) => a.symbol === leg.asset);
    const price = prices.get(leg.asset);
    const v = asset && price !== undefined ? BigInt(leg.amount) * price : null;
    if (v === null) return { fee: null, suggestedCap: null };
    const raw = (v * FEE_BPS) / 10_000n;
    fee += raw > min ? raw : min;
  }
  // Cap at 1.5x the estimate, rounded up to two significant digits.
  return { fee, suggestedCap: niceCeil(fromBaseUnits((fee * 3n + 1n) / 2n, yellowDecimals)) };
}

/** Rounds an exact decimal string up to two significant digits, without floats. */
function niceCeil(exact: string): string {
  const d = parseDecimal(exact);
  if (!d || d.int <= 0n) return "0.0001";
  const fractionDigits = (d.scale.toString().length - 1) as number;
  // Decimal exponent of the value: floor(log10(int / scale)).
  const exponent = d.int.toString().length - fractionDigits - 1;
  // Round up to a multiple of 10^(exponent - 1) in scaled units, never finer
  // than the value's own precision.
  const stepExponent = exponent - 1 + fractionDigits;
  const step = stepExponent > 0 ? 10n ** BigInt(stepExponent) : 1n;
  const rounded = ((d.int + step - 1n) / step) * step;
  return scaledToDecimal(rounded, fractionDigits) || "0.0001";
}

/** Formats a scaled integer as a decimal string, trimming fractional zeros only. */
function scaledToDecimal(value: bigint, fractionDigits: number): string {
  if (fractionDigits === 0) return value.toString();
  const digits = value.toString().padStart(fractionDigits + 1, "0");
  const whole = digits.slice(0, digits.length - fractionDigits);
  const fraction = digits.slice(digits.length - fractionDigits).replace(/0+$/, "");
  return fraction ? `${whole}.${fraction}` : whole;
}
