"use client";

import * as Dialog from "@radix-ui/react-dialog";
import { clsx } from "clsx";
import type { ReactNode } from "react";
import { IconClose } from "./icons";

/**
 * Bottom sheet on phones, centered dialog from `sm` up. Focus is trapped and
 * Escape closes it (Radix Dialog).
 */
export function Sheet({
  open,
  onOpenChange,
  title,
  description,
  children,
  className,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <Dialog.Root open={open} onOpenChange={onOpenChange}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/45 backdrop-blur-[2px] data-[state=open]:animate-[fade-in_180ms_ease-out]" />
        <Dialog.Content
          className={clsx(
            "sheet fixed inset-x-0 bottom-0 z-50 max-h-[92dvh] overflow-y-auto rounded-b-none p-5 pb-[max(1.25rem,env(safe-area-inset-bottom))] shadow-2xl outline-none",
            "data-[state=open]:animate-[sheet-up_260ms_cubic-bezier(0.2,0.9,0.1,1)]",
            "sm:inset-x-auto sm:bottom-auto sm:left-1/2 sm:top-1/2 sm:w-[min(520px,calc(100vw-2rem))] sm:-translate-x-1/2 sm:-translate-y-1/2 sm:rounded-b-[var(--radius-sheet)] sm:p-7 sm:data-[state=open]:animate-[pop-in_200ms_cubic-bezier(0.2,0.9,0.1,1)]",
            className,
          )}
        >
          <div className="mx-auto mb-4 h-1.5 w-10 rounded-full bg-line sm:hidden" aria-hidden />
          <div className="mb-5 flex items-start justify-between gap-4">
            <div>
              <Dialog.Title className="type-title text-2xl">{title}</Dialog.Title>
              {description ? <Dialog.Description className="mt-1 text-[15px] text-muted">{description}</Dialog.Description> : <Dialog.Description className="sr-only">{title}</Dialog.Description>}
            </div>
            <Dialog.Close className="-mr-2 -mt-1 grid size-10 shrink-0 place-items-center rounded-control hover:bg-line" aria-label="Close">
              <IconClose />
            </Dialog.Close>
          </div>
          {children}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
