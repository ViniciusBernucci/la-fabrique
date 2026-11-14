import { afterEach, describe, expect, it } from "vitest";
import { applyTheme, readTheme } from "./theme";

afterEach(() => {
  localStorage.clear();
  delete document.documentElement.dataset.theme;
});

describe("tema", () => {
  it("usa a preferência salva", () => {
    localStorage.setItem("le-fabrique.theme", "light");
    expect(readTheme()).toBe("light");
  });
  it("sem preferência, cai no escuro quando o sistema não pede claro", () => {
    expect(readTheme()).toBe("dark");
  });
  it("aplica o tema no elemento raiz", () => {
    applyTheme("light");
    expect(document.documentElement.dataset.theme).toBe("light");
  });
});
