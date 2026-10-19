"use client";

import { clsx } from "clsx";
import { useState, type ReactNode } from "react";
import { IconCheck, IconCopy } from "./icons";

export function useCopy(timeout = 1600) {
  const [copied, setCopied] = useState(false);
  const copy = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
    } catch {
      const t = document.createElement("textarea");
      t.value = text;
      document.body.appendChild(t);
      t.select();
      document.execCommand("copy");
      t.remove();
    }
    setCopied(true);
    window.setTimeout(() => setCopied(false), timeout);
  };
  return { copied, copy };
}

export function CopyButton({ value, label = "Copy", className, children }: { value: string; label?: string; className?: string; children?: ReactNode }) {
  const { copied, copy } = useCopy();
  return (
    <button
      type="button"
      onClick={() => copy(value)}
      className={clsx("inline-flex items-center gap-1.5 rounded-control transition-colors hover:bg-current/8", className)}
      aria-label={copied ? "Copied" : label}
    >
      {children}
      <span className="relative grid size-5 place-items-center">
        {copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
      </span>
      <span className="sr-only" aria-live="polite">
        {copied ? "Copied" : ""}
      </span>
    </button>
  );
}
