export const SOLANA_CUSTODY_PROGRAM_ID =
  "98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg";

export const SOLANA_NATIVE_ASSET = "";

export const SOLANA_SYSTEM_PROGRAM_ID =
  "11111111111111111111111111111111";

export const SOLANA_TOKEN_PROGRAM_ID =
  "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA";

export const SOLANA_ASSOCIATED_TOKEN_PROGRAM_ID =
  "ATokenGPvbdGVxr1b2hvZbsiqW5xWH25efTNsLJA8knL";

export const DEFAULT_SOLANA_COMMITMENT = "finalized";

export const DEFAULT_RECEIPT_TIMEOUT_MS = 60_000;

export const POLL_INTERVAL_MS = 250;

export const DEPOSIT_SOL_DISCRIMINATOR = [
  108, 81, 78, 117, 125, 155, 56, 200,
] as const;

export const DEPOSIT_SPL_DISCRIMINATOR = [
  224, 0, 198, 175, 198, 47, 105, 204,
] as const;

export const DEPOSITED_EVENT_DISCRIMINATOR = [
  111, 141, 26, 45, 161, 35, 100, 57,
] as const;

export const EXECUTED_EVENT_DISCRIMINATOR = [
  8, 232, 139, 132, 197, 45, 29, 164,
] as const;

// anchor_lang's EVENT_IX_TAG_LE: the 8-byte prefix of the self-CPI instruction
// data emit_cpi! produces.
export const EVENT_IX_TAG = [
  0xe4, 0x45, 0xa5, 0x2e, 0x51, 0xcb, 0x9a, 0x1d,
] as const;

// Event body lengths below which a Deposited or Executed event does not decode
// and the CPI is not counted as an event.
export const DEPOSITED_EVENT_MIN_LEN = 32 + 20 + 32 + 32 + 8;
export const EXECUTED_EVENT_MIN_LEN = 32 + 32 + 32 + 8;
