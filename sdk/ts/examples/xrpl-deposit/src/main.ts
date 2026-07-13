import {
  getAddress,
  getNetwork,
  isInstalled,
  signTransaction,
} from "@gemwallet/api";
import {
  XRPL_NATIVE_ASSET,
  XrplVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import type {
  Bytes32Hex,
  TxRef,
  XrplPreparedPayment,
  XrplSigner,
} from "@yellow-org/clearnet-sdk";
import { Client, Wallet, hashes, type SubmittableTransaction } from "xrpl";
import {
  createLocalXrplSigner,
  LOCAL_XRPL_GENESIS_SEED,
} from "./local-signer.js";

const form = mustElement<HTMLFormElement>("deposit-form");
const localSignerButton = mustElement<HTMLButtonElement>("connect-local");
const gemWalletButton = mustElement<HTMLButtonElement>("connect-gemwallet");
const fundButton = mustElement<HTMLButtonElement>("fund");
const submitButton = mustElement<HTMLButtonElement>("submit");
const verifyButton = mustElement<HTMLButtonElement>("verify");
const logOutput = mustElement<HTMLOutputElement>("log");

let signer: XrplSigner | undefined;
let lastRef: TxRef | undefined;
let depositor: XrplVaultDepositor | undefined;

const GEMWALLET_NETWORK_TIMEOUT_MS = 8_000;
const GEMWALLET_ADDRESS_TIMEOUT_MS = 60_000;
const NETWORK_PROBE_TIMEOUT_MS = 5_000;

type GemWalletNetwork = {
  chain: string;
  network: string;
  websocket: string;
};

type NetworkIdentity = {
  networkId: number;
  url: string;
};

localSignerButton.addEventListener("click", () => {
  void connectLocalSigner();
});

gemWalletButton.addEventListener("click", () => {
  void connectGemWallet();
});

fundButton.addEventListener("click", () => {
  void fundWallet();
});

form.addEventListener("submit", (event) => {
  event.preventDefault();
  void submitDeposit();
});

verifyButton.addEventListener("click", () => {
  void verifyLastTx();
});

for (const id of ["rpc-url", "vault-address"]) {
  mustElement<HTMLInputElement>(id).addEventListener("input", () => {
    void clearSubmittedDeposit().catch(console.error);
  });
}

writeLog("Use a local signer, fund it, then submit an XRPL deposit.");

window.addEventListener("pagehide", () => {
  void disposeDepositor().catch(console.error);
});

async function connectLocalSigner(): Promise<void> {
  setBusy(localSignerButton, true);
  try {
    const localSigner = createLocalXrplSigner(readOptional("local-seed"));
    await replaceSigner(localSigner);
    if (localSigner.seed !== undefined) {
      setInput("local-seed", localSigner.seed);
    }
    writeLog(
      `Using local signer ${localSigner.classicAddress}\n` +
        "Click Fund Wallet before submitting on the local devnet.",
    );
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(localSignerButton, false);
  }
}

async function connectGemWallet(): Promise<void> {
  setBusy(gemWalletButton, true);
  try {
    writeLog("Waiting for GemWallet address approval...");
    const installed = await isInstalled();
    if (installed.result.isInstalled !== true) {
      throw new Error("GemWallet extension is not installed");
    }
    const response = await withTimeout(
      getAddress(),
      "GemWallet address approval",
      GEMWALLET_ADDRESS_TIMEOUT_MS,
    );
    const address = response.result?.address;
    if (address === undefined || address === "") {
      throw new Error("GemWallet did not return an address");
    }
    await replaceSigner(new GemWalletSigner(address));
    writeLog(
      `Connected GemWallet ${address}\n` +
        "Network will be verified before signing.",
    );
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(gemWalletButton, false);
  }
}

async function fundWallet(): Promise<void> {
  if (signer === undefined) {
    await connectLocalSigner();
  }
  if (signer === undefined) {
    return;
  }

  setBusy(fundButton, true);
  const client = new Client(readInput("rpc-url"));
  try {
    const master = Wallet.fromSeed(LOCAL_XRPL_GENESIS_SEED);
    await client.connect();
    const prepared = await client.autofill({
      TransactionType: "Payment",
      Account: master.classicAddress,
      Destination: signer.classicAddress,
      Amount: readInput("fund-drops"),
    });
    const signed = master.sign(prepared);
    const result = await client.submit(signed.tx_blob, { autofill: false });
    const engineResult = result.result.engine_result;
    if (engineResult !== "tesSUCCESS" && engineResult !== "terQUEUED") {
      throw new Error(`Fund rejected: ${engineResult}`);
    }
    await ledgerAccept();
    const account = await client.request({
      command: "account_info",
      account: signer.classicAddress,
      ledger_index: "validated",
    });
    writeLog(
      `Funded ${signer.classicAddress}\n` +
        `hash: ${signed.hash}\n` +
        `balance: ${account.result.account_data.Balance} drops`,
    );
  } catch (error) {
    writeError(error);
  } finally {
    if (client.isConnected()) {
      await client.disconnect();
    }
    setBusy(fundButton, false);
  }
}

async function submitDeposit(): Promise<void> {
  if (signer === undefined) {
    await connectLocalSigner();
  }
  if (signer === undefined) {
    return;
  }

  setBusy(submitButton, true);
  let activeDepositor: XrplVaultDepositor | undefined;
  try {
    if (signer instanceof GemWalletSigner) {
      await assertGemWalletMatchesApp();
    }

    const ref = readOptional("reference");
    const maxFeeDrops = readOptional("max-fee-drops");
    await clearSubmittedDeposit();
    activeDepositor = new XrplVaultDepositor({
      rpcUrl: readInput("rpc-url"),
      vaultAddress: readInput("vault-address"),
      signer,
      ...(maxFeeDrops === undefined
        ? {}
        : { maxFeeDrops: BigInt(maxFeeDrops) }),
    });
    depositor = activeDepositor;
    const asset = readInput("asset");
    const submittedRef = await activeDepositor.submitDeposit(
      isNativeAsset(asset)
        ? {
            destination: {
              account: readInput("account"),
              ...(ref === undefined ? {} : { ref: ref as Bytes32Hex }),
            },
            asset: asset === "" ? "" : XRPL_NATIVE_ASSET,
            amount: BigInt(readInput("amount")),
          }
        : {
            destination: {
              account: readInput("account"),
              ...(ref === undefined ? {} : { ref: ref as Bytes32Hex }),
            },
            asset: asset as `${string}.${string}` | `${string}:${string}`,
            amount: BigInt(readInput("amount")),
          },
      {
        onSubmitted(ref) {
          if (depositor === activeDepositor) {
            lastRef = ref;
            writeLog(`Submitted ${ref.raw}\nhash: ${ref.hash}`);
          }
        },
      },
    );
    await ledgerAccept();
    if (depositor === activeDepositor) {
      lastRef = submittedRef;
      verifyButton.disabled = false;
      writeLog(`Accepted ${submittedRef.raw}\nhash: ${submittedRef.hash}`);
    }
  } catch (error) {
    const txRef = errorTxRef(error);
    if (
      lastRef !== undefined &&
      activeDepositor !== undefined &&
      depositor === activeDepositor
    ) {
      verifyButton.disabled = false;
    }
    writeError(error, txRef === undefined ? undefined : `TxRef ${txRef.raw}`);
  } finally {
    setBusy(submitButton, false);
  }
}

async function verifyLastTx(): Promise<void> {
  if (lastRef === undefined || depositor === undefined) {
    return;
  }

  setBusy(verifyButton, true);
  try {
    const status = await depositor.verifyDeposit(lastRef, 0);
    writeLog(`Verify ${lastRef.raw}\nstatus: ${status}`);
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(verifyButton, false);
  }
}

async function replaceSigner(nextSigner: XrplSigner): Promise<void> {
  await clearSubmittedDeposit();
  signer = nextSigner;
}

async function clearSubmittedDeposit(): Promise<void> {
  const cleanup = disposeDepositor();
  lastRef = undefined;
  verifyButton.disabled = true;
  await cleanup;
}

async function disposeDepositor(): Promise<void> {
  const current = depositor;
  depositor = undefined;
  if (current !== undefined) {
    await current.disconnect();
  }
}

async function ledgerAccept(): Promise<void> {
  const response = await fetch(readInput("admin-rpc-url"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ method: "ledger_accept", params: [] }),
  });
  if (!response.ok) {
    throw new Error(`ledger_accept failed with HTTP ${response.status}`);
  }
  const body = (await response.json()) as {
    result?: { status?: string };
    error?: unknown;
  };
  if (body.result?.status !== "success") {
    throw new Error(`ledger_accept failed: ${JSON.stringify(body)}`);
  }
}

class GemWalletSigner implements XrplSigner {
  constructor(readonly classicAddress: string) {}

  async sign(payment: XrplPreparedPayment): Promise<{ txBlob: string; hash: string }> {
    const transaction = { ...(payment as SubmittableTransaction) };
    // GemWallet 3.8.2 crashes its review UI for custom NetworkID values, but
    // it autofills NetworkID from the selected custom endpoint during signing.
    delete (transaction as { NetworkID?: number }).NetworkID;
    const response = await signTransaction({
      transaction,
    });
    const txBlob = response.result?.signature;
    if (txBlob == null || txBlob === "") {
      throw new Error("GemWallet did not return a signed transaction");
    }
    return {
      txBlob,
      hash: hashes.hashSignedTx(txBlob),
    };
  }
}

async function assertGemWalletMatchesApp(): Promise<{
  walletNetwork: GemWalletNetwork;
  appIdentity: NetworkIdentity;
  walletIdentity: NetworkIdentity;
}> {
  const walletNetwork = await getGemWalletNetwork();
  writeLog(
    `GemWallet selected ${describeGemWalletNetwork(walletNetwork)}\n` +
      "Checking network IDs...",
  );
  const [appIdentity, walletIdentity] = await Promise.all([
    getNetworkIdentity(readInput("rpc-url")),
    getNetworkIdentity(walletNetwork.websocket),
  ]);

  if (walletIdentity.networkId !== appIdentity.networkId) {
    throw new Error(
      `GemWallet is on ${describeGemWalletNetwork(walletNetwork)} ` +
        `(network_id ${walletIdentity.networkId}).\n` +
        `The demo RPC is ${appIdentity.url} ` +
        `(network_id ${appIdentity.networkId}).\n` +
        "Switch GemWallet to a custom WSS endpoint for this chain, or use the local signer.",
    );
  }

  return { walletNetwork, appIdentity, walletIdentity };
}

async function getGemWalletNetwork(): Promise<GemWalletNetwork> {
  const response = await withTimeout(
    getNetwork(),
    "GemWallet network check",
    GEMWALLET_NETWORK_TIMEOUT_MS,
  );
  const result = response.result;
  if (
    result === undefined ||
    result.websocket === undefined ||
    result.websocket === ""
  ) {
    throw new Error("GemWallet did not return its selected network");
  }
  return {
    chain: String(result.chain),
    network: String(result.network),
    websocket: result.websocket,
  };
}

async function getNetworkIdentity(url: string): Promise<NetworkIdentity> {
  const client = new Client(url);
  try {
    await withTimeout(
      client.connect(),
      `Connect to ${url}`,
      NETWORK_PROBE_TIMEOUT_MS,
    );
    const response = await withTimeout(
      client.request({ command: "server_info" }),
      `server_info for ${url}`,
      NETWORK_PROBE_TIMEOUT_MS,
    );
    const info = response.result.info as { network_id?: number };
    return {
      networkId: info.network_id ?? 0,
      url,
    };
  } finally {
    if (client.isConnected()) {
      await client.disconnect();
    }
  }
}

function describeGemWalletNetwork(network: GemWalletNetwork): string {
  return `${network.chain} ${network.network} (${network.websocket})`;
}

async function withTimeout<T>(
  operation: Promise<T>,
  label: string,
  timeoutMs: number,
): Promise<T> {
  let timeoutId: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      operation,
      new Promise<never>((_, reject) => {
        timeoutId = setTimeout(() => {
          reject(new Error(`${label} timed out after ${timeoutMs}ms`));
        }, timeoutMs);
      }),
    ]);
  } finally {
    if (timeoutId !== undefined) {
      clearTimeout(timeoutId);
    }
  }
}

function isNativeAsset(asset: string): boolean {
  return asset === "" || asset.toUpperCase() === XRPL_NATIVE_ASSET;
}

function readInput(id: string): string {
  return mustElement<HTMLInputElement>(id).value.trim();
}

function setInput(id: string, value: string): void {
  mustElement<HTMLInputElement>(id).value = value;
}

function readOptional(id: string): string | undefined {
  const value = readInput(id);
  return value === "" ? undefined : value;
}

function mustElement<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (element === null) {
    throw new Error(`Missing element #${id}`);
  }
  return element as T;
}

function setBusy(button: HTMLButtonElement, busy: boolean): void {
  button.disabled = busy;
}

function writeLog(message: string): void {
  logOutput.value = message;
}

function writeError(error: unknown, prefix?: string): void {
  const message =
    error instanceof Error ? error.message : JSON.stringify(error, null, 2);
  writeLog(prefix === undefined ? message : `${prefix}\n${message}`);
}

function errorTxRef(error: unknown): TxRef | undefined {
  if (error && typeof error === "object" && "txRef" in error) {
    return (error as { txRef?: TxRef }).txRef;
  }
  return undefined;
}
