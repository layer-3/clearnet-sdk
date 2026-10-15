"use client";

// Durable, cross-tab journal of signed transfers in IndexedDB. Every write is
// awaited before the next protocol step (sign -> save -> mark -> POST).

import { createStore, del, entries as idbEntries, get, set, type UseStore } from "idb-keyval";
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
let channel: BroadcastChannel | undefined;

function emit() {
  for (const l of listeners) l();
}

async function reload() {
  const rows = await idbEntries<string, JournalEntry>(db());
  snapshot = rows.map(([, v]) => v).sort((a, b) => b.createdAt - a.createdAt);
  loaded = true;
  emit();
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
  async all(): Promise<JournalEntry[]> {
    await reload();
    return snapshot;
  },
  async get(id: string): Promise<JournalEntry | undefined> {
    return get<JournalEntry>(id, db());
  },
  /** Durably writes an entry. Throws if the browser refused the write. */
  async put(entry: JournalEntry): Promise<void> {
    await set(entry.id, entry, db());
    const check = await get<JournalEntry>(entry.id, db());
    if (!check || check.state !== entry.state) throw new Error("This browser didn't save the transfer record.");
    await reload();
    channel?.postMessage("changed");
  },
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
