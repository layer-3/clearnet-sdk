"use client";

import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useMemo, useSyncExternalStore } from "react";
import { useAccount } from "wagmi";
import { journal } from "@/lib/journal/store";
import type { JournalEntry } from "@/lib/journal/copy";
import { useNetwork } from "@/lib/network-context";
import { buildPortfolio } from "@/lib/portfolio";

export function useInfo() {
  const { network, client } = useNetwork();
  return useQuery({
    queryKey: ["info", network.slug],
    queryFn: ({ signal }) => client.info({ signal }),
    staleTime: 30_000,
    refetchInterval: 60_000,
  });
}

export function useBalance(address: string | undefined) {
  const { network, client } = useNetwork();
  return useQuery({
    queryKey: ["balance", network.slug, address?.toLowerCase()],
    queryFn: ({ signal }) => client.balance(address!, signal),
    enabled: !!address,
    refetchInterval: 5_000,
  });
}

export function usePools() {
  const { network, client } = useNetwork();
  return useQuery({
    queryKey: ["pools", network.slug],
    queryFn: ({ signal }) => client.pools(signal),
    staleTime: 60_000,
  });
}

export function usePortfolio(address: string | undefined) {
  const info = useInfo();
  const balance = useBalance(address);
  const pools = usePools();
  const portfolio = useMemo(
    () => (info.data && balance.data ? buildPortfolio(info.data.assets, balance.data, pools.data) : undefined),
    [info.data, balance.data, pools.data],
  );
  return { portfolio, info, balance, pools, error: info.error ?? balance.error, isLoading: info.isLoading || balance.isLoading };
}

export const HISTORY_PAGE = 20;

export function useTransfers(address: string | undefined) {
  const { network, client } = useNetwork();
  return useInfiniteQuery({
    queryKey: ["transfers", network.slug, address?.toLowerCase()],
    queryFn: ({ pageParam, signal }) =>
      client.transfers(address!, { limit: HISTORY_PAGE, signal, ...(pageParam && { cursor: pageParam }) }),
    initialPageParam: "",
    getNextPageParam: (last) => last.nextCursor || undefined,
    enabled: !!address,
    refetchInterval: 10_000,
  });
}

export function useJournalEntries(): JournalEntry[] {
  return useSyncExternalStore(journal.subscribe, journal.getSnapshot, journal.getServerSnapshot);
}

/** Saved transfers for the connected account on the current network. */
export function useMyOperations() {
  const all = useJournalEntries();
  const { network } = useNetwork();
  const { address } = useAccount();
  return useMemo(
    () => all.filter((e) => e.networkSlug === network.slug && !!address && e.sender.toLowerCase() === address.toLowerCase()),
    [all, network.slug, address],
  );
}
