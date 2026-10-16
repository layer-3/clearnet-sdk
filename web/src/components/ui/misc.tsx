"use client";

import { clsx } from "clsx";
import type { ReactNode } from "react";
import { amountParts } from "@/lib/format";
import { Button } from "./button";
import { IconAlert, IconRefresh } from "./icons";

export const ASSET_NAMES: Record<string, string> = {
  YELLOW: "Yellow",
  ETH: "Ether",
  WETH: "Wrapped Ether",
  USDC: "USD Coin",
  USDT: "Tether",
  BTC: "Bitcoin",
  SOL: "Solana",
  XRP: "XRP",
};

/** Exact amount with the fraction de-emphasized. Never rounds up. */
export function Amount({ raw, decimals, maxFraction = 6, className, fractionClass = "opacity-55" }: { raw: bigint | string; decimals: number; maxFraction?: number; className?: string; fractionClass?: string }) {
  const p = amountParts(raw, decimals, maxFraction);
  return (
    <span className={clsx("num", className)}>
      {p.whole}
      {p.fraction && <span className={fractionClass}>.{p.fraction}</span>}
    </span>
  );
}

export type Tone = "ok" | "wait" | "warn" | "bad" | "neutral";

export function StatusPill({ tone, children, className }: { tone: Tone; children: ReactNode; className?: string }) {
  return (
    <span
      className={clsx(
        "type-label inline-flex h-7 items-center gap-1.5 rounded-control px-2.5 text-[12.5px]",
        tone === "ok" && "bg-leaf/14 text-leaf",
        tone === "wait" && "bg-current/8",
        tone === "warn" && "bg-sodium/25 text-[#7a5200] dark:text-signal",
        tone === "bad" && "bg-flare/12 text-flare",
        tone === "neutral" && "bg-current/8",
        className,
      )}
    >
      <span className={clsx("size-1.5 rounded-full bg-current", tone === "wait" && "animate-[pulse-dot_1.2s_ease-in-out_infinite]")} aria-hidden />
      {children}
    </span>
  );
}

export function ErrorPanel({ error, onRetry, className }: { error: unknown; onRetry?: () => void; className?: string }) {
  const kind = error && typeof error === "object" && "kind" in error ? String((error as { kind: unknown }).kind) : "";
  const title =
    kind === "trust"
      ? "Network identity changed"
      : kind === "identity"
        ? "Wrong network behind this address"
        : kind === "capability"
          ? "Not available on this network"
          : "Can't reach Yellow Network";
  const raw = error instanceof Error ? error.message : "Something went wrong.";
  const status = error && typeof error === "object" && "httpStatus" in error ? Number((error as { httpStatus: unknown }).httpStatus) : 0;
  const unreachable = kind === "network" || status >= 500;
  const message = unreachable ? "The network didn't answer. Check your connection, or try again in a moment." : raw;
  const detail = unreachable && raw !== message ? raw : undefined;
  return (
    <div role="alert" className={clsx("sheet flex flex-col items-start gap-4 p-6 sm:flex-row sm:items-center", className)}>
      <span className="grid size-11 shrink-0 place-items-center rounded-full bg-flare/12 text-flare">
        <IconAlert />
      </span>
      <div className="min-w-0 flex-1">
        <p className="type-title text-[17px]">{title}</p>
        <p className="mt-1 text-[15px] text-muted">{message}</p>
        {detail && <p className="mt-1 font-mono text-[12px] text-muted opacity-70">{detail}</p>}
      </div>
      {onRetry && (
        <Button variant="quiet" onClick={onRetry} icon={<IconRefresh size={18} />}>
          Try again
        </Button>
      )}
    </div>
  );
}

export function SectionHeader({ title, action, className }: { title: string; action?: ReactNode; className?: string }) {
  return (
    <div className={clsx("flex items-center justify-between gap-4", className)}>
      <h2 className="type-title text-[20px]">{title}</h2>
      {action}
    </div>
  );
}
