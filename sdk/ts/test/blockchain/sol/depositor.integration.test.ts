import {
  Connection,
  Keypair,
  LAMPORTS_PER_SOL,
  PublicKey,
  sendAndConfirmTransaction,
  Transaction,
} from "@solana/web3.js";
import bs58 from "bs58";
import { beforeAll, describe, expect, it } from "vitest";

import {
  SOLANA_NATIVE_ASSET,
  SolanaVaultDepositor,
  vaultPda,
} from "../../../src/index.js";
import type { SolanaSigner } from "../../../src/index.js";
import { DEPOSITED_EVENT_DISCRIMINATOR } from "../../../src/blockchain/sol/constants.js";
import { bytes32Hex } from "../../../src/blockchain/sol/validation.js";
import {
  createAssociatedTokenAccountIdempotent,
  createMint,
  getAssociatedTokenAddress,
  mintTo,
  tokenBalance,
} from "./spl-test-helpers.js";

const RPC_URL = process.env.SOL_RPC_URL ?? "http://127.0.0.1:8899";
const ACCOUNT = "00000000000000000000000000000000000000a1";
const REFERENCE =
  "0x3333333333333333333333333333333333333333333333333333333333333333";
const connection = new Connection(RPC_URL, "confirmed");
const DEPOSITED_EVENT_SIZE = 8 + 32 + 20 + 32 + 32 + 8;

describe("SolanaVaultDepositor validator integration", () => {
  beforeAll(async () => {
    const version = await connection.getVersion();
    expect(version["solana-core"]).toBeTruthy();
  }, 60_000);

  it("deposits native SOL and verifies the deposit tx", async () => {
    const depositorKeypair = Keypair.generate();
    await airdrop(depositorKeypair.publicKey, LAMPORTS_PER_SOL);
    const signer = new KeypairSolanaSigner(depositorKeypair);
    const depositor = new SolanaVaultDepositor({
      rpcUrl: RPC_URL,
      signer,
      commitment: "confirmed",
    });
    const vault = vaultPda();
    const amount = 100_000_000n;
    const beforeBalance = BigInt(await connection.getBalance(vault));

    const ref = await depositor.submitDeposit({
      asset: SOLANA_NATIVE_ASSET,
      amount: "0.1",
      destination: { account: ACCOUNT, ref: REFERENCE },
    });

    const afterBalance = await waitForLamports(vault, beforeBalance + amount);
    expect(afterBalance - beforeBalance).toBe(amount);
    await expect(depositor.verifyDeposit(ref, 0)).resolves.toBe("confirmed");
    await expectDepositedEvent(ref.raw, {
      depositor: depositorKeypair.publicKey,
      account: ACCOUNT,
      reference: REFERENCE,
      mint: PublicKey.default,
      amount,
    });
  }, 120_000);

  it("deposits SPL tokens and verifies the deposit tx", async () => {
    const payer = Keypair.generate();
    await airdrop(payer.publicKey, LAMPORTS_PER_SOL);
    const signer = new KeypairSolanaSigner(payer);
    const depositor = new SolanaVaultDepositor({
      rpcUrl: RPC_URL,
      signer,
      commitment: "confirmed",
    });
    const mint = await createMint(connection, payer, payer.publicKey, 0);
    const depositorAta = await createAssociatedTokenAccountIdempotent(
      connection,
      payer,
      mint,
      payer.publicKey,
    );
    const amount = 25n;
    await mintTo(connection, payer, mint, depositorAta, payer, amount);
    const vaultTokenAccount = await createAssociatedTokenAccountIdempotent(
      connection,
      payer,
      mint,
      vaultPda(),
    );
    const vaultAta = getAssociatedTokenAddress(mint, vaultPda());
    expect(vaultTokenAccount.toBase58()).toBe(vaultAta.toBase58());
    const beforeBalance = await tokenBalance(connection, vaultAta);
    expect(beforeBalance).toBe(0n);

    const ref = await depositor.submitDeposit({
      asset: mint.toBase58(),
      amount: "25",
      destination: { account: ACCOUNT, ref: REFERENCE },
    });

    const afterBalance = await waitForTokenBalance(vaultAta, beforeBalance + amount);
    expect(afterBalance - beforeBalance).toBe(amount);
    await expect(depositor.verifyDeposit(ref, 0)).resolves.toBe("confirmed");
    await expectDepositedEvent(ref.raw, {
      depositor: payer.publicKey,
      account: ACCOUNT,
      reference: REFERENCE,
      mint,
      amount,
    });
  }, 120_000);
});

class KeypairSolanaSigner implements SolanaSigner {
  readonly publicKey: string;

  constructor(private readonly keypair: Keypair) {
    this.publicKey = keypair.publicKey.toBase58();
  }

  async signAndSend(transaction: Transaction): Promise<string> {
    return sendAndConfirmTransaction(connection, transaction, [this.keypair], {
      commitment: "confirmed",
      preflightCommitment: "confirmed",
    });
  }
}

async function airdrop(pubkey: PublicKey, lamports: number): Promise<void> {
  const signature = await connection.requestAirdrop(pubkey, lamports);
  await connection.confirmTransaction(signature, "confirmed");
}

interface ExpectedDepositedEvent {
  depositor: PublicKey;
  account: string;
  reference: string;
  mint: PublicKey;
  amount: bigint;
}

interface DepositedEvent {
  depositor: PublicKey;
  account: Uint8Array;
  reference: Uint8Array;
  mint: PublicKey;
  amount: bigint;
}

async function expectDepositedEvent(
  signature: string,
  expected: ExpectedDepositedEvent,
): Promise<void> {
  const event = await readDepositedEvent(signature);
  expect(event.depositor.toBase58()).toBe(expected.depositor.toBase58());
  expect(stripHexPrefix(bytes32Hex(event.account))).toBe(
    stripHexPrefix(expected.account),
  );
  expect(stripHexPrefix(bytes32Hex(event.reference))).toBe(
    stripHexPrefix(expected.reference),
  );
  expect(event.mint.toBase58()).toBe(expected.mint.toBase58());
  expect(event.amount).toBe(expected.amount);
}

async function readDepositedEvent(signature: string): Promise<DepositedEvent> {
  const deadline = Date.now() + 30_000;
  for (;;) {
    const transaction = await connection.getTransaction(signature, {
      commitment: "confirmed",
      maxSupportedTransactionVersion: 0,
    });
    const innerInstructions = transaction?.meta?.innerInstructions ?? [];
    for (const group of innerInstructions) {
      for (const instruction of group.instructions) {
        const event = decodeDepositedEvent(bs58.decode(instruction.data));
        if (event !== undefined) {
          return event;
        }
      }
    }
    if (Date.now() >= deadline) {
      throw new Error(`Deposited event not found in ${signature}`);
    }
    await sleep(250);
  }
}

function decodeDepositedEvent(data: Uint8Array): DepositedEvent | undefined {
  const eventOffset = findBytes(data, DEPOSITED_EVENT_DISCRIMINATOR);
  if (eventOffset < 0 || data.length < eventOffset + DEPOSITED_EVENT_SIZE) {
    return undefined;
  }
  let cursor = eventOffset + DEPOSITED_EVENT_DISCRIMINATOR.length;
  const depositor = new PublicKey(data.subarray(cursor, cursor + 32));
  cursor += 32;
  const account = data.slice(cursor, cursor + 20);
  cursor += 20;
  const reference = data.slice(cursor, cursor + 32);
  cursor += 32;
  const mint = new PublicKey(data.subarray(cursor, cursor + 32));
  cursor += 32;
  const amountBytes = data.subarray(cursor, cursor + 8);
  const amount = new DataView(
    amountBytes.buffer,
    amountBytes.byteOffset,
    amountBytes.byteLength,
  ).getBigUint64(0, true);
  return { depositor, account, reference, mint, amount };
}

function findBytes(data: Uint8Array, needle: readonly number[]): number {
  for (let offset = 0; offset <= data.length - needle.length; offset += 1) {
    if (needle.every((byte, index) => data[offset + index] === byte)) {
      return offset;
    }
  }
  return -1;
}

function stripHexPrefix(value: string): string {
  return value.startsWith("0x") ? value.slice(2) : value;
}

async function waitForLamports(
  pubkey: PublicKey,
  target: bigint,
): Promise<bigint> {
  const deadline = Date.now() + 30_000;
  for (;;) {
    const balance = BigInt(await connection.getBalance(pubkey));
    if (balance >= target) {
      return balance;
    }
    if (Date.now() >= deadline) {
      throw new Error(`timed out waiting for ${pubkey.toBase58()} balance`);
    }
    await sleep(250);
  }
}

async function waitForTokenBalance(
  address: PublicKey,
  target: bigint,
): Promise<bigint> {
  const deadline = Date.now() + 30_000;
  for (;;) {
    const balance = await tokenBalance(connection, address);
    if (balance >= target) {
      return balance;
    }
    if (Date.now() >= deadline) {
      throw new Error(`timed out waiting for ${address.toBase58()} token balance`);
    }
    await sleep(250);
  }
}

async function sleep(ms: number): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, ms));
}
