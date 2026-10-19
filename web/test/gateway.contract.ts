import { expect, it } from "vitest";
import { GatewayClient, type Balance } from "@yellow-org/clearnet-sdk/gateway";
import { privateKeyToAccount, generatePrivateKey } from "viem/accounts";
import { buildPortfolio, estimateFee } from "@/lib/portfolio";

it("values production gateway prices and estimates fees from its effective EMA", async () => {
  const baseUrl = process.env.YELLOW_GATEWAY_CONTRACT_URL;
  if (!baseUrl || !/^http:\/\/127\.0\.0\.1:\d+$/.test(baseUrl)) throw new Error("Run scripts/test-gateway-contract.sh with its local gateway harness.");
  const client = new GatewayClient({ baseUrl, networkId: "YDEV" });
  const info = await client.info();
  const pools = await client.pools();
  const e18 = 10n ** 18n;
  const account = privateKeyToAccount(generatePrivateKey());
  const balance: Balance = { address: account.address, nonce: 0, balances: [
    { asset: "ETH", symbol: "ETH", clearnetBalance: e18.toString() },
    { asset: "USDC", symbol: "USDC", clearnetBalance: "100000000" },
    { asset: "YELLOW", symbol: "YELLOW", clearnetBalance: (100n * e18).toString() },
  ] };
  expect(info.assets.find((a) => a.symbol === "USDC")?.decimals).toBe(6);
  const portfolio = buildPortfolio(info.assets, balance, pools);
  expect(portfolio.holdings.find((h) => h.asset.symbol === "ETH")?.yellowValue).toBe(24850n * e18);
  expect(portfolio.holdings.find((h) => h.asset.symbol === "USDC")?.yellowValue).toBe(1000n * e18);
  // ETH is past bootstrap and has a different EMA; USDC is still at reserves.
  expect(estimateFee([{ asset: "ETH", amount: e18.toString() }, { asset: "USDC", amount: "100000000" }], info.assets, pools).fee).toBe(21n * e18);
  const intent = await client.prepareTransfer({ from: account.address, recipient: "0x00000000000000000000000000000000000000bb", assets: [{ asset: "USDC", amount: "1" }], maxFee: "1" });
  expect(intent.assets).toEqual([{ asset: "USDC", amount: "1000000" }]);
  const signed = await client.signTransfer(intent, (data) => account.signTypedData(data));
  expect(signed.intent.catalogueFingerprint).toBe(intent.catalogueFingerprint);
});
