import bs58 from "bs58";
import {
  Connection,
  PublicKey,
  SystemProgram,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";

import { ClearnetSdkError } from "../../core/errors.js";
import type {
  DepositStatus,
  SubmitDepositOptions,
  VaultDepositor,
} from "../../core/types.js";
import { decimalToBaseUnits } from "../amounts.js";
import {
  DEFAULT_RECEIPT_TIMEOUT_MS,
  DEPOSIT_SOL_DISCRIMINATOR,
  DEPOSIT_SPL_DISCRIMINATOR,
  POLL_INTERVAL_MS,
  SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID,
  SOLANA_CUSTODY_PROGRAM_ID,
  SOLANA_TOKEN_PROGRAM_ID,
} from "./constants.js";
import { encodeDepositData } from "./encoding.js";
import type {
  SolanaCommitment,
  SolanaDepositorConfig,
  SolanaSubmitDepositInput,
  SolanaSigner,
} from "./types.js";
import {
  normalizeCommitment,
  normalizeMinConfirmations,
  publicKeyFromString,
  requireClearnetAccount,
  requireDepositDestination,
  requireProgramId,
  requireReceiptTimeout,
  requireReference,
  requireRpcUrl,
  requireSigner,
  requireTxID,
  resolveMint,
} from "./validation.js";

type SignatureStatusValue = Awaited<
  ReturnType<Connection["getSignatureStatuses"]>
>["value"][number];

const SOLANA_CUSTODY_PUBLIC_KEY = new PublicKey(SOLANA_CUSTODY_PROGRAM_ID);
const SOLANA_TOKEN_PROGRAM_PUBLIC_KEY = new PublicKey(SOLANA_TOKEN_PROGRAM_ID);
const SOLANA_ASSOCIATED_TOKEN_PROGRAM_PUBLIC_KEY = new PublicKey(
  SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID,
);
const SPL_MINT_SIZE = 82;
const SPL_MINT_DECIMALS_OFFSET = 44;
const VAULT_SEED = new TextEncoder().encode("vault");
const EVENT_AUTHORITY_SEED = new TextEncoder().encode("__event_authority");

export class SolanaVaultDepositor
  implements VaultDepositor<SolanaSubmitDepositInput>
{
  private readonly signer: SolanaSigner;
  private readonly depositor: PublicKey;
  private readonly programId: PublicKey;
  private readonly commitment: SolanaCommitment;
  private readonly receiptTimeoutMs: number;
  private readonly connection: Connection;
  private readonly vault: PublicKey;
  private readonly eventAuthority: PublicKey;
  private readonly mintDecimals = new Map<string, number>();

  constructor(config: SolanaDepositorConfig) {
    const rpcUrl = requireRpcUrl(config.rpcUrl);
    this.signer = requireSigner(config.signer);
    this.depositor = publicKeyFromString(this.signer.publicKey, "signer.publicKey");
    this.programId = requireProgramId(config.programId);
    this.vault = vaultPda(this.programId);
    this.eventAuthority = eventAuthorityPda(this.programId);
    this.commitment = normalizeCommitment(config.commitment);
    this.receiptTimeoutMs =
      config.receiptTimeoutMs === undefined
        ? DEFAULT_RECEIPT_TIMEOUT_MS
        : requireReceiptTimeout(config.receiptTimeoutMs);
    this.connection = new Connection(rpcUrl, {
      commitment: this.commitment,
      fetch: (input, init) => globalThis.fetch(input, init),
    });
  }

  async submitDeposit(
    input: SolanaSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<string> {
    const waitOptions = requireSubmitDepositOptions(options);
    const fields =
      input && typeof input === "object"
        ? (input as Partial<SolanaSubmitDepositInput>)
        : {};
    const destination = requireDepositDestination(fields.destination);
    const account = requireClearnetAccount(destination.account);
    const reference = requireReference(destination.ref);
    const mint = resolveMint(fields.asset);
    const amount = decimalToBaseUnits(
      fields.amount,
      mint === undefined ? 9 : await this.assetDecimals(mint),
    );
    if (amount > (1n << 64n) - 1n) {
      throw new ClearnetSdkError("INVALID_AMOUNT", "amount must fit in uint64");
    }
    validateWaitOptions(waitOptions);
    const transaction = new Transaction();
    transaction.feePayer = this.depositor;
    transaction.add(
      mint === undefined
        ? this.depositSolInstruction(account, reference, amount)
        : this.depositSplInstruction(mint, account, reference, amount),
    );

    const signature = await this.signAndSend(transaction);
    const txID = normalizeSolanaTxID(signature);
    waitOptions.onSubmitted?.(txID);
    await this.waitForCommitment(signature, txID, waitOptions);
    return txID;
  }

  async verifyDeposit(
    txID: string,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    requireTxID(txID);
    const minConf = normalizeMinConfirmations(minConfirmations);
    const status = await this.getSignatureStatus(txID, txID);
    return mapStatus(status, minConf);
  }

  private depositSolInstruction(
    account: Uint8Array,
    reference: Uint8Array,
    amount: bigint,
  ): TransactionInstruction {
    return new TransactionInstruction({
      programId: this.programId,
      keys: [
        { pubkey: this.depositor, isSigner: true, isWritable: true },
        { pubkey: this.vault, isSigner: false, isWritable: true },
        { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
        {
          pubkey: this.eventAuthority,
          isSigner: false,
          isWritable: false,
        },
        { pubkey: this.programId, isSigner: false, isWritable: false },
      ],
      data: encodeDepositData(
        DEPOSIT_SOL_DISCRIMINATOR,
        account,
        reference,
        amount,
      ),
    });
  }

  private depositSplInstruction(
    mint: PublicKey,
    account: Uint8Array,
    reference: Uint8Array,
    amount: bigint,
  ): TransactionInstruction {
    return new TransactionInstruction({
      programId: this.programId,
      keys: [
        { pubkey: this.depositor, isSigner: true, isWritable: true },
        { pubkey: mint, isSigner: false, isWritable: false },
        {
          pubkey: associatedTokenAddress(this.depositor, mint),
          isSigner: false,
          isWritable: true,
        },
        { pubkey: this.vault, isSigner: false, isWritable: false },
        {
          pubkey: associatedTokenAddress(this.vault, mint),
          isSigner: false,
          isWritable: true,
        },
        {
          pubkey: SOLANA_TOKEN_PROGRAM_PUBLIC_KEY,
          isSigner: false,
          isWritable: false,
        },
        {
          pubkey: SOLANA_ASSOCIATED_TOKEN_PROGRAM_PUBLIC_KEY,
          isSigner: false,
          isWritable: false,
        },
        {
          pubkey: this.eventAuthority,
          isSigner: false,
          isWritable: false,
        },
        { pubkey: this.programId, isSigner: false, isWritable: false },
      ],
      data: encodeDepositData(
        DEPOSIT_SPL_DISCRIMINATOR,
        account,
        reference,
        amount,
      ),
    });
  }

  private async signAndSend(transaction: Transaction): Promise<string> {
    try {
      return await this.signer.signAndSend(transaction);
    } catch (error) {
      if (error instanceof ClearnetSdkError) {
        throw error;
      }
      throw new ClearnetSdkError("RPC_ERROR", "sol: sign and send", {
        cause: error,
      });
    }
  }

  private async assetDecimals(mint: PublicKey): Promise<number> {
    const key = mint.toBase58();
    const cached = this.mintDecimals.get(key);
    if (cached !== undefined) {
      return cached;
    }
    let account;
    try {
      account = await this.connection.getAccountInfo(mint, this.commitment);
    } catch (error) {
      throw new ClearnetSdkError("RPC_ERROR", "sol: read mint", {
        cause: error,
      });
    }
    if (account === null) {
      throw new ClearnetSdkError("INVALID_ADDRESS", `sol: mint ${key} not found`);
    }
    // TODO: Verify the mint account owner is the SPL Token program before
    // trusting mint account data.
    if (account.data.length < SPL_MINT_SIZE) {
      throw new ClearnetSdkError("INVALID_ADDRESS", `sol: mint ${key} is invalid`);
    }
    const decimals = account.data[SPL_MINT_DECIMALS_OFFSET];
    if (decimals === undefined) {
      throw new ClearnetSdkError("INVALID_ADDRESS", `sol: mint ${key} is invalid`);
    }
    this.mintDecimals.set(key, decimals);
    return decimals;
  }

  private async waitForCommitment(
    signature: string,
    txID: string,
    options: SubmitDepositOptions,
  ): Promise<void> {
    const timeoutMs = requireReceiptTimeout(
      options.receiptTimeoutMs ?? this.receiptTimeoutMs,
    );
    const deadline = Date.now() + timeoutMs;
    for (;;) {
      if (options.signal?.aborted === true) {
        throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted", {
          txID,
        });
      }
      const status = await waitWithControls(
        () => this.getSignatureStatus(signature, txID),
        remainingMs(deadline, txID),
        options.signal,
        txID,
      );
      if (status?.err != null) {
        throw new ClearnetSdkError("TX_REVERTED", "sol: transaction failed", {
          txID,
        });
      }
      if (statusSatisfiesCommitment(status, this.commitment)) {
        return;
      }
      if (Date.now() >= deadline) {
        throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt timeout", {
          txID,
        });
      }
      await sleep(Math.min(POLL_INTERVAL_MS, remainingMs(deadline, txID)), options.signal, txID);
    }
  }

  private async getSignatureStatus(
    signature: string,
    txID: string | undefined,
  ): Promise<SignatureStatusValue> {
    try {
      const out = await this.connection.getSignatureStatuses([signature], {
        searchTransactionHistory: true,
      });
      return out.value[0] ?? null;
    } catch (error) {
      throw new ClearnetSdkError(
        "RPC_ERROR",
        "sol: signature status",
        txID === undefined ? { cause: error } : { txID, cause: error },
      );
    }
  }
}

export function vaultPda(programId = SOLANA_CUSTODY_PUBLIC_KEY): PublicKey {
  return PublicKey.findProgramAddressSync([VAULT_SEED], programId)[0];
}

export function eventAuthorityPda(
  programId = SOLANA_CUSTODY_PUBLIC_KEY,
): PublicKey {
  return PublicKey.findProgramAddressSync([EVENT_AUTHORITY_SEED], programId)[0];
}

function associatedTokenAddress(owner: PublicKey, mint: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync(
    [
      owner.toBytes(),
      SOLANA_TOKEN_PROGRAM_PUBLIC_KEY.toBytes(),
      mint.toBytes(),
    ],
    SOLANA_ASSOCIATED_TOKEN_PROGRAM_PUBLIC_KEY,
  )[0];
}

function normalizeSolanaTxID(signature: string): string {
  let signatureBytes: Uint8Array;
  try {
    signatureBytes = bs58.decode(signature);
  } catch (error) {
    throw new ClearnetSdkError("INVALID_TX_ID", "Solana signature must be base58", {
      cause: error,
    });
  }
  if (signatureBytes.length !== 64) {
    throw new ClearnetSdkError(
      "INVALID_TX_ID",
      "Solana signature must decode to 64 bytes",
    );
  }
  return signature;
}

function mapStatus(
  status: SignatureStatusValue,
  minConfirmations: bigint,
): DepositStatus {
  if (status == null || status.err != null) {
    return "absent";
  }
  switch (status.confirmationStatus) {
    case "finalized":
      return "confirmed";
    case "confirmed":
      return minConfirmations === 0n ? "confirmed" : "pending";
    default:
      return "pending";
  }
}

function statusSatisfiesCommitment(
  status: SignatureStatusValue,
  commitment: SolanaCommitment,
): boolean {
  if (status == null || status.err != null) {
    return false;
  }
  if (commitment === "processed") {
    return true;
  }
  if (commitment === "confirmed") {
    return (
      status.confirmationStatus === "confirmed" ||
      status.confirmationStatus === "finalized"
    );
  }
  return status.confirmationStatus === "finalized";
}

function validateWaitOptions(options: SubmitDepositOptions): void {
  if (options.receiptTimeoutMs !== undefined) {
    requireReceiptTimeout(options.receiptTimeoutMs);
  }
  if (options.signal?.aborted === true) {
    throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted");
  }
}

function requireSubmitDepositOptions(options: unknown): SubmitDepositOptions {
  if (options === null || typeof options !== "object") {
    throw new ClearnetSdkError(
      "RECEIPT_TIMEOUT",
      "submit options must be an object",
    );
  }
  return options;
}

function remainingMs(deadline: number, txID: string): number {
  const remaining = deadline - Date.now();
  if (remaining <= 0) {
    throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt timeout", {
      txID,
    });
  }
  return remaining;
}

async function waitWithControls<T>(
  wait: () => Promise<T>,
  timeoutMs: number,
  signal: AbortSignal | undefined,
  txID: string,
): Promise<T> {
  if (signal?.aborted === true) {
    throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted", {
      txID,
    });
  }

  let timeoutId: ReturnType<typeof setTimeout> | undefined;
  let abortHandler: (() => void) | undefined;

  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutId = setTimeout(() => {
      reject(
        new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt timeout", {
          txID,
        }),
      );
    }, timeoutMs);
  });

  const abortPromise =
    signal === undefined
      ? undefined
      : new Promise<never>((_, reject) => {
          abortHandler = () => {
            reject(
              new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted", {
                txID,
              }),
            );
          };
          signal.addEventListener("abort", abortHandler, { once: true });
        });

  try {
    return await Promise.race(
      abortPromise === undefined
        ? [wait(), timeoutPromise]
        : [wait(), timeoutPromise, abortPromise],
    );
  } finally {
    if (timeoutId !== undefined) {
      clearTimeout(timeoutId);
    }
    if (signal !== undefined && abortHandler !== undefined) {
      signal.removeEventListener("abort", abortHandler);
    }
  }
}

async function sleep(
  ms: number,
  signal: AbortSignal | undefined,
  txID: string,
): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    if (signal?.aborted === true) {
      reject(
        new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted", {
          txID,
        }),
      );
      return;
    }
    let abortHandler: (() => void) | undefined;
    let timeout: ReturnType<typeof setTimeout> | undefined;
    const cleanup = () => {
      if (timeout !== undefined) {
        clearTimeout(timeout);
      }
      if (signal !== undefined && abortHandler !== undefined) {
        signal.removeEventListener("abort", abortHandler);
      }
    };
    timeout = setTimeout(() => {
      cleanup();
      resolve();
    }, ms);
    if (signal !== undefined) {
      abortHandler = () => {
        cleanup();
        reject(
          new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted", {
            txID,
          }),
        );
      };
      signal.addEventListener("abort", abortHandler, { once: true });
    }
  });
}
