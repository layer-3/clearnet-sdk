import type { Metadata } from "next";
import { AgentView } from "@/features/agent/agent-view";

export const metadata: Metadata = { title: "AI agent" };

export default function AgentPage() {
  return <AgentView />;
}
