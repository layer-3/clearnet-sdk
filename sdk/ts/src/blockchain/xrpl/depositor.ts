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
  normalizeMinConfirmations,
  normalizeTxHash,
  requireClassicAddress,
  requireClearnetAccount,
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
  private readonly client: Client;

  constructor(config: XrplDepositorConfig) {
    this.signer = requireSigner(config.signer);
    this.vaultAddress = requireClassicAddress(config.vaultAddress, "vaultAddress");
    this.maxFeeDrops =
      config.maxFeeDrops === undefined
        ? undefined
        : normalizeFeeDrops(config.maxFeeDrops);
    this.client = new Client(requireRpcUrl(config.rpcUrl));
  }

  async submitDeposit(
    input: XrplSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<TxRef> {
    const account = requireClearnetAccount(input.destination.account);
    const reference = requireReference(input.destination.ref);
    const amount = resolveAmount(input.asset, input.amount);
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
    options.onSubmitted?.(ref);
    return ref;
  }

  async verifyDeposit(
    ref: TxRef,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    const normalized = requireTxRef(ref);
    normalizeMinConfirmations(minConfirmations);
    await this.ensureConnected();
    try {
      const response = await this.client.request({
        command: "tx",
        transaction: normalized.raw,
      });
      return response.result.validated === true ? "confirmed" : "pending";
    } catch (error) {
      if (isTxnNotFound(error)) {
        return "absent";
      }
      throw new ClearnetSdkError("RPC_ERROR", "xrpl: tx lookup", {
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
          "RPC_ERROR",
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
  const message =
    "message" in error && typeof error.message === "string" ? error.message : "";
  const data = "data" in error ? String(error.data) : "";
  return message.includes("txnNotFound") || data.includes("txnNotFound");
}
