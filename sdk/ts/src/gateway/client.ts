// Clearnet public gateway client: a TypeScript port of pkg/client.
//
// Scoped to one expected network. Every write path re-validates the gateway
// identity first; reads may reuse an identity cached for a few seconds.
// Signing is delegated to any EIP-712 typed-data signer (a browser wallet via
// wagmi/viem, or a local viem account), so no private key passes through here.

import { isAddressEqual } from "viem";
import { toBaseUnits } from "./amounts.js";
import {
  networkChainId,
  normalizeSignature,
  recoverTransferSigner,
  resolveRecipient,
  sortTransferAssets,
  transferOperationId,
  transferTypedData,
  validOperationId,
  VIRTUAL_ROUTER,
  type TransferTypedData,
} from "./eip712.js";
import { APIError, GatewayError, SubmissionError } from "./errors.js";
import { networkFingerprint } from "./fingerprint.js";
import {
  CAPABILITY_RECENT_TRANSFERS,
  CAPABILITY_TRANSFER_STATUS,
  type AssetInfo,
  type Balance,
  type NetworkInfo,
  type OperationStatus,
  type PoolSummary,
  type Receipt,
  type SignedTransfer,
  type SubmitResult,
  type TransferAmount,
  type TransferAsset,
  type TransferIntent,
  type TransferPage,
} from "./types.js";

const MAX_RESPONSE_BYTES = 2 << 20;
const INFO_TTL_MS = 20_000;
// 409 is a pre-admission conflict (-32009/-32002): the gateway refused the
// intent before changing any state, so the outcome is definite, not unknown.
const DEFINITE_REJECTION_STATUSES = new Set([400, 401, 403, 409, 422, 429]);

export interface GatewayClientOptions {
  /** Absolute HTTPS (or loopback HTTP) base URL, or a same-origin path such as "/api/gw/testnet". */
  baseUrl: string;
  /** Four-character network ID the gateway must serve (YNET, YTES, YDEV). */
  networkId: string;
  /** Optional pinned identity fingerprint, the value `yellow network show` prints. */
  fingerprint?: string;
  /** Per-request timeout. Defaults to 15 s, like the CLI. */
  timeoutMs?: number;
  fetch?: typeof fetch;
}

export interface PrepareTransferInput {
  from: string;
  /** 0x address or yellow://ynet/user/<address>. */
  recipient: string;
  /** Human decimal amounts; converted exactly with gateway-reported decimals. */
  assets: TransferAmount[];
  /** Explicit YELLOW fee ceiling in decimal units. "0" authorizes no fee. */
  maxFee: string;
  signal?: AbortSignal;
}

/**
 * Anything that signs EIP-712 typed data and returns a 65-byte signature:
 * wagmi `signTypedDataAsync`, viem `walletClient.signTypedData`, or
 * `account.signTypedData` from viem/accounts.
 */
export type TypedDataSigner = (typedData: TransferTypedData) => Promise<string>;

export class GatewayClient {
  readonly baseUrl: string;
  readonly networkId: string;
  readonly chainId: number;
  private readonly fingerprint: string | undefined;
  private readonly timeoutMs: number;
  private readonly fetchImpl: typeof fetch;
  private infoCache: { at: number; value: Promise<NetworkInfo> } | undefined;

  constructor(options: GatewayClientOptions) {
    try {
      this.chainId = networkChainId(options.networkId);
    } catch (cause) {
      throw new GatewayError("input", "A four-character Clearnet network ID is required.", cause);
    }
    this.networkId = options.networkId;
    const base = options.baseUrl.replace(/\/+$/, "");
    if (/^https?:\/\//i.test(base)) {
      const u = new URL(base);
      if (u.username || u.password || u.search || u.hash) {
        throw new GatewayError("input", "Gateway URL must not contain credentials, a query or a fragment.");
      }
      if (u.protocol === "http:" && !isLoopback(u.hostname)) {
        throw new GatewayError("input", "Remote gateways must use HTTPS.");
      }
    } else if (!base.startsWith("/") || base.startsWith("//") || base.startsWith("/\\")) {
      // "//host" and "/\host" are authority-relative: the browser would resolve
      // them to another origin and post a signed transfer there.
      throw new GatewayError("input", "Gateway URL must be an absolute HTTP(S) URL or a same-origin path.");
    }
    this.baseUrl = base;
    this.fingerprint = options.fingerprint?.toLowerCase();
    this.timeoutMs = options.timeoutMs ?? 15_000;
    this.fetchImpl = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  private async request<T>(method: "GET" | "POST", path: string, body?: unknown, signal?: AbortSignal): Promise<{ status: number; data: T }> {
    const timeout = AbortSignal.timeout(this.timeoutMs);
    const init: RequestInit = {
      method,
      headers: body === undefined ? { Accept: "application/json" } : { Accept: "application/json", "Content-Type": "application/json" },
      // A redirect must never replay a signed POST somewhere else.
      redirect: "error",
      cache: "no-store",
      signal: signal ? AbortSignal.any([signal, timeout]) : timeout,
    };
    if (body !== undefined) init.body = JSON.stringify(body);
    let res: Response;
    try {
      res = await this.fetchImpl(this.baseUrl + path, init);
    } catch (cause) {
      if (signal?.aborted) throw cause;
      throw new GatewayError("network", timeout.aborted ? "The network didn't respond in time." : "Couldn't reach the Yellow Network gateway.", cause);
    }
    let raw: string;
    try {
      raw = await readCapped(res, MAX_RESPONSE_BYTES);
    } catch (cause) {
      if (cause instanceof GatewayError) throw cause;
      if (signal?.aborted) throw cause;
      throw new GatewayError("network", timeout.aborted ? "The network didn't respond in time." : "Couldn't read the gateway response.", cause);
    }
    if (res.status < 200 || res.status >= 300) {
      let wire: { error?: { code?: number; message?: string; field?: string; data?: unknown } } | undefined;
      try {
        wire = JSON.parse(raw);
      } catch {
        wire = undefined;
      }
      const e = wire?.error;
      if (!e?.message) {
        throw new APIError({ httpStatus: res.status, message: "Unrecognized gateway error.", recognized: false });
      }
      throw new APIError({
        httpStatus: res.status,
        message: e.message,
        ...(e.code !== undefined && { code: e.code }),
        ...(e.field !== undefined && { field: e.field }),
        ...(e.data !== undefined && { data: e.data }),
      });
    }
    try {
      return { status: res.status, data: (raw ? JSON.parse(raw) : undefined) as T };
    } catch (cause) {
      throw new GatewayError("network", "Gateway response wasn't valid JSON.", cause);
    }
  }

  /** Self-described network metadata, rejected unless it is the expected deployment. */
  info(options: { fresh?: boolean; signal?: AbortSignal } = {}): Promise<NetworkInfo> {
    const now = Date.now();
    // Only signal-less reads are cached: a cached promise bound to one caller's
    // signal would let that caller's abort cancel everyone else's read.
    const cacheable = !options.signal;
    if (!options.fresh && cacheable && this.infoCache && now - this.infoCache.at < INFO_TTL_MS) {
      return this.infoCache.value;
    }
    const value = this.fetchInfo(options.signal);
    if (cacheable) {
      this.infoCache = { at: now, value };
      value.catch(() => {
        if (this.infoCache?.value === value) this.infoCache = undefined;
      });
    }
    return value;
  }

  private async fetchInfo(signal?: AbortSignal): Promise<NetworkInfo> {
    const { data: info } = await this.request<NetworkInfo>("GET", "/v1/gateway/info", undefined, signal);
    if (!info || info.networkId !== this.networkId || info.chainId !== this.chainId) {
      throw new GatewayError("identity", `This gateway serves a different network (expected ${this.networkId}, got ${info?.networkId ?? "nothing"}).`);
    }
    if (typeof info.virtualRouter !== "string" || info.virtualRouter.toLowerCase() !== VIRTUAL_ROUTER.toLowerCase()) {
      throw new GatewayError("identity", "The gateway's signing domain doesn't match Yellow Network.");
    }
    const assets = Array.isArray(info.assets) ? info.assets : [];
    const seen = new Set<string>();
    for (const a of assets) {
      if (!a.symbol || seen.has(a.symbol)) throw new GatewayError("identity", "The gateway's asset list has an empty or duplicated symbol.");
      // Amounts are converted with 10 ** decimals; an out-of-range value from a
      // hostile gateway would hang or overflow the conversion.
      if (!Number.isInteger(a.decimals) || a.decimals < 0 || a.decimals > 77) {
        throw new GatewayError("identity", `The gateway reports an unsupported decimal count for ${a.symbol}.`);
      }
      seen.add(a.symbol);
    }
    info.assets = [...assets].sort((a, b) => compare(a.symbol, b.symbol));
    info.capabilities = Array.isArray(info.capabilities) ? info.capabilities : [];
    if (this.fingerprint) {
      let actual: string;
      try {
        actual = networkFingerprint(info);
      } catch (cause) {
        throw new GatewayError("identity", "The gateway's asset catalogue is invalid.", cause);
      }
      if (actual !== this.fingerprint) {
        throw new GatewayError("trust", "The network's identity changed since it was reviewed. Signing is paused until it is reviewed again.");
      }
    }
    return info;
  }

  /** Live balances and the account nonce, straight from the account's shard. */
  async balance(account: string, signal?: AbortSignal): Promise<Balance> {
    const address = normalizeUserAccount(account);
    await this.info(signal ? { signal } : {});
    const { data } = await this.request<Balance>("GET", `/v1/accounts/${encodeURIComponent(address)}/balances`, undefined, signal);
    return { ...data, balances: Array.isArray(data?.balances) ? data.balances : [] };
  }

  /** The gateway's recent transfer cache (not an archive; retention is in the page). */
  async transfers(account: string, options: { limit?: number; cursor?: string; signal?: AbortSignal } = {}): Promise<TransferPage> {
    const address = normalizeUserAccount(account);
    const info = await this.info(options.signal ? { signal: options.signal } : {});
    if (!info.capabilities.includes(CAPABILITY_RECENT_TRANSFERS)) {
      throw new GatewayError("capability", "This network doesn't provide transfer history yet.");
    }
    const limit = options.limit ?? 0;
    if (!Number.isInteger(limit) || limit < 0 || limit > 100) throw new GatewayError("input", "Transfer page limit must be between 1 and 100.");
    const q = new URLSearchParams();
    if (limit > 0) q.set("limit", String(limit));
    if (options.cursor) q.set("cursor", options.cursor);
    const qs = q.toString();
    const { data } = await this.request<TransferPage>("GET", `/v1/accounts/${encodeURIComponent(address)}/transfers${qs ? `?${qs}` : ""}`, undefined, options.signal);
    return { ...data, items: Array.isArray(data?.items) ? data.items : [] };
  }

  /**
   * Status of one transfer at this gateway. "unknown" is NOT evidence that a
   * fresh nonce is safe to sign.
   */
  async operation(operationId: string, signal?: AbortSignal): Promise<OperationStatus> {
    if (!validOperationId(operationId)) throw new GatewayError("input", "Operation ID must be a lowercase 0x-prefixed 32-byte hash.");
    const info = await this.info(signal ? { signal } : {});
    if (!info.capabilities.includes(CAPABILITY_TRANSFER_STATUS)) {
      throw new GatewayError("capability", "This network can't report transfer status.");
    }
    const { data } = await this.request<OperationStatus>("GET", `/v1/operations/${operationId}`, undefined, signal);
    if (data?.operationId !== operationId || data.kind !== "transfer") {
      throw new GatewayError("network", "The gateway answered about a different operation.");
    }
    if (!["pending", "sealed", "rejected", "unknown"].includes(data.status)) {
      throw new GatewayError("network", `The gateway returned an unsupported status "${String(data.status)}".`);
    }
    return data;
  }

  /** Best-effort AMM pool list (every pool is priced against YELLOW). Empty when unavailable. */
  async pools(signal?: AbortSignal): Promise<PoolSummary[]> {
    // Pool prices feed valuation and fee estimates, so the gateway identity is
    // checked first even though the list itself is best-effort.
    await this.info(signal ? { signal } : {});
    try {
      const { data } = await this.request<{ items?: PoolSummary[] }>("GET", "/v1/pools", undefined, signal);
      return Array.isArray(data?.items) ? data.items : [];
    } catch (cause) {
      if (cause instanceof GatewayError && cause.kind === "network") return [];
      throw cause;
    }
  }

  /**
   * Resolves the live nonce and converts decimal amounts to sorted base-unit
   * legs. Does not sign, submit, or pick a fee ceiling for the user.
   */
  async prepareTransfer(input: PrepareTransferInput): Promise<TransferIntent> {
    const from = normalizeUserAccount(input.from);
    let to: ReturnType<typeof resolveRecipient>;
    try {
      to = resolveRecipient(input.recipient);
    } catch (cause) {
      throw new GatewayError("input", "Transfer recipient must be a nonzero user address.", cause);
    }
    const info = await this.info(input.signal ? { fresh: true, signal: input.signal } : { fresh: true });
    if (!info.capabilities.includes(CAPABILITY_TRANSFER_STATUS)) {
      throw new GatewayError("capability", "This network can't resolve uncertain transfers, so sending is disabled.");
    }
    const catalogue = new Map<string, AssetInfo>(info.assets.map((a) => [a.symbol, a]));
    const yellow = catalogue.get("YELLOW");
    if (!yellow || input.maxFee === "") throw new GatewayError("input", "An explicit YELLOW fee cap is required.");
    let fee: bigint;
    try {
      fee = toBaseUnits(input.maxFee, yellow.decimals);
    } catch (cause) {
      throw new GatewayError("input", "Invalid YELLOW fee cap.", cause);
    }
    if (input.assets.length === 0) throw new GatewayError("input", "Add at least one asset.");
    const seen = new Set<string>();
    const legs: TransferAsset[] = input.assets.map((leg) => {
      const meta = catalogue.get(leg.asset);
      if (!meta || seen.has(leg.asset)) throw new GatewayError("input", `Unknown or duplicate asset ${leg.asset}.`);
      seen.add(leg.asset);
      let units: bigint;
      try {
        units = toBaseUnits(leg.amount, meta.decimals);
      } catch (cause) {
        throw new GatewayError("input", `Invalid ${leg.asset} amount.`, cause);
      }
      if (units <= 0n) throw new GatewayError("input", `${leg.asset} amount must be greater than zero.`);
      return { asset: leg.asset, amount: units.toString() };
    });
    const balance = await this.balance(from, input.signal);
    return {
      catalogueFingerprint: networkFingerprint(info),
      networkId: info.networkId,
      chainId: info.chainId,
      from,
      to,
      assets: sortTransferAssets(legs),
      maxFee: fee.toString(),
      nonce: balance.nonce,
    };
  }

  /** Asks `signer` to sign the exact intent, then binds and verifies the signature. */
  async signTransfer(intent: TransferIntent, signer: TypedDataSigner): Promise<SignedTransfer> {
    const info = await this.info({ fresh: true });
    if (!info.capabilities.includes(CAPABILITY_TRANSFER_STATUS) || intent.networkId !== info.networkId || intent.chainId !== info.chainId) {
      throw new GatewayError("identity", "The network changed since this transfer was prepared.");
    }
    if (intent.catalogueFingerprint !== networkFingerprint(info)) {
      throw new GatewayError("trust", "The asset catalogue changed or is missing. Prepare and review the transfer again.");
    }
    const frozen = freeze(intent);
    return this.finalizeSignature(frozen, await signer(transferTypedData(frozen)));
  }

  /**
   * Binds a signature produced elsewhere to its intent: canonicalizes V,
   * computes the operation ID, and checks it recovers to the sender.
   */
  async finalizeSignature(intent: TransferIntent, rawSignature: string): Promise<SignedTransfer> {
    const signature = normalizeSignature(rawSignature);
    const frozen = freeze(intent);
    const recovered = await recoverTransferSigner(frozen, signature);
    if (!isAddressEqual(recovered, frozen.from)) {
      throw new GatewayError("input", "The signature came from a different account than the sender. Switch your wallet back to the sending account.");
    }
    return { operationId: transferOperationId(frozen), intent: frozen, signature };
  }

  /**
   * Submits a signed transfer unchanged, exactly once. There is no automatic
   * retry: a timeout may happen after the network included it. Persist the
   * signed transfer before calling this.
   */
  async submitTransfer(signed: SignedTransfer, signal?: AbortSignal): Promise<SubmitResult> {
    const id = signed.operationId;
    const preflightError = (cause: unknown) => new SubmissionError({ submitted: false, operationId: id, outcomeUnknown: false, cause });
    if (!validOperationId(id)) throw preflightError(new Error("Invalid operation ID."));
    try {
      const info = await this.info(signal ? { fresh: true, signal } : { fresh: true });
      if (!info.capabilities.includes(CAPABILITY_TRANSFER_STATUS) || signed.intent.networkId !== info.networkId || signed.intent.chainId !== info.chainId) {
        throw new Error("The network or its transfer-status support changed since signing.");
      }
      if (signed.intent.catalogueFingerprint !== networkFingerprint(info)) {
        throw new Error("The asset catalogue changed or is missing. Review the transfer again before submission.");
      }
      if (transferOperationId(signed.intent) !== id) throw new Error("Signed transfer operation ID doesn't match its fields.");
      const recovered = await recoverTransferSigner(signed.intent, signed.signature);
      if (!isAddressEqual(recovered, signed.intent.from)) throw new Error("Signed transfer doesn't match its sender or payload.");
    } catch (cause) {
      throw preflightError(cause);
    }
    const body = {
      to: signed.intent.to,
      assets: signed.intent.assets,
      maxFee: signed.intent.maxFee,
      nonce: signed.intent.nonce,
      signature: signed.signature,
    };
    // A cancelled request that never left this device is not an unknown
    // outcome: reporting it as one would block recovery forever.
    if (signal?.aborted) throw preflightError(new Error("Submission was cancelled before it was sent."));
    let response: { status: number; data: unknown };
    try {
      response = await this.request("POST", "/v1/transfers", body, signal);
    } catch (cause) {
      const definite = cause instanceof APIError && DEFINITE_REJECTION_STATUSES.has(cause.httpStatus);
      throw new SubmissionError({ submitted: true, operationId: id, outcomeUnknown: !definite, cause });
    }
    const unknown = (message: string) => new SubmissionError({ submitted: true, operationId: id, outcomeUnknown: true, cause: new Error(message) });
    if (response.status === 202) {
      const pending = response.data as Partial<OperationStatus> | undefined;
      if (pending?.operationId !== id || pending.status !== "pending") throw unknown("Invalid pending transfer response.");
      return { operationId: id, status: "pending" };
    }
    const receipt = response.data as Partial<Receipt> | undefined;
    if (receipt?.operationId !== id || !receipt.blockHash) throw unknown("Invalid verified transfer receipt.");
    return { operationId: id, status: "sealed", receipt: receipt as Receipt };
  }
}

function compare(a: string, b: string) {
  return a < b ? -1 : a > b ? 1 : 0;
}

/**
 * Reads a response body, refusing anything larger than maxBytes while it
 * streams, so a hostile gateway cannot exhaust memory before the check.
 */
async function readCapped(res: Response, maxBytes: number): Promise<string> {
  const declared = Number(res.headers.get("Content-Length") ?? "");
  if (Number.isFinite(declared) && declared > maxBytes) {
    throw new GatewayError("network", "Gateway response was too large.");
  }
  if (!res.body) {
    const text = await res.text();
    if (new TextEncoder().encode(text).length > maxBytes) {
      throw new GatewayError("network", "Gateway response was too large.");
    }
    return text;
  }
  const reader = res.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    if (!value) continue;
    total += value.byteLength;
    if (total > maxBytes) {
      await reader.cancel().catch(() => undefined);
      throw new GatewayError("network", "Gateway response was too large.");
    }
    chunks.push(value);
  }
  const merged = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    merged.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(merged);
}

function freeze(intent: TransferIntent): TransferIntent {
  return { ...intent, assets: intent.assets.map((a) => ({ ...a })) };
}

function isLoopback(host: string) {
  return host === "localhost" || host === "127.0.0.1" || host === "[::1]" || host === "::1";
}

/** Normalizes a user account (0x address or yellow:// user URI) to a checksummed address. */
export function normalizeUserAccount(account: string): `0x${string}` {
  try {
    return resolveRecipient(account);
  } catch (cause) {
    throw new GatewayError("input", "Account must be a nonzero Ethereum address.", cause);
  }
}
