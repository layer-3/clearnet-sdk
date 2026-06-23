import {
  SOLANA_CUSTODY_PROGRAM_ID,
  SolanaVaultDepositor,
} from "@yellow-org/clearnet-sdk";
import {
  SolanaSignTransaction,
  type SolanaSignTransactionFeature,
} from "@solana/wallet-standard-features";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { getWallets } from "@wallet-standard/app";
import type {
  Wallet,
  WalletAccount,
  WalletWithFeatures,
} from "@wallet-standard/base";
import {
  StandardConnect,
  type StandardConnectFeature,
} from "@wallet-standard/features";
import type {
  SolanaCommitment,
  SolanaSigner,
  TxRef,
} from "@yellow-org/clearnet-sdk";

type SolanaWalletChain =
  | "solana:localnet"
  | "solana:devnet"
  | "solana:testnet"
  | "solana:mainnet";

type StandardSolanaWallet = WalletWithFeatures<
  StandardConnectFeature & SolanaSignTransactionFeature
>;

const form = mustElement<HTMLFormElement>("deposit-form");
const connectButton = mustElement<HTMLButtonElement>("connect");
const submitButton = mustElement<HTMLButtonElement>("submit");
const verifyButton = mustElement<HTMLButtonElement>("verify");
const logOutput = mustElement<HTMLOutputElement>("log");

let signer: BrowserSolanaSigner | undefined;
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

writeLog("Connect a Solana browser wallet to the configured RPC.");

async function connectWallet(): Promise<void> {
  setBusy(connectButton, true);
  try {
    const chain = readWalletChain();
    const wallet = requireWallet(chain);
    const result = await wallet.features[StandardConnect].connect();
    const account = firstSupportedAccount(result.accounts, chain);
    signer = new BrowserSolanaSigner(wallet, account);
    const balance = await connection().getBalance(new PublicKey(account.address));
    writeLog(
      `Connected ${wallet.name} ${account.address}\nWallet balance: ${balance} lamports`,
    );
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
  const depositor = new SolanaVaultDepositor({
    rpcUrl: readInput("rpc-url"),
    signer,
    programId: readInput("program-id"),
    commitment: readCommitment(),
  });

  setBusy(submitButton, true);
  try {
    lastRef = await depositor.submitDeposit(
      {
        destination: {
          account: readInput("account"),
          ...(ref === undefined ? {} : { ref: ref as `0x${string}` }),
        },
        asset: readInput("asset"),
        amount: BigInt(readInput("amount")),
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
    writeLog(`Confirmed ${lastRef.raw}\nhash: ${lastRef.hash}`);
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
  const depositor = new SolanaVaultDepositor({
    rpcUrl: readInput("rpc-url"),
    signer,
    programId: readInput("program-id"),
    commitment: readCommitment(),
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

class BrowserSolanaSigner implements SolanaSigner {
  constructor(
    private readonly wallet: StandardSolanaWallet,
    private readonly account: WalletAccount,
  ) {}

  get publicKey(): string {
    return this.account.address;
  }

  async signAndSend(transaction: Transaction): Promise<string> {
    const latest = await connection().getLatestBlockhash(readCommitment());
    transaction.recentBlockhash = latest.blockhash;
    transaction.feePayer ??= new PublicKey(this.publicKey);
    const [result] = await this.wallet.features[SolanaSignTransaction].signTransaction({
      account: this.account,
      chain: readWalletChain(),
      transaction: transaction.serialize({
        requireAllSignatures: false,
        verifySignatures: false,
      }),
      options: {
        preflightCommitment: readCommitment(),
      },
    });
    if (result?.signedTransaction === undefined) {
      throw new Error("wallet did not return a signed transaction");
    }
    return await connection().sendRawTransaction(result.signedTransaction, {
      preflightCommitment: readCommitment(),
    });
  }
}

function connection(): Connection {
  return new Connection(readInput("rpc-url"), readCommitment());
}

function readCommitment(): SolanaCommitment {
  const value = readInput("commitment");
  if (value !== "confirmed" && value !== "finalized") {
    throw new Error("commitment must be confirmed or finalized");
  }
  return value;
}

function readWalletChain(): SolanaWalletChain {
  const value = readInput("wallet-chain");
  if (
    value !== "solana:localnet" &&
    value !== "solana:devnet" &&
    value !== "solana:testnet" &&
    value !== "solana:mainnet"
  ) {
    throw new Error("wallet chain must be a supported Solana chain");
  }
  return value;
}

function requireWallet(chain: SolanaWalletChain): StandardSolanaWallet {
  const wallet = getWallets()
    .get()
    .find((wallet) => supportsRequiredFeatures(wallet, chain));
  if (wallet === undefined) {
    throw new Error(
      `No Wallet Standard Solana wallet found for ${chain}`,
    );
  }
  return wallet;
}

function supportsRequiredFeatures(
  wallet: Wallet,
  chain: SolanaWalletChain,
): wallet is StandardSolanaWallet {
  return (
    wallet.chains.includes(chain) &&
    StandardConnect in wallet.features &&
    SolanaSignTransaction in wallet.features
  );
}

function firstSupportedAccount(
  accounts: readonly WalletAccount[],
  chain: SolanaWalletChain,
): WalletAccount {
  const account = accounts.find(
    (account) =>
      account.chains.includes(chain) &&
      account.features.includes(SolanaSignTransaction),
  );
  if (account === undefined) {
    throw new Error(`wallet did not return an account for ${chain}`);
  }
  return account;
}

function errorTxRef(error: unknown): TxRef | undefined {
  if (error && typeof error === "object" && "txRef" in error) {
    return (error as { txRef?: TxRef }).txRef;
  }
  return undefined;
}

function writeError(error: unknown, prefix?: string): void {
  const code = errorCode(error);
  const codeText = code === undefined ? "" : ` [${String(code)}]`;
  writeLog(
    [prefix, `${codeText} ${errorMessage(error)}`.trim()]
      .filter(Boolean)
      .join("\n"),
  );
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

function readInput(id: string): string {
  return mustElement<HTMLInputElement | HTMLSelectElement>(id).value.trim();
}

function readOptional(id: string): string | undefined {
  const value = readInput(id);
  return value === "" ? undefined : value;
}

function setBusy(button: HTMLButtonElement, busy: boolean): void {
  button.disabled = busy;
}

function writeLog(message: string): void {
  logOutput.value = message;
}

function mustElement<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (element === null) {
    throw new Error(`missing #${id}`);
  }
  return element as T;
}

if (readInput("program-id") === "") {
  mustElement<HTMLInputElement>("program-id").value = SOLANA_CUSTODY_PROGRAM_ID;
}
