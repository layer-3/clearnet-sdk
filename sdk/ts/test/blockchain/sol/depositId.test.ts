import { sha256 } from "@noble/hashes/sha2.js";
import { describe, expect, it } from "vitest";

import { solanaDepositId } from "../../../src/index.js";
import {
  DEPOSITED_EVENT_DISCRIMINATOR,
  EXECUTED_EVENT_DISCRIMINATOR,
} from "../../../src/blockchain/sol/constants.js";
import { parseSolanaDepositId } from "../../../src/blockchain/sol/depositId.js";
import { loadDepositIdVectors } from "../deposit-id-vectors.js";

interface DepositIdVector {
  name: string;
  signature: string;
  // A decimal string: uint64 values do not survive JSON.parse as numbers.
  index: string;
  id: string;
}

const vectors = loadDepositIdVectors<DepositIdVector>("sol");

const SIGNATURE =
  "1GMkH3brNXiNNs1tiFZHu4yZSRrzJwxi5wB9bHFtMinfCXNnR1adh8Vo8NTheK4evneedH4qmvjeqcBBNAefgS";
const HASH = "0xfdeab9acf3710362bd2658cdc9a29e8f9c757fcf9811603a8c447cd1d9151108";

describe("solanaDepositId golden vectors", () => {
  it("loads the shared vectors", () => {
    expect(vectors.length).toBeGreaterThan(0);
  });

  for (const v of vectors) {
    it(v.name, () => {
      expect(solanaDepositId(v.signature, BigInt(v.index))).toBe(v.id);
      expect(parseSolanaDepositId(v.signature, v.id)).toBe(BigInt(v.index));
    });
  }

  it("accepts a number index", () => {
    expect(solanaDepositId(SIGNATURE, 1)).toBe(`${HASH}:1`);
  });
});

describe("solanaDepositId input validation", () => {
  it.each(["", "not base58 0OIl", "1111"])("rejects signature %s", (signature) => {
    expect(() => solanaDepositId(signature, 0)).toThrow(
      expect.objectContaining({ code: "INVALID_TX_ID" }),
    );
  });

  it.each([-1n, 1n << 64n, -1, 1.5, Number.MAX_SAFE_INTEGER + 1])(
    "rejects index %s",
    (index) => {
      expect(() => solanaDepositId(SIGNATURE, index)).toThrow(
        expect.objectContaining({ code: "INVALID_INPUT" }),
      );
    },
  );
});

describe("parseSolanaDepositId", () => {
  it.each([
    ["the signature", SIGNATURE],
    ["the bare hash", HASH],
    ["a missing index", `${HASH}:`],
    ["a leading zero", `${HASH}:01`],
    ["a plus sign", `${HASH}:+1`],
    ["a minus sign", `${HASH}:-1`],
    ["an overflowing index", `${HASH}:18446744073709551616`],
    ["an upper-case hash", `0x${HASH.slice(2).toUpperCase()}:0`],
    ["a missing 0x prefix", `${HASH.slice(2)}:0`],
    ["another signature's hash", `0x${"00".repeat(32)}:0`],
    ["a trailing separator", `${HASH}:0:`],
  ])("rejects %s", (_name, depositId) => {
    expect(() => parseSolanaDepositId(SIGNATURE, depositId)).toThrow(
      expect.objectContaining({ code: "INVALID_DEPOSIT_ID" }),
    );
  });

  it("accepts the largest event index", () => {
    expect(parseSolanaDepositId(SIGNATURE, `${HASH}:18446744073709551615`)).toBe(
      (1n << 64n) - 1n,
    );
  });

  const notShape = "Solana deposit ID must be 0x<signature hash>:<index>";
  const mismatch = "Solana deposit ID does not match the signature";
  it.each<[unknown, string]>([
    [SIGNATURE, notShape],
    [HASH, notShape],
    ["", notShape],
    [7, notShape],
    [`${HASH}:`, mismatch],
    [`${HASH}:01`, mismatch],
    [`${HASH}:18446744073709551616`, mismatch],
    [`${HASH}:0:`, mismatch],
    [`${HASH}::0`, mismatch],
    [":0", mismatch],
    [`x${HASH}:0`, mismatch],
  ])("reports %s with its message", (depositId, message) => {
    expect(() => parseSolanaDepositId(SIGNATURE, depositId)).toThrow(
      expect.objectContaining({ code: "INVALID_DEPOSIT_ID", message }),
    );
  });
});

describe("custody event discriminators", () => {
  it.each([
    ["Deposited", DEPOSITED_EVENT_DISCRIMINATOR],
    ["Executed", EXECUTED_EVENT_DISCRIMINATOR],
  ])("%s is the first 8 bytes of sha256(event:<name>)", (name, discriminator) => {
    const hash = sha256(new TextEncoder().encode(`event:${name}`));
    expect([...discriminator]).toEqual([...hash.slice(0, 8)]);
  });
});
