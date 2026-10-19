import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { spawn, execFile } from "node:child_process";
import { promisify } from "node:util";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { generatePrivateKey, privateKeyToAccount } from "viem/accounts";
import { createPublicClient, http } from "viem";
import { GatewayClient, networkFingerprint } from "../../src/gateway/index.js";

const exec = promisify(execFile);
const repo = fileURLToPath(new URL("../../../..", import.meta.url));
// Generated LOCAL TEST KEY from clearbox startup; never reuse for real funds.
let seed: ReturnType<typeof privateKeyToAccount>;
let directory: string;
let process: ReturnType<typeof spawn>;
let client: GatewayClient;

async function settled(id: string) {
  for (let i = 0; i < 100; i++) {
    const op = await client.operation(id);
    if (op.status === "sealed") return op;
    await new Promise((resolve) => setTimeout(resolve, 20));
  }
  throw new Error("clearbox did not seal the transfer");
}

beforeAll(async () => {
  directory = await mkdtemp(join(tmpdir(), "sdk-clearbox-"));
  const binary = join(directory, "clearbox");
  await exec("go", ["build", "-o", binary, "./cmd/clearbox"], { cwd: repo });
  process = spawn(binary, ["-db", join(directory, "clearbox.db"), "-listen", "127.0.0.1:0", "-public-url", "https://sandbox.example", "-rate-limit", "1000", "-seal-delay", "50ms", "-delivery-delay", "50ms"], { cwd: repo, stdio: ["ignore", "pipe", "pipe"] });
  const url = await new Promise<string>((resolve, reject) => {
    const timer = setTimeout(() => reject(new Error("clearbox startup timed out")), 15_000);
    let output = "";
    process.on("error", reject);
    process.on("exit", (code) => { clearTimeout(timer); reject(new Error(`clearbox exited ${code}: ${output}`)); });
    process.stderr!.on("data", (data) => { output += String(data); });
    process.stdout!.on("data", (data) => {
      output += String(data);
      const match = output.match(/Clearbox listening at (http:\/\/[^\s]+)/);
      const key = output.match(/Private key: ([0-9a-f]{64})/);
      if (match && key) {
        seed = privateKeyToAccount(`0x${key[1]!}`);
        clearTimeout(timer);
        resolve(match[1]!);
      }
    });
  });
  client = new GatewayClient({ baseUrl: url, networkId: "YDEV" });
}, 120_000);

afterAll(async () => {
  if (process && process.exitCode === null && process.signalCode === null) {
    const exited = new Promise<void>((resolve) => process.once("exit", () => resolve()));
    process.kill("SIGTERM");
    await exited;
  }
  if (directory) await rm(directory, { recursive: true, force: true });
});

describe("public Go clearbox", () => {
  it("supports actual wallet RPC reads through viem", async () => {
    const walletRPC = createPublicClient({ transport: http(`${client.baseUrl}/rpc`) });
    expect(await walletRPC.getChainId()).toBe(1497646422);
    expect(await walletRPC.getBalance({ address: seed.address })).toBe(1_000_000n * 10n ** 18n);
    expect(await walletRPC.getTransactionCount({ address: seed.address })).toBe(0);
    // Clearnet block RPC requires an account; there is no global EVM chain.
    const response = await fetch(`${client.baseUrl}/rpc`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "eth_getBlockByNumber", params: [seed.address, "latest", false] }),
    });
    expect(await response.json()).toEqual({ jsonrpc: "2.0", id: 1, result: null });
  });

  it("uses genuine TS signatures, operation IDs, metadata and bidirectional transfers", async () => {
    const info = await client.info();
    expect(info.networkId).toBe("YDEV");
    expect(info.chainId).toBe(1497646422);
    expect(info.rpcUrl).toBe("https://sandbox.example/rpc");
    expect(networkFingerprint(info)).toMatch(/^[0-9a-f]{64}$/);
    const bob = privateKeyToAccount(generatePrivateKey());
    expect((await client.balance(bob.address)).balances.every((b) => b.clearnetBalance === "0")).toBe(true);
    const intent = await client.prepareTransfer({ from: seed.address, recipient: bob.address, assets: [{ asset: "YELLOW", amount: "25" }], maxFee: "1" });
    const signed = await client.signTransfer(intent, (data) => seed.signTypedData(data));
    const result = await client.submitTransfer(signed);
    expect(result.operationId).toBe(signed.operationId);
    expect(result.status).toBe("pending");
    const op = await settled(signed.operationId);
    expect(op.receipt?.blockHash).toMatch(/^0x[0-9a-f]{64}$/);
    expect(op.receipt?.entryHash).toMatch(/^0x[0-9a-f]{64}$/);
    const replay = await client.submitTransfer(signed);
    expect(replay.receipt).toEqual(op.receipt);
    await new Promise((resolve) => setTimeout(resolve, 100));
    const funded = await client.balance(bob.address);
    expect(funded.balances.find((b) => b.asset === "USDC")?.clearnetBalance).toBe("0");
    expect(funded.balances.find((b) => b.asset === "YELLOW")?.clearnetBalance).toBe("25000000000000000000");
    const incoming = await client.transfers(bob.address);
    expect(incoming.items[0]?.direction).toBe("in");
    expect(incoming.items[0]?.delivery).toBe("not_verified");
    const backIntent = await client.prepareTransfer({ from: bob.address, recipient: seed.address, assets: [{ asset: "YELLOW", amount: "2" }], maxFee: "0.1" });
    const back = await client.signTransfer(backIntent, (data) => bob.signTypedData(data));
    await client.submitTransfer(back);
    await settled(back.operationId);
    expect((await client.balance(bob.address)).nonce).toBe(1);
    expect((await client.transfers(bob.address)).items).toHaveLength(2);
  });

});
