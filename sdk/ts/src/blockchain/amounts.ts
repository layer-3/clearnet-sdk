import { ClearnetSdkError } from "../core/errors.js";

export function decimalToBaseUnits(
  amount: unknown,
  decimals: number,
  label = "amount",
): bigint {
  if (!Number.isInteger(decimals) || decimals < 0 || decimals > 255) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      `${label} decimals must be an integer from 0 to 255`,
    );
  }
  if (typeof amount !== "string" || !/^[0-9]+(?:\.[0-9]+)?$/.test(amount)) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      `${label} must be a positive decimal string`,
    );
  }
  const [whole, fractional = ""] = amount.split(".");
  if (fractional.length > decimals) {
    throw new ClearnetSdkError(
      "INVALID_AMOUNT",
      `${label} has more than ${decimals} decimal places`,
    );
  }
  const base = BigInt(
    `${whole}${fractional.padEnd(decimals, "0")}`.replace(/^0+(?=\d)/, ""),
  );
  if (base <= 0n) {
    throw new ClearnetSdkError("INVALID_AMOUNT", `${label} must be positive`);
  }
  return base;
}

export function baseUnitsToDecimal(baseUnits: bigint, decimals: number): string {
  if (decimals === 0) {
    return baseUnits.toString();
  }
  const raw = baseUnits.toString().padStart(decimals + 1, "0");
  const whole = raw.slice(0, -decimals);
  const fractional = raw.slice(-decimals).replace(/0+$/, "");
  return fractional === "" ? whole : `${whole}.${fractional}`;
}
