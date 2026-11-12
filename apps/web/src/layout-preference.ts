import { useCallback, useEffect, useState } from "react";

/** Visual escolhido no seletor Novo/Antigo; "novo" é a proposta sóbria, "antigo" o HUD. */
export type LayoutChoice = "novo" | "antigo";

export const LAYOUT_STORAGE_KEY = "le-fabrique.layout.v1";
const DEFAULT_LAYOUT: LayoutChoice = "novo";

export function readLayout(): LayoutChoice {
  try {
    const stored = globalThis.localStorage?.getItem(LAYOUT_STORAGE_KEY);
    return stored === "antigo" || stored === "novo" ? stored : DEFAULT_LAYOUT;
  } catch {
    return DEFAULT_LAYOUT;
  }
}

/** Marca o documento para que o CSS do tema escolhido entre em vigor. */
export function applyLayout(layout: LayoutChoice): void {
  if (typeof document !== "undefined") document.documentElement.dataset.layout = layout;
}

export function useLayoutPreference(): [LayoutChoice, (layout: LayoutChoice) => void] {
  const [layout, setLayoutState] = useState<LayoutChoice>(readLayout);
  useEffect(() => applyLayout(layout), [layout]);
  const setLayout = useCallback((next: LayoutChoice) => {
    try {
      globalThis.localStorage?.setItem(LAYOUT_STORAGE_KEY, next);
    } catch {
      // Preferência apenas visual: sem armazenamento, vale até recarregar a página.
    }
    setLayoutState(next);
  }, []);
  return [layout, setLayout];
}
