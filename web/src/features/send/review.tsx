"use client";

import { TRANSFER_OPERATION_DOMAIN, type AssetInfo, type TransferIntent } from "@yellow-org/clearnet-sdk/gateway";
import { AssetIcon } from "@/components/ui/asset-icon";
import { Identicon } from "@/components/ui/identicon";
import { Amount } from "@/components/ui/misc";
import { DetailRow } from "@/components/transfer-detail";
import type { YellowNetwork } from "@/config/networks";
import { formatUnits, groupedHex } from "@/lib/format";
import { YELLOW } from "@/lib/portfolio";

export function Review({ intent, assets, network }: { intent: TransferIntent; assets: AssetInfo[]; network: YellowNetwork }) {
  const yellow = assets.find((a) => a.symbol === YELLOW);
  return (
    <div>
      <p className="type-label text-[15px] text-muted">You&apos;re sending</p>
      <ul className="mt-3 space-y-2">
        {intent.assets.map((leg) => {
          const meta = assets.find((a) => a.symbol === leg.asset);
          return (
            <li key={leg.asset} className="flex items-center gap-3">
              <AssetIcon symbol={leg.asset} size={40} />
              <span className="type-amount min-w-0 flex-1 truncate">
                {meta ? <Amount raw={leg.amount} decimals={meta.decimals} maxFraction={meta.decimals} /> : leg.amount}
              </span>
              <span className="type-title text-[20px]">{leg.asset}</span>
            </li>
          );
        })}
      </ul>

      <div className="mt-6 rounded-field bg-surface-2 p-4">
        <p className="type-label text-[14px] text-muted">To</p>
        <div className="mt-2 flex items-center gap-3">
          <Identicon address={intent.to} size={40} />
          <p className="min-w-0 break-all font-mono text-[15px] leading-6" aria-label={intent.to}>
            {groupedHex(intent.to).map((g, i) => (
              <span key={i} className={i % 2 ? "text-muted" : undefined}>
                {g}
              </span>
            ))}
          </p>
        </div>
      </div>

      <dl className="mt-4 divide-y divide-line">
        <DetailRow label="Most you'll pay in fees" value={yellow ? `${formatUnits(intent.maxFee, yellow.decimals, 8)} YELLOW` : intent.maxFee} />
        <DetailRow label="Network" value={`${network.name} (${network.networkId})`} />
      </dl>

      <details className="group mt-2 rounded-field px-1">
        <summary className="type-label flex cursor-pointer list-none items-center justify-between py-3 text-[14px] text-muted">
          What you&apos;re signing
          <span className="transition-transform group-open:rotate-45" aria-hidden>
            +
          </span>
        </summary>
        <dl className="divide-y divide-line pb-2">
          <DetailRow label="From" value={intent.from} mono />
          <DetailRow label="Account nonce" value={intent.nonce} />
          {intent.assets.map((leg) => (
            <DetailRow key={leg.asset} label={`${leg.asset} base units`} value={leg.amount} mono />
          ))}
          <DetailRow label="Fee cap base units" value={intent.maxFee} mono />
          <DetailRow label="Signing domain" value={`Clearnet v1, chain ${intent.chainId}`} />
          <DetailRow label="Operation ID scheme" value={TRANSFER_OPERATION_DOMAIN} mono />
        </dl>
        <p className="pb-3 text-[13px] leading-relaxed text-muted">
          Your wallet shows this as a typed-data signature request (EIP-712). It authorizes this exact transfer once and can&apos;t be reused for another.
        </p>
      </details>
    </div>
  );
}
