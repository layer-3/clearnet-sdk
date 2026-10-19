import type { Metadata } from "next";
import { ComingSoon } from "@/features/soon/coming-soon";

export const metadata: Metadata = { title: "Withdraw" };

export default function WithdrawPage() {
  return (
    <ComingSoon
      title="Withdraw"
      lede="Move assets from your Yellow account to an address on another blockchain."
      options={[
        { chain: "evm", title: "EVM Chains" },
        { chain: "sol", title: "Solana" },
        { chain: "xrpl", title: "XRP Ledger" },
        { chain: "btc", title: "Bitcoin" },
      ]}
    />
  );
}
