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

/** Prices as YELLOW-per-whole-token, keyed by symbol. YELLOW itself is 1. */
export function priceMap(pools: PoolSummary[] | undefined): Map<string, string> {
  const m = new Map<string, string>([[YELLOW, "1"]]);
  for (const p of pools ?? []) {
    if (p.tokenB === YELLOW && p.spotPrice) m.set(p.tokenA, p.spotPrice);
    else if (p.tokenA === YELLOW && p.spotPrice) {
      const d = parseDecimal(p.spotPrice);
      if (d && d.int > 0n) m.set(p.tokenB, (Number(d.scale) / Number(d.int)).toPrecision(12));
    }
  }
  return m;
}

export function toYellow(amount: bigint, asset: AssetInfo, yellowDecimals: number, prices: Map<string, string>): bigint | null {
  if (asset.symbol === YELLOW) return amount;
  const price = prices.get(asset.symbol);
  if (!price) return null;
  const d = parseDecimal(price);
  if (!d) return null;
  return (amount * d.int * 10n ** BigInt(yellowDecimals)) / (10n ** BigInt(asset.decimals) * d.scale);
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
      yellowValue: toYellow(available, asset, yellowDecimals, prices),
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
  /** Suggested ceiling in YELLOW (decimal string). */
  suggestedCap: string;
}

/**
 * Client-side estimate of the network fee, from the protocol rule (fixed 10 bps
 * of each leg's YELLOW value, minimum 0.0001 YELLOW per leg). The signed cap protects
 * the user if the real fee differs: the network rejects, nothing is charged.
 */
export function estimateFee(legs: TransferAsset[], assets: AssetInfo[], pools: PoolSummary[] | undefined): FeeEstimate {
  const yellowDecimals = assets.find((a) => a.symbol === YELLOW)?.decimals ?? 18;
  const min = 10n ** BigInt(Math.max(0, yellowDecimals - 4));
  const prices = priceMap(pools);
  let fee = 0n;
  for (const leg of legs) {
    const asset = assets.find((a) => a.symbol === leg.asset);
    const v = asset ? toYellow(BigInt(leg.amount), asset, yellowDecimals, prices) : null;
    if (v === null) return { fee: null, suggestedCap: "0.1" };
    const raw = (v * FEE_BPS) / 10_000n;
    fee += raw > min ? raw : min;
  }
  // Cap at 1.5x the estimate, rounded up to two significant digits.
  return { fee, suggestedCap: niceCeil(fromBaseUnits((fee * 3n + 1n) / 2n, yellowDecimals)) };
}

function niceCeil(exact: string): string {
  const n = Number(exact);
  if (!Number.isFinite(n) || n <= 0) return "0.0001";
  const mag = 10 ** (Math.floor(Math.log10(n)) - 1);
  const v = Math.ceil(n / mag) * mag;
  const digits = Math.max(0, -Math.floor(Math.log10(mag)));
  return v.toFixed(Math.min(digits, 18)).replace(/\.?0+$/, "") || "0.0001";
}
