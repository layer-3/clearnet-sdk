import { findNetwork } from "@/config/networks";

// Same-origin relay for gateways that don't send CORS headers. Only networks
// configured with `"proxy": true` are relayed, only gateway routes, and
// redirects are never followed (a signed POST must not be replayed elsewhere).

export const dynamic = "force-dynamic";

type Ctx = { params: Promise<{ network: string; path: string[] }> };

const ALLOWED = /^(v1\/[A-Za-z0-9._~\-/]+|rpc)$/;
const MAX_BODY = 64 * 1024;
const MAX_RESPONSE = 2 << 20;

// JSON-RPC-style codes the SDK already understands; proxy-local failures must
// not reuse the gateway's own codes.
const CODE_ROUTE_NOT_FOUND = -32601;
const CODE_INVALID_REQUEST = -32600;
const CODE_BACKEND_UNAVAILABLE = -32008;

function fail(status: number, code: number, message: string) {
  return Response.json({ error: { code, message } }, { status, headers: { "Cache-Control": "no-store" } });
}

/**
 * Reads a stream up to `limit` bytes. Returns null when the body is larger, so
 * an oversized request or response is rejected while it is still streaming
 * instead of being buffered in full first.
 */
async function readLimited(stream: ReadableStream<Uint8Array> | null, limit: number): Promise<string | null> {
  if (!stream) return "";
  const reader = stream.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    if (!value) continue;
    total += value.byteLength;
    if (total > limit) {
      await reader.cancel().catch(() => undefined);
      return null;
    }
    chunks.push(value);
  }
  const merged = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    merged.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return new TextDecoder().decode(merged);
}

/** Returns the declared body size when the header is present and usable. */
function declaredLength(req: Request): number | null {
  const raw = req.headers.get("content-length");
  if (raw === null) return null;
  const value = Number(raw);
  return Number.isFinite(value) && value >= 0 ? value : null;
}

async function relay(req: Request, ctx: Ctx, method: "GET" | "POST") {
  const { network, path } = await ctx.params;
  const n = findNetwork(network);
  if (!n || !n.proxy) return fail(404, CODE_ROUTE_NOT_FOUND, "unknown network");
  const route = path.join("/");
  if (!ALLOWED.test(route) || route.includes("..")) return fail(404, CODE_ROUTE_NOT_FOUND, "route not found");

  const requestURL = new URL(req.url);
  // Browser writes must originate from this app. Never make this endpoint a
  // blind cross-site relay to a configured gateway.
  if (method === "POST") {
    const origin = req.headers.get("Origin");
    if ((origin && origin !== requestURL.origin) || req.headers.get("Sec-Fetch-Site") === "cross-site") {
      return fail(403, CODE_INVALID_REQUEST, "cross-origin write refused");
    }
  }
  const target = new URL(`${n.gateway.replace(/\/+$/, "")}/${route}`);
  target.search = requestURL.search;

  let body: string | undefined;
  if (method === "POST") {
    const declared = declaredLength(req);
    if (declared !== null && declared > MAX_BODY) return fail(413, CODE_INVALID_REQUEST, "request body too large");
    try {
      const raw = await readLimited(req.body, MAX_BODY);
      if (raw === null) return fail(413, CODE_INVALID_REQUEST, "request body too large");
      body = raw;
    } catch {
      return fail(400, CODE_INVALID_REQUEST, "cannot read request body");
    }
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
    return fail(502, CODE_BACKEND_UNAVAILABLE, "gateway unreachable");
  }
  if (upstream.status >= 300 && upstream.status < 400) {
    return fail(502, CODE_BACKEND_UNAVAILABLE, "gateway attempted a redirect");
  }
  const declared = Number(upstream.headers.get("content-length") ?? "");
  if (Number.isFinite(declared) && declared > MAX_RESPONSE) {
    return fail(502, CODE_BACKEND_UNAVAILABLE, "gateway response too large");
  }
  let text: string | null;
  try {
    text = await readLimited(upstream.body, MAX_RESPONSE);
  } catch {
    // The gateway may have accepted a signed POST before its response failed.
    return fail(502, CODE_BACKEND_UNAVAILABLE, "gateway response unreadable");
  }
  if (text === null) return fail(502, CODE_BACKEND_UNAVAILABLE, "gateway response too large");
  // A 204/205/304 response must have a null body, not even an empty string.
  // JSON-RPC notifications legitimately produce 204 (including in clearbox).
  return new Response([204, 205, 304].includes(upstream.status) ? null : text, {
    status: upstream.status,
    // Every relayed route answers JSON. Never forward the upstream content type
    // verbatim: a gateway answering text/html would be served on this origin.
    headers: { "Content-Type": "application/json", "Cache-Control": "no-store" },
  });
}

export function GET(req: Request, ctx: Ctx) {
  return relay(req, ctx, "GET");
}

export function POST(req: Request, ctx: Ctx) {
  return relay(req, ctx, "POST");
}
