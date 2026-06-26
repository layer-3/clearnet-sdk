import { Wallet, hashes, type SubmittableTransaction } from "xrpl";

import type {
  XrplPreparedPayment,
  XrplSignedTransaction,
  XrplSigner,
} from "@yellow-org/clearnet-sdk";

export const LOCAL_XRPL_GENESIS_SEED = "snoPBrXtMeMyMHUVTgbuqAfg1SUTb";

export class LocalXrplSigner implements XrplSigner {
  constructor(private readonly wallet: Wallet) {}

  get classicAddress(): string {
    return this.wallet.classicAddress;
  }

  get seed(): string | undefined {
    return this.wallet.seed;
  }

  async sign(payment: XrplPreparedPayment): Promise<XrplSignedTransaction> {
    const signed = this.wallet.sign(payment as SubmittableTransaction);
    return {
      txBlob: signed.tx_blob,
      hash: hashes.hashSignedTx(signed.tx_blob),
    };
  }
}

export function createLocalXrplSigner(seed?: string): LocalXrplSigner {
  const normalized = seed?.trim();
  const wallet =
    normalized === undefined || normalized === ""
      ? Wallet.generate()
      : Wallet.fromSeed(normalized);
  return new LocalXrplSigner(wallet);
}
