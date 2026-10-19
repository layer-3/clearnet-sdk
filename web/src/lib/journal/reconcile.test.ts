import { beforeEach, describe, expect, it, vi } from "vitest";
import { GatewayClient, newOperationRecord, type SignedTransfer, type TransferActivity } from "@yellow-org/clearnet-sdk/gateway";
import type { JournalEntry } from "./copy";
import { reconcileSendOperations } from "./reconcile";

const mocks = vi.hoisted(() => ({
  rows: new Map<string, unknown>(),
  clientFor: vi.fn(),
  put: vi.fn(),
}));
vi.mock("@/lib/network-context", () => ({ clientFor: mocks.clientFor }));
vi.mock("./store", () => ({ journal: {
  all: async () => Array.from(mocks.rows.values()),
  loadError: () => undefined,
  get: async (id: string) => mocks.rows.get(id),
  put: mocks.put,
} }));

const signed: SignedTransfer = {
  operationId: `0x${"ab".repeat(32)}`, signature: "0x01",
  intent: { catalogueFingerprint: "test", networkId: "YDEV", chainId: 1497646422,
    from: "0x00000000000000000000000000000000000000aa", to: "0x00000000000000000000000000000000000000bb",
    nonce: 6, assets: [{ asset: "YELLOW", amount: "1" }], maxFee: "0" },
};
const activity: TransferActivity = {
  operationId: `0x${"cd".repeat(32)}`, from: signed.intent.from, to: signed.intent.to,
  nonce: 6, direction: "out", delivery: "sender_sealed", blockHash: "block", entryHash: "entry", entryIndex: 0, sealedAt: 42, assets: [],
};
let client: GatewayClient;
let rejected: JournalEntry;

beforeEach(() => {
  mocks.rows.clear();
  mocks.put.mockReset();
  mocks.put.mockImplementation(async (entry: JournalEntry) => mocks.rows.set(entry.id, entry));
  rejected = { ...newOperationRecord(signed, "/api/gw/local"), networkSlug: "local", state: "rejected", reason: "insufficient balance" };
  mocks.rows.set(rejected.id, rejected);
  client = new GatewayClient({ baseUrl: rejected.gateway, networkId: rejected.networkId });
  vi.spyOn(client, "operation").mockResolvedValue({ operationId: rejected.id, kind: "transfer", status: "unknown" });
  vi.spyOn(client, "sealedTransferAtNonce").mockResolvedValue(activity);
  mocks.clientFor.mockReturnValue(client);
});

describe("automatic send reconciliation", () => {
  it("saves external seal evidence before returning permission to continue", async () => {
    const records = await reconcileSendOperations("YDEV", rejected.sender, 7);
    expect(records[0]?.nonceConsumedBy).toEqual(activity);
    expect(mocks.put).toHaveBeenCalledOnce();
    expect(mocks.clientFor).toHaveBeenCalledWith(rejected.gateway, rejected.networkId, undefined);
    await reconcileSendOperations("YDEV", rejected.sender, 8);
    expect(client.sealedTransferAtNonce).toHaveBeenCalledOnce();
  });

  it("persists an original seal but stops the replacement send for review", async () => {
    vi.mocked(client.operation).mockResolvedValue({ operationId: rejected.id, kind: "transfer", status: "sealed", receipt: {
      operationId: rejected.id, blockHash: "block", stateRoot: "root", entryIndex: 0, sealedAt: 42,
    } });
    await expect(reconcileSendOperations("YDEV", rejected.sender, 7)).rejects.toThrow(/previously marked rejected actually sealed/);
    expect((mocks.rows.get(rejected.id) as JournalEntry).state).toBe("sealed");
    expect(client.sealedTransferAtNonce).not.toHaveBeenCalled();
  });

  it("never clears the block without evidence or a successful durable write", async () => {
    vi.mocked(client.sealedTransferAtNonce).mockResolvedValue(undefined);
    await expect(reconcileSendOperations("YDEV", rejected.sender, 7)).rejects.toThrow(/No matching outgoing seal/);
    expect(mocks.put).not.toHaveBeenCalled();
    vi.mocked(client.sealedTransferAtNonce).mockResolvedValue(activity);
    mocks.put.mockRejectedValue(new Error("storage unavailable"));
    await expect(reconcileSendOperations("YDEV", rejected.sender, 7)).rejects.toThrow(/storage unavailable/);
    expect(mocks.rows.get(rejected.id)).toBe(rejected);
  });

  it("leaves other accounts, networks, and nonadvanced nonces alone", async () => {
    await reconcileSendOperations("YTES", rejected.sender, 7);
    await reconcileSendOperations("YDEV", signed.intent.to, 7);
    await reconcileSendOperations("YDEV", rejected.sender, 6);
    expect(client.operation).not.toHaveBeenCalled();
    expect(mocks.put).not.toHaveBeenCalled();
  });

  it("keeps pending and unresolved operations blocked without looking up history", async () => {
    for (const state of ["pending", "unresolved"] as const) {
      mocks.rows.set(rejected.id, { ...rejected, state });
      await expect(reconcileSendOperations("YDEV", rejected.sender, 7)).rejects.toThrow(/isn't settled yet/);
    }
    expect(client.operation).not.toHaveBeenCalled();
  });
});
