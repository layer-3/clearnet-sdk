import { Buffer } from "buffer";

import type { Memo } from "xrpl";

import { XRPL_MEMO_TYPE } from "./constants.js";

export function encodeClearnetMemo(
  account: Uint8Array,
  reference: Uint8Array,
): Memo[] {
  return [
    {
      Memo: {
        MemoType: XRPL_MEMO_TYPE,
        MemoData: hex(account) + hex(reference),
      },
    },
  ];
}

export function hex(bytes: Uint8Array): string {
  return Buffer.from(bytes).toString("hex");
}
