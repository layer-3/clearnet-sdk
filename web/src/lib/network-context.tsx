"use client";

import { GatewayClient } from "@yellow-org/clearnet-sdk/gateway";
import { createContext, useCallback, useContext, useMemo, useSyncExternalStore, type ReactNode } from "react";
import { apiBase, DEFAULT_NETWORK_SLUG, findNetwork, NETWORKS, type YellowNetwork } from "@/config/networks";

const KEY = "yellow:network";

interface NetworkContextValue {
  network: YellowNetwork;
  client: GatewayClient;
  setNetwork: (slug: string) => void;
}

const NetworkContext = createContext<NetworkContextValue | null>(null);

const listeners = new Set<() => void>();
function subscribe(l: () => void) {
  listeners.add(l);
  const onStorage = (e: StorageEvent) => e.key === KEY && l();
  window.addEventListener("storage", onStorage);
  return () => {
    listeners.delete(l);
    window.removeEventListener("storage", onStorage);
  };
}
function readSlug() {
  const stored = window.localStorage.getItem(KEY);
  return findNetwork(stored)?.slug ?? DEFAULT_NETWORK_SLUG;
}

const clients = new Map<string, GatewayClient>();

/** One SDK client per (base URL, network ID), shared app-wide. */
export function clientFor(baseUrl: string, networkId: string, fingerprint?: string): GatewayClient {
  const key = `${baseUrl}|${networkId}|${fingerprint ?? ""}`;
  let c = clients.get(key);
  if (!c) {
    c = new GatewayClient({ baseUrl, networkId, ...(fingerprint && { fingerprint }) });
    clients.set(key, c);
  }
  return c;
}

export function clientForNetwork(n: YellowNetwork) {
  return clientFor(apiBase(n), n.networkId, n.fingerprint);
}

export function NetworkProvider({ children }: { children: ReactNode }) {
  const slug = useSyncExternalStore(subscribe, readSlug, () => DEFAULT_NETWORK_SLUG);
  const network = findNetwork(slug) ?? NETWORKS[0]!;
  const setNetwork = useCallback((next: string) => {
    if (!findNetwork(next)) return;
    window.localStorage.setItem(KEY, next);
    listeners.forEach((l) => l());
  }, []);
  const value = useMemo(() => ({ network, client: clientForNetwork(network), setNetwork }), [network, setNetwork]);
  return <NetworkContext.Provider value={value}>{children}</NetworkContext.Provider>;
}

export function useNetwork() {
  const ctx = useContext(NetworkContext);
  if (!ctx) throw new Error("useNetwork must be used inside NetworkProvider");
  return ctx;
}
