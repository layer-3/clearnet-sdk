import { parseEventLogs, zeroAddress } from "viem";
import type { Address, Hash, TransactionReceipt } from "viem";

import { ClearnetSdkError } from "../../core/errors.js";
import type {
  DepositStatus,
  EvmDepositorConfig,
  EvmSubmitDepositInput,
  SubmitDepositOptions,
  VaultDepositor,
} from "../../core/types.js";
import { decimalToBaseUnits } from "../amounts.js";
import { custodyAbi, erc20Abi } from "./abi.js";
import { DEFAULT_RECEIPT_TIMEOUT_MS } from "./constants.js";
import {
  isTransactionNotFound,
  requireDepositDestination,
  normalizeMinConfirmations,
  normalizeNativeDecimals,
  requireAddress,
  requireAsset,
  requireChainId,
  requireTxID,
  requireWalletAccount,
  walletAccountAddress,
  type ValidatedDepositDestination,
} from "./validation.js";

type AsyncValidation = Promise<ClearnetSdkError | undefined>;

export class EvmVaultDepositor implements VaultDepositor<EvmSubmitDepositInput> {
  private readonly config: EvmDepositorConfig;
  private readonly nativeDecimals: number;
  private readonly tokenDecimals = new Map<string, number>();
  private readonly initialPublicChainValidation: AsyncValidation;
  private readonly initialWriteChainValidation: AsyncValidation;

  constructor(config: EvmDepositorConfig) {
    requireAddress(config.custodyAddress, "custodyAddress");
    const walletAccount = requireWalletAccount(config.walletAccount);
    const clientAccount = config.walletClient.account;
    if (
      clientAccount !== undefined &&
      walletAccountAddress(clientAccount).toLowerCase() !==
        walletAccountAddress(walletAccount).toLowerCase()
    ) {
      throw new ClearnetSdkError(
        "MISSING_WALLET_ACCOUNT",
        "walletClient account does not match walletAccount",
      );
    }
    requireChainId(config.chainId);
    if (config.receiptTimeoutMs !== undefined) {
      requireReceiptTimeout(config.receiptTimeoutMs);
    }
    this.nativeDecimals = normalizeNativeDecimals(config.nativeDecimals);
    this.config = { ...config };
    this.initialPublicChainValidation = captureValidation(
      this.checkPublicChain(),
    );
    this.initialWriteChainValidation = captureValidation(
      this.checkInitialWriteChain(),
    );
  }

  async submitDeposit(
    input: EvmSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<string> {
    const destination = requireDepositDestination(input.destination);
    const asset = requireAsset(input.asset);
    await this.ensureWriteChain();

    if (asset === "") {
      const amount = decimalToBaseUnits(input.amount, this.nativeDecimals);
      return this.submitNativeDeposit(destination, amount, options);
    }
    const amount = decimalToBaseUnits(
      input.amount,
      await this.assetDecimals(asset),
    );
    return this.submitErc20Deposit(destination, asset, amount, options);
  }

  async verifyDeposit(
    txID: string,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const parsedTxID = requireTxID(txID);
    const minConf = normalizeMinConfirmations(minConfirmations);
    await this.ensurePublicChain();

    let receipt: TransactionReceipt;
    try {
      receipt = await this.config.publicClient.getTransactionReceipt({
        hash: parsedTxID.hash,
      });
    } catch (error) {
      if (!isTransactionNotFound(error)) {
        throw new ClearnetSdkError("RPC_ERROR", "evm: tx receipt", {
          cause: error,
        });
      }
      return this.pendingOrAbsent(parsedTxID.hash);
    }

    if (receipt.status !== "success") {
      return "absent";
    }
    if (
      parsedTxID.logIndex !== undefined &&
      !hasDepositedLog(receipt, this.config.custodyAddress, parsedTxID.logIndex)
    ) {
      return "absent";
    }

    let headBlockNumber: bigint;
    try {
      headBlockNumber = await this.config.publicClient.getBlockNumber({
        cacheTime: 0,
      });
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "evm: block number", {
        cause: error,
      });
    }

    const confirmations =
      headBlockNumber >= receipt.blockNumber
        ? headBlockNumber - receipt.blockNumber + 1n
        : 0n;
    return confirmations >= minConf ? "confirmed" : "pending";
  }

  private async submitNativeDeposit(
    destination: ValidatedDepositDestination,
    amount: bigint,
    options: SubmitDepositOptions,
  ): Promise<string> {
    const hash = await this.writeContractRpc(() =>
      this.config.walletClient.writeContract({
        address: this.config.custodyAddress,
        abi: custodyAbi,
        functionName: "deposit",
        args: [destination.account, zeroAddress, amount, destination.ref],
        value: amount,
        account: this.config.walletAccount,
        chain: this.config.walletClient.chain ?? null,
      }),
    );
    const receipt = await this.waitForSuccessfulReceipt(hash, hash, options);
    const txID = depositTxID(
      receipt,
      this.config.custodyAddress,
      destination.account,
      zeroAddress,
      amount,
      destination.ref,
    );
    options.onSubmitted?.(txID);
    return txID;
  }

  private async submitErc20Deposit(
    destination: ValidatedDepositDestination,
    asset: Address,
    amount: bigint,
    options: SubmitDepositOptions,
  ): Promise<string> {
    const approvalHash = await this.writeContractRpc(() =>
      this.config.walletClient.writeContract({
        address: asset,
        abi: erc20Abi,
        functionName: "approve",
        args: [this.config.custodyAddress, amount],
        account: this.config.walletAccount,
        chain: this.config.walletClient.chain ?? null,
      }),
    );
    await this.waitForSuccessfulReceipt(approvalHash, approvalHash, options);

    const depositHash = await this.writeContractRpc(() =>
      this.config.walletClient.writeContract({
        address: this.config.custodyAddress,
        abi: custodyAbi,
        functionName: "deposit",
        args: [destination.account, asset, amount, destination.ref],
        account: this.config.walletAccount,
        chain: this.config.walletClient.chain ?? null,
      }),
    );
    const receipt = await this.waitForSuccessfulReceipt(depositHash, depositHash, options);
    const txID = depositTxID(
      receipt,
      this.config.custodyAddress,
      destination.account,
      asset,
      amount,
      destination.ref,
    );
    options.onSubmitted?.(txID);
    return txID;
  }

  private async assetDecimals(asset: Address): Promise<number> {
    const key = asset.toLowerCase();
    const cached = this.tokenDecimals.get(key);
    if (cached !== undefined) {
      return cached;
    }
    try {
      const decimals = await this.config.publicClient.readContract({
        address: asset,
        abi: erc20Abi,
        functionName: "decimals",
      });
      this.tokenDecimals.set(key, decimals);
      return decimals;
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "evm: token decimals", {
        cause: error,
      });
    }
  }

  private async pendingOrAbsent(hash: Hash): Promise<DepositStatus> {
    try {
      await this.config.publicClient.getTransaction({ hash });
      return "pending";
    } catch (error) {
      if (isTransactionNotFound(error)) {
        return "absent";
      }
      throw new ClearnetSdkError("RPC_ERROR", "evm: tx lookup", {
        cause: error,
      });
    }
  }

  private async ensureWriteChain(): Promise<void> {
    await throwValidationError(this.initialWriteChainValidation);
    await this.checkPublicChain();
    await this.checkWalletChain();
  }

  private async ensurePublicChain(): Promise<void> {
    await throwValidationError(this.initialPublicChainValidation);
    await this.checkPublicChain();
  }

  private async checkInitialWriteChain(): Promise<void> {
    await throwValidationError(this.initialPublicChainValidation);
    await this.checkWalletChain();
  }

  private async checkWalletChain(): Promise<void> {
    const walletClient = this.config.walletClient as {
      getChainId?: () => Promise<number>;
    };
    if (typeof walletClient.getChainId !== "function") {
      return;
    }
    let walletChainId: number;
    try {
      walletChainId = await walletClient.getChainId();
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "evm: wallet chain id", {
        cause: error,
      });
    }
    if (walletChainId !== this.config.chainId) {
      throw new ClearnetSdkError(
        "CHAIN_MISMATCH",
        `wallet chain ${walletChainId} does not match expected chain ${this.config.chainId}`,
      );
    }
  }

  private async checkPublicChain(): Promise<void> {
    let publicChainId: number;
    try {
      publicChainId = await this.config.publicClient.getChainId();
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "evm: public chain id", {
        cause: error,
      });
    }
    if (publicChainId !== this.config.chainId) {
      throw new ClearnetSdkError(
        "CHAIN_MISMATCH",
        `public chain ${publicChainId} does not match expected chain ${this.config.chainId}`,
      );
    }
  }

  private async writeContractRpc(write: () => Promise<Hash>): Promise<Hash> {
    try {
      return await write();
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "evm: write contract", {
        cause: error,
      });
    }
  }

  private async waitForSuccessfulReceipt(
    hash: Hash,
    txID: string | undefined,
    options: SubmitDepositOptions,
  ): Promise<TransactionReceipt> {
    const timeoutMs = requireReceiptTimeout(
      options.receiptTimeoutMs ?? this.config.receiptTimeoutMs ?? DEFAULT_RECEIPT_TIMEOUT_MS,
    );
    let receipt: TransactionReceipt;
    try {
      receipt = await waitWithControls(
        () => this.config.publicClient.waitForTransactionReceipt({ hash }),
        timeoutMs,
        options.signal,
        txID,
      );
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "evm: wait receipt", {
        cause: error,
        ...(txID !== undefined ? { txID } : {}),
      });
    }

    if (receipt.status !== "success") {
      throw new ClearnetSdkError(
        "TX_REVERTED",
        `transaction reverted (tx=${hash})`,
        txID !== undefined ? { txID } : {},
      );
    }
    return receipt;
  }
}

function depositTxID(
  receipt: TransactionReceipt,
  custodyAddress: Address,
  account: Address,
  asset: Address,
  amount: bigint,
  reference: Hash,
): string {
  const log = parseEventLogs({
    abi: custodyAbi,
    eventName: "Deposited",
    logs: [...receipt.logs],
  }).find(
    (candidate) =>
      candidate.address.toLowerCase() === custodyAddress.toLowerCase() &&
      candidate.args.account.toLowerCase() === account.toLowerCase() &&
      candidate.args.depositReference.toLowerCase() === reference.toLowerCase() &&
      candidate.args.asset.toLowerCase() === asset.toLowerCase() &&
      candidate.args.amount === amount,
  );
  if (log === undefined) {
    throw new ClearnetSdkError("RPC_ERROR", "evm: deposited event not found", {
      txID: receipt.transactionHash,
    });
  }
  return `${log.transactionHash}/${log.logIndex}`;
}

function hasDepositedLog(
  receipt: TransactionReceipt,
  custodyAddress: Address,
  logIndex: number,
): boolean {
  return parseEventLogs({
    abi: custodyAbi,
    eventName: "Deposited",
    logs: [...receipt.logs],
  }).some(
    (log) =>
      log.address.toLowerCase() === custodyAddress.toLowerCase() &&
      log.logIndex === logIndex,
  );
}

function captureValidation(validation: Promise<void>): AsyncValidation {
  return validation.then(
    () => undefined,
    (error: unknown) =>
      error instanceof ClearnetSdkError
        ? error
        : new ClearnetSdkError("RPC_ERROR", "evm: validation", {
            cause: error,
          }),
  );
}

async function throwValidationError(validation: AsyncValidation): Promise<void> {
  const error = await validation;
  if (error !== undefined) {
    throw error;
  }
}

async function waitWithControls(
  wait: () => Promise<TransactionReceipt>,
  timeoutMs: number,
  signal: AbortSignal | undefined,
  ref: string | undefined,
): Promise<TransactionReceipt> {
  if (signal?.aborted) {
    throw new ClearnetSdkError(
      "RECEIPT_TIMEOUT",
      "receipt wait aborted",
      ref !== undefined ? { txID: ref } : {},
    );
  }

  let timeoutId: ReturnType<typeof setTimeout> | undefined;
  let abortHandler: (() => void) | undefined;

  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutId = setTimeout(() => {
      reject(
        new ClearnetSdkError(
          "RECEIPT_TIMEOUT",
          `receipt wait timed out after ${timeoutMs}ms`,
          ref !== undefined ? { txID: ref } : {},
        ),
      );
    }, timeoutMs);
  });

  const abortPromise =
    signal === undefined
      ? undefined
      : new Promise<never>((_, reject) => {
          abortHandler = () => {
            reject(
              new ClearnetSdkError(
                "RECEIPT_TIMEOUT",
                "receipt wait aborted",
                ref !== undefined ? { txID: ref } : {},
              ),
            );
          };
          signal.addEventListener("abort", abortHandler, { once: true });
        });

  try {
    return await Promise.race(
      abortPromise === undefined
        ? [wait(), timeoutPromise]
        : [wait(), timeoutPromise, abortPromise],
    );
  } finally {
    if (timeoutId !== undefined) {
      clearTimeout(timeoutId);
    }
    if (signal !== undefined && abortHandler !== undefined) {
      signal.removeEventListener("abort", abortHandler);
    }
  }
}

function requireReceiptTimeout(timeoutMs: number): number {
  if (!Number.isSafeInteger(timeoutMs) || timeoutMs <= 0) {
    throw new ClearnetSdkError(
      "RECEIPT_TIMEOUT",
      "receiptTimeoutMs must be a positive safe integer",
    );
  }
  return timeoutMs;
}
