import { displayParts, fromBaseUnits } from "@yellow-org/clearnet-sdk/gateway";

export function shortAddress(address: string, head = 6, tail = 4): string {
  if (address.length <= head + tail + 1) return address;
  return `${address.slice(0, head)}…${address.slice(-tail)}`;
}

/** Groups a hex address in fours for reading aloud or comparing. */
export function groupedHex(address: string): string[] {
  const body = address.startsWith("0x") ? address.slice(2) : address;
  return ["0x", ...(body.match(/.{1,4}/g) ?? [])];
}

export function amountParts(raw: bigint | string, decimals: number, maxFraction = 6) {
  return displayParts(fromBaseUnits(raw, decimals), maxFraction);
}

export function formatUnits(raw: bigint | string, decimals: number, maxFraction = 6): string {
  const p = amountParts(raw, decimals, maxFraction);
  return p.fraction ? `${p.whole}.${p.fraction}` : p.whole;
}


const rtf = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

export function timeAgo(unixSeconds: number, now = Date.now()): string {
  const diff = Math.round(unixSeconds - now / 1000);
  const abs = Math.abs(diff);
  if (abs < 45) return "just now";
  if (abs < 3600) return rtf.format(Math.round(diff / 60), "minute");
  if (abs < 86_400) return rtf.format(Math.round(diff / 3600), "hour");
  if (abs < 86_400 * 7) return rtf.format(Math.round(diff / 86_400), "day");
  return new Date(unixSeconds * 1000).toLocaleDateString("en", { month: "short", day: "numeric" });
}

export function dayLabel(unixSeconds: number, now = new Date()): string {
  const d = new Date(unixSeconds * 1000);
  const startOf = (x: Date) => new Date(x.getFullYear(), x.getMonth(), x.getDate()).getTime();
  const days = Math.round((startOf(now) - startOf(d)) / 86_400_000);
  if (days === 0) return "Today";
  if (days === 1) return "Yesterday";
  return d.toLocaleDateString("en", { weekday: "long", month: "long", day: "numeric", ...(d.getFullYear() !== now.getFullYear() && { year: "numeric" }) });
}

export function dateTime(unixSeconds: number): string {
  return new Date(unixSeconds * 1000).toLocaleString("en", { dateStyle: "medium", timeStyle: "medium" });
}

export function isIncoming(direction: string) {
  return direction === "incoming" || direction === "inbound" || direction === "in";
}
