"use client";

import { clsx } from "clsx";
import { useCopy } from "./copy-button";
import { IconCheck, IconCopy } from "./icons";

/** A command or prompt the reader copies as-is. Wraps on small screens. */
export function CodeBlock({ code, label, prose = false, className }: { code: string; label: string; prose?: boolean; className?: string }) {
  const { copied, copy } = useCopy();
  return (
    <div className={clsx("relative rounded-field border border-line bg-surface-2", className)}>
      <pre
        className={clsx(
          "overflow-x-auto py-3.5 pl-4 text-[13.5px] leading-6",
          prose ? "whitespace-pre-wrap break-words pr-24 font-sans text-[15px]" : "scrollbar-none whitespace-pre pr-4 pt-14 font-mono sm:pr-24 sm:pt-3.5",
        )}
        aria-label={label}
      >
        {code}
      </pre>
      <button
        type="button"
        onClick={() => copy(code)}
        className="type-label absolute right-2 top-2 inline-flex h-9 items-center gap-1.5 rounded-control border border-line bg-surface px-3 text-[13px] transition-colors hover:border-on-surface"
      >
        {copied ? <IconCheck size={15} /> : <IconCopy size={15} />}
        {copied ? "Copied" : "Copy"}
        <span className="sr-only"> {label}</span>
      </button>
    </div>
  );
}
