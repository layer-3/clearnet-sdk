import {
  BITCOIN_NATIVE_ASSET,
  BitcoinCoreRpcClient,
  BitcoinVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import type {
  BitcoinDepositorConfig,
  BitcoinNetwork,
  BitcoinSigner,
} from "@yellow-org/clearnet-sdk";

import {
  createLocalBitcoinSigner,
  generateVaultPubkeys,
} from "./local-signer.js";
import {
  configureXverseRegtestNetwork,
  connectXverseWallet,
  signPsbtWithXverse,
} from "./xverse.js";
import type { XverseWallet } from "./xverse.js";

const form = mustElement<HTMLFormElement>("deposit-form");
const generateButton = mustElement<HTMLButtonElement>("generate");
const setupButton = mustElement<HTMLButtonElement>("setup");
const submitButton = mustElement<HTMLButtonElement>("submit");
const addXverseNetworkButton = mustElement<HTMLButtonElement>("add-xverse-network");
const connectXverseButton = mustElement<HTMLButtonElement>("connect-xverse");
const fundXverseButton = mustElement<HTMLButtonElement>("fund-xverse");
const submitXverseButton = mustElement<HTMLButtonElement>("submit-xverse");
const mineButton = mustElement<HTMLButtonElement>("mine");
const verifyButton = mustElement<HTMLButtonElement>("verify");
const logOutput = mustElement<HTMLOutputElement>("log");

let signer: BitcoinSigner | undefined;
let depositor: BitcoinVaultDepositor | undefined;
let xverseWallet: XverseWallet | undefined;
let lastRef: string | undefined;

generateButton.addEventListener("click", () => {
  generateKeys();
});

setupButton.addEventListener("click", () => {
  void setupAndFund();
});

form.addEventListener("submit", (event) => {
  event.preventDefault();
  void submitLocalDeposit();
});

addXverseNetworkButton.addEventListener("click", () => {
  void addXverseNetwork();
});

connectXverseButton.addEventListener("click", () => {
  void connectXverse();
});

fundXverseButton.addEventListener("click", () => {
  void fundXverseWallet();
});

submitXverseButton.addEventListener("click", () => {
  void submitXverseDeposit();
});

mineButton.addEventListener("click", () => {
  void mineBlock();
});

verifyButton.addEventListener("click", () => {
  void verifyLastTx();
});

initializeXverseRpcUrl();
generateKeys();
writeLog("Generate keys, fund a signer, then submit a native BTC deposit.");

function initializeXverseRpcUrl(): void {
  const input = mustElement<HTMLInputElement>("xverse-rpc-url");
  if (input.value.trim() === "") {
    input.value = `${window.location.origin}/btc-rpc`;
  }
}

function generateKeys(): void {
  const local = createLocalBitcoinSigner();
  signer = local.signer;
  setInput("depositor-private-key", local.privateKeyHex);
  setTextarea("vault-pubkeys", generateVaultPubkeys(3).join("\n"));
  depositor = undefined;
  lastRef = undefined;
  verifyButton.disabled = true;
  writeLog(`Depositor public key:\n${local.publicKeyHex}`);
}

async function setupAndFund(): Promise<void> {
  setBusy(setupButton, true);
  try {
    const activeDepositor = getLocalDepositor();
    await ensureWallet();
    const miningAddress = await walletRpc<string>("getnewaddress", ["", "bech32"]);
    await rootRpc("generatetoaddress", [101, miningAddress]);
    const fundingAddress = await activeDepositor.depositorAddress();
    const depositAddress = activeDepositor.depositAddress(readInput("account"));
    await importAddress(fundingAddress);
    await importAddress(depositAddress);
    const fundTxid = await walletRpc<string>("sendtoaddress", [
      fundingAddress,
      satsToBtc(readBigInt("fund-sats")),
    ]);
    await rootRpc("generatetoaddress", [1, miningAddress]);
    const fundingUtxos = await walletRpc<readonly unknown[]>("listunspent", [
      1,
      9999999,
      [fundingAddress],
    ]);
    writeLog(
      `Funded local signer ${fundingAddress}\n` +
        `txid: ${fundTxid}\n` +
        `deposit address: ${depositAddress}\n` +
        `utxos: ${fundingUtxos.length}`,
    );
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(setupButton, false);
  }
}

async function submitLocalDeposit(): Promise<void> {
  setBusy(submitButton, true);
  try {
    const activeDepositor = getLocalDepositor();
    const ref = readOptional("reference");
    if (ref !== undefined) {
      throw new Error("Bitcoin deposits do not support a reference value");
    }
    lastRef = await activeDepositor.submitDeposit(
      {
        destination: {
          account: readInput("account"),
        },
        asset: readInput("asset") || BITCOIN_NATIVE_ASSET,
        amount: readInput("amount"),
      },
      {
        onSubmitted(ref) {
          lastRef = ref;
          verifyButton.disabled = false;
          writeLog(`Submitted ${ref}\nhash: ${ref}`);
        },
      },
    );
    verifyButton.disabled = false;
    writeLog(
      `Submitted local signer tx ${lastRef}\n` +
        `hash: ${lastRef}\n` +
        "Verify before mining for pending status.",
    );
  } catch (error) {
    const txID = errorTxID(error);
    if (txID !== undefined) {
      lastRef = txID;
      verifyButton.disabled = false;
    }
    writeError(error, txID === undefined ? undefined : `Submitted ${txID}`);
  } finally {
    setBusy(submitButton, false);
  }
}

async function addXverseNetwork(): Promise<void> {
  setBusy(addXverseNetworkButton, true);
  try {
    await configureXverseRegtestNetwork({
      name: readInput("xverse-network-name"),
      rpcUrl: readInput("xverse-rpc-url"),
    });
    writeLog(
      `Xverse network add/switch request accepted\n` +
        `name: ${readInput("xverse-network-name")}\n` +
        `rpc: ${readInput("xverse-rpc-url")}`,
    );
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(addXverseNetworkButton, false);
  }
}

async function connectXverse(): Promise<void> {
  setBusy(connectXverseButton, true);
  try {
    writeLog("Waiting for Xverse address approval...");
    xverseWallet = await connectXverseWallet();
    setInput("xverse-address", xverseWallet.address);
    setInput("xverse-public-key", xverseWallet.publicKey);
    setInput("xverse-address-type", xverseWallet.addressType);
    setInput("xverse-account-id", xverseWallet.accountId ?? "");
    writeLog(
      `Connected Xverse\n` +
        `address: ${xverseWallet.address}\n` +
        `type: ${xverseWallet.addressType}\n` +
        `network: ${xverseWallet.network ?? "unknown"}\n` +
        `publicKey: ${xverseWallet.publicKey}`,
    );
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(connectXverseButton, false);
  }
}

async function fundXverseWallet(): Promise<void> {
  setBusy(fundXverseButton, true);
  try {
    const wallet = getXverseWallet();
    assertWalletAddressMatchesNetwork(wallet.address);
    await ensureWallet();
    const miningAddress = await walletRpc<string>("getnewaddress", ["", "bech32"]);
    await rootRpc("generatetoaddress", [101, miningAddress]);
    await importAddress(wallet.address);
    const fundTxid = await walletRpc<string>("sendtoaddress", [
      wallet.address,
      satsToBtc(readBigInt("fund-sats")),
    ]);
    await rootRpc("generatetoaddress", [1, miningAddress]);
    const fundingUtxos = await walletRpc<readonly unknown[]>("listunspent", [
      1,
      9999999,
      [wallet.address],
    ]);
    writeLog(
      `Funded Xverse ${wallet.address}\n` +
        `txid: ${fundTxid}\n` +
        `utxos: ${fundingUtxos.length}`,
    );
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(fundXverseButton, false);
  }
}

async function submitXverseDeposit(): Promise<void> {
  setBusy(submitXverseButton, true);
  try {
    const wallet = getXverseWallet();
    assertWalletAddressMatchesNetwork(wallet.address);
    const activeDepositor = getWalletDepositor();
    const ref = readOptional("reference");
    if (ref !== undefined) {
      throw new Error("Bitcoin deposits do not support a reference value");
    }
    const prepared = await activeDepositor.prepareDepositPsbt(
      {
        destination: {
          account: readInput("account"),
        },
        asset: readInput("asset") || BITCOIN_NATIVE_ASSET,
        amount: readInput("amount"),
      },
      {
        publicKey: wallet.publicKey,
        address: wallet.address,
        addressType: wallet.addressType,
      },
    );
    writeLog(
      `Prepared PSBT unsigned txid ${prepared.unsignedTxID}\n` +
        `inputs: ${prepared.inputIndexesToSign.join(", ")}\n` +
        "Waiting for Xverse signature...",
    );
    const signedPsbt = await signPsbtWithXverse({
      psbtHex: prepared.psbtHex,
      inputIndexesToSign: prepared.inputIndexesToSign,
      address: wallet.address,
    });
    lastRef = await activeDepositor.submitSignedDepositPsbt(signedPsbt, {
      onSubmitted(ref) {
        lastRef = ref;
        verifyButton.disabled = false;
        writeLog(`Submitted Xverse tx ${ref}\nhash: ${ref}`);
      },
    });
    depositor = activeDepositor;
    verifyButton.disabled = false;
    writeLog(
      `Submitted Xverse tx ${lastRef}\n` +
        `hash: ${lastRef}\n` +
        "Verify before mining for pending status.",
    );
  } catch (error) {
    const txID = errorTxID(error);
    if (txID !== undefined) {
      lastRef = txID;
      verifyButton.disabled = false;
    }
    writeError(error, txID === undefined ? undefined : `Submitted ${txID}`);
  } finally {
    setBusy(submitXverseButton, false);
  }
}

async function mineBlock(): Promise<void> {
  setBusy(mineButton, true);
  try {
    await ensureWallet();
    const miningAddress = await walletRpc<string>("getnewaddress", ["", "bech32"]);
    await rootRpc("generatetoaddress", [1, miningAddress]);
    writeLog(`Mined 1 block to ${miningAddress}`);
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(mineButton, false);
  }
}

async function verifyLastTx(): Promise<void> {
  if (lastRef === undefined) {
    return;
  }
  setBusy(verifyButton, true);
  try {
    const status = await (depositor ?? getWalletDepositor()).verifyDeposit(
      lastRef,
      readBigInt("min-confirmations"),
    );
    writeLog(`Verify ${lastRef}\nstatus: ${status}`);
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(verifyButton, false);
  }
}

function getLocalDepositor(): BitcoinVaultDepositor {
  const privateKey = readOptional("depositor-private-key");
  const local = createLocalBitcoinSigner(privateKey);
  signer = local.signer;
  setInput("depositor-private-key", local.privateKeyHex);
  return createDepositor(signer);
}

function getWalletDepositor(): BitcoinVaultDepositor {
  return createDepositor();
}

function createDepositor(depositSigner?: BitcoinSigner): BitcoinVaultDepositor {
  const rpc = new BitcoinCoreRpcClient({
    url: readInput("rpc-url"),
    wallet: readInput("wallet-name"),
  });
  const config: BitcoinDepositorConfig = {
    network: readBitcoinNetwork(),
    rpc,
    vaultPubkeys: readVaultPubkeys(),
    threshold: readNumber("threshold"),
    minFundingConfirmations: 1,
    fallbackFeeRateSatPerVByte: readBigInt("fee-rate"),
  };
  if (depositSigner !== undefined) {
    config.signer = depositSigner;
  }
  depositor = new BitcoinVaultDepositor(config);
  return depositor;
}

function getXverseWallet(): XverseWallet {
  const address = readInput("xverse-address");
  const publicKey = readInput("xverse-public-key");
  const addressType = readInput("xverse-address-type");
  if (address === "" || publicKey === "") {
    throw new Error("Connect Xverse before using the Xverse signer");
  }
  if (addressType !== "p2wpkh" && addressType !== "p2sh") {
    throw new Error("Xverse address type must be p2wpkh or p2sh");
  }
  const wallet: XverseWallet = {
    address,
    publicKey,
    addressType,
    accountId: readOptional("xverse-account-id"),
    network: readOptional("xverse-network") ?? xverseWallet?.network,
  };
  xverseWallet = wallet;
  return wallet;
}

async function ensureWallet(): Promise<void> {
  const wallet = readInput("wallet-name");
  try {
    await rootRpc("createwallet", [wallet, false, false, "", false, false]);
  } catch (error) {
    const message = error instanceof Error ? error.message.toLowerCase() : "";
    if (message.includes("already loaded")) {
      return;
    }
    if (!message.includes("already exists")) {
      throw error;
    }
    try {
      await rootRpc("loadwallet", [wallet]);
    } catch (loadError) {
      const loadMessage =
        loadError instanceof Error ? loadError.message.toLowerCase() : "";
      if (!loadMessage.includes("already loaded")) {
        throw loadError;
      }
    }
  }
}

async function importAddress(address: string): Promise<void> {
  try {
    await walletRpc("importaddress", [address, "", false]);
  } catch (error) {
    const message = error instanceof Error ? error.message.toLowerCase() : "";
    if (!message.includes("already have")) {
      throw error;
    }
  }
}

async function rootRpc<T = unknown>(
  method: string,
  params: readonly unknown[] = [],
): Promise<T> {
  return rpcCall(readInput("rpc-url"), method, params);
}

async function walletRpc<T = unknown>(
  method: string,
  params: readonly unknown[] = [],
): Promise<T> {
  const wallet = encodeURIComponent(readInput("wallet-name"));
  return rpcCall(`${readInput("rpc-url")}/wallet/${wallet}`, method, params);
}

async function rpcCall<T>(
  url: string,
  method: string,
  params: readonly unknown[],
): Promise<T> {
  const response = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      jsonrpc: "1.0",
      id: "demo",
      method,
      params,
    }),
  });
  const envelope = (await response.json()) as {
    result?: T;
    error?: { code: number; message: string } | null;
  };
  if (envelope.error !== null && envelope.error !== undefined) {
    throw new Error(`bitcoind ${method} error ${envelope.error.code}: ${envelope.error.message}`);
  }
  return envelope.result as T;
}

function readBitcoinNetwork(): BitcoinNetwork {
  const value = readInput("network");
  if (
    value !== "mainnet" &&
    value !== "testnet" &&
    value !== "signet" &&
    value !== "regtest"
  ) {
    throw new Error("Network must be mainnet, testnet, signet, or regtest");
  }
  return value;
}

function assertWalletAddressMatchesNetwork(address: string): void {
  const normalized = address.toLowerCase();
  const isRegtestSegwit = normalized.startsWith("bcrt1");
  const isRegtestNestedSegwit = normalized.startsWith("2");
  if (
    readBitcoinNetwork() === "regtest" &&
    !isRegtestSegwit &&
    !isRegtestNestedSegwit
  ) {
    throw new Error(
      `Xverse returned ${address}. Local regtest can only fund and spend bcrt1 Native SegWit or 2... nested-SegWit addresses.`,
    );
  }
}

function readVaultPubkeys(): readonly string[] {
  const keys = mustElement<HTMLTextAreaElement>("vault-pubkeys")
    .value.split(/\s+/)
    .map((key) => key.trim())
    .filter((key) => key !== "");
  if (keys.length === 0) {
    throw new Error("At least one vault public key is required");
  }
  return keys;
}

function satsToBtc(sats: bigint): string {
  const sign = sats < 0n ? "-" : "";
  const absolute = sats < 0n ? -sats : sats;
  const whole = absolute / 100_000_000n;
  const frac = (absolute % 100_000_000n).toString().padStart(8, "0");
  return `${sign}${whole}.${frac}`;
}

function readInput(id: string): string {
  return mustElement<HTMLInputElement>(id).value.trim();
}

function setInput(id: string, value: string): void {
  mustElement<HTMLInputElement>(id).value = value;
}

function setTextarea(id: string, value: string): void {
  mustElement<HTMLTextAreaElement>(id).value = value;
}

function readOptional(id: string): string | undefined {
  const value = readInput(id);
  return value === "" ? undefined : value;
}

function readNumber(id: string): number {
  const value = Number(readInput(id));
  if (!Number.isSafeInteger(value)) {
    throw new Error(`${id} must be a safe integer`);
  }
  return value;
}

function readBigInt(id: string): bigint {
  const value = readInput(id);
  if (!/^(?:0|[1-9][0-9]*)$/.test(value)) {
    throw new Error(`${id} must be a non-negative integer`);
  }
  return BigInt(value);
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

function errorTxID(error: unknown): string | undefined {
  if (error && typeof error === "object" && "txID" in error) {
    return (error as { txID?: string }).txID;
  }
  return undefined;
}
