import { Wallet, hashes, type Payment } from "xrpl";
import { describe, expect, it } from "vitest";

import {
  createLocalXrplSigner,
  LOCAL_XRPL_GENESIS_SEED,
} from "../../../examples/xrpl-deposit/src/local-signer.js";

describe("XRPL deposit demo local signer", () => {
  it("creates a signer from a seed and signs a prepared payment", async () => {
    const signer = createLocalXrplSigner(LOCAL_XRPL_GENESIS_SEED);
    const wallet = Wallet.fromSeed(LOCAL_XRPL_GENESIS_SEED);
    const payment: Payment = {
      TransactionType: "Payment",
      Account: signer.classicAddress,
      Destination: wallet.classicAddress,
      Amount: "1000",
      Fee: "10",
      Sequence: 1,
      LastLedgerSequence: 10,
      NetworkID: 31337,
    };

    const signed = await signer.sign(payment);

    expect(signer.classicAddress).toBe(wallet.classicAddress);
    expect(signed.hash).toBe(hashes.hashSignedTx(signed.txBlob));
    expect(signed.hash).toMatch(/^[A-F0-9]{64}$/);
  });
});
