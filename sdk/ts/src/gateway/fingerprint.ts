import { sha256, stringToBytes } from "viem";
import type { NetworkInfo } from "./types.js";

const SYMBOL = /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,47}$/;

/**
 * The reviewed-identity fingerprint the CLI pins for a network profile
 * (internal/yellow/network.go `reviewed`). The JSON field order and
 * normalization must stay identical so both tools print the same value.
 */
export function networkFingerprint(info: NetworkInfo): string {
  const assets = info.assets.map((a) => {
    if (!SYMBOL.test(a.symbol)) throw new Error("Gateway catalogue has an invalid asset symbol.");
    return { symbol: a.symbol, chainId: a.chainId, address: (a.address ?? "").toLowerCase(), decimals: a.decimals, native: a.native ?? false };
  });
  assets.sort((x, y) => (x.symbol < y.symbol ? -1 : x.symbol > y.symbol ? 1 : 0));
  const identity = {
    networkId: info.networkId,
    chainId: info.chainId,
    router: info.virtualRouter.toLowerCase(),
    assets: assets.length ? assets : null,
  };
  return sha256(stringToBytes(JSON.stringify(identity))).slice(2);
}
