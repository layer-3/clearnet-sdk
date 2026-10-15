"use client";

import type { AssetInfo, TransferActivity } from "@yellow-org/clearnet-sdk/gateway";
import type { ReactNode } from "react";
import { AssetIcon } from "@/components/ui/asset-icon";
import { CopyButton } from "@/components/ui/copy-button";
import { Amount } from "@/components/ui/misc";
import { Sheet } from "@/components/ui/sheet";
import { dateTime, isIncoming, shortAddress } from "@/lib/format";

export function DetailRow({ label, value, copy, mono }: { label: string; value: ReactNode; copy?: string; mono?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <dt className="shrink-0 text-[14px] text-muted">{label}</dt>
      <dd className={mono ? "flex min-w-0 items-center gap-1 font-mono text-[13px]" : "flex min-w-0 items-center gap-1 text-right text-[15px]"}>
        <span className="truncate">{value}</span>
        {copy && <CopyButton value={copy} label={`Copy ${label.toLowerCase()}`} className="-mr-1 p-1.5" />}
      </dd>
    </div>
  );
}

export function TransferDetailSheet({ item, assets, onClose }: { item: TransferActivity | null; assets: AssetInfo[]; onClose: () => void }) {
  const incoming = item ? isIncoming(item.direction) : false;
  return (
    <Sheet open={!!item} onOpenChange={(o) => !o && onClose()} title={incoming ? "Received" : "Sent"} description={item ? dateTime(item.sealedAt) : undefined}>
      {item && (
        <>
          <ul className="space-y-3 rounded-field bg-surface-2 p-4">
            {item.assets.map((leg) => {
              const meta = assets.find((a) => a.symbol === leg.asset);
              return (
                <li key={leg.asset} className="flex items-center gap-3">
                  <AssetIcon symbol={leg.asset} size={36} />
                  <span className="type-amount flex-1 text-[30px]">
                    {incoming ? "+" : "−"}
                    {meta ? <Amount raw={leg.amount} decimals={meta.decimals} maxFraction={meta.decimals} /> : leg.amount}
                  </span>
                  <span className="type-label text-[15px] text-muted">{leg.asset}</span>
                </li>
              );
            })}
          </ul>
          <dl className="mt-3 divide-y divide-line">
            <DetailRow label="From" value={shortAddress(item.from, 8, 6)} copy={item.from} mono />
            <DetailRow label="To" value={shortAddress(item.to, 8, 6)} copy={item.to} mono />
            <DetailRow label="Block" value={shortAddress(item.blockHash, 10, 8)} copy={item.blockHash} mono />
            <DetailRow label="Entry" value={`#${item.entryIndex}`} />
            <DetailRow label="Sender nonce" value={item.nonce} />
            {item.operationId && <DetailRow label="Operation" value={shortAddress(item.operationId, 10, 8)} copy={item.operationId} mono />}
          </dl>
        </>
      )}
    </Sheet>
  );
}
