"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, type ReactNode } from "react";
import { NetworkProvider } from "@/lib/network-context";
import { WalletProvider } from "@/lib/wallet/wallet-provider";

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            retry: (count, error) => count < 2 && !(error instanceof Error && error.name === "GatewayError" && "kind" in error && error.kind !== "network"),
            refetchOnWindowFocus: true,
          },
        },
      }),
  );
  return (
    <QueryClientProvider client={queryClient}>
      <WalletProvider>
        <NetworkProvider>{children}</NetworkProvider>
      </WalletProvider>
    </QueryClientProvider>
  );
}
