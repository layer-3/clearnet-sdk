import type { OperationRecord, OperationState } from "@yellow-org/clearnet-sdk/gateway";

/** A saved transfer, tagged with the deployment network it belongs to. */
export interface JournalEntry extends OperationRecord {
  networkSlug: string;
}

export const STATE_COPY: Record<OperationState, { label: string; message: string; tone: "ok" | "wait" | "warn" | "bad" }> = {
  signed_unsubmitted: {
    label: "Signed, not sent",
    message: "Saved on this device but never sent. You can send this exact authorization.",
    tone: "warn",
  },
  pending: {
    label: "Waiting for seal",
    message: "The network accepted it and validators are sealing it. It isn't final yet.",
    tone: "wait",
  },
  unresolved: {
    label: "Outcome unknown",
    message: "This transfer may have gone through. Don't send it again as a new transfer; we'll keep checking.",
    tone: "warn",
  },
  sealed: { label: "Sealed", message: "Sealed by the network. This transfer is final.", tone: "ok" },
  rejected: { label: "Rejected", message: "The network rejected this transfer. No funds moved.", tone: "bad" },
};
