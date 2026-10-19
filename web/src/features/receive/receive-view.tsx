"use client";

import { USER_URI_PREFIX } from "@yellow-org/clearnet-sdk/gateway";
import { clsx } from "clsx";
import QRCode from "qrcode";
import { useEffect, useMemo, useState } from "react";
import { RequireWallet } from "@/components/connect-screen";
import { YellowMark } from "@/components/ui/brand";
import { Button } from "@/components/ui/button";
import { useCopy } from "@/components/ui/copy-button";
import { IconCheck, IconCopy, IconLink, IconShare } from "@/components/ui/icons";
import { groupedHex } from "@/lib/format";
import { useNetwork } from "@/lib/network-context";

export function ReceiveView() {
  return <RequireWallet>{(address) => <Receive address={address} />}</RequireWallet>;
}

type Format = "address" | "link";

function Receive({ address }: { address: `0x${string}` }) {
  const { network } = useNetwork();
  const [format, setFormat] = useState<Format>("address");
  const [svg, setSvg] = useState("");
  const addressCopy = useCopy();
  const linkCopy = useCopy();
  const value = format === "address" ? address : `${USER_URI_PREFIX}${address.toLowerCase()}`;

  const payLink = useMemo(() => {
    if (typeof window === "undefined") return "";
    const u = new URL("/send", window.location.origin);
    u.searchParams.set("to", address);
    return u.toString();
  }, [address]);

  useEffect(() => {
    let alive = true;
    QRCode.toString(value, { type: "svg", margin: 0, errorCorrectionLevel: "H", color: { dark: "#000000", light: "#ffffff" } }).then((s) => alive && setSvg(s));
    return () => {
      alive = false;
    };
  }, [value]);

  const share = async () => {
    const data = { title: "My Yellow account", text: `Send to my Yellow account on ${network.name}`, url: payLink };
    if (navigator.share) {
      try {
        await navigator.share(data);
        return;
      } catch {
        /* dismissed */
      }
    }
    await linkCopy.copy(payLink);
  };

  return (
    <div className="pt-2 lg:pt-6">
      <h1 className="type-h1">Receive</h1>
      <div className="mt-8 grid gap-6 lg:grid-cols-[420px_minmax(0,1fr)] lg:items-start">
        <section className="sheet p-6 sm:p-8" aria-label="Your address">
          <div role="tablist" aria-label="Address format" className="mb-6 grid grid-cols-2 rounded-field bg-surface-2 p-1">
            {(["address", "link"] as const).map((f) => (
              <button
                key={f}
                role="tab"
                aria-selected={format === f}
                onClick={() => setFormat(f)}
                className={clsx("type-label h-10 rounded-control text-[14px] transition-colors", format === f ? "bg-on-surface text-surface" : "text-muted")}
              >
                {f === "address" ? "Address" : "Yellow link"}
              </button>
            ))}
          </div>
          <div className="relative mx-auto aspect-square w-full max-w-[300px] rounded-field bg-white p-5 ring-1 ring-line">
            {svg ? <div className="size-full [&_svg]:size-full" dangerouslySetInnerHTML={{ __html: svg }} role="img" aria-label={`QR code for ${value}`} /> : <div className="skeleton size-full" />}
            <span className="absolute left-1/2 top-1/2 grid size-16 -translate-x-1/2 -translate-y-1/2 place-items-center rounded-full bg-white">
              <YellowMark size={52} />
            </span>
          </div>
          <p className="mt-6 break-all text-center font-mono text-[16px] leading-7" aria-label={value}>
            {format === "address"
              ? groupedHex(address).map((g, i) => (
                  <span key={i} className={clsx(i % 2 ? "text-muted" : undefined, "mr-[0.35em]")}>
                    {g}
                  </span>
                ))
              : value}
          </p>
          <div className="mt-6 grid grid-cols-2 gap-2">
            <Button size="lg" onClick={() => addressCopy.copy(value)} icon={addressCopy.copied ? <IconCheck /> : <IconCopy />}>
              {addressCopy.copied ? "Copied" : "Copy"}
            </Button>
            <Button size="lg" variant="quiet" onClick={share} icon={linkCopy.copied ? <IconCheck /> : <IconShare />}>
              {linkCopy.copied ? "Link copied" : "Share"}
            </Button>
          </div>
        </section>

        <div className="space-y-6">
          <section className="sheet p-6 sm:p-8">
            <h2 className="type-title text-[20px]">Payment link</h2>
            <p className="mt-2 max-w-[52ch] text-[15px] leading-relaxed text-muted">
              Share a link that opens Send with your address already filled in.
            </p>
            <div className="mt-4 flex items-center gap-2 rounded-field bg-surface-2 py-2 pl-4 pr-2">
              <IconLink size={18} className="shrink-0 text-muted" />
              <span className="min-w-0 flex-1 truncate font-mono text-[13px]">{payLink}</span>
              <Button size="sm" variant="primary" onClick={() => linkCopy.copy(payLink)}>
                {linkCopy.copied ? "Copied" : "Copy link"}
              </Button>
            </div>
          </section>

          <section className="rounded-sheet p-6 shadow-[inset_0_0_0_2px_var(--on-canvas)] sm:p-8">
            <h2 className="type-title text-[20px]">Only for transfers on Yellow Network</h2>
            <p className="mt-2 max-w-[56ch] text-[15px] leading-relaxed text-on-canvas-muted">
              This address looks like an Ethereum address, but it receives transfers inside Yellow Network ({network.name}). Tokens sent to it on Ethereum or another blockchain
              won&apos;t appear here.
            </p>
          </section>
        </div>
      </div>
    </div>
  );
}
