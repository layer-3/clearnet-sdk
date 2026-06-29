import { Buffer } from "buffer";

import bs58 from "bs58";
import { sha256 } from "@noble/hashes/sha2.js";
import { PublicKey } from "@solana/web3.js";

import { ClearnetSdkError } from "../../core/errors.js";
import type { Bytes32Hex, DepositDestination, TxRef } from "../../core/types.js";
import {
  BYTES32_HEX_PATTERN,
  normalizeMinConfirmations,
} from "../../core/validation.js";
import {
  DEFAULT_SOLANA_COMMITMENT,
  SOLANA_CUSTODY_PROGRAM_ID,
} from "./constants.js";
import type { SolanaCommitment, SolanaSigner } from "./types.js";

const UINT64_MAX = (1n << 64n) - 1n;

export function requireRpcUrl(rpcUrl: unknown): string {
  if (typeof rpcUrl !== "string" || rpcUrl.trim() === "") {
    throw new ClearnetSdkError("RPC_ERROR", "rpcUrl is required");
  }
  return rpcUrl;
}

export function requireSigner(signer: unknown): SolanaSigner {
  if (!signer || typeof signer !== "object") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "Solana signer is required",
    );
  }
  const candidate = signer as Partial<SolanaSigner>;
  publicKeyFromString(candidate.publicKey, "signer.publicKey");
  if (typeof candidate.signAndSend !== "function") {
    throw new ClearnetSdkError(
      "MISSING_WALLET_ACCOUNT",
      "Solana signer.signAndSend is required",
    );
  }
  return candidate as SolanaSigner;
}

export function requireProgramId(value: unknown): PublicKey {
  const programId =
    value === undefined
      ? new PublicKey(SOLANA_CUSTODY_PROGRAM_ID)
      : publicKeyFromString(value, "programId");
  if (programId.toBase58() !== SOLANA_CUSTODY_PROGRAM_ID) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `programId must be ${SOLANA_CUSTODY_PROGRAM_ID}`,
    );
  }
  return programId;
}

export function normalizeCommitment(
  commitment: SolanaCommitment | undefined,
): SolanaCommitment {
  if (commitment === undefined) {
    return DEFAULT_SOLANA_COMMITMENT;
  }
  if (
    commitment !== "processed" &&
    commitment !== "confirmed" &&
    commitment !== "finalized"
  ) {
    throw new ClearnetSdkError(
      "RPC_ERROR",
      "commitment must be processed, confirmed, or finalized",
    );
  }
  return commitment;
}

export function requireReceiptTimeout(value: number): number {
  if (!Number.isSafeInteger(value) || value <= 0) {
    throw new ClearnetSdkError(
      "RECEIPT_TIMEOUT",
      "receiptTimeoutMs must be a positive safe integer",
    );
  }
  return value;
}

export function requireAmount(amount: unknown): bigint {
  if (typeof amount !== "bigint") {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must be a bigint in base units",
    );
  }
  if (amount <= 0n) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      "amount must be greater than zero",
    );
  }
  if (amount > UINT64_MAX) {
    throw new ClearnetSdkError("INVALID_AMOUNT", "amount must fit in uint64");
  }
  return amount;
}

export function requireDepositDestination(
  destination: unknown,
): DepositDestination {
  if (!destination || typeof destination !== "object") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  return destination as DepositDestination;
}

export function requireClearnetAccount(account: unknown): Uint8Array {
  if (typeof account !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  const segment = account.slice(account.lastIndexOf("/") + 1);
  const hex = segment.toLowerCase().replace(/^0x/, "");
  if (!/^[a-f0-9]+$/.test(hex) || hex.length !== 40) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  return Uint8Array.from(Buffer.from(hex, "hex"));
}

export function requireReference(reference: unknown): Uint8Array {
  if (reference === undefined || reference === "") {
    return new Uint8Array(32);
  }
  if (typeof reference !== "string" || !BYTES32_HEX_PATTERN.test(reference)) {
    throw new ClearnetSdkError(
      "INVALID_REFERENCE",
      "destination.ref must be a 32-byte hex value",
    );
  }
  return Uint8Array.from(Buffer.from(reference.slice(2), "hex"));
}

export function resolveMint(asset: unknown): PublicKey | undefined {
  if (
    asset === "" ||
    asset === "native" ||
    asset === "SOL" ||
    asset === "sol"
  ) {
    return undefined;
  }
  return publicKeyFromString(asset, "asset");
}

export function publicKeyFromString(value: unknown, field: string): PublicKey {
  if (typeof value !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `${field} must be a valid Solana public key`,
    );
  }
  try {
    return new PublicKey(value);
  } catch (error) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `${field} must be a valid Solana public key`,
      { cause: error },
    );
  }
}

export function requireTxRef(ref: unknown): Uint8Array {
  if (!ref || typeof ref !== "object") {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.raw must be a Solana signature",
    );
  }
  const fields = ref as Partial<TxRef>;
  if (typeof fields.hash !== "string" || !BYTES32_HEX_PATTERN.test(fields.hash)) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must be a 32-byte hex value",
    );
  }
  if (typeof fields.raw !== "string") {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.raw must be a Solana signature",
    );
  }
  let signature: Uint8Array;
  try {
    signature = bs58.decode(fields.raw);
  } catch (error) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.raw must be a Solana signature",
      { cause: error },
    );
  }
  if (signature.length !== 64) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.raw must decode to a 64-byte Solana signature",
    );
  }
  if (fields.hash.toLowerCase() !== bytes32Hex(sha256(signature))) {
    throw new ClearnetSdkError(
      "INVALID_TX_REF",
      "ref.hash must match the Solana signature hash",
    );
  }
  return signature;
}

export { normalizeMinConfirmations };

export function bytes32Hex(bytes: Uint8Array): Bytes32Hex {
  const hex = [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  return `0x${hex}`;
}
