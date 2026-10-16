"use client";

import {
  blockingOperation,
  isFinalOperationState,
  normalizeAmountInput,
  staleNonceOperation,
  type AssetInfo,
  type PoolSummary,
  type TransferIntent,
} from "@yellow-org/clearnet-sdk/gateway";
import { useSearchParams } from "next/navigation";
import { useMemo, useState } from "react";
import { Clearing } from "@/components/clearing";
import { PageSkeleton, RequireWallet } from "@/components/connect-screen";
import { Button, ButtonLink } from "@/components/ui/button";
import { IconAlert, IconChevronLeft } from "@/components/ui/icons";
import { ErrorPanel } from "@/components/ui/misc";
import { useMyOperations, usePortfolio } from "@/hooks/gateway";
import { STATE_COPY } from "@/lib/journal/copy";
import { formatUnits } from "@/lib/format";
import { useNetwork } from "@/lib/network-context";
import { YELLOW, type Portfolio } from "@/lib/portfolio";
import { Compose } from "./compose";
import { newLeg, validate, type ComposeState } from "./model";
import { Progress } from "./progress";
import { Review } from "./review";
import { useSendTransfer } from "./use-send";

export function SendView() {
  return <RequireWallet>{(address) => <SendLoader address={address} />}</RequireWallet>;
}

function SendLoader({ address }: { address: `0x${string}` }) {
  const { network } = useNetwork();
  const { portfolio, info, pools, error, isLoading } = usePortfolio(address);
  if (error && !portfolio) return <ErrorPanel error={error} onRetry={() => void info.refetch()} className="mt-6" />;
  if (isLoading || !portfolio || !info.data) return <PageSkeleton />;
  return <Send key={`${network.slug}:${address}`} address={address} portfolio={portfolio} assets={info.data.assets} pools={pools.data} />;
}

function initialState(params: URLSearchParams, portfolio: Portfolio): ComposeState {
  const wanted = params.get("asset");
  const asset =
    portfolio.holdings.find((h) => h.asset.symbol === wanted)?.asset.symbol ??
    portfolio.holdings.find((h) => h.available > 0n && h.asset.symbol !== YELLOW)?.asset.symbol ??
    portfolio.holdings[0]?.asset.symbol ??
    YELLOW;
  return { recipient: params.get("to") ?? "", legs: [newLeg(asset)], feeCap: null };
}

function Send({ address, portfolio, assets, pools }: { address: `0x${string}`; portfolio: Portfolio; assets: AssetInfo[]; pools: PoolSummary[] | undefined }) {
  const params = useSearchParams();
  const { network, client } = useNetwork();
  const ops = useMyOperations();
  const [state, setState] = useState<ComposeState>(() => initialState(params, portfolio));
  const [step, setStep] = useState<"compose" | "review">("compose");
  const [intent, setIntent] = useState<TransferIntent | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showErrors, setShowErrors] = useState(false);
  const { progress, send, reset } = useSendTransfer();

  const result = useMemo(() => validate(state, portfolio, assets, pools, address), [state, portfolio, assets, pools, address]);
  const legLabels = useMemo(
    () =>
      state.legs.map((l) => {
        const amount = normalizeAmountInput(l.amount);
        return amount && /^\d+(\.\d+)?$/.test(amount) ? `${amount.length > 10 ? Number(amount).toPrecision(6) : amount} ${l.asset}` : `— ${l.asset}`;
      }),
    [state.legs],
  );

  const blocking = blockingOperation(ops, network.networkId, address);
  if (progress) {
    return (
      <div className="pt-4 lg:pt-10">
        <Progress
          progress={progress}
          assets={assets}
          networkName={network.name}
          legs={intent ? intent.assets.map((a) => `${formatUnits(a.amount, assets.find((x) => x.symbol === a.asset)?.decimals ?? 0, 6)} ${a.asset}`) : legLabels}
          onReset={() => {
            reset();
            setIntent(null);
            setStep("compose");
            setShowErrors(false);
            setState(initialState(new URLSearchParams(), portfolio));
          }}
          onEdit={() => {
            reset();
            setStep("compose");
          }}
        />
      </div>
    );
  }
  if (blocking) return <Blocked id={blocking.id} state={blocking.state} />;

  const prepare = async () => {
    setError(null);
    if (!result.ready || !result.recipient) {
      setShowErrors(true);
      if (result.formError) setError(result.formError);
      return;
    }
    setBusy(true);
    try {
      const prepared = await client.prepareTransfer({
        from: address,
        recipient: result.recipient,
        assets: state.legs.map((l) => ({ asset: l.asset, amount: normalizeAmountInput(l.amount) })),
        maxFee: normalizeAmountInput(result.feeCap),
      });
      const stale = staleNonceOperation(ops, network.networkId, address, prepared.nonce);
      if (stale) {
        setError("The network is still catching up with your last transfer. Wait a few seconds and review again.");
        return;
      }
      setIntent(prepared);
      setStep("review");
      window.scrollTo({ top: 0, behavior: "smooth" });
    } catch (e) {
      setError(e instanceof Error ? e.message : "Couldn't prepare the transfer.");
    } finally {
      setBusy(false);
    }
  };

  const sign = async () => {
    if (!intent) return;
    setError(null);
    setBusy(true);
    const out = await send(intent);
    setBusy(false);
    if (!out.ok) setError(out.error);
  };

  return (
    <div className="grid gap-8 pt-2 lg:grid-cols-[minmax(0,1fr)_minmax(320px,440px)] lg:gap-12 lg:pt-6">
      <div className="min-w-0">
        {step === "review" ? (
          <button type="button" onClick={() => { setStep("compose"); setError(null); }} className="type-label -ml-2 mb-2 inline-flex h-10 items-center gap-1 rounded-control pl-1 pr-3 text-[15px] hover:bg-current/7">
            <IconChevronLeft size={20} /> Edit
          </button>
        ) : null}
        <h1 className="type-h1">{step === "review" ? "Review" : "Send"}</h1>

        <div className="sheet mt-6 p-5 sm:p-7">
          {step === "compose" ? (
            <Compose state={state} onChange={setState} result={result} portfolio={portfolio} assets={assets} showErrors={showErrors} />
          ) : (
            intent && <Review intent={intent} assets={assets} network={network} />
          )}
        </div>

        {error && (
          <p role="alert" className="mt-4 flex items-start gap-3 rounded-field bg-surface p-4 text-[15px] text-flare">
            <IconAlert size={20} className="mt-0.5 shrink-0" /> {error}
          </p>
        )}

        <div className="sticky bottom-[calc(88px+env(safe-area-inset-bottom))] z-20 mt-5 lg:static">
          {step === "compose" ? (
            <Button size="lg" className="w-full" onClick={prepare} disabled={busy}>
              {busy ? "Preparing…" : "Review transfer"}
            </Button>
          ) : (
            <Button size="lg" className="w-full" onClick={sign} disabled={busy}>
              {busy ? "Waiting for wallet…" : "Sign and send"}
            </Button>
          )}
        </div>
      </div>

      <aside className="hidden lg:block">
        <div className="sticky top-28">
          <Clearing stage="idle" legs={legLabels} />
          <p className="mx-auto mt-2 max-w-[34ch] text-center text-[15px] leading-relaxed text-on-canvas-muted">
            {state.legs.length > 1 ? "One signature covers every asset in this transfer." : "You sign once. Validators near your account co-sign and seal it."}
          </p>
        </div>
      </aside>
    </div>
  );
}

function Blocked({ id, state }: { id: string; state: keyof typeof STATE_COPY }) {
  return (
    <div className="pt-4 lg:pt-10">
      <h1 className="type-h1">One moment</h1>
      <div className="sheet mt-6 max-w-[620px] p-6">
        <p className="type-title text-[20px]">Your last transfer isn&apos;t settled yet</p>
        <p className="mt-2 text-[15px] leading-relaxed text-muted">
          {STATE_COPY[state].message} To keep you from paying twice, new transfers wait until it&apos;s settled.
        </p>
        <ButtonLink href={`/activity/${id}`} className="mt-5">
          {isFinalOperationState(state) ? "View transfer" : "Check its status"}
        </ButtonLink>
      </div>
    </div>
  );
}
