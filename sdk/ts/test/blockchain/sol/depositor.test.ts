import { createHash } from "node:crypto";

import bs58 from "bs58";
import {
  PublicKey,
  SystemProgram,
  Transaction,
  type TransactionInstruction,
} from "@solana/web3.js";
import { afterEach, describe, expect, expectTypeOf, it, vi } from "vitest";

import {
  ClearnetSdkError,
  SOLANA_CUSTODY_PROGRAM_ID,
  SOLANA_NATIVE_ASSET,
  SolanaVaultDepositor,
} from "../../../src/index.js";
import type {
  Bytes32Hex,
  DepositStatus,
  SolanaSigner,
  SolanaSubmitDepositInput,
  TxRef,
  VaultDepositor,
} from "../../../src/index.js";

const RPC_URL = "http://127.0.0.1:8899";
const EXPECTED_PROGRAM_ID = "98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg";
const PROGRAM_ID = new PublicKey(EXPECTED_PROGRAM_ID);
const DEPOSITOR = publicKey(11);
const MINT = publicKey(22);
const ACCOUNT = "0x1111111111111111111111111111111111111111";
const ACCOUNT_URI = `yellow://local/user/${ACCOUNT.slice(2)}`;
const REFERENCE =
  "0x2222222222222222222222222222222222222222222222222222222222222222" as Bytes32Hex;
const SIGNATURE = bs58.encode(Uint8Array.from({ length: 64 }, (_, i) => i + 1));

const DEPOSIT_SOL_DISCRIMINATOR = [108, 81, 78, 117, 125, 155, 56, 200];
const DEPOSIT_SPL_DISCRIMINATOR = [224, 0, 198, 175, 198, 47, 105, 204];
const SYSTEM_PROGRAM_ID = SystemProgram.programId.toBase58();
const TOKEN_PROGRAM_ID = "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA";
const ASSOCIATED_TOKEN_PROGRAM_ID =
  "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL";

interface MockSigner extends SolanaSigner {
  signAndSend: ReturnType<
    typeof vi.fn<(transaction: Transaction) => Promise<string>>
  >;
}

describe("SolanaVaultDepositor", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("matches the public depositor and result type contracts", () => {
    expectTypeOf<SolanaVaultDepositor>().toMatchTypeOf<
      VaultDepositor<SolanaSubmitDepositInput>
    >();
    expectTypeOf<TxRef>().toEqualTypeOf<{ hash: Bytes32Hex; raw: string }>();
    expectTypeOf<DepositStatus>().toEqualTypeOf<
      "absent" | "pending" | "confirmed"
    >();
    expect(SOLANA_NATIVE_ASSET).toBe("SOL");
    expect(SOLANA_CUSTODY_PROGRAM_ID).toBe(EXPECTED_PROGRAM_ID);
  });

  it("submits native SOL with the deposit_sol layout and Go-compatible tx ref", async () => {
    stubSignatureStatus({ confirmationStatus: "finalized" });
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const onSubmitted = vi.fn();

    const ref = await depositor.submitDeposit(
      {
        asset: SOLANA_NATIVE_ASSET,
        amount: 10n,
        destination: { account: ACCOUNT_URI, ref: REFERENCE },
      },
      { onSubmitted },
    );

    expect(ref).toEqual(txRefForSignature(SIGNATURE));
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);

    const tx = signedTransaction(signer);
    expect(tx.feePayer?.toBase58()).toBe(DEPOSITOR.toBase58());
    expect(tx.instructions).toHaveLength(1);

    const instruction = tx.instructions[0]!;
    expect(instruction.programId.toBase58()).toBe(EXPECTED_PROGRAM_ID);
    expect(metas(instruction)).toEqual([
      meta(DEPOSITOR, true, true),
      meta(vaultPda(), false, true),
      meta(SYSTEM_PROGRAM_ID, false, false),
      meta(eventAuthorityPda(), false, false),
      meta(PROGRAM_ID, false, false),
    ]);
    expect([...instruction.data]).toEqual([
      ...DEPOSIT_SOL_DISCRIMINATOR,
      ...hexBytes(ACCOUNT),
      ...hexBytes(REFERENCE),
      ...u64(10n),
    ]);
  });

  it("submits SPL tokens with ATA derivation and the deposit_spl layout", async () => {
    stubSignatureStatus({ confirmationStatus: "finalized" });
    const signer = createSigner();
    const depositor = createDepositor(signer);

    const ref = await depositor.submitDeposit({
      asset: MINT.toBase58(),
      amount: 25n,
      destination: { account: ACCOUNT },
    });

    expect(ref).toEqual(txRefForSignature(SIGNATURE));

    const instruction = signedTransaction(signer).instructions[0]!;
    expect(instruction.programId.toBase58()).toBe(EXPECTED_PROGRAM_ID);
    expect(metas(instruction)).toEqual([
      meta(DEPOSITOR, true, true),
      meta(MINT, false, false),
      meta(ata(DEPOSITOR, MINT), false, true),
      meta(vaultPda(), false, false),
      meta(ata(vaultPda(), MINT), false, true),
      meta(TOKEN_PROGRAM_ID, false, false),
      meta(ASSOCIATED_TOKEN_PROGRAM_ID, false, false),
      meta(eventAuthorityPda(), false, false),
      meta(PROGRAM_ID, false, false),
    ]);
    expect([...instruction.data]).toEqual([
      ...DEPOSIT_SPL_DISCRIMINATOR,
      ...hexBytes(ACCOUNT),
      ...new Uint8Array(32),
      ...u64(25n),
    ]);
  });

  it("rejects invalid deposit input before signing", async () => {
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    const signer = createSigner();
    const depositor = createDepositor(signer);

    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: 1n,
        destination: { account: "0x1234" },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: "not-base58",
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: 0n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT, ref: "invoice-1" as Bytes32Hex },
      }),
    ).rejects.toMatchObject({ code: "INVALID_REFERENCE" });

    expect(signer.signAndSend).not.toHaveBeenCalled();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("requires the default program ID for v1", () => {
    const signer = createSigner();

    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer,
        programId: publicKey(33).toBase58(),
      }),
    ).toThrow(ClearnetSdkError);
  });

  it("attaches txRef when a post-broadcast status lookup fails", async () => {
    const rpcError = new Error("node offline");
    stubRpcFailure(rpcError);
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txRefForSignature(SIGNATURE);

    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({
      code: "RPC_ERROR",
      txRef: expectedRef,
      cause: rpcError,
    });
  });

  it("attaches txRef when a submitted transaction reports an execution error", async () => {
    stubSignatureStatus({
      confirmationStatus: "confirmed",
      err: { InstructionError: [0, "Custom"] },
    });
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txRefForSignature(SIGNATURE);

    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({
      code: "TX_REVERTED",
      txRef: expectedRef,
    });
  });

  it("attaches txRef when a submitted transaction times out", async () => {
    stubSignatureStatus(null);
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txRefForSignature(SIGNATURE);

    await expect(
      depositor.submitDeposit(
        {
          asset: SOLANA_NATIVE_ASSET,
          amount: 1n,
          destination: { account: ACCOUNT },
        },
        { receiptTimeoutMs: 1 },
      ),
    ).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txRef: expectedRef,
    });
  });

  it("maps Solana signature statuses to the shared deposit status", async () => {
    const depositor = createDepositor(createSigner());

    stubSignatureStatus({ confirmationStatus: "confirmed" });
    await expect(depositor.verifyDeposit(txRefForSignature(SIGNATURE), 0)).resolves.toBe(
      "confirmed",
    );

    stubSignatureStatus({ confirmationStatus: "confirmed" });
    await expect(depositor.verifyDeposit(txRefForSignature(SIGNATURE), 1)).resolves.toBe(
      "pending",
    );

    stubSignatureStatus({ confirmationStatus: "finalized" });
    await expect(depositor.verifyDeposit(txRefForSignature(SIGNATURE), 1n)).resolves.toBe(
      "confirmed",
    );

    stubSignatureStatus({ confirmationStatus: "processed" });
    await expect(depositor.verifyDeposit(txRefForSignature(SIGNATURE), 0)).resolves.toBe(
      "pending",
    );

    stubSignatureStatus(null);
    await expect(depositor.verifyDeposit(txRefForSignature(SIGNATURE), 0)).resolves.toBe(
      "absent",
    );

    stubSignatureStatus({ confirmationStatus: "finalized", err: { InstructionError: [0, "Custom"] } });
    await expect(depositor.verifyDeposit(txRefForSignature(SIGNATURE), 0)).resolves.toBe(
      "absent",
    );
  });

  it("validates tx refs and confirmation depths before RPC", async () => {
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    const depositor = createDepositor(createSigner());

    await expect(
      depositor.verifyDeposit({ hash: txRefForSignature(SIGNATURE).hash, raw: "bad sig" }, 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
    await expect(
      depositor.verifyDeposit({ hash: "0x1234" as Bytes32Hex, raw: SIGNATURE }, 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
    await expect(
      depositor.verifyDeposit(txRefForSignature(SIGNATURE), -1),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });
    await expect(
      depositor.verifyDeposit(txRefForSignature(SIGNATURE), 1.5),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });

    expect(fetch).not.toHaveBeenCalled();
  });
});

function createDepositor(signer: SolanaSigner): SolanaVaultDepositor {
  return new SolanaVaultDepositor({ rpcUrl: RPC_URL, signer });
}

function createSigner(): MockSigner {
  return {
    publicKey: DEPOSITOR.toBase58(),
    signAndSend: vi.fn<(transaction: Transaction) => Promise<string>>()
      .mockResolvedValue(SIGNATURE),
  };
}

function signedTransaction(signer: MockSigner): Transaction {
  const call = signer.signAndSend.mock.calls[0];
  if (call === undefined) {
    throw new Error("signAndSend was not called");
  }
  return call[0];
}

function publicKey(seed: number): PublicKey {
  return new PublicKey(Uint8Array.from({ length: 32 }, (_, i) => seed + i));
}

function vaultPda(): string {
  return pda("vault");
}

function eventAuthorityPda(): string {
  return pda("__event_authority");
}

function pda(seed: string): string {
  return PublicKey.findProgramAddressSync(
    [new TextEncoder().encode(seed)],
    PROGRAM_ID,
  )[0].toBase58();
}

function ata(owner: PublicKey | string, mint: PublicKey): string {
  const ownerKey = typeof owner === "string" ? new PublicKey(owner) : owner;
  return PublicKey.findProgramAddressSync(
    [
      ownerKey.toBytes(),
      new PublicKey(TOKEN_PROGRAM_ID).toBytes(),
      mint.toBytes(),
    ],
    new PublicKey(ASSOCIATED_TOKEN_PROGRAM_ID),
  )[0].toBase58();
}

function meta(
  pubkey: PublicKey | string,
  isSigner: boolean,
  isWritable: boolean,
): { pubkey: string; isSigner: boolean; isWritable: boolean } {
  return {
    pubkey: typeof pubkey === "string" ? pubkey : pubkey.toBase58(),
    isSigner,
    isWritable,
  };
}

function metas(instruction: TransactionInstruction): ReturnType<typeof meta>[] {
  return instruction.keys.map((key) =>
    meta(key.pubkey, key.isSigner, key.isWritable),
  );
}

function hexBytes(value: string): Uint8Array {
  const hex = value.toLowerCase().replace(/^0x/, "");
  return Uint8Array.from(Buffer.from(hex, "hex"));
}

function u64(value: bigint): Uint8Array {
  const bytes = new Uint8Array(8);
  new DataView(bytes.buffer).setBigUint64(0, value, true);
  return bytes;
}

function txRefForSignature(signature: string): TxRef {
  const signatureBytes = bs58.decode(signature);
  const hash = createHash("sha256").update(signatureBytes).digest("hex");
  return { hash: `0x${hash}`, raw: signature };
}

function stubSignatureStatus(
  value: null | { confirmationStatus: string; err?: unknown },
): void {
  const response = {
    jsonrpc: "2.0",
    id: "1",
    result: {
      context: { slot: 1 },
      value: [
        value === null
          ? null
          : {
              slot: 1,
              confirmations:
                value.confirmationStatus === "finalized" ? null : 1,
              err: value.err ?? null,
              confirmationStatus: value.confirmationStatus,
            },
      ],
    },
  };
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify(response), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
      ),
    ),
  );
}

function stubRpcFailure(error: Error): void {
  vi.stubGlobal("fetch", vi.fn().mockRejectedValue(error));
}
