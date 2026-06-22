import type { Address, Hash, TransactionReceipt } from "viem";

import { ClearnetSdkError } from "../../core/errors.js";
import type {
  DepositStatus,
  EvmDepositorConfig,
  EvmSubmitDepositInput,
  SubmitDepositOptions,
  TxRef,
  VaultDepositor,
} from "../../core/types.js";
import { custodyAbi, erc20Abi } from "./abi.js";
import {
  DEFAULT_RECEIPT_TIMEOUT_MS,
  EVM_NATIVE_ASSET,
} from "./constants.js";
import {
  isTransactionNotFound,
  requireDepositDestination,
  normalizeMinConfirmations,
  requireAddress,
  requireAmount,
  requireChainId,
  requireTxRef,
  requireWalletAccount,
  txRef,
  walletAccountAddress,
  type ValidatedDepositDestination,
} from "./validation.js";

type AsyncValidation = Promise<ClearnetSdkError | undefined>;

export class EvmVaultDepositor implements VaultDepositor<EvmSubmitDepositInput> {
  private readonly config: EvmDepositorConfig;
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
  ): Promise<TxRef> {
    const destination = requireDepositDestination(input.destination);
    const asset = requireAddress(input.asset, "asset");
    const amount = requireAmount(input.amount);
    await this.ensureWriteChain();

    if (asset === EVM_NATIVE_ASSET) {
      return this.submitNativeDeposit(destination, amount, options);
    }
    return this.submitErc20Deposit(destination, asset, amount, options);
  }

  async verifyDeposit(
    ref: TxRef,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const hash = requireTxRef(ref);
    const minConf = normalizeMinConfirmations(minConfirmations);
    await this.ensurePublicChain();

    let receipt: TransactionReceipt;
    try {
      receipt = await this.config.publicClient.getTransactionReceipt({ hash });
    } catch (error) {
      if (!isTransactionNotFound(error)) {
        throw new ClearnetSdkError("RPC_ERROR", "evm: tx receipt", {
          cause: error,
        });
      }
      return this.pendingOrAbsent(hash);
    }

    if (receipt.status !== "success") {
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
  ): Promise<TxRef> {
    const hash = await this.writeContractRpc(() =>
      this.config.walletClient.writeContract({
        address: this.config.custodyAddress,
        abi: custodyAbi,
        functionName: "deposit",
        args: [destination.account, EVM_NATIVE_ASSET, amount, destination.ref],
        value: amount,
        account: this.config.walletAccount,
        chain: this.config.walletClient.chain ?? null,
      }),
    );
    const ref = txRef(hash);
    options.onSubmitted?.(ref);
    await this.waitForSuccessfulReceipt(hash, ref, options);
    return ref;
  }

  private async submitErc20Deposit(
    destination: ValidatedDepositDestination,
    asset: Address,
    amount: bigint,
    options: SubmitDepositOptions,
  ): Promise<TxRef> {
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
    await this.waitForSuccessfulReceipt(approvalHash, txRef(approvalHash), options);

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
    const ref = txRef(depositHash);
    options.onSubmitted?.(ref);
    await this.waitForSuccessfulReceipt(depositHash, ref, options);
    return ref;
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
    ref: TxRef | undefined,
    options: SubmitDepositOptions,
  ): Promise<void> {
    const timeoutMs = requireReceiptTimeout(
      options.receiptTimeoutMs ?? this.config.receiptTimeoutMs ?? DEFAULT_RECEIPT_TIMEOUT_MS,
    );
    let receipt: TransactionReceipt;
    try {
      receipt = await waitWithControls(
        () => this.config.publicClient.waitForTransactionReceipt({ hash }),
        timeoutMs,
        options.signal,
        ref,
      );
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "evm: wait receipt", {
        cause: error,
        ...(ref !== undefined ? { txRef: ref } : {}),
      });
    }

    if (receipt.status !== "success") {
      throw new ClearnetSdkError(
        "TX_REVERTED",
        `transaction reverted (tx=${hash})`,
        ref !== undefined ? { txRef: ref } : {},
      );
    }
  }
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
  ref: TxRef | undefined,
): Promise<TransactionReceipt> {
  if (signal?.aborted) {
    throw new ClearnetSdkError(
      "RECEIPT_TIMEOUT",
      "receipt wait aborted",
      ref !== undefined ? { txRef: ref } : {},
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
          ref !== undefined ? { txRef: ref } : {},
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
                ref !== undefined ? { txRef: ref } : {},
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
