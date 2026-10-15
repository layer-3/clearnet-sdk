import { describe, expect, it } from "vitest";
import type { AssetInfo, Balance } from "@yellow-org/clearnet-sdk/gateway";
import { validate, newLeg, maxSendable } from "@/features/send/model";
import { buildPortfolio, estimateFee } from "./portfolio";

const assets: AssetInfo[] = [
  { symbol: "ETH", chainId: 1, address: "", decimals: 18, native: true },
  { symbol: "USDC", chainId: 1, address: "0x1", decimals: 6, native: false },
  { symbol: "YELLOW", chainId: 1, address: "0x2", decimals: 18, native: false },
];
const pools = [
  { poolId: "eth", tokenA: "ETH", tokenB: "YELLOW", spotPrice: "24850" },
  { poolId: "usdc", tokenA: "USDC", tokenB: "YELLOW", spotPrice: "10" },
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
    expect(estimateFee([{ asset: "ETH", amount: "1" }], assets, []).fee).toBeNull();
  });

  it("applies the fee floor separately to every asset leg", () => {
    expect(estimateFee([{ asset: "ETH", amount: "1" }, { asset: "YELLOW", amount: "1" }], assets, pools).fee).toBe(2n * 10n ** 14n);
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
