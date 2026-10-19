import type { ComponentType, SVGProps } from "react";
import { IconActivity, IconAgent, IconFund, IconHome, IconReceive, IconSend, IconSwap, IconWithdraw } from "@/components/ui/icons";

export interface NavItem {
  href: string;
  label: string;
  icon: ComponentType<SVGProps<SVGSVGElement> & { size?: number }>;
  soon?: boolean;
}

export const PRIMARY_NAV: NavItem[] = [
  { href: "/", label: "Account", icon: IconHome },
  { href: "/send", label: "Send", icon: IconSend },
  { href: "/activity", label: "Activity", icon: IconActivity },
  { href: "/receive", label: "Receive", icon: IconReceive },
];

/** Rail on desktop; top of "More" on phones (the tab bar holds PRIMARY_NAV). */
export const AGENT_NAV: NavItem = { href: "/agent", label: "AI agent", icon: IconAgent };

export const SOON_NAV: NavItem[] = [
  { href: "/fund", label: "Fund", icon: IconFund, soon: true },
  { href: "/withdraw", label: "Withdraw", icon: IconWithdraw, soon: true },
  { href: "/trade", label: "Trade", icon: IconSwap, soon: true },
];

export function isActive(pathname: string, href: string) {
  return href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);
}
