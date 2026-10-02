const DECIMAL_PATTERN = /^(?:0|[1-9][0-9]*)$/;

/**
 * Returns n when depositId is exactly `${prefix}:${n}`, with n in canonical
 * decimal (no sign, no leading zeros) and n <= max; otherwise undefined.
 */
export function parseDepositIdIndex(
  depositId: string,
  prefix: string,
  max: bigint,
): bigint | undefined {
  const head = `${prefix}:`;
  if (!depositId.startsWith(head)) {
    return undefined;
  }
  const digits = depositId.slice(head.length);
  if (!DECIMAL_PATTERN.test(digits)) {
    return undefined;
  }
  const n = BigInt(digits);
  return n <= max ? n : undefined;
}
