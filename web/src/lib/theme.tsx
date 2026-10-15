"use client";

import { useCallback, useSyncExternalStore } from "react";

export type ThemePref = "light" | "dark" | "system";
const KEY = "yellow:theme";

const listeners = new Set<() => void>();
function subscribe(l: () => void) {
  listeners.add(l);
  const mq = window.matchMedia("(prefers-color-scheme: dark)");
  const onChange = () => {
    apply(read());
    l();
  };
  mq.addEventListener("change", onChange);
  return () => {
    listeners.delete(l);
    mq.removeEventListener("change", onChange);
  };
}
function read(): ThemePref {
  const v = window.localStorage.getItem(KEY);
  return v === "light" || v === "dark" ? v : "system";
}
function isDark(p: ThemePref) {
  return p === "dark" || (p === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
}
function apply(p: ThemePref) {
  document.documentElement.classList.toggle("dark", isDark(p));
}

export function useTheme() {
  const pref = useSyncExternalStore(subscribe, read, () => "system" as ThemePref);
  const dark = useSyncExternalStore(subscribe, () => isDark(read()), () => false);
  const setPref = useCallback((p: ThemePref) => {
    window.localStorage.setItem(KEY, p);
    apply(p);
    listeners.forEach((l) => l());
  }, []);
  const toggle = useCallback(() => setPref(isDark(read()) ? "light" : "dark"), [setPref]);
  return { pref, dark, setPref, toggle };
}
