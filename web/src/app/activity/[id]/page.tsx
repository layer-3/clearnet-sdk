import type { Metadata } from "next";
import { OperationView } from "@/features/activity/operation-view";

export const metadata: Metadata = { title: "Transfer" };

export default async function OperationPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <OperationView id={id.toLowerCase()} />;
}
