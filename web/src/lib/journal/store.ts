"use client";

// Durable, cross-tab journal of signed transfers in IndexedDB. Every write is
// awaited before the next protocol step (sign -> save -> mark -> POST).

import { createStore, del, entries as idbEntries, get, set, type UseStore } from "idb-keyval";
import { isFinalOperationState } from "@yellow-org/clearnet-sdk/gateway";
import type { JournalEntry } from "./copy";

let store: UseStore | undefined;
function db(): UseStore {
  if (!store) store = createStore("yellow-journal", "operations");
  return store;
}

type Listener = () => void;
const listeners = new Set<Listener>();
let snapshot: JournalEntry[] = [];
let loaded = false;
let loadError: string | undefined;
let channel: BroadcastChannel | undefined;

function emit() {
  for (const l of listeners) l();
}

async function reload() {
  try {
    const rows = await idbEntries<string, JournalEntry>(db());
    snapshot = rows.map(([, v]) => v).sort((a, b) => b.createdAt - a.createdAt);
    loadError = undefined;
  } catch (cause) {
    // Storage can be unavailable (private mode, blocked third-party storage).
    // Report it instead of leaving the UI on a skeleton forever.
    loadError = cause instanceof Error ? cause.message : "Local storage is unavailable.";
  } finally {
    loaded = true;
    emit();
  }
}

function ensureChannel() {
  if (channel || typeof BroadcastChannel === "undefined") return;
  channel = new BroadcastChannel("yellow-journal");
  channel.onmessage = () => void reload();
}

export const journal = {
  subscribe(listener: Listener) {
    listeners.add(listener);
    ensureChannel();
    if (!loaded) void reload();
    return () => listeners.delete(listener);
  },
  getSnapshot(): JournalEntry[] {
    return snapshot;
  },
  getServerSnapshot(): JournalEntry[] {
    return EMPTY;
  },
  isLoaded() {
    return loaded;
  },
  /** Set when local storage could not be read; cleared on the next success. */
  loadError() {
    return loadError;
  },
  async all(): Promise<JournalEntry[]> {
    await reload();
    return snapshot;
  },
  async get(id: string): Promise<JournalEntry | undefined> {
    return get<JournalEntry>(id, db());
  },
  /**
   * Durably writes an entry. Throws if the browser refused the write. A record
   * that is already final is never downgraded: a stale poll must not turn a
   * sealed transfer back into a pending or unknown one.
   */
  async put(entry: JournalEntry): Promise<void> {
    const stored = await get<JournalEntry>(entry.id, db());
    if (stored && isFinalOperationState(stored.state) && !isFinalOperationState(entry.state)) return;
    await set(entry.id, entry, db());
    const check = await get<JournalEntry>(entry.id, db());
    if (!check || check.state !== entry.state) throw new Error("This browser didn't save the transfer record.");
    await reload();
    channel?.postMessage("changed");
  },
  /** Drops a saved record. Only safe for an authorization that was never sent. */
  async remove(id: string): Promise<void> {
    await del(id, db());
    await reload();
    channel?.postMessage("changed");
  },
};

const EMPTY: JournalEntry[] = [];

/**
 * Runs `fn` while holding an exclusive cross-tab lock, so two tabs can't sign
 * from the same account at once (the CLI uses a file lock for this).
 */
export async function withAccountLock<T>(key: string, fn: () => Promise<T>): Promise<T> {
  if (typeof navigator !== "undefined" && "locks" in navigator && navigator.locks) {
    const result = await navigator.locks.request(`yellow-send:${key}`, { ifAvailable: true }, async (lock) => {
      if (!lock) throw new LockBusyError();
      return { value: await fn() };
    });
    return (result as { value: T }).value;
  }
  return fn();
}

export class LockBusyError extends Error {
  constructor() {
    super("Another tab is sending from this account. Finish or close it first.");
    this.name = "LockBusyError";
  }
}
