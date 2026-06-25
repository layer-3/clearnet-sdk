import { Buffer } from "buffer";

import {
  Connection,
  Keypair,
  PublicKey,
  sendAndConfirmTransaction,
  SystemProgram,
  Transaction,
  TransactionInstruction,
} from "@solana/web3.js";

import {
  SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID,
  SOLANA_TOKEN_PROGRAM_ID,
} from "../../../src/blockchain/sol/constants.js";

const MINT_ACCOUNT_SIZE = 82;
const INITIALIZE_MINT2_INSTRUCTION = 20;
const CREATE_IDEMPOTENT_ATA_INSTRUCTION = 1;
const MINT_TO_INSTRUCTION = 7;
const TOKEN_AMOUNT_OFFSET = 1;
const TOKEN_AMOUNT_BYTES = 8;
const TOKEN_AMOUNT_LENGTH = TOKEN_AMOUNT_OFFSET + TOKEN_AMOUNT_BYTES;

const TOKEN_PROGRAM_ID = new PublicKey(SOLANA_TOKEN_PROGRAM_ID);
const ASSOCIATED_TOKEN_PROGRAM_ID = new PublicKey(
  SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID,
);

export function getAssociatedTokenAddress(
  mint: PublicKey,
  owner: PublicKey,
): PublicKey {
  return PublicKey.findProgramAddressSync(
    [owner.toBuffer(), TOKEN_PROGRAM_ID.toBuffer(), mint.toBuffer()],
    ASSOCIATED_TOKEN_PROGRAM_ID,
  )[0];
}

export async function createMint(
  connection: Connection,
  payer: Keypair,
  mintAuthority: PublicKey,
  decimals: number,
): Promise<PublicKey> {
  const mint = Keypair.generate();
  const lamports =
    await connection.getMinimumBalanceForRentExemption(MINT_ACCOUNT_SIZE);
  const transaction = new Transaction().add(
    SystemProgram.createAccount({
      fromPubkey: payer.publicKey,
      newAccountPubkey: mint.publicKey,
      lamports,
      space: MINT_ACCOUNT_SIZE,
      programId: TOKEN_PROGRAM_ID,
    }),
    new TransactionInstruction({
      programId: TOKEN_PROGRAM_ID,
      keys: [{ pubkey: mint.publicKey, isSigner: false, isWritable: true }],
      data: Buffer.concat([
        Buffer.from([INITIALIZE_MINT2_INSTRUCTION, decimals]),
        mintAuthority.toBuffer(),
        Buffer.from([0]),
      ]),
    }),
  );
  await sendAndConfirmTransaction(connection, transaction, [payer, mint], {
    commitment: "confirmed",
  });
  return mint.publicKey;
}

export async function createAssociatedTokenAccountIdempotent(
  connection: Connection,
  payer: Keypair,
  mint: PublicKey,
  owner: PublicKey,
): Promise<PublicKey> {
  const ata = getAssociatedTokenAddress(mint, owner);
  const transaction = new Transaction().add(
    new TransactionInstruction({
      programId: ASSOCIATED_TOKEN_PROGRAM_ID,
      keys: [
        { pubkey: payer.publicKey, isSigner: true, isWritable: true },
        { pubkey: ata, isSigner: false, isWritable: true },
        { pubkey: owner, isSigner: false, isWritable: false },
        { pubkey: mint, isSigner: false, isWritable: false },
        { pubkey: SystemProgram.programId, isSigner: false, isWritable: false },
        { pubkey: TOKEN_PROGRAM_ID, isSigner: false, isWritable: false },
      ],
      data: Buffer.from([CREATE_IDEMPOTENT_ATA_INSTRUCTION]),
    }),
  );
  await sendAndConfirmTransaction(connection, transaction, [payer], {
    commitment: "confirmed",
  });
  return ata;
}

export async function mintTo(
  connection: Connection,
  payer: Keypair,
  mint: PublicKey,
  destination: PublicKey,
  authority: Keypair,
  amount: bigint,
): Promise<void> {
  const data = Buffer.alloc(TOKEN_AMOUNT_LENGTH);
  data[0] = MINT_TO_INSTRUCTION;
  data.writeBigUInt64LE(amount, TOKEN_AMOUNT_OFFSET);
  const transaction = new Transaction().add(
    new TransactionInstruction({
      programId: TOKEN_PROGRAM_ID,
      keys: [
        { pubkey: mint, isSigner: false, isWritable: true },
        { pubkey: destination, isSigner: false, isWritable: true },
        { pubkey: authority.publicKey, isSigner: true, isWritable: false },
      ],
      data,
    }),
  );
  await sendAndConfirmTransaction(
    connection,
    transaction,
    uniqueSigners([payer, authority]),
    { commitment: "confirmed" },
  );
}

export async function tokenBalance(
  connection: Connection,
  ata: PublicKey,
): Promise<bigint> {
  const account = await connection.getAccountInfo(ata);
  if (account === null) {
    return 0n;
  }
  const balance = await connection.getTokenAccountBalance(ata);
  return BigInt(balance.value.amount);
}

function uniqueSigners(signers: Keypair[]): Keypair[] {
  const out = new Map<string, Keypair>();
  for (const signer of signers) {
    out.set(signer.publicKey.toBase58(), signer);
  }
  return [...out.values()];
}
