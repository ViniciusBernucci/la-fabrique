import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { DecisionRequest } from "../lib/api";
import { SessionProvider } from "../lib/session";

const session = {
  id: "s1",
  admin_id: "a1",
  username: "admin",
  expires_at: "2030-01-01T00:00:00Z",
  csrf_token: "csrf-1",
  reauth_until: "2030-01-01T00:00:00Z",
  totp_reset_required: false,
};

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
const err = (status: number, code: string, message: string) =>
  json({ error: { code, message, action: "Recarregue.", correlation_id: "c-1" } }, status);

interface Call {
  method: string;
  url: string;
  body?: Record<string, unknown>;
  csrf?: string;
}

function stub(routes: Record<string, (call: Call) => Response>) {
  const calls: Call[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input).replace(/^\/api/, "");
      const method = (init.method ?? "GET").toUpperCase();
      const call: Call = {
        method,
        url,
        body: init.body ? JSON.parse(String(init.body)) : undefined,
        csrf: (init.headers as Record<string, string> | undefined)?.["X-CSRF-Token"],
      };
      calls.push(call);
      const handler = routes[`${method} ${url}`];
      return handler ? handler(call) : err(404, "not_found", "Rota inexistente.");
    }),
  );
  return calls;
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SessionProvider>
        <App />
      </SessionProvider>
    </MemoryRouter>,
  );
}

const ID = "77777777-7777-4777-8777-777777777777";

function decision(over: Partial<DecisionRequest> = {}): DecisionRequest {
  return {
    id: ID,
    project_id: "p",
    execution_id: "e",
    node_id: "dev",
    kind: "conflict",
    title: "Conflito de requisitos",
    problem: "Duas tarefas pedem comportamentos incompatíveis.",
    impact: "A feature fica parada.",
    evidence: [{ label: "Tarefa A", text: "Pede X" }],
    options: [
      { id: "seguir-a", label: "Seguir A", description: "Ignora B" },
      { id: "seguir-b", label: "Seguir B", description: "Ignora A" },
    ],
    state: "pending",
    execution_version: 2,
    resolution_option: "",
    resolution_note: "",
    resolved_at: null,
    version: 1,
    created_at: "2026-11-12T10:00:00Z",
    ...over,
  };
}

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("Central de Decisões", () => {
  it("mostra problema, impacto, evidência e alternativas, e só habilita depois de escolher", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /decisions?state=pending": () => json({ items: [decision()] }),
    });
    renderAt("/decisions");
    expect(
      await screen.findByText("Duas tarefas pedem comportamentos incompatíveis."),
    ).toBeInTheDocument();
    expect(screen.getByText(/A feature fica parada/)).toBeInTheDocument();
    expect(screen.getByText("Pede X")).toBeInTheDocument();
    expect(screen.getByText("Conflito")).toBeInTheDocument();
    const button = screen.getByRole("button", { name: "Registrar decisão" });
    expect(button).toBeDisabled();
    await userEvent.click(screen.getByRole("radio", { name: /Seguir B/ }));
    expect(button).toBeEnabled();
  });

  it("registra a decisão com a opção e a observação e recarrega a lista", async () => {
    let resolved = false;
    const calls = stub({
      "GET /session": () => json(session),
      "GET /decisions?state=pending": () => json({ items: resolved ? [] : [decision()] }),
      [`POST /decisions/${ID}/resolve`]: () => {
        resolved = true;
        return json(decision({ state: "resolved", resolution_option: "seguir-b" }));
      },
    });
    renderAt("/decisions");
    await userEvent.click(await screen.findByRole("radio", { name: /Seguir B/ }));
    await userEvent.type(screen.getByLabelText("Observação (opcional)"), "Cliente confirmou");
    await userEvent.click(screen.getByRole("button", { name: "Registrar decisão" }));
    expect(await screen.findByText("Nenhuma decisão pendente")).toBeInTheDocument();
    const post = calls.find((c) => c.method === "POST");
    expect(post?.body).toEqual({ option: "seguir-b", note: "Cliente confirmou" });
    expect(post?.csrf).toBe("csrf-1");
  });

  it("explica o pedido obsoleto e recarrega", async () => {
    let stale = false;
    stub({
      "GET /session": () => json(session),
      "GET /decisions?state=pending": () => json({ items: stale ? [] : [decision()] }),
      [`POST /decisions/${ID}/resolve`]: () => {
        stale = true;
        return err(409, "decision_stale", "A execução mudou desde o pedido: ele ficou obsoleto.");
      },
    });
    renderAt("/decisions");
    await userEvent.click(await screen.findByRole("radio", { name: /Seguir A/ }));
    await userEvent.click(screen.getByRole("button", { name: "Registrar decisão" }));
    await screen.findByText("Nenhuma decisão pendente");
  });

  it("mostra o motivo quando outra decisão já foi registrada", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /decisions?state=pending": () => json({ items: [decision()] }),
      [`POST /decisions/${ID}/resolve`]: () =>
        err(409, "decision_conflict", "O pedido já foi resolvido com outra opção."),
    });
    renderAt("/decisions");
    await userEvent.click(await screen.findByRole("radio", { name: /Seguir A/ }));
    await userEvent.click(screen.getByRole("button", { name: "Registrar decisão" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("já foi resolvido com outra opção");
  });

  it("mostra o histórico de resolvidas e as obsoletas sem formulário", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /decisions?state=pending": () => json({ items: [] }),
      "GET /decisions?state=resolved": () =>
        json({
          items: [
            decision({
              state: "resolved",
              resolution_option: "seguir-a",
              resolution_note: "Cliente confirmou",
              resolved_at: "2026-11-12T11:00:00Z",
            }),
          ],
        }),
      "GET /decisions?state=stale": () =>
        json({ items: [decision({ id: "88888888-8888-4888-8888-888888888888", state: "stale" })] }),
    });
    renderAt("/decisions");
    await screen.findByText("Nenhuma decisão pendente");
    await userEvent.click(screen.getByRole("tab", { name: "Resolvidas" }));
    expect(await screen.findByText(/Seguir A/)).toBeInTheDocument();
    expect(screen.getByText(/Cliente confirmou/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Registrar decisão" })).not.toBeInTheDocument();
    await userEvent.click(screen.getByRole("tab", { name: "Obsoletas" }));
    expect(await screen.findByText(/a execução mudou depois do pedido/)).toBeInTheDocument();
  });
});
