import { useCallback, useEffect, useState } from "react";

export type Theme = "dark" | "light";

const THEME_KEY = "le-fabrique.theme";

/** Preferência salva; sem ela, segue o sistema operacional. */
export function readTheme(): Theme {
  try {
    const saved = globalThis.localStorage?.getItem(THEME_KEY);
    if (saved === "light" || saved === "dark") return saved;
  } catch {
    // preferência apenas visual
  }
  return globalThis.matchMedia?.("(prefers-color-scheme: light)").matches ? "light" : "dark";
}

export function applyTheme(theme: Theme) {
  document.documentElement.dataset.theme = theme;
  document
    .querySelector('meta[name="theme-color"]')
    ?.setAttribute("content", theme === "light" ? "#f4f5f8" : "#0b0d12");
}

export function useTheme(): [Theme, () => void] {
  const [theme, setTheme] = useState<Theme>(readTheme);
  useEffect(() => applyTheme(theme), [theme]);
  const toggle = useCallback(() => {
    setTheme((current) => {
      const next = current === "light" ? "dark" : "light";
      try {
        globalThis.localStorage?.setItem(THEME_KEY, next);
      } catch {
        // preferência apenas visual
      }
      return next;
    });
  }, []);
  return [theme, toggle];
}
