import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import type { Address } from "viem";

import {
  composeNonce,
  depositId,
  requireDepositId,
  splitNonce,
} from "../../../src/blockchain/evm/depositId.js";
import { ClearnetSdkError } from "../../../src/index.js";

interface DepositIdVector {
  name: string;
  chainId: number;
  vault: string;
  depositor: string;
  key: string;
  // A decimal string, not a JSON number: the max_nonce vector's sequence is
  // 2^64-1, which JSON.parse would silently round (it parses all numbers as
  // float64) if this were a bare number.
  sequence: string;
  nonce: string;
  id: string;
}

const vectorsPath = fileURLToPath(
  // Shared with the Go tests; the repository keeps a single copy.
  new URL(
    "../../../../../pkg/blockchain/evm/testdata/deposit_id_vectors.json",
    import.meta.url,
  ),
);
const vectors: DepositIdVector[] = JSON.parse(
  readFileSync(vectorsPath, "utf8"),
).vectors;

// The SDK's depositId reproduces the exact bytes custody's Foundry tests
// and the Go SDK compute for the same shared vectors.
describe("depositId golden vectors", () => {
  for (const v of vectors) {
    it(v.name, () => {
      const nonce = BigInt(v.nonce);
      const got = depositId(
        BigInt(v.chainId),
        v.vault as Address,
        v.depositor as Address,
        nonce,
      );
      expect(got).toBe(v.id);

      const composed = composeNonce(BigInt(v.key), BigInt(v.sequence));
      expect(composed).toBe(nonce);
      const split = splitNonce(nonce);
      expect(split.key).toBe(BigInt(v.key));
      expect(split.sequence).toBe(BigInt(v.sequence));
    });
  }
});

// Flipping any single hash input changes the ID.
describe("depositId changes with each input", () => {
  const vault = "0x1111111111111111111111111111111111111111" as Address;
  const depositor = "0x2222222222222222222222222222222222222222" as Address;
  const nonce = composeNonce(7n, 3n);
  const base = depositId(1n, vault, depositor, nonce);

  it("chainId", () => {
    expect(depositId(2n, vault, depositor, nonce)).not.toBe(base);
  });
  it("vault", () => {
    expect(
      depositId(
        1n,
        "0x3333333333333333333333333333333333333333" as Address,
        depositor,
        nonce,
      ),
    ).not.toBe(base);
  });
  it("depositor", () => {
    expect(
      depositId(
        1n,
        vault,
        "0x4444444444444444444444444444444444444444" as Address,
        nonce,
      ),
    ).not.toBe(base);
  });
  it("nonce", () => {
    expect(depositId(1n, vault, depositor, composeNonce(7n, 4n))).not.toBe(
      base,
    );
  });
});

describe("requireDepositId", () => {
  it("accepts a well-formed deposit ID", () => {
    const id = depositId(
      1n,
      "0x1111111111111111111111111111111111111111" as Address,
      "0x2222222222222222222222222222222222222222" as Address,
      0n,
    );
    expect(requireDepositId(id)).toBe(id);
  });

  it.each([
    "",
    "0x1234",
    "a".repeat(64),
    `0x${"A".repeat(64)}`,
    `0x${"g".repeat(64)}`,
    `0x${"de".repeat(32)}/3`,
  ])("rejects %s", (value) => {
    expect(() => requireDepositId(value)).toThrow(ClearnetSdkError);
  });
});

describe("composeNonce range checks", () => {
  it("accepts the extreme valid key and sequence", () => {
    const maxKey = (1n << 192n) - 1n;
    const maxSequence = (1n << 64n) - 1n;
    expect(composeNonce(maxKey, maxSequence)).toBe((1n << 256n) - 1n);
  });

  it.each([
    ["negative key", -1n, 0n],
    ["key of 2^192", 1n << 192n, 0n],
    ["negative sequence", 0n, -1n],
    ["sequence of 2^64", 0n, 1n << 64n],
  ])("rejects a %s", (_name, key, sequence) => {
    expect(() => composeNonce(key, sequence)).toThrow(
      expect.objectContaining({ code: "INVALID_INPUT" }),
    );
  });
});
