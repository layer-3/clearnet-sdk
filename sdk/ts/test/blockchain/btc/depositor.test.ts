import { SigHash, Transaction } from "@scure/btc-signer";
import { describe, expect, expectTypeOf, it, vi } from "vitest";

import {
  BITCOIN_NATIVE_ASSET,
  bitcoinDepositId,
  BitcoinCoreRpcClient,
  BitcoinRpcError,
  BitcoinVaultDepositor,
  ClearnetSdkError,
} from "../../../src/index.js";
import type {
  BitcoinDepositorConfig,
  BitcoinRpc,
  BitcoinSigner,
  BitcoinSubmitDepositInput,
  BitcoinPsbtSignerInfo,
  BitcoinRawTransaction,
  Bytes32Hex,
  SubmitDepositResult,
  VaultDepositor,
} from "../../../src/index.js";
import { depositPayment } from "../../../src/blockchain/btc/address.js";
import {
  encodeMarkerScript,
  GENERIC_DEPOSIT_TAG_PREIMAGE,
  MARKER_VERSION_1,
} from "../../../src/blockchain/btc/marker.js";
import { estimateDepositFeeSats } from "../../../src/blockchain/btc/utxo.js";
import { requireClearnetAccount } from "../../../src/blockchain/btc/validation.js";
import {
  bytesToHex,
  concatBytes,
  hexToBytes,
} from "../../../src/core/bytes.js";

const ZERO_REF =
  "0x0000000000000000000000000000000000000000000000000000000000000000" as Bytes32Hex;
const NON_ZERO_REF =
  "0x0000000000000000000000000000000000000000000000000000000000000001" as Bytes32Hex;
// 20-byte hex Clearnet account address (ADR-023 §3)
const ACCOUNT = "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1";
const PUBKEY_A =
  "02c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5";
const PUBKEY_B =
  "02f9308a019258c31049344f85f89d5229b531c845836f99b08601f113bce036f9";
const SIGNER_PUBKEY =
  "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798";
const DISPLAY_TXID =
  "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f";
const FUNDING_SCRIPT = "0014751e76e8199196d454941c45d1b3a323f1433bd6";
const NESTED_SEGWIT_ADDRESS = "2NAUYAHhujozruyzpsFRP63mbrdaU5wnEpN";
const NESTED_SEGWIT_SCRIPT = "a914bcfeb728b584253d5f3f70bcb780e9ef218a68f487";
const DEPOSIT_SCRIPT = bytesToHex(
  depositPayment(
    "regtest",
    GENERIC_DEPOSIT_TAG_PREIMAGE,
    2,
    [hexToBytes(PUBKEY_A, "PUBKEY_A"), hexToBytes(PUBKEY_B, "PUBKEY_B")],
  ).script,
);
const MARKER_SCRIPT = bytesToHex(
  encodeMarkerScript({
    version: MARKER_VERSION_1,
    address: hexToBytes(ACCOUNT, "ACCOUNT"),
  }),
);

describe("BitcoinVaultDepositor", () => {
  it("matches the public depositor and input contracts", () => {
    expectTypeOf<BitcoinVaultDepositor>().toMatchTypeOf<
      VaultDepositor<BitcoinSubmitDepositInput>
    >();
    expectTypeOf<BitcoinSubmitDepositInput["amount"]>().toEqualTypeOf<string>();
    expectTypeOf<SubmitDepositResult>().toEqualTypeOf<{
      txHash: string;
      depositId: string;
    }>();
    expect(BITCOIN_NATIVE_ASSET).toBe("");
  });

  it("derives a stable regtest depositor address, a stable generic deposit address, and txIDs from txid bytes", async () => {
    const depositor = createDepositor();

    await expect(depositor.depositorAddress()).resolves.toBe(
      "bcrt1qw508d6qejxtdg4y5r3zarvary0c5xw7kygt080",
    );
    expect(depositor.depositAddress()).toMatch(/^bcrt1q[023456789acdefghjklmnpqrstuvwxyz]+$/);
    expect(depositor.depositAddress()).toBe(depositor.depositAddress());

    expect(depositor.txIDFromTxid(DISPLAY_TXID)).toBe(DISPLAY_TXID);
  });

  it("validates constructor, amount, asset, reference, and options before RPC work", async () => {
    expect(
      () =>
        new BitcoinVaultDepositor({
          ...baseConfig(),
          threshold: 3,
        }),
    ).toThrowError(ClearnetSdkError);

    const rpc = createRpc();
    const depositor = createDepositor({ rpc });

    await expect(
      depositor.submitDeposit(null as unknown as BitcoinSubmitDepositInput),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "destination is required and must be an object",
    });
    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: "DOGE",
        amount: "1",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "1",
        destination: { account: "not-a-hex-address" },
      }),
    ).rejects.toMatchObject({ code: "INVALID_ADDRESS" });
    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "1",
        destination: { account: ACCOUNT, ref: "invoice-1" as Bytes32Hex },
      }),
    ).rejects.toMatchObject({
      code: "INVALID_REFERENCE",
      message: "destination.ref must be a 32-byte hex value",
    });
    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "1",
        destination: { account: "" },
      }),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "destination.account must be a non-empty string",
    });
    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "1",
        destination: { account: 123 as unknown as string },
      }),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "destination.account must be a non-empty string",
    });
    await expect(
      depositor.submitDeposit(
        {
          asset: BITCOIN_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT, ref: ZERO_REF },
        },
        null as never,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: "submit options must be an object",
    });
    expect(rpc.listUnspent).not.toHaveBeenCalled();
  });

  it("selects UTXOs deterministically and rejects insufficient eligible balance", async () => {
    const rpc = createRpc({
      listUnspent: [
        utxo("ff".repeat(32), 0, 40_000n),
        utxo("00".repeat(32), 1, 60_000n),
      ],
    });
    const depositor = createDepositor({ rpc });

    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0012",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INSUFFICIENT_FUNDS" });

    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("submits a signed native BTC deposit and returns a display txid", async () => {
    const rpc = createRpc({
      listUnspent: [
        utxo("01".repeat(32), 0, 100_000n, FUNDING_SCRIPT),
        utxo("02".repeat(32), 0, 30_000n, FUNDING_SCRIPT),
      ],
      sendRawTransaction: undefined,
    });
    const signer = createSigner();
    const depositor = createDepositor({ rpc, signer });
    const onSubmitted = vi.fn();

    const txID = await depositor.submitDeposit(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT, ref: ZERO_REF },
      },
      { onSubmitted },
    );

    expect(txID.txHash).toMatch(/^[a-f0-9]{64}$/);
    expect(txID.depositId).toEqual(bitcoinDepositId(txID.txHash, 0));
    expect(txID.depositId).toEqual(`${txID.txHash}:0`);
    expect(broadcastOutputScript(rpc, 0)).toBe(DEPOSIT_SCRIPT);
    expect(rpc.listUnspent).toHaveBeenCalledExactlyOnceWith(1, [
      "bcrt1qw508d6qejxtdg4y5r3zarvary0c5xw7kygt080",
    ]);
    expect(rpc.estimateSmartFeeSatPerVByte).toHaveBeenCalledExactlyOnceWith(6, 5n);
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
    expect(rpc.sendRawTransaction).toHaveBeenCalledWith(expect.stringMatching(/^[a-f0-9]+$/));
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(txID);
    expect(signer.getPublicKeyCompressed).toHaveBeenCalledTimes(1);
  });

  it("accepts a non-zero destination.ref instead of rejecting it", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("0a".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      sendRawTransaction: undefined,
    });
    const depositor = createDepositor({ rpc, signer: createSigner() });

    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT, ref: NON_ZERO_REF },
      }),
    ).resolves.toMatchObject({ txHash: expect.stringMatching(/^[a-f0-9]{64}$/) });
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
  });

  it("prepares an unsigned PSBT for wallet signing without a configured local signer", async () => {
    const rpc = createRpc({
      listUnspent: [
        utxo("01".repeat(32), 0, 100_000n, FUNDING_SCRIPT),
        utxo("02".repeat(32), 0, 30_000n, FUNDING_SCRIPT),
      ],
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const wallet: BitcoinPsbtSignerInfo = {
      publicKey: SIGNER_PUBKEY,
      address: "bcrt1qw508d6qejxtdg4y5r3zarvary0c5xw7kygt080",
    };

    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.000995",
        destination: { account: ACCOUNT },
      },
      wallet,
    );

    expect(prepared.psbtHex).toMatch(/^70736274ff[a-f0-9]+$/);
    expect(prepared.inputIndexesToSign).toEqual([0, 1]);
    expect(prepared.fundingAddress).toBe(wallet.address);
    expect(prepared.depositAddress).toMatch(/^bcrt1q/);
    expect(prepared.unsignedTxID).toMatch(/^[a-f0-9]{64}$/);
    expect(rpc.listUnspent).toHaveBeenCalledExactlyOnceWith(1, [wallet.address]);
    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("finalizes and broadcasts a wallet-signed PSBT", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("05".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      sendRawTransaction: undefined,
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      { publicKey: SIGNER_PUBKEY },
    );
    const tx = Transaction.fromPSBT(hexToBytes(prepared.psbtHex, "prepared.psbtHex"));
    for (const index of prepared.inputIndexesToSign) {
      tx.updateInput(
        index,
        {
          partialSig: [[
            hexToBytes(SIGNER_PUBKEY, "SIGNER_PUBKEY"),
            concatBytes(fakeDerSignature(), new Uint8Array([SigHash.ALL])),
          ]],
        },
        true,
      );
    }
    const onSubmitted = vi.fn();

    const txID = await depositor.submitSignedDepositPsbt(
      bytesToHex(tx.toPSBT()),
      prepared.expectedOutputs,
      { onSubmitted },
    );

    expect(txID.txHash).toEqual(prepared.unsignedTxID);
    expect(txID.depositId).toEqual(bitcoinDepositId(prepared.unsignedTxID, 0));
    expect(broadcastOutputScript(rpc, 0)).toBe(DEPOSIT_SCRIPT);
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
    expect(rpc.sendRawTransaction).toHaveBeenCalledWith(expect.stringMatching(/^[a-f0-9]+$/));
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(txID);
  });

  it("prepares and broadcasts a wallet-signed nested SegWit PSBT", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("06".repeat(32), 0, 100_000n, NESTED_SEGWIT_SCRIPT)],
      sendRawTransaction: undefined,
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      {
        publicKey: SIGNER_PUBKEY,
        address: NESTED_SEGWIT_ADDRESS,
        addressType: "p2sh",
      },
    );
    const tx = Transaction.fromPSBT(hexToBytes(prepared.psbtHex, "prepared.psbtHex"));
    for (const index of prepared.inputIndexesToSign) {
      tx.updateInput(
        index,
        {
          partialSig: [[
            hexToBytes(SIGNER_PUBKEY, "SIGNER_PUBKEY"),
            concatBytes(fakeDerSignature(), new Uint8Array([SigHash.ALL])),
          ]],
        },
        true,
      );
    }

    const txID = await depositor.submitSignedDepositPsbt(
      bytesToHex(tx.toPSBT()),
      prepared.expectedOutputs,
    );

    expect(prepared.fundingAddress).toBe(NESTED_SEGWIT_ADDRESS);
    expect(txID.txHash).toMatch(/^[a-f0-9]{64}$/);
    expect(txID.txHash).not.toEqual(prepared.unsignedTxID);
    expect(txID.depositId).toEqual(bitcoinDepositId(txID.txHash, 0));
    expect(rpc.listUnspent).toHaveBeenCalledExactlyOnceWith(1, [
      NESTED_SEGWIT_ADDRESS,
    ]);
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
  });

  it("rejects submitSignedDepositPsbt when expectedOutputs is missing, empty, or malformed", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("0b".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      { publicKey: SIGNER_PUBKEY },
    );
    const signedPsbtHex = bytesToHex(signInputs(prepared).toPSBT());

    await expect(
      depositor.submitSignedDepositPsbt(
        signedPsbtHex,
        undefined as unknown as typeof prepared.expectedOutputs,
      ),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    await expect(
      depositor.submitSignedDepositPsbt(signedPsbtHex, []),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    await expect(
      depositor.submitSignedDepositPsbt(signedPsbtHex, [{ script: "zz", amount: 0n }]),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    await expect(
      depositor.submitSignedDepositPsbt(signedPsbtHex, [
        { script: "6a00", amount: -1n },
      ]),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("rejects a signed PSBT with a stripped marker output", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("0c".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      { publicKey: SIGNER_PUBKEY },
    );
    // Marker output is expectedOutputs[1] (see prepareUnsignedDepositTx: deposit, marker, [change]).
    const strippedOutputs = prepared.expectedOutputs.filter((_, index) => index !== 1);
    const maliciousPsbtHex = signInputs(
      prepared,
      rebuildTransaction(prepared.psbtHex, strippedOutputs),
    ).toPSBT();

    await expect(
      depositor.submitSignedDepositPsbt(
        bytesToHex(maliciousPsbtHex),
        prepared.expectedOutputs,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: expect.stringContaining("output(s), expected"),
    });
    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("rejects a signed PSBT with an altered marker script", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("0d".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      { publicKey: SIGNER_PUBKEY },
    );
    const alteredOutputs = prepared.expectedOutputs.map((output, index) =>
      index === 1 ? { ...output, script: flipLastHexChar(output.script) } : output,
    );
    const maliciousPsbtHex = signInputs(
      prepared,
      rebuildTransaction(prepared.psbtHex, alteredOutputs),
    ).toPSBT();

    await expect(
      depositor.submitSignedDepositPsbt(
        bytesToHex(maliciousPsbtHex),
        prepared.expectedOutputs,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: expect.stringContaining("scriptPubKey does not match"),
    });
    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("rejects a signed PSBT whose marker output carries a non-zero amount", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("0e".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      { publicKey: SIGNER_PUBKEY },
    );
    const alteredOutputs = prepared.expectedOutputs.map((output, index) =>
      index === 1 ? { ...output, amount: 546n } : output,
    );
    const maliciousPsbtHex = signInputs(
      prepared,
      rebuildTransaction(prepared.psbtHex, alteredOutputs),
    ).toPSBT();

    await expect(
      depositor.submitSignedDepositPsbt(
        bytesToHex(maliciousPsbtHex),
        prepared.expectedOutputs,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: expect.stringContaining("amount does not match"),
    });
    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("rejects a signed PSBT with an altered value-output amount", async () => {
    const rpc = createRpc({
      listUnspent: [utxo("0f".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
    });
    const depositor = createDepositor({ rpc, signer: undefined });
    const prepared = await depositor.prepareDepositPsbt(
      {
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      },
      { publicKey: SIGNER_PUBKEY },
    );
    const alteredOutputs = prepared.expectedOutputs.map((output, index) =>
      index === 0 ? { ...output, amount: output.amount + 1n } : output,
    );
    const maliciousPsbtHex = signInputs(
      prepared,
      rebuildTransaction(prepared.psbtHex, alteredOutputs),
    ).toPSBT();

    await expect(
      depositor.submitSignedDepositPsbt(
        bytesToHex(maliciousPsbtHex),
        prepared.expectedOutputs,
      ),
    ).rejects.toMatchObject({
      code: "INVALID_INPUT",
      message: expect.stringContaining("amount does not match"),
    });
    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it.each([
    ["pays another script", { script: FUNDING_SCRIPT }],
    ["carries no value", { amount: 0n }],
  ])(
    "rejects a signed PSBT whose output 0 %s, even when expectedOutputs agree",
    async (_name, change) => {
      const rpc = createRpc({
        listUnspent: [utxo("10".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      });
      const depositor = createDepositor({ rpc, signer: undefined });
      const prepared = await depositor.prepareDepositPsbt(
        {
          asset: BITCOIN_NATIVE_ASSET,
          amount: "0.0005",
          destination: { account: ACCOUNT },
        },
        { publicKey: SIGNER_PUBKEY },
      );
      const alteredOutputs = prepared.expectedOutputs.map((output, index) =>
        index === 0 ? { ...output, ...change } : output,
      );
      const signedPsbtHex = signInputs(
        prepared,
        rebuildTransaction(prepared.psbtHex, alteredOutputs),
      ).toPSBT();

      await expect(
        depositor.submitSignedDepositPsbt(bytesToHex(signedPsbtHex), alteredOutputs),
      ).rejects.toMatchObject({
        code: "INVALID_INPUT",
        message: expect.stringContaining(
          `output 0 must pay the deposit address ${depositor.depositAddress()}`,
        ),
      });
      expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
    },
  );

  it("rejects PSBT preparation when wallet address and public key do not match", async () => {
    const depositor = createDepositor({ signer: undefined });

    await expect(
      depositor.prepareDepositPsbt(
        {
          asset: BITCOIN_NATIVE_ASSET,
          amount: "1",
          destination: { account: ACCOUNT },
        },
        {
          publicKey: SIGNER_PUBKEY,
          address: "bcrt1qexampleaddressdoesnotmatch",
        },
      ),
    ).rejects.toMatchObject({
      code: "INVALID_ADDRESS",
      message: "wallet address does not match wallet public key",
    });
  });

  it("handles already-known and missing-input broadcast outcomes distinctly", async () => {
    const alreadyKnownRpc = createRpc({
      listUnspent: [utxo("03".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      sendRawTransactionError: new BitcoinRpcError(-27, "transaction already in block chain"),
    });
    const alreadyKnownTxID = await createDepositor({ rpc: alreadyKnownRpc }).submitDeposit({
      asset: BITCOIN_NATIVE_ASSET,
      amount: "0.0005",
      destination: { account: ACCOUNT },
    });
    expect(alreadyKnownTxID.txHash).toMatch(/^[a-f0-9]{64}$/);
    expect(alreadyKnownTxID.depositId).toEqual(
      bitcoinDepositId(alreadyKnownTxID.txHash, 0),
    );

    const missingRpc = createRpc({
      listUnspent: [utxo("04".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      sendRawTransactionError: new BitcoinRpcError(-25, "bad-txns-inputs-missingorspent"),
      rawTransaction: { txid: "not-the-computed-txid", confirmations: 0, outputs: [] },
    });
    await expect(
      createDepositor({ rpc: missingRpc }).submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "RPC_ERROR", txHash: expect.any(String) });

    const sendError = new BitcoinRpcError(-25, "bad-txns-inputs-missingorspent");
    const lookupError = new Error("lookup failed");
    const lookupFailsRpc = createRpc({
      listUnspent: [utxo("07".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      sendRawTransactionError: sendError,
      rawTransactionError: lookupError,
    });
    await expect(
      createDepositor({ rpc: lookupFailsRpc }).submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: "0.0005",
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({
      code: "RPC_ERROR",
      txHash: expect.any(String),
      cause: sendError,
    });
  });

  it("verifies absent, pending, confirmed, and malformed txIDs", async () => {
    const txID = DISPLAY_TXID;
    const depositId = bitcoinDepositId(txID, 0);
    const depositor = createDepositor({
      rpc: createRpc({ rawTransaction: null }),
    });
    await expect(depositor.chainDepositStatus(txID, depositId, 1)).resolves.toBe("absent");

    const pending = createDepositor({
      rpc: createRpc({ rawTransaction: depositRawTransaction(0) }),
    });
    await expect(pending.chainDepositStatus(txID, depositId, 1)).resolves.toBe("pending");
    await expect(pending.chainDepositStatus(txID, depositId, 0)).resolves.toBe("pending");

    const oneConf = createDepositor({
      rpc: createRpc({ rawTransaction: depositRawTransaction(1) }),
    });
    await expect(oneConf.chainDepositStatus(txID, depositId, 0)).resolves.toBe("confirmed");

    const confirmed = createDepositor({
      rpc: createRpc({ rawTransaction: depositRawTransaction(2) }),
    });
    await expect(confirmed.chainDepositStatus(txID, depositId, 2)).resolves.toBe("confirmed");
    await expect(confirmed.chainDepositStatus(txID, depositId, 2n)).resolves.toBe("confirmed");
    await expect(confirmed.chainDepositStatus(txID, depositId, 3)).resolves.toBe("pending");
    await expect(
      confirmed.chainDepositStatus("not-a-txid", "not-a-txid", 1),
    ).rejects.toMatchObject({ code: "INVALID_TX_ID" });

    const invalidMinConfRpc = createRpc({
      rawTransaction: depositRawTransaction(2),
    });
    const invalidMinConf = createDepositor({ rpc: invalidMinConfRpc });
    for (const minConfirmations of [
      1.5,
      Number.MAX_SAFE_INTEGER + 1,
      1n << 80n,
    ]) {
      await expect(
        invalidMinConf.chainDepositStatus(txID, depositId, minConfirmations),
      ).rejects.toMatchObject({
        code: "INVALID_CONFIRMATIONS",
        message: "minConfirmations must be a non-negative safe integer",
      });
    }
    expect(invalidMinConfRpc.getRawTransaction).not.toHaveBeenCalled();
  });

  it("requires output vout to pay the generic deposit address and exactly one valid marker", async () => {
    const txID = DISPLAY_TXID;
    const change = { valueSats: 1_000n, scriptPubKey: FUNDING_SCRIPT };
    const deposit = { valueSats: 50_000n, scriptPubKey: DEPOSIT_SCRIPT };
    const marker = { valueSats: 0n, scriptPubKey: MARKER_SCRIPT };
    const cases: [string, BitcoinRawTransaction["outputs"], number, string][] = [
      ["deposit at output 0", [deposit, marker, change], 0, "confirmed"],
      ["deposit at output 2", [change, marker, deposit], 2, "confirmed"],
      ["upper-case script hex", [{ ...deposit, scriptPubKey: DEPOSIT_SCRIPT.toUpperCase() }, marker], 0, "confirmed"],
      ["vout is the change output", [deposit, marker, change], 2, "absent"],
      ["vout is the marker", [deposit, marker, change], 1, "absent"],
      ["vout out of range", [deposit, marker], 5, "absent"],
      ["zero-value deposit output", [{ ...deposit, valueSats: 0n }, marker], 0, "absent"],
      ["no marker", [deposit, change], 0, "absent"],
      ["two markers", [deposit, marker, marker], 0, "absent"],
      ["invalid marker candidate", [deposit, { valueSats: 0n, scriptPubKey: `6a1a${MARKER_SCRIPT.slice(4)}00` }], 0, "absent"],
      ["non-canonical marker push", [deposit, { valueSats: 0n, scriptPubKey: `6a4c19${MARKER_SCRIPT.slice(4)}` }], 0, "absent"],
      ["undecodable script hex", [deposit, marker, { valueSats: 1n, scriptPubKey: "zz" }], 0, "absent"],
      ["odd-length script hex", [deposit, marker, { valueSats: 1n, scriptPubKey: "001" }], 0, "absent"],
    ];
    for (const [name, outputs, vout, want] of cases) {
      const depositor = createDepositor({
        rpc: createRpc({ rawTransaction: { txid: txID, confirmations: 1, outputs } }),
      });
      await expect(
        depositor.chainDepositStatus(txID, bitcoinDepositId(txID, vout), 1),
        name,
      ).resolves.toBe(want);
    }
  });

  it("rejects a deposit ID that is not <txHash>:<vout> before RPC", async () => {
    const rpc = createRpc({ rawTransaction: depositRawTransaction(1) });
    const depositor = createDepositor({ rpc });
    for (const depositId of [
      DISPLAY_TXID,
      `${DISPLAY_TXID}:01`,
      `${DISPLAY_TXID}:-1`,
      `${DISPLAY_TXID}:4294967296`,
      `${"ff".repeat(32)}:0`,
      `${DISPLAY_TXID.toUpperCase()}:0`,
    ]) {
      await expect(
        depositor.chainDepositStatus(DISPLAY_TXID, depositId, 1),
      ).rejects.toMatchObject({ code: "INVALID_DEPOSIT_ID" });
    }
    await expect(
      depositor.chainDepositStatus(DISPLAY_TXID.toUpperCase(), bitcoinDepositId(DISPLAY_TXID, 0), 1),
    ).rejects.toMatchObject({ code: "INVALID_DEPOSIT_ID" });
    expect(rpc.getRawTransaction).not.toHaveBeenCalled();
    // The deposit ID keeps the txid's case; the RPC lookup does not depend on it.
    const upper = DISPLAY_TXID.toUpperCase();
    await expect(
      depositor.chainDepositStatus(upper, bitcoinDepositId(upper, 0), 1),
    ).resolves.toBe("confirmed");
  });

  it("parses getrawtransaction outputs with exact satoshi values", async () => {
    const fetchMock = vi.fn(async () =>
      jsonRpcResponse({
        txid: DISPLAY_TXID,
        confirmations: 3,
        vout: [
          { value: 20999999.9769, n: 0, scriptPubKey: { hex: DEPOSIT_SCRIPT, type: "witness_v0_scripthash" } },
          { value: 0, n: 1, scriptPubKey: { hex: MARKER_SCRIPT, type: "nulldata" } },
          { value: 0.00000001, n: 2, scriptPubKey: { hex: FUNDING_SCRIPT } },
        ],
      }),
    );
    const client = new BitcoinCoreRpcClient({
      url: "/btc-rpc",
      fetch: fetchMock as unknown as typeof fetch,
    });

    await expect(client.getRawTransaction(DISPLAY_TXID)).resolves.toEqual({
      txid: DISPLAY_TXID,
      confirmations: 3,
      outputs: [
        { valueSats: 2099999997690000n, scriptPubKey: DEPOSIT_SCRIPT },
        { valueSats: 0n, scriptPubKey: MARKER_SCRIPT },
        { valueSats: 1n, scriptPubKey: FUNDING_SCRIPT },
      ],
    });

    fetchMock.mockImplementationOnce(async () =>
      jsonRpcResponse({ txid: DISPLAY_TXID, confirmations: 3 }),
    );
    await expect(client.getRawTransaction(DISPLAY_TXID)).rejects.toMatchObject({
      code: "RPC_ERROR",
    });
  });

  it("allows zero funding confirmations but keeps fee knobs positive", () => {
    expect(
      () => new BitcoinVaultDepositor(baseConfig({ minFundingConfirmations: 0 })),
    ).not.toThrow();
    expect(
      () => new BitcoinVaultDepositor(baseConfig({ feeTargetBlocks: 0 })),
    ).toThrowError(ClearnetSdkError);
    expect(
      () => new BitcoinVaultDepositor(baseConfig({ fallbackFeeRateSatPerVByte: 0n })),
    ).toThrowError(ClearnetSdkError);
    expect(
      () => new BitcoinVaultDepositor(baseConfig({ dustThresholdSats: 0 })),
    ).toThrowError(ClearnetSdkError);
  });

  it("wraps local transaction finalization failures as invalid input", async () => {
    const cause = new Error("finalize failed");
    const finalize = vi
      .spyOn(Transaction.prototype, "finalize")
      .mockImplementationOnce(() => {
        throw cause;
      });
    const depositor = createDepositor({
      rpc: createRpc({
        listUnspent: [utxo("08".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      }),
    });

    try {
      await expect(
        depositor.submitDeposit({
          asset: BITCOIN_NATIVE_ASSET,
          amount: "0.0005",
          destination: { account: ACCOUNT },
        }),
      ).rejects.toMatchObject({
        code: "INVALID_INPUT",
        message: "btc: transaction finalization failed",
        cause,
      });
    } finally {
      finalize.mockRestore();
    }
  });

  it("uses address-type-aware fee estimates that account for the marker output (DoD item 18)", () => {
    // hasReference defaults to false: a 27-byte v0x01 marker adds 36 vbytes
    // (85 base + 36 marker + 68 * inputs) * feeRate.
    expect(estimateDepositFeeSats(1, 1n, "p2wpkh")).toBe(189n);
    expect(estimateDepositFeeSats(2, 5n, "p2wpkh")).toBe(1285n);
    expect(estimateDepositFeeSats(1, 1n, "p2sh")).toBe(213n);
    expect(estimateDepositFeeSats(2, 5n, "p2sh")).toBe(1525n);

    // hasReference: true selects the 59-byte v0x02 marker (68 vbytes) instead.
    expect(estimateDepositFeeSats(1, 1n, "p2wpkh", true)).toBe(221n);
    expect(estimateDepositFeeSats(2, 5n, "p2wpkh", true)).toBe(1445n);
  });

  it("estimates at least the finalized signed transaction vsize", async () => {
    const p2wpkh = await signedPreparedTransaction("p2wpkh", FUNDING_SCRIPT);
    expect(estimateDepositFeeSats(1, 1n, "p2wpkh")).toBeGreaterThanOrEqual(
      BigInt(p2wpkh.vsize),
    );

    const p2sh = await signedPreparedTransaction("p2sh", NESTED_SEGWIT_SCRIPT);
    expect(estimateDepositFeeSats(1, 1n, "p2sh")).toBeGreaterThanOrEqual(
      BigInt(p2sh.vsize),
    );
  });

  it("sends Bitcoin Core RPC Basic Auth only when both credentials are supplied", async () => {
    const fetchMock = vi.fn(async () => jsonRpcResponse([]));
    const client = new BitcoinCoreRpcClient({
      url: "/btc-rpc",
      wallet: "sdk",
      fetch: fetchMock as unknown as typeof fetch,
    });

    await client.listUnspent(1, ["bcrt1qexample"]);

    expect(fetchMock).toHaveBeenCalledExactlyOnceWith(
      "/btc-rpc/wallet/sdk",
      expect.objectContaining({
        method: "POST",
        headers: expect.not.objectContaining({ Authorization: expect.any(String) }),
      }),
    );
    const listUnspentRequest = (
      fetchMock.mock.calls as unknown as Array<[string, RequestInit]>
    )[0]?.[1];
    expect(JSON.parse(listUnspentRequest?.body as string).params).toEqual([
      1,
      9_999_999,
      ["bcrt1qexample"],
    ]);

    const authedFetch = vi.fn(async () => jsonRpcResponse("ok"));
    const authed = new BitcoinCoreRpcClient({
      url: "http://127.0.0.1:18443",
      username: "sdk",
      password: "sdk",
      fetch: authedFetch as unknown as typeof fetch,
    });
    await authed.sendRawTransaction("00");
    expect(authedFetch).toHaveBeenCalledWith(
      "http://127.0.0.1:18443",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: `Basic ${btoa("sdk:sdk")}`,
        }),
      }),
    );
    expect(
      () => new BitcoinCoreRpcClient({ url: "/btc-rpc", username: "sdk" }),
    ).toThrowError(ClearnetSdkError);
  });

  it("converts Bitcoin Core BTC amounts to satoshis with 8-decimal precision", async () => {
    const fetchMock = vi.fn(async () =>
      jsonRpcResponse([
        {
          txid: DISPLAY_TXID,
          vout: 0,
          amount: 1.23456789,
          confirmations: 1,
          scriptPubKey: FUNDING_SCRIPT,
        },
        {
          txid: DISPLAY_TXID,
          vout: 1,
          amount: 20999999.9769,
          confirmations: 1,
          scriptPubKey: FUNDING_SCRIPT,
        },
      ]),
    );
    const client = new BitcoinCoreRpcClient({
      url: "/btc-rpc",
      wallet: "sdk",
      fetch: fetchMock as unknown as typeof fetch,
    });

    await expect(client.listUnspent(1, ["bcrt1qexample"])).resolves.toEqual([
      expect.objectContaining({ amountSats: 123456789n }),
      expect.objectContaining({ amountSats: 2099999997690000n }),
    ]);
  });

  it("preserves JSON-RPC errors and wraps HTTP parse failures", async () => {
    const rpcErrorFetch = vi.fn(async () =>
      new Response(
        JSON.stringify({
          result: null,
          error: { code: -25, message: "bad-txns-inputs-missingorspent" },
        }),
        { status: 500, headers: { "content-type": "application/json" } },
      ),
    );
    const rpcErrorClient = new BitcoinCoreRpcClient({
      url: "/btc-rpc",
      fetch: rpcErrorFetch as unknown as typeof fetch,
    });
    await expect(rpcErrorClient.sendRawTransaction("00")).rejects.toMatchObject({
      code: -25,
      message: "bad-txns-inputs-missingorspent",
    });

    const htmlFetch = vi.fn(async () =>
      new Response("<html>not json</html>", {
        status: 500,
        headers: { "content-type": "text/html" },
      }),
    );
    const htmlClient = new BitcoinCoreRpcClient({
      url: "/btc-rpc",
      fetch: htmlFetch as unknown as typeof fetch,
    });
    await expect(htmlClient.sendRawTransaction("00")).rejects.toMatchObject({
      code: "RPC_ERROR",
      message: "btc rpc sendrawtransaction HTTP 500",
    });

    const networkError = new TypeError("fetch failed");
    const networkFetch = vi.fn(async () => {
      throw networkError;
    });
    const networkClient = new BitcoinCoreRpcClient({
      url: "/btc-rpc",
      fetch: networkFetch as unknown as typeof fetch,
    });
    await expect(networkClient.sendRawTransaction("00")).rejects.toMatchObject({
      code: "RPC_ERROR",
      message: "btc rpc sendrawtransaction request failed",
      cause: networkError,
    });
  });
});

// Pins the exact accepted/rejected input set for requireClearnetAccount,
// bare hex, an optional case-insensitive "0x" prefix, a yellow://.../user/<hex>
// URI's last segment, and surrounding whitespace. It also pins that an ADR-015
// sub-account URI (yellow://.../user/<addr>/tag/<32-byte-ref>) is rejected
// rather than silently parsed as the trailing 32-byte reference.
describe("requireClearnetAccount", () => {
  const addrHex = "000102030405060708090a0b0c0d0e0f10111213";
  const refHex =
    "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f";
  const want = hexToBytes(addrHex, "want");

  it.each([
    ["bare hex", addrHex],
    ["0x prefix", `0x${addrHex}`],
    ["0X prefix", `0X${addrHex}`],
    ["uppercase hex", addrHex.toUpperCase()],
    ["whitespace padded", `  ${addrHex}  `],
    ["tab/newline padded", `\t${addrHex}\n`],
    ["yellow URI", `yellow://ynet/user/${addrHex}`],
    ["whitespace padded yellow URI", `  yellow://ynet/user/${addrHex}  `],
  ])("accepts %s", (_name, input) => {
    expect(requireClearnetAccount(input)).toEqual(want);
  });

  it.each([
    ["empty string", ""],
    ["non-hex", "not-a-hex-address"],
    ["19 bytes", addrHex.slice(0, 38)],
    ["21 bytes", `${addrHex}00`],
    [
      "ADR-015 sub-account URI (last segment is the 32-byte reference)",
      `yellow://ynet/user/${addrHex}/tag/${refHex}`,
    ],
  ])("rejects %s", (_name, input) => {
    expect(() => requireClearnetAccount(input)).toThrowError(
      expect.objectContaining({ code: "INVALID_ADDRESS" }),
    );
  });
});

function baseConfig(
  overrides: Partial<BitcoinDepositorConfig> = {},
): BitcoinDepositorConfig {
  return {
    network: "regtest",
    rpc: createRpc(),
    signer: createSigner(),
    vaultPubkeys: [PUBKEY_B, PUBKEY_A],
    threshold: 2,
    fallbackFeeRateSatPerVByte: 5n,
    ...overrides,
  };
}

function createDepositor(
  overrides: Partial<BitcoinDepositorConfig> = {},
): BitcoinVaultDepositor {
  return new BitcoinVaultDepositor(baseConfig(overrides));
}

function createSigner(): BitcoinSigner {
  return {
    algorithm: "secp256k1",
    getPublicKeyCompressed: vi.fn(async () => hexToBytes(SIGNER_PUBKEY, "SIGNER_PUBKEY")),
    signDigest32: vi.fn(async () => fakeDerSignature()),
  };
}

async function signedPreparedTransaction(
  addressType: "p2wpkh" | "p2sh",
  scriptPubKey: string,
): Promise<Transaction> {
  const wallet =
    addressType === "p2sh"
      ? {
          publicKey: SIGNER_PUBKEY,
          address: NESTED_SEGWIT_ADDRESS,
          addressType,
        }
      : { publicKey: SIGNER_PUBKEY, addressType };
  const depositor = createDepositor({
    rpc: createRpc({
      listUnspent: [utxo("09".repeat(32), 0, 100_000n, scriptPubKey)],
    }),
    signer: undefined,
  });
  const prepared = await depositor.prepareDepositPsbt(
    {
      asset: BITCOIN_NATIVE_ASSET,
      amount: "0.0005",
      destination: { account: ACCOUNT },
    },
    wallet,
  );
  const tx = Transaction.fromPSBT(hexToBytes(prepared.psbtHex, "prepared.psbtHex"));
  for (const index of prepared.inputIndexesToSign) {
    tx.updateInput(
      index,
      {
        partialSig: [[
          hexToBytes(SIGNER_PUBKEY, "SIGNER_PUBKEY"),
          concatBytes(fakeDerSignature(), new Uint8Array([SigHash.ALL])),
        ]],
      },
      true,
    );
  }
  tx.finalize();
  return tx;
}

/**
 * Applies fake partial signatures to every input prepareDepositPsbt marked
 * for wallet signing. Defaults to parsing `prepared.psbtHex` fresh, but a
 * caller can pass an already-rebuilt (potentially output-tampered) Transaction
 * instead - the signatures are never checked for validity by finalize(), so
 * this is sufficient to reach a "wallet-signed" PSBT in either case.
 */
function signInputs(
  prepared: { psbtHex: string; inputIndexesToSign: readonly number[] },
  tx: Transaction = Transaction.fromPSBT(hexToBytes(prepared.psbtHex, "prepared.psbtHex")),
): Transaction {
  for (const index of prepared.inputIndexesToSign) {
    tx.updateInput(
      index,
      {
        partialSig: [[
          hexToBytes(SIGNER_PUBKEY, "SIGNER_PUBKEY"),
          concatBytes(fakeDerSignature(), new Uint8Array([SigHash.ALL])),
        ]],
      },
      true,
    );
  }
  return tx;
}

/**
 * Simulates a wallet that re-derives the transaction with a different output
 * set before signing (e.g. one that stripped or altered the marker output):
 * same inputs as the original PSBT, but exactly the given outputs, in order.
 */
function rebuildTransaction(
  originalPsbtHex: string,
  outputs: readonly { script: string; amount: bigint }[],
): Transaction {
  const original = Transaction.fromPSBT(hexToBytes(originalPsbtHex, "originalPsbtHex"));
  const rebuilt = new Transaction({ version: 1, allowUnknownOutputs: true });
  for (let index = 0; index < original.inputsLength; index += 1) {
    rebuilt.addInput(original.getInput(index));
  }
  for (const output of outputs) {
    rebuilt.addOutput({
      script: hexToBytes(output.script, "output.script"),
      amount: output.amount,
    });
  }
  return rebuilt;
}

function flipLastHexChar(hex: string): string {
  const last = hex.at(-1) ?? "0";
  const flipped = last === "0" ? "1" : "0";
  return `${hex.slice(0, -1)}${flipped}`;
}

function fakeDerSignature(): Uint8Array {
  return hexToBytes(
    "304402207fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a002207fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0",
    "fakeDerSignature",
  );
}

function createRpc(overrides: Partial<MockRpcState> = {}): BitcoinRpc {
  const state: MockRpcState = {
    listUnspent: [],
    rawTransaction: null,
    sendRawTransaction: DISPLAY_TXID,
    sendRawTransactionError: undefined,
    rawTransactionError: undefined,
    ...overrides,
  };
  return {
    listUnspent: vi.fn(async () => state.listUnspent),
    estimateSmartFeeSatPerVByte: vi.fn(async () => 5n),
    sendRawTransaction: vi.fn(async (hexTx: string) => {
      if (state.sendRawTransactionError !== undefined) {
        throw state.sendRawTransactionError;
      }
      return state.sendRawTransaction ?? txidFromRawTx(hexTx);
    }),
    getRawTransaction: vi.fn(async () => {
      if (state.rawTransactionError !== undefined) {
        throw state.rawTransactionError;
      }
      return state.rawTransaction;
    }),
  };
}

interface MockRpcState {
  listUnspent: Awaited<ReturnType<BitcoinRpc["listUnspent"]>>;
  rawTransaction: Awaited<ReturnType<BitcoinRpc["getRawTransaction"]>>;
  rawTransactionError: unknown;
  sendRawTransaction: string | undefined;
  sendRawTransactionError: unknown;
}

function utxo(
  txid: string,
  vout: number,
  amountSats: bigint,
  scriptPubKey = "",
) {
  return {
    txid,
    vout,
    amountSats,
    confirmations: 1,
    scriptPubKey,
  };
}

function depositRawTransaction(confirmations: number): BitcoinRawTransaction {
  return {
    txid: DISPLAY_TXID,
    confirmations,
    outputs: [
      { valueSats: 50_000n, scriptPubKey: DEPOSIT_SCRIPT },
      { valueSats: 0n, scriptPubKey: MARKER_SCRIPT },
    ],
  };
}

function broadcastOutputScript(rpc: BitcoinRpc, index: number): string {
  const call = vi.mocked(rpc.sendRawTransaction).mock.calls[0];
  if (call === undefined) {
    throw new Error("sendRawTransaction was not called");
  }
  const tx = Transaction.fromRaw(hexToBytes(call[0], "hexTx"), {
    allowUnknownOutputs: true,
  });
  return bytesToHex(tx.getOutput(index).script ?? new Uint8Array());
}

function txidFromRawTx(hexTx: string): string {
  expect(hexTx).toMatch(/^[a-f0-9]+$/);
  return DISPLAY_TXID;
}

function jsonRpcResponse(result: unknown): Response {
  return new Response(JSON.stringify({ result, error: null }), {
    status: 200,
    headers: { "content-type": "application/json" },
  });
}
