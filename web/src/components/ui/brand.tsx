import { clsx } from "clsx";

/**
 * The Yellow "Y" mark. `brand` is the canonical yellow disc; `canvas` adapts
 * to the page: an ink disc on the yellow page, the yellow disc on black.
 */
export function YellowMark({ size = 32, tone = "brand", className }: { size?: number; tone?: "brand" | "canvas"; className?: string }) {
  const disc = tone === "canvas" ? "var(--mark-disc)" : "#FCD000";
  const y = tone === "canvas" ? "var(--mark-y)" : "#000";
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" className={className} aria-hidden>
      <rect width="32" height="32" rx="16" fill={disc} />
      <path
        fill={y}
        d="M25.72 6.64c.19 0 .32.19.25.36l-7.59 16.33c-.97 2.3-3.6 4.19-6.25 4.19H9.97c-.15-.02-.27-.13-.27-.27v-3.6c0-.14.11-.26.26-.26h2.16c.92 0 1.77-.38 2.18-1.51l.1-.31 6.57-14.13.3-.59c.05-.13.18-.21.32-.21h4.13Z"
      />
      <path fill={y} d="M12.1 6.62c.14 0 .26.08.31.21l3.7 7.26-2.4 5.16L7.77 6.98c-.08-.17.05-.36.24-.36h4.09Z" />
    </svg>
  );
}

export function Wordmark({ className }: { className?: string }) {
  return (
    <span className={clsx("inline-flex items-center gap-2.5", className)}>
      <YellowMark size={32} tone="canvas" />
      <span className="type-title text-[22px] leading-none">Yellow</span>
    </span>
  );
}
