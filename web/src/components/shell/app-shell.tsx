"use client";

import { clsx } from "clsx";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState, type ReactNode } from "react";
import { CONFIG_ERROR } from "@/config/networks";
import { useOperationPoller } from "@/hooks/operation-poller";
import { useMyOperations } from "@/hooks/gateway";
import { useTheme } from "@/lib/theme";
import { Wordmark, YellowMark } from "@/components/ui/brand";
import { IconMoon, IconMore, IconSun } from "@/components/ui/icons";
import { Sheet } from "@/components/ui/sheet";
import { AccountButton } from "./account-button";
import { AGENT_NAV, isActive, PRIMARY_NAV, SOON_NAV, type NavItem } from "./nav";
import { NetworkPill } from "./network-pill";

export function AppShell({ children }: { children: ReactNode }) {
  useOperationPoller();
  return (
    <div className="min-h-dvh lg:pl-[272px]">
      <Rail />
      <header className="pt-safe sticky top-0 z-30 bg-canvas/85 backdrop-blur-md supports-[backdrop-filter]:bg-canvas/70">
        <div className="mx-auto flex h-16 max-w-[1120px] items-center gap-3 px-4 sm:px-6 lg:h-20 lg:px-10">
          <Link href="/" className="lg:hidden" aria-label="Yellow home">
            <YellowMark size={36} tone="canvas" />
          </Link>
          <div className="flex-1" />
          <NetworkPill />
          <AccountButton />
        </div>
      </header>
      {CONFIG_ERROR && (
        <p role="alert" className="mx-auto mb-2 max-w-[1120px] rounded-field bg-flare px-4 py-3 text-[14px] text-white">
          {CONFIG_ERROR}
        </p>
      )}
      <main id="main" className="mx-auto max-w-[1120px] px-4 pb-32 sm:px-6 lg:px-10 lg:pb-20">
        {children}
      </main>
      <TabBar />
    </div>
  );
}

function OpenBadge() {
  const open = useMyOperations().filter((o) => o.state !== "sealed" && o.state !== "rejected").length;
  if (!open) return null;
  return (
    <span className="num ml-auto grid min-w-5 place-items-center rounded-full bg-flare px-1.5 text-[11px] font-semibold text-white" aria-label={`${open} transfer${open > 1 ? "s" : ""} need attention`}>
      {open}
    </span>
  );
}

function RailLink({ item, pathname }: { item: NavItem; pathname: string }) {
  const active = isActive(pathname, item.href);
  const Icon = item.icon;
  return (
    <Link
      href={item.href}
      aria-current={active ? "page" : undefined}
      className={clsx(
        "group flex h-12 items-center gap-3.5 rounded-control px-4 text-[16px] transition-colors",
        active ? "bg-nav-active text-on-nav-active" : "hover:bg-current/7",
        item.soon && !active && "text-on-canvas-muted",
      )}
    >
      <Icon size={20} />
      <span className="type-label">{item.label}</span>
      {item.href === "/activity" && <OpenBadge />}
      {item.soon && <span className={clsx("ml-auto text-[12px]", active ? "opacity-80" : "opacity-70")}>Soon</span>}
    </Link>
  );
}

function ThemeToggle({ className }: { className?: string }) {
  const { dark, toggle } = useTheme();
  return (
    <button type="button" onClick={toggle} className={clsx("inline-flex h-11 items-center gap-3 rounded-control px-4 text-[15px] hover:bg-current/7", className)}>
      {dark ? <IconSun size={19} /> : <IconMoon size={19} />}
      <span className="type-label">{dark ? "Light mode" : "Dark mode"}</span>
    </button>
  );
}

function Rail() {
  const pathname = usePathname();
  return (
    <aside className="fixed inset-y-0 left-0 z-40 hidden w-[272px] flex-col border-r border-canvas-line px-5 py-7 lg:flex">
      <Link href="/" className="mb-10 px-3" aria-label="Yellow home">
        <Wordmark />
      </Link>
      <nav aria-label="Main" className="flex flex-col gap-1">
        {PRIMARY_NAV.map((item) => (
          <RailLink key={item.href} item={item} pathname={pathname} />
        ))}
        <div className="mx-4 my-4 h-px bg-canvas-line" />
        <RailLink item={AGENT_NAV} pathname={pathname} />
        <div className="mx-4 my-4 h-px bg-canvas-line" />
        {SOON_NAV.map((item) => (
          <RailLink key={item.href} item={item} pathname={pathname} />
        ))}
      </nav>
      <div className="mt-auto">
        <ThemeToggle />
      </div>
    </aside>
  );
}

function TabBar() {
  const pathname = usePathname();
  const [more, setMore] = useState(false);
  const moreActive = [AGENT_NAV, ...SOON_NAV].some((i) => isActive(pathname, i.href));
  const itemClass = (active: boolean) =>
    clsx("relative flex h-full flex-1 flex-col items-center justify-center gap-0.5 text-[11px] transition-colors", active ? "text-on-surface" : "text-muted");
  const iconClass = (active: boolean) =>
    clsx("relative grid h-8 w-11 place-items-center rounded-control transition-colors", active && "bg-accent text-on-accent");
  return (
    <>
      <nav aria-label="Main" className="fixed inset-x-3 bottom-[max(0.75rem,env(safe-area-inset-bottom))] z-40 lg:hidden">
        <div className="mx-auto flex h-[64px] max-w-md items-stretch rounded-field border border-line bg-surface/95 px-1.5 text-on-surface shadow-[0_10px_30px_-12px_rgba(0,0,0,0.25)] backdrop-blur-md dark:shadow-[0_10px_30px_-10px_rgba(0,0,0,0.8)]">
          {PRIMARY_NAV.map((item) => {
            const active = isActive(pathname, item.href);
            const Icon = item.icon;
            return (
              <Link key={item.href} href={item.href} aria-current={active ? "page" : undefined} className={itemClass(active)}>
                <span className={iconClass(active)}>
                  <Icon size={21} />
                  {item.href === "/activity" && <TabDot />}
                </span>
                <span className="type-label">{item.label}</span>
              </Link>
            );
          })}
          <button type="button" onClick={() => setMore(true)} className={itemClass(moreActive)} aria-haspopup="dialog">
            <span className={iconClass(moreActive)}>
              <IconMore size={21} />
            </span>
            <span className="type-label">More</span>
          </button>
        </div>
      </nav>
      <Sheet open={more} onOpenChange={setMore} title="More">
        <div className="grid gap-2">
          <Link href={AGENT_NAV.href} onClick={() => setMore(false)} className="flex h-14 items-center gap-4 rounded-field bg-surface-2 px-4">
            <AGENT_NAV.icon />
            <span className="type-label text-[16px]">{AGENT_NAV.label}</span>
          </Link>
          {SOON_NAV.map((item) => {
            const Icon = item.icon;
            return (
              <Link key={item.href} href={item.href} onClick={() => setMore(false)} className="flex h-14 items-center gap-4 rounded-field bg-surface-2 px-4">
                <Icon />
                <span className="type-label text-[16px]">{item.label}</span>
                <span className="ml-auto text-[13px] text-muted">Coming soon</span>
              </Link>
            );
          })}
          <ThemeToggle className="mt-2 h-14 justify-start rounded-field bg-surface-2 !px-4 hover:bg-surface-2" />
        </div>
      </Sheet>
    </>
  );
}

function TabDot() {
  const open = useMyOperations().some((o) => o.state !== "sealed" && o.state !== "rejected");
  return open ? <span className="absolute -right-1 -top-0.5 size-2.5 rounded-full bg-flare ring-2 ring-surface" aria-label="A transfer needs attention" /> : null;
}
