import { Buffer } from "buffer";

export function bytesToHex(bytes: Uint8Array): string {
  return Buffer.from(bytes).toString("hex");
}

export function hexToBytes(hex: string, field: string): Uint8Array {
  if (!/^(?:[a-fA-F0-9]{2})*$/.test(hex)) {
    throw new Error(`${field} must be even-length hex`);
  }
  return Uint8Array.from(Buffer.from(hex, "hex"));
}

export function concatBytes(...chunks: readonly Uint8Array[]): Uint8Array {
  const length = chunks.reduce((total, chunk) => total + chunk.length, 0);
  const out = new Uint8Array(length);
  let offset = 0;
  for (const chunk of chunks) {
    out.set(chunk, offset);
    offset += chunk.length;
  }
  return out;
}

export function reverseBytes(bytes: Uint8Array): Uint8Array {
  return Uint8Array.from([...bytes].reverse());
}

export function equalBytes(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) {
    return false;
  }
  for (let i = 0; i < a.length; i += 1) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
}

export function compareBytes(a: Uint8Array, b: Uint8Array): number {
  const length = Math.min(a.length, b.length);
  for (let i = 0; i < length; i += 1) {
    const av = a[i] ?? 0;
    const bv = b[i] ?? 0;
    if (av !== bv) {
      return av - bv;
    }
  }
  return a.length - b.length;
}
