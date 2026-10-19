"use client";

import { isFinalOperationState, observeOperation } from "@yellow-org/clearnet-sdk/gateway";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { findNetwork } from "@/config/networks";
import { journal } from "@/lib/journal/store";
import { clientFor } from "@/lib/network-context";
import { useJournalEntries } from "./gateway";

/**
 * Background reconciliation, like `yellow operation show`: every open saved
 * transfer is checked against the gateway it was sent to until it is final.
 */
export function useOperationPoller() {
  const entries = useJournalEntries();
  const queryClient = useQueryClient();
  const open = entries.filter((e) => !isFinalOperationState(e.state) && e.submittedAt !== undefined);
  const key = open.map((e) => `${e.id}:${e.state}`).join(",");

  useEffect(() => {
    if (!key) return;
    let stopped = false;
    let running = false;
    const controller = new AbortController();
    const tick = async () => {
      // A slow gateway must not stack ticks on top of each other.
      if (running) return;
      running = true;
      try {
        for (const entry of await journal.all()) {
          if (stopped || isFinalOperationState(entry.state) || entry.submittedAt === undefined) continue;
          try {
            const fingerprint = findNetwork(entry.networkSlug)?.fingerprint;
            const status = await clientFor(entry.gateway, entry.networkId, fingerprint).operation(entry.id, controller.signal);
            const { record, error } = observeOperation(entry, status);
            if (!error && (record.state !== entry.state || record.receipt !== entry.receipt)) {
              await journal.put(record);
              if (record.state === "sealed") void queryClient.invalidateQueries({ queryKey: ["balance"] });
              if (record.state === "sealed") void queryClient.invalidateQueries({ queryKey: ["transfers"] });
            }
          } catch {
            // Unreachable gateway: keep the saved state and try again next tick.
          }
        }
      } finally {
        running = false;
      }
    };
    void tick();
    const id = window.setInterval(tick, 3_000);
    return () => {
      stopped = true;
      controller.abort();
      window.clearInterval(id);
    };
  }, [key, queryClient]);
}
