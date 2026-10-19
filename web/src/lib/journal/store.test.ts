import { beforeEach, describe, expect, it, vi } from "vitest";
import { newOperationRecord, type SignedTransfer } from "@yellow-org/clearnet-sdk/gateway";
import { journal } from "./store";
import type { JournalEntry } from "./copy";

const memory = vi.hoisted(() => ({ rows: new Map<string, unknown>() }));
vi.mock("idb-keyval", () => ({
  createStore: () => ({}),
  entries: async () => Array.from(memory.rows.entries()),
  get: async (key: string) => memory.rows.get(key),
  update: async (key: string, updater: (old: unknown) => unknown) => {
    memory.rows.set(key, updater(memory.rows.get(key)));
  },
}));

const id = `0x${"ab".repeat(32)}` as const;
const signed: SignedTransfer = {
  operationId: id, signature: "0x01", intent: {
    catalogueFingerprint: "test-catalogue", networkId: "YDEV", chainId: 1497646422,
    from: "0x00000000000000000000000000000000000000aa",
    to: "0x00000000000000000000000000000000000000bb",
    nonce: 1, assets: [{ asset: "YELLOW", amount: "1" }], maxFee: "0",
  },
};

function record(state: JournalEntry["state"]): JournalEntry {
  return { ...newOperationRecord(signed, "/api/gw/local"), networkSlug: "local", state };
}

beforeEach(() => memory.rows.clear());

describe("durable journal state transitions", () => {
  it("does not overwrite a sealed record with stale pending or rejected responses", async () => {
    await journal.put(record("pending"));
    await journal.put(record("sealed"));
    await journal.put(record("pending"));
    await journal.put(record("rejected"));
    expect((await journal.get(id))?.state).toBe("sealed");
  });

  it("permits a verified seal to supersede an earlier reported rejection", async () => {
    await journal.put(record("rejected"));
    await journal.put(record("sealed"));
    expect((await journal.get(id))?.state).toBe("sealed");
  });

  it("retains saved nonce evidence when another tab writes a stale rejection", async () => {
    const rejected = record("rejected");
    const evidence = { operationId: `0x${"cd".repeat(32)}`, blockHash: "block", entryHash: "entry", entryIndex: 0, sealedAt: 42, from: rejected.sender, to: rejected.sender, nonce: rejected.nonce, direction: "out", delivery: "sender_sealed", assets: [] };
    await journal.put({ ...rejected, nonceConsumedBy: evidence });
    await journal.put(rejected);
    expect((await journal.get(id))?.nonceConsumedBy).toEqual(evidence);
  });
});
