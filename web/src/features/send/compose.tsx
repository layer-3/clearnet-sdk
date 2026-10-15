"use client";

import { fromBaseUnits, normalizeAmountInput, toBaseUnits, type AssetInfo } from "@yellow-org/clearnet-sdk/gateway";
import { clsx } from "clsx";
import { useId, useState } from "react";
import { AssetIcon } from "@/components/ui/asset-icon";
import { IconChevronDown, IconClose, IconPaste, IconPlus } from "@/components/ui/icons";
import { Identicon } from "@/components/ui/identicon";
import { Amount, ASSET_NAMES } from "@/components/ui/misc";
import { Sheet } from "@/components/ui/sheet";
import { formatUnits, shortAddress } from "@/lib/format";
import { YELLOW, type Portfolio } from "@/lib/portfolio";
import { maxSendable, newLeg, type ComposeResult, type ComposeState, type LegInput } from "./model";

interface Props {
  state: ComposeState;
  onChange: (next: ComposeState) => void;
  result: ComposeResult;
  portfolio: Portfolio;
  assets: AssetInfo[];
  showErrors: boolean;
}

export function Compose({ state, onChange, result, portfolio, assets, showErrors }: Props) {
  const yellow = assets.find((a) => a.symbol === YELLOW);
  const capUnits = (() => {
    try {
      return yellow ? toBaseUnits(normalizeAmountInput(result.feeCap), yellow.decimals) : 0n;
    } catch {
      return 0n;
    }
  })();
  const unused = portfolio.holdings.filter((h) => !state.legs.some((l) => l.asset === h.asset.symbol));
  const setLeg = (id: string, patch: Partial<LegInput>) => onChange({ ...state, legs: state.legs.map((l) => (l.id === id ? { ...l, ...patch } : l)) });

  return (
    <div className="space-y-5">
      <RecipientField value={state.recipient} onChange={(recipient) => onChange({ ...state, recipient })} result={result} showErrors={showErrors} />

      <fieldset className="space-y-3">
        <legend className="type-label mb-2 text-[15px]">Amount</legend>
        {state.legs.map((leg) => (
          <LegRow
            key={leg.id}
            leg={leg}
            portfolio={portfolio}
            assets={assets}
            taken={state.legs.filter((l) => l.id !== leg.id).map((l) => l.asset)}
            error={result.legErrors[leg.id] && (showErrors || leg.amount) ? result.legErrors[leg.id] : undefined}
            missing={showErrors && !leg.amount}
            onChange={(patch) => setLeg(leg.id, patch)}
            onRemove={state.legs.length > 1 ? () => onChange({ ...state, legs: state.legs.filter((l) => l.id !== leg.id) }) : undefined}
            onMax={() => {
              const meta = assets.find((a) => a.symbol === leg.asset);
              if (meta) setLeg(leg.id, { amount: fromBaseUnits(maxSendable(meta, portfolio, capUnits), meta.decimals) });
            }}
          />
        ))}
        {unused.length > 0 && (
          <button
            type="button"
            onClick={() => onChange({ ...state, legs: [...state.legs, newLeg(unused[0]!.asset.symbol)] })}
            className="type-label flex h-14 w-full items-center justify-center gap-2 rounded-field border-2 border-dashed border-line text-[15px] text-muted transition-colors hover:border-current hover:text-on-surface"
          >
            <IconPlus size={18} /> Add another asset
          </button>
        )}
        {state.legs.length > 1 && <p className="text-[13.5px] text-muted">All assets go to the same recipient in one transfer, with one signature.</p>}
      </fieldset>

      <FeeField result={result} yellow={yellow} onChange={(feeCap) => onChange({ ...state, feeCap })} />
    </div>
  );
}

function RecipientField({ value, onChange, result, showErrors }: { value: string; onChange: (v: string) => void; result: ComposeResult; showErrors: boolean }) {
  const id = useId();
  const [touched, setTouched] = useState(false);
  const error = result.recipientError && (touched || showErrors) ? result.recipientError : showErrors && !value.trim() ? "Enter who you're sending to." : undefined;
  const canPaste = typeof navigator !== "undefined" && !!navigator.clipboard?.readText;
  return (
    <div>
      <label htmlFor={id} className="type-label mb-2 block text-[15px]">
        To
      </label>
      <div className={clsx("flex items-center gap-2 rounded-field bg-surface-2 pl-4 pr-2 shadow-[inset_0_0_0_1.5px_transparent] transition-shadow focus-within:shadow-[inset_0_0_0_2px_var(--on-surface)]", error && "!shadow-[inset_0_0_0_2px_var(--flare)]")}>
        {result.recipient && !result.recipientError ? <Identicon address={result.recipient} size={26} /> : null}
        <input
          id={id}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onBlur={() => setTouched(true)}
          placeholder="0x… address or yellow:// link"
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck={false}
          inputMode="text"
          aria-invalid={!!error}
          aria-describedby={`${id}-hint`}
          className="h-14 min-w-0 flex-1 bg-transparent font-mono text-[15px] outline-none placeholder:font-sans placeholder:text-muted"
        />
        {value ? (
          <button type="button" onClick={() => onChange("")} className="grid size-10 place-items-center rounded-control text-muted hover:bg-line" aria-label="Clear recipient">
            <IconClose size={18} />
          </button>
        ) : canPaste ? (
          <button
            type="button"
            onClick={async () => {
              try {
                onChange((await navigator.clipboard.readText()).trim());
                setTouched(true);
              } catch {
                /* clipboard permission denied */
              }
            }}
            className="type-label inline-flex h-10 items-center gap-1.5 rounded-control border border-line bg-surface px-3 text-[13px] hover:bg-line"
          >
            <IconPaste size={16} /> Paste
          </button>
        ) : null}
      </div>
      <p id={`${id}-hint`} className={clsx("mt-2 min-h-5 text-[13.5px]", error ? "text-flare" : "text-muted")} aria-live="polite">
        {error ?? (result.recipient ? <>Sending to <span className="font-mono">{shortAddress(result.recipient, 8, 6)}</span>. Check it matches.</> : "Yellow accounts use Ethereum-style addresses.")}
      </p>
    </div>
  );
}

function LegRow({
  leg,
  portfolio,
  assets,
  taken,
  error,
  missing,
  onChange,
  onRemove,
  onMax,
}: {
  leg: LegInput;
  portfolio: Portfolio;
  assets: AssetInfo[];
  taken: string[];
  error?: string;
  missing: boolean;
  onChange: (p: Partial<LegInput>) => void;
  onRemove?: () => void;
  onMax: () => void;
}) {
  const id = useId();
  const [picking, setPicking] = useState(false);
  const meta = assets.find((a) => a.symbol === leg.asset);
  const holding = portfolio.holdings.find((h) => h.asset.symbol === leg.asset);
  return (
    <div className={clsx("rounded-field bg-surface-2 p-4 transition-shadow focus-within:shadow-[inset_0_0_0_2px_var(--on-surface)]", (error || missing) && "!shadow-[inset_0_0_0_2px_var(--flare)]")}>
      <div className="flex items-center gap-2">
        <button type="button" onClick={() => setPicking(true)} className="inline-flex h-11 items-center gap-2 rounded-control border border-line bg-surface py-1 pl-1 pr-3 transition-colors hover:border-on-surface" aria-label={`Asset: ${leg.asset}. Change asset`}>
          <AssetIcon symbol={leg.asset} size={34} />
          <span className="type-label text-[16px]">{leg.asset}</span>
          <IconChevronDown size={16} className="text-muted" />
        </button>
        <div className="flex-1" />
        <button type="button" onClick={onMax} className="type-label h-9 rounded-control px-3 text-[13px] hover:bg-line">
          Max
        </button>
        {onRemove && (
          <button type="button" onClick={onRemove} className="grid size-9 place-items-center rounded-control text-muted hover:bg-line" aria-label={`Remove ${leg.asset}`}>
            <IconClose size={17} />
          </button>
        )}
      </div>
      <label htmlFor={id} className="sr-only">
        {leg.asset} amount
      </label>
      <input
        id={id}
        value={leg.amount}
        onChange={(e) => onChange({ amount: e.target.value.replace(/[^\d.,\s]/g, "") })}
        inputMode="decimal"
        autoComplete="off"
        placeholder="0"
        aria-invalid={!!error}
        aria-describedby={`${id}-hint`}
        className="type-amount mt-3 block w-full bg-transparent outline-none placeholder:text-current placeholder:opacity-20"
      />
      <p id={`${id}-hint`} className={clsx("mt-2 text-[13.5px]", error ? "text-flare" : "text-muted")} aria-live="polite">
        {error ?? (meta && holding ? <>Available <Amount raw={holding.available} decimals={meta.decimals} fractionClass="" /> {leg.asset}</> : null)}
      </p>
      <Sheet open={picking} onOpenChange={setPicking} title="Choose asset">
        <ul className="space-y-1">
          {portfolio.holdings.map((h) => {
            const disabled = taken.includes(h.asset.symbol);
            return (
              <li key={h.asset.symbol}>
                <button
                  type="button"
                  disabled={disabled}
                  onClick={() => {
                    onChange({ asset: h.asset.symbol, amount: "" });
                    setPicking(false);
                  }}
                  className={clsx("flex w-full items-center gap-4 rounded-field p-3 text-left transition-colors hover:bg-surface-2 disabled:opacity-40", h.asset.symbol === leg.asset && "bg-surface-2")}
                >
                  <AssetIcon symbol={h.asset.symbol} size={42} />
                  <span className="flex-1">
                    <span className="type-label block text-[16px]">{h.asset.symbol}</span>
                    <span className="block text-[13px] text-muted">{disabled ? "Already in this transfer" : (ASSET_NAMES[h.asset.symbol] ?? h.asset.symbol)}</span>
                  </span>
                  <span className="type-label num text-[16px]">{formatUnits(h.available, h.asset.decimals, 6)}</span>
                </button>
              </li>
            );
          })}
        </ul>
      </Sheet>
    </div>
  );
}

function FeeField({ result, yellow, onChange }: { result: ComposeResult; yellow: AssetInfo | undefined; onChange: (v: string | null) => void }) {
  const id = useId();
  const [editing, setEditing] = useState(false);
  return (
    <div className="rounded-field shadow-[inset_0_0_0_1.5px_var(--line)] p-4">
      <div className="flex items-baseline justify-between gap-4">
        <span className="type-label text-[15px]">Network fee</span>
        <span className="num text-[15px]">{result.fee !== null && yellow ? `≈ ${formatUnits(result.fee, yellow.decimals, 6)} YELLOW` : "Estimated at review"}</span>
      </div>
      <div className="mt-2 flex items-center justify-between gap-4">
        <label htmlFor={id} className="shrink-0 whitespace-nowrap text-[14px] text-muted">
          Most you&apos;ll pay
        </label>
        {editing ? (
          <span className="flex items-center gap-2">
            <input
              id={id}
              autoFocus
              value={result.feeCap}
              onChange={(e) => onChange(e.target.value.replace(/[^\d.,]/g, ""))}
              inputMode="decimal"
              className="num h-10 w-28 rounded-control bg-surface-2 px-3 text-right text-[15px] outline-none focus:shadow-[inset_0_0_0_2px_var(--on-surface)]"
              aria-invalid={!!result.feeCapError}
            />
            <span className="text-[14px]">YELLOW</span>
          </span>
        ) : (
          <button type="button" onClick={() => setEditing(true)} aria-label={`Fee cap ${result.feeCap} YELLOW. Change fee cap`} className="num inline-flex items-center gap-2 rounded-control px-2 py-1 text-[15px] hover:bg-surface-2">
            {result.feeCap} YELLOW <span className="type-label text-[13px] underline underline-offset-2">Change</span>
          </button>
        )}
      </div>
      {editing && (
        <button type="button" className="type-label mt-1 text-[13px] text-muted underline underline-offset-2" onClick={() => { onChange(null); setEditing(false); }}>
          Use suggested cap
        </button>
      )}
      <p className={clsx("mt-3 text-[13.5px] leading-relaxed", result.feeCapError ? "text-flare" : "text-muted")} aria-live="polite">
        {result.feeCapError ?? "Fees are 0.10% of the value, paid in YELLOW. You never pay more than your cap."}
      </p>
    </div>
  );
}

