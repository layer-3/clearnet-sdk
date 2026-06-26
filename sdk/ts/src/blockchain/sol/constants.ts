export const SOLANA_CUSTODY_PROGRAM_ID =
  "98eVpih8X9CAcgU9bzNB9V7VtkRrnFZUmqzEnsq7cfmg";

export const SOLANA_NATIVE_ASSET = "SOL";

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
