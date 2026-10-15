import { clsx } from "clsx";
import Link from "next/link";
import type { ComponentProps, ReactNode } from "react";

type Variant = "primary" | "signal" | "ghost" | "outline" | "quiet";
type Size = "sm" | "md" | "lg";

const base =
  "relative inline-flex select-none items-center justify-center gap-2 rounded-control type-label whitespace-nowrap disabled:opacity-40";

const variants: Record<Variant, string> = {
  primary: "btn border border-[var(--btn-edge)] bg-primary text-on-primary",
  signal: "btn border border-[var(--btn-edge)] bg-signal text-ink",
  ghost: "bg-transparent text-current transition-colors hover:bg-current/8",
  outline: "btn btn-quiet-shadow border border-current bg-transparent text-current",
  quiet: "btn btn-quiet-shadow border border-line bg-surface-2 text-on-surface",
};

const sizes: Record<Size, string> = {
  sm: "h-9 px-3.5 text-[13px]",
  md: "h-11 px-5 text-[15px]",
  lg: "h-13 px-6 text-[16px]",
};

export function buttonClass(variant: Variant = "primary", size: Size = "md", className?: string) {
  return clsx(base, variants[variant], sizes[size], className);
}

interface Common {
  variant?: Variant;
  size?: Size;
  icon?: ReactNode;
}

export function Button({ variant, size, icon, className, children, type = "button", ...rest }: Common & ComponentProps<"button">) {
  return (
    <button type={type} className={buttonClass(variant, size, className)} {...rest}>
      {icon}
      {children}
    </button>
  );
}

export function ButtonLink({ variant, size, icon, className, children, ...rest }: Common & ComponentProps<typeof Link>) {
  return (
    <Link className={buttonClass(variant, size, className)} {...rest}>
      {icon}
      {children}
    </Link>
  );
}
