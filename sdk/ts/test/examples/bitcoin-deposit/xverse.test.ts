import { describe, expect, it, vi } from "vitest";

import {
  configureXverseRegtestNetwork,
  connectXverseWallet,
  signPsbtWithXverse,
} from "../../../examples/bitcoin-deposit/src/xverse.js";

const PAYMENT_ADDRESS = "2NAUYAHhujozruyzpsFRP63mbrdaU5wnEpN";
const PAYMENT_PUBLIC_KEY =
  "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798";

describe("bitcoin demo Xverse adapter", () => {
  it("adds the configured regtest network with switch requested", async () => {
    const request = vi.fn(async () => ({ status: "success", result: null }));

    await configureXverseRegtestNetwork(
      {
        name: "clearnet-regtest",
        rpcUrl: "http://127.0.0.1:5174/btc-rpc",
      },
      request,
    );

    expect(request).toHaveBeenCalledExactlyOnceWith("wallet_addNetwork", {
      chain: "bitcoin",
      type: "Regtest",
      name: "clearnet-regtest",
      rpcUrl: "http://127.0.0.1:5174/btc-rpc",
      switch: true,
    });
  });

  it("connects the Xverse payment address through getAccounts", async () => {
    const request = vi.fn(async () => ({
      status: "success",
      result: [
        {
          purpose: "ordinals",
          addressType: "p2tr",
          address: "bcrt1pordinals",
          publicKey: "03".repeat(32),
          walletType: "software",
        },
        {
          purpose: "payment",
          addressType: "p2sh",
          address: PAYMENT_ADDRESS,
          publicKey: PAYMENT_PUBLIC_KEY,
          walletType: "software",
        },
      ],
    }));

    const wallet = await connectXverseWallet(request);

    expect(request).toHaveBeenCalledExactlyOnceWith("getAccounts", {
      purposes: ["payment"],
      message: "Connect Bitcoin deposit demo",
    });
    expect(wallet).toMatchObject({
      address: PAYMENT_ADDRESS,
      addressType: "p2sh",
      publicKey: PAYMENT_PUBLIC_KEY,
      walletType: "software",
    });
  });

  it("falls back to getAddresses when getAccounts is unavailable", async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce({
        status: "error",
        error: {
          code: -32601,
          message: "Method not found",
        },
      })
      .mockResolvedValueOnce({
        status: "success",
        result: {
          id: "account-1",
          walletType: "software",
          addresses: [
            {
              purpose: "payment",
              addressType: "p2sh",
              address: PAYMENT_ADDRESS,
              publicKey: PAYMENT_PUBLIC_KEY,
              walletType: "software",
            },
          ],
          network: {
            bitcoin: { name: "Regtest" },
          },
        },
      });

    const wallet = await connectXverseWallet(request);

    expect(request).toHaveBeenNthCalledWith(1, "getAccounts", {
      purposes: ["payment"],
      message: "Connect Bitcoin deposit demo",
    });
    expect(request).toHaveBeenNthCalledWith(2, "getAddresses", {
      purposes: ["payment"],
      message: "Connect Bitcoin deposit demo",
    });
    expect(wallet).toMatchObject({
      address: PAYMENT_ADDRESS,
      addressType: "p2sh",
      publicKey: PAYMENT_PUBLIC_KEY,
      walletType: "software",
    });
  });

  it("fails fast when getAccounts is access-denied", async () => {
    const request = vi
      .fn()
      .mockResolvedValueOnce({
        status: "error",
        error: {
          code: 4001,
          message: "Access denied",
        },
      });

    await expect(connectXverseWallet(request)).rejects.toThrow(
      "Xverse getAccounts failed: Access denied",
    );

    expect(request).toHaveBeenCalledExactlyOnceWith("getAccounts", {
      purposes: ["payment"],
      message: "Connect Bitcoin deposit demo",
    });
  });

  it("signs PSBT hex through Xverse's base64 API", async () => {
    const signedPsbtHex = "70736274ff010203";
    const request = vi.fn(async () => ({
      status: "success",
      result: {
        psbt: hexToBase64(signedPsbtHex),
      },
    }));

    const signed = await signPsbtWithXverse(
      {
        psbtHex: "70736274ff",
        inputIndexesToSign: [0, 1],
        address: PAYMENT_ADDRESS,
      },
      request,
    );

    expect(request).toHaveBeenCalledExactlyOnceWith("signPsbt", {
      psbt: hexToBase64("70736274ff"),
      signInputs: {
        [PAYMENT_ADDRESS]: [0, 1],
      },
      broadcast: false,
    });
    expect(signed).toBe(signedPsbtHex);
  });
});

function hexToBase64(hex: string): string {
  return Buffer.from(hex, "hex").toString("base64");
}
