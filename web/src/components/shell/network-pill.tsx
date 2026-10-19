"use client";

import { clsx } from "clsx";
import { useState } from "react";
import { NETWORKS } from "@/config/networks";
import { useInfo } from "@/hooks/gateway";
import { useNetwork } from "@/lib/network-context";
import { Sheet } from "@/components/ui/sheet";
import { IconCheck, IconChevronDown } from "@/components/ui/icons";

function useStatus() {
  const info = useInfo();
  if (info.isPending) return { tone: "wait" as const, text: "Checking" };
  if (info.error) return { tone: "bad" as const, text: "Unreachable" };
  return { tone: "ok" as const, text: "Live" };
}

function Dot({ tone }: { tone: "ok" | "wait" | "bad" }) {
  return (
    <span
      className={clsx(
        "size-2 shrink-0 rounded-full",
        tone === "ok" && "bg-leaf",
        tone === "wait" && "animate-[pulse-dot_1.2s_ease-in-out_infinite] bg-current",
        tone === "bad" && "bg-flare",
      )}
      aria-hidden
    />
  );
}

export function NetworkPill({ className }: { className?: string }) {
  const { network, setNetwork } = useNetwork();
  const status = useStatus();
  const [open, setOpen] = useState(false);
  const multi = NETWORKS.length > 1;

  const content = (
    <>
      <Dot tone={status.tone} />
      <span className="type-label text-[13px]">{network.name}</span>
      <span className="sr-only">network, {status.text}</span>
      {multi && <IconChevronDown size={14} className="-mr-0.5 opacity-60" />}
    </>
  );

  const pillClass = clsx("inline-flex h-10 items-center gap-2 rounded-control border border-line bg-surface px-3.5 text-on-surface", className);

  if (!multi) {
    return (
      <span className={pillClass} title={`${network.name} (${network.networkId}): ${status.text}`}>
        {content}
      </span>
    );
  }
  return (
    <>
      <button type="button" className={clsx(pillClass, "hover:bg-current/6")} onClick={() => setOpen(true)} aria-haspopup="dialog">
        {content}
      </button>
      <Sheet open={open} onOpenChange={setOpen} title="Network" description="Balances and history are separate on each network.">
        <ul className="space-y-2">
          {NETWORKS.map((n) => (
            <li key={n.slug}>
              <button
                type="button"
                onClick={() => {
                  setNetwork(n.slug);
                  setOpen(false);
                }}
                className={clsx(
                  "flex w-full items-center gap-4 rounded-field p-4 text-left transition-colors",
                  n.slug === network.slug ? "bg-surface-2 hairline" : "hover:bg-surface-2",
                )}
              >
                <span className="grid size-10 place-items-center rounded-control bg-signal font-mono text-[11px] font-medium text-ink">{n.networkId}</span>
                <span className="min-w-0 flex-1">
                  <span className="type-label block text-[16px]">{n.name}</span>
                  <span className="block text-[13px] text-muted">{n.networkId === "YNET" ? "Main network" : "Test network"}</span>
                </span>
                {n.slug === network.slug && <IconCheck />}
              </button>
            </li>
          ))}
        </ul>
      </Sheet>
    </>
  );
}
