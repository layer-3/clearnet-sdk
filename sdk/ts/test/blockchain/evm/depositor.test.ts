import { describe, expect, expectTypeOf, it, vi } from "vitest";
import { encodeAbiParameters, encodeEventTopics, zeroAddress, zeroHash } from "viem";
import type {
  Address,
  Hash,
  PublicClient,
  TransactionReceipt,
  WalletClient,
} from "viem";

import {
  ClearnetSdkError,
  EVM_NATIVE_ASSET,
  EvmVaultDepositor,
  depositId,
} from "../../../src/index.js";
import type {
  DepositStatus,
  EvmSubmitDepositInput,
  SubmitDepositResult,
  VaultDepositor,
} from "../../../src/index.js";
import { custodyAbi } from "../../../src/blockchain/evm/abi.js";
import { requireDepositDestination } from "../../../src/blockchain/evm/validation.js";

const CHAIN_ID = 31_337;
const CUSTODY_ADDRESS =
  "0x0000000000000000000000000000000000001000" as Address;
const ACCOUNT = "0x0000000000000000000000000000000000002000" as Address;
const TOKEN = "0x0000000000000000000000000000000000003000" as Address;
const DEPOSIT_HASH =
  "0x1111111111111111111111111111111111111111111111111111111111111111" as Hash;
const APPROVAL_HASH =
  "0x2222222222222222222222222222222222222222222222222222222222222222" as Hash;
const DEPOSIT_REFERENCE =
  "0x3333333333333333333333333333333333333333333333333333333333333333" as Hash;
// The depositor address is ACCOUNT, nonce key 0, sequence 0 (the default
// getNonce mock), matching how createClients()/createDepositor() are wired.
const DEPOSIT_ID = depositId(BigInt(CHAIN_ID), CUSTODY_ADDRESS, ACCOUNT, 0n);

interface ClientMocks {
  publicClient: PublicClient;
  walletClient: WalletClient;
  publicMock: {
    getChainId: ReturnType<typeof vi.fn>;
    readContract: ReturnType<typeof vi.fn>;
    waitForTransactionReceipt: ReturnType<typeof vi.fn>;
    getTransactionReceipt: ReturnType<typeof vi.fn>;
    getTransaction: ReturnType<typeof vi.fn>;
    getBlockNumber: ReturnType<typeof vi.fn>;
  };
  walletMock: {
    getChainId: ReturnType<typeof vi.fn>;
    writeContract: ReturnType<typeof vi.fn>;
  };
}

describe("EvmVaultDepositor", () => {
  it("matches the public depositor and result type contracts", () => {
    expectTypeOf<EvmVaultDepositor>().toMatchTypeOf<
      VaultDepositor<EvmSubmitDepositInput>
    >();
    expectTypeOf<SubmitDepositResult>().toEqualTypeOf<{
      txHash: string;
      depositId: string;
    }>();
    expectTypeOf<DepositStatus>().toEqualTypeOf<
      "absent" | "pending" | "confirmed"
    >();
  });

  it("exports the native depositor asset marker", () => {
    expect(EVM_NATIVE_ASSET).toBe("");
  });

  it("submits a native ETH deposit with matching value and returns both identifiers", async () => {
    const clients = createClients();
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);

    const depositor = createDepositor(clients);
    const onSubmitted = vi.fn();
    const result = await depositor.submitDeposit(
      {
        destination: { account: ACCOUNT, ref: DEPOSIT_REFERENCE },
        asset: EVM_NATIVE_ASSET,
        amount: "10",
      },
      { onSubmitted },
    );

    expect(result).toEqual({ txHash: DEPOSIT_HASH, depositId: DEPOSIT_ID });
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(result);
    expect(clients.walletMock.writeContract).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        address: CUSTODY_ADDRESS,
        functionName: "deposit",
        args: [ACCOUNT, zeroAddress, 10n, DEPOSIT_REFERENCE, 0n],
        value: 10n,
        account: ACCOUNT,
        chain: null,
      }),
    );
    expect(clients.publicMock.waitForTransactionReceipt).toHaveBeenCalledWith({
      hash: DEPOSIT_HASH,
    });
  });

  it("uses the nonce key an EvmSubmitDepositInput requests", async () => {
    const key = 7n;
    const nonce = (key << 64n) | 3n;
    const clients = createClients({ nonce, waitReceipt: receipt({ nonce }) });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);

    const result = await depositor.submitDeposit({
      destination: { account: ACCOUNT },
      asset: EVM_NATIVE_ASSET,
      amount: "1",
      nonceKey: key,
    });

    expect(result.depositId).toBe(
      depositId(BigInt(CHAIN_ID), CUSTODY_ADDRESS, ACCOUNT, nonce),
    );
    expect(clients.publicMock.readContract).toHaveBeenCalledWith(
      expect.objectContaining({ functionName: "getNonce", args: [ACCOUNT, key] }),
    );
    expect(clients.walletMock.writeContract).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        args: [ACCOUNT, zeroAddress, 1n, zeroHash, nonce],
      }),
    );
  });

  it.each([
    ["negative", -1n],
    ["2^192", 1n << 192n],
    ["a number", 7],
  ])("rejects a %s nonceKey before reading the nonce", async (_name, nonceKey) => {
    const clients = createClients();
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
        nonceKey: nonceKey as bigint,
      }),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    expect(clients.publicMock.readContract).not.toHaveBeenCalledWith(
      expect.objectContaining({ functionName: "getNonce" }),
    );
    expect(clients.walletMock.writeContract).not.toHaveBeenCalled();
  });

  it("approves an exact ERC-20 amount before depositing and returns both identifiers", async () => {
    const clients = createClients();
    clients.publicMock.waitForTransactionReceipt
      .mockResolvedValueOnce(receipt())
      .mockResolvedValueOnce(
        receipt({ asset: TOKEN, amount: 25n, reference: zeroHash }),
      );
    clients.walletMock.writeContract
      .mockResolvedValueOnce(APPROVAL_HASH)
      .mockResolvedValueOnce(DEPOSIT_HASH);

    const depositor = createDepositor(clients);
    const onSubmitted = vi.fn();
    const result = await depositor.submitDeposit(
      { destination: { account: ACCOUNT }, asset: TOKEN, amount: "25" },
      { onSubmitted },
    );

    expect(result).toEqual({ txHash: DEPOSIT_HASH, depositId: DEPOSIT_ID });
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(result);
    expect(clients.walletMock.writeContract).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({
        address: TOKEN,
        functionName: "approve",
        args: [CUSTODY_ADDRESS, 25n],
        account: ACCOUNT,
        chain: null,
      }),
    );
    expect(clients.walletMock.writeContract).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({
        address: CUSTODY_ADDRESS,
        functionName: "deposit",
        args: [ACCOUNT, TOKEN, 25n, zeroHash, 0n],
        account: ACCOUNT,
        chain: null,
      }),
    );
    expect(clients.publicMock.waitForTransactionReceipt).toHaveBeenNthCalledWith(
      1,
      { hash: APPROVAL_HASH },
    );
    expect(clients.publicMock.waitForTransactionReceipt).toHaveBeenNthCalledWith(
      2,
      { hash: DEPOSIT_HASH },
    );
  });

  it("reads the nonce before the allowance and before the approve is sent", async () => {
    const clients = createClients();
    const order: string[] = [];
    const readImpl = defaultRead(clients);
    clients.publicMock.readContract.mockImplementation(
      (args: { functionName: string }) => {
        order.push(args.functionName);
        return readImpl(args);
      },
    );
    clients.walletMock.writeContract.mockImplementation(
      (args: { functionName: string }) => {
        order.push(`write:${args.functionName}`);
        return Promise.resolve(
          args.functionName === "approve" ? APPROVAL_HASH : DEPOSIT_HASH,
        );
      },
    );
    clients.publicMock.waitForTransactionReceipt.mockResolvedValue(
      receipt({ asset: TOKEN, amount: 25n, reference: zeroHash }),
    );
    const depositor = createDepositor(clients);

    await depositor.submitDeposit({
      destination: { account: ACCOUNT },
      asset: TOKEN,
      amount: "25",
    });

    expect(order.filter((c) => c !== "decimals")).toEqual([
      "getNonce",
      "allowance",
      "write:approve",
      "write:deposit",
    ]);
  });

  it("skips a redundant ERC-20 approval when the allowance already covers the amount", async () => {
    const clients = createClients({ allowance: 100n });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    clients.publicMock.waitForTransactionReceipt.mockResolvedValueOnce(
      receipt({ asset: TOKEN, amount: 25n, reference: zeroHash }),
    );
    const depositor = createDepositor(clients);

    const result = await depositor.submitDeposit({
      destination: { account: ACCOUNT },
      asset: TOKEN,
      amount: "25",
    });

    expect(result.txHash).toBe(DEPOSIT_HASH);
    expect(clients.walletMock.writeContract).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({ functionName: "deposit" }),
    );
  });

  it("throws TX_REVERTED with the deposit step, txHash and depositId when the deposit receipt fails", async () => {
    const clients = createClients({
      waitReceipt: receipt({ status: "reverted" }),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
      }),
    ).rejects.toMatchObject({
      code: "TX_REVERTED",
      txHash: DEPOSIT_HASH,
      step: "deposit",
      depositId: DEPOSIT_ID,
    });
  });

  it("throws STALE_NONCE with txHash when a mined deposit reverts and getNonce has moved past its nonce", async () => {
    const clients = createClients({
      waitReceipt: receipt({ status: "reverted" }),
    });
    let nonceReads = 0;
    const readImpl = defaultRead(clients);
    clients.publicMock.readContract.mockImplementation(
      (args: { functionName: string }) => {
        if (args.functionName === "getNonce") {
          nonceReads++;
          return Promise.resolve(nonceReads === 1 ? 0n : 1n);
        }
        return readImpl(args);
      },
    );
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
      }),
    ).rejects.toMatchObject({
      code: "STALE_NONCE",
      txHash: DEPOSIT_HASH,
      step: "deposit",
      depositId: DEPOSIT_ID,
      nonceKey: 0n,
      nonce: 0n,
    });
  });

  it.each([
    ["has no Deposited log", { ...receipt(), logs: [] } as TransactionReceipt],
    [
      "has the Deposited topic only from another contract",
      {
        ...receipt(),
        logs: receipt().logs.map((log) => ({
          ...log,
          address: "0x0000000000000000000000000000000000009999" as Address,
        })),
      } as TransactionReceipt,
    ],
    ["has a Deposited log for another nonce", receipt({ nonce: 1n })],
  ])(
    "throws DEPOSIT_EVENT_NOT_FOUND with txHash and depositId when the mined receipt %s",
    async (_name, mined) => {
      const clients = createClients({ waitReceipt: mined });
      clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
      const depositor = createDepositor(clients);
      const onSubmitted = vi.fn();

      await expect(
        depositor.submitDeposit(
          { destination: { account: ACCOUNT }, asset: EVM_NATIVE_ASSET, amount: "1" },
          { onSubmitted },
        ),
      ).rejects.toMatchObject({
        code: "DEPOSIT_EVENT_NOT_FOUND",
        txHash: DEPOSIT_HASH,
        step: "deposit",
        depositId: DEPOSIT_ID,
        nonceKey: 0n,
        nonce: 0n,
      });
      expect(onSubmitted).not.toHaveBeenCalled();
    },
  );

  it("throws RECEIPT_TIMEOUT with the deposit identity when the receipt wait is aborted", async () => {
    const clients = createClients({
      waitReceiptPromise: new Promise<TransactionReceipt>(() => undefined),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);
    const controller = new AbortController();

    const pending = depositor.submitDeposit(
      { destination: { account: ACCOUNT }, asset: EVM_NATIVE_ASSET, amount: "1" },
      { signal: controller.signal, receiptTimeoutMs: 60_000 },
    );
    setTimeout(() => controller.abort(), 1);

    await expect(pending).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txHash: DEPOSIT_HASH,
      step: "deposit",
      depositId: DEPOSIT_ID,
      nonceKey: 0n,
      nonce: 0n,
    });
  });

  it("carries the deposit identity on an allowance read failure, before anything is sent", async () => {
    const clients = createClients();
    const readImpl = defaultRead(clients);
    clients.publicMock.readContract.mockImplementation(
      (args: { functionName: string }) =>
        args.functionName === "allowance"
          ? Promise.reject(new Error("rpc down"))
          : readImpl(args),
    );
    const depositor = createDepositor(clients);

    const error = await depositor
      .submitDeposit({ destination: { account: ACCOUNT }, asset: TOKEN, amount: "1" })
      .catch((e: unknown) => e);

    expect(error).toMatchObject({
      code: "RPC_ERROR",
      step: "approve",
      depositId: DEPOSIT_ID,
      nonceKey: 0n,
      nonce: 0n,
    });
    expect((error as ClearnetSdkError).txHash).toBeUndefined();
    expect(clients.walletMock.writeContract).not.toHaveBeenCalled();
  });

  it("throws TX_REVERTED with the approve step when the ERC-20 approval receipt fails", async () => {
    const clients = createClients({
      waitReceipt: receipt({ status: "reverted" }),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(APPROVAL_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: TOKEN,
        amount: "1",
      }),
    ).rejects.toMatchObject({
      code: "TX_REVERTED",
      txHash: APPROVAL_HASH,
      step: "approve",
      depositId: DEPOSIT_ID,
    });
    expect(clients.walletMock.writeContract).toHaveBeenCalledTimes(1);
  });

  it("throws RECEIPT_TIMEOUT with txHash after a submitted deposit times out", async () => {
    const clients = createClients({
      waitReceiptPromise: new Promise<TransactionReceipt>(() => undefined),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit(
        { destination: { account: ACCOUNT }, asset: EVM_NATIVE_ASSET, amount: "1" },
        { receiptTimeoutMs: 1 },
      ),
    ).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txHash: DEPOSIT_HASH,
      step: "deposit",
      depositId: DEPOSIT_ID,
    });
  });

  it("classifies an Invalid nonce revert as STALE_NONCE, carrying the depositId but no txHash", async () => {
    const clients = createClients();
    clients.walletMock.writeContract.mockRejectedValueOnce(
      new Error("execution reverted: Invalid nonce"),
    );
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
      }),
    ).rejects.toMatchObject({
      code: "STALE_NONCE",
      step: "deposit",
      depositId: DEPOSIT_ID,
      nonceKey: 0n,
      nonce: 0n,
      txHash: undefined,
    });
  });

  it("rejects invalid receipt timeout configuration", () => {
    const clients = createClients();

    expect(
      () =>
        new EvmVaultDepositor({
          publicClient: clients.publicClient,
          walletClient: clients.walletClient,
          walletAccount: ACCOUNT,
          custodyAddress: CUSTODY_ADDRESS,
          chainId: CHAIN_ID,
          receiptTimeoutMs: 0,
        }),
    ).toThrowError(
      expect.objectContaining({
        code: "RECEIPT_TIMEOUT",
        message: "receiptTimeoutMs must be a positive safe integer",
      }),
    );
  });

  it("validates deposit reference before chain checks or signing", async () => {
    const clients = createClients();
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT, ref: "invoice-1" as Hash },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
      }),
    ).rejects.toMatchObject({ code: "INVALID_REFERENCE" });
    expect(clients.walletMock.writeContract).not.toHaveBeenCalled();
  });

  it("rejects invalid input before signing", async () => {
    const clients = createClients();
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: "not-an-address" as Address,
        amount: "1",
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        destination: { account: "not-an-address" as Address },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: zeroAddress,
        amount: "1",
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: EVM_NATIVE_ASSET,
        amount: "0",
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    expect(clients.walletMock.writeContract).not.toHaveBeenCalled();
  });

  it("fails chain mismatch before signing", async () => {
    const clients = createClients({ publicChainId: 1 });
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: EVM_NATIVE_ASSET,
        amount: "1",
      }),
    ).rejects.toMatchObject({ code: "CHAIN_MISMATCH" });
    expect(clients.walletMock.writeContract).not.toHaveBeenCalled();
  });

  it("requires a wallet account in constructor", () => {
    const clients = createClients();

    expect(
      () =>
        new EvmVaultDepositor({
          publicClient: clients.publicClient,
          walletClient: clients.walletClient,
          walletAccount: undefined as unknown as Address,
          custodyAddress: CUSTODY_ADDRESS,
          chainId: CHAIN_ID,
        }),
    ).toThrow(ClearnetSdkError);
  });

  it("rejects a wallet client account that cannot sign for walletAccount", () => {
    const clients = createClients({ walletAccount: TOKEN });

    expect(
      () =>
        new EvmVaultDepositor({
          publicClient: clients.publicClient,
          walletClient: clients.walletClient,
          walletAccount: ACCOUNT,
          custodyAddress: CUSTODY_ADDRESS,
          chainId: CHAIN_ID,
        }),
    ).toThrow(ClearnetSdkError);
  });

  it("accepts equivalent wallet client and walletAccount address casing", () => {
    const lowerAccount = ACCOUNT.toLowerCase() as Address;
    const clients = createClients({ walletAccount: lowerAccount });

    expect(() => createDepositor(clients)).not.toThrow();
  });

  it("maps known successful receipts to confirmed or pending with inclusive confirmations", async () => {
    const clients = createClients({
      txReceipt: receipt({ status: "success", blockNumber: 10n }),
      headBlock: 10n,
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1),
    ).resolves.toBe("confirmed");
    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 2),
    ).resolves.toBe("pending");
    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1n << 80n),
    ).resolves.toBe("pending");
    expect(clients.publicMock.getBlockNumber).toHaveBeenCalledWith({
      cacheTime: 0,
    });
  });

  it("maps a mismatched depositId to absent even with a successful receipt", async () => {
    const clients = createClients({
      txReceipt: receipt({ status: "success", blockNumber: 10n }),
      headBlock: 10n,
    });
    const depositor = createDepositor(clients);
    const otherId = depositId(BigInt(CHAIN_ID), CUSTODY_ADDRESS, ACCOUNT, 1n);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, otherId, 1),
    ).resolves.toBe("absent");
  });

  it("maps a same-topic Deposited log from another contract to absent", async () => {
    const mined = receipt();
    const clients = createClients({
      txReceipt: {
        ...mined,
        logs: mined.logs.map((log) => ({
          ...log,
          address: "0x0000000000000000000000000000000000009999" as Address,
        })),
      } as TransactionReceipt,
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1),
    ).resolves.toBe("absent");
  });

  it("maps failed receipts to absent", async () => {
    const clients = createClients({
      txReceipt: receipt({ status: "reverted", blockNumber: 10n }),
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1n),
    ).resolves.toBe("absent");
  });

  it("maps missing receipt to pending when the transaction is known", async () => {
    const clients = createClients({
      txReceiptError: transactionNotFound("TransactionReceiptNotFoundError"),
      pendingTransactionKnown: true,
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1),
    ).resolves.toBe("pending");
  });

  it("maps missing receipt to absent when the transaction is unknown", async () => {
    const clients = createClients({
      txReceiptError: transactionNotFound("TransactionReceiptNotFoundError"),
      pendingTransactionKnown: false,
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1),
    ).resolves.toBe("absent");
  });

  it("throws RPC_ERROR with cause for real verify RPC failures", async () => {
    const rpcError = new Error("node offline");
    const clients = createClients({ txReceiptError: rpcError });
    const depositor = createDepositor(clients);

    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1),
    ).rejects.toMatchObject({ code: "RPC_ERROR", cause: rpcError });
  });

  it("validates the tx hash, deposit ID and confirmation depths", async () => {
    const depositor = createDepositor(createClients());

    await expect(
      depositor.chainDepositStatus("0x1234", DEPOSIT_ID, 1),
    ).rejects.toMatchObject({ code: "INVALID_TX_ID" });
    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, "0x1234", 1),
    ).rejects.toMatchObject({ code: "INVALID_DEPOSIT_ID" });
    await expect(
      depositor.chainDepositStatus(
        DEPOSIT_HASH,
        `${DEPOSIT_HASH}/0`,
        1,
      ),
    ).rejects.toMatchObject({ code: "INVALID_DEPOSIT_ID" });
    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, -1),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });
    await expect(
      depositor.chainDepositStatus(DEPOSIT_HASH, DEPOSIT_ID, 1.5),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });
  });
});

function createDepositor(clients: ClientMocks): EvmVaultDepositor {
  return new EvmVaultDepositor({
    publicClient: clients.publicClient,
    walletClient: clients.walletClient,
    walletAccount: ACCOUNT,
    custodyAddress: CUSTODY_ADDRESS,
    chainId: CHAIN_ID,
    nativeDecimals: 0,
  });
}

function createClients(options: {
  publicChainId?: number;
  walletChainId?: number;
  walletAccount?: Address;
  waitReceipt?: TransactionReceipt;
  waitReceiptPromise?: Promise<TransactionReceipt>;
  txReceipt?: TransactionReceipt;
  txReceiptError?: unknown;
  headBlock?: bigint;
  pendingTransactionKnown?: boolean;
  nonce?: bigint;
  allowance?: bigint;
  tokenDecimals?: number;
} = {}): ClientMocks {
  const publicMock = {
    getChainId: vi.fn().mockResolvedValue(options.publicChainId ?? CHAIN_ID),
    readContract: vi.fn().mockImplementation((args: { functionName: string }) => {
      switch (args.functionName) {
        case "getNonce":
          return Promise.resolve(options.nonce ?? 0n);
        case "allowance":
          return Promise.resolve(options.allowance ?? 0n);
        case "decimals":
          return Promise.resolve(options.tokenDecimals ?? 0);
        default:
          return Promise.resolve(0n);
      }
    }),
    waitForTransactionReceipt: vi
      .fn()
      .mockImplementation(() =>
        options.waitReceiptPromise === undefined
          ? Promise.resolve(options.waitReceipt ?? receipt())
          : options.waitReceiptPromise,
      ),
    getTransactionReceipt: vi.fn().mockImplementation(() => {
      if (options.txReceiptError !== undefined) {
        return Promise.reject(options.txReceiptError);
      }
      return Promise.resolve(options.txReceipt ?? receipt());
    }),
    getTransaction: vi.fn().mockImplementation(() => {
      if (options.pendingTransactionKnown === false) {
        return Promise.reject(transactionNotFound("TransactionNotFoundError"));
      }
      return Promise.resolve({ hash: DEPOSIT_HASH });
    }),
    getBlockNumber: vi.fn().mockResolvedValue(options.headBlock ?? 1n),
  };
  const walletMock = {
    ...(options.walletAccount !== undefined
      ? { account: { address: options.walletAccount } }
      : {}),
    getChainId: vi.fn().mockResolvedValue(options.walletChainId ?? CHAIN_ID),
    writeContract: vi.fn(),
  };

  return {
    publicClient: publicMock as unknown as PublicClient,
    walletClient: walletMock as unknown as WalletClient,
    publicMock,
    walletMock,
  };
}

function receipt(options: {
  status?: TransactionReceipt["status"];
  blockNumber?: bigint;
  asset?: Address;
  amount?: bigint;
  reference?: Hash;
  nonce?: bigint;
} = {}): TransactionReceipt {
  const topics = encodeEventTopics({
    abi: custodyAbi,
    eventName: "Deposited",
    args: {
      account: ACCOUNT,
      depositReference: options.reference ?? DEPOSIT_REFERENCE,
    },
  });
  const data = encodeAbiParameters(
    [
      { name: "depositor", type: "address" },
      { name: "asset", type: "address" },
      { name: "amount", type: "uint256" },
      { name: "nonce", type: "uint256" },
    ],
    [ACCOUNT, options.asset ?? zeroAddress, options.amount ?? 10n, options.nonce ?? 0n],
  );
  return {
    status: options.status ?? "success",
    blockNumber: options.blockNumber ?? 1n,
    transactionHash: DEPOSIT_HASH,
    logs: [
      {
        address: CUSTODY_ADDRESS,
        data,
        topics,
        transactionHash: DEPOSIT_HASH,
        logIndex: 0,
      },
    ],
  } as TransactionReceipt;
}

// defaultRead returns the readContract behaviour createClients installed, so
// a test can wrap it.
function defaultRead(
  clients: ClientMocks,
): (args: { functionName: string }) => Promise<unknown> {
  return clients.publicMock.readContract.getMockImplementation() as (args: {
    functionName: string;
  }) => Promise<unknown>;
}

function transactionNotFound(name: string): Error {
  const error = new Error("transaction not found");
  error.name = name;
  return error;
}

// Pins the exact accepted/rejected input set for
// requireDepositDestination's destination.account decoding, bare hex, an
// optional case-insensitive "0x" prefix, a yellow://.../user/<hex> URI's
// last segment, and surrounding whitespace. It also pins that an
// ADR-015 sub-account URI (yellow://.../user/<addr>/tag/<32-byte-ref>) is
// rejected rather than silently parsed as the trailing 32-byte reference.
describe("requireDepositDestination account parsing", () => {
  const addrHex = "000102030405060708090a0b0c0d0e0f10111213";
  const refHex =
    "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f";
  const want = `0x${addrHex}` as Address;

  it.each([
    ["bare hex", addrHex],
    ["0x prefix", `0x${addrHex}`],
    ["0X prefix", `0X${addrHex}`],
    ["uppercase hex", addrHex.toUpperCase()],
    ["whitespace padded", `  ${addrHex}  `],
    ["tab/newline padded", `\t${addrHex}\n`],
    ["yellow URI", `yellow://ynet/user/${addrHex}`],
    ["whitespace padded yellow URI", `  yellow://ynet/user/${addrHex}  `],
  ])("accepts %s", (_name, input) => {
    expect(requireDepositDestination({ account: input }).account).toBe(want);
  });

  it.each([
    ["empty string", ""],
    ["non-hex", "not-a-hex-address"],
    ["19 bytes", addrHex.slice(0, 38)],
    ["21 bytes", `${addrHex}00`],
    [
      "ADR-015 sub-account URI (last segment is the 32-byte reference)",
      `yellow://ynet/user/${addrHex}/tag/${refHex}`,
    ],
  ])("rejects %s", (_name, input) => {
    expect(() => requireDepositDestination({ account: input })).toThrowError(
      expect.objectContaining({ code: "INVALID_ADDRESS" }),
    );
  });
});
