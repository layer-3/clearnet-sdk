import type { Metadata } from "next";
import { ComingSoon } from "@/features/soon/coming-soon";

export const metadata: Metadata = { title: "Fund" };

export default function FundPage() {
  return (
    <ComingSoon
      title="Fund"
      lede="Add assets to your Yellow account from the chains you already use. They appear in your balance once the deposit is confirmed."
      options={[
        { chain: "mint", title: "Mint test YELLOW", tag: "Testnet" },
        { chain: "evm", title: "EVM Chains" },
        { chain: "sol", title: "Solana" },
        { chain: "xrpl", title: "XRP Ledger" },
        { chain: "btc", title: "Bitcoin" },
      ]}
    />
  );
}
