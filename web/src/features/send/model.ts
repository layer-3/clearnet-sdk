import {
  AmountError,
  normalizeAmountInput,
  resolveRecipient,
  toBaseUnits,
  type AssetInfo,
  type PoolSummary,
  type TransferAsset,
} from "@yellow-org/clearnet-sdk/gateway";
import { formatUnits } from "@/lib/format";
import { estimateFee, YELLOW, type Portfolio } from "@/lib/portfolio";

export interface LegInput {
  id: string;
  asset: string;
  amount: string;
}

export interface ComposeState {
  recipient: string;
  legs: LegInput[];
  /** null = use the suggested cap */
  feeCap: string | null;
}

export interface ComposeResult {
  recipient?: `0x${string}`;
  recipientError?: string;
  legErrors: Record<string, string>;
  legs: TransferAsset[];
  fee: bigint | null;
  feeCap: string;
  feeCapError?: string;
  formError?: string;
  ready: boolean;
}

let counter = 0;
export function newLeg(asset: string): LegInput {
  counter += 1;
  return { id: `leg-${counter}`, asset, amount: "" };
}

export function validate(state: ComposeState, portfolio: Portfolio, assets: AssetInfo[], pools: PoolSummary[] | undefined, self: string): ComposeResult {
  const out: ComposeResult = { legErrors: {}, legs: [], fee: null, feeCap: "", ready: false };

  if (state.recipient.trim()) {
    try {
      out.recipient = resolveRecipient(state.recipient);
      if (out.recipient.toLowerCase() === self.toLowerCase()) out.recipientError = "That's your own account. Enter someone else's address.";
    } catch (e) {
      out.recipientError = e instanceof Error ? e.message : "Enter a valid address.";
    }
  }

  const held = new Map(portfolio.holdings.map((h) => [h.asset.symbol, h]));
  for (const leg of state.legs) {
    const meta = assets.find((a) => a.symbol === leg.asset);
    if (!meta) {
      out.legErrors[leg.id] = "Choose an asset.";
      continue;
    }
    const raw = normalizeAmountInput(leg.amount);
    if (!raw) continue;
    try {
      const units = toBaseUnits(raw, meta.decimals);
      if (units === 0n) {
        out.legErrors[leg.id] = "Enter an amount above zero.";
        continue;
      }
      const available = held.get(leg.asset)?.available ?? 0n;
      if (units > available) {
        out.legErrors[leg.id] = `You have ${formatUnits(available, meta.decimals)} ${leg.asset}.`;
        continue;
      }
      out.legs.push({ asset: leg.asset, amount: units.toString() });
    } catch (e) {
      out.legErrors[leg.id] = e instanceof AmountError ? e.message : "Enter a valid amount.";
    }
  }

  const estimate = estimateFee(out.legs, assets, pools);
  out.fee = estimate.fee;
  out.feeCap = state.feeCap ?? estimate.suggestedCap;
  const yellow = assets.find((a) => a.symbol === YELLOW);
  let capUnits = 0n;
  if (!yellow) out.formError = "This network has no YELLOW for fees, so sending is unavailable.";
  else {
    try {
      capUnits = toBaseUnits(normalizeAmountInput(out.feeCap), yellow.decimals);
      if (out.fee !== null && capUnits < out.fee) out.feeCapError = `The estimated fee is ${formatUnits(out.fee, yellow.decimals)} YELLOW. A lower cap will be rejected.`;
    } catch (e) {
      out.feeCapError = e instanceof Error ? e.message : "Enter a fee cap.";
    }
  }

  // Clearnet requires the 10 YELLOW user floor after principal and fees.
  if (yellow && !out.feeCapError) {
    const sending = BigInt(out.legs.find((l) => l.asset === YELLOW)?.amount ?? "0");
    const available = held.get(YELLOW)?.available ?? 0n;
    const floor = 10n * 10n ** BigInt(yellow.decimals);
    if (sending + capUnits + floor > available) {
      const leg = state.legs.find((l) => l.asset === YELLOW);
      const room = available > capUnits + floor ? available - capUnits - floor : 0n;
      const message = `Leave room for the fee and the 10 YELLOW minimum. You can send up to ${formatUnits(room, yellow.decimals)} YELLOW.`;
      if (leg && sending > 0n) out.legErrors[leg.id] = message;
      else out.feeCapError = `Keep 10 YELLOW after the fee. Your fee cap is ${formatUnits(capUnits, yellow.decimals)} YELLOW.`;
    }
  }

  if (!portfolio.activated) out.formError = "Your account needs at least 10 YELLOW available before it can send.";
  out.ready =
    !!out.recipient &&
    !out.recipientError &&
    out.legs.length === state.legs.length &&
    out.legs.length > 0 &&
    Object.keys(out.legErrors).length === 0 &&
    !out.feeCapError &&
    !out.formError;
  return out;
}

/** The most you can send, retaining the user floor and fee cap for YELLOW. */
export function maxSendable(asset: AssetInfo, portfolio: Portfolio, feeCapUnits: bigint): bigint {
  const available = portfolio.holdings.find((h) => h.asset.symbol === asset.symbol)?.available ?? 0n;
  if (asset.symbol !== YELLOW) return available;
  const reserved = feeCapUnits + 10n * 10n ** BigInt(asset.decimals);
  return available > reserved ? available - reserved : 0n;
}
