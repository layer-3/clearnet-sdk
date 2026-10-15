import type { Metadata } from "next";
import { Suspense } from "react";
import { PageSkeleton } from "@/components/connect-screen";
import { SendView } from "@/features/send/send-view";

export const metadata: Metadata = { title: "Send" };

export default function SendPage() {
  return (
    <Suspense fallback={<PageSkeleton />}>
      <SendView />
    </Suspense>
  );
}
