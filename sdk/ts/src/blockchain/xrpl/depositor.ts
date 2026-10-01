import { Client } from "xrpl";
import type { Payment, TxResponse } from "xrpl";

import { ClearnetSdkError } from "../../core/errors.js";
import type {
  DepositStatus,
  SubmitDepositOptions,
  SubmitDepositResult,
  VaultDepositor,
} from "../../core/types.js";
import { XRPL_MEMO_TYPE } from "./constants.js";
import { xrplDepositId } from "./depositId.js";
import { encodeClearnetMemo } from "./encoding.js";
import type {
  XrplDepositorConfig,
  XrplSigner,
  XrplSubmitDepositInput,
} from "./types.js";
import {
  normalizeFeeDrops,
  normalizeIssuedAssetDecimals,
  normalizeMinConfirmations,
  normalizeTxHash,
  requireClassicAddress,
  requireClearnetAccount,
  requireDepositDestination,
  requireReference,
  requireRpcUrl,
  requireSigner,
  requireTxID,
  resolveAmount,
} from "./validation.js";

const HEX_PATTERN = /^(?:[a-fA-F0-9]{2})*$/;

export class XrplVaultDepositor
  implements VaultDepositor<XrplSubmitDepositInput>
{
  private readonly signer: XrplSigner;
  private readonly vaultAddress: string;
  private readonly maxFeeDrops: bigint | undefined;
  private readonly issuedAssetDecimals: ReadonlyMap<string, number>;
  private readonly client: Client;
  private connecting: Promise<void> | undefined;

  constructor(config: XrplDepositorConfig) {
    this.signer = requireSigner(config.signer);
    this.vaultAddress = requireClassicAddress(config.vaultAddress, "vaultAddress");
    this.maxFeeDrops =
      config.maxFeeDrops === undefined
        ? undefined
        : normalizeFeeDrops(config.maxFeeDrops);
    this.issuedAssetDecimals = normalizeIssuedAssetDecimals(
      config.issuedAssetDecimals,
    );
    this.client = new Client(requireRpcUrl(config.rpcUrl));
  }

  // txHash is the upper-case transaction hash; depositId is
  // xrplDepositId(txHash).
  async submitDeposit(
    input: XrplSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<SubmitDepositResult> {
    const submitOptions = requireSubmitDepositOptions(options);
    const prepared = await this.prepareDeposit(input);
    const signed = await this.sign(prepared);
    const txID = normalizeTxHash(signed.hash);
    await this.submit(signed.txBlob, txID);
    const result: SubmitDepositResult = {
      txHash: txID,
      depositId: xrplDepositId(txID),
    };
    submitOptions.onSubmitted?.(result);
    return result;
  }

  /** Builds and autofills the exact unsigned custody payment. */
  async prepareDeposit(input: XrplSubmitDepositInput): Promise<Payment> {
    const fields =
      input && typeof input === "object"
        ? (input as Partial<XrplSubmitDepositInput>)
        : {};
    const destination = requireDepositDestination(fields.destination);
    const account = requireClearnetAccount(destination.account);
    const reference = requireReference(destination.ref);
    const amount = resolveAmount(
      fields.asset,
      fields.amount,
      this.issuedAssetDecimals,
    );
    const payment: Payment = {
      TransactionType: "Payment",
      Account: this.signer.classicAddress,
      Destination: this.vaultAddress,
      Amount: amount.amount,
      Memos: encodeClearnetMemo(account, reference),
    };

    const prepared = await this.autofill(payment);
    this.enforceFee(prepared);
    return prepared;
  }

  // depositId must be xrplDepositId(txHash). The deposit is absent unless the
  // transaction is a Payment from another account to the vault carrying a
  // ynet-account memo with a non-zero account, and, once validated, its result
  // is tesSUCCESS. Custody's crediting rules (partial payments, asset support)
  // are not applied, so "confirmed" does not guarantee a credit.
  async chainDepositStatus(
    txHash: string,
    depositId: string,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const normalized = requireTxID(txHash);
    if (depositId !== xrplDepositId(normalized)) {
      throw new ClearnetSdkError(
        "INVALID_DEPOSIT_ID",
        "XRPL deposit ID must be the lower-cased transaction hash",
      );
    }
    const minConf = normalizeMinConfirmations(minConfirmations);
    await this.ensureConnected();
    let response: TxResponse;
    try {
      response = await this.client.request({
        command: "tx",
        transaction: normalized,
      });
    } catch (error) {
      if (isTxnNotFound(error)) {
        return "absent";
      }
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: tx lookup", {
        cause: error,
      });
    }
    return xrplDepositStatus(response.result, this.vaultAddress, minConf);
  }

  async disconnect(): Promise<void> {
    const connecting = this.connecting;
    if (connecting !== undefined) {
      await connecting;
    }
    if (!this.client.isConnected()) {
      return;
    }
    try {
      await this.client.disconnect();
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: disconnect", {
        cause: error,
      });
    }
  }

  private async autofill(payment: Payment): Promise<Payment> {
    await this.ensureConnected();
    try {
      return await this.client.autofill(payment);
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: autofill", {
        cause: error,
      });
    }
  }

  private enforceFee(prepared: Payment): void {
    if (this.maxFeeDrops === undefined) {
      return;
    }
    if (typeof prepared.Fee !== "string" || !/^[0-9]+$/.test(prepared.Fee)) {
      throw new ClearnetSdkError(
        "RPC_ERROR",
        "xrpl: autofilled fee is missing or invalid",
      );
    }
    if (BigInt(prepared.Fee) > this.maxFeeDrops) {
      throw new ClearnetSdkError(
        "INVALID_AMOUNT",
        "xrpl: autofilled fee exceeds maxFeeDrops",
      );
    }
  }

  private async sign(prepared: Payment): Promise<{ txBlob: string; hash: string }> {
    try {
      return await this.signer.sign(prepared);
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: sign", {
        cause: error,
      });
    }
  }

  private async submit(txBlob: string, txID: string): Promise<void> {
    try {
      const response = await this.client.submit(txBlob, { autofill: false });
      const engineResult = response.result.engine_result;
      if (engineResult !== "tesSUCCESS" && engineResult !== "terQUEUED") {
        throw new ClearnetSdkError(
          "TX_REVERTED",
          `xrpl: deposit rejected: ${engineResult}`,
          { txHash: txID },
        );
      }
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: submit", {
        txHash: txID,
        cause: error,
      });
    }
  }

  private async ensureConnected(): Promise<void> {
    if (this.client.isConnected()) {
      return;
    }
    if (this.connecting !== undefined) {
      await this.connecting;
      return;
    }
    this.connecting = this.connect();
    try {
      await this.connecting;
    } finally {
      this.connecting = undefined;
    }
  }

  private async connect(): Promise<void> {
    try {
      await this.client.connect();
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: connect", {
        cause: error,
      });
    }
  }
}

function isTxnNotFound(error: unknown): boolean {
  if (!error || typeof error !== "object") {
    return false;
  }
  if (readStringProperty(error, "error") === "txnNotFound") {
    return true;
  }
  const data = "data" in error ? error.data : undefined;
  if (
    data &&
    typeof data === "object" &&
    readStringProperty(data, "error") === "txnNotFound"
  ) {
    return true;
  }
  const message =
    "message" in error && typeof error.message === "string" ? error.message : "";
  return message.includes("txnNotFound");
}

function readStringProperty(value: object, key: string): string | undefined {
  const field = (value as Record<string, unknown>)[key];
  return typeof field === "string" ? field : undefined;
}

function xrplDepositStatus(
  result: TxResponse["result"],
  vaultAddress: string,
  minConfirmations: bigint,
): DepositStatus {
  // XRPL finality is binary: a transaction in a validated ledger is final.
  // The shared minConfirmations argument is validated for API parity only.
  void minConfirmations;
  const tx: Record<string, unknown> =
    result.tx_json && typeof result.tx_json === "object"
      ? (result.tx_json as unknown as Record<string, unknown>)
      : {};
  if (
    tx.TransactionType !== "Payment" ||
    tx.Account === vaultAddress ||
    tx.Destination !== vaultAddress ||
    !hasDepositMemo(tx.Memos)
  ) {
    return "absent";
  }
  if (result.validated !== true) {
    return "pending";
  }
  const meta = result.meta;
  const transactionResult =
    meta && typeof meta === "object" ? meta.TransactionResult : undefined;
  return transactionResult === "tesSUCCESS" ? "confirmed" : "absent";
}

// Reports whether memos carries a ynet-account memo whose MemoData is a
// non-zero 20-byte account followed by a 32-byte reference. The first
// ynet-account memo with well-formed MemoData decides.
function hasDepositMemo(memos: unknown): boolean {
  if (!Array.isArray(memos)) {
    return false;
  }
  for (const entry of memos) {
    const memo =
      entry && typeof entry === "object"
        ? (entry as Record<string, unknown>).Memo
        : undefined;
    if (!memo || typeof memo !== "object") {
      continue;
    }
    const fields = memo as Record<string, unknown>;
    if (
      typeof fields.MemoType !== "string" ||
      !HEX_PATTERN.test(fields.MemoType) ||
      fields.MemoType.toLowerCase() !== XRPL_MEMO_TYPE
    ) {
      continue;
    }
    const data = fields.MemoData;
    if (typeof data !== "string" || !HEX_PATTERN.test(data) || data.length !== 2 * (20 + 32)) {
      continue;
    }
    return !/^0{40}$/.test(data.slice(0, 40));
  }
  return false;
}

function requireSubmitDepositOptions(options: unknown): SubmitDepositOptions {
  if (options === null || typeof options !== "object") {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "submit options must be an object",
    );
  }
  const candidate = options as Partial<SubmitDepositOptions>;
  if (
    candidate.onSubmitted !== undefined &&
    typeof candidate.onSubmitted !== "function"
  ) {
    throw new ClearnetSdkError(
      "INVALID_INPUT",
      "submit options.onSubmitted must be a function",
    );
  }
  return options;
}
