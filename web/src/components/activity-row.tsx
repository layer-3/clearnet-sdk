"use client";

import type { AssetInfo, TransferActivity } from "@yellow-org/clearnet-sdk/gateway";
import { clsx } from "clsx";
import { Identicon } from "@/components/ui/identicon";
import { IconReceive, IconSend } from "@/components/ui/icons";
import { Amount } from "@/components/ui/misc";
import { isIncoming, shortAddress, timeAgo } from "@/lib/format";

export function ActivityRow({ item, assets, onSelect }: { item: TransferActivity; assets: AssetInfo[]; onSelect?: () => void }) {
  const incoming = isIncoming(item.direction);
  const self = item.direction === "self";
  const counterparty = incoming ? item.from : item.to;
  return (
    <li>
      <button type="button" onClick={onSelect} className="flex w-full items-center gap-3.5 rounded-field px-3 py-3 text-left transition-colors hover:bg-surface-2 sm:gap-4">
        <span className="relative shrink-0">
          <Identicon address={counterparty} size={42} />
          <span
            className={clsx(
              "absolute -bottom-1 -right-1 grid size-[22px] place-items-center rounded-full ring-[3px] ring-surface",
              incoming ? "bg-leaf text-white" : "bg-ink text-signal dark:bg-signal dark:text-ink",
            )}
            aria-hidden
          >
            {incoming ? <IconReceive size={13} strokeWidth={2.6} /> : <IconSend size={13} strokeWidth={2.6} />}
          </span>
        </span>
        <span className="min-w-0 flex-1">
          <span className="type-label block truncate text-[15.5px]">{self ? "To yourself" : incoming ? "Received" : "Sent"}</span>
          <span className="block truncate text-[13.5px] text-muted">
            {incoming ? "from " : "to "}
            <span className="font-mono text-[12.5px]">{shortAddress(counterparty)}</span>
          </span>
        </span>
        <span className="flex shrink-0 flex-col items-end">
          {item.assets.map((leg) => {
            const meta = assets.find((a) => a.symbol === leg.asset);
            return (
              <span key={leg.asset} className={clsx("type-label text-[15.5px]", incoming && "text-leaf")}>
                {incoming ? "+" : "−"}
                {meta ? <Amount raw={leg.amount} decimals={meta.decimals} maxFraction={4} fractionClass="" /> : leg.amount} {leg.asset}
              </span>
            );
          })}
          <span className="text-[12.5px] text-muted">{timeAgo(item.sealedAt)}</span>
        </span>
      </button>
    </li>
  );
}

export function ActivitySkeleton({ rows = 3 }: { rows?: number }) {
  return (
    <ul className="space-y-1" aria-hidden>
      {Array.from({ length: rows }, (_, i) => (
        <li key={i} className="flex items-center gap-4 px-3 py-3">
          <span className="skeleton size-[42px] !rounded-full" />
          <span className="flex-1 space-y-2">
            <span className="skeleton block h-4 w-24" />
            <span className="skeleton block h-3 w-32" />
          </span>
          <span className="skeleton h-4 w-20" />
        </li>
      ))}
    </ul>
  );
}
