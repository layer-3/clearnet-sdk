import { z } from "zod";
import { networkChainId } from "@yellow-org/clearnet-sdk/gateway";

const networkSchema = z
  .object({
    slug: z.string().regex(/^[a-z0-9][a-z0-9-]{0,31}$/, "slug must be lowercase letters, digits or dashes"),
    name: z.string().min(1).max(40),
    gateway: z.url({ protocol: /^https?$/ }),
    networkId: z.string().length(4),
    proxy: z.boolean().optional(),
    fingerprint: z.string().regex(/^[0-9a-fA-F]{64}$/).optional(),
    rpcUrl: z.url().optional(),
    explorer: z.url().optional(),
  })
  .strict();

export type NetworkConfig = z.infer<typeof networkSchema>;

export interface YellowNetwork extends NetworkConfig {
  chainId: number;
}

const LOCAL: NetworkConfig = { slug: "local", name: "Local", gateway: "http://127.0.0.1:8080", networkId: "YDEV", proxy: true };

function load(): { networks: YellowNetwork[]; defaultSlug: string; error?: string } {
  // Must be referenced literally so Next.js inlines it at build time.
  const raw = process.env.NEXT_PUBLIC_YELLOW_NETWORKS;
  let configs: NetworkConfig[] = [LOCAL];
  let error: string | undefined;
  if (raw && raw.trim()) {
    try {
      const parsed = z.array(networkSchema).min(1).parse(JSON.parse(raw));
      const slugs = new Set(parsed.map((n) => n.slug));
      if (slugs.size !== parsed.length) throw new Error("network slugs must be unique");
      configs = parsed;
    } catch (e) {
      error = `NEXT_PUBLIC_YELLOW_NETWORKS is invalid: ${e instanceof Error ? e.message : String(e)}`;
      if (process.env.NODE_ENV === "production") throw new Error(error);
      console.error(error);
    }
  }
  const networks = configs.map((c) => ({ ...c, chainId: networkChainId(c.networkId) }));
  const wanted = process.env.NEXT_PUBLIC_YELLOW_DEFAULT_NETWORK;
  const defaultSlug = networks.find((n) => n.slug === wanted)?.slug ?? networks[0]!.slug;
  return { networks, defaultSlug, error };
}

const loaded = load();

export const NETWORKS: readonly YellowNetwork[] = loaded.networks;
export const DEFAULT_NETWORK_SLUG = loaded.defaultSlug;
export const CONFIG_ERROR = loaded.error;

export function findNetwork(slug: string | null | undefined): YellowNetwork | undefined {
  return NETWORKS.find((n) => n.slug === slug);
}

/** Base URL the browser uses for REST calls. */
export function apiBase(n: YellowNetwork): string {
  if (n.proxy) return `/api/gw/${n.slug}`;
  return n.gateway;
}

/** JSON-RPC URL handed to wallets when adding the Yellow chain. Must be absolute. */
export function walletRpcUrl(n: YellowNetwork, origin: string): string {
  if (n.rpcUrl) return n.rpcUrl;
  if (n.proxy) return `${origin}/api/gw/${n.slug}/rpc`;
  return `${n.gateway.replace(/\/+$/, "")}/rpc`;
}

export const REOWN_PROJECT_ID = process.env.NEXT_PUBLIC_REOWN_PROJECT_ID ?? "";
export const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? "";
