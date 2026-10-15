// Exact decimal <-> base-unit conversion. Ports client.AmountInBaseUnits and
// the CLI's displayUnits; never uses floating point for amounts.

const MAX_UINT256 = (1n << 256n) - 1n;
const MAX_DECIMALS = 77; // 10^77 fits in uint256.

export class AmountError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "AmountError";
  }
}

/**
 * Converts a human decimal amount ("1.25") to integer base units.
 * Accepts digits with one optional "." separator; no sign, exponent, or grouping.
 */
function checkDecimals(decimals: number) {
  if (!Number.isInteger(decimals) || decimals < 0 || decimals > MAX_DECIMALS) {
    throw new AmountError("Asset decimals must be an integer between 0 and 77.");
  }
}

export function toBaseUnits(amount: string, decimals: number): bigint {
  checkDecimals(decimals);
  if (amount === "") throw new AmountError("Enter an amount.");
  const parts = amount.split(".");
  if (parts.length > 2 || parts[0] === "") throw new AmountError("Enter a number like 12.5.");
  for (const part of parts) {
    if (!/^[0-9]*$/.test(part)) throw new AmountError("Use digits and one decimal point only.");
  }
  const fraction = parts[1];
  if (fraction !== undefined && (fraction.length === 0 || fraction.length > decimals)) {
    throw new AmountError(
      fraction.length === 0
        ? "Add digits after the decimal point."
        : `This asset supports up to ${decimals} decimal places.`,
    );
  }
  const base = 10n ** BigInt(decimals);
  let result = BigInt(parts[0]!) * base;
  if (fraction !== undefined) {
    result += BigInt(fraction) * 10n ** BigInt(decimals - fraction.length);
  }
  if (result > MAX_UINT256) throw new AmountError("Amount is too large.");
  return result;
}

/**
 * Normalizes what a person types into the strict form toBaseUnits accepts:
 * trims spaces, turns a single comma into a decimal point, drops thousands
 * separators when a decimal point is also present, and adds a leading zero.
 */
export function normalizeAmountInput(raw: string): string {
  let s = raw.trim().replace(/[\s_\u00a0']/g, "");
  if (s.includes(",") && s.includes(".")) s = s.replace(/,/g, "");
  else if ((s.match(/,/g) ?? []).length === 1) s = s.replace(",", ".");
  if (s.startsWith(".")) s = `0${s}`;
  return s;
}

/** Exact decimal string with trailing zeros trimmed (CLI displayUnits). */
export function fromBaseUnits(raw: string | bigint, decimals: number): string {
  checkDecimals(decimals);
  let v: bigint;
  try {
    v = typeof raw === "bigint" ? raw : BigInt(raw);
  } catch {
    throw new AmountError("Invalid base-unit amount.");
  }
  if (v < 0n) throw new AmountError("Invalid base-unit amount.");
  let digits = v.toString();
  if (decimals === 0) return digits;
  if (digits.length <= decimals) digits = "0".repeat(decimals - digits.length + 1) + digits;
  const whole = digits.slice(0, digits.length - decimals);
  const fraction = digits.slice(digits.length - decimals).replace(/0+$/, "");
  return fraction ? `${whole}.${fraction}` : whole;
}

/** Strict integer base-unit parse (gateway wire values). */
export function parseBaseUnitInteger(raw: string): bigint {
  if (!/^[0-9]+$/.test(raw)) throw new AmountError("Integer must contain only decimal digits.");
  const v = BigInt(raw);
  if (v > MAX_UINT256) throw new AmountError("Integer exceeds uint256.");
  return v;
}

export interface DisplayParts {
  whole: string;
  fraction: string;
  /** True when digits were cut for display. The exact value stays in base units. */
  truncated: boolean;
}

/**
 * Splits an exact decimal string for display with grouping and at most
 * `maxFraction` fractional digits. Truncates (never rounds up) so a displayed
 * balance is never more than what is held.
 */
export function displayParts(exact: string, maxFraction = 6): DisplayParts {
  const [w = "0", f = ""] = exact.split(".");
  const whole = w.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  let fraction = f.slice(0, maxFraction).replace(/0+$/, "");
  const truncated = f.length > maxFraction && /[1-9]/.test(f.slice(maxFraction));
  if (!fraction && truncated && w === "0") {
    // Tiny non-zero amount: show its first significant digits instead of 0.
    const firstSig = f.search(/[1-9]/);
    fraction = f.slice(0, firstSig + 2).replace(/0+$/, "");
  }
  return { whole, fraction, truncated };
}

export function formatAmount(exact: string, maxFraction = 6): string {
  const p = displayParts(exact, maxFraction);
  return p.fraction ? `${p.whole}.${p.fraction}` : p.whole;
}

export function formatBaseUnits(raw: string | bigint, decimals: number, maxFraction = 6): string {
  return formatAmount(fromBaseUnits(raw, decimals), maxFraction);
}
