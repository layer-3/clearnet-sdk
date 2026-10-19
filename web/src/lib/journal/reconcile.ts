"use client";

import { blockingOperation, reconcileRejectedOperation, unexplainedRejectedOperation } from "@yellow-org/clearnet-sdk/gateway";
import { findNetwork } from "@/config/networks";
import { clientFor } from "@/lib/network-context";
import { journal } from "./store";

/** Caller holds the account lock through reconciliation and any signing. */
export async function reconcileSendOperations(networkId: string, sender: string, nonce: number) {
  let records = await journal.all();
  if (journal.loadError()) throw new Error("This browser couldn't read its saved transfers. Restore access to local storage before sending.");
  if (blockingOperation(records, networkId, sender)) throw new Error("An earlier transfer from this account isn't settled yet. Check it in Activity first.");
  for (;;) {
    const rejected = unexplainedRejectedOperation(records, networkId, sender, nonce);
    if (!rejected) return records;
    const client = clientFor(rejected.gateway, rejected.networkId, findNetwork(rejected.networkSlug)?.fingerprint);
    const updated = await reconcileRejectedOperation(rejected, client);
    await journal.put(updated);
    const saved = await journal.get(rejected.id);
    if (!saved) throw new Error("This browser couldn't read the saved nonce reconciliation. Nothing new was signed.");
    if (saved.state === "sealed") {
      throw new Error("A transfer previously marked rejected actually sealed. Nothing new was signed. Review that payment in Activity before sending again.");
    }
    records = records.map((record) => record.id === saved.id ? saved : record);
  }
}
