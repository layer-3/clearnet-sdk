import bs58 from "bs58";
import type { PublicKey, VersionedTransactionResponse } from "@solana/web3.js";

import { ClearnetSdkError } from "../../core/errors.js";
import {
  DEPOSITED_EVENT_DISCRIMINATOR,
  DEPOSITED_EVENT_MIN_LEN,
  EVENT_IX_TAG,
  EXECUTED_EVENT_DISCRIMINATOR,
  EXECUTED_EVENT_MIN_LEN,
} from "./constants.js";

// Reports whether the custody event at index is a Deposited event. Events are
// the programId inner instructions carrying a decodable Deposited or Executed
// event, counted in order across all inner instructions. Mirrors Go
// sol.depositEventAt.
export function depositEventAt(
  response: VersionedTransactionResponse,
  programId: PublicKey,
  index: bigint,
  txHash: string,
): boolean {
  const meta = response.meta;
  if (meta === null) {
    return false;
  }
  const keys = [
    ...response.transaction.message.staticAccountKeys,
    ...(meta.loadedAddresses?.writable ?? []),
    ...(meta.loadedAddresses?.readonly ?? []),
  ];
  let next = 0n;
  for (const inner of meta.innerInstructions ?? []) {
    for (const instruction of inner.instructions) {
      const program = keys[instruction.programIdIndex];
      if (program === undefined || !program.equals(programId)) {
        continue;
      }
      let data: Uint8Array;
      try {
        data = bs58.decode(instruction.data);
      } catch (error) {
        throw new ClearnetSdkError(
          "RPC_ERROR",
          "sol: inner instruction data is not base58",
          { txHash, cause: error },
        );
      }
      if (data.length < 16 || !startsWith(data, EVENT_IX_TAG, 0)) {
        continue;
      }
      const bodyLength = data.length - 16;
      let deposited: boolean;
      if (startsWith(data, DEPOSITED_EVENT_DISCRIMINATOR, 8)) {
        if (bodyLength < DEPOSITED_EVENT_MIN_LEN) {
          continue;
        }
        deposited = true;
      } else if (startsWith(data, EXECUTED_EVENT_DISCRIMINATOR, 8)) {
        if (bodyLength < EXECUTED_EVENT_MIN_LEN) {
          continue;
        }
        deposited = false;
      } else {
        continue;
      }
      if (next === index) {
        return deposited;
      }
      next += 1n;
    }
  }
  return false;
}

function startsWith(
  data: Uint8Array,
  prefix: readonly number[],
  offset: number,
): boolean {
  return prefix.every((byte, i) => data[offset + i] === byte);
}
