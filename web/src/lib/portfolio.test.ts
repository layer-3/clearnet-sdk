import { describe, expect, it } from "vitest";
import type { AssetInfo, Balance } from "@yellow-org/clearnet-sdk/gateway";
import { validate, newLeg, maxSendable } from "@/features/send/model";
import { buildPortfolio, estimateFee, priceMap, toYellow } from "./portfolio";

const assets: AssetInfo[] = [
  { symbol: "ETH", chainId: 1, address: "", decimals: 18, native: true },
  { symbol: "USDC", chainId: 1, address: "0x1", decimals: 6, native: false },
  { symbol: "YELLOW", chainId: 1, address: "0x2", decimals: 18, native: false },
];
const pools = [
  { poolId: "eth", tokenA: "ETH", tokenB: "YELLOW", spotPrice: "24850000000000000000000", transferPrice: "24850" },
  { poolId: "usdc", tokenA: "USDC", tokenB: "YELLOW", spotPrice: "10000000000000000000000000000000", transferPrice: "10000000000000" },
];
const E18 = 10n ** 18n;
const balance: Balance = {
  address: "0x00000000000000000000000000000000000000aa",
  nonce: 3,
  balances: [
    { asset: "YELLOW", symbol: "YELLOW", clearnetBalance: (20n * E18).toString() },
    { asset: "USDC", symbol: "USDC", clearnetBalance: "100000000" },
  ],
};
const SELF = "0x00000000000000000000000000000000000000aa";
const OTHER = "0x00000000000000000000000000000000000000bb";

describe("portfolio", () => {
  it("values holdings in YELLOW and checks activation", () => {
    const p = buildPortfolio(assets, balance, pools);
    expect(p.totalYellow).toBe(20n * E18 + 1000n * E18);
    expect(p.activated).toBe(true);
    expect(p.holdings[0]!.asset.symbol).toBe("YELLOW");
    expect(buildPortfolio(assets, { ...balance, balances: [] }, pools).activated).toBe(false);
  });

  it("leaves the total unknown when a held asset has no price", () => {
    expect(buildPortfolio(assets, balance, []).totalYellow).toBeNull();
  });

  it("estimates fees at 10 bps with a 0.0001 YELLOW floor", () => {
    expect(estimateFee([{ asset: "USDC", amount: "100000000" }], assets, pools).fee).toBe(E18); // 1000 YELLOW -> 1
    expect(estimateFee([{ asset: "YELLOW", amount: "1" }], assets, pools).fee).toBe(10n ** 14n);
    expect(estimateFee([{ asset: "USDC", amount: "100000000" }], assets, pools).suggestedCap).toBe("1.5");
    expect(estimateFee([{ asset: "ETH", amount: "1" }], assets, [])).toMatchObject({ fee: null, suggestedCap: null });
  });

  it("applies the fee floor separately to every asset leg", () => {
    expect(estimateFee([{ asset: "ETH", amount: "1" }, { asset: "YELLOW", amount: "1" }], assets, pools).fee).toBe(2n * 10n ** 14n);
  });

  it("suggests a two-significant-digit cap without dropping trailing zeros", () => {
    const capFor = (yellow: string) => estimateFee([{ asset: "YELLOW", amount: yellow }], assets, []).suggestedCap;
    const at = (n: number) => (BigInt(n) * E18).toString();
    expect(capFor(at(1_000))).toBe("1.5"); // fee 1 YELLOW
    expect(capFor(at(20_000))).toBe("30"); // fee 20 YELLOW (regression: was "3")
    expect(capFor(at(100_000))).toBe("150"); // fee 100 YELLOW (was "15")
    expect(capFor(at(1_250_000))).toBe("1900"); // fee 1250 YELLOW (was "19")
  });

  it("keeps an enormous suggested cap in plain decimal notation", () => {
    // 7e41 base units is far beyond Number.MAX_SAFE_INTEGER and beyond 1e21,
    // where toFixed would switch to exponential notation.
    const cap = estimateFee([{ asset: "YELLOW", amount: (7n * 10n ** 41n).toString() }], assets, []).suggestedCap;
    expect(cap).toBe((11n * 10n ** 20n).toString());
    expect(cap).toMatch(/^\d+(\.\d+)?$/);
  });

  it("values production fixed-point prices and uses a distinct EMA for fees", () => {
    const production = [{ ...pools[0]!, transferPrice: "20000" }];
    expect(toYellow(E18, assets[0]!, priceMap(production))).toBe(24850n * E18);
    expect(estimateFee([{ asset: "ETH", amount: E18.toString() }], assets, production).fee).toBe(20n * E18);
    expect(estimateFee([{ asset: "ETH", amount: "1" }], assets, [{ ...pools[0]!, transferPrice: undefined }]).fee).toBeNull();
  });

  it("rejects malformed, zero and oversized price fields", () => {
    for (const raw of ["1.25", "-1", "0", "1e18", "9".repeat(100), (2n ** 256n).toString()]) {
      const invalid = [{ ...pools[0]!, spotPrice: raw, transferPrice: raw }];
      expect(toYellow(E18, assets[0]!, priceMap(invalid))).toBeNull();
      expect(estimateFee([{ asset: "ETH", amount: E18.toString() }], assets, invalid).fee).toBeNull();
    }
  });

  it("inverts YELLOW-quoted base-unit prices exactly across different decimals", () => {
    // 0.25 USDC per YELLOW: 250000 / 1e18 * 1e18 = 250000.
    const quoted = [{ poolId: "yellow-usdc", tokenA: "YELLOW", tokenB: "USDC", spotPrice: "250000" }];
    expect(toYellow(1_000_000n, assets[1]!, priceMap(quoted))).toBe(4n * E18);
  });
});

describe("send validation", () => {
  const p = buildPortfolio(assets, balance, pools);
  const leg = (asset: string, amount: string) => ({ ...newLeg(asset), amount });

  it("accepts a valid multi-asset transfer", () => {
    const r = validate({ recipient: OTHER, legs: [leg("USDC", "10"), leg("YELLOW", "2")], feeCap: null }, p, assets, pools, SELF);
    expect(r.ready).toBe(true);
    expect(r.legs).toEqual([
      { asset: "USDC", amount: "10000000" },
      { asset: "YELLOW", amount: (2n * E18).toString() },
    ]);
  });

  it("refuses self-sends, overdrafts and fee room", () => {
    expect(validate({ recipient: SELF, legs: [leg("USDC", "1")], feeCap: null }, p, assets, pools, SELF).recipientError).toMatch(/own account/);
    const over = validate({ recipient: OTHER, legs: [leg("USDC", "101")], feeCap: null }, p, assets, pools, SELF);
    expect(Object.values(over.legErrors)[0]).toMatch(/You have 100 USDC/);
    const noRoom = validate({ recipient: OTHER, legs: [leg("YELLOW", "20")], feeCap: null }, p, assets, pools, SELF);
    expect(Object.values(noRoom.legErrors)[0]).toMatch(/Leave room for the fee/);
  });

  it("flags a fee cap below the estimate", () => {
    const r = validate({ recipient: OTHER, legs: [leg("USDC", "100")], feeCap: "0.5" }, p, assets, pools, SELF);
    expect(r.feeCapError).toMatch(/lower cap will be rejected/);
  });

  it("requires a consciously chosen fee cap when the pool price is unavailable", () => {
    const state = { recipient: OTHER, legs: [leg("USDC", "1")], feeCap: null };
    const unknown = validate(state, p, assets, [], SELF);
    expect(unknown.fee).toBeNull();
    expect(unknown.feeCap).toBe("");
    expect(unknown.feeCapError).toMatch(/Enter a YELLOW fee cap/);
    expect(unknown.ready).toBe(false);
    const explicit = validate({ ...state, feeCap: "1" }, p, assets, [], SELF);
    expect(explicit.fee).toBeNull();
    expect(explicit.ready).toBe(true);
    expect(explicit.feeCap).toBe("1");
  });

  it("retains the post-transfer 10 YELLOW floor, including for Max", () => {
    const r = validate({ recipient: OTHER, legs: [leg("YELLOW", "10")], feeCap: "0.1" }, p, assets, pools, SELF);
    expect(r.ready).toBe(false);
    expect(Object.values(r.legErrors)[0]).toMatch(/10 YELLOW minimum/);
    expect(maxSendable(assets[2]!, p, E18 / 10n)).toBe(99n * E18 / 10n);
  });

  it("accepts comma decimals", () => {
    const r = validate({ recipient: OTHER, legs: [leg("USDC", "1,5")], feeCap: null }, p, assets, pools, SELF);
    expect(r.legs[0]!.amount).toBe("1500000");
  });
});
