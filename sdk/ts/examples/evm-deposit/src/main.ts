import { EVM_NATIVE_ASSET, EvmVaultDepositor } from "@yellow-org/clearnet-sdk";
import {
  createPublicClient,
  createWalletClient,
  custom,
  formatEther,
  getAddress,
  http,
  parseUnits,
} from "viem";
import type { Address, Hash } from "viem";

interface Eip1193Provider {
  request(args: { method: string; params?: unknown[] }): Promise<unknown>;
}

declare global {
  interface Window {
    ethereum?: Eip1193Provider;
  }
}

const form = mustElement<HTMLFormElement>("deposit-form");
const connectButton = mustElement<HTMLButtonElement>("connect");
const submitButton = mustElement<HTMLButtonElement>("submit");
const verifyButton = mustElement<HTMLButtonElement>("verify");
const logOutput = mustElement<HTMLOutputElement>("log");

let walletAccount: Address | undefined;
let lastRef: string | undefined;

connectButton.addEventListener("click", () => {
  void connectWallet();
});

form.addEventListener("submit", (event) => {
  event.preventDefault();
  void submitDeposit();
});

verifyButton.addEventListener("click", () => {
  void verifyLastTx();
});

writeLog("Connect a wallet to the configured local Anvil network.");

async function connectWallet(): Promise<void> {
  const provider = requireProvider();
  setBusy(connectButton, true);
  try {
    const chainId = readChainId();
    const rpcUrl = readInput("rpc-url");
    await requireConfiguredRpcChain(rpcUrl, chainId);
    await ensureWalletChain(provider, chainId, rpcUrl);
    const accounts = await provider.request({ method: "eth_requestAccounts" });
    if (!Array.isArray(accounts) || typeof accounts[0] !== "string") {
      throw new Error("wallet did not return an account");
    }
    walletAccount = getAddress(accounts[0]);
    setInput("account", walletAccount);
    const balanceMessage = await walletBalanceMessage(
      provider,
      walletAccount,
      readInput("rpc-url"),
    );
    writeLog(`Connected ${walletAccount}\n${balanceMessage}`);
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(connectButton, false);
  }
}

async function submitDeposit(): Promise<void> {
  if (walletAccount === undefined) {
    await connectWallet();
  }
  if (walletAccount === undefined) {
    return;
  }

  const provider = requireProvider();
  const chainId = readChainId();
  const rpcUrl = readInput("rpc-url");
  const custodyAddress = getAddress(readInput("custody-address"));
  const account = getAddress(readInput("account"));
  const assetInput = readInput("asset");
  const asset = assetInput === "" ? EVM_NATIVE_ASSET : getAddress(assetInput);
  const ref = readOptionalHash("reference");
  const amount = readInput("amount");
  const publicClient = createPublicClient({ transport: http(rpcUrl) });
  const walletClient = createWalletClient({
    account: walletAccount,
    transport: custom(provider),
  });
  const depositor = new EvmVaultDepositor({
    publicClient,
    walletClient,
    walletAccount,
    custodyAddress,
    chainId,
    nativeDecimals: readInt("decimals"),
  });

  setBusy(submitButton, true);
  try {
    await requireConfiguredRpcChain(rpcUrl, chainId);
    lastRef = await depositor.submitDeposit(
      {
        destination: {
          account,
          ...(ref === undefined ? {} : { ref }),
        },
        asset,
        amount,
      },
      {
        onSubmitted(ref) {
          lastRef = ref;
          verifyButton.disabled = false;
          writeLog(`Submitted ${ref}`);
        },
      },
    );
    verifyButton.disabled = false;
    writeLog(`Mined ${lastRef}\nraw: ${lastRef}`);
  } catch (error) {
    const txHash = errorTxHash(error);
    writeError(error, txHash === undefined ? undefined : `Submitted ${txHash}`);
  } finally {
    setBusy(submitButton, false);
  }
}

async function verifyLastTx(): Promise<void> {
  if (lastRef === undefined) {
    return;
  }
  if (walletAccount === undefined) {
    await connectWallet();
  }
  if (walletAccount === undefined) {
    return;
  }
  const chainId = readChainId();
  const rpcUrl = readInput("rpc-url");
  await requireConfiguredRpcChain(rpcUrl, chainId);
  const publicClient = createPublicClient({
    transport: http(rpcUrl),
  });
  const provider = requireProvider();
  const walletClient = createWalletClient({
    account: walletAccount,
    transport: custom(provider),
  });
  const depositor = new EvmVaultDepositor({
    publicClient,
    walletClient,
    walletAccount,
    custodyAddress: getAddress(readInput("custody-address")),
    chainId,
  });

  setBusy(verifyButton, true);
  try {
    const status = await depositor.verifyDeposit(lastRef, 1);
    writeLog(`Verify ${lastRef}\nstatus: ${status}`);
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(verifyButton, false);
  }
}

async function ensureWalletChain(
  provider: Eip1193Provider,
  chainId: number,
  rpcUrl: string,
): Promise<void> {
  const hexChainId = `0x${chainId.toString(16)}`;
  try {
    await provider.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: hexChainId }],
    });
  } catch (error) {
    if (errorCode(error) !== 4902) {
      throw error;
    }
    await provider.request({
      method: "wallet_addEthereumChain",
      params: [
        {
          chainId: hexChainId,
          chainName: "Anvil",
          nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
          rpcUrls: [rpcUrl],
        },
      ],
    });
  }
}

async function requireConfiguredRpcChain(
  rpcUrl: string,
  chainId: number,
): Promise<void> {
  const publicClient = createPublicClient({ transport: http(rpcUrl) });
  const rpcChainId = await publicClient.getChainId();
  if (rpcChainId !== chainId) {
    throw new Error(
      `RPC URL reports chain ${rpcChainId}, but Chain ID is ${chainId}. Update both fields to the same network.`,
    );
  }
}

function requireProvider(): Eip1193Provider {
  if (window.ethereum === undefined) {
    throw new Error("No EIP-1193 wallet found");
  }
  return window.ethereum;
}

function errorTxHash(error: unknown): Hash | undefined {
  if (error && typeof error === "object" && "txID" in error) {
    const txID = (error as { txID?: string }).txID;
    return txID?.split("/", 1)[0] as Hash | undefined;
  }
  return undefined;
}

async function walletBalanceMessage(
  provider: Eip1193Provider,
  account: Address,
  rpcUrl: string,
): Promise<string> {
  const publicClient = createPublicClient({ transport: http(rpcUrl) });
  const [walletBalanceHex, configuredRpcBalance] = await Promise.all([
    provider.request({
      method: "eth_getBalance",
      params: [account, "latest"],
    }),
    publicClient.getBalance({ address: account }),
  ]);
  const walletBalance =
    typeof walletBalanceHex === "string" ? BigInt(walletBalanceHex) : undefined;
  const walletText =
    walletBalance === undefined ? "unknown" : `${formatEther(walletBalance)} ETH`;
  const configuredText = `${formatEther(configuredRpcBalance)} ETH`;
  if (walletBalance !== configuredRpcBalance) {
    return [
      `Wallet RPC balance: ${walletText}`,
      `Configured RPC balance: ${configuredText}`,
      "Wallet network RPC does not match the RPC URL above.",
    ].join("\n");
  }
  return `Wallet balance: ${walletText}`;
}

function writeError(error: unknown, prefix?: string): void {
  const message = errorMessage(error);
  const code = errorCode(error);
  const codeText = code === undefined ? "" : ` [${String(code)}]`;
  writeLog([prefix, `${codeText} ${message}`.trim()].filter(Boolean).join("\n"));
}

function errorCode(error: unknown): number | string | undefined {
  if (error && typeof error === "object" && "code" in error) {
    const code = (error as { code?: unknown }).code;
    if (typeof code === "number" || typeof code === "string") {
      return code;
    }
  }
  return undefined;
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (error && typeof error === "object" && "message" in error) {
    const message = (error as { message?: unknown }).message;
    if (typeof message === "string") {
      return message;
    }
  }
  return String(error);
}

function writeLog(message: string): void {
  logOutput.value = message;
}

function setBusy(button: HTMLButtonElement, busy: boolean): void {
  button.disabled = busy;
}

function readInput(id: string): string {
  const value = mustElement<HTMLInputElement>(id).value.trim();
  if (value.length === 0) {
    throw new Error(`${id} is required`);
  }
  return value;
}

function readOptionalHash(id: string): Hash | undefined {
  const value = mustElement<HTMLInputElement>(id).value.trim();
  if (value.length === 0) {
    return undefined;
  }
  if (!/^0x[a-fA-F0-9]{64}$/.test(value)) {
    throw new Error(`${id} must be a 32-byte hex value`);
  }
  return value as Hash;
}

function readInt(id: string): number {
  const value = Number(readInput(id));
  if (!Number.isSafeInteger(value) || value < 0) {
    throw new Error(`${id} must be a non-negative integer`);
  }
  return value;
}

function readChainId(): number {
  const chainId = readInt("chain-id");
  if (chainId <= 0) {
    throw new Error("chain-id must be positive");
  }
  return chainId;
}

function setInput(id: string, value: string): void {
  mustElement<HTMLInputElement>(id).value = value;
}

function mustElement<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (element === null) {
    throw new Error(`missing element #${id}`);
  }
  return element as T;
}
