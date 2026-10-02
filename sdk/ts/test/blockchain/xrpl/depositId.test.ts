import { describe, expect, it } from "vitest";

import { xrplDepositId } from "../../../src/index.js";
import { loadDepositIdVectors } from "../deposit-id-vectors.js";

interface DepositIdVector {
  name: string;
  hash: string;
  id: string;
}

const vectors = loadDepositIdVectors<DepositIdVector>("xrpl");

describe("xrplDepositId golden vectors", () => {
  it("loads the shared vectors", () => {
    expect(vectors.length).toBeGreaterThan(0);
  });

  for (const v of vectors) {
    it(v.name, () => {
      expect(xrplDepositId(v.hash)).toBe(v.id);
    });
  }
});

describe("xrplDepositId input validation", () => {
  it.each(["", "0x" + "a".repeat(64), "a".repeat(63), "g".repeat(64)])(
    "rejects %s",
    (hash) => {
      expect(() => xrplDepositId(hash)).toThrow(
        expect.objectContaining({ code: "INVALID_TX_ID" }),
      );
    },
  );
});
