import { describe, expect, it } from "vitest";
import { privateKeyToAccount } from "viem/accounts";
import {
  AmountError,
  APIError,
  displayParts,
  fromBaseUnits,
  GatewayClient,
  GatewayError,
  networkChainId,
  networkFingerprint,
  normalizeAmountInput,
  normalizeSignature,
  resolveRecipient,
  SubmissionError,
  toBaseUnits,
  transferOperationId,
  transferRecipientSlot,
  transferTypedData,
  VIRTUAL_ROUTER,
  type NetworkInfo,
  type TransferIntent,
} from "../../src/gateway/index.js";

const golden: TransferIntent = {
  catalogueFingerprint: "golden-vector-only",
  networkId: "YDEV",
  chainId: networkChainId("YDEV"),
  from: "0x00000000000000000000000000000000000000aa",
  to: "0x00000000000000000000000000000000000000bb",
  assets: [{ asset: "YELLOW", amount: "1000" }],
  maxFee: "100",
  nonce: 7,
};

describe("eip712", () => {
  it("derives chain IDs from network IDs", () => {
    expect(networkChainId("YDEV")).toBe(1497646422);
    expect(networkChainId("YTES")).toBe(0x59544553);
    expect(networkChainId("YNET")).toBe(1498301780);
    expect(() => networkChainId("YES")).toThrow();
  });

  it("matches the Go SDK operation ID golden vector", () => {
    expect(transferOperationId(golden)).toBe("0x6584fcefdda0d7cd8a6e664361851647fcc942bec3d39ae0fc0329646451165f");
  });

  it("matches the Go SDK's canonical URI recipient and operation ID", () => {
    const uri = "yellow://ynet/user/0x00000000000000000000000000000000000000bb" as const;
    const intent: TransferIntent = { ...golden, to: uri };
    expect(transferTypedData(intent).message.to).toBe(transferRecipientSlot(uri));
    expect(transferRecipientSlot(uri)).not.toBe(golden.to);
    expect(transferOperationId(intent)).toBe("0x01440c9c01d11dbc51566065cb6d27d89647992df4ea4a6d98ee6c865ad6bfb9");
    expect(() => transferRecipientSlot("yellow://ynet/node/nope")).toThrow(/node ID/);
    expect(() => transferRecipientSlot("yellow://ynet/treasury/notreal")).toThrow(/treasury/);
  });

  it("rejects unsorted asset legs", () => {
    const bad = { ...golden, assets: [{ asset: "YELLOW", amount: "1" }, { asset: "ETH", amount: "1" }] };
    expect(() => transferTypedData(bad)).toThrow(/sorted/);
  });

  it("resolves recipients like the CLI", () => {
    const addr = "0x236eB848C95b231299B4AA9f56c73D6893462720";
    expect(resolveRecipient(addr.toLowerCase())).toBe(addr);
    expect(resolveRecipient(addr.slice(2).toLowerCase())).toBe(addr);
    expect(resolveRecipient(`  yellow://ynet/user/${addr.toLowerCase()} `)).toBe(addr);
    expect(() => resolveRecipient("yellow://ynet/pool/eth")).toThrow(/user account/);
    expect(() => resolveRecipient("0x0000000000000000000000000000000000000000")).toThrow(/zero/);
    expect(() => resolveRecipient("0x236eb848C95b231299B4AA9f56c73D6893462720")).toThrow(/checksum/);
    expect(() => resolveRecipient("0x1234")).toThrow();
  });

  it("canonicalizes signature V to 27/28", () => {
    const base = `0x${"11".repeat(64)}`;
    expect(normalizeSignature(`${base}00`).endsWith("1b")).toBe(true);
    expect(normalizeSignature(`${base}1c`).endsWith("1c")).toBe(true);
    expect(() => normalizeSignature(`0x${"11".repeat(100)}`)).toThrow(/smart-contract/);
  });
});

describe("amounts", () => {
  it("converts exactly", () => {
    expect(toBaseUnits("1.25", 18)).toBe(1250000000000000000n);
    expect(toBaseUnits("0", 18)).toBe(0n);
    expect(toBaseUnits("10", 0)).toBe(10n);
    expect(() => toBaseUnits("1.", 6)).toThrow(AmountError);
    expect(() => toBaseUnits("1.1234567", 6)).toThrow(/6 decimal/);
    expect(() => toBaseUnits("-1", 6)).toThrow();
    expect(() => toBaseUnits("1e3", 6)).toThrow();
    expect(() => toBaseUnits("1", 1000000000)).toThrow(AmountError);
    expect(() => fromBaseUnits("1", -1)).toThrow(AmountError);
  });

  it("formats exactly", () => {
    expect(fromBaseUnits("1250000000000000000", 18)).toBe("1.25");
    expect(fromBaseUnits("5", 6)).toBe("0.000005");
    expect(fromBaseUnits("1000000", 6)).toBe("1");
    expect(displayParts("1234567.891234567", 4)).toEqual({ whole: "1,234,567", fraction: "8912", truncated: true });
    expect(displayParts("0.0000000042", 6).fraction).toBe("0000000042");
  });

  it("normalizes human input", () => {
    expect(normalizeAmountInput(" 1,5 ")).toBe("1.5");
    expect(normalizeAmountInput("1,234.5")).toBe("1234.5");
    expect(normalizeAmountInput(".5")).toBe("0.5");
  });
});

describe("fingerprint", () => {
  it("matches Go escaping and zero values for omitted asset fields", () => {
    const info = JSON.parse('{"networkId":"YDEV","chainId":1497646422,"virtualRouter":"0x00000000000000000000000000000000434C5200","assets":[{"symbol":"YELLOW","address":"<>&\\u2028\\u2029","decimals":18}]}') as NetworkInfo;
    const expected = "7ce75d8e8477b20bf1bff4bbb4d813153fc21aadf9c2181ce5f9205f179ce0cf";
    expect(networkFingerprint(info)).toBe(expected);
    const explicit = { ...info, assets: info.assets.map((a) => ({ ...a, chainId: 0, native: false })) };
    expect(networkFingerprint(explicit)).toBe(expected);
  });

  it("matches the CLI reviewed identity hash", () => {
    const info: NetworkInfo = {
      networkId: "YDEV",
      chainId: 1497646422,
      virtualRouter: VIRTUAL_ROUTER,
      capabilities: [],
      assets: [
        { symbol: "YELLOW", chainId: 31337, address: "0x236eB848C95b231299B4AA9f56c73D6893462720", decimals: 18, native: false },
        { symbol: "ETH", chainId: 31337, address: "", decimals: 18, native: true },
      ],
    };
    // Computed with Go encoding/json over the same identity struct.
    expect(networkFingerprint(info)).toBe("5401edc080f28328873bad1792572d10b8cb482be574fcc8dace40136509501e");
  });
});

const INFO = {
  networkId: "YDEV",
  chainId: 1497646422,
  virtualRouter: VIRTUAL_ROUTER.toLowerCase(),
  assets: [
    { symbol: "YELLOW", decimals: 18, chainId: 31337, address: "", native: false },
    { symbol: "ETH", decimals: 18, chainId: 31337, address: "", native: true },
  ],
  capabilities: ["transfer-operation-status-v1", "recent-transfers-v1"],
};

type Handler = (url: string, init?: RequestInit) => { status: number; body: unknown };

function fakeFetch(handler: Handler, info: () => typeof INFO = () => INFO): typeof fetch {
  return (async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = String(input);
    if (url.endsWith("/v1/gateway/info")) return new Response(JSON.stringify(info()), { status: 200 });
    const { status, body } = handler(url, init);
    return new Response(JSON.stringify(body), { status });
  }) as typeof fetch;
}

const account = privateKeyToAccount("0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d");

async function signedFor(client: GatewayClient) {
  const intent = await client.prepareTransfer({
    from: account.address,
    recipient: "0x00000000000000000000000000000000000000bb",
    assets: [{ asset: "YELLOW", amount: "1.5" }, { asset: "ETH", amount: "0.1" }],
    maxFee: "0.1",
  });
  return client.signTransfer(intent, (typedData) => account.signTypedData(typedData));
}

describe("GatewayClient", () => {
  it("rejects a gateway for another network", async () => {
    const c = new GatewayClient({ baseUrl: "https://gw.test", networkId: "YTES", fetch: fakeFetch(() => ({ status: 200, body: {} })) });
    await expect(c.info()).rejects.toMatchObject({ kind: "identity" });
  });

  it("rejects a gateway whose fingerprint changed", async () => {
    const c = new GatewayClient({ baseUrl: "https://gw.test", networkId: "YDEV", fingerprint: "00", fetch: fakeFetch(() => ({ status: 200, body: {} })) });
    await expect(c.info()).rejects.toMatchObject({ kind: "trust" });
  });

  it("refuses remote plain HTTP", () => {
    expect(() => new GatewayClient({ baseUrl: "http://gw.test", networkId: "YDEV" })).toThrow(GatewayError);
    expect(() => new GatewayClient({ baseUrl: "http://localhost:8080", networkId: "YDEV" })).not.toThrow();
  });

  it("prepares sorted legs with the live nonce, signs, and gets a receipt", async () => {
    let posted: Record<string, unknown> | undefined;
    const c = new GatewayClient({
      baseUrl: "https://gw.test",
      networkId: "YDEV",
      fetch: fakeFetch((url, init) => {
        if (url.includes("/balances")) return { status: 200, body: { address: account.address, nonce: 4, balances: [] } };
        posted = JSON.parse(String(init?.body));
        return { status: 200, body: { blockHash: "0xabc", entryIndex: 0, sealedAt: 1, stateRoot: "0x0", operationId: signedOpId } };
      }),
    });
    let signedOpId = "";
    const signed = await signedFor(c);
    signedOpId = signed.operationId;
    expect(signed.intent.nonce).toBe(4);
    expect(signed.intent.assets.map((a) => a.asset)).toEqual(["ETH", "YELLOW"]);
    expect(signed.intent.maxFee).toBe("100000000000000000");
    const result = await c.submitTransfer(signed);
    expect(result.status).toBe("sealed");
    expect(posted).toMatchObject({ nonce: 4, maxFee: "100000000000000000", to: expect.stringMatching(/^0x0{38}bb$/i) });
  });

  it("binds and submits an externally signed canonical URI intent unchanged", async () => {
    const uri = "yellow://ynet/user/0x00000000000000000000000000000000000000bb" as const;
    const intent: TransferIntent = { ...golden, catalogueFingerprint: networkFingerprint(INFO), from: account.address, to: uri };
    let posted: Record<string, unknown> | undefined;
    let expectedId = "";
    const c = new GatewayClient({
      baseUrl: "https://gw.test", networkId: "YDEV",
      fetch: fakeFetch((_url, init) => {
        posted = JSON.parse(String(init?.body));
        return { status: 202, body: { operationId: expectedId, kind: "transfer", status: "pending" } };
      }),
    });
    const signed = await c.finalizeSignature(intent, await account.signTypedData(transferTypedData(intent)));
    expectedId = signed.operationId;
    expect((await c.submitTransfer(signed)).status).toBe("pending");
    expect(posted).toMatchObject({ to: uri, nonce: 7, maxFee: "100" });
  });

  it("classifies submission outcomes", async () => {
    const outcomes: Array<[number, unknown, (e: SubmissionError) => void]> = [
      [422, { error: { code: -32003, message: "insufficient balance" } }, (e) => expect(e.definiteRejection).toBeInstanceOf(APIError)],
      [504, { error: { code: -32006, message: "outcome unknown" } }, (e) => expect(e.outcomeUnknown).toBe(true)],
      [409, { error: { code: -32009, message: "conflict" } }, (e) => expect(e.definiteRejection).toBeInstanceOf(APIError)],
      [400, "not json", (e) => expect(e.definiteRejection).toBeUndefined()],
    ];
    for (const [status, body, check] of outcomes) {
      const c = new GatewayClient({
        baseUrl: "https://gw.test",
        networkId: "YDEV",
        fetch: fakeFetch((url) => (url.includes("/balances") ? { status: 200, body: { nonce: 0, balances: [] } } : { status, body })),
      });
      const signed = await signedFor(c);
      const err = await c.submitTransfer(signed, { firstSubmission: true }).catch((e) => e);
      expect(err).toBeInstanceOf(SubmissionError);
      expect(err.submitted).toBe(true);
      check(err);
    }
  });

  it("keeps every refused replay unknown without an explicit first-submission assertion", async () => {
    for (const status of [400, 401, 403, 409, 422, 429]) {
      const posted: string[] = [];
      const c = new GatewayClient({
        baseUrl: "https://gw.test", networkId: "YDEV",
        fetch: fakeFetch((url, init) => {
          if (url.includes("/balances")) return { status: 200, body: { nonce: 7, balances: [] } };
          posted.push(String(init?.body));
          return posted.length === 1
            ? { status: 504, body: { error: { code: -32006, message: "outcome unknown" } } }
            : { status, body: { error: { code: -32002, message: "stale nonce" } } };
        }),
      });
      const signed = await signedFor(c);
      const first = await c.submitTransfer(signed, { firstSubmission: true }).catch((e: unknown) => e);
      expect(first).toMatchObject({ submitted: true, outcomeUnknown: true });
      for (const options of [undefined, { firstSubmission: false }]) {
        const replay = await c.submitTransfer(signed, options).catch((e: unknown) => e);
        expect(replay).toBeInstanceOf(SubmissionError);
        expect(replay).toMatchObject({ submitted: true, outcomeUnknown: true, operationId: signed.operationId });
        expect((replay as SubmissionError).definiteRejection).toBeUndefined();
      }
      expect(posted).toHaveLength(3);
      expect(new Set(posted).size).toBe(1);
    }
  });

  it("does not submit a tampered authorization", async () => {
    let posts = 0;
    const c = new GatewayClient({
      baseUrl: "https://gw.test",
      networkId: "YDEV",
      fetch: fakeFetch((url) => {
        if (url.includes("/balances")) return { status: 200, body: { nonce: 0, balances: [] } };
        posts++;
        return { status: 200, body: {} };
      }),
    });
    const signed = await signedFor(c);
    const tampered = { ...signed, intent: { ...signed.intent, maxFee: "999" } };
    const err = await c.submitTransfer(tampered).catch((e) => e);
    expect(err).toBeInstanceOf(SubmissionError);
    expect(err.submitted).toBe(false);
    expect(posts).toBe(0);
  });

  it("rejects a signature from another account", async () => {
    const c = new GatewayClient({
      baseUrl: "https://gw.test",
      networkId: "YDEV",
      fetch: fakeFetch(() => ({ status: 200, body: { nonce: 0, balances: [] } })),
    });
    const intent = await c.prepareTransfer({ from: "0x00000000000000000000000000000000000000aa", recipient: account.address, assets: [{ asset: "ETH", amount: "1" }], maxFee: "0" });
    const sig = await account.signTypedData(transferTypedData(intent));
    await expect(c.finalizeSignature(intent, sig)).rejects.toThrow(/different account/);
  });

  it("treats a cancelled submission as unsent", async () => {
    let posts = 0;
    const c = new GatewayClient({
      baseUrl: "https://gw.test",
      networkId: "YDEV",
      fetch: fakeFetch((url) => {
        if (url.includes("/balances")) return { status: 200, body: { nonce: 0, balances: [] } };
        posts++;
        return { status: 200, body: {} };
      }),
    });
    const signed = await signedFor(c);
    const controller = new AbortController();
    controller.abort();
    const err = await c.submitTransfer(signed, controller.signal).catch((e) => e);
    expect(err).toBeInstanceOf(SubmissionError);
    expect(err.submitted).toBe(false);
    expect(err.outcomeUnknown).toBe(false);
    expect(posts).toBe(0);
  });

  it("refuses authority-relative gateway URLs", () => {
    for (const baseUrl of ["//evil.test", "/\\evil.test"]) {
      expect(() => new GatewayClient({ baseUrl, networkId: "YDEV" })).toThrow(GatewayError);
    }
    expect(() => new GatewayClient({ baseUrl: "/api/gw/testnet", networkId: "YDEV" })).not.toThrow();
  });

  it("refuses an unsupported decimal count from the gateway", async () => {
    const bad = { ...INFO, assets: [{ symbol: "YELLOW", decimals: 1000000000, chainId: 1, address: "", native: false }] };
    const c = new GatewayClient({
      baseUrl: "https://gw.test",
      networkId: "YDEV",
      fetch: (async () => new Response(JSON.stringify(bad), { status: 200 })) as typeof fetch,
    });
    await expect(c.info()).rejects.toThrow(/unsupported decimal count/);
  });

  it("refuses an oversized gateway response", async () => {
    const huge = "x".repeat(2 << 20);
    const c = new GatewayClient({
      baseUrl: "https://gw.test",
      networkId: "YDEV",
      fetch: (async () => new Response(JSON.stringify({ ...INFO, version: huge }), { status: 200 })) as typeof fetch,
    });
    await expect(c.info()).rejects.toThrow(/too large/);
  });
});


describe("prepared catalogue binding", () => {
  const changes = {
    decimals: (info: typeof INFO) => { info.assets[1]!.decimals = 6; },
    backingAddress: (info: typeof INFO) => { info.assets[1]!.address = "0x00000000000000000000000000000000000000cc"; },
    backingChain: (info: typeof INFO) => { info.assets[1]!.chainId = 1; },
    native: (info: typeof INFO) => { info.assets[1]!.native = false; },
    feeDecimals: (info: typeof INFO) => { info.assets[0]!.decimals = 6; },
    removedAsset: (info: typeof INFO) => { info.assets.pop(); },
  };
  for (const [name, change] of Object.entries(changes)) {
    it(`rejects ${name} drift before signing and before submission`, async () => {
      let info = structuredClone(INFO);
      let posts = 0;
      const c = new GatewayClient({ baseUrl: "https://gw.test", networkId: "YDEV", fetch: fakeFetch((url) => {
        if (url.includes("/balances")) return { status: 200, body: { nonce: 7, balances: [] } };
        posts++;
        throw new Error("drift reached POST");
      }, () => info) });
      const intent = await c.prepareTransfer({ from: account.address, recipient: golden.to, assets: [{ asset: "ETH", amount: "1" }], maxFee: "0.1" });
      expect(intent.catalogueFingerprint).toBe(networkFingerprint(info));
      const signed = await c.signTransfer(intent, (data) => account.signTypedData(data));
      const restored = JSON.parse(JSON.stringify(signed));
      change(info);
      let signatures = 0;
      await expect(c.signTransfer(intent, async (data) => { signatures++; return account.signTypedData(data); })).rejects.toMatchObject({ kind: "trust" });
      await expect(c.submitTransfer(restored)).rejects.toMatchObject({ submitted: false, outcomeUnknown: false });
      expect(signatures).toBe(0);
      expect(posts).toBe(0);
      info = structuredClone(INFO);
      info.assets.reverse();
      info.assets[0]!.address = info.assets[0]!.address.toUpperCase();
      await expect(c.signTransfer(intent, (data) => account.signTypedData(data))).resolves.toMatchObject({ operationId: signed.operationId });
    });
  }
  it("refuses intents without retained catalogue metadata", async () => {
    const c = new GatewayClient({ baseUrl: "https://gw.test", networkId: "YDEV", fetch: fakeFetch(() => ({ status: 200, body: { nonce: 7, balances: [] } })) });
    const signed = await signedFor(c);
    const legacy = JSON.parse(JSON.stringify(signed));
    delete legacy.intent.catalogueFingerprint;
    await expect(c.signTransfer(legacy.intent, (data) => account.signTypedData(data))).rejects.toMatchObject({ kind: "trust" });
    await expect(c.submitTransfer(legacy)).rejects.toMatchObject({ submitted: false });
  });
});
