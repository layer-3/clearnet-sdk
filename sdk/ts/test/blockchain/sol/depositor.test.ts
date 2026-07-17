import bs58 from "bs58";
import { sha256 } from "@noble/hashes/sha2.js";
import {
  Connection,
  PublicKey,
  SystemProgram,
  Transaction,
  type TransactionInstruction,
} from "@solana/web3.js";
import { afterEach, describe, expect, expectTypeOf, it, vi } from "vitest";

import {
  ClearnetSdkError,
  eventAuthorityPda as sdkEventAuthorityPda,
  SOLANA_CUSTODY_PROGRAM_ID,
  SOLANA_NATIVE_ASSET,
  SolanaVaultDepositor,
  vaultPda as sdkVaultPda,
} from "../../../src/index.js";
import {
  DEPOSIT_SOL_DISCRIMINATOR,
  DEPOSIT_SPL_DISCRIMINATOR,
  SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID,
  SOLANA_TOKEN_PROGRAM_ID,
} from "../../../src/blockchain/sol/constants.js";
import { bytes32Hex } from "../../../src/blockchain/sol/validation.js";
import type {
  Bytes32Hex,
  DepositStatus,
  SolanaDepositorConfig,
  SolanaSigner,
  SolanaSubmitDepositInput,
  SubmitDepositOptions,
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

const SYSTEM_PROGRAM_ID = SystemProgram.programId.toBase58();
const TOKEN_PROGRAM_ID = SOLANA_TOKEN_PROGRAM_ID;
const ASSOCIATED_TOKEN_PROGRAM_ID = SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID;
const VAULT_PDA = sdkVaultPda(PROGRAM_ID).toBase58();
const EVENT_AUTHORITY_PDA = sdkEventAuthorityPda(PROGRAM_ID).toBase58();

interface MockSigner extends SolanaSigner {
  signAndSend: ReturnType<
    typeof vi.fn<(transaction: Transaction) => Promise<string>>
  >;
}

describe("SolanaVaultDepositor", () => {
  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("matches the public depositor and result type contracts", () => {
    expectTypeOf<SolanaVaultDepositor>().toMatchTypeOf<
      VaultDepositor<SolanaSubmitDepositInput>
    >();
    expectTypeOf<string>().toEqualTypeOf<string>();
    expectTypeOf<DepositStatus>().toEqualTypeOf<
      "absent" | "pending" | "confirmed"
    >();
    expect(SOLANA_NATIVE_ASSET).toBe("");
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
        amount: "0.00000001",
        destination: { account: ACCOUNT_URI, ref: REFERENCE },
      },
      { onSubmitted },
    );

    expect(ref).toEqual(txIDForSignature(SIGNATURE));
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);

    const tx = signedTransaction(signer);
    expect(tx.feePayer?.toBase58()).toBe(DEPOSITOR.toBase58());
    expect(tx.instructions).toHaveLength(1);

    const instruction = tx.instructions[0]!;
    expect(instruction.programId.toBase58()).toBe(EXPECTED_PROGRAM_ID);
    expect(metas(instruction)).toEqual([
      meta(DEPOSITOR, true, true),
      meta(VAULT_PDA, false, true),
      meta(SYSTEM_PROGRAM_ID, false, false),
      meta(EVENT_AUTHORITY_PDA, false, false),
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
    const getAccountInfo = vi
      .spyOn(Connection.prototype, "getAccountInfo")
      .mockResolvedValue({ data: splMintData(0) } as never);
    const signer = createSigner();
    const depositor = createDepositor(signer);

    const ref = await depositor.submitDeposit({
      asset: MINT.toBase58(),
      amount: "25",
      destination: { account: ACCOUNT },
    });

    expect(ref).toEqual(txIDForSignature(SIGNATURE));

    const instruction = signedTransaction(signer).instructions[0]!;
    expect(instruction.programId.toBase58()).toBe(EXPECTED_PROGRAM_ID);
    expect(metas(instruction)).toEqual([
      meta(DEPOSITOR, true, true),
      meta(MINT, false, false),
      meta(ata(DEPOSITOR, MINT), false, true),
      meta(VAULT_PDA, false, false),
      meta(ata(VAULT_PDA, MINT), false, true),
      meta(TOKEN_PROGRAM_ID, false, false),
      meta(ASSOCIATED_TOKEN_PROGRAM_ID, false, false),
      meta(EVENT_AUTHORITY_PDA, false, false),
      meta(PROGRAM_ID, false, false),
    ]);
    expect([...instruction.data]).toEqual([
      ...DEPOSIT_SPL_DISCRIMINATOR,
      ...hexBytes(ACCOUNT),
      ...new Uint8Array(32),
      ...u64(25n),
    ]);
    expect(getAccountInfo).toHaveBeenCalledWith(MINT, "finalized");
  });

  it("rejects invalid deposit input before signing", async () => {
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    const signer = createSigner();
    const depositor = createDepositor(signer);

    await expect(
      depositor.submitDeposit(null as unknown as SolanaSubmitDepositInput),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit(
        {
          asset: SOLANA_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT },
        },
        null as unknown as SubmitDepositOptions,
      ),
    ).rejects.toMatchObject({ code: "RECEIPT_TIMEOUT" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: null as unknown as SolanaSubmitDepositInput["destination"],
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: "0x1234" },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: "not-base58",
        amount: "1",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "0",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: 1 as unknown as string,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "18446744073709551616",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT, ref: "invoice-1" as Bytes32Hex },
      }),
    ).rejects.toMatchObject({ code: "INVALID_REFERENCE" });

    expect(signer.signAndSend).not.toHaveBeenCalled();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("uses a custom program ID for SOL instruction and PDA derivation", async () => {
    stubSignatureStatus({ confirmationStatus: "finalized" });
    const signer = createSigner();
    const customProgramId = publicKey(33);
    const customVault = sdkVaultPda(customProgramId);
    const customEventAuthority = sdkEventAuthorityPda(customProgramId);
    const depositor = createDepositor(signer, {
      programId: customProgramId.toBase58(),
    });

    await depositor.submitDeposit({
      asset: SOLANA_NATIVE_ASSET,
      amount: "0.00000001",
      destination: { account: ACCOUNT },
    });

    const instruction = signedTransaction(signer).instructions[0]!;
    expect(instruction.programId.toBase58()).toBe(customProgramId.toBase58());
    expect(metas(instruction)).toEqual([
      meta(DEPOSITOR, true, true),
      meta(customVault, false, true),
      meta(SYSTEM_PROGRAM_ID, false, false),
      meta(customEventAuthority, false, false),
      meta(customProgramId, false, false),
    ]);
  });

  it("rejects invalid Solana configuration", () => {
    const signer = createSigner();

    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer,
        programId: "not-base58",
      }),
    ).toThrow(ClearnetSdkError);
    expect(() =>
      new SolanaVaultDepositor({ rpcUrl: "", signer }),
    ).toThrow(ClearnetSdkError);
    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer: undefined as unknown as SolanaSigner,
      }),
    ).toThrow(ClearnetSdkError);
    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer: {
          publicKey: "not-base58",
          signAndSend: async () => SIGNATURE,
        },
      }),
    ).toThrow(ClearnetSdkError);
    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer: { publicKey: DEPOSITOR.toBase58() } as unknown as SolanaSigner,
      }),
    ).toThrow(ClearnetSdkError);
    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer,
        commitment: "recent" as never,
      }),
    ).toThrow(ClearnetSdkError);
    expect(() =>
      new SolanaVaultDepositor({
        rpcUrl: RPC_URL,
        signer,
        receiptTimeoutMs: 0,
      }),
    ).toThrow(ClearnetSdkError);
  });

  it("wraps signer failures before a tx ref exists", async () => {
    const cause = new Error("wallet rejected");
    stubSignatureStatus({ confirmationStatus: "finalized" });
    const signer = createSigner();
    signer.signAndSend.mockRejectedValue(cause);
    const depositor = createDepositor(signer);
    const onSubmitted = vi.fn();

    await expect(
      depositor.submitDeposit(
        {
          asset: SOLANA_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT },
        },
        { onSubmitted },
      ),
    ).rejects.toMatchObject({
      code: "RPC_ERROR",
      cause,
      txID: undefined,
    });
    expect(onSubmitted).not.toHaveBeenCalled();
  });

  it("rejects invalid signer-returned signatures before submission callback", async () => {
    stubSignatureStatus({ confirmationStatus: "finalized" });
    const signer = createSigner();
    signer.signAndSend.mockResolvedValue("not a base58 signature");
    const depositor = createDepositor(signer);
    const onSubmitted = vi.fn();

    await expect(
      depositor.submitDeposit(
        {
          asset: SOLANA_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT },
        },
        { onSubmitted },
      ),
    ).rejects.toMatchObject({ code: "INVALID_TX_ID" });
    expect(onSubmitted).not.toHaveBeenCalled();
  });

  it("attaches txID when a post-broadcast status lookup fails", async () => {
    const rpcError = new Error("node offline");
    stubRpcFailure(rpcError);
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txIDForSignature(SIGNATURE);

    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({
      code: "RPC_ERROR",
      txID: expectedRef,
      cause: rpcError,
    });
  });

  it("attaches txID when a submitted transaction reports an execution error", async () => {
    stubSignatureStatus({
      confirmationStatus: "confirmed",
      err: { InstructionError: [0, "Custom"] },
    });
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txIDForSignature(SIGNATURE);

    await expect(
      depositor.submitDeposit({
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({
      code: "TX_REVERTED",
      txID: expectedRef,
    });
  });

  it("attaches txID when a submitted transaction times out", async () => {
    vi.useFakeTimers();
    stubSignatureStatus(null);
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txIDForSignature(SIGNATURE);

    const promise = depositor.submitDeposit(
      {
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT },
      },
      { receiptTimeoutMs: 1_000 },
    );
    const assertion = expect(promise).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txID: expectedRef,
    });
    await vi.advanceTimersByTimeAsync(1_000);
    await assertion;
  });

  it("attaches txID when a submitted transaction is aborted while waiting", async () => {
    vi.useFakeTimers();
    stubSignatureStatus(null);
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const controller = new AbortController();
    const expectedRef = txIDForSignature(SIGNATURE);

    const promise = depositor.submitDeposit(
      {
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT },
      },
      {
        signal: controller.signal,
        receiptTimeoutMs: 10_000,
        onSubmitted() {
          setTimeout(() => controller.abort(), 1);
        },
      },
    );
    const assertion = expect(promise).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txID: expectedRef,
    });
    await vi.advanceTimersByTimeAsync(1);
    await assertion;
  });

  it("rejects invalid wait options before signing", async () => {
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const controller = new AbortController();
    controller.abort();

    await expect(
      depositor.submitDeposit(
        {
          asset: SOLANA_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT },
        },
        { receiptTimeoutMs: 0 },
      ),
    ).rejects.toMatchObject({ code: "RECEIPT_TIMEOUT" });
    await expect(
      depositor.submitDeposit(
        {
          asset: SOLANA_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT },
        },
        { signal: controller.signal },
      ),
    ).rejects.toMatchObject({ code: "RECEIPT_TIMEOUT" });
    expect(signer.signAndSend).not.toHaveBeenCalled();
  });

  it("bounds a hung post-submit status lookup", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));
    const signer = createSigner();
    const depositor = createDepositor(signer);
    const expectedRef = txIDForSignature(SIGNATURE);

    const promise = depositor.submitDeposit(
      {
        asset: SOLANA_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT },
      },
      { receiptTimeoutMs: 1_000 },
    );
    const assertion = expect(promise).rejects.toMatchObject({
      code: "RECEIPT_TIMEOUT",
      txID: expectedRef,
    });
    await vi.advanceTimersByTimeAsync(1_000);
    await assertion;
  });

  it("maps Solana signature statuses to the shared deposit status", async () => {
    const depositor = createDepositor(createSigner());

    stubSignatureStatus({ confirmationStatus: "confirmed" });
    await expect(depositor.verifyDeposit(txIDForSignature(SIGNATURE), 0)).resolves.toBe(
      "confirmed",
    );

    stubSignatureStatus({ confirmationStatus: "confirmed" });
    await expect(depositor.verifyDeposit(txIDForSignature(SIGNATURE), 1)).resolves.toBe(
      "pending",
    );

    stubSignatureStatus({ confirmationStatus: "finalized" });
    await expect(depositor.verifyDeposit(txIDForSignature(SIGNATURE), 1n)).resolves.toBe(
      "confirmed",
    );

    stubSignatureStatus({ confirmationStatus: "finalized" });
    await expect(
      depositor.verifyDeposit(txIDForSignature(SIGNATURE), 1n << 80n),
    ).resolves.toBe("confirmed");

    stubSignatureStatus({ confirmationStatus: "processed" });
    await expect(depositor.verifyDeposit(txIDForSignature(SIGNATURE), 0)).resolves.toBe(
      "pending",
    );

    stubSignatureStatus(null);
    await expect(depositor.verifyDeposit(txIDForSignature(SIGNATURE), 0)).resolves.toBe(
      "absent",
    );

    stubSignatureStatus({ confirmationStatus: "finalized", err: { InstructionError: [0, "Custom"] } });
    await expect(depositor.verifyDeposit(txIDForSignature(SIGNATURE), 0)).resolves.toBe(
      "absent",
    );
  });

  it("validates tx refs and confirmation depths before RPC", async () => {
    const fetch = vi.fn();
    vi.stubGlobal("fetch", fetch);
    const depositor = createDepositor(createSigner());

    await expect(
      depositor.verifyDeposit("bad sig", 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_ID" });
    await expect(
      depositor.verifyDeposit("0x1234", 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_ID" });
    await expect(
      depositor.verifyDeposit(bs58.encode(new Uint8Array(63).fill(9)), 0),
    ).rejects.toMatchObject({ code: "INVALID_TX_ID" });
    await expect(
      depositor.verifyDeposit(txIDForSignature(SIGNATURE), -1),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });
    await expect(
      depositor.verifyDeposit(txIDForSignature(SIGNATURE), 1.5),
    ).rejects.toMatchObject({ code: "INVALID_CONFIRMATIONS" });

    expect(fetch).not.toHaveBeenCalled();
  });

  it("preserves txID when verifyDeposit status lookup fails", async () => {
    const rpcError = new Error("node offline");
    stubRpcFailure(rpcError);
    const depositor = createDepositor(createSigner());
    const ref = txIDForSignature(SIGNATURE);

    await expect(depositor.verifyDeposit(ref, 0)).rejects.toMatchObject({
      code: "RPC_ERROR",
      txID: ref,
      cause: rpcError,
    });
  });
});

function createDepositor(
  signer: SolanaSigner,
  overrides: Partial<Omit<SolanaDepositorConfig, "rpcUrl" | "signer">> = {},
): SolanaVaultDepositor {
  return new SolanaVaultDepositor({ rpcUrl: RPC_URL, signer, ...overrides });
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

function splMintData(decimals: number): Uint8Array {
  const data = new Uint8Array(82);
  data[44] = decimals;
  return data;
}

function txIDForSignature(signature: string): string {
  return signature;
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
