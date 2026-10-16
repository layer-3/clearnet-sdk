import { APIError, GATEWAY_ERROR_CODES, SubmissionError } from "@yellow-org/clearnet-sdk/gateway";

/** True when the person declined in their wallet (EIP-1193 4001 / viem UserRejectedRequestError). */
export function isUserRejection(e: unknown): boolean {
  let cur: unknown = e;
  for (let i = 0; i < 6 && cur; i++) {
    if (typeof cur === "object" && cur !== null) {
      const o = cur as { code?: unknown; name?: unknown; message?: unknown; cause?: unknown };
      if (o.code === 4001 || o.name === "UserRejectedRequestError") return true;
      if (typeof o.message === "string" && /user (rejected|denied|cancel)/i.test(o.message)) return true;
      cur = o.cause;
    } else break;
  }
  return false;
}

export function isChainMismatch(e: unknown): boolean {
  const m = e instanceof Error ? `${e.message} ${(e.cause as Error | undefined)?.message ?? ""}` : String(e);
  return /chain ?id|chainid|active chain|does not match|switch.*network/i.test(m);
}

/** Turns a gateway rejection into one plain sentence that says what to do. */
export function rejectionMessage(reason: string | undefined, api?: APIError): string {
  const r = (reason ?? api?.message ?? "").toLowerCase();
  if (/fee.*exceed|exceeds maxfee|max ?fee/.test(r)) return "The network fee was higher than your cap. Raise the fee cap and send again.";
  if (/inactive|activation|at least 10/.test(r)) return "Your account needs at least 10 YELLOW available before it can send.";
  if (api?.code === GATEWAY_ERROR_CODES.insufficientBalance || /insufficient/.test(r)) return "Not enough balance for this amount plus the network fee.";
  if (api?.code === GATEWAY_ERROR_CODES.nonceMismatch || /nonce/.test(r)) return "Your account changed since you reviewed this transfer. Review it again.";
  if (api?.code === GATEWAY_ERROR_CODES.invalidSignature || /signature/.test(r)) return "The network couldn't verify the signature. Sign again from the sending account.";
  if (/same account/.test(r)) return "You can't send to the same account you're sending from.";
  if (/ceiling|security/.test(r)) return "This transfer is above what the network can secure in one block right now. Try a smaller amount.";
  return reason ? `The network said: ${reason}` : "The network rejected this transfer.";
}

export function submissionRejection(e: unknown): string | undefined {
  if (e instanceof SubmissionError && e.definiteRejection) return rejectionMessage(e.definiteRejection.message, e.definiteRejection);
  return undefined;
}
