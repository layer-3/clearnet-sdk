import bs58 from "bs58";
import { sha256 } from "@noble/hashes/sha2.js";
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
  TxRef,
  VaultDepositor,
} from "../../core/types.js";
import {
  DEFAULT_RECEIPT_TIMEOUT_MS,
  DEPOSIT_SOL_DISCRIMINATOR,
  DEPOSIT_SPL_DISCRIMINATOR,
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
  bytes32Hex,
  normalizeCommitment,
  normalizeMinConfirmations,
  publicKeyFromString,
  requireAmount,
  requireClearnetAccount,
  requireProgramId,
  requireReceiptTimeout,
  requireReference,
  requireRpcUrl,
  requireSigner,
  requireTxRef,
  resolveMint,
} from "./validation.js";

type SignatureStatusValue = Awaited<
  ReturnType<Connection["getSignatureStatuses"]>
>["value"][number];

export class SolanaVaultDepositor
  implements VaultDepositor<SolanaSubmitDepositInput>
{
  private readonly rpcUrl: string;
  private readonly signer: SolanaSigner;
  private readonly depositor: PublicKey;
  private readonly programId: PublicKey;
  private readonly commitment: SolanaCommitment;
  private readonly receiptTimeoutMs: number;
  private readonly connection: Connection;

  constructor(config: SolanaDepositorConfig) {
    this.rpcUrl = requireRpcUrl(config.rpcUrl);
    this.signer = requireSigner(config.signer);
    this.depositor = publicKeyFromString(this.signer.publicKey, "signer.publicKey");
    this.programId = requireProgramId(config.programId);
    this.commitment = normalizeCommitment(config.commitment);
    this.receiptTimeoutMs =
      config.receiptTimeoutMs === undefined
        ? DEFAULT_RECEIPT_TIMEOUT_MS
        : requireReceiptTimeout(config.receiptTimeoutMs);
    this.connection = new Connection(this.rpcUrl, {
      commitment: this.commitment,
      fetch: (input, init) => globalThis.fetch(input, init),
    });
  }

  async submitDeposit(
    input: SolanaSubmitDepositInput,
    options: SubmitDepositOptions = {},
  ): Promise<TxRef> {
    const account = requireClearnetAccount(input.destination.account);
    const reference = requireReference(input.destination.ref);
    const amount = requireAmount(input.amount);
    const mint = resolveMint(input.asset);
    const transaction = new Transaction();
    transaction.feePayer = this.depositor;
    transaction.add(
      mint === undefined
        ? this.depositSolInstruction(account, reference, amount)
        : this.depositSplInstruction(mint, account, reference, amount),
    );

    const signature = await this.signAndSend(transaction);
    const ref = txRef(signature);
    options.onSubmitted?.(ref);
    await this.waitForCommitment(signature, ref, options);
    return ref;
  }

  async verifyDeposit(
    ref: TxRef,
    minConfirmations: bigint | number,
  ): Promise<DepositStatus> {
    requireTxRef(ref);
    const minConf = normalizeMinConfirmations(minConfirmations);
    const status = await this.getSignatureStatus(ref.raw, undefined);
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
        { pubkey: vaultPda(this.programId), isSigner: false, isWritable: true },
        { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
        {
          pubkey: eventAuthorityPda(this.programId),
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
    const vault = vaultPda(this.programId);
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
        { pubkey: vault, isSigner: false, isWritable: false },
        {
          pubkey: associatedTokenAddress(vault, mint),
          isSigner: false,
          isWritable: true,
        },
        {
          pubkey: new PublicKey(SOLANA_TOKEN_PROGRAM_ID),
          isSigner: false,
          isWritable: false,
        },
        {
          pubkey: new PublicKey(SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID),
          isSigner: false,
          isWritable: false,
        },
        {
          pubkey: eventAuthorityPda(this.programId),
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

  private async waitForCommitment(
    signature: string,
    ref: TxRef,
    options: SubmitDepositOptions,
  ): Promise<void> {
    const timeoutMs = options.receiptTimeoutMs ?? this.receiptTimeoutMs;
    if (options.receiptTimeoutMs !== undefined) {
      requireReceiptTimeout(options.receiptTimeoutMs);
    }
    const deadline = Date.now() + timeoutMs;
    for (;;) {
      if (options.signal?.aborted === true) {
        throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted", {
          txRef: ref,
        });
      }
      const status = await this.getSignatureStatus(signature, ref);
      if (status?.err != null) {
        throw new ClearnetSdkError("TX_REVERTED", "sol: transaction failed", {
          txRef: ref,
        });
      }
      if (statusSatisfiesCommitment(status, this.commitment)) {
        return;
      }
      if (Date.now() >= deadline) {
        throw new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt timeout", {
          txRef: ref,
        });
      }
      await sleep(250, options.signal);
    }
  }

  private async getSignatureStatus(
    signature: string,
    txRef: TxRef | undefined,
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
        txRef === undefined ? { cause: error } : { txRef, cause: error },
      );
    }
  }
}

export function vaultPda(programId = new PublicKey(SOLANA_CUSTODY_PROGRAM_ID)): PublicKey {
  return PublicKey.findProgramAddressSync(
    [new TextEncoder().encode("vault")],
    programId,
  )[0];
}

export function eventAuthorityPda(
  programId = new PublicKey(SOLANA_CUSTODY_PROGRAM_ID),
): PublicKey {
  return PublicKey.findProgramAddressSync(
    [new TextEncoder().encode("__event_authority")],
    programId,
  )[0];
}

function associatedTokenAddress(owner: PublicKey, mint: PublicKey): PublicKey {
  return PublicKey.findProgramAddressSync(
    [
      owner.toBytes(),
      new PublicKey(SOLANA_TOKEN_PROGRAM_ID).toBytes(),
      mint.toBytes(),
    ],
    new PublicKey(SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID),
  )[0];
}

function txRef(signature: string): TxRef {
  const signatureBytes = bs58.decode(signature);
  if (signatureBytes.length !== 64) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "Solana signature must decode to 64 bytes",
    );
  }
  return { hash: bytes32Hex(sha256(signatureBytes)), raw: signature };
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

async function sleep(ms: number, signal: AbortSignal | undefined): Promise<void> {
  await new Promise<void>((resolve, reject) => {
    if (signal?.aborted === true) {
      reject(new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted"));
      return;
    }
    const timeout = setTimeout(resolve, ms);
    signal?.addEventListener(
      "abort",
      () => {
        clearTimeout(timeout);
        reject(new ClearnetSdkError("RECEIPT_TIMEOUT", "sol: receipt aborted"));
      },
      { once: true },
    );
  });
}
