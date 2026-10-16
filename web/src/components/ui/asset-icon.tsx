import { clsx } from "clsx";
import { YellowMark } from "./brand";

const KNOWN: Record<string, { bg: string; fg: string }> = {
  ETH: { bg: "#627eea", fg: "#ffffff" },
  WETH: { bg: "#627eea", fg: "#ffffff" },
  USDC: { bg: "#2775ca", fg: "#ffffff" },
  USDT: { bg: "#26a17b", fg: "#ffffff" },
  BTC: { bg: "#f7931a", fg: "#ffffff" },
  SOL: { bg: "#14151a", fg: "#14f195" },
  XRP: { bg: "#23292f", fg: "#ffffff" },
};

export function AssetIcon({ symbol, size = 40, className }: { symbol: string; size?: number; className?: string }) {
  if (symbol === "YELLOW") return <YellowMark size={size} className={clsx("shrink-0", className)} />;
  const c = KNOWN[symbol] ?? { bg: "#d9d6cc", fg: "#000" };
  return (
    <span
      className={clsx("inline-grid shrink-0 place-items-center rounded-full", className)}
      style={{ width: size, height: size, background: c.bg, color: c.fg }}
      aria-hidden
    >
      {symbol === "ETH" || symbol === "WETH" ? (
        <svg width={size * 0.5} height={size * 0.5} viewBox="0 0 24 24" fill="currentColor">
          <path d="M12 2 5.5 12.3 12 16l6.5-3.7L12 2Z" opacity=".95" />
          <path d="M12 17.3 5.5 13.6 12 22l6.5-8.4-6.5 3.7Z" opacity=".7" />
        </svg>
      ) : symbol === "USDC" || symbol === "USDT" ? (
        <svg width={size * 0.56} height={size * 0.56} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
          <path d="M15.5 8.5c-.6-1-1.9-1.6-3.5-1.6-2 0-3.4 1-3.4 2.5 0 3.6 7.2 1.8 7.2 5.3 0 1.5-1.5 2.5-3.6 2.5-1.7 0-3-.7-3.6-1.8M12 5v2M12 17v2" />
        </svg>
      ) : (
        <span className="type-title" style={{ fontSize: size * 0.38 }}>
          {symbol.slice(0, 1)}
        </span>
      )}
    </span>
  );
}
