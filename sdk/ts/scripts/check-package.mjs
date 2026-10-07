import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdtempSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const root = dirname(dirname(fileURLToPath(import.meta.url)));
const staging = mkdtempSync(join(tmpdir(), "clearnet-sdk-package-"));
const npm = process.platform === "win32" ? "npm.cmd" : "npm";
const run = (command, args, cwd, capture = false) => execFileSync(command, args, {
  cwd, stdio: capture ? ["ignore", "pipe", "inherit"] : "inherit",
  encoding: "utf8", maxBuffer: 32 * 1024 * 1024,
});

try {
  const [packed] = JSON.parse(run(npm, ["pack", "--json", "--pack-destination", staging], root, true));
  assert(packed.bundled.includes("@solana/web3.js"), "Missing patched Solana bundle");
  const manifest = JSON.parse(readFileSync(join(root, "package.json"), "utf8"));
  assert.equal(packed.version, manifest.version);

  const consumer = join(staging, "consumer");
  mkdirSync(consumer);
  writeFileSync(join(consumer, "package.json"), JSON.stringify({
    name: "clearnet-sdk-package-check", private: true, type: "module",
  }));
  // No SDK overrides, workspace links, or install hooks in this consumer.
  run(npm, ["install", join(staging, packed.filename), "--ignore-scripts", "--no-audit", "--no-fund"], consumer);
  run(npm, ["ls", "--all", "--omit=dev"], consumer, true);
  run(npm, ["audit", "--omit=dev", "--audit-level=moderate"], consumer);

  writeFileSync(join(consumer, "smoke.mjs"), `
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import * as sdk from "@yellow-org/clearnet-sdk";
const fromSdk = createRequire(import.meta.resolve("@yellow-org/clearnet-sdk"));
const web3 = sdk.solana;
assert.equal(createRequire(fromSdk.resolve("@solana/web3.js"))("jayson/package.json").version, "5.0.0");
for (const name of ["EvmVaultDepositor", "BitcoinVaultDepositor", "SolanaVaultDepositor", "XrplVaultDepositor"]) {
  assert.equal(typeof sdk[name], "function", name);
}
assert.deepEqual(sdk.splitNonce(sdk.composeNonce(7n, 3n)), { key: 7n, sequence: 3n });
const keypair = web3.Keypair.generate();
const depositor = new sdk.SolanaVaultDepositor({
  rpcUrl: "http://127.0.0.1:8899",
  signer: { publicKey: keypair.publicKey.toBase58(), signAndSend: async () => { throw new Error("Not used"); } },
});
const transaction = await depositor.prepareDeposit({
  asset: "", amount: "0.1", destination: { account: "00000000000000000000000000000000000000a1" },
});
assert(transaction instanceof web3.Transaction);
transaction.recentBlockhash = keypair.publicKey.toBase58();
transaction.partialSign(keypair);
const bytes = transaction.serialize();
assert(bytes instanceof Uint8Array && bytes.length > 0);
const connection = new web3.Connection("http://127.0.0.1:8899", {
  fetch: async (_, init) => {
    const request = JSON.parse(init.body);
    const values = {
      getBalance: 42,
      getLatestBlockhash: { blockhash: keypair.publicKey.toBase58(), lastValidBlockHeight: 100 },
      simulateTransaction: { err: null, logs: [], unitsConsumed: 1 },
    };
    assert(request.method in values, request.method);
    return new Response(JSON.stringify({ jsonrpc: "2.0", id: request.id, result: { context: { slot: 1 }, value: values[request.method] } }));
  },
});
assert.equal(await connection.getBalance(keypair.publicKey), 42);
assert.equal((await connection.simulateTransaction(transaction, [keypair])).value.err, null);
console.log("Packed SDK: public exports, Solana preparation/signing/simulation, and JSON-RPC passed");
`);
  run(process.execPath, ["smoke.mjs"], consumer);
  writeFileSync(join(consumer, "smoke.ts"), `
import { solana, type SubmitDepositResult, type VaultDepositor, type DepositStatus, type SolanaSigner } from "@yellow-org/clearnet-sdk";
const connection: solana.Connection = new solana.Connection("http://127.0.0.1:8899");
void connection;
const result: SubmitDepositResult = { txHash: "tx", depositId: "deposit" };
async function status(depositor: VaultDepositor): Promise<DepositStatus> {
  return depositor.chainDepositStatus(result.txHash, result.depositId, 2);
}
const signer: SolanaSigner = {
  publicKey: "key",
  async signAndSend(transaction) {
    const bytes: Uint8Array = transaction.serialize();
    return String(bytes.length);
  },
};
void [status, signer];
`);
  run(process.execPath, [join(root, "node_modules/typescript/bin/tsc"), "--noEmit", "--strict",
    "--skipLibCheck", "--target", "ES2022", "--module", "NodeNext", "--moduleResolution", "NodeNext", "smoke.ts"], consumer);
  console.log(`Verified ${packed.name}@${packed.version} (${packed.size} packed bytes)`);
} finally {
  rmSync(staging, { recursive: true, force: true });
}
