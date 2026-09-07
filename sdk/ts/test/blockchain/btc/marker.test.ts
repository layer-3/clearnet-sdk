import { readFileSync } from "node:fs";

import { p2wsh, Transaction } from "@scure/btc-signer";
import { describe, expect, it } from "vitest";

import { BITCOIN_NATIVE_ASSET, BitcoinVaultDepositor } from "../../../src/index.js";
import type { BitcoinRpc, Bytes32Hex } from "../../../src/index.js";
import { depositPayment, taggedRedeemScript } from "../../../src/blockchain/btc/address.js";
import {
  encodeMarkerPayload,
  encodeMarkerScript,
  GENERIC_DEPOSIT_TAG_HEX,
  GENERIC_DEPOSIT_TAG_PREIMAGE,
  MARKER_VERSION_1,
  MARKER_VERSION_2,
} from "../../../src/blockchain/btc/marker.js";
import { networkParams } from "../../../src/blockchain/btc/networks.js";
import { bytesToHex, hexToBytes } from "../../../src/core/bytes.js";

// This is the writer<->reader contract with the Go marker package: a TS marker
// Go's parser rejects is an uncredited deposit. Load the SAME golden vectors
// Go tests assert against - do not duplicate the literals here, they would drift
// from the source of truth.
const VECTORS_PATH = new URL(
  "../../../../../pkg/blockchain/btc/marker/testdata/vectors.json",
  import.meta.url,
);

interface EncodeVector {
  case: string;
  marker: { version: number; addressHex: string; referenceHex: string | null };
  payloadHex?: string;
  scriptHex?: string;
  expectError?: string;
}

interface VectorsFile {
  genericTag: { preimage: string; sha256Hex: string };
  encode: readonly EncodeVector[];
}

const vectors: VectorsFile = JSON.parse(readFileSync(VECTORS_PATH, "utf8"));

const PUBKEY_A =
  "02c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5";
const PUBKEY_B =
  "02f9308a019258c31049344f85f89d5229b531c845836f99b08601f113bce036f9";
const SIGNER_PUBKEY =
  "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798";
const FUNDING_SCRIPT = "0014751e76e8199196d454941c45d1b3a323f1433bd6";
const ACCOUNT = "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1";
const NON_ZERO_REF =
  "0x0000000000000000000000000000000000000000000000000000000000000001" as Bytes32Hex;

describe("BTC deposit marker vectors (pkg/blockchain/btc/marker/testdata/vectors.json)", () => {
  it("has the 10 encode cases guardrail 4 requires, 4 of them expectError", () => {
    expect(vectors.encode).toHaveLength(10);
    expect(vectors.encode.filter((c) => c.expectError !== undefined)).toHaveLength(4);
  });

  it.each(vectors.encode)("encode: $case", (vector) => {
    const marker = {
      version: vector.marker.version as typeof MARKER_VERSION_1 | typeof MARKER_VERSION_2,
      address: hexToBytes(vector.marker.addressHex, "addressHex"),
      reference:
        vector.marker.referenceHex === null
          ? undefined
          : hexToBytes(vector.marker.referenceHex, "referenceHex"),
    };

    if (vector.expectError !== undefined) {
      expect(() => encodeMarkerPayload(marker)).toThrow();
      expect(() => encodeMarkerScript(marker)).toThrow();
      return;
    }

    expect(vector.payloadHex).toBeDefined();
    expect(vector.scriptHex).toBeDefined();
    expect(bytesToHex(encodeMarkerPayload(marker))).toBe(vector.payloadHex);
    expect(bytesToHex(encodeMarkerScript(marker))).toBe(vector.scriptHex);
  });

  it("pins the TS generic tag preimage and hash against vectors.json", () => {
    expect(GENERIC_DEPOSIT_TAG_PREIMAGE).toBe(vectors.genericTag.preimage);
    expect(GENERIC_DEPOSIT_TAG_HEX).toBe(vectors.genericTag.sha256Hex);
  });

  it("depositAddress() returns the P2WSH of taggedRedeemScript(<generic tag preimage>, threshold, pubkeys)", () => {
    const depositor = createDepositor();
    const pubkeys = [hexToBytes(PUBKEY_A, "PUBKEY_A"), hexToBytes(PUBKEY_B, "PUBKEY_B")];
    const script = taggedRedeemScript(GENERIC_DEPOSIT_TAG_PREIMAGE, 2, pubkeys);
    const expectedAddress = p2wsh({ type: "ms", script }, networkParams("regtest")).address;
    expect(depositor.depositAddress()).toBe(expectedAddress);
  });

  it("builds a 27-byte v0x01 marker for an absent reference and a 59-byte v0x02 marker for a non-zero one", async () => {
    const withoutRef = await preparedPsbtTransaction({ account: ACCOUNT });
    const withoutRefMarker = markerOutput(withoutRef.tx);
    expect(withoutRefMarker.amount).toBe(0n);
    expect(withoutRefMarker.script.length).toBe(27);
    expect(bytesToHex(withoutRefMarker.script)).toBe(
      bytesToHex(encodeMarkerScript({ version: MARKER_VERSION_1, address: hexToBytes(ACCOUNT, "ACCOUNT") })),
    );

    const withRef = await preparedPsbtTransaction({ account: ACCOUNT, ref: NON_ZERO_REF });
    const withRefMarker = markerOutput(withRef.tx);
    expect(withRefMarker.amount).toBe(0n);
    expect(withRefMarker.script.length).toBe(59);
    expect(bytesToHex(withRefMarker.script)).toBe(
      bytesToHex(
        encodeMarkerScript({
          version: MARKER_VERSION_2,
          address: hexToBytes(ACCOUNT, "ACCOUNT"),
          reference: hexToBytes(NON_ZERO_REF.slice(2), "NON_ZERO_REF"),
        }),
      ),
    );
  });

  it("asserts the built transaction's output set: generic value output, zero-value OP_RETURN marker, unchanged change", async () => {
    const prepared = await preparedPsbtTransaction({ account: ACCOUNT });

    expect(prepared.tx.outputsLength).toBe(3);
    const valueOutput = prepared.tx.getOutput(0);
    const marker = markerOutput(prepared.tx);
    const change = prepared.tx.getOutput(2);

    const pubkeys = [hexToBytes(PUBKEY_A, "PUBKEY_A"), hexToBytes(PUBKEY_B, "PUBKEY_B")];
    const deposit = depositPayment("regtest", GENERIC_DEPOSIT_TAG_PREIMAGE, 2, pubkeys);
    expect(bytesToHex(valueOutput.script ?? new Uint8Array())).toBe(bytesToHex(deposit.script));
    expect(valueOutput.amount).toBe(50_000n);

    expect(marker.amount).toBe(0n);
    expect(marker.script[0]).toBe(0x6a); // OP_RETURN
    expect(marker.script.length).toBe(27);

    expect(change.amount).toBeGreaterThan(0n);
    expect(bytesToHex(change.script ?? new Uint8Array())).toBe(FUNDING_SCRIPT);

    // prepareDepositPsbt's inputIndexesToSign still covers exactly the inputs.
    expect(prepared.inputIndexesToSign).toEqual([0]);
  });
});

function markerOutput(tx: Transaction): { script: Uint8Array; amount: bigint } {
  for (let i = 0; i < tx.outputsLength; i += 1) {
    const output = tx.getOutput(i);
    if (output.script !== undefined && output.script[0] === 0x6a) {
      return { script: output.script, amount: output.amount ?? 0n };
    }
  }
  throw new Error("no OP_RETURN marker output found");
}

async function preparedPsbtTransaction(destination: {
  account: string;
  ref?: Bytes32Hex;
}): Promise<{ tx: Transaction; inputIndexesToSign: readonly number[] }> {
  const depositor = createDepositor();
  const prepared = await depositor.prepareDepositPsbt(
    {
      asset: BITCOIN_NATIVE_ASSET,
      amount: "0.0005",
      destination,
    },
    { publicKey: SIGNER_PUBKEY },
  );
  const tx = Transaction.fromPSBT(hexToBytes(prepared.psbtHex, "psbtHex"));
  return { tx, inputIndexesToSign: prepared.inputIndexesToSign };
}

function createDepositor(rpc: BitcoinRpc = createRpc()): BitcoinVaultDepositor {
  return new BitcoinVaultDepositor({
    network: "regtest",
    rpc,
    vaultPubkeys: [PUBKEY_B, PUBKEY_A],
    threshold: 2,
    fallbackFeeRateSatPerVByte: 5n,
  });
}

function createRpc(): BitcoinRpc {
  return {
    listUnspent: async () => [
      {
        txid: "01".repeat(32),
        vout: 0,
        amountSats: 100_000n,
        confirmations: 1,
        scriptPubKey: FUNDING_SCRIPT,
      },
    ],
    estimateSmartFeeSatPerVByte: async () => 5n,
    sendRawTransaction: async () => "00".repeat(32),
    getRawTransaction: async () => null,
  };
}
