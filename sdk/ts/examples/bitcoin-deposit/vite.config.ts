import { Buffer } from "node:buffer";
import type { IncomingMessage, ServerResponse } from "node:http";

import { defineConfig } from "vite";
import type { Plugin, ViteDevServer } from "vite";

const rpcTarget = process.env.BTC_RPC_URL ?? "http://127.0.0.1:18443";
const rpcUser = process.env.BTC_RPC_USER ?? "sdk";
const rpcPass = process.env.BTC_RPC_PASS ?? "sdk";
const rpcWallet = process.env.BTC_RPC_WALLET ?? "sdk";
const rpcAuth = Buffer.from(`${rpcUser}:${rpcPass}`).toString("base64");
const rpcBaseUrl = trimTrailingSlash(rpcTarget);

export default defineConfig({
  plugins: [bitcoinElectrsFacade()],
  server: {
    cors: true,
    proxy: {
      "/btc-rpc": {
        target: rpcTarget,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/btc-rpc/, "") || "/",
        configure(proxy) {
          proxy.on("proxyReq", (proxyReq) => {
            proxyReq.setHeader("Authorization", `Basic ${rpcAuth}`);
          });
        },
      },
    },
  },
});

function bitcoinElectrsFacade(): Plugin {
  return {
    name: "bitcoin-electrs-facade",
    configureServer(server: ViteDevServer) {
      server.middlewares.use(async (
        req: IncomingMessage,
        res: ServerResponse,
        next: () => void,
      ) => {
        if (req.url === undefined || !req.url.startsWith("/btc-rpc/")) {
          next();
          return;
        }
        const url = new URL(req.url, "http://127.0.0.1");
        if (req.method === "OPTIONS") {
          sendEmpty(res, 204);
          return;
        }
        try {
          if (req.method === "GET") {
            if (await handleElectrsGet(url, res)) {
              return;
            }
          } else if (req.method === "POST") {
            if (await handleElectrsPost(req, url, res)) {
              return;
            }
          }
          next();
        } catch (error) {
          sendJson(res, 500, {
            error: error instanceof Error ? error.message : String(error),
          });
        }
      });
    },
  };
}

async function handleElectrsGet(
  url: URL,
  res: ServerResponse,
): Promise<boolean> {
  const pathname = decodeURIComponent(url.pathname);
  const addressUtxoMatch = pathname.match(/^\/btc-rpc\/address\/([^/]+)\/utxo$/u);
  if (addressUtxoMatch !== null) {
    const address = addressUtxoMatch[1];
    if (address === undefined || address === "") {
      sendJson(res, 400, { error: "address is required" });
      return true;
    }
    const utxos = await rpcCall<BitcoinCoreUnspent[]>("wallet", "listunspent", [
      0,
      9999999,
      [address],
      true,
    ]);
    const tipHeight = await rpcCall<number>("root", "getblockcount", []);
    sendJson(
      res,
      200,
      utxos.map((utxo) => electrsUtxo(utxo, tipHeight)),
    );
    return true;
  }

  const txHexMatch = pathname.match(/^\/btc-rpc\/tx\/([0-9a-fA-F]{64})\/hex$/u);
  if (txHexMatch !== null) {
    const txid = txHexMatch[1];
    sendText(res, 200, await rpcCall<string>("root", "getrawtransaction", [txid, false]));
    return true;
  }

  const txStatusMatch = pathname.match(/^\/btc-rpc\/tx\/([0-9a-fA-F]{64})\/status$/u);
  if (txStatusMatch !== null) {
    const txid = txStatusMatch[1];
    const tx = await rpcCall<BitcoinCoreRawTransaction>("root", "getrawtransaction", [
      txid,
      true,
    ]);
    sendJson(res, 200, electrsStatus(tx));
    return true;
  }

  const txMatch = pathname.match(/^\/btc-rpc\/tx\/([0-9a-fA-F]{64})$/u);
  if (txMatch !== null) {
    const txid = txMatch[1];
    const tx = await rpcCall<BitcoinCoreRawTransaction>("root", "getrawtransaction", [
      txid,
      true,
    ]);
    sendJson(res, 200, electrsTransaction(tx));
    return true;
  }

  if (pathname === "/btc-rpc/blocks/tip/height") {
    sendText(res, 200, String(await rpcCall<number>("root", "getblockcount", [])));
    return true;
  }

  return false;
}

async function handleElectrsPost(
  req: IncomingMessage,
  url: URL,
  res: ServerResponse,
): Promise<boolean> {
  if (url.pathname !== "/btc-rpc/tx") {
    return false;
  }
  const rawTx = (await readRequestBody(req)).trim();
  if (rawTx === "") {
    sendJson(res, 400, { error: "raw transaction hex is required" });
    return true;
  }
  sendText(res, 200, await rpcCall<string>("root", "sendrawtransaction", [rawTx]));
  return true;
}

async function rpcCall<T>(
  scope: "root" | "wallet",
  method: string,
  params: unknown[],
): Promise<T> {
  const endpoint =
    scope === "wallet"
      ? `${rpcBaseUrl}/wallet/${encodeURIComponent(rpcWallet)}`
      : rpcBaseUrl;
  const response = await fetch(endpoint, {
    method: "POST",
    headers: {
      Authorization: `Basic ${rpcAuth}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      jsonrpc: "1.0",
      id: "demo-electrs",
      method,
      params,
    }),
  });
  if (!response.ok) {
    throw new Error(`bitcoind ${method} HTTP ${response.status}`);
  }
  const envelope = (await response.json()) as {
    result?: T;
    error?: { code: number; message: string } | null;
  };
  if (envelope.error !== null && envelope.error !== undefined) {
    throw new Error(`bitcoind ${method} error ${envelope.error.code}: ${envelope.error.message}`);
  }
  return envelope.result as T;
}

function electrsUtxo(utxo: BitcoinCoreUnspent, tipHeight: number): ElectrsUtxo {
  const confirmations = numberField(utxo.confirmations, "confirmations");
  const status: ElectrsStatus = {
    confirmed: confirmations > 0,
  };
  if (confirmations > 0) {
    status.block_height = tipHeight - confirmations + 1;
  }
  return {
    txid: stringField(utxo.txid, "txid"),
    vout: numberField(utxo.vout, "vout"),
    value: btcToSats(utxo.amount),
    status,
  };
}

function electrsTransaction(tx: BitcoinCoreRawTransaction): ElectrsTransaction {
  return {
    txid: stringField(tx.txid, "txid"),
    version: optionalNumber(tx.version) ?? 1,
    locktime: optionalNumber(tx.locktime) ?? 0,
    vin: Array.isArray(tx.vin) ? tx.vin : [],
    vout: Array.isArray(tx.vout) ? tx.vout.map(electrsVout) : [],
    size: optionalNumber(tx.size) ?? 0,
    weight: optionalNumber(tx.weight) ?? 0,
    fee: optionalNumber(tx.fee) === undefined ? undefined : btcToSats(tx.fee),
    status: electrsStatus(tx),
  };
}

function electrsVout(vout: BitcoinCoreVout): ElectrsVout {
  const script = vout.scriptPubKey;
  return {
    scriptpubkey: script?.hex ?? "",
    scriptpubkey_asm: script?.asm ?? "",
    scriptpubkey_type: script?.type ?? "unknown",
    scriptpubkey_address: script?.address,
    value: btcToSats(vout.value),
  };
}

function electrsStatus(tx: BitcoinCoreRawTransaction): ElectrsStatus {
  const confirmations = optionalNumber(tx.confirmations) ?? 0;
  const status: ElectrsStatus = {
    confirmed: confirmations > 0,
  };
  if (confirmations > 0) {
    status.block_height = optionalNumber(tx.blockheight);
    status.block_hash = typeof tx.blockhash === "string" ? tx.blockhash : undefined;
    status.block_time = optionalNumber(tx.blocktime);
  }
  return status;
}

function readRequestBody(req: IncomingMessage): Promise<string> {
  return new Promise((resolve, reject) => {
    let body = "";
    req.setEncoding("utf8");
    req.on("data", (chunk: string) => {
      body += chunk;
    });
    req.on("end", () => {
      resolve(body);
    });
    req.on("error", reject);
  });
}

function sendJson(res: ServerResponse, status: number, body: unknown): void {
  sendHeaders(res, status, "application/json");
  res.end(JSON.stringify(body));
}

function sendText(res: ServerResponse, status: number, body: string): void {
  sendHeaders(res, status, "text/plain");
  res.end(body);
}

function sendEmpty(res: ServerResponse, status: number): void {
  sendHeaders(res, status, "text/plain");
  res.end();
}

function sendHeaders(
  res: ServerResponse,
  status: number,
  contentType: string,
): void {
  res.statusCode = status;
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET,POST,OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "content-type,authorization,x-client-version");
  res.setHeader("Content-Type", contentType);
}

function btcToSats(value: unknown): number {
  if (typeof value === "number") {
    if (!Number.isFinite(value) || value < 0) {
      throw new Error("Bitcoin amount must be a non-negative number");
    }
    return Math.round(value * 100_000_000);
  }
  if (typeof value === "string" && /^(?:0|[1-9][0-9]*)(?:\.[0-9]{1,8})?$/u.test(value)) {
    const [whole = "0", frac = ""] = value.split(".");
    return Number(BigInt(whole) * 100_000_000n + BigInt(frac.padEnd(8, "0")));
  }
  throw new Error("Bitcoin amount is invalid");
}

function stringField(value: unknown, field: string): string {
  if (typeof value !== "string") {
    throw new Error(`${field} must be a string`);
  }
  return value;
}

function numberField(value: unknown, field: string): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) {
    throw new Error(`${field} must be a non-negative safe integer`);
  }
  return value;
}

function optionalNumber(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function trimTrailingSlash(value: string): string {
  return value.endsWith("/") ? value.slice(0, -1) : value;
}

interface BitcoinCoreUnspent {
  txid?: unknown;
  vout?: unknown;
  amount?: unknown;
  confirmations?: unknown;
}

interface BitcoinCoreRawTransaction {
  txid?: unknown;
  version?: unknown;
  locktime?: unknown;
  vin?: unknown[];
  vout?: BitcoinCoreVout[];
  size?: unknown;
  weight?: unknown;
  fee?: unknown;
  confirmations?: unknown;
  blockheight?: unknown;
  blockhash?: unknown;
  blocktime?: unknown;
}

interface BitcoinCoreVout {
  value?: unknown;
  scriptPubKey?: {
    hex?: string;
    asm?: string;
    type?: string;
    address?: string;
  };
}

interface ElectrsStatus {
  confirmed: boolean;
  block_height?: number | undefined;
  block_hash?: string | undefined;
  block_time?: number | undefined;
}

interface ElectrsUtxo {
  txid: string;
  vout: number;
  value: number;
  status: ElectrsStatus;
}

interface ElectrsTransaction {
  txid: string;
  version: number;
  locktime: number;
  vin: unknown[];
  vout: ElectrsVout[];
  size: number;
  weight: number;
  fee?: number | undefined;
  status: ElectrsStatus;
}

interface ElectrsVout {
  scriptpubkey: string;
  scriptpubkey_asm: string;
  scriptpubkey_type: string;
  scriptpubkey_address?: string | undefined;
  value: number;
}
