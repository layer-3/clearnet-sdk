import { Buffer } from "buffer";

export function encodeDepositData(
  discriminator: readonly number[],
  account: Uint8Array,
  reference: Uint8Array,
  amount: bigint,
): Buffer {
  const data = new Uint8Array(8 + 20 + 32 + 8);
  data.set(discriminator, 0);
  data.set(account, 8);
  data.set(reference, 28);
  new DataView(data.buffer).setBigUint64(60, amount, true);
  return Buffer.from(data);
}
