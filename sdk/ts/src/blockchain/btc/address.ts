import { p2sh, p2wpkh, p2wsh, Script } from "@scure/btc-signer";
import { sha256 } from "@noble/hashes/sha2.js";
import { Point } from "@noble/secp256k1";

import { compareBytes, hexToBytes } from "../../core/bytes.js";
import { ClearnetSdkError } from "../../core/errors.js";
import { BITCOIN_MAX_VAULT_PUBKEYS } from "./constants.js";
import { networkParams } from "./networks.js";
import type {
  BitcoinNetwork,
  BitcoinWalletAddressType,
} from "./types.js";

const TEXT_ENCODER = new TextEncoder();

export function accountTag(account: string): Uint8Array {
  return sha256(TEXT_ENCODER.encode(account));
}

export function requireCompressedPublicKey(
  value: Uint8Array | string,
  field: string,
): Uint8Array {
  const bytes =
    typeof value === "string"
      ? hexToBytes(value.replace(/^0x/i, ""), field)
      : Uint8Array.from(value);
  if (
    bytes.length !== 33 ||
    (bytes[0] !== 0x02 && bytes[0] !== 0x03)
  ) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `${field} must be a compressed secp256k1 public key`,
    );
  }
  try {
    Point.fromBytes(bytes);
  } catch (error) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `${field} must be a valid secp256k1 public key`,
      { cause: error },
    );
  }
  return bytes;
}

export function normalizeVaultPubkeys(
  pubkeys: readonly (Uint8Array | string)[],
): Uint8Array[] {
  if (!Array.isArray(pubkeys) || pubkeys.length === 0) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      "vaultPubkeys must contain at least one public key",
    );
  }
  if (pubkeys.length > BITCOIN_MAX_VAULT_PUBKEYS) {
    throw new ClearnetSdkError(
      "INVALID_ADDRESS",
      `vaultPubkeys must contain at most ${BITCOIN_MAX_VAULT_PUBKEYS} keys`,
    );
  }
  return pubkeys.map((pubkey, index) =>
    requireCompressedPublicKey(pubkey, `vaultPubkeys[${index}]`),
  );
}

export function sortedPubkeys(pubkeys: readonly Uint8Array[]): Uint8Array[] {
  return [...pubkeys].sort(compareBytes);
}

export function taggedRedeemScript(
  account: string,
  threshold: number,
  pubkeys: readonly Uint8Array[],
): Uint8Array {
  return Script.encode([
    accountTag(account),
    "DROP",
    threshold,
    ...sortedPubkeys(pubkeys),
    pubkeys.length,
    "CHECKMULTISIG",
  ]);
}

export function depositPayment(
  network: BitcoinNetwork,
  account: string,
  threshold: number,
  pubkeys: readonly Uint8Array[],
) {
  const script = taggedRedeemScript(account, threshold, pubkeys);
  return p2wsh({ type: "ms", script }, networkParams(network));
}

export function depositAddress(
  network: BitcoinNetwork,
  account: string,
  threshold: number,
  pubkeys: readonly Uint8Array[],
): string {
  const address = depositPayment(network, account, threshold, pubkeys).address;
  if (address === undefined) {
    throw new ClearnetSdkError("INVALID_ADDRESS", "failed to derive Bitcoin deposit address");
  }
  return address;
}

export interface FundingPayment {
  address: string;
  script: Uint8Array;
  pubkeyHash: Uint8Array;
  redeemScript?: Uint8Array;
}

export function fundingPayment(
  network: BitcoinNetwork,
  publicKey: Uint8Array,
  addressType: BitcoinWalletAddressType = "p2wpkh",
): FundingPayment {
  const native = p2wpkh(publicKey, networkParams(network));
  if (addressType === "p2wpkh") {
    return {
      address: native.address,
      script: native.script,
      pubkeyHash: native.hash,
    };
  }
  const nested = p2sh(native, networkParams(network));
  return {
    address: nested.address,
    script: nested.script,
    pubkeyHash: native.hash,
    redeemScript: nested.redeemScript,
  };
}

export function fundingAddress(
  network: BitcoinNetwork,
  publicKey: Uint8Array,
): string {
  return fundingPayment(network, publicKey).address;
}
