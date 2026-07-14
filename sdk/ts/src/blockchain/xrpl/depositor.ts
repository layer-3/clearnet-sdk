import { Client } from "xrpl";
import type { Payment } from "xrpl";

import { ClearnetSdkError } from "../../core/errors.js";
import type {
  DepositStatus,
  SubmitDepositOptions,
  TxRef,
  VaultDepositor,
} from "../../core/types.js";
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
  requireTxRef,
  resolveAmount,
} from "./validation.js";

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

  async submitDeposit(
    input: XrplSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<TxRef> {
    const fields =
      input && typeof input === "object"
        ? (input as Partial<XrplSubmitDepositInput>)
        : {};
    const submitOptions = requireSubmitDepositOptions(options);
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
    const signed = await this.sign(prepared);
    const ref = normalizeTxHash(signed.hash);
    await this.submit(signed.txBlob, ref);
    submitOptions.onSubmitted?.(ref);
    return ref;
  }

  async verifyDeposit(
    ref: TxRef,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const normalized = requireTxRef(ref);
    const minConf = normalizeMinConfirmations(minConfirmations);
    await this.ensureConnected();
    try {
      const response = await this.client.request({
        command: "tx",
        transaction: normalized.raw,
      });
      return xrplDepositStatus(response.result.validated === true, minConf);
    } catch (error) {
      if (isTxnNotFound(error)) {
        return "absent";
      }
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: tx lookup", {
        cause: error,
      });
    }
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

  private async submit(txBlob: string, ref: TxRef): Promise<void> {
    try {
      const response = await this.client.submit(txBlob, { autofill: false });
      const engineResult = response.result.engine_result;
      if (engineResult !== "tesSUCCESS" && engineResult !== "terQUEUED") {
        throw new ClearnetSdkError(
          "TX_REVERTED",
          `xrpl: deposit rejected: ${engineResult}`,
          { txRef: ref },
        );
      }
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: submit", {
        txRef: ref,
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

function xrplDepositStatus(validated: boolean, minConfirmations: bigint): DepositStatus {
  // XRPL finality is binary: a transaction in a validated ledger is final.
  // The shared minConfirmations argument is validated for API parity only.
  void minConfirmations;
  return validated ? "confirmed" : "pending";
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
