import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

// Shared with the Go tests; the repository keeps a single copy.
const vectorsPath = fileURLToPath(
  new URL("../../../../testdata/deposit_id_vectors.json", import.meta.url),
);

/** Returns one chain's section of the shared deposit-ID golden vectors. */
export function loadDepositIdVectors<V>(chain: "evm" | "btc" | "sol" | "xrpl"): V[] {
  return JSON.parse(readFileSync(vectorsPath, "utf8"))[chain].vectors;
}
