"use client";

import {
  applySubmission,
  isFinalOperationState,
  markSubmitting,
  observeOperation,
  retryDecision,
  type AssetInfo,
} from "@yellow-org/clearnet-sdk/gateway";
import { useQueryClient } from "@tanstack/react-query";
import { clsx } from "clsx";
import { useState } from "react";
import { ButtonLink, Button } from "@/components/ui/button";
import { IconCheck, IconChevronLeft, IconRefresh } from "@/components/ui/icons";
import { Amount, StatusPill } from "@/components/ui/misc";
import { Sheet } from "@/components/ui/sheet";
import { DetailRow } from "@/components/transfer-detail";
import { findNetwork } from "@/config/networks";
import { useInfo, useJournalEntries } from "@/hooks/gateway";
import { dateTime, formatUnits, shortAddress } from "@/lib/format";
import { STATE_COPY, type JournalEntry } from "@/lib/journal/copy";
import { journal } from "@/lib/journal/store";
import { clientFor } from "@/lib/network-context";
import { rejectionMessage, submissionRejection } from "@/features/send/errors";

export function OperationView({ id }: { id: string }) {
  const entries = useJournalEntries();
  const entry = entries.find((e) => e.id === id);
  if (journal.loadError()) {
    return (
      <div className="pt-4 lg:pt-10">
        <BackLink />
        <h1 className="type-h1">Local storage is unavailable</h1>
        <p className="lede mt-3">Transfers are saved in this browser, but it refused to read them. Allow site storage for this page, then reload.</p>
      </div>
    );
  }
  if (!journal.isLoaded() && !entry) return <div className="skeleton mt-10 h-64 !rounded-sheet" />;
  if (!entry) {
    return (
      <div className="pt-4 lg:pt-10">
        <BackLink />
        <h1 className="type-h1">Not on this device</h1>
        <p className="lede mt-3">Transfers are saved in the browser you signed them in. Open this link there, or find it in your recent activity.</p>
        <ButtonLink href="/activity" className="mt-6">
          Go to Activity
        </ButtonLink>
      </div>
    );
  }
  return <Operation entry={entry} />;
}

function BackLink() {
  return (
    <ButtonLink href="/activity" variant="ghost" className="-ml-3 mb-2 !px-3" icon={<IconChevronLeft size={20} />}>
      Activity
    </ButtonLink>
  );
}

function Operation({ entry }: { entry: JournalEntry }) {
  const network = findNetwork(entry.networkSlug);
  const info = useInfo();
  const assets: AssetInfo[] = info.data?.networkId === entry.networkId ? info.data.assets : [];
  const queryClient = useQueryClient();
  const copy = STATE_COPY[entry.state];
  const [checking, setChecking] = useState(false);
  const [note, setNote] = useState<string | null>(null);
  const [retryAllowed, setRetryAllowed] = useState(false);
  const [confirm, setConfirm] = useState(false);
  const [sending, setSending] = useState(false);
  const client = clientFor(entry.gateway, entry.networkId, network?.fingerprint);
  const yellow = assets.find((a) => a.symbol === "YELLOW");

  const check = async () => {
    setChecking(true);
    setNote(null);
    try {
      const status = await client.operation(entry.id);
      const { record, error } = observeOperation(entry, status);
      if (error) setNote(error);
      else await journal.put(record);
      const decision = retryDecision(error ? entry : record, status);
      setRetryAllowed(decision.allowed);
      if (!decision.allowed && !isFinalOperationState(record.state)) setNote(decision.reason);
      if (record.state === "sealed") void queryClient.invalidateQueries({ queryKey: ["balance"] });
    } catch (e) {
      setNote(`Couldn't reach the network. Your saved status is unchanged. ${e instanceof Error ? e.message : ""}`);
    } finally {
      setChecking(false);
    }
  };

  const replay = async () => {
    setSending(true);
    setConfirm(false);
    const submitting = markSubmitting(entry);
    try {
      await journal.put(submitting);
    } catch {
      setNote("This browser couldn't save the update, so nothing was sent.");
      setSending(false);
      return;
    }
    let final: JournalEntry;
    try {
      final = applySubmission(entry, submitting, { result: await client.submitTransfer(entry.signed) });
    } catch (e) {
      final = applySubmission(entry, submitting, { error: e });
      const rejection = submissionRejection(e);
      setNote(rejection ?? (e instanceof Error ? e.message : null));
    }
    try {
      await journal.put(final);
    } catch {
      setNote("The outcome couldn't be saved on this device. Check Activity before sending again.");
    }
    setRetryAllowed(false);
    setSending(false);
    void queryClient.invalidateQueries({ queryKey: ["balance"] });
  };

  const steps = [
    { label: "Signed", at: entry.createdAt, done: true },
    { label: "Sent to the network", at: entry.submittedAt, done: entry.submittedAt !== undefined },
    {
      label: entry.state === "rejected" ? "Rejected" : "Sealed",
      at: entry.receipt ? entry.receipt.sealedAt * 1000 : entry.state === "rejected" ? entry.updatedAt : undefined,
      done: isFinalOperationState(entry.state),
      bad: entry.state === "rejected",
    },
  ];

  return (
    <div className="pt-2 lg:pt-6">
      <BackLink />
      <div className="flex flex-wrap items-center gap-3">
        <StatusPill tone="neutral">{network?.name ?? entry.networkId}</StatusPill>
      </div>
      <h1 className="type-h1 mt-3">{copy.label}</h1>
      <p className="lede mt-3" aria-live="polite">
        {entry.state === "rejected" && entry.reason ? rejectionMessage(entry.reason) : copy.message}
      </p>

      <div className="mt-8 grid gap-5 lg:grid-cols-[minmax(0,1fr)_380px] lg:items-start">
        <section className="sheet p-5 sm:p-7">
          <ul className="space-y-1">
            {entry.signed.intent.assets.map((leg) => {
              const meta = assets.find((a) => a.symbol === leg.asset);
              return (
                <li key={leg.asset} className="type-amount">
                  {meta ? <Amount raw={leg.amount} decimals={meta.decimals} maxFraction={meta.decimals} /> : <span className="skeleton inline-block h-10 w-32 align-middle" />} <span className="type-title text-[0.4em]">{leg.asset}</span>
                </li>
              );
            })}
          </ul>
          <dl className="mt-4 divide-y divide-line">
            <DetailRow label="To" value={shortAddress(entry.signed.intent.to, 8, 6)} copy={entry.signed.intent.to} mono />
            <DetailRow label="From" value={shortAddress(entry.sender, 8, 6)} copy={entry.sender} mono />
            <DetailRow label="Fee cap" value={yellow ? `${formatUnits(entry.signed.intent.maxFee, yellow.decimals, 8)} YELLOW` : `${entry.signed.intent.maxFee} base units`} />
            <DetailRow label="Account nonce" value={entry.nonce} />
            <DetailRow label="Operation" value={shortAddress(entry.id, 10, 8)} copy={entry.id} mono />
            {entry.receipt && <DetailRow label="Block" value={shortAddress(entry.receipt.blockHash, 10, 8)} copy={entry.receipt.blockHash} mono />}
            {entry.receipt && <DetailRow label="Entry" value={`#${entry.receipt.entryIndex}`} />}
          </dl>
        </section>

        <aside className="space-y-5">
          <section className="sheet p-5 sm:p-6" aria-label="Progress">
            <ol className="relative space-y-6">
              {steps.map((s, i) => (
                <li key={s.label} className="relative flex gap-4">
                  {i < steps.length - 1 && <span className={clsx("absolute left-[13px] top-8 h-[calc(100%-4px)] w-0.5", steps[i + 1]!.done ? "bg-on-surface" : "bg-line")} aria-hidden />}
                  <span
                    className={clsx(
                      "grid size-7 shrink-0 place-items-center rounded-full",
                      s.done ? (s.bad ? "bg-flare text-white" : "bg-on-surface text-surface") : "shadow-[inset_0_0_0_2px_var(--line)]",
                    )}
                    aria-hidden
                  >
                    {s.done && <IconCheck size={15} strokeWidth={3} />}
                  </span>
                  <span>
                    <span className="type-label block text-[15.5px]">{s.label}</span>
                    <span className="block text-[13.5px] text-muted">{s.at ? dateTime(Math.floor(s.at / 1000)) : s.done ? "" : "Waiting"}</span>
                  </span>
                </li>
              ))}
            </ol>
          </section>

          {!isFinalOperationState(entry.state) && (
            <section className="sheet space-y-3 p-5 sm:p-6">
              <Button variant="quiet" className="w-full" onClick={check} disabled={checking || sending} icon={<IconRefresh size={18} />}>
                {checking ? "Checking…" : "Check status now"}
              </Button>
              {retryAllowed && (
                <Button className="w-full" onClick={() => setConfirm(true)} disabled={sending}>
                  {sending ? "Sending…" : "Send saved transfer"}
                </Button>
              )}
              {note && <p className="text-[14px] leading-relaxed text-muted" aria-live="polite">{note}</p>}
              <p className="text-[13px] leading-relaxed text-muted">
                We check automatically every few seconds. Don&apos;t sign this transfer again as a new one: if both went through, you&apos;d pay twice.
              </p>
            </section>
          )}
        </aside>
      </div>

      <Sheet open={confirm} onOpenChange={setConfirm} title="Send saved transfer?" description="This sends the exact transfer you already signed, with the same nonce, so it can only go through once.">
        <p className="text-[15px] leading-relaxed text-muted">
          The network reports it never sealed this transfer and its window has closed. Sending it again is safe: if it somehow went through, the network rejects the duplicate.
        </p>
        <div className="mt-6 grid gap-2 sm:grid-cols-2">
          <Button variant="quiet" size="lg" onClick={() => setConfirm(false)}>
            Cancel
          </Button>
          <Button size="lg" onClick={replay}>
            Send it
          </Button>
        </div>
      </Sheet>
    </div>
  );
}
