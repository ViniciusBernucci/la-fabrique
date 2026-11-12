import { afterEach, expect, it, vi } from "vitest";
import { LAYOUT_STORAGE_KEY, readLayout } from "./layout-preference";

afterEach(() => vi.unstubAllGlobals());

it("usa o visual novo por padrão e respeita a escolha salva", () => {
  expect(readLayout()).toBe("novo");
  const store = new Map<string, string>([[LAYOUT_STORAGE_KEY, "antigo"]]);
  vi.stubGlobal("localStorage", { getItem: (key: string) => store.get(key) ?? null });
  expect(readLayout()).toBe("antigo");
  store.set(LAYOUT_STORAGE_KEY, "qualquer");
  expect(readLayout()).toBe("novo");
});

it("não quebra quando o armazenamento do navegador está bloqueado", () => {
  vi.stubGlobal("localStorage", {
    getItem: () => {
      throw new Error("bloqueado");
    },
  });
  expect(readLayout()).toBe("novo");
});
