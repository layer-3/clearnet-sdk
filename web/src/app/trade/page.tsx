import type { Metadata } from "next";
import { ComingSoon, TradePreview } from "@/features/soon/coming-soon";

export const metadata: Metadata = { title: "Trade" };

export default function TradePage() {
  return (
    <ComingSoon title="Trade" lede="Swap between the assets in your account." aside={<TradePreview />} />
  );
}
