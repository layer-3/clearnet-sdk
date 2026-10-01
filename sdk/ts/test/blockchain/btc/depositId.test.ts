import { describe, expect, it } from "vitest";

import { bitcoinDepositId } from "../../../src/index.js";
import { parseBitcoinDepositId } from "../../../src/blockchain/btc/depositId.js";
import { loadDepositIdVectors } from "../deposit-id-vectors.js";

interface DepositIdVector {
  name: string;
  txid: string;
  vout: number;
  id: string;
}

const vectors = loadDepositIdVectors<DepositIdVector>("btc");

const TXID = "4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b";

describe("bitcoinDepositId golden vectors", () => {
  it("loads the shared vectors", () => {
    expect(vectors.length).toBeGreaterThan(0);
  });

  for (const v of vectors) {
    it(v.name, () => {
      expect(bitcoinDepositId(v.txid, v.vout)).toBe(v.id);
      expect(parseBitcoinDepositId(v.txid, v.id)).toBe(v.vout);
    });
  }
});

describe("bitcoinDepositId input validation", () => {
  it("keeps the txid's case", () => {
    expect(bitcoinDepositId(TXID.toUpperCase(), 3)).toBe(`${TXID.toUpperCase()}:3`);
  });

  it.each(["", "0x" + TXID, TXID.slice(1), "g".repeat(64)])(
    "rejects txid %s",
    (txid) => {
      expect(() => bitcoinDepositId(txid, 0)).toThrow(
        expect.objectContaining({ code: "INVALID_TX_ID" }),
      );
    },
  );

  it.each([-1, 2 ** 32, 1.5, Number.NaN, Number.MAX_SAFE_INTEGER + 1])(
    "rejects vout %s",
    (vout) => {
      expect(() => bitcoinDepositId(TXID, vout)).toThrow(
        expect.objectContaining({ code: "INVALID_INPUT" }),
      );
    },
  );
});

describe("parseBitcoinDepositId", () => {
  it.each([
    ["the bare txid", TXID],
    ["a missing index", `${TXID}:`],
    ["a leading zero", `${TXID}:01`],
    ["a plus sign", `${TXID}:+1`],
    ["a minus sign", `${TXID}:-1`],
    ["whitespace", `${TXID}: 1`],
    ["an overflowing index", `${TXID}:4294967296`],
    ["a huge index", `${TXID}:${"9".repeat(30)}`],
    ["an upper-case txid", `${TXID.toUpperCase()}:0`],
    ["another txid", `${"00".repeat(32)}:0`],
    ["a trailing separator", `${TXID}:0:`],
    ["a decimal index", `${TXID}:1.0`],
  ])("rejects %s", (_name, depositId) => {
    expect(() => parseBitcoinDepositId(TXID, depositId)).toThrow(
      expect.objectContaining({ code: "INVALID_DEPOSIT_ID" }),
    );
  });

  it("rejects a non-string deposit ID", () => {
    expect(() => parseBitcoinDepositId(TXID, 0)).toThrow(
      expect.objectContaining({ code: "INVALID_DEPOSIT_ID" }),
    );
  });

  it("accepts the largest output index", () => {
    expect(parseBitcoinDepositId(TXID, `${TXID}:4294967295`)).toBe(4294967295);
  });

  const notShape = "Bitcoin deposit ID must be <txid>:<vout> for the given txid";
  const badIndex = "Bitcoin deposit ID has an invalid output index";
  it.each<[unknown, string]>([
    [TXID, notShape],
    [`${TXID.toUpperCase()}:0`, notShape],
    [`x${TXID}:0`, notShape],
    [0, notShape],
    [`${TXID}:`, badIndex],
    [`${TXID}:01`, badIndex],
    [`${TXID}:4294967296`, badIndex],
    [`${TXID}:${"9".repeat(30)}`, badIndex],
    [`${TXID}:0:`, badIndex],
    [`${TXID}::0`, badIndex],
  ])("reports %s with its message", (depositId, message) => {
    expect(() => parseBitcoinDepositId(TXID, depositId)).toThrow(
      expect.objectContaining({ code: "INVALID_DEPOSIT_ID", message }),
    );
  });
});
