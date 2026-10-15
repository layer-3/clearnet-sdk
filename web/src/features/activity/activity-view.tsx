"use client";

import type { AssetInfo, TransferActivity } from "@yellow-org/clearnet-sdk/gateway";
import Link from "next/link";
import { useMemo, useState } from "react";
import { ActivityRow, ActivitySkeleton } from "@/components/activity-row";
import { RequireWallet } from "@/components/connect-screen";
import { TransferDetailSheet } from "@/components/transfer-detail";
import { Button, ButtonLink } from "@/components/ui/button";
import { IconChevronRight } from "@/components/ui/icons";
import { Amount, StatusPill } from "@/components/ui/misc";
import { useInfo, useMyOperations, useTransfers } from "@/hooks/gateway";
import { dayLabel, shortAddress, timeAgo } from "@/lib/format";
import { STATE_COPY, type JournalEntry } from "@/lib/journal/copy";

export function ActivityView() {
  return <RequireWallet>{(address) => <Activity address={address} />}</RequireWallet>;
}

function Activity({ address }: { address: `0x${string}` }) {
  const info = useInfo();
  const assets = info.data?.assets ?? [];
  return (
    <div className="pt-2 lg:pt-6">
      <h1 className="type-h1">Activity</h1>
      <div className="mt-8 grid gap-6 lg:grid-cols-[minmax(0,1fr)_380px] lg:items-start">
        <History address={address} assets={assets} />
        <Saved assets={assets} />
      </div>
    </div>
  );
}

function History({ address, assets }: { address: `0x${string}`; assets: AssetInfo[] }) {
  const q = useTransfers(address);
  const [selected, setSelected] = useState<TransferActivity | null>(null);
  const items = useMemo(() => q.data?.pages.flatMap((p) => p.items) ?? [], [q.data]);
  const groups = useMemo(() => {
    const out: Array<{ label: string; items: TransferActivity[] }> = [];
    for (const item of items) {
      const label = dayLabel(item.sealedAt);
      const last = out[out.length - 1];
      if (last?.label === label) last.items.push(item);
      else out.push({ label, items: [item] });
    }
    return out;
  }, [items]);

  return (
    <section className="sheet p-3 sm:p-4" aria-labelledby="history-title">
      <div className="px-3 pb-1 pt-3">
        <h2 id="history-title" className="type-title text-[20px]">
          Transfers
        </h2>
      </div>
      {q.isPending ? (
        <ActivitySkeleton rows={6} />
      ) : q.error ? (
        <div className="px-3 py-8">
          <p className="type-title text-[17px]">History isn&apos;t available right now</p>
          <p className="mt-1 text-[15px] text-muted">{"httpStatus" in q.error && q.error.httpStatus === 503 ? "The network is still indexing recent transfers. Try again in a minute." : q.error.message}</p>
          <Button variant="quiet" className="mt-4" onClick={() => void q.refetch()}>
            Try again
          </Button>
        </div>
      ) : items.length === 0 ? (
        <div className="px-3 py-10">
          <p className="type-title text-[17px]">No transfers yet</p>
          <p className="mt-1 text-[15px] text-muted">When you send or receive, it shows up here.</p>
          <ButtonLink href="/receive" variant="quiet" className="mt-4">
            Show my address
          </ButtonLink>
        </div>
      ) : (
        <>
          {groups.map((g) => (
            <div key={g.label} className="mt-3">
              <h3 className="type-label px-3 pb-1 text-[13px] text-muted">{g.label}</h3>
              <ul>
                {g.items.map((item) => (
                  <ActivityRow key={`${item.blockHash}-${item.entryIndex}-${item.direction}`} item={item} assets={assets} onSelect={() => setSelected(item)} />
                ))}
              </ul>
            </div>
          ))}
          {q.hasNextPage && (
            <div className="p-3">
              <Button variant="quiet" className="w-full" onClick={() => void q.fetchNextPage()} disabled={q.isFetchingNextPage}>
                {q.isFetchingNextPage ? "Loading…" : "Load older transfers"}
              </Button>
            </div>
          )}
        </>
      )}
      <TransferDetailSheet item={selected} assets={assets} onClose={() => setSelected(null)} />
    </section>
  );
}

function Saved({ assets }: { assets: AssetInfo[] }) {
  const ops = useMyOperations();
  const [all, setAll] = useState(false);
  if (!ops.length) return null;
  const shown = all ? ops : ops.slice(0, 5);
  return (
    <section className="sheet p-3 sm:p-4" aria-labelledby="saved-title">
      <div className="px-3 pb-2 pt-3">
        <h2 id="saved-title" className="type-title text-[20px]">
          Sent from this device
        </h2>
        <p className="mt-1 text-[14px] text-muted">Every transfer you sign here is saved, so none is lost or sent twice.</p>
      </div>
      <ul>
        {shown.map((op) => (
          <SavedRow key={op.id} op={op} assets={assets} />
        ))}
      </ul>
      {ops.length > 5 && (
        <Button variant="ghost" className="w-full" onClick={() => setAll(!all)}>
          {all ? "Show fewer" : `Show all ${ops.length}`}
        </Button>
      )}
    </section>
  );
}

function SavedRow({ op, assets }: { op: JournalEntry; assets: AssetInfo[] }) {
  const copy = STATE_COPY[op.state];
  return (
    <li>
      <Link href={`/activity/${op.id}`} className="flex items-center gap-3 rounded-field px-3 py-3 transition-colors hover:bg-surface-2">
        <span className="min-w-0 flex-1">
          <span className="type-label block truncate text-[15px]">
            {op.signed.intent.assets.map((leg, i) => {
              const meta = assets.find((a) => a.symbol === leg.asset);
              return (
                <span key={leg.asset}>
                  {i > 0 && " + "}
                  {meta ? <Amount raw={leg.amount} decimals={meta.decimals} maxFraction={4} fractionClass="" /> : <span className="skeleton inline-block h-3.5 w-10 align-middle" />} {leg.asset}
                </span>
              );
            })}
          </span>
          <span className="block truncate text-[13px] text-muted">
            to <span className="font-mono text-[12px]">{shortAddress(op.signed.intent.to)}</span>, {timeAgo(Math.floor(op.createdAt / 1000))}
          </span>
        </span>
        <StatusPill tone={copy.tone}>{copy.label}</StatusPill>
        <IconChevronRight size={18} className="shrink-0 text-muted" />
      </Link>
    </li>
  );
}
