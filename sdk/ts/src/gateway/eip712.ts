// Clearnet EIP-712 Transfer schema, operation IDs, and recipient rules.
// Byte-for-byte compatible with pkg/eip712 (see the golden vector test).

import {
  concat,
  getAddress,
  hashTypedData,
  isAddress,
  keccak256,
  recoverTypedDataAddress,
  stringToBytes,
  toHex,
  zeroAddress,
  type Hex,
} from "viem";
import type { TransferAsset, TransferIntent } from "./types.js";
import { parseBaseUnitInteger } from "./amounts.js";

export const VIRTUAL_ROUTER = "0x00000000000000000000000000000000434C5200" as const;
export const TRANSFER_OPERATION_DOMAIN = "clearnet/transfer-operation/v1";
export const USER_URI_PREFIX = "yellow://ynet/user/";

export const TRANSFER_TYPES = {
  Transfer: [
    { name: "to", type: "address" },
    { name: "assets", type: "TransferAsset[]" },
    { name: "maxFee", type: "uint256" },
    { name: "nonce", type: "uint64" },
  ],
  TransferAsset: [
    { name: "asset", type: "string" },
    { name: "amount", type: "uint256" },
  ],
} as const;

/** Big-endian uint32 of the four ASCII bytes of a network ID ("YDEV" -> 1497646422). */
export function networkChainId(networkId: string): number {
  if (networkId.length !== 4) throw new Error("Clearnet network ID must be four characters.");
  let id = 0;
  for (let i = 0; i < 4; i++) {
    const c = networkId.charCodeAt(i);
    if (c > 0xff) throw new Error("Clearnet network ID must be ASCII.");
    id = id * 256 + c;
  }
  if (id === 0) throw new Error("Clearnet network ID must be nonzero.");
  return id;
}

export function clearnetDomain(chainId: number): {
  readonly name: "Clearnet";
  readonly version: "1";
  readonly chainId: number;
  readonly verifyingContract: `0x${string}`;
} {
  return {
    name: "Clearnet",
    version: "1",
    chainId,
    // The EIP-55 checksum of the router is all-lowercase.
    verifyingContract: "0x00000000000000000000000000000000434c5200",
  };
}

export class RecipientError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "RecipientError";
  }
}

/**
 * Resolves what someone pastes as a recipient to a checksummed user address.
 * Accepts a 0x address or yellow://ynet/user/<address>. Like the CLI, only user
 * accounts are accepted and the zero address is refused.
 */
export function resolveRecipient(raw: string): `0x${string}` {
  let value = raw.trim();
  if (value.includes("://")) {
    const lower = value.toLowerCase();
    if (!lower.startsWith("yellow://")) throw new RecipientError("Use a 0x address or a yellow://ynet/user/ link.");
    if (!lower.startsWith(USER_URI_PREFIX)) throw new RecipientError("You can only send to a user account.");
    value = lower.slice(USER_URI_PREFIX.length);
    if (value.includes("/")) throw new RecipientError("Sub-account links aren't supported yet. Use the account address.");
  }
  if (!/^0x[0-9a-fA-F]{40}$/.test(value)) throw new RecipientError("That isn't a full Ethereum address. It should be 0x followed by 40 characters.");
  // Mixed-case input must carry a valid EIP-55 checksum.
  if (!isAddress(value, { strict: true }) && value !== value.toLowerCase() && value.slice(2) !== value.slice(2).toUpperCase()) {
    throw new RecipientError("The address checksum doesn't match. Check for a typo.");
  }
  const address = getAddress(value);
  if (address === zeroAddress) throw new RecipientError("You can't send to the zero address.");
  return address;
}

export function sortTransferAssets(assets: TransferAsset[]): TransferAsset[] {
  return [...assets].sort((a, b) => (a.asset < b.asset ? -1 : a.asset > b.asset ? 1 : 0));
}

function message(intent: Pick<TransferIntent, "to" | "assets" | "maxFee" | "nonce">) {
  if (intent.assets.length === 0) throw new Error("Transfer assets required.");
  intent.assets.forEach((leg, i) => {
    const prev = intent.assets[i - 1];
    if (prev && prev.asset >= leg.asset) throw new Error("Transfer assets must be sorted and unique.");
    if (!leg.asset || parseBaseUnitInteger(leg.amount) <= 0n) throw new Error(`Invalid transfer asset ${i} amount.`);
  });
  if (!Number.isSafeInteger(intent.nonce) || intent.nonce < 0) throw new Error("Invalid nonce.");
  return {
    to: resolveRecipient(intent.to),
    assets: intent.assets.map((a) => ({ asset: a.asset, amount: parseBaseUnitInteger(a.amount) })),
    maxFee: parseBaseUnitInteger(intent.maxFee),
    nonce: BigInt(intent.nonce),
  };
}

export interface TransferTypedData {
  domain: ReturnType<typeof clearnetDomain>;
  types: typeof TRANSFER_TYPES;
  primaryType: "Transfer";
  message: {
    to: `0x${string}`;
    assets: { asset: string; amount: bigint }[];
    maxFee: bigint;
    nonce: bigint;
  };
}

/** Typed data ready for a wallet's signTypedData (eth_signTypedData_v4). */
export function transferTypedData(intent: TransferIntent): TransferTypedData {
  return {
    domain: clearnetDomain(intent.chainId),
    types: TRANSFER_TYPES,
    primaryType: "Transfer",
    message: message(intent),
  };
}

export function transferDigest(intent: TransferIntent): Hex {
  return hashTypedData(transferTypedData(intent));
}

/** keccak256("clearnet/transfer-operation/v1" || signer20 || digest32), lowercase. */
export function transferOperationId(intent: TransferIntent): `0x${string}` {
  const preimage = concat([toHex(stringToBytes(TRANSFER_OPERATION_DOMAIN)), getAddress(intent.from), transferDigest(intent)]);
  return keccak256(preimage).toLowerCase() as `0x${string}`;
}

export function validOperationId(id: string): id is `0x${string}` {
  return /^0x[0-9a-f]{64}$/.test(id);
}

export class SignatureError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "SignatureError";
  }
}

/** Canonical 65-byte R||S||V with V in {27,28}. */
export function normalizeSignature(signature: string): `0x${string}` {
  if (!/^0x[0-9a-fA-F]*$/.test(signature)) throw new SignatureError("The wallet returned a malformed signature.");
  if (signature.length !== 132) {
    throw new SignatureError(
      "This wallet returned a smart-contract signature. Yellow needs a regular account signature, so use an externally owned account (for example MetaMask, Rabby or a hardware wallet).",
    );
  }
  const v = parseInt(signature.slice(130), 16);
  const canonicalV = v < 27 ? v + 27 : v;
  if (canonicalV !== 27 && canonicalV !== 28) throw new SignatureError("The wallet returned an invalid signature.");
  return `${signature.slice(0, 130).toLowerCase()}${canonicalV.toString(16)}` as `0x${string}`;
}

export async function recoverTransferSigner(intent: TransferIntent, signature: `0x${string}`) {
  return recoverTypedDataAddress({ ...transferTypedData(intent), signature });
}
