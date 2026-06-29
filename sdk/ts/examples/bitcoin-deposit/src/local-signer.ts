import {
  pubECDSA,
  randomPrivateKeyBytes,
  signECDSA,
} from "@scure/btc-signer/utils.js";
import type { BitcoinSigner } from "@yellow-org/clearnet-sdk";

export interface LocalBitcoinSignerResult {
  signer: BitcoinSigner;
  privateKeyHex: string;
  publicKeyHex: string;
}

class LocalBitcoinSigner implements BitcoinSigner {
  readonly algorithm = "secp256k1";

  constructor(private readonly privateKey: Uint8Array) {}

  getPublicKeyCompressed(): Uint8Array {
    return pubECDSA(this.privateKey, true);
  }

  signDigest32(digest: Uint8Array): Uint8Array {
    return signECDSA(digest, this.privateKey);
  }
}

export function createLocalBitcoinSigner(
  privateKeyHex?: string,
): LocalBitcoinSignerResult {
  const privateKey =
    privateKeyHex === undefined || privateKeyHex === ""
      ? randomPrivateKeyBytes()
      : hexToBytes(privateKeyHex, "private key");
  const signer = new LocalBitcoinSigner(privateKey);
  const publicKey = signer.getPublicKeyCompressed();
  return {
    signer,
    privateKeyHex: bytesToHex(privateKey),
    publicKeyHex: bytesToHex(publicKey),
  };
}

export function generateVaultPubkeys(count: number): readonly string[] {
  return Array.from({ length: count }, () => createLocalBitcoinSigner().publicKeyHex);
}

export function bytesToHex(bytes: Uint8Array): string {
  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("");
}

function hexToBytes(value: string, label: string): Uint8Array {
  const normalized = value.trim();
  if (!/^(?:[a-fA-F0-9]{2})+$/.test(normalized)) {
    throw new Error(`${label} must be even-length hex`);
  }
  return Uint8Array.from(
    normalized.match(/../g)?.map((byte) => Number.parseInt(byte, 16)) ?? [],
  );
}
