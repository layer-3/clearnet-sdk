import { describe, expect, it, vi } from "vitest";
import {
  APIError,
  applySubmission,
  blockingOperation as blockingEntry,
  markSubmitting,
  newOperationRecord,
  observeOperation,
  reconcileRejectedOperation,
  GatewayClient,
  retryDecision,
  staleNonceOperation as staleNonceEntry,
  SubmissionError,
  unexplainedRejectedOperation,
  type OperationRecord,
  type TransferActivity,
} from "../../src/gateway/index.js";

const observe = (r: OperationRecord, s: Parameters<typeof observeOperation>[1]) => {
  const { record, error } = observeOperation(r, s);
  return { entry: record, error };
};

const ID = `0x${"ab".repeat(32)}` as const;
const base: OperationRecord = {
  id: ID,
  networkId: "YDEV",
  chainId: 1497646422,
  gateway: "/api/mock",
  sender: "0x00000000000000000000000000000000000000aA",
  nonce: 3,
  state: "signed_unsubmitted",
  createdAt: 1_000,
  updatedAt: 1_000,
  signed: {
    operationId: ID,
    signature: "0x",
    intent: { catalogueFingerprint: "test-catalogue", networkId: "YDEV", chainId: 1, from: "0x00000000000000000000000000000000000000aA", to: "0x00000000000000000000000000000000000000bb", assets: [], maxFee: "0", nonce: 3 },
  },
};

describe("journal rules", () => {
  it("copies the signed body before journaling it", () => {
    const signed = { ...base.signed, intent: { ...base.signed.intent, assets: [{ asset: "YELLOW", amount: "1000" }] } };
    const record = newOperationRecord(signed, "/api/mock", 1_000);
    signed.intent.assets[0]!.amount = "9999";
    signed.intent.to = "0x00000000000000000000000000000000000000cc";
    expect(record.signed.intent.assets[0]!.amount).toBe("1000");
    expect(record.signed.intent.to).toBe(base.signed.intent.to);
  });

  it("requires a matching receipt to seal", () => {
    expect(observe(base, { operationId: ID, kind: "transfer", status: "sealed" }).error).toBeTruthy();
    const ok = observe(base, { operationId: ID, kind: "transfer", status: "sealed", receipt: { blockHash: "0x1", entryIndex: 0, sealedAt: 1, stateRoot: "0x", operationId: ID } });
    expect(ok.entry.state).toBe("sealed");
  });

  it("never downgrades a sealed transfer", () => {
    const sealed = { ...base, state: "sealed" as const };
    expect(observe(sealed, { operationId: ID, kind: "transfer", status: "rejected", reason: "x" }).entry.state).toBe("sealed");
    expect(observe(sealed, { operationId: ID, kind: "transfer", status: "pending" }).entry.state).toBe("sealed");
  });

  it("marks unknown as unresolved only after submission", () => {
    expect(observe(base, { operationId: ID, kind: "transfer", status: "unknown" }).entry.state).toBe("signed_unsubmitted");
    const sent = { ...base, state: "pending" as const, submittedAt: 5_000 };
    expect(observe(sent, { operationId: ID, kind: "transfer", status: "unknown" }).entry.state).toBe("unresolved");
  });

  it("allows replay only under the CLI conditions", () => {
    const unknown = { operationId: ID, kind: "transfer", status: "unknown" as const };
    expect(retryDecision(base, unknown).allowed).toBe(true);
    const sent = { ...base, state: "unresolved" as const, submittedAt: 10_000 };
    expect(retryDecision(sent, unknown, 60_000).allowed).toBe(false);
    expect(retryDecision(sent, { ...unknown, pendingUntil: 50 }, 60_000).allowed).toBe(true);
    expect(retryDecision(sent, { ...unknown, pendingUntil: 5 }, 60_000).allowed).toBe(false);
    expect(retryDecision(sent, { ...unknown, pendingUntil: 120 }, 60_000).allowed).toBe(false);
  });

  it("blocks new sends while an earlier one is open", () => {
    expect(blockingEntry([base], "YDEV", base.sender.toLowerCase())).toBe(base);
    expect(blockingEntry([{ ...base, state: "sealed" }], "YDEV", base.sender)).toBeUndefined();
    expect(staleNonceEntry([{ ...base, state: "sealed" }], "YDEV", base.sender, 3)).toBeTruthy();
    expect(staleNonceEntry([{ ...base, state: "sealed" }], "YDEV", base.sender, 4)).toBeUndefined();
  });

  it("applies submission outcomes like the CLI", () => {
    const sending = markSubmitting(base, 10);
    expect(sending.state).toBe("unresolved");
    expect(sending.submittedAt).toBe(10);
    const preflight = new SubmissionError({ submitted: false, operationId: ID, outcomeUnknown: false, cause: new Error("x") });
    expect(applySubmission(base, sending, { error: preflight }).state).toBe("signed_unsubmitted");
    const rejected = new SubmissionError({ submitted: true, operationId: ID, outcomeUnknown: false, cause: new APIError({ httpStatus: 422, code: -32003, message: "insufficient YELLOW" }) });
    expect(applySubmission(base, sending, { error: rejected })).toMatchObject({ state: "rejected", reason: "insufficient YELLOW" });
    const unknown = new SubmissionError({ submitted: true, operationId: ID, outcomeUnknown: true, cause: new Error("timeout") });
    expect(applySubmission(base, sending, { error: unknown }).state).toBe("unresolved");
    expect(applySubmission(base, sending, { result: { operationId: ID, status: "pending" } }).state).toBe("pending");
  });

  it("blocks nonce advances past a rejected transfer unless a seal explains that same nonce", () => {
    const rejected = { ...base, state: "rejected" as const, submittedAt: 10 };
    const check = (records: OperationRecord[], nonce = 4) => unexplainedRejectedOperation(records, "YDEV", base.sender.toLowerCase(), nonce);
    expect(check([rejected])).toBe(rejected);
    expect(check([rejected], 3)).toBeUndefined();
    expect(check([{ ...rejected, networkId: "YTES" }])).toBeUndefined();
    expect(check([{ ...rejected, sender: "0x00000000000000000000000000000000000000bb" }])).toBeUndefined();
    const sealed = { ...base, state: "sealed" as const };
    expect(check([rejected, sealed])).toBeUndefined();
    expect(check([rejected, { ...sealed, nonce: 2 }])).toBe(rejected);
    expect(check([rejected, { ...sealed, nonce: 4 }], 5)).toBe(rejected);
    expect(check([rejected, { ...sealed, networkId: "YTES" }])).toBe(rejected);
  });

  it("persists external nonce evidence without changing the rejected authorization", async () => {
    const rejected = { ...base, state: "rejected" as const, reason: "insufficient balance" };
    const client = new GatewayClient({ baseUrl: base.gateway, networkId: base.networkId });
    vi.spyOn(client, "operation").mockResolvedValue({ operationId: ID, kind: "transfer", status: "unknown" });
    const activity: TransferActivity = { operationId: `0x${"cd".repeat(32)}`, from: base.sender, to: base.sender, nonce: 3, direction: "out", delivery: "sender_sealed", blockHash: "block", entryHash: "entry", entryIndex: 0, sealedAt: 42, assets: [] };
    vi.spyOn(client, "sealedTransferAtNonce").mockResolvedValue(activity);
    const updated = await reconcileRejectedOperation(rejected, client, 5_000);
    expect(updated).toMatchObject({ state: "rejected", reason: rejected.reason, nonceConsumedBy: activity, updatedAt: 5_000 });
    expect(updated.signed).toBe(base.signed);
    expect(rejected).not.toHaveProperty("nonceConsumedBy");
    expect(unexplainedRejectedOperation([updated], "YDEV", base.sender, 4)).toBeUndefined();
    expect(unexplainedRejectedOperation([rejected, updated], "YDEV", base.sender, 4)).toBe(rejected);
  });

  it("updates an original payment found sealed and rejects missing or contradictory evidence", async () => {
    const rejected: OperationRecord = { ...base, state: "rejected", reason: "not admitted" };
    const client = new GatewayClient({ baseUrl: base.gateway, networkId: base.networkId });
    const status = vi.spyOn(client, "operation").mockResolvedValue({ operationId: ID, kind: "transfer", status: "sealed", receipt: { operationId: ID, blockHash: "block", entryIndex: 0, sealedAt: 42, stateRoot: "root" } });
    const history = vi.spyOn(client, "sealedTransferAtNonce");
    const sealed = await reconcileRejectedOperation(rejected, client);
    expect(sealed.state).toBe("sealed");
    expect(sealed.reason).toBeUndefined();
    expect(history).not.toHaveBeenCalled();
    status.mockResolvedValue({ operationId: ID, kind: "transfer", status: "rejected", reason: "not admitted" });
    history.mockResolvedValue(undefined);
    await expect(reconcileRejectedOperation(rejected, client)).rejects.toThrow(/No matching outgoing seal/);
    history.mockResolvedValue({ operationId: ID } as TransferActivity);
    await expect(reconcileRejectedOperation(rejected, client)).rejects.toThrow(/disagrees/);
    const other = new GatewayClient({ baseUrl: "/other", networkId: base.networkId });
    await expect(reconcileRejectedOperation(rejected, other)).rejects.toThrow(/original gateway/);
  });

  it("keeps a refused replay unresolved instead of rejecting it", () => {
    const sent = { ...base, state: "unresolved" as const, submittedAt: 9_000 };
    const sending = markSubmitting(sent, 10_000);
    const rejected = new SubmissionError({ submitted: true, operationId: ID, outcomeUnknown: false, cause: new APIError({ httpStatus: 409, code: -32009, message: "conflict" }) });
    expect(applySubmission(sent, sending, { error: rejected })).toMatchObject({ state: "unresolved" });
  });
});
