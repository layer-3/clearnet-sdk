"use client";

import { useState } from "react";
import { useAccount, useDisconnect } from "wagmi";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/components/ui/copy-button";
import { Identicon } from "@/components/ui/identicon";
import { IconCheck, IconCopy, IconLogout, IconWallet } from "@/components/ui/icons";
import { Sheet } from "@/components/ui/sheet";
import { groupedHex, shortAddress } from "@/lib/format";
import { useHydrated } from "@/lib/hydration";
import { useWalletUi } from "@/lib/wallet/wallet-provider";

export function AccountButton() {
  const { address, status, connector } = useAccount();
  const { disconnect } = useDisconnect();
  const { open, universal } = useWalletUi();
  const [sheet, setSheet] = useState(false);
  const { copied, copy } = useCopy();
  const hydrated = useHydrated();

  const connectButton = (
    <Button size="sm" className="h-10 px-4" onClick={() => open("Connect")} icon={<IconWallet size={18} />}>
      <span>
        Connect<span className="hidden sm:inline"> wallet</span>
      </span>
    </Button>
  );
  const pending = <span className="skeleton block h-10 w-36 !rounded-control" aria-label="Connecting wallet" />;
  if (!hydrated) {
    return (
      <>
        <span className="when-wallet-pending">{pending}</span>
        <span className="unless-wallet-pending">{connectButton}</span>
      </>
    );
  }
  if (status === "reconnecting" || status === "connecting") return pending;
  if (!address) return connectButton;
  return (
    <>
      <button
        type="button"
        onClick={() => setSheet(true)}
        className="inline-flex h-10 items-center gap-2 rounded-control border border-line bg-surface py-1 pl-1 pr-3.5 text-on-surface transition-colors hover:border-on-surface"
        aria-label={`Account ${address}`}
      >
        <Identicon address={address} size={32} />
        <span className="font-mono text-[13px]">{shortAddress(address, 5, 4)}</span>
      </button>
      <Sheet open={sheet} onOpenChange={setSheet} title="Your account" description={connector ? `Connected with ${connector.name}` : undefined}>
        <div className="flex items-center gap-4 rounded-field bg-surface-2 p-4">
          <Identicon address={address} size={52} />
          <p className="min-w-0 flex-1 break-all font-mono text-[14px] leading-6" aria-label={address}>
            {groupedHex(address).map((g, i) => (
              <span key={i} className={i % 2 ? "text-muted" : undefined}>
                {g}
              </span>
            ))}
          </p>
        </div>
        <div className="mt-4 grid gap-2">
          <Button variant="quiet" size="lg" className="justify-start" onClick={() => copy(address)} icon={copied ? <IconCheck /> : <IconCopy />}>
            {copied ? "Address copied" : "Copy address"}
          </Button>
          {universal && (
            <Button variant="quiet" size="lg" className="justify-start" onClick={() => { setSheet(false); open("Account"); }} icon={<IconWallet />}>
              Wallet details
            </Button>
          )}
          <Button
            variant="quiet"
            size="lg"
            className="justify-start text-flare"
            onClick={() => {
              disconnect();
              setSheet(false);
            }}
            icon={<IconLogout />}
          >
            Disconnect
          </Button>
        </div>
      </Sheet>
    </>
  );
}
