import { describe, expect, it } from "vitest";
import { GatewayClient, networkChainId, VIRTUAL_ROUTER, type TransferActivity, type TransferPage } from "../../src/gateway/index.js";

const sender = "0x00000000000000000000000000000000000000aa";
const entry: TransferActivity = {
  operationId: `0x${"ab".repeat(32)}`, from: `yellow://ynet/user/${sender}`, to: sender,
  nonce: 7, direction: "out", delivery: "sender_sealed", assets: [],
  blockHash: `0x${"cd".repeat(32)}`, entryHash: `0x${"ef".repeat(32)}`, entryIndex: 2, sealedAt: 42,
};

function fixture(page: (cursor: string) => TransferPage) {
  let calls = 0;
  const client = new GatewayClient({
    baseUrl: "/api/mock", networkId: "YDEV",
    fetch: async (input, init) => {
      expect(init?.method).toBe("GET");
      const url = new URL(String(input), "https://test.example");
      if (url.pathname.endsWith("/info")) {
        return Response.json({ networkId: "YDEV", chainId: networkChainId("YDEV"), virtualRouter: VIRTUAL_ROUTER, assets: [], capabilities: ["recent-transfers-v1"] });
      }
      calls++;
      expect(url.searchParams.get("limit")).toBe("100");
      return Response.json(page(url.searchParams.get("cursor") ?? ""));
    },
  });
  return { client, calls: () => calls };
}

describe("sealed transfer nonce lookup", () => {
  it("finds exact outgoing evidence beyond the first history page", async () => {
    const f = fixture((cursor) => cursor ? { items: [entry], retentionDays: 30 } : {
      items: [{ ...entry, nonce: 8 }, { ...entry, from: "0x00000000000000000000000000000000000000bb" }], nextCursor: "second", retentionDays: 30,
    });
    expect(await f.client.sealedTransferAtNonce(sender, 7)).toEqual(entry);
    expect(f.calls()).toBe(2);
  });

  it.each([
    { direction: "in", delivery: "not_verified" },
    { operationId: "" }, { blockHash: "block" }, { entryHash: "" },
    { entryIndex: -1 }, { entryIndex: 0.5 }, { sealedAt: 0 },
  ])("rejects incomplete or incoming seal evidence: %j", async (change) => {
    const f = fixture(() => ({ items: [{ ...entry, ...change }], retentionDays: 30 }));
    await expect(f.client.sealedTransferAtNonce(sender, 7)).rejects.toThrow(/incomplete seal evidence/);
  });

  it("accepts a sender-sealed self transfer", async () => {
    const f = fixture(() => ({ items: [{ ...entry, direction: "self" }], retentionDays: 30 }));
    expect((await f.client.sealedTransferAtNonce(sender, 7))?.operationId).toBe(entry.operationId);
  });

  it("does not mistake missing history for proof of failure", async () => {
    const f = fixture(() => ({ items: [], retentionDays: 30 }));
    expect(await f.client.sealedTransferAtNonce(sender, 7)).toBeUndefined();
  });

  it("bounds repeated cursors and long histories", async () => {
    const repeated = fixture(() => ({ items: [], nextCursor: "same", retentionDays: 30 }));
    await expect(repeated.client.sealedTransferAtNonce(sender, 7)).rejects.toThrow(/repeated/);
    expect(repeated.calls()).toBe(2);
    const long = fixture((cursor) => ({ items: [], nextCursor: `${cursor}x`, retentionDays: 30 }));
    await expect(long.client.sealedTransferAtNonce(sender, 7)).rejects.toThrow(/1,000/);
    expect(long.calls()).toBe(10);
  });

  it("rejects imprecise nonces before making a request", async () => {
    const f = fixture(() => ({ items: [], retentionDays: 30 }));
    await expect(f.client.sealedTransferAtNonce(sender, Number.MAX_SAFE_INTEGER + 1)).rejects.toThrow(/safe integer/);
    expect(f.calls()).toBe(0);
  });
});
