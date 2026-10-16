import { afterEach, describe, expect, it, vi } from "vitest";
import { POST } from "./[network]/[...path]/route";

vi.mock("@/config/networks", () => ({
  findNetwork: (slug: string) => slug === "local" ? { gateway: "https://gateway.test", proxy: true } : undefined,
}));

const ctx = () => ({ params: Promise.resolve({ network: "local", path: ["rpc"] }) });
const request = (body: string, headers: Record<string, string> = {}) => new Request("https://wallet.test/api/gw/local/rpc", {
  method: "POST", body, headers: { "Content-Type": "application/json", Origin: "https://wallet.test", ...headers },
});

afterEach(() => vi.unstubAllGlobals());

describe("gateway relay", () => {
  it("returns JSON-RPC notification 204s with no body", async () => {
    const upstream = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", upstream);
    const res = await POST(request('{"jsonrpc":"2.0","method":"net_listening"}'), ctx());
    expect(res.status).toBe(204);
    expect(await res.text()).toBe("");
    expect(upstream).toHaveBeenCalledTimes(1);
  });

  it("never serves gateway-supplied HTML on this origin", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => new Response("<script>bad()</script>", {
      headers: { "Content-Type": "text/html" },
    })));
    const res = await POST(request("{}"), ctx());
    expect(res.headers.get("Content-Type")).toBe("application/json");
    expect(res.headers.get("Cache-Control")).toBe("no-store");
  });

  it("rejects cross-site writes without contacting the gateway", async () => {
    const upstream = vi.fn();
    vi.stubGlobal("fetch", upstream);
    const res = await POST(request("{}", { Origin: "https://attacker.test" }), ctx());
    expect(res.status).toBe(403);
    expect(upstream).not.toHaveBeenCalled();
  });

  it("caps uploads by bytes before relaying them", async () => {
    const upstream = vi.fn();
    vi.stubGlobal("fetch", upstream);
    const res = await POST(request("a".repeat(64 * 1024 + 1)), ctx());
    expect(res.status).toBe(413);
    expect(upstream).not.toHaveBeenCalled();
  });
});
