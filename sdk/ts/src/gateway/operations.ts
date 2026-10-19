// Operation journal rules, ported from the Yellow CLI (internal/yellow
// operation.go and transfer.go). They keep a signed authorization from being
// lost, or re-signed at a fresh nonce, while its outcome is uncertain.
//
// Storage is the caller's job (a file, IndexedDB, ...). The protocol is:
//   1. sign -> persist as "signed_unsubmitted"
//   2. persist as "unresolved" with submittedAt BEFORE the POST
//   3. submitTransfer once -> applySubmission
//   4. poll operation() -> observeOperation until sealed or rejected

import { SubmissionError } from "./errors.js";
import type { GatewayClient } from "./client.js";
import type { OperationStatus, Receipt, SignedTransfer, SubmitResult, TransferActivity } from "./types.js";

export type OperationState = "signed_unsubmitted" | "pending" | "unresolved" | "sealed" | "rejected";

export interface OperationRecord {
  id: `0x${string}`;
  networkId: string;
  chainId: number;
  /** Base URL of the gateway the authorization was (or will be) sent to. */
  gateway: string;
  sender: `0x${string}`;
  nonce: number;
  state: OperationState;
  /** Unix milliseconds. */
  createdAt: number;
  updatedAt: number;
  submittedAt?: number;
  receipt?: Receipt;
  reason?: string;
  /** A different sender-sealed transfer proven by the original gateway's history at this nonce. */
  nonceConsumedBy?: TransferActivity;
  signed: SignedTransfer;
}

export function isFinalOperationState(state: OperationState): boolean {
  return state === "sealed" || state === "rejected";
}

/** Creates the record to persist right after signing and before any submission. */
export function newOperationRecord(signed: SignedTransfer, gateway: string, now = Date.now()): OperationRecord {
  return {
    id: signed.operationId,
    networkId: signed.intent.networkId,
    chainId: signed.intent.chainId,
    gateway,
    sender: signed.intent.from,
    nonce: signed.intent.nonce,
    state: "signed_unsubmitted",
    createdAt: now,
    updatedAt: now,
    // Journal the exact authorization at this moment, not a mutable caller
    // object that might be edited after the operation ID was derived.
    signed: { ...signed, intent: { ...signed.intent, assets: signed.intent.assets.map((a) => ({ ...a })) } },
  };
}

/** The record to persist immediately BEFORE posting, so a crash can't hide a submission. */
export function markSubmitting<T extends OperationRecord>(record: T, now = Date.now()): T {
  return { ...record, state: "unresolved", submittedAt: now, updatedAt: now };
}

/**
 * Applies the outcome of one submitTransfer call. `previous` is the record as
 * it was before markSubmitting, restored when the failure was local preflight.
 * A definite rejection only ends a *first* submission: a refused replay of an
 * authorization that was already sent proves nothing about the original
 * outcome, so it must stay unresolved.
 */
export function applySubmission<T extends OperationRecord>(
  previous: T,
  submitting: T,
  outcome: { result: SubmitResult } | { error: unknown },
  now = Date.now(),
): T {
  if ("result" in outcome) {
    const next: T = { ...submitting, state: outcome.result.status, updatedAt: now };
    if (outcome.result.receipt) next.receipt = outcome.result.receipt;
    return next;
  }
  const e = outcome.error;
  if (e instanceof SubmissionError && !e.submitted) return { ...previous, updatedAt: now };
  if (e instanceof SubmissionError && e.definiteRejection && previous.submittedAt === undefined) {
    return { ...submitting, state: "rejected", reason: e.definiteRejection.message, updatedAt: now };
  }
  return { ...submitting, state: "unresolved", updatedAt: now };
}

/** Applies a gateway status report. Returns an error when the report isn't trustworthy. */
export function observeOperation<T extends OperationRecord>(record: T, status: OperationStatus, now = Date.now()): { record: T; error?: string } {
  const next: T = { ...record, updatedAt: now };
  switch (status.status) {
    case "sealed":
      if (!status.receipt?.blockHash || status.receipt.operationId !== record.id) {
        return { record, error: "The network reported a seal without a matching receipt, so the outcome is still unknown." };
      }
      next.state = "sealed";
      next.receipt = status.receipt;
      break;
    case "rejected":
      if (!status.reason) return { record, error: "The network reported a rejection without a reason, so the outcome is still unknown." };
      if (record.state !== "sealed") {
        next.state = "rejected";
        next.reason = status.reason;
      }
      break;
    case "pending":
      if (!isFinalOperationState(record.state)) next.state = "pending";
      break;
    case "unknown":
      if (record.submittedAt !== undefined && !isFinalOperationState(record.state)) next.state = "unresolved";
      break;
  }
  return { record: next };
}

/**
 * Whether the exact saved authorization may be replayed. Allowed only if it
 * was never submitted, or the original gateway reports unknown with an
 * admission window (pendingUntil) that ended after our submission.
 */
export function retryDecision(record: OperationRecord, status: OperationStatus, nowMs = Date.now()): { allowed: boolean; reason: string } {
  if (isFinalOperationState(record.state)) return { allowed: false, reason: `This transfer is already ${record.state}.` };
  const nowSec = Math.floor(nowMs / 1000);
  const neverSubmitted = record.submittedAt === undefined && record.state === "signed_unsubmitted" && status.status === "unknown";
  const until = status.pendingUntil ?? 0;
  const windowEnded =
    status.status === "unknown" &&
    until > 0 &&
    until <= nowSec &&
    record.submittedAt !== undefined &&
    until >= Math.floor(record.submittedAt / 1000);
  if (neverSubmitted || windowEnded) return { allowed: true, reason: "" };
  return {
    allowed: false,
    reason: "The original gateway hasn't provided an ended routing window for this authorization, so replay remains blocked.",
  };
}

/** An earlier transfer from this sender whose outcome must be settled before signing a new one. */
export function blockingOperation<T extends OperationRecord>(records: readonly T[], networkId: string, sender: string): T | undefined {
  return records.find((r) => r.networkId === networkId && r.sender.toLowerCase() === sender.toLowerCase() && !isFinalOperationState(r.state));
}

/** A sealed transfer whose nonce the gateway's live index hasn't moved past yet. */
export function staleNonceOperation<T extends OperationRecord>(records: readonly T[], networkId: string, sender: string, nonce: number): T | undefined {
  return records.find(
    (r) => r.networkId === networkId && r.sender.toLowerCase() === sender.toLowerCase() && r.state === "sealed" && r.nonce >= nonce,
  );
}

/** A rejected authorization whose nonce was consumed without a locally known seal. */
export function unexplainedRejectedOperation<T extends OperationRecord>(records: readonly T[], networkId: string, sender: string, nonce: number): T | undefined {
  const scoped = records.filter((r) => r.networkId === networkId && r.sender.toLowerCase() === sender.toLowerCase());
  const sealed = new Set(scoped.filter((r) => r.state === "sealed").map((r) => r.nonce));
  return scoped.find((r) => r.state === "rejected" && r.nonce < nonce && !sealed.has(r.nonce) && !r.nonceConsumedBy);
}

/** Reconciles a rejected authorization against its original gateway. Persist the
 * returned record before allowing a new signature. If it becomes sealed, stop
 * the replacement send and let the user review the actual payment outcome. */
export async function reconcileRejectedOperation<T extends OperationRecord>(record: T, client: GatewayClient, now = Date.now()): Promise<T> {
  if (record.state !== "rejected" || record.nonceConsumedBy) return record;
  if (client.baseUrl !== record.gateway || client.networkId !== record.networkId) {
    throw new Error("Nonce reconciliation must use the operation's original gateway and network.");
  }
  const status = await client.operation(record.id);
  if (status.status === "sealed") {
    const observed = observeOperation(record, status, now);
    if (observed.error) throw new Error(observed.error);
    delete observed.record.reason;
    return observed.record;
  }
  const activity = await client.sealedTransferAtNonce(record.sender, record.nonce);
  if (!activity) throw new Error("No matching outgoing seal is available in the original gateway's recent history.");
  if (activity.operationId === record.id) {
    throw new Error("History reports this payment sealed but its operation status disagrees. The original gateway must reconcile its index.");
  }
  return { ...record, nonceConsumedBy: activity, updatedAt: now };
}
