import { zeroAddress } from "viem";
import type { Address, Hash, TransactionReceipt } from "viem";

import { ClearnetSdkError } from "../../core/errors.js";
import { normalizeReceiptTimeoutMs } from "../../core/validation.js";
import type {
  DepositStatus,
  EvmDepositorConfig,
  EvmSubmitDepositInput,
  SubmitDepositOptions,
  SubmitDepositResult,
  VaultDepositor,
} from "../../core/types.js";
import { decimalToBaseUnits } from "../amounts.js";
import { custodyAbi, erc20Abi } from "./abi.js";
import { DEFAULT_RECEIPT_TIMEOUT_MS } from "./constants.js";
import { hasDepositedLog } from "./depositEvents.js";
import {
  depositId as computeDepositId,
  requireDepositId,
  requireNonceKey,
} from "./depositId.js";
import {
  isTransactionNotFound,
  requireDepositDestination,
  normalizeMinConfirmations,
  normalizeNativeDecimals,
  requireAddress,
  requireAsset,
  requireChainId,
  requireTxHash,
  requireWalletAccount,
  walletAccountAddress,
  type ValidatedDepositDestination,
} from "./validation.js";

type AsyncValidation = Promise<ClearnetSdkError | undefined>;

// DepositIdentity is what is known about a deposit once its nonce has been
// read; every error raised after that point carries these fields.
interface DepositIdentity {
  depositor: Address;
  nonceKey: bigint;
  nonce: bigint;
  depositId: string;
}

function identityFields(identity: DepositIdentity): {
  nonceKey: bigint;
  nonce: bigint;
  depositId: string;
} {
  return {
    nonceKey: identity.nonceKey,
    nonce: identity.nonce,
    depositId: identity.depositId,
  };
}

// A stale nonce (already consumed by submission time, e.g. a lagging
// load-balanced RPC node) is detected from Custody's revert reason or, for a
// mined revert without a reason, a fresh getNonce read. Liveness only: a fresh
// submitDeposit call reads the current nonce.
const STALE_NONCE_REVERT_TEXT = "Invalid nonce";

function isStaleNonceRevert(error: unknown): boolean {
  return error instanceof Error && error.message.includes(STALE_NONCE_REVERT_TEXT);
}

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

  // Do not call concurrently for the same (depositor, nonceKey): both race one
  // on-chain counter and one reverts. After an error, check the thrown
  // ClearnetSdkError's step/txHash/depositId/nonceKey/nonce before retrying; a
  // blind retry reads the next nonce and can create a second real deposit.
  async submitDeposit(
    input: EvmSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<SubmitDepositResult> {
    const destination = requireDepositDestination(input.destination);
    const asset = requireAsset(input.asset);
    const key = requireNonceKey(input.nonceKey);
    await this.ensureWriteChain();

    const amount =
      asset === ""
        ? decimalToBaseUnits(input.amount, this.nativeDecimals)
        : decimalToBaseUnits(input.amount, await this.assetDecimals(asset));

    const depositorAddr = walletAccountAddress(this.config.walletAccount);
    // Read at `latest`, never `pending` (that would turn a blind retry into a
    // second deposit), before any approve.
    const nonce = await this.readNonce(depositorAddr, key);
    const identity: DepositIdentity = {
      depositor: depositorAddr,
      nonceKey: key,
      nonce,
      depositId: computeDepositId(
        BigInt(this.config.chainId),
        this.config.custodyAddress,
        depositorAddr,
        nonce,
      ),
    };

    if (asset === "") {
      return this.submitNativeDeposit(destination, amount, identity, options);
    }
    return this.submitErc20Deposit(destination, asset, amount, identity, options);
  }

  // Fetches the receipt at txHash and finds the vault's Deposited log whose
  // depositor and nonce reproduce depositId. Pure on-chain read: custody's
  // crediting rules are not applied, so "confirmed" does not guarantee a credit.
  async chainDepositStatus(
    txHash: string,
    depositId: string,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const hash = requireTxHash(txHash);
    const wantId = requireDepositId(depositId).toLowerCase();
    const minConf = normalizeMinConfirmations(minConfirmations);
    await this.ensurePublicChain();

    let receipt: TransactionReceipt;
    try {
      receipt = await this.config.publicClient.getTransactionReceipt({
        hash,
      });
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
    const chainId = BigInt(this.config.chainId);
    if (!hasDepositedLog(receipt, this.config.custodyAddress, chainId, wantId)) {
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

  private async readNonce(depositor: Address, key: bigint): Promise<bigint> {
    try {
      return await this.config.publicClient.readContract({
        address: this.config.custodyAddress,
        abi: custodyAbi,
        functionName: "getNonce",
        args: [depositor, key],
      });
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "evm: get nonce", {
        cause: error,
      });
    }
  }

  private async submitNativeDeposit(
    destination: ValidatedDepositDestination,
    amount: bigint,
    identity: DepositIdentity,
    options: SubmitDepositOptions,
  ): Promise<SubmitDepositResult> {
    const hash = await this.writeContractRpc(
      () =>
        this.config.walletClient.writeContract({
          address: this.config.custodyAddress,
          abi: custodyAbi,
          functionName: "deposit",
          args: [
            destination.account,
            zeroAddress,
            amount,
            destination.ref,
            identity.nonce,
          ],
          value: amount,
          account: this.config.walletAccount,
          chain: this.config.walletClient.chain ?? null,
        }),
      "deposit",
      identity,
    );
    await this.waitForDeposit(hash, identity, options);
    const result: SubmitDepositResult = { txHash: hash, depositId: identity.depositId };
    options.onSubmitted?.(result);
    return result;
  }

  private async submitErc20Deposit(
    destination: ValidatedDepositDestination,
    asset: Address,
    amount: bigint,
    identity: DepositIdentity,
    options: SubmitDepositOptions,
  ): Promise<SubmitDepositResult> {
    // Approve exactly amount unless the allowance already covers it.
    // Known gap: a nonzero allowance below amount still sends a
    // nonzero-to-nonzero approve, which USDT-style tokens reject; reset the
    // allowance to zero first.
    const currentAllowance = await this.readAllowance(asset, identity);
    if (currentAllowance < amount) {
      const approvalHash = await this.writeContractRpc(
        () =>
          this.config.walletClient.writeContract({
            address: asset,
            abi: erc20Abi,
            functionName: "approve",
            args: [this.config.custodyAddress, amount],
            account: this.config.walletAccount,
            chain: this.config.walletClient.chain ?? null,
          }),
        "approve",
        identity,
      );
      await this.waitForSuccessfulReceipt(approvalHash, "approve", identity, options);
    }

    const depositHash = await this.writeContractRpc(
      () =>
        this.config.walletClient.writeContract({
          address: this.config.custodyAddress,
          abi: custodyAbi,
          functionName: "deposit",
          args: [destination.account, asset, amount, destination.ref, identity.nonce],
          account: this.config.walletAccount,
          chain: this.config.walletClient.chain ?? null,
        }),
      "deposit",
      identity,
    );
    await this.waitForDeposit(depositHash, identity, options);
    const result: SubmitDepositResult = {
      txHash: depositHash,
      depositId: identity.depositId,
    };
    options.onSubmitted?.(result);
    return result;
  }

  private async readAllowance(
    asset: Address,
    identity: DepositIdentity,
  ): Promise<bigint> {
    try {
      return await this.config.publicClient.readContract({
        address: asset,
        abi: erc20Abi,
        functionName: "allowance",
        args: [identity.depositor, this.config.custodyAddress],
      });
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "evm: read allowance", {
        step: "approve",
        ...identityFields(identity),
        cause: error,
      });
    }
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

  // writeContractRpc wraps a write call so its error carries step, nonceKey,
  // nonce and depositId. viem's writeContract signs and sends atomically, so
  // no txHash is available when it throws: read getNonce(depositor, nonceKey)
  // at `latest`; a value above the error's nonce means that deposit landed,
  // otherwise a retry is safe.
  private async writeContractRpc(
    write: () => Promise<Hash>,
    step: "approve" | "deposit",
    identity: DepositIdentity,
  ): Promise<Hash> {
    try {
      return await write();
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      if (step === "deposit" && isStaleNonceRevert(error)) {
        throw new ClearnetSdkError("STALE_NONCE", `evm: write contract (${step})`, {
          step,
          ...identityFields(identity),
          cause: error,
        });
      }
      throw new ClearnetSdkError("RPC_ERROR", `evm: write contract (${step})`, {
        step,
        ...identityFields(identity),
        cause: error,
      });
    }
  }

  // waitForDeposit waits for the deposit transaction, then requires the
  // vault's Deposited log for identity.depositId. A mined revert is
  // STALE_NONCE when a fresh getNonce shows the nonce was consumed, else
  // TX_REVERTED.
  private async waitForDeposit(
    hash: Hash,
    identity: DepositIdentity,
    options: SubmitDepositOptions,
  ): Promise<void> {
    let receipt: TransactionReceipt;
    try {
      receipt = await this.waitForSuccessfulReceipt(hash, "deposit", identity, options);
    } catch (error) {
      if (
        error instanceof ClearnetSdkError &&
        error.code === "TX_REVERTED" &&
        (await this.nonceConsumed(identity))
      ) {
        throw new ClearnetSdkError(
          "STALE_NONCE",
          `evm: deposit reverted; nonce already consumed (tx=${hash})`,
          { txHash: hash, step: "deposit", ...identityFields(identity), cause: error },
        );
      }
      throw error;
    }
    const chainId = BigInt(this.config.chainId);
    if (
      !hasDepositedLog(
        receipt,
        this.config.custodyAddress,
        chainId,
        identity.depositId.toLowerCase(),
      )
    ) {
      throw new ClearnetSdkError(
        "DEPOSIT_EVENT_NOT_FOUND",
        `evm: deposited event not found (tx=${hash})`,
        { txHash: hash, step: "deposit", ...identityFields(identity) },
      );
    }
  }

  // nonceConsumed reports whether the vault's counter for identity's key has
  // moved past identity.nonce. A failed read counts as not consumed, so the
  // original error is kept.
  private async nonceConsumed(identity: DepositIdentity): Promise<boolean> {
    try {
      const current = await this.readNonce(identity.depositor, identity.nonceKey);
      return current > identity.nonce;
    } catch {
      return false;
    }
  }

  private async waitForSuccessfulReceipt(
    hash: Hash,
    step: "approve" | "deposit",
    identity: DepositIdentity,
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
        hash,
        step,
        identity,
      );
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", `evm: wait receipt (${step})`, {
        cause: error,
        txHash: hash,
        step,
        ...identityFields(identity),
      });
    }

    if (receipt.status !== "success") {
      throw new ClearnetSdkError(
        "TX_REVERTED",
        `transaction reverted (tx=${hash})`,
        { txHash: hash, step, ...identityFields(identity) },
      );
    }
    return receipt;
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
  hash: Hash,
  step: "approve" | "deposit",
  identity: DepositIdentity,
): Promise<TransactionReceipt> {
  if (signal?.aborted) {
    throw new ClearnetSdkError("RECEIPT_TIMEOUT", "receipt wait aborted", {
      txHash: hash,
      step,
      ...identityFields(identity),
    });
  }

  let timeoutId: ReturnType<typeof setTimeout> | undefined;
  let abortHandler: (() => void) | undefined;

  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutId = setTimeout(() => {
      reject(
        new ClearnetSdkError(
          "RECEIPT_TIMEOUT",
          `receipt wait timed out after ${timeoutMs}ms`,
          { txHash: hash, step, ...identityFields(identity) },
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
              new ClearnetSdkError("RECEIPT_TIMEOUT", "receipt wait aborted", {
                txHash: hash,
                step,
                ...identityFields(identity),
              }),
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
  return normalizeReceiptTimeoutMs(timeoutMs);
}
