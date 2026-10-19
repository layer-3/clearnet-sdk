"use client";

import type { TransferActivity } from "@yellow-org/clearnet-sdk/gateway";
import { isFinalOperationState } from "@yellow-org/clearnet-sdk/gateway";
import { clsx } from "clsx";
import Link from "next/link";
import { useState, type ComponentType, type SVGProps } from "react";
import { ActivityRow, ActivitySkeleton } from "@/components/activity-row";
import { RequireWallet } from "@/components/connect-screen";
import { TransferDetailSheet } from "@/components/transfer-detail";
import { AssetIcon } from "@/components/ui/asset-icon";
import { ButtonLink } from "@/components/ui/button";
import { IconAlert, IconChevronRight, IconFund, IconReceive, IconSend, IconWithdraw } from "@/components/ui/icons";
import { Amount, ASSET_NAMES, ErrorPanel, SectionHeader } from "@/components/ui/misc";
import { useMyOperations, usePortfolio, useTransfers } from "@/hooks/gateway";
import { amountParts, formatUnits } from "@/lib/format";
import { STATE_COPY } from "@/lib/journal/copy";
import { ACTIVATION_YELLOW, type Portfolio } from "@/lib/portfolio";

export function AccountView() {
  return <RequireWallet>{(address) => <Account address={address} />}</RequireWallet>;
}

function Account({ address }: { address: `0x${string}` }) {
  const { portfolio, info, balance, error, isLoading } = usePortfolio(address);
  const yellowDecimals = portfolio?.yellow?.asset.decimals ?? 18;

  return (
    <div className="pt-2 lg:pt-6">
      <OpenOperations />
      {error && !portfolio ? (
        <ErrorPanel error={error} onRetry={() => void (info.error ? info.refetch() : balance.refetch())} className="mt-4" />
      ) : (
        <>
          <Hero portfolio={portfolio} loading={isLoading} yellowDecimals={yellowDecimals} />
          <Actions />
          {portfolio && !portfolio.activated && <ActivationNotice portfolio={portfolio} />}
          <div className="mt-8 grid gap-5 lg:mt-12 lg:grid-cols-[1.2fr_1fr] lg:gap-6">
            <Tokens portfolio={portfolio} loading={isLoading} yellowDecimals={yellowDecimals} />
            <Recent address={address} />
          </div>
        </>
      )}
    </div>
  );
}

function Hero({ portfolio, loading, yellowDecimals }: { portfolio: Portfolio | undefined; loading: boolean; yellowDecimals: number }) {
  if (loading || !portfolio) {
    return (
      <div aria-busy="true" aria-label="Loading balance">
        <div className="skeleton h-5 w-32" />
        <div className="skeleton mt-4 h-[clamp(72px,16vw,160px)] w-[min(80%,560px)] !rounded-field" />
      </div>
    );
  }
  const hasTotal = portfolio.totalYellow !== null && portfolio.holdings.some((h) => h.asset.symbol !== "YELLOW" && h.available > 0n);
  const value = hasTotal ? portfolio.totalYellow! : (portfolio.yellow?.available ?? 0n);
  const p = amountParts(value, yellowDecimals, 2);
  return (
    <section aria-labelledby="balance-label">
      <p id="balance-label" className="type-label text-[15px] text-on-canvas-muted">
        {hasTotal ? "Total balance" : "Balance"}
      </p>
      <p className="mt-2 flex flex-wrap items-baseline gap-x-3" aria-describedby="balance-label">
        <span className="type-balance text-hero">
          {p.whole}
          {p.fraction && <span className="text-[0.5em] opacity-60">.{p.fraction}</span>}
        </span>
        <span className="type-title text-[clamp(20px,2.6vw,28px)]">YELLOW</span>
      </p>
    </section>
  );
}

function ActionTile({ href, label, icon: Icon, primary, soon }: { href: string; label: string; icon: ComponentType<SVGProps<SVGSVGElement> & { size?: number }>; primary?: boolean; soon?: boolean }) {
  return (
    <Link href={href} className="group flex flex-col items-center gap-2.5 sm:hidden">
      <span
        className={clsx(
          "grid size-[60px] place-items-center rounded-control",
          primary ? "btn border border-[var(--btn-edge)] bg-primary text-on-primary" : "btn btn-quiet-shadow border border-on-canvas",
          soon && "opacity-55",
        )}
      >
        <Icon size={24} />
      </span>
      <span className="type-label text-[13px]">
        {label}
        {soon && <span className="sr-only"> (coming soon)</span>}
      </span>
    </Link>
  );
}

function Actions() {
  return (
    <nav aria-label="Account actions" className="mt-8">
      <div className="grid grid-cols-4 gap-2">
        <ActionTile href="/send" label="Send" icon={IconSend} primary />
        <ActionTile href="/receive" label="Receive" icon={IconReceive} />
        <ActionTile href="/fund" label="Fund" icon={IconFund} soon />
        <ActionTile href="/withdraw" label="Withdraw" icon={IconWithdraw} soon />
      </div>
      <div className="hidden flex-wrap gap-3 sm:flex">
        <ButtonLink href="/send" size="lg" icon={<IconSend />}>
          Send
        </ButtonLink>
        <ButtonLink href="/receive" size="lg" variant="outline" icon={<IconReceive />}>
          Receive
        </ButtonLink>
        <ButtonLink href="/fund" size="lg" variant="outline" className="opacity-60" icon={<IconFund />}>
          Fund <span className="text-[12px] opacity-80">Soon</span>
        </ButtonLink>
        <ButtonLink href="/withdraw" size="lg" variant="outline" className="opacity-60" icon={<IconWithdraw />}>
          Withdraw <span className="text-[12px] opacity-80">Soon</span>
        </ButtonLink>
      </div>
    </nav>
  );
}

function ActivationNotice({ portfolio }: { portfolio: Portfolio }) {
  const y = portfolio.yellow;
  return (
    <div className="sheet mt-6 flex flex-col gap-3 p-5 sm:flex-row sm:items-center sm:gap-5">
      <span className="grid size-11 shrink-0 place-items-center rounded-full bg-signal text-ink">
        <IconAlert size={20} />
      </span>
      <p className="flex-1 text-[15px] leading-relaxed">
        <strong className="type-label">Hold at least {ACTIVATION_YELLOW} YELLOW to start sending.</strong>{" "}
        <span className="text-muted">
          You have {y ? formatUnits(y.available, y.asset.decimals, 4) : "0"} YELLOW. Network fees are paid in YELLOW too.
        </span>
      </p>
      <ButtonLink href="/receive" variant="quiet">
        Receive YELLOW
      </ButtonLink>
    </div>
  );
}

function Tokens({ portfolio, loading, yellowDecimals }: { portfolio: Portfolio | undefined; loading: boolean; yellowDecimals: number }) {
  return (
    <section className="sheet p-3 sm:p-4" aria-labelledby="tokens-title">
      <div className="px-3 pb-2 pt-3">
        <h2 id="tokens-title" className="type-title text-[20px]">
          Tokens
        </h2>
      </div>
      {loading || !portfolio ? (
        <ActivitySkeleton rows={3} />
      ) : (
        <ul>
          {portfolio.holdings.map((h) => {
            const empty = h.available === 0n;
            return (
              <li key={h.asset.symbol}>
                <Link
                  href={`/send?asset=${encodeURIComponent(h.asset.symbol)}`}
                  className={clsx("flex items-center gap-4 rounded-field px-3 py-3.5 transition-colors hover:bg-surface-2", empty && "opacity-50")}
                  aria-label={`${h.asset.symbol}: ${formatUnits(h.available, h.asset.decimals)}. Send ${h.asset.symbol}`}
                >
                  <AssetIcon symbol={h.asset.symbol} size={44} />
                  <span className="min-w-0 flex-1">
                    <span className="type-label block text-[16.5px]">{h.asset.symbol}</span>
                    <span className="block truncate text-[13.5px] text-muted">
                      {h.asset.symbol === "YELLOW" ? "Yellow" : (ASSET_NAMES[h.asset.symbol] ?? h.asset.symbol)}
                    </span>
                  </span>
                  <span className="text-right">
                    <span className="type-title block text-[19px]">
                      <Amount raw={h.available} decimals={h.asset.decimals} maxFraction={h.asset.decimals > 8 ? 6 : h.asset.decimals} />
                    </span>
                    {h.asset.symbol !== "YELLOW" && h.yellowValue !== null && h.available > 0n && (
                      <span className="num block text-[13px] text-muted">≈ {formatUnits(h.yellowValue, yellowDecimals, 2)} YELLOW</span>
                    )}
                    {h.pendingWithdrawal > 0n && (
                      <span className="num block text-[13px] text-muted">{formatUnits(h.pendingWithdrawal, h.asset.decimals, 4)} withdrawing</span>
                    )}
                  </span>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

function Recent({ address }: { address: `0x${string}` }) {
  const transfers = useTransfers(address);
  const { info } = usePortfolio(address);
  const [selected, setSelected] = useState<TransferActivity | null>(null);
  const items = transfers.data?.pages[0]?.items.slice(0, 5) ?? [];
  const assets = info.data?.assets ?? [];
  return (
    <section className="sheet p-3 sm:p-4" aria-labelledby="recent-title">
      <SectionHeader
        title="Recent activity"
        className="px-3 pb-2 pt-3"
        action={
          <Link href="/activity" className="type-label inline-flex items-center gap-0.5 rounded-control px-2 py-1 text-[14px] text-muted hover:text-on-surface">
            View all <IconChevronRight size={16} />
          </Link>
        }
      />
      {transfers.isPending ? (
        <ActivitySkeleton rows={4} />
      ) : transfers.error ? (
        <p className="px-3 py-6 text-[15px] text-muted">{transfers.error.message}</p>
      ) : items.length === 0 ? (
        <div className="px-3 py-8">
          <p className="type-title text-[17px]">No transfers yet</p>
          <p className="mt-1 text-[15px] text-muted">Share your address to receive your first transfer.</p>
          <ButtonLink href="/receive" variant="quiet" className="mt-4">
            Show my address
          </ButtonLink>
        </div>
      ) : (
        <ul>
          {items.map((item) => (
            <ActivityRow key={`${item.blockHash}-${item.entryIndex}-${item.direction}`} item={item} assets={assets} onSelect={() => setSelected(item)} />
          ))}
        </ul>
      )}
      <TransferDetailSheet item={selected} assets={assets} onClose={() => setSelected(null)} />
    </section>
  );
}

function OpenOperations() {
  const open = useMyOperations().filter((o) => !isFinalOperationState(o.state));
  if (!open.length) return null;
  return (
    <div className="mb-6 space-y-2">
      {open.map((op) => (
        <Link key={op.id} href={`/activity/${op.id}`} className="sheet flex items-center gap-4 p-4 transition-transform active:scale-[0.99]">
          <span className={clsx("grid size-10 shrink-0 place-items-center rounded-full", op.state === "pending" ? "bg-surface-2" : "bg-signal text-ink")}>
            {op.state === "pending" ? <span className="size-2.5 animate-[pulse-dot_1.2s_ease-in-out_infinite] rounded-full bg-current" /> : <IconAlert size={19} />}
          </span>
          <span className="min-w-0 flex-1">
            <span className="type-label block text-[15.5px]">{STATE_COPY[op.state].label}</span>
            <span className="block truncate text-[13.5px] text-muted">{STATE_COPY[op.state].message}</span>
          </span>
          <IconChevronRight className="shrink-0 text-muted" />
        </Link>
      ))}
    </div>
  );
}
