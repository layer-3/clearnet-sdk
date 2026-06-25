import type { Payment, SubmittableTransaction } from "xrpl";
import { Client, Wallet } from "xrpl";
import { afterAll, beforeAll, describe, expect, it } from "vitest";

import {
  XRPL_NATIVE_ASSET,
  XrplVaultDepositor,
} from "../../../src/index.js";
import type {
  Bytes32Hex,
  TxRef,
  XrplSigner,
} from "../../../src/index.js";

const XRPL_WS_URL = env("XRPL_WS_URL", "ws://127.0.0.1:6006");
const XRPL_ADMIN_RPC_URL = env("XRPL_ADMIN_RPC_URL", "http://127.0.0.1:5005");
const GENESIS_SEED = "snoPBrXtMeMyMHUVTgbuqAfg1SUTb";
const ACCOUNT = "0x00000000000000000000000000000000000000a1";
const REFERENCE =
  "0x1111111111111111111111111111111111111111111111111111111111111111" as Bytes32Hex;
const MEMO_TYPE = "796E65742D6163636F756E74";
const ASF_DEFAULT_RIPPLE = 8;

type FetchedPayment = Omit<Payment, "Amount"> & {
  Amount?: Payment["Amount"];
  DeliverMax?: Payment["Amount"];
};

describe("XrplVaultDepositor integration", () => {
  let client: Client;
  let admin: XrplAdmin;
  let master: Wallet;
  let vault: Wallet;
  let depositorWallet: Wallet;

  beforeAll(async () => {
    client = new Client(XRPL_WS_URL);
    await client.connect();
    admin = new XrplAdmin(XRPL_ADMIN_RPC_URL);
    master = Wallet.fromSeed(GENESIS_SEED);
    vault = Wallet.generate();
    depositorWallet = Wallet.generate();
    await fund(client, admin, master, vault.classicAddress, "1000000000");
    await fund(client, admin, master, depositorWallet.classicAddress, "1000000000");
  }, 60_000);

  afterAll(async () => {
    if (client?.isConnected()) {
      await client.disconnect();
    }
  });

  it("submits and verifies a native XRP deposit", async () => {
    const sdk = new XrplVaultDepositor({
      rpcUrl: XRPL_WS_URL,
      vaultAddress: vault.classicAddress,
      signer: signerFromWallet(depositorWallet),
    });

    const ref = await sdk.submitDeposit({
      asset: XRPL_NATIVE_ASSET,
      amount: 10_000_000n,
      destination: { account: ACCOUNT, ref: REFERENCE },
    });
    await admin.ledgerAccept();

    await expect(sdk.verifyDeposit(ref, 0)).resolves.toBe("confirmed");
    const payment = await fetchPayment(client, ref);
    expect(payment.Account).toBe(depositorWallet.classicAddress);
    expect(payment.Destination).toBe(vault.classicAddress);
    expect(paymentAmount(payment)).toBe("10000000");
    expect(payment.Memos).toEqual(expectedMemo(ACCOUNT, REFERENCE));
  }, 60_000);

  it("submits and verifies an issued-currency deposit", async () => {
    const issuer = Wallet.generate();
    await fund(client, admin, master, issuer.classicAddress, "1000000000");
    await enableDefaultRipple(client, admin, issuer);
    await trustSet(client, admin, depositorWallet, issuer.classicAddress, "USD", "1000");
    await trustSet(client, admin, vault, issuer.classicAddress, "USD", "1000");
    await issueCurrency(
      client,
      admin,
      issuer,
      depositorWallet.classicAddress,
      "USD",
      "100",
    );

    const sdk = new XrplVaultDepositor({
      rpcUrl: XRPL_WS_URL,
      vaultAddress: vault.classicAddress,
      signer: signerFromWallet(depositorWallet),
    });
    const ref = await sdk.submitDeposit({
      asset: `USD.${issuer.classicAddress}`,
      amount: "25",
      destination: { account: ACCOUNT },
    });
    await admin.ledgerAccept();

    await expect(sdk.verifyDeposit(ref, 0)).resolves.toBe("confirmed");
    const payment = await fetchPayment(client, ref);
    expect(paymentAmount(payment)).toEqual({
      currency: "USD",
      issuer: issuer.classicAddress,
      value: "25",
    });
    expect(payment.Memos).toEqual(expectedMemo(ACCOUNT));
  }, 90_000);
});

function signerFromWallet(wallet: Wallet): XrplSigner {
  return {
    classicAddress: wallet.classicAddress,
    sign: async (payment) => {
      const signed = wallet.sign(payment as SubmittableTransaction);
      return { txBlob: signed.tx_blob, hash: signed.hash };
    },
  };
}

async function fund(
  client: Client,
  admin: XrplAdmin,
  source: Wallet,
  destination: string,
  amountDrops: string,
): Promise<void> {
  await submitAndAccept(client, admin, source, {
    TransactionType: "Payment",
    Account: source.classicAddress,
    Destination: destination,
    Amount: amountDrops,
  });
}

async function enableDefaultRipple(
  client: Client,
  admin: XrplAdmin,
  issuer: Wallet,
): Promise<void> {
  await submitAndAccept(client, admin, issuer, {
    TransactionType: "AccountSet",
    Account: issuer.classicAddress,
    SetFlag: ASF_DEFAULT_RIPPLE,
  });
}

async function trustSet(
  client: Client,
  admin: XrplAdmin,
  wallet: Wallet,
  issuer: string,
  currency: string,
  value: string,
): Promise<void> {
  await submitAndAccept(client, admin, wallet, {
    TransactionType: "TrustSet",
    Account: wallet.classicAddress,
    LimitAmount: { currency, issuer, value },
  });
}

async function issueCurrency(
  client: Client,
  admin: XrplAdmin,
  issuer: Wallet,
  destination: string,
  currency: string,
  value: string,
): Promise<void> {
  await submitAndAccept(client, admin, issuer, {
    TransactionType: "Payment",
    Account: issuer.classicAddress,
    Destination: destination,
    Amount: { currency, issuer: issuer.classicAddress, value },
  });
}

async function submitAndAccept(
  client: Client,
  admin: XrplAdmin,
  wallet: Wallet,
  tx: SubmittableTransaction,
): Promise<string> {
  const prepared = await client.autofill(tx);
  const signed = wallet.sign(prepared);
  const result = await client.submit(signed.tx_blob, { autofill: false });
  if (
    result.result.engine_result !== "tesSUCCESS" &&
    result.result.engine_result !== "terQUEUED"
  ) {
    throw new Error(
      `XRPL setup tx rejected: ${result.result.engine_result} ${result.result.engine_result_message}`,
    );
  }
  await admin.ledgerAccept();
  return signed.hash;
}

async function fetchPayment(client: Client, ref: TxRef): Promise<FetchedPayment> {
  const response = await client.request({
    command: "tx",
    transaction: ref.raw,
  });
  const result = response.result as unknown as {
    tx_json?: FetchedPayment;
    tx?: FetchedPayment;
  };
  return (result.tx_json ?? result.tx) as FetchedPayment;
}

function paymentAmount(payment: FetchedPayment): Payment["Amount"] | undefined {
  return payment.Amount ?? payment.DeliverMax;
}

function expectedMemo(account: string, ref?: Bytes32Hex): Payment["Memos"] {
  return [
    {
      Memo: {
        MemoType: MEMO_TYPE,
        MemoData: `${account.replace(/^0x/, "")}${(ref ?? zeroRef()).slice(2)}`.toUpperCase(),
      },
    },
  ];
}

function zeroRef(): Bytes32Hex {
  return `0x${"00".repeat(32)}`;
}

class XrplAdmin {
  constructor(private readonly rpcUrl: string) {}

  async ledgerAccept(): Promise<void> {
    const response = await fetch(this.rpcUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ method: "ledger_accept", params: [] }),
    });
    if (!response.ok) {
      throw new Error(`ledger_accept failed with HTTP ${response.status}`);
    }
    const body = (await response.json()) as { result?: { status?: string } };
    if (body.result?.status !== "success") {
      throw new Error(`ledger_accept failed: ${JSON.stringify(body)}`);
    }
  }
}

function env(key: string, fallback: string): string {
  const value = process.env[key];
  return value === undefined || value === "" ? fallback : value;
}
