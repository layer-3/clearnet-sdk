import Wallet from "sats-connect";

export type XverseBitcoinNetwork =
  | "Mainnet"
  | "Testnet"
  | "Testnet4"
  | "Signet"
  | "Regtest";

export interface XverseWallet {
  accountId?: string | undefined;
  address: string;
  publicKey: string;
  addressType: "p2wpkh" | "p2sh";
  network?: XverseBitcoinNetwork | string | undefined;
  walletType?: string | undefined;
}

export interface XverseRegtestNetwork {
  name: string;
  rpcUrl: string;
}

export type XverseRequest = (
  method: string,
  params: unknown,
) => Promise<unknown>;

const XVERSE_CONNECT_TIMEOUT_MS = 12_000;

interface XverseAddress {
  address?: unknown;
  publicKey?: unknown;
  purpose?: unknown;
  addressType?: unknown;
  walletType?: unknown;
}

interface XverseResultEnvelope {
  status?: unknown;
  result?: unknown;
  error?: { code?: unknown; message?: unknown };
}

export async function configureXverseRegtestNetwork(
  network: XverseRegtestNetwork,
  requestImpl: XverseRequest = requestXverse,
): Promise<void> {
  requireNonEmpty(network.name, "Xverse network name");
  requireNonEmpty(network.rpcUrl, "Xverse RPC URL");
  const addResponse = await requestImpl("wallet_addNetwork", {
    chain: "bitcoin",
    type: "Regtest",
    name: network.name,
    rpcUrl: network.rpcUrl,
    switch: true,
  });
  assertXverseSuccess(addResponse, "wallet_addNetwork", {
    allowAlreadyExists: true,
  });
}

export async function connectXverseWallet(
  requestImpl: XverseRequest = requestXverse,
): Promise<XverseWallet> {
  const result = await requestXverseAddresses(requestImpl);
  const paymentAddress = extractAddresses(result).find(
    (address) =>
      address.purpose === "payment" &&
      (address.addressType === "p2wpkh" || address.addressType === "p2sh") &&
      typeof address.address === "string" &&
      typeof address.publicKey === "string",
  );
  if (paymentAddress === undefined) {
    throw new Error("Xverse did not return a supported payment address");
  }
  return {
    accountId: stringField(result, "id"),
    address: paymentAddress.address as string,
    publicKey: paymentAddress.publicKey as string,
    addressType: paymentAddress.addressType as "p2wpkh" | "p2sh",
    network: bitcoinNetworkName(result),
    walletType:
      typeof paymentAddress.walletType === "string"
        ? paymentAddress.walletType
        : stringField(result, "walletType"),
  };
}

async function requestXverseAddresses(
  requestImpl: XverseRequest,
): Promise<unknown> {
  const attempts: ReadonlyArray<{
    method: string;
    params: unknown;
  }> = [
    {
      method: "getAccounts",
      params: {
        purposes: ["payment"],
        message: "Connect Bitcoin deposit demo",
      },
    },
    {
      method: "getAddresses",
      params: {
        purposes: ["payment"],
        message: "Connect Bitcoin deposit demo",
      },
    },
  ];
  const errors: string[] = [];
  for (const attempt of attempts) {
    try {
      const response = await withTimeout(
        requestImpl(attempt.method, attempt.params),
        XVERSE_CONNECT_TIMEOUT_MS,
        `Xverse ${attempt.method} timed out after ${XVERSE_CONNECT_TIMEOUT_MS}ms`,
      );
      return unwrapXverseResult(response, attempt.method);
    } catch (error) {
      const message = errorMessage(error);
      if (
        attempt.method === "getAccounts" &&
        /denied|reject|cancel/i.test(message)
      ) {
        throw error;
      }
      errors.push(message);
    }
  }
  throw new Error(`Xverse address request failed: ${errors.join("; ")}`);
}

export async function signPsbtWithXverse(
  input: {
    psbtHex: string;
    inputIndexesToSign: readonly number[];
    address: string;
  },
  requestImpl: XverseRequest = requestXverse,
): Promise<string> {
  requireNonEmpty(input.address, "Xverse address");
  if (input.inputIndexesToSign.length === 0) {
    throw new Error("PSBT has no inputs for Xverse to sign");
  }
  const response = await requestImpl("signPsbt", {
    psbt: hexToBase64(input.psbtHex),
    signInputs: {
      [input.address]: [...input.inputIndexesToSign],
    },
    broadcast: false,
  });
  const result = unwrapXverseResult(response, "signPsbt");
  const psbt = stringField(result, "psbt");
  if (psbt === undefined) {
    throw new Error("Xverse did not return a signed PSBT");
  }
  return base64ToHex(psbt);
}

function requestXverse(method: string, params: unknown): Promise<unknown> {
  return (Wallet.request as unknown as XverseRequest)(method, params);
}

function withTimeout<T>(
  promise: Promise<T>,
  timeoutMs: number,
  message: string,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const timeout = globalThis.setTimeout(() => {
      reject(new Error(message));
    }, timeoutMs);
    promise.then(
      (value) => {
        globalThis.clearTimeout(timeout);
        resolve(value);
      },
      (error: unknown) => {
        globalThis.clearTimeout(timeout);
        reject(error);
      },
    );
  });
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function assertXverseSuccess(
  response: unknown,
  method: string,
  options: { allowAlreadyExists?: boolean } = {},
): void {
  const envelope = response as XverseResultEnvelope;
  if (envelope?.status !== "error") {
    return;
  }
  const message =
    typeof envelope.error?.message === "string" ? envelope.error.message : method;
  if (
    options.allowAlreadyExists === true &&
    /already|exist/i.test(message)
  ) {
    return;
  }
  throw new Error(`Xverse ${method} failed: ${message}`);
}

function unwrapXverseResult(response: unknown, method: string): unknown {
  assertXverseSuccess(response, method);
  const envelope = response as XverseResultEnvelope;
  if (envelope?.status === "success") {
    return envelope.result;
  }
  return response;
}

function extractAddresses(result: unknown): XverseAddress[] {
  if (Array.isArray(result)) {
    return result as XverseAddress[];
  }
  if (result && typeof result === "object") {
    const addresses = (result as { addresses?: unknown }).addresses;
    if (Array.isArray(addresses)) {
      return addresses as XverseAddress[];
    }
  }
  throw new Error("Xverse response did not include addresses");
}

function bitcoinNetworkName(result: unknown): string | undefined {
  if (!result || typeof result !== "object") {
    return undefined;
  }
  const network = (result as { network?: unknown }).network;
  if (!network || typeof network !== "object") {
    return undefined;
  }
  const bitcoin = (network as { bitcoin?: unknown }).bitcoin;
  if (!bitcoin || typeof bitcoin !== "object") {
    return undefined;
  }
  return stringField(bitcoin, "name");
}

function stringField(value: unknown, field: string): string | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  const fieldValue = (value as Record<string, unknown>)[field];
  return typeof fieldValue === "string" ? fieldValue : undefined;
}

function requireNonEmpty(value: unknown, field: string): string {
  if (typeof value !== "string" || value.trim() === "") {
    throw new Error(`${field} is required`);
  }
  return value.trim();
}

function hexToBase64(hex: string): string {
  if (!/^(?:[a-fA-F0-9]{2})*$/.test(hex)) {
    throw new Error("PSBT must be even-length hex");
  }
  let binary = "";
  for (let index = 0; index < hex.length; index += 2) {
    binary += String.fromCharCode(Number.parseInt(hex.slice(index, index + 2), 16));
  }
  return btoa(binary);
}

function base64ToHex(base64: string): string {
  const binary = atob(base64);
  let hex = "";
  for (let index = 0; index < binary.length; index += 1) {
    hex += binary.charCodeAt(index).toString(16).padStart(2, "0");
  }
  return hex;
}
