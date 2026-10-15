"use client";

import { createContext, useContext, useEffect, useMemo, useRef, type ReactNode } from "react";
import { WagmiProvider, createConfig, http, useAccount, useConnect, useConnectors, type Config } from "wagmi";
import { injected } from "wagmi/connectors";
import type { Chain } from "viem";
import { NETWORKS, REOWN_PROJECT_ID, SITE_URL } from "@/config/networks";
import { createAppKit } from "@reown/appkit/react";
import { WagmiAdapter } from "@reown/appkit-adapter-wagmi";
import { clearWalletHint } from "@/lib/hydration";
import { yellowChain } from "./chains";

type OpenView = "Connect" | "Account";

interface WalletUi {
  /** Opens the wallet picker (or account view when connected). */
  open: (view?: OpenView) => void;
  /** True when Reown AppKit (WalletConnect, 500+ wallets) is configured. */
  universal: boolean;
}

const WalletUiContext = createContext<WalletUi>({ open: () => undefined, universal: false });

export function useWalletUi() {
  return useContext(WalletUiContext);
}

const origin = typeof window !== "undefined" ? window.location.origin : SITE_URL || "http://localhost:3000";
const chains = NETWORKS.map((n) => yellowChain(n, origin)) as unknown as [Chain, ...Chain[]];

interface Built {
  config: Config;
  open?: (view: OpenView) => void;
}

function build(): Built {
  if (REOWN_PROJECT_ID && typeof window !== "undefined") {
    const adapter = new WagmiAdapter({ networks: chains, projectId: REOWN_PROJECT_ID, ssr: true });
    const dark = document.documentElement.classList.contains("dark");
    const modal = createAppKit({
      adapters: [adapter],
      networks: chains,
      projectId: REOWN_PROJECT_ID,
      metadata: {
        name: "Yellow",
        description: "Your account on Yellow Network",
        url: origin,
        icons: [`${origin}/icon.svg`],
      },
      allowUnsupportedChain: true,
      themeMode: dark ? "dark" : "light",
      themeVariables: {
        "--apkt-accent": dark ? "#FCD000" : "#000000",
        "--apkt-color-mix": "#FCD000",
        "--apkt-color-mix-strength": 0,
        "--apkt-border-radius-master": "3px",
        "--apkt-font-family": "var(--font-archivo), ui-sans-serif, system-ui",
      },
      // Yellow transfers need a plain account signature (EOA), so embedded
      // smart-account logins and AppKit's own wallet features are off.
      features: { email: false, socials: false, analytics: false, swaps: false, onramp: false, send: false, history: false, pay: false },
    });
    const observer = new MutationObserver(() => {
      const isDark = document.documentElement.classList.contains("dark");
      modal.setThemeMode(isDark ? "dark" : "light");
      modal.setThemeVariables({ "--apkt-accent": isDark ? "#FCD000" : "#000000" });
    });
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ["class"] });
    return { config: adapter.wagmiConfig, open: (view) => void modal.open({ view }) };
  }
  const transports = Object.fromEntries(chains.map((c) => [c.id, http(c.rpcUrls.default.http[0])]));
  return {
    config: createConfig({ chains, connectors: [injected({ shimDisconnect: true })], transports, ssr: true }),
  };
}

let built: Built | undefined;
function getBuilt() {
  built ??= build();
  return built;
}

export function WalletProvider({ children }: { children: ReactNode }) {
  const { config, open } = getBuilt();
  return (
    <WagmiProvider config={config} reconnectOnMount>
      <WalletUiBridge open={open}>{children}</WalletUiBridge>
    </WagmiProvider>
  );
}

function WalletUiBridge({ open, children }: { open?: (view: OpenView) => void; children: ReactNode }) {
  const connectors = useConnectors();
  const { mutate: connect } = useConnect();
  const { status } = useAccount();
  const sawReconnect = useRef(false);
  useEffect(() => {
    if (status === "reconnecting" || status === "connecting") sawReconnect.current = true;
    if (status === "connected" || (status === "disconnected" && sawReconnect.current)) clearWalletHint();
  }, [status]);
  useEffect(() => {
    const id = window.setTimeout(clearWalletHint, 4000);
    return () => window.clearTimeout(id);
  }, []);
  const value = useMemo<WalletUi>(() => {
    if (open) return { open: (view = "Connect") => open(view), universal: true };
    return {
      universal: false,
      open: () => {
        const connector = connectors.find((c) => c.type === "injected") ?? connectors[0];
        if (connector) connect({ connector });
      },
    };
  }, [open, connectors, connect]);
  return <WalletUiContext.Provider value={value}>{children}</WalletUiContext.Provider>;
}
