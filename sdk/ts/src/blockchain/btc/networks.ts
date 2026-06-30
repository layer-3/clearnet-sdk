import { NETWORK, TEST_NETWORK } from "@scure/btc-signer";

import { ClearnetSdkError } from "../../core/errors.js";
import type { BitcoinNetwork } from "./types.js";

export type BitcoinNetworkParams = typeof NETWORK;

export function networkParams(network: BitcoinNetwork): BitcoinNetworkParams {
  switch (network) {
    case "mainnet":
      return NETWORK;
    case "testnet":
    case "signet":
      return TEST_NETWORK;
    case "regtest":
      return { ...TEST_NETWORK, bech32: "bcrt" };
    default:
      throw new ClearnetSdkError("CHAIN_MISMATCH", "unsupported Bitcoin network");
  }
}
