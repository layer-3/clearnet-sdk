import type { Payment, SubmitResponse, TxResponse } from "xrpl";
import { afterEach, beforeEach, describe, expect, expectTypeOf, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  clientConstructor: vi.fn(),
}));

vi.mock("xrpl", async (importOriginal) => {
  const actual = await importOriginal<typeof import("xrpl")>();
  return {
    ...actual,
    Client: mocks.clientConstructor,
  };
});

import {
  ClearnetSdkError,
  XrplVaultDepositor,
  XRPL_NATIVE_ASSET,
} from "../../../src/index.js";
import type {
  Bytes32Hex,
  DepositStatus,
  SubmitDepositOptions,
  TxRef,
  VaultDepositor,
  XrplDepositDestination,
  XrplIssuedDepositInput,
  XrplNativeDepositInput,
  XrplSigner,
  XrplSubmitDepositInput,
} from "../../../src/index.js";

const RPC_URL = "ws://127.0.0.1:6006";
const VAULT_ADDRESS = "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh";
const DEPOSITOR_ADDRESS = "rPT1Sjq2YGrBMTttX4GZHjKu9dyfzbpAYe";
const ISSUER_ADDRESS = "rN7n7otQDd6FczFgLdSqtcsAUxDkw6fzRH";
const ACCOUNT = "0x1111111111111111111111111111111111111111";
const ACCOUNT_NO_PREFIX = ACCOUNT.slice(2);
const REFERENCE =
  "0x2222222222222222222222222222222222222222222222222222222222222222" as Bytes32Hex;
const HASH_RAW =
  "ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789ABCDEF0123456789";
const HASH_REF = {
  hash: `0x${HASH_RAW.toLowerCase()}`,
  raw: HASH_RAW,
} satisfies TxRef;
const TX_BLOB = "1200002280000000240000000161400000000000000A68400000000000000C";
const MEMO_TYPE = "796e65742d6163636f756e74";

interface MockXrplClient {
  connect: ReturnType<typeof vi.fn<() => Promise<void>>>;
  disconnect: ReturnType<typeof vi.fn<() => Promise<void>>>;
  isConnected: ReturnType<typeof vi.fn<() => boolean>>;
  autofill: ReturnType<typeof vi.fn<(payment: Payment) => Promise<Payment>>>;
  submit: ReturnType<typeof vi.fn<(txBlob: string, options?: unknown) => Promise<SubmitResponse>>>;
  request: ReturnType<typeof vi.fn<(request: unknown) => Promise<TxResponse>>>;
}

interface MockSigner extends XrplSigner {
  sign: ReturnType<typeof vi.fn<(payment: Payment) => Promise<{ txBlob: string; hash: string }>>>;
}

describe("XrplVaultDepositor", () => {
  let client: MockXrplClient;

  beforeEach(() => {
    client = createClient();
    mocks.clientConstructor.mockReset();
    mocks.clientConstructor.mockImplementation(function Client() {
      return client;
    });
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("matches the public depositor and result type contracts", () => {
    expectTypeOf<XrplVaultDepositor>().toMatchTypeOf<
      VaultDepositor<XrplSubmitDepositInput>
    >();
    expectTypeOf<XrplSubmitDepositInput>().toEqualTypeOf<
      XrplNativeDepositInput | XrplIssuedDepositInput
    >();
    expectTypeOf<XrplNativeDepositInput["amount"]>().toEqualTypeOf<bigint>();
    expectTypeOf<XrplIssuedDepositInput["amount"]>().toEqualTypeOf<string>();
    expectTypeOf<{
      asset: "XRP";
      amount: string;
      destination: XrplDepositDestination;
    }>().not.toMatchTypeOf<XrplSubmitDepositInput>();
    expectTypeOf<{
      asset: `USD.${string}`;
      amount: bigint;
      destination: XrplDepositDestination;
    }>().not.toMatchTypeOf<XrplSubmitDepositInput>();
    expectTypeOf<TxRef>().toEqualTypeOf<{ hash: Bytes32Hex; raw: string }>();
    expectTypeOf<DepositStatus>().toEqualTypeOf<
      "absent" | "pending" | "confirmed"
    >();
    expect(XRPL_NATIVE_ASSET).toBe("XRP");
  });

  it("submits native XRP drops with the ynet-account memo and Go-compatible tx ref", async () => {
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const onSubmitted = vi.fn();

    const ref = await depositor.submitDeposit(
      {
        asset: XRPL_NATIVE_ASSET,
        amount: 10n,
        destination: { account: ` ${ACCOUNT} `, ref: REFERENCE },
      },
      { onSubmitted },
    );

    expect(mocks.clientConstructor).toHaveBeenCalledExactlyOnceWith(RPC_URL);
    expect(client.connect).toHaveBeenCalledOnce();
    expect(client.autofill).toHaveBeenCalledExactlyOnceWith({
      TransactionType: "Payment",
      Account: DEPOSITOR_ADDRESS,
      Destination: VAULT_ADDRESS,
      Amount: "10",
      Memos: [
        {
          Memo: {
            MemoType: MEMO_TYPE,
            MemoData: `${ACCOUNT_NO_PREFIX}${REFERENCE.slice(2)}`,
          },
        },
      ],
    });
    expect(signer.sign).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ Fee: "12", Sequence: 1 }),
    );
    expect(client.submit).toHaveBeenCalledExactlyOnceWith(TX_BLOB, {
      autofill: false,
    });
    expect(ref).toEqual(HASH_REF);
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);
  });

  it("submits issued-currency deposits using both supported asset delimiters", async () => {
    const signer = createSigner();
    const depositor = createDepositor(signer);

    await depositor.submitDeposit({
      asset: `USD.${ISSUER_ADDRESS}`,
      amount: "12.345",
      destination: { account: ACCOUNT },
    });
    await depositor.submitDeposit({
      asset: `EUR:${ISSUER_ADDRESS}`,
      amount: "1",
      destination: { account: ACCOUNT },
    });

    expect(client.autofill).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        Amount: {
          currency: "USD",
          issuer: ISSUER_ADDRESS,
          value: "12.345",
        },
      }),
    );
    expect(client.autofill).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        Amount: {
          currency: "EUR",
          issuer: ISSUER_ADDRESS,
          value: "1",
        },
      }),
    );
  });

  it("accepts terQUEUED submit results and returns without waiting for validation", async () => {
    client.submit.mockResolvedValueOnce(submitResponse("terQUEUED"));
    const signer = createSigner();
    const depositor = createDepositor(signer);

    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).resolves.toEqual(HASH_REF);

    expect(client.request).not.toHaveBeenCalled();
  });

  it("rejects invalid inputs before autofill, signing, or submission", async () => {
    const signer = createSigner();
    const depositor = createDepositor(signer);

    await expect(
      depositor.submitDeposit(null as unknown as XrplSubmitDepositInput),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "destination is required and must be an object",
    });
    await expect(
      depositor.submitDeposit(
        {
          asset: XRPL_NATIVE_ASSET,
          amount: 1n,
          destination: { account: ACCOUNT },
        },
        null as never,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: "submit options must be an object",
    });
    await expect(
      depositor.submitDeposit(
        {
          asset: XRPL_NATIVE_ASSET,
          amount: 1n,
          destination: { account: ACCOUNT },
        },
        { onSubmitted: "bad" } as unknown as SubmitDepositOptions,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: "submit options.onSubmitted must be a function",
    });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: null as unknown as XrplDepositDestination,
      }),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "destination is required and must be an object",
    });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: "bad" as unknown as XrplDepositDestination,
      }),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "destination is required and must be an object",
    });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 0n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1.5 as unknown as bigint,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: `USD.${ISSUER_ADDRESS}`,
        amount: "1e2",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: `USD.${ISSUER_ADDRESS}`,
        amount: "0",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: "USD" as XrplIssuedDepositInput["asset"],
        amount: "1",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: "USD.rBad",
        amount: "1",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: "0x1234" },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: `yellow://local/user/${ACCOUNT}` },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT, ref: "memo-1" as Bytes32Hex },
      }),
    ).rejects.toMatchObject({ code: "INVALID_REFERENCE" });

    expect(client.autofill).not.toHaveBeenCalled();
    expect(signer.sign).not.toHaveBeenCalled();
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("rejects invalid constructor inputs with XRPL validation errors", () => {
    expect(() =>
      new XrplVaultDepositor({
        rpcUrl: RPC_URL,
        vaultAddress: VAULT_ADDRESS,
        signer: undefined as unknown as XrplSigner,
      }),
    ).toThrowError(
      expect.objectContaining({ code: "MISSING_WALLET_ACCOUNT" }),
    );
    expect(() =>
      new XrplVaultDepositor({
        rpcUrl: RPC_URL,
        vaultAddress: VAULT_ADDRESS,
        signer: { classicAddress: "rBad" } as XrplSigner,
      }),
    ).toThrowError(
      expect.objectContaining({ code: "INVALID_ADDRESS" }),
    );
    expect(() =>
      new XrplVaultDepositor({
        rpcUrl: RPC_URL,
        vaultAddress: VAULT_ADDRESS,
        signer: { classicAddress: DEPOSITOR_ADDRESS } as XrplSigner,
      }),
    ).toThrowError(
      expect.objectContaining({ code: "MISSING_WALLET_ACCOUNT" }),
    );
    expect(() =>
      new XrplVaultDepositor({
        rpcUrl: "http://127.0.0.1:5005",
        vaultAddress: VAULT_ADDRESS,
        signer: createSigner(),
      }),
    ).toThrowError(
      expect.objectContaining({ code: "RPC_ERROR" }),
    );
  });

  it("enforces maxFeeDrops after autofill and before signing", async () => {
    client.autofill.mockResolvedValueOnce(preparedPayment({ Fee: "13" }));
    const signer = createSigner();
    const depositor = createDepositor(signer, { maxFeeDrops: 12n });

    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });

    expect(signer.sign).not.toHaveBeenCalled();
    expect(client.submit).not.toHaveBeenCalled();
  });

  it("rejects failed submit engine results and malformed signer hashes", async () => {
    client.submit.mockResolvedValueOnce(submitResponse("tecNO_DST"));
    const signer = createSigner();
    const depositor = createDepositor(signer);

    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "TX_REVERTED", txRef: HASH_REF });

    signer.sign.mockResolvedValueOnce({ txBlob: TX_BLOB, hash: "not-a-hash" });
    await expect(
      depositor.submitDeposit({
        asset: XRPL_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
  });

  it("maps XRPL tx lookup status to the shared deposit status", async () => {
    const depositor = createDepositor(createSigner());

    client.request.mockResolvedValueOnce(txResponse(true));
    await expect(depositor.verifyDeposit(HASH_REF, 100)).resolves.toBe(
      "confirmed",
    );

    client.request.mockResolvedValueOnce(txResponse(true));
    await expect(depositor.verifyDeposit(HASH_REF, 1n << 80n)).resolves.toBe(
      "confirmed",
    );

    client.request.mockResolvedValueOnce(txResponse(false));
    await expect(depositor.verifyDeposit(HASH_REF, 1n)).resolves.toBe("pending");

    client.request.mockRejectedValueOnce({
      message: "Transaction not found.",
      data: { error: "txnNotFound" },
    });
    await expect(depositor.verifyDeposit(HASH_REF, 0)).resolves.toBe("absent");

    const rpcError = new Error("node offline");
    client.request.mockRejectedValueOnce(rpcError);
    await expect(depositor.verifyDeposit(HASH_REF, 0)).rejects.toMatchObject({
      code: "RPC_ERROR",
      cause: rpcError,
    });
  });

  it("validates tx refs and min confirmations before tx lookup", async () => {
    const depositor = createDepositor(createSigner());

    await expect(
      depositor.verifyDeposit({ hash: "0x1234" as Bytes32Hex, raw: HASH_RAW }, 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
    await expect(
      depositor.verifyDeposit({ hash: HASH_REF.hash, raw: "not-a-hash" }, 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
    await expect(depositor.verifyDeposit(HASH_REF, -1)).rejects.toMatchObject({
      code: "INVALID_CONFIRMATIONS",
    });

    expect(client.request).not.toHaveBeenCalled();
  });

  it("disconnects the underlying XRPL client only when connected", async () => {
    const depositor = createDepositor(createSigner());

    client.isConnected.mockReturnValueOnce(false);
    await depositor.disconnect();
    expect(client.disconnect).not.toHaveBeenCalled();

    client.isConnected.mockReturnValueOnce(true);
    await depositor.disconnect();
    expect(client.disconnect).toHaveBeenCalledOnce();
  });

  it("wraps disconnect failures as RPC errors", async () => {
    const depositor = createDepositor(createSigner());
    const cause = new Error("socket close failed");
    client.isConnected.mockReturnValueOnce(true);
    client.disconnect.mockRejectedValueOnce(cause);

    await expect(depositor.disconnect()).rejects.toMatchObject({
      code: "RPC_ERROR",
      message: "xrpl: disconnect",
      cause,
    });
  });

  it("reconnects after disconnect for later verification", async () => {
    const depositor = createDepositor(createSigner());

    client.isConnected.mockReturnValueOnce(true);
    await depositor.disconnect();

    client.isConnected.mockReturnValueOnce(false);
    await expect(depositor.verifyDeposit(HASH_REF, 0)).resolves.toBe(
      "confirmed",
    );
    expect(client.connect).toHaveBeenCalledOnce();
  });

  it("disconnects after an in-flight connect settles", async () => {
    const connect = deferred<void>();
    client.connect.mockImplementationOnce(() => connect.promise);
    const depositor = createDepositor(createSigner());

    const verification = depositor.verifyDeposit(HASH_REF, 0);
    await Promise.resolve();

    const disconnect = depositor.disconnect();
    await Promise.resolve();
    expect(client.disconnect).not.toHaveBeenCalled();

    client.isConnected.mockReturnValue(true);
    connect.resolve();
    await disconnect;
    await expect(verification).resolves.toBe("confirmed");
    expect(client.disconnect).toHaveBeenCalledOnce();
  });
});

function createDepositor(
  signer = createSigner(),
  overrides: Partial<ConstructorParameters<typeof XrplVaultDepositor>[0]> = {},
): XrplVaultDepositor {
  return new XrplVaultDepositor({
    rpcUrl: RPC_URL,
    vaultAddress: VAULT_ADDRESS,
    signer,
    ...overrides,
  });
}

function createClient(): MockXrplClient {
  return {
    connect: vi.fn(async () => undefined),
    disconnect: vi.fn(async () => undefined),
    isConnected: vi.fn(() => false),
    autofill: vi.fn(async (payment) => preparedPayment(payment)),
    submit: vi.fn(async () => submitResponse("tesSUCCESS")),
    request: vi.fn(async () => txResponse(true)),
  };
}

function createSigner(): MockSigner {
  return {
    classicAddress: DEPOSITOR_ADDRESS,
    sign: vi.fn(async () => ({ txBlob: TX_BLOB, hash: HASH_RAW.toLowerCase() })),
  };
}

function preparedPayment(payment: Partial<Payment> = {}): Payment {
  return {
    TransactionType: "Payment",
    Account: DEPOSITOR_ADDRESS,
    Destination: VAULT_ADDRESS,
    Amount: "10",
    ...payment,
    Fee: payment.Fee ?? "12",
    Sequence: payment.Sequence ?? 1,
  } as Payment;
}

function submitResponse(engineResult: string): SubmitResponse {
  return {
    type: "response",
    result: {
      engine_result: engineResult,
      engine_result_code: engineResult === "tesSUCCESS" ? 0 : -1,
      engine_result_message: engineResult,
      tx_blob: TX_BLOB,
      tx_json: { TransactionType: "Payment" },
      accepted: true,
      account_sequence_available: 1,
      account_sequence_next: 2,
      applied: engineResult === "tesSUCCESS",
      broadcast: true,
      kept: true,
      queued: engineResult === "terQUEUED",
      open_ledger_cost: "10",
      validated_ledger_index: 1,
    },
  } as unknown as SubmitResponse;
}

function txResponse(validated: boolean): TxResponse {
  return {
    type: "response",
    result: {
      hash: HASH_RAW,
      validated,
      tx_json: { TransactionType: "Payment" },
    },
  } as unknown as TxResponse;
}

function deferred<T>(): {
  promise: Promise<T>;
  resolve: (value: T | PromiseLike<T>) => void;
  reject: (reason?: unknown) => void;
} {
  let resolve!: (value: T | PromiseLike<T>) => void;
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((resolvePromise, rejectPromise) => {
    resolve = resolvePromise;
    reject = rejectPromise;
  });
  return { promise, resolve, reject };
}
