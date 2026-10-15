import { findNetwork } from "@/config/networks";

// Same-origin relay for gateways that don't send CORS headers. Only networks
// configured with `"proxy": true` are relayed, only gateway routes, and
// redirects are never followed (a signed POST must not be replayed elsewhere).

export const dynamic = "force-dynamic";

type Ctx = { params: Promise<{ network: string; path: string[] }> };

const ALLOWED = /^(v1\/[A-Za-z0-9._~\-/]+|rpc)$/;
const MAX_BODY = 64 * 1024;
const MAX_RESPONSE = 2 << 20;

function fail(status: number, message: string) {
  return Response.json({ error: { code: -32008, message } }, { status, headers: { "Cache-Control": "no-store" } });
}

async function relay(req: Request, ctx: Ctx, method: "GET" | "POST") {
  const { network, path } = await ctx.params;
  const n = findNetwork(network);
  if (!n || !n.proxy) return fail(404, "unknown network");
  const route = path.join("/");
  if (!ALLOWED.test(route) || route.includes("..")) return fail(404, "route not found");

  const target = new URL(`${n.gateway.replace(/\/+$/, "")}/${route}`);
  target.search = new URL(req.url).search;

  let body: string | undefined;
  if (method === "POST") {
    body = await req.text();
    if (body.length > MAX_BODY) return fail(413, "request body too large");
  }

  let upstream: Response;
  try {
    upstream = await fetch(target, {
      method,
      headers: { Accept: "application/json", ...(body !== undefined && { "Content-Type": "application/json" }) },
      ...(body !== undefined && { body }),
      redirect: "manual",
      cache: "no-store",
      signal: AbortSignal.timeout(20_000),
    });
  } catch {
    // For a POST this means the outcome is unknown; the client treats 502 that way.
    return fail(502, "gateway unreachable");
  }
  if (upstream.status >= 300 && upstream.status < 400) return fail(502, "gateway attempted a redirect");
  const text = await upstream.text();
  if (text.length > MAX_RESPONSE) return fail(502, "gateway response too large");
  return new Response(text, {
    status: upstream.status,
    headers: { "Content-Type": upstream.headers.get("Content-Type") ?? "application/json", "Cache-Control": "no-store" },
  });
}

export function GET(req: Request, ctx: Ctx) {
  return relay(req, ctx, "GET");
}

export function POST(req: Request, ctx: Ctx) {
  return relay(req, ctx, "POST");
}
