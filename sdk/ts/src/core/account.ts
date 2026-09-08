import { hexToBytes } from "./bytes.js";
import { ClearnetSdkError } from "./errors.js";

/**
 * Parses a clearnet account (ADR-023) — the 20-byte address every chain
 * depositor's `destination.account` carries:
 *
 *   - a bare 40-character hex string, with or without a "0x"/"0X" prefix
 *   - the same, surrounded by whitespace
 *   - a yellow://ynet/user/<hex> URI (any case); the last "/"-delimited
 *     segment is taken as the address
 *
 * An ADR-015 sub-account URI (yellow://.../user/<addr>/tag/<ref>) is
 * rejected — its last segment is the 32-byte reference, not the address.
 */
export function parseClearnetAccount(account: unknown): Uint8Array {
  if (typeof account !== "string") {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "destination.account must be a 20-byte hex address",
    );
  }
  const trimmed = account.trim();
  const segment = trimmed.slice(trimmed.lastIndexOf("/") + 1);
  const hex = segment.toLowerCase().replace(/^0x/, "");
  if (!/^[a-f0-9]{40}$/.test(hex)) {
    const hint = trimmed.includes("/tag/")
      ? " (a yellow://.../user/<addr>/tag/<ref> URI must be split: pass <addr> as destination.account and <ref> as destination.ref)"
      : "";
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `destination.account must be a 20-byte hex address${hint}`,
    );
  }
  return hexToBytes(hex, "destination.account");
}
