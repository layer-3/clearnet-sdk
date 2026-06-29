import { SigHash, Transaction } from "@scure/btc-signer";
import { describe, expect, expectTypeOf, it, vi } from "vitest";

import {
  BITCOIN_NATIVE_ASSET,
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
  Bytes32Hex,
  TxRef,
  VaultDepositor,
} from "../../../src/index.js";
import { estimateDepositFeeSats } from "../../../src/blockchain/btc/utxo.js";
import {
  bytesToHex,
  concatBytes,
  hexToBytes,
} from "../../../src/core/bytes.js";

const ZERO_REF =
  "0x0000000000000000000000000000000000000000000000000000000000000000" as Bytes32Hex;
const NON_ZERO_REF =
  "0x0000000000000000000000000000000000000000000000000000000000000001" as Bytes32Hex;
const ACCOUNT = "clearnet:bitcoin:account-a";
const PUBKEY_A =
  "02c6047f9441ed7d6d3045406e95c07cd85c778e4b8cef3ca7abac09b95c709ee5";
const PUBKEY_B =
  "02f9308a019258c31049344f85f89d5229b531c845836f99b08601f113bce036f9";
const SIGNER_PUBKEY =
  "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798";
const DISPLAY_TXID =
  "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f";
const INTERNAL_HASH =
  "0x1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100";
const FUNDING_SCRIPT = "0014751e76e8199196d454941c45d1b3a323f1433bd6";
const NESTED_SEGWIT_ADDRESS = "2NAUYAHhujozruyzpsFRP63mbrdaU5wnEpN";
const NESTED_SEGWIT_SCRIPT = "a914bcfeb728b584253d5f3f70bcb780e9ef218a68f487";

describe("BitcoinVaultDepositor", () => {
  it("matches the public depositor and input contracts", () => {
    expectTypeOf<BitcoinVaultDepositor>().toMatchTypeOf<
      VaultDepositor<BitcoinSubmitDepositInput>
    >();
    expectTypeOf<BitcoinSubmitDepositInput["amount"]>().toEqualTypeOf<bigint>();
    expectTypeOf<TxRef>().toEqualTypeOf<{ hash: Bytes32Hex; raw: string }>();
    expect(BITCOIN_NATIVE_ASSET).toBe("BTC");
  });

  it("derives stable regtest addresses and tx refs from account and txid bytes", async () => {
    const depositor = createDepositor();

    await expect(depositor.depositorAddress()).resolves.toBe(
      "bcrt1qw508d6qejxtdg4y5r3zarvary0c5xw7kygt080",
    );
    expect(depositor.depositAddress(ACCOUNT)).toMatch(/^bcrt1q[023456789acdefghjklmnpqrstuvwxyz]+$/);
    expect(depositor.depositAddress(ACCOUNT)).toBe(depositor.depositAddress(ACCOUNT));
    expect(depositor.depositAddress("clearnet:bitcoin:account-b")).not.toBe(
      depositor.depositAddress(ACCOUNT),
    );

    expect(depositor.txRefFromTxid(DISPLAY_TXID)).toEqual({
      raw: DISPLAY_TXID,
      hash: INTERNAL_HASH,
    });
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
        amount: 0n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_AMOUNT" });
    await expect(
      depositor.submitDeposit({
        asset: "DOGE",
        amount: 1n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INVALID_INPUT" });
    await expect(
      depositor.submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: 1n,
        destination: { account: ACCOUNT, ref: NON_ZERO_REF },
      }),
    ).rejects.toMatchObject({ code: "INVALID_REFERENCE" });
    await expect(
      depositor.submitDeposit(
        {
          asset: BITCOIN_NATIVE_ASSET,
          amount: 1n,
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
        asset: " btc ",
        amount: 120_000n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "INSUFFICIENT_FUNDS" });

    expect(rpc.sendRawTransaction).not.toHaveBeenCalled();
  });

  it("submits a signed native BTC deposit and returns a byte-order-safe tx ref", async () => {
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

    const ref = await depositor.submitDeposit(
      {
        asset: " btc ",
        amount: 50_000n,
        destination: { account: ACCOUNT, ref: ZERO_REF },
      },
      { onSubmitted },
    );

    expect(ref.raw).toMatch(/^[a-f0-9]{64}$/);
    expect(ref.hash).toBe(`0x${ref.raw.match(/../g)?.reverse().join("")}`);
    expect(rpc.listUnspent).toHaveBeenCalledExactlyOnceWith(1, [
      "bcrt1qw508d6qejxtdg4y5r3zarvary0c5xw7kygt080",
    ]);
    expect(rpc.estimateSmartFeeSatPerVByte).toHaveBeenCalledExactlyOnceWith(6, 5n);
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
    expect(rpc.sendRawTransaction).toHaveBeenCalledWith(expect.stringMatching(/^[a-f0-9]+$/));
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);
    expect(signer.getPublicKeyCompressed).toHaveBeenCalledTimes(1);
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
        amount: 99_500n,
        destination: { account: ACCOUNT },
      },
      wallet,
    );

    expect(prepared.psbtHex).toMatch(/^70736274ff[a-f0-9]+$/);
    expect(prepared.inputIndexesToSign).toEqual([0, 1]);
    expect(prepared.fundingAddress).toBe(wallet.address);
    expect(prepared.depositAddress).toMatch(/^bcrt1q/);
    expect(prepared.unsignedRef.raw).toMatch(/^[a-f0-9]{64}$/);
    expect(prepared.unsignedRef.hash).toBe(
      `0x${prepared.unsignedRef.raw.match(/../g)?.reverse().join("")}`,
    );
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
        amount: 50_000n,
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

    const ref = await depositor.submitSignedDepositPsbt(bytesToHex(tx.toPSBT()), {
      onSubmitted,
    });

    expect(ref).toEqual(prepared.unsignedRef);
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
    expect(rpc.sendRawTransaction).toHaveBeenCalledWith(expect.stringMatching(/^[a-f0-9]+$/));
    expect(onSubmitted).toHaveBeenCalledExactlyOnceWith(ref);
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
        amount: 50_000n,
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

    const ref = await depositor.submitSignedDepositPsbt(bytesToHex(tx.toPSBT()));

    expect(prepared.fundingAddress).toBe(NESTED_SEGWIT_ADDRESS);
    expect(ref.raw).toMatch(/^[a-f0-9]{64}$/);
    expect(ref).not.toEqual(prepared.unsignedRef);
    expect(rpc.listUnspent).toHaveBeenCalledExactlyOnceWith(1, [
      NESTED_SEGWIT_ADDRESS,
    ]);
    expect(rpc.sendRawTransaction).toHaveBeenCalledOnce();
  });

  it("rejects PSBT preparation when wallet address and public key do not match", async () => {
    const depositor = createDepositor({ signer: undefined });

    await expect(
      depositor.prepareDepositPsbt(
        {
          asset: BITCOIN_NATIVE_ASSET,
          amount: 1n,
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
    const alreadyKnownRef = await createDepositor({ rpc: alreadyKnownRpc }).submitDeposit({
      asset: BITCOIN_NATIVE_ASSET,
      amount: 50_000n,
      destination: { account: ACCOUNT },
    });
    expect(alreadyKnownRef.raw).toMatch(/^[a-f0-9]{64}$/);

    const missingRpc = createRpc({
      listUnspent: [utxo("04".repeat(32), 0, 100_000n, FUNDING_SCRIPT)],
      sendRawTransactionError: new BitcoinRpcError(-25, "bad-txns-inputs-missingorspent"),
      rawTransaction: { txid: "not-the-computed-txid", confirmations: 0 },
    });
    await expect(
      createDepositor({ rpc: missingRpc }).submitDeposit({
        asset: BITCOIN_NATIVE_ASSET,
        amount: 50_000n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({ code: "RPC_ERROR", txRef: expect.any(Object) });

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
        amount: 50_000n,
        destination: { account: ACCOUNT },
      }),
    ).rejects.toMatchObject({
      code: "RPC_ERROR",
      txRef: expect.any(Object),
      cause: sendError,
    });
  });

  it("verifies absent, pending, confirmed, and malformed tx refs", async () => {
    const ref = {
      raw: DISPLAY_TXID,
      hash: INTERNAL_HASH as Bytes32Hex,
    };
    const depositor = createDepositor({
      rpc: createRpc({ rawTransaction: null }),
    });
    await expect(depositor.verifyDeposit(ref, 1)).resolves.toBe("absent");

    const pending = createDepositor({
      rpc: createRpc({ rawTransaction: { txid: DISPLAY_TXID, confirmations: 0 } }),
    });
    await expect(pending.verifyDeposit(ref, 1)).resolves.toBe("pending");
    await expect(pending.verifyDeposit(ref, 0)).resolves.toBe("confirmed");

    const confirmed = createDepositor({
      rpc: createRpc({ rawTransaction: { txid: DISPLAY_TXID, confirmations: 2 } }),
    });
    await expect(confirmed.verifyDeposit(ref, 2)).resolves.toBe("confirmed");
    await expect(
      confirmed.verifyDeposit({ ...ref, hash: ZERO_REF }, 1),
    ).rejects.toMatchObject({ code: "INVALID_TX_REF" });
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
          amount: 50_000n,
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

  it("uses address-type-aware fee estimates", () => {
    expect(estimateDepositFeeSats(1, 1n, "p2wpkh")).toBe(153n);
    expect(estimateDepositFeeSats(2, 5n, "p2wpkh")).toBe(1105n);
    expect(estimateDepositFeeSats(1, 1n, "p2sh")).toBe(177n);
    expect(estimateDepositFeeSats(2, 5n, "p2sh")).toBe(1345n);
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
      amount: 50_000n,
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
