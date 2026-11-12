import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { Tool } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { signature } from "./ToolsPage";

const session = {
  id: "s1",
  admin_id: "a1",
  username: "admin",
  expires_at: "2030-01-01T00:00:00Z",
  csrf_token: "csrf-1",
  reauth_until: "2030-01-01T00:00:00Z",
  totp_reset_required: false,
};

const items: Tool[] = [
  {
    name: "read_file",
    kind: "native",
    description: "Lê um arquivo do workspace.",
    input: [
      { name: "path", type: "path", required: true },
      { name: "max_bytes", type: "int", required: false },
    ],
    output: [{ name: "content", type: "string", required: true }],
    permissions: ["read"],
    limits: { timeout_seconds: 60, max_output_bytes: 1048576 },
  },
  {
    name: "git_status",
    kind: "native",
    description: "Estado do repositório.",
    input: [],
    output: [{ name: "output", type: "string", required: true }],
    permissions: ["git"],
    limits: { timeout_seconds: 60, max_output_bytes: 1048576 },
    requires: ["git"],
  },
  {
    name: "contar_linhas",
    kind: "script",
    description: "Conta as linhas.",
    input: [{ name: "path", type: "path", required: true }],
    output: [],
    permissions: ["read"],
    limits: { timeout_seconds: 30, max_output_bytes: 65536 },
  },
];

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("catálogo de ferramentas", () => {
  it("resume assinaturas com os obrigatórios marcados", () => {
    expect(signature(items[0].input)).toBe("path*: path, max_bytes: int");
    expect(signature([])).toBe("sem parâmetros");
  });

  it("lista nativas e personalizadas com permissões, limites e requisitos", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input).replace(/^\/api/, "");
        if (url === "/session") return new Response(JSON.stringify(session));
        if (url === "/tools") return new Response(JSON.stringify({ items }));
        return new Response(
          JSON.stringify({ error: { code: "not_found", message: "x", correlation_id: "c" } }),
          { status: 404 },
        );
      }),
    );
    render(
      <MemoryRouter initialEntries={["/tools"]}>
        <SessionProvider>
          <App />
        </SessionProvider>
      </MemoryRouter>,
    );
    expect(await screen.findByText("read_file")).toBeInTheDocument();
    expect(screen.getByText(/Entrada: path\*: path, max_bytes: int/)).toBeInTheDocument();
    expect(screen.getByText(/exige git na imagem/)).toBeInTheDocument();
    expect(screen.getAllByText("Nativa")).toHaveLength(2);
    expect(screen.getByText("Script")).toBeInTheDocument();
    expect(screen.getAllByText(/60 s · 1024 KiB/)).toHaveLength(2);
  });
});
