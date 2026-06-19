import { describe, expect, expectTypeOf, it, vi } from "vitest";
import { zeroAddress, zeroHash } from "viem";
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
} from "../../../src/index.js";
import type {
  DepositStatus,
  EvmSubmitDepositInput,
  TxRef,
  VaultDepositor,
} from "../../../src/index.js";

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

interface ClientMocks {
  publicClient: PublicClient;
  walletClient: WalletClient;
  publicMock: {
    getChainId: ReturnType<typeof vi.fn>;
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
    expectTypeOf<TxRef>().toEqualTypeOf<{ hash: Hash; raw: string }>();
    expectTypeOf<DepositStatus>().toEqualTypeOf<
      "absent" | "pending" | "confirmed"
    >();
  });

  it("exports the native zero-address constant", () => {
    expect(EVM_NATIVE_ASSET).toBe(zeroAddress);
  });

  it("submits a native ETH deposit with matching value and returns the deposit hash", async () => {
    const clients = createClients();
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);

    const depositor = createDepositor(clients);
    const onSubmitted = vi.fn();
    const ref = await depositor.submitDeposit(
      {
        destination: { account: ACCOUNT, ref: DEPOSIT_REFERENCE },
        asset: zeroAddress,
        amount: 10n,
      },
      { onSubmitted },
    );

    expect(ref).toEqual({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH });
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);
    expect(clients.walletMock.writeContract).toHaveBeenCalledExactlyOnceWith(
      expect.objectContaining({
        address: CUSTODY_ADDRESS,
        functionName: "deposit",
        args: [ACCOUNT, zeroAddress, 10n, DEPOSIT_REFERENCE],
        value: 10n,
        account: ACCOUNT,
        chain: null,
      }),
    );
    expect(clients.publicMock.waitForTransactionReceipt).toHaveBeenCalledWith({
      hash: DEPOSIT_HASH,
    });
  });

  it("approves an exact ERC-20 amount before depositing and returns the deposit hash", async () => {
    const clients = createClients();
    clients.walletMock.writeContract
      .mockResolvedValueOnce(APPROVAL_HASH)
      .mockResolvedValueOnce(DEPOSIT_HASH);

    const depositor = createDepositor(clients);
    const onSubmitted = vi.fn();
    const ref = await depositor.submitDeposit(
      { destination: { account: ACCOUNT }, asset: TOKEN, amount: 25n },
      { onSubmitted },
    );

    expect(ref).toEqual({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH });
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);
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
        args: [ACCOUNT, TOKEN, 25n, zeroHash],
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

  it("throws TX_REVERTED with txRef when the deposit receipt fails", async () => {
    const clients = createClients({
      waitReceipt: receipt({ status: "reverted" }),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: zeroAddress,
        amount: 1n,
      }),
    ).rejects.toMatchObject({
      code: "TX_REVERTED",
      txRef: { hash: DEPOSIT_HASH, raw: DEPOSIT_HASH },
    });
  });

  it("throws TX_REVERTED when the ERC-20 approval receipt fails", async () => {
    const clients = createClients({
      waitReceipt: receipt({ status: "reverted" }),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(APPROVAL_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: TOKEN,
        amount: 1n,
      }),
    ).rejects.toMatchObject({
      code: "TX_REVERTED",
      txRef: { hash: APPROVAL_HASH, raw: APPROVAL_HASH },
    });
    expect(clients.walletMock.writeContract).toHaveBeenCalledTimes(1);
  });

  it("throws RECEIPT_TIMEOUT with txRef after a submitted deposit times out", async () => {
    const clients = createClients({
      waitReceiptPromise: new Promise<TransactionReceipt>(() => undefined),
    });
    clients.walletMock.writeContract.mockResolvedValueOnce(DEPOSIT_HASH);
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit(
        { destination: { account: ACCOUNT }, asset: zeroAddress, amount: 1n },
        { receiptTimeoutMs: 1 },
      ),
    ).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txRef: { hash: DEPOSIT_HASH, raw: DEPOSIT_HASH },
    });
  });

  it("validates deposit reference before chain checks or signing", async () => {
    const clients = createClients();
    const depositor = createDepositor(clients);

    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT, ref: "invoice-1" as Hash },
        asset: zeroAddress,
        amount: 1n,
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
        amount: 1n,
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        destination: { account: "not-an-address" as Address },
        asset: zeroAddress,
        amount: 1n,
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        destination: { account: ACCOUNT },
        asset: zeroAddress,
        amount: 0n,
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
        asset: zeroAddress,
        amount: 1n,
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
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 1),
    ).resolves.toBe("confirmed");
    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 2),
    ).resolves.toBe("pending");
    expect(clients.publicMock.getBlockNumber).toHaveBeenCalledWith({
      cacheTime: 0,
    });
  });

  it("maps failed receipts to absent", async () => {
    const clients = createClients({
      txReceipt: receipt({ status: "reverted", blockNumber: 10n }),
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 1n),
    ).resolves.toBe("absent");
  });

  it("maps missing receipt to pending when the transaction is known", async () => {
    const clients = createClients({
      txReceiptError: transactionNotFound("TransactionReceiptNotFoundError"),
      pendingTransactionKnown: true,
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 1),
    ).resolves.toBe("pending");
  });

  it("maps missing receipt to absent when the transaction is unknown", async () => {
    const clients = createClients({
      txReceiptError: transactionNotFound("TransactionReceiptNotFoundError"),
      pendingTransactionKnown: false,
    });
    const depositor = createDepositor(clients);

    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 1),
    ).resolves.toBe("absent");
  });

  it("throws RPC_ERROR with cause for real verify RPC failures", async () => {
    const rpcError = new Error("node offline");
    const clients = createClients({ txReceiptError: rpcError });
    const depositor = createDepositor(clients);

    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 1),
    ).rejects.toMatchObject({ code: "RPC_ERROR", cause: rpcError });
  });

  it("validates tx refs and confirmation depths", async () => {
    const depositor = createDepositor(createClients());

    await expect(
      depositor.verifyDeposit({ hash: "0x1234" as Hash, raw: "0x1234" }, 1),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, -1),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });
    await expect(
      depositor.verifyDeposit({ hash: DEPOSIT_HASH, raw: DEPOSIT_HASH }, 1.5),
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
} = {}): ClientMocks {
  const publicMock = {
    getChainId: vi.fn().mockResolvedValue(options.publicChainId ?? CHAIN_ID),
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
} = {}): TransactionReceipt {
  return {
    status: options.status ?? "success",
    blockNumber: options.blockNumber ?? 1n,
  } as TransactionReceipt;
}

function transactionNotFound(name: string): Error {
  const error = new Error("transaction not found");
  error.name = name;
  return error;
}
