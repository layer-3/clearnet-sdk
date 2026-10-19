// End-to-end smoke test and screenshot tour against the demo network.
//
//   pnpm build && pnpm start -p 3200 &
//   BASE_URL=http://localhost:3200 pnpm shots
//
// A test EIP-1193 wallet is injected; it signs with a throwaway key in Node.
// Screenshots land in ./screenshots (git-ignored).

import { chromium } from "@playwright/test";
import { mkdirSync } from "node:fs";
import assert from "node:assert/strict";
import { privateKeyToAccount } from "viem/accounts";
import { GatewayClient } from "@yellow-org/clearnet-sdk/gateway";

const BASE = process.env.BASE_URL ?? "http://localhost:3200";
const OUT = new URL("../screenshots/", import.meta.url).pathname;
mkdirSync(OUT, { recursive: true });

// Generated LOCAL TEST KEY from clearbox initialization. Never use a real wallet key.
const rawKey = process.env.CLEARBOX_PRIVATE_KEY?.replace(/^0x/, "");
if (!rawKey || !/^[0-9a-fA-F]{64}$/.test(rawKey)) throw new Error("Set CLEARBOX_PRIVATE_KEY to the generated local clearbox key printed at initialization; never use a real wallet key.");
const account = privateKeyToAccount(`0x${rawKey}`);
const gateway = new GatewayClient({ baseUrl: process.env.CLEARBOX_URL ?? "http://127.0.0.1:8080", networkId: "YDEV" });
const RECIPIENT = "0x9a3f0b6d8c21e44f7b0a5c3e1d2f6a8b4c7e9d10";

function walletScript(address) {
  return `(() => {
    const listeners = {};
    let chainId = sessionStorage.getItem("e2e-chain") || "0x1";
    let connected = sessionStorage.getItem("e2e-connected") === "1";
    const setConnected = (v) => { connected = v; sessionStorage.setItem("e2e-connected", v ? "1" : "0"); };
    const emit = (e, v) => (listeners[e] || []).forEach((f) => f(v));
    const provider = {
      isMetaMask: true,
      on: (e, f) => ((listeners[e] ||= []).push(f), provider),
      removeListener: (e, f) => ((listeners[e] = (listeners[e] || []).filter((x) => x !== f)), provider),
      async request({ method, params }) {
        switch (method) {
          case "eth_requestAccounts": setConnected(true); emit("connect", { chainId }); return ["${address}"];
          case "eth_accounts": return connected ? ["${address}"] : [];
          case "eth_chainId": return chainId;
          case "net_version": return String(parseInt(chainId, 16));
          case "wallet_requestPermissions": setConnected(true); return [{ parentCapability: "eth_accounts" }];
          case "wallet_getPermissions": return connected ? [{ parentCapability: "eth_accounts" }] : [];
          case "wallet_revokePermissions": setConnected(false); return null;
          case "wallet_switchEthereumChain": chainId = params[0].chainId; sessionStorage.setItem("e2e-chain", chainId); emit("chainChanged", chainId); return null;
          case "wallet_addEthereumChain": chainId = params[0].chainId; sessionStorage.setItem("e2e-chain", chainId); emit("chainChanged", chainId); return null;
          case "eth_signTypedData_v4": {
            const data = JSON.parse(params[1]);
            if (Number(data.domain.chainId) !== parseInt(chainId, 16)) throw Object.assign(new Error("Provided chainId does not match the active chain"), { code: -32603 });
            return window.__walletSign(params[1]);
          }
          default: throw Object.assign(new Error("Unsupported " + method), { code: 4200 });
        }
      },
    };
    window.ethereum = provider;
    const info = { uuid: "e2e-wallet", name: "Test Wallet", icon: "data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg'/>", rdns: "io.metamask" };
    const announce = () => window.dispatchEvent(new CustomEvent("eip6963:announceProvider", { detail: Object.freeze({ info, provider }) }));
    window.addEventListener("eip6963:requestProvider", announce);
    announce();
  })();`;
}

async function newPage(browser, { width, height, dark, mobile }) {
  const context = await browser.newContext({
    viewport: { width, height },
    deviceScaleFactor: 2,
    colorScheme: dark ? "dark" : "light",
    isMobile: mobile,
    hasTouch: mobile,
  });
  await context.exposeFunction("__walletSign", async (json) => {
    const data = JSON.parse(json);
    const { EIP712Domain, ...types } = data.types;
    void EIP712Domain;
    return account.signTypedData({ domain: data.domain, types, primaryType: data.primaryType, message: data.message });
  });
  await context.addInitScript(walletScript(account.address));
  const page = await context.newPage();
  page.on("pageerror", (e) => console.error("  page error:", e.message));
  page.on("console", (m) => m.type() === "error" && console.error("  console:", m.text().slice(0, 400)));
  return { context, page };
}

async function shot(page, name) {
  await page.waitForTimeout(450);
  await page.screenshot({ path: `${OUT}${name}.png`, fullPage: true });
  console.log("  saved", name);
}

async function connect(page) {
  await page.goto(BASE + "/");
  const rpc = await page.request.post(BASE + "/api/gw/local/rpc", { data: { jsonrpc: "2.0", id: 1, method: "eth_chainId", params: [] } });
  assert.equal((await rpc.json()).result, "0x59444556", "wallet RPC must reach the Go clearbox through the web relay");
  await page.getByRole("button", { name: /connect/i }).first().click();
  await page.getByText(/total balance|balance/i).first().waitFor({ timeout: 15_000 });
}

async function sendFlow(page, prefix) {
  const before = await gateway.balance(account.address);
  const recipientBefore = await gateway.balance(RECIPIENT);
  await page.goto(BASE + "/send?asset=YELLOW");
  await page.getByLabel("To", { exact: true }).fill(RECIPIENT);
  await page.getByLabel("YELLOW amount").fill("25");
  await shot(page, `${prefix}-send-compose`);
  await page.getByRole("button", { name: "Review transfer" }).click();
  await page.getByRole("heading", { name: "Review" }).waitFor();
  await shot(page, `${prefix}-send-review`);
  const submitted = page.waitForRequest((r) => r.method() === "POST" && r.url().endsWith("/v1/transfers"));
  await page.getByRole("button", { name: "Sign and send" }).click();
  const body = (await submitted).postDataJSON();
  await page.getByRole("heading", { name: /^Sent$/ }).waitFor({ timeout: 20_000 });
  await page.waitForTimeout(900);
  const after = await gateway.balance(account.address);
  let recipientAfter;
  const units = (b, asset) => BigInt(b.balances.find((leg) => leg.asset === asset)?.clearnetBalance ?? "0");
  for (let i = 0; i < 30; i++) {
    recipientAfter = await gateway.balance(RECIPIENT);
    if (body.assets.every((leg) => units(recipientAfter, leg.asset) - units(recipientBefore, leg.asset) === BigInt(leg.amount))) break;
    await page.waitForTimeout(100);
  }
  assert.equal(after.nonce, before.nonce + 1, "browser send must consume exactly one shared nonce");
  for (const leg of body.assets) {
    assert.equal(units(recipientAfter, leg.asset) - units(recipientBefore, leg.asset), BigInt(leg.amount), "recipient must receive the signed amount");
    if (leg.asset !== "YELLOW") assert.equal(units(before, leg.asset) - units(after, leg.asset), BigInt(leg.amount));
  }
  const principal = BigInt(body.assets.find((leg) => leg.asset === "YELLOW")?.amount ?? "0");
  const fee = units(before, "YELLOW") - units(after, "YELLOW") - principal;
  assert(fee > 0n && fee <= BigInt(body.maxFee), "shared server must enforce the signed fee cap");
  assert(units(after, "YELLOW") >= 10n ** 19n, "sender must retain the real user minimum");
  const history = await gateway.transfers(RECIPIENT);
  assert(history.items.some((item) => item.from.toLowerCase() === account.address.toLowerCase() && item.nonce === body.nonce && item.direction === "in"));
  await shot(page, `${prefix}-send-sealed`);
}

async function rejectionFlow(page, prefix) {
  await page.goto(BASE + "/send?asset=YELLOW");
  await page.getByLabel("To", { exact: true }).fill(RECIPIENT);
  await page.getByLabel("YELLOW amount").fill("5");
  await page.getByRole("button", { name: /Change fee cap/ }).click();
  await page.getByLabel("Most you'll pay").fill("0.00001");
  await page.getByRole("button", { name: "Review transfer" }).click();
  const blocked = await page.getByText(/A lower cap will be rejected/).isVisible();
  if (!blocked) throw new Error("fee cap below estimate was not flagged");
  await shot(page, `${prefix}-send-fee-warning`);
}

async function operationPage(page, prefix) {
  await page.goto(BASE + "/activity");
  await page.getByText("Sent from this device").waitFor();
  await page.locator('a[href^="/activity/0x"]').first().click();
  await page.getByRole("heading", { name: "Sealed" }).waitFor();
  await shot(page, `${prefix}-operation`);
}

const browser = await chromium.launch();
const failures = [];
const runs = [
  { prefix: "desktop-light", width: 1440, height: 960 },
  { prefix: "desktop-dark", width: 1440, height: 960, dark: true },
  { prefix: "mobile-light", width: 390, height: 844, mobile: true },
  { prefix: "mobile-dark", width: 390, height: 844, mobile: true, dark: true },
  { prefix: "small-light", width: 360, height: 740, mobile: true },
];

for (const run of runs) {
  console.log(run.prefix);
  const { context, page } = await newPage(browser, run);
  try {
    await page.goto(BASE + "/");
    await page.getByRole("heading", { level: 1 }).waitFor();
    await page.waitForTimeout(2600);
    await shot(page, `${run.prefix}-connect`);
    await connect(page);
    await page.waitForTimeout(800);
    await shot(page, `${run.prefix}-account`);
    await sendFlow(page, run.prefix);
    await page.goto(BASE + "/activity");
    await page.getByText("Sent from this device").waitFor();
    await page.getByText(/^(Sent|Received)$/).first().waitFor();
    await shot(page, `${run.prefix}-activity`);
    await page.goto(BASE + "/receive");
    await page.getByRole("img", { name: /QR code/ }).waitFor();
    await shot(page, `${run.prefix}-receive`);
    if (run.prefix === "desktop-light" || run.prefix === "mobile-light") {
      await rejectionFlow(page, run.prefix);
      await operationPage(page, run.prefix);
    }
    if (!run.mobile || run.prefix === "mobile-light") {
      for (const p of ["fund", "trade", "agent"]) {
        await page.goto(`${BASE}/${p}`);
        await shot(page, `${run.prefix}-${p}`);
      }
    }
  } catch (e) {
    failures.push(`${run.prefix}: ${e.message.split("\n")[0]}`);
    await page.screenshot({ path: `${OUT}${run.prefix}-FAILURE.png`, fullPage: true }).catch(() => {});
  }
  await context.close();
}
await browser.close();
if (failures.length) {
  console.error("\nFailures:\n" + failures.join("\n"));
  process.exit(1);
}
console.log("\nAll flows passed.");
