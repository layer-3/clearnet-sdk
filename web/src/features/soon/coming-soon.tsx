import { clsx } from "clsx";
import type { ReactNode } from "react";
import { AssetIcon } from "@/components/ui/asset-icon";
import { IconChevronDown, IconSwap } from "@/components/ui/icons";

function ChainGlyph({ chain }: { chain: "evm" | "sol" | "xrpl" | "btc" | "mint" }) {
  const box = "grid size-12 shrink-0 place-items-center rounded-field ring-1 ring-inset ring-white/12";
  switch (chain) {
    case "evm":
      return (
        <span className={clsx(box, "bg-[#627eea] text-white")}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
            <path d="M12 2 5.5 12.3 12 16l6.5-3.7L12 2Z" />
            <path d="M12 17.3 5.5 13.6 12 22l6.5-8.4-6.5 3.7Z" opacity=".65" />
          </svg>
        </span>
      );
    case "sol":
      return (
        <span className={clsx(box, "bg-[#14151a]")}>
          <svg width="24" height="20" viewBox="0 0 24 20" aria-hidden>
            <defs>
              <linearGradient id="sol" x1="0" x2="1">
                <stop offset="0" stopColor="#9945ff" />
                <stop offset="1" stopColor="#14f195" />
              </linearGradient>
            </defs>
            <path fill="url(#sol)" d="M4 1h19l-3 4H1zM1 8h19l3 4H4zM4 15h19l-3 4H1z" />
          </svg>
        </span>
      );
    case "xrpl":
      return (
        <span className={clsx(box, "bg-[#23292f] text-white")}>
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" aria-hidden>
            <path d="M4 4l5.2 5.2a4 4 0 0 0 5.6 0L20 4M4 20l5.2-5.2a4 4 0 0 1 5.6 0L20 20" />
          </svg>
        </span>
      );
    case "btc":
      return (
        <span className={clsx(box, "bg-[#f7931a] text-white")}>
          <span className="type-title text-[24px]">₿</span>
        </span>
      );
    case "mint":
      return (
        <span className={clsx(box, "bg-signal")}>
          <AssetIcon symbol="YELLOW" size={36} />
        </span>
      );
  }
}

export interface Option {
  chain: "evm" | "sol" | "xrpl" | "btc" | "mint";
  title: string;
  tag?: string;
}

/**
 * Shared layout for features that aren't live yet: heading and a short intro
 * on the left, a preview of what's coming on the right (stacked on phones).
 */
export function ComingSoon({ title, lede, options, aside, children }: { title: string; lede: string; options?: Option[]; aside?: ReactNode; children?: ReactNode }) {
  return (
    <div className="grid gap-8 pt-2 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)] lg:gap-14 lg:pt-6">
      <header>
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="type-h1">{title}</h1>
          <span className="type-label inline-flex h-7 items-center rounded-control bg-accent px-3 text-[13px] text-on-accent">Coming soon</span>
        </div>
        <p className="lede mt-3">{lede}</p>
        {children}
      </header>
      {options && (
        <ul className="sheet divide-y divide-line p-2 sm:p-3">
          {options.map((o) => (
            <li key={o.title} className="flex items-center gap-4 px-3 py-3.5">
              <ChainGlyph chain={o.chain} />
              <p className="type-label min-w-0 flex-1 text-[16px]">{o.title}</p>
              {o.tag && <span className="type-label shrink-0 rounded-control bg-surface-2 px-2.5 py-1 text-[12px] text-muted">{o.tag}</span>}
            </li>
          ))}
        </ul>
      )}
      {aside}
    </div>
  );
}

export function TradePreview() {
  return (
    <div className="sheet relative p-5 sm:p-6" aria-hidden>
      {[
        { label: "You pay", symbol: "ETH", amount: "0.5" },
        { label: "You get", symbol: "YELLOW", amount: "12,425" },
      ].map((row, i) => (
        <div key={row.label}>
          {i === 1 && (
            <div className="relative z-10 -my-3 flex justify-center">
              <span className="grid size-11 place-items-center rounded-control bg-surface text-on-surface shadow-[0_0_0_5px_var(--surface)] ring-1 ring-line">
                <IconSwap size={18} className="rotate-90" />
              </span>
            </div>
          )}
          <div className="rounded-field bg-surface-2 p-4">
            <p className="text-[13.5px] text-muted">{row.label}</p>
            <div className="mt-2 flex items-center justify-between gap-3">
              <span className="type-amount opacity-40">{row.amount}</span>
              <span className="inline-flex h-10 items-center gap-2 rounded-control bg-surface py-1 pl-1 pr-3">
                <AssetIcon symbol={row.symbol} size={30} />
                <span className="type-label text-[15px]">{row.symbol}</span>
                <IconChevronDown size={14} className="text-muted" />
              </span>
            </div>
          </div>
        </div>
      ))}
      <div className="type-label mt-4 grid h-13 place-items-center rounded-control bg-accent/35 text-[16px] text-on-surface/60">Swap</div>
    </div>
  );
}
