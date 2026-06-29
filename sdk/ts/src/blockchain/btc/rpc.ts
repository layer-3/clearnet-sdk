import { Buffer } from "buffer";

import { ClearnetSdkError } from "../../core/errors.js";
import type {
  BitcoinCoreRpcClientConfig,
  BitcoinRawTransaction,
  BitcoinRpc,
  BitcoinUnspent,
} from "./types.js";
export { BitcoinRpcError } from "./types.js";
import { BitcoinRpcError } from "./types.js";

interface JsonRpcEnvelope {
  result?: unknown;
  error?: { code: number; message: string } | null;
}

export class BitcoinCoreRpcClient implements BitcoinRpc {
  private readonly url: string;
  private readonly username: string | undefined;
  private readonly password: string | undefined;
  private readonly wallet: string | undefined;
  private readonly fetchImpl: typeof fetch;

  constructor(config: BitcoinCoreRpcClientConfig) {
    if (typeof config.url !== "string" || config.url.trim() === "") {
      throw new ClearnetSdkError("RPC_ERROR", "Bitcoin Core RPC url is required");
    }
    const hasUsername = config.username !== undefined;
    const hasPassword = config.password !== undefined;
    if (hasUsername !== hasPassword) {
      throw new ClearnetSdkError(
        "INVALID_INPUT",
        "Bitcoin Core RPC username and password must be supplied together",
      );
    }
    this.url = trimTrailingSlash(config.url);
    this.username = config.username;
    this.password = config.password;
    this.wallet = config.wallet;
    this.fetchImpl = config.fetch ?? globalThis.fetch.bind(globalThis);
  }

  async listUnspent(
    minConfirmations: number,
    addresses: readonly string[],
  ): Promise<readonly BitcoinUnspent[]> {
    const result = await this.call("wallet", "listunspent", [
      minConfirmations,
      9999999,
      addresses,
    ]);
    if (!Array.isArray(result)) {
      throw new ClearnetSdkError("RPC_ERROR", "btc rpc listunspent returned invalid result");
    }
    return result.map((entry) => {
      if (!entry || typeof entry !== "object") {
        throw new ClearnetSdkError("RPC_ERROR", "btc rpc listunspent entry is invalid");
      }
      const fields = entry as Record<string, unknown>;
      return {
        txid: requireString(fields.txid, "txid"),
        vout: requireNumber(fields.vout, "vout"),
        amountSats: btcToSats(fields.amount),
        confirmations: requireNumber(fields.confirmations, "confirmations"),
        scriptPubKey: requireString(fields.scriptPubKey, "scriptPubKey"),
      };
    });
  }

  async estimateSmartFeeSatPerVByte(
    confirmationTarget: number,
    fallbackRate: bigint,
  ): Promise<bigint> {
    const result = await this.call("root", "estimatesmartfee", [confirmationTarget]);
    if (!result || typeof result !== "object") {
      return fallbackRate;
    }
    const feeRate = (result as Record<string, unknown>).feerate;
    if (typeof feeRate !== "number" || !Number.isFinite(feeRate) || feeRate <= 0) {
      return fallbackRate;
    }
    const sats = Math.round(feeRate * 100_000_000 / 1000);
    return sats > 0 ? BigInt(sats) : fallbackRate;
  }

  async sendRawTransaction(hexTx: string): Promise<string> {
    const result = await this.call("root", "sendrawtransaction", [hexTx]);
    return requireString(result, "sendrawtransaction result");
  }

  async getRawTransaction(txid: string): Promise<BitcoinRawTransaction | null> {
    const result = await this.call("root", "getrawtransaction", [txid, true]);
    if (result === null) {
      return null;
    }
    if (!result || typeof result !== "object") {
      throw new ClearnetSdkError("RPC_ERROR", "btc rpc getrawtransaction returned invalid result");
    }
    const fields = result as Record<string, unknown>;
    return {
      txid: requireString(fields.txid, "txid"),
      confirmations:
        fields.confirmations === undefined
          ? 0
          : requireNumber(fields.confirmations, "confirmations"),
    };
  }

  private async call(scope: "root" | "wallet", method: string, params: unknown[]): Promise<unknown> {
    const endpoint =
      scope === "wallet" && this.wallet !== undefined
        ? `${this.url}/wallet/${encodeURIComponent(this.wallet)}`
        : this.url;
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
    };
    if (this.username !== undefined && this.password !== undefined) {
      headers.Authorization = `Basic ${Buffer.from(
        `${this.username}:${this.password}`,
      ).toString("base64")}`;
    }
    const response = await this.fetchImpl(endpoint, {
      method: "POST",
      headers,
      body: JSON.stringify({
        jsonrpc: "1.0",
        id: "sdk",
        method,
        params,
      }),
    });
    const envelope = (await response.json()) as JsonRpcEnvelope;
    if (envelope.error !== null && envelope.error !== undefined) {
      throw new BitcoinRpcError(envelope.error.code, envelope.error.message);
    }
    return envelope.result;
  }
}

function btcToSats(value: unknown): bigint {
  if (typeof value === "number") {
    if (!Number.isFinite(value) || value < 0) {
      throw new ClearnetSdkError("RPC_ERROR", "Bitcoin Core amount must be non-negative");
    }
    return BigInt(Math.round(value * 100_000_000));
  }
  if (typeof value === "string" && /^(?:0|[1-9][0-9]*)(?:\.[0-9]{1,8})?$/.test(value)) {
    const [whole = "0", frac = ""] = value.split(".");
    return BigInt(whole) * 100_000_000n + BigInt(frac.padEnd(8, "0"));
  }
  throw new ClearnetSdkError("RPC_ERROR", "Bitcoin Core amount is invalid");
}

function requireString(value: unknown, field: string): string {
  if (typeof value !== "string") {
    throw new ClearnetSdkError("RPC_ERROR", `${field} must be a string`);
  }
  return value;
}

function requireNumber(value: unknown, field: string): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) {
    throw new ClearnetSdkError("RPC_ERROR", `${field} must be a non-negative safe integer`);
  }
  return value;
}

function trimTrailingSlash(value: string): string {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}
