"use client";

import type { ReactNode } from "react";
import { useAccount } from "wagmi";
import { ClearingShowcase } from "@/components/clearing";
import { Button } from "@/components/ui/button";
import { IconWallet } from "@/components/ui/icons";
import { useHydrated } from "@/lib/hydration";
import { useWalletUi } from "@/lib/wallet/wallet-provider";

/** The one screen shown anywhere a wallet is needed and none is connected. */
export function ConnectScreen() {
  const { open } = useWalletUi();
  return (
    <section className="pt-2 lg:pt-6" aria-labelledby="connect-title">
      <div className="grid items-center gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)] lg:gap-14">
        <div>
          <h1 id="connect-title" className="type-hero max-w-[16ch]">
            Settle across chains on one network.
          </h1>
          <p className="lede mt-5">
            Yellow Network brings assets from different blockchains into one account. Connect your wallet to move value across chains in seconds.
          </p>
          <Button size="lg" className="mt-8" onClick={() => open("Connect")} icon={<IconWallet />}>
            Connect wallet
          </Button>
        </div>
        <ClearingShowcase className="mx-auto w-full max-w-[340px] sm:max-w-[420px] lg:max-w-[480px]" />
      </div>
    </section>
  );
}

export function RequireWallet({ children }: { children: (address: `0x${string}`) => ReactNode }) {
  const { address, status } = useAccount();
  const hydrated = useHydrated();
  if (!hydrated) {
    // Matches the server HTML; CSS picks the right one for returning wallets.
    return (
      <>
        <div className="when-wallet-pending">
          <PageSkeleton />
        </div>
        <div className="unless-wallet-pending">
          <ConnectScreen />
        </div>
      </>
    );
  }
  if (status === "reconnecting" || status === "connecting") return <PageSkeleton />;
  if (!address) return <ConnectScreen />;
  return <>{children(address)}</>;
}

export function PageSkeleton() {
  return (
    <div className="space-y-6 pt-4" aria-busy="true" aria-label="Loading">
      <div className="skeleton h-5 w-40" />
      <div className="skeleton h-24 w-[70%]" />
      <div className="skeleton h-64 w-full !rounded-sheet" />
    </div>
  );
}
