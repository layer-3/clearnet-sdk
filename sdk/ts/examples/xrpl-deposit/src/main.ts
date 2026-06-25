import {
  getAddress,
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
import { hashes, type SubmittableTransaction } from "xrpl";

const form = mustElement<HTMLFormElement>("deposit-form");
const connectButton = mustElement<HTMLButtonElement>("connect");
const submitButton = mustElement<HTMLButtonElement>("submit");
const verifyButton = mustElement<HTMLButtonElement>("verify");
const logOutput = mustElement<HTMLOutputElement>("log");

let signer: GemWalletSigner | undefined;
let lastRef: TxRef | undefined;

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

writeLog("Connect GemWallet to submit an XRPL deposit.");

async function connectWallet(): Promise<void> {
  setBusy(connectButton, true);
  try {
    const installed = await isInstalled();
    if (installed.result.isInstalled !== true) {
      throw new Error("GemWallet extension is not installed");
    }
    const response = await getAddress();
    const address = response.result?.address;
    if (address === undefined || address === "") {
      throw new Error("GemWallet did not return an address");
    }
    signer = new GemWalletSigner(address);
    writeLog(`Connected GemWallet ${address}`);
  } catch (error) {
    writeError(error);
  } finally {
    setBusy(connectButton, false);
  }
}

async function submitDeposit(): Promise<void> {
  if (signer === undefined) {
    await connectWallet();
  }
  if (signer === undefined) {
    return;
  }

  const ref = readOptional("reference");
  const maxFeeDrops = readOptional("max-fee-drops");
  const depositor = new XrplVaultDepositor({
    rpcUrl: readInput("rpc-url"),
    vaultAddress: readInput("vault-address"),
    signer,
    ...(maxFeeDrops === undefined ? {} : { maxFeeDrops: BigInt(maxFeeDrops) }),
  });

  setBusy(submitButton, true);
  try {
    lastRef = await depositor.submitDeposit(
      {
        destination: {
          account: readInput("account"),
          ...(ref === undefined ? {} : { ref: ref as Bytes32Hex }),
        },
        asset: readInput("asset"),
        amount: readAmount(),
      },
      {
        onSubmitted(ref) {
          lastRef = ref;
          verifyButton.disabled = false;
          writeLog(`Submitted ${ref.raw}\nhash: ${ref.hash}`);
        },
      },
    );
    verifyButton.disabled = false;
    writeLog(`Accepted ${lastRef.raw}\nhash: ${lastRef.hash}`);
  } catch (error) {
    const txRef = errorTxRef(error);
    writeError(error, txRef === undefined ? undefined : `Submitted ${txRef.raw}`);
  } finally {
    setBusy(submitButton, false);
  }
}

async function verifyLastTx(): Promise<void> {
  if (lastRef === undefined || signer === undefined) {
    return;
  }
  const maxFeeDrops = readOptional("max-fee-drops");
  const depositor = new XrplVaultDepositor({
    rpcUrl: readInput("rpc-url"),
    vaultAddress: readInput("vault-address"),
    signer,
    ...(maxFeeDrops === undefined ? {} : { maxFeeDrops: BigInt(maxFeeDrops) }),
  });

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

class GemWalletSigner implements XrplSigner {
  constructor(readonly classicAddress: string) {}

  async sign(payment: XrplPreparedPayment): Promise<{ txBlob: string; hash: string }> {
    const response = await signTransaction({
      transaction: payment as SubmittableTransaction,
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

function readAmount(): bigint | string {
  const asset = readInput("asset");
  const amount = readInput("amount");
  if (asset === "" || asset.toUpperCase() === XRPL_NATIVE_ASSET) {
    return BigInt(amount);
  }
  return amount;
}

function readInput(id: string): string {
  return mustElement<HTMLInputElement>(id).value.trim();
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
