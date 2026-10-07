import { parseEventLogs } from "viem";
import type { Address, TransactionReceipt } from "viem";

import { custodyAbi } from "./abi.js";
import { depositId as computeDepositId } from "./depositId.js";

export function hasDepositedLog(
  receipt: TransactionReceipt,
  custodyAddress: Address,
  chainId: bigint,
  wantDepositIdLower: string,
): boolean {
  return parseEventLogs({
    abi: custodyAbi,
    eventName: "Deposited",
    logs: [...receipt.logs],
  }).some((log) => {
    if (log.address.toLowerCase() !== custodyAddress.toLowerCase()) {
      return false;
    }
    const gotId = computeDepositId(
      chainId,
      custodyAddress,
      log.args.depositor,
      log.args.nonce,
    );
    return gotId.toLowerCase() === wantDepositIdLower;
  });
}
