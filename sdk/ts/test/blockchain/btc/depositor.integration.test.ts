import {
  pubECDSA,
  randomPrivateKeyBytes,
  signECDSA,
} from "@scure/btc-signer/utils.js";
import { beforeAll, describe, expect, it } from "vitest";

import {
  BITCOIN_NATIVE_ASSET,
  BitcoinCoreRpcClient,
  BitcoinVaultDepositor,
} from "../../../src/index.js";
import type { BitcoinSigner } from "../../../src/index.js";

const BTC_RPC_URL = env("BTC_RPC_URL", "http://127.0.0.1:18443");
const BTC_RPC_USER = env("BTC_RPC_USER", "sdk");
const BTC_RPC_PASS = env("BTC_RPC_PASS", "sdk");
const BTC_RPC_WALLET = env("BTC_RPC_WALLET", "sdk");
const ACCOUNT = `yellow://ynet/user/btc-ts-${Date.now()}`;

describe("BitcoinVaultDepositor regtest integration", () => {
  const admin = new RawBitcoinRpc(BTC_RPC_URL, BTC_RPC_USER, BTC_RPC_PASS);
  const wallet = new RawBitcoinRpc(
    `${BTC_RPC_URL}/wallet/${encodeURIComponent(BTC_RPC_WALLET)}`,
    BTC_RPC_USER,
    BTC_RPC_PASS,
  );

  beforeAll(async () => {
    await ensureWallet(admin, BTC_RPC_WALLET);
    const miningAddress = await wallet.call<string>("getnewaddress", ["", "bech32"]);
    await admin.call("generatetoaddress", [101, miningAddress]);
  }, 120_000);

  it("submits and verifies a native BTC deposit", async () => {
    const vaultSigners = [localSigner(), localSigner(), localSigner()];
    const depositorSigner = localSigner();
    const rpc = new BitcoinCoreRpcClient({
      url: BTC_RPC_URL,
      username: BTC_RPC_USER,
      password: BTC_RPC_PASS,
      wallet: BTC_RPC_WALLET,
    });
    const depositor = new BitcoinVaultDepositor({
      network: "regtest",
      rpc,
      signer: depositorSigner,
      vaultPubkeys: await Promise.all(
        vaultSigners.map((signer) => signer.getPublicKeyCompressed()),
      ),
      threshold: 2,
      minFundingConfirmations: 1,
      feeTargetBlocks: 6,
      fallbackFeeRateSatPerVByte: 5,
    });
    const fundingAddress = await depositor.depositorAddress();
    const depositAddress = depositor.depositAddress(ACCOUNT);
    await wallet.call("importaddress", [fundingAddress, "", false]);
    await wallet.call("importaddress", [depositAddress, "", false]);
    await wallet.call("sendtoaddress", [fundingAddress, 1.0]);
    const miningAddress = await wallet.call<string>("getnewaddress", ["", "bech32"]);
    await admin.call("generatetoaddress", [1, miningAddress]);

    const txID = await depositor.submitDeposit({
      asset: BITCOIN_NATIVE_ASSET,
      amount: "0.2",
      destination: { account: ACCOUNT },
    });

    expect(txID).toMatch(/^[a-f0-9]{64}$/);
    await expect(depositor.verifyDeposit(txID, 1)).resolves.toBe("pending");
    await admin.call("generatetoaddress", [1, miningAddress]);
    await expect(depositor.verifyDeposit(txID, 1)).resolves.toBe("confirmed");
  }, 120_000);
});

class LocalBitcoinSigner implements BitcoinSigner {
  readonly algorithm = "secp256k1";

  constructor(private readonly privateKey: Uint8Array) {}

  getPublicKeyCompressed(): Uint8Array {
    return pubECDSA(this.privateKey, true);
  }

  signDigest32(digest: Uint8Array): Uint8Array {
    return signECDSA(digest, this.privateKey);
  }
}

class RawBitcoinRpc {
  constructor(
    private readonly url: string,
    private readonly username: string,
    private readonly password: string,
  ) {}

  async call<T = unknown>(method: string, params: readonly unknown[] = []): Promise<T> {
    const response = await fetch(this.url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Basic ${Buffer.from(
          `${this.username}:${this.password}`,
        ).toString("base64")}`,
      },
      body: JSON.stringify({
        jsonrpc: "1.0",
        id: "sdk",
        method,
        params,
      }),
    });
    const envelope = (await response.json()) as {
      result?: T;
      error?: { code: number; message: string } | null;
    };
    if (envelope.error !== null && envelope.error !== undefined) {
      throw new Error(`bitcoind rpc error ${envelope.error.code}: ${envelope.error.message}`);
    }
    return envelope.result as T;
  }
}

async function ensureWallet(admin: RawBitcoinRpc, wallet: string): Promise<void> {
  try {
    await admin.call("createwallet", [wallet, false, false, "", false, false]);
  } catch (error) {
    const message = error instanceof Error ? error.message.toLowerCase() : "";
    if (message.includes("already loaded")) {
      return;
    }
    if (message.includes("already exists")) {
      try {
        await admin.call("loadwallet", [wallet]);
      } catch (loadError) {
        const loadMessage =
          loadError instanceof Error ? loadError.message.toLowerCase() : "";
        if (!loadMessage.includes("already loaded")) {
          throw loadError;
        }
      }
      return;
    }
    throw error;
  }
}

function localSigner(): LocalBitcoinSigner {
  return new LocalBitcoinSigner(randomPrivateKeyBytes());
}

function env(key: string, fallback: string): string {
  return process.env[key] ?? fallback;
}
