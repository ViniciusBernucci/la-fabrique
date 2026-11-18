import "@testing-library/jest-dom/vitest";
import { cleanup, configure } from "@testing-library/react";
import { afterEach } from "vitest";

// Máquinas carregadas (Docker, proxy e worker de desenvolvimento rodando) estouravam o limite padrão de 1 s dos findBy/waitFor.
configure({ asyncUtilTimeout: 5000 });

afterEach(() => cleanup());

// O React Flow mede o canvas com ResizeObserver e DOMMatrixReadOnly, que o jsdom não traz.
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
class DOMMatrixReadOnlyStub {
  m22: number;
  constructor(transform?: string) {
    const scale = transform?.match(/scale\(([\d.]+)\)/)?.[1];
    this.m22 = scale ? Number(scale) : 1;
  }
}
globalThis.ResizeObserver ??= ResizeObserverStub as unknown as typeof ResizeObserver;
globalThis.DOMMatrixReadOnly ??= DOMMatrixReadOnlyStub as unknown as typeof DOMMatrixReadOnly;
