"use client";

import {
  applySubmission,
  blockingOperation,
  isFinalOperationState,
  markSubmitting,
  newOperationRecord,
  observeOperation,
  type TransferIntent,
} from "@yellow-org/clearnet-sdk/gateway";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useRef, useState } from "react";
import { useAccount, useSignTypedData, useSwitchChain } from "wagmi";
import type { JournalEntry } from "@/lib/journal/copy";
import { journal, LockBusyError, withAccountLock } from "@/lib/journal/store";
import { useNetwork } from "@/lib/network-context";
import { isChainMismatch, isUserRejection, rejectionMessage, submissionRejection } from "./errors";

export type SendStage = "switching" | "signing" | "submitting" | "pending" | "sealed" | "rejected" | "unresolved" | "unsent";

export interface SendProgress {
  stage: SendStage;
  record?: JournalEntry;
  message?: string;
}

export type SendOutcome = { ok: true } | { ok: false; error: string };

const POLL_MS = 1_000;
const POLL_FOR_MS = 45_000;

/**
 * Runs one transfer with the CLI's durability protocol:
 * sign -> save -> mark submitting -> POST once -> record outcome -> poll.
 */
export function useSendTransfer() {
  const { network, client } = useNetwork();
  const { address, chainId } = useAccount();
  const { mutateAsync: switchChain } = useSwitchChain();
  const { mutateAsync: signTypedData } = useSignTypedData();
  const queryClient = useQueryClient();
  const [progress, setProgress] = useState<SendProgress | null>(null);
  const cancelled = useRef(false);

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["balance", network.slug] });
    void queryClient.invalidateQueries({ queryKey: ["transfers", network.slug] });
  }, [queryClient, network.slug]);

  const send = useCallback(
    async (intent: TransferIntent): Promise<SendOutcome> => {
      if (!address) return { ok: false, error: "Connect your wallet first." };
      if (address.toLowerCase() !== intent.from.toLowerCase()) {
        return { ok: false, error: "Your wallet switched accounts. Review the transfer again from the account you want to send from." };
      }
      cancelled.current = false;
      try {
        return await withAccountLock(`${network.networkId}:${address.toLowerCase()}`, async () => {
          const open = blockingOperation(await journal.all(), network.networkId, address);
          if (open) return { ok: false, error: "An earlier transfer from this account isn't settled yet. Check it in Activity first." };

          // 1. Wallets like MetaMask only sign when the EIP-712 domain chain is active.
          if (chainId !== network.chainId) {
            setProgress({ stage: "switching" });
            try {
              await switchChain({ chainId: network.chainId });
            } catch (e) {
              if (isUserRejection(e)) {
                setProgress(null);
                return { ok: false, error: `Switch to Yellow ${network.name} in your wallet to sign. Nothing was sent.` };
              }
              // Some wallets can't switch but still sign typed data. Try anyway.
            }
          }

          // 2. Sign the exact intent.
          setProgress({ stage: "signing" });
          let signed;
          try {
            signed = await client.signTransfer(intent, (typedData) => signTypedData({ ...typedData, account: address }));
          } catch (e) {
            setProgress(null);
            if (isUserRejection(e)) return { ok: false, error: "You declined the signature. Nothing was sent." };
            if (isChainMismatch(e)) return { ok: false, error: `Your wallet must be on Yellow ${network.name} to sign. Approve the network switch in your wallet, then try again.` };
            return { ok: false, error: e instanceof Error ? e.message : "The wallet couldn't sign this transfer." };
          }

          // 3. Save before anything leaves this device.
          const record: JournalEntry = { ...newOperationRecord(signed, client.baseUrl), networkSlug: network.slug };
          try {
            await journal.put(record);
          } catch {
            setProgress(null);
            return { ok: false, error: "This browser couldn't save the signed transfer, so nothing was sent. Check that site storage is allowed." };
          }

          // 4. Mark it as possibly submitted BEFORE posting.
          const submitting = markSubmitting(record);
          try {
            await journal.put(submitting);
          } catch {
            setProgress({ stage: "unsent", record, message: "Saved but not sent. Send it from Activity." });
            return { ok: true };
          }

          // 5. Post exactly once.
          setProgress({ stage: "submitting", record: submitting });
          let final: JournalEntry;
          let error: unknown;
          try {
            final = applySubmission(record, submitting, { result: await client.submitTransfer(signed) });
          } catch (e) {
            error = e;
            final = applySubmission(record, submitting, { error: e });
          }
          await journal.put(final).catch(() => undefined);
          refresh();

          if (final.state === "signed_unsubmitted") {
            setProgress({ stage: "unsent", record: final, message: error instanceof Error ? error.message : undefined });
            return { ok: true };
          }
          if (final.state === "rejected") {
            setProgress({ stage: "rejected", record: final, message: submissionRejection(error) ?? rejectionMessage(final.reason) });
            return { ok: true };
          }
          if (final.state === "unresolved" || final.state === "sealed") {
            setProgress({ stage: final.state, record: final });
            return { ok: true };
          }

          // 6. Pending: watch it seal.
          setProgress({ stage: "pending", record: final });
          let current = final;
          const until = Date.now() + POLL_FOR_MS;
          while (!cancelled.current && Date.now() < until && !isFinalOperationState(current.state)) {
            await new Promise((r) => setTimeout(r, POLL_MS));
            try {
              const status = await client.operation(current.id);
              const next = observeOperation(current, status);
              if (!next.error && next.record.state !== current.state) {
                current = next.record;
                await journal.put(current).catch(() => undefined);
              }
            } catch {
              // Keep waiting; the background poller also reconciles.
            }
          }
          refresh();
          setProgress({
            stage: current.state === "sealed" ? "sealed" : current.state === "rejected" ? "rejected" : current.state === "unresolved" ? "unresolved" : "pending",
            record: current,
            ...(current.state === "rejected" && { message: rejectionMessage(current.reason) }),
          });
          return { ok: true };
        });
      } catch (e) {
        setProgress(null);
        if (e instanceof LockBusyError) return { ok: false, error: e.message };
        return { ok: false, error: e instanceof Error ? e.message : "Something went wrong. Nothing was sent." };
      }
    },
    [address, chainId, client, network, refresh, signTypedData, switchChain],
  );

  const reset = useCallback(() => {
    cancelled.current = true;
    setProgress(null);
  }, []);

  return { progress, send, reset };
}
