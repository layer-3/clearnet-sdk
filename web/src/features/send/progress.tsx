"use client";

import type { AssetInfo } from "@yellow-org/clearnet-sdk/gateway";
import { motion, useReducedMotion } from "motion/react";
import { Clearing, type ClearingStage } from "@/components/clearing";
import { Button, ButtonLink } from "@/components/ui/button";
import { Amount } from "@/components/ui/misc";
import { DetailRow } from "@/components/transfer-detail";
import { STATE_COPY } from "@/lib/journal/copy";
import { formatUnits, shortAddress } from "@/lib/format";
import type { SendProgress } from "./use-send";

const VISUAL: Record<SendProgress["stage"], ClearingStage> = {
  switching: "signing",
  signing: "signing",
  submitting: "cosigning",
  pending: "cosigning",
  sealed: "sealed",
  rejected: "rejected",
  unresolved: "cosigning",
  unsent: "idle",
};

export function Progress({ progress, assets, networkName, legs, onReset, onEdit }: { progress: SendProgress; assets: AssetInfo[]; networkName: string; legs: string[]; onReset: () => void; onEdit: () => void }) {
  const reduce = useReducedMotion();
  const { stage, record } = progress;
  const done = stage === "sealed" || stage === "rejected" || stage === "unresolved" || stage === "unsent";
  const title: Record<SendProgress["stage"], string> = {
    switching: `Switch to Yellow ${networkName}`,
    signing: "Approve in your wallet",
    submitting: "Sending",
    pending: "Sealing",
    sealed: "Sent",
    rejected: "Not sent",
    unresolved: "Checking",
    unsent: "Saved, not sent",
  };
  const body: Record<SendProgress["stage"], string> = {
    switching: "Your wallet will ask to add or switch to the Yellow network. It's needed to sign Yellow transfers.",
    signing: "Check the amounts and the recipient in your wallet, then sign. On a phone, open your wallet app.",
    submitting: "Your signed transfer is on its way to validators near your account.",
    pending: "Validators have it and are sealing the block. This usually takes a moment.",
    sealed: "Sealed by the network. The transfer is final.",
    rejected: progress.message ?? "The network rejected this transfer. No funds moved.",
    unresolved: STATE_COPY.unresolved.message,
    unsent: progress.message ? `${STATE_COPY.signed_unsubmitted.message} (${progress.message})` : STATE_COPY.signed_unsubmitted.message,
  };

  return (
    <div className="grid items-center gap-6 lg:grid-cols-[1fr_1fr] lg:gap-12">
      <div className="order-2 lg:order-1" aria-live="polite">
        <h1 className="type-h1 !text-[clamp(36px,5vw,52px)]">
          {stage === "sealed" && !reduce ? (
            <motion.span className="inline-block" initial={{ fontVariationSettings: '"wdth" 125', opacity: 0.2 }} animate={{ fontVariationSettings: '"wdth" 82', opacity: 1 }} transition={{ duration: 0.6, ease: [0.2, 0.9, 0.1, 1] }}>
              {title[stage]}
            </motion.span>
          ) : (
            title[stage]
          )}
        </h1>
        <p className="lede mt-3">{body[stage]}</p>

        {record && (stage === "sealed" || stage === "pending" || stage === "unresolved") && (
          <div className="sheet mt-6 p-5">
            <ul className="space-y-1">
              {record.signed.intent.assets.map((leg) => {
                const meta = assets.find((a) => a.symbol === leg.asset);
                return (
                  <li key={leg.asset} className="type-title text-[24px]">
                    {meta ? <Amount raw={leg.amount} decimals={meta.decimals} maxFraction={meta.decimals} /> : leg.amount} {leg.asset}
                  </li>
                );
              })}
            </ul>
            <dl className="mt-2 divide-y divide-line">
              <DetailRow label="To" value={shortAddress(record.signed.intent.to, 8, 6)} copy={record.signed.intent.to} mono />
              {record.receipt && <DetailRow label="Block" value={shortAddress(record.receipt.blockHash, 10, 8)} copy={record.receipt.blockHash} mono />}
              {assets.find((a) => a.symbol === "YELLOW") && (
                <DetailRow label="Fee cap" value={`${formatUnits(record.signed.intent.maxFee, assets.find((a) => a.symbol === "YELLOW")!.decimals, 8)} YELLOW`} />
              )}
            </dl>
          </div>
        )}

        {done && (
          <div className="mt-8 flex flex-wrap gap-3">
            {stage === "sealed" && (
              <>
                <ButtonLink href="/" size="lg">
                  Done
                </ButtonLink>
                <Button size="lg" variant="outline" onClick={onReset}>
                  Send another
                </Button>
              </>
            )}
            {stage === "rejected" && (
              <>
                <Button size="lg" onClick={onEdit}>
                  Edit and try again
                </Button>
                <ButtonLink href="/" size="lg" variant="outline">
                  Back to account
                </ButtonLink>
              </>
            )}
            {(stage === "unresolved" || stage === "unsent") && record && (
              <ButtonLink href={`/activity/${record.id}`} size="lg">
                View transfer status
              </ButtonLink>
            )}
            {record && stage !== "unresolved" && stage !== "unsent" && (
              <ButtonLink href={`/activity/${record.id}`} size="lg" variant="ghost">
                Details
              </ButtonLink>
            )}
          </div>
        )}
        {stage === "pending" && record && (
          <p className="mt-6 text-[14px] text-on-canvas-muted">
            You can leave this page. We&apos;ll keep checking, and you&apos;ll find it in{" "}
            <a className="underline underline-offset-2" href={`/activity/${record.id}`}>
              Activity
            </a>
            .
          </p>
        )}
      </div>
      <div className="order-1 mx-auto w-full max-w-[300px] sm:max-w-[380px] lg:order-2 lg:max-w-[500px]">
        <Clearing stage={VISUAL[stage]} legs={legs} />
      </div>
    </div>
  );
}
