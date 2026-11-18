import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { WorkflowRun } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { FactoryRuns, terminalLine } from "./FactoryRuns";

const session = {
  id: "s1",
  admin_id: "a1",
  username: "admin",
  expires_at: "2030-01-01T00:00:00Z",
  csrf_token: "csrf-1",
  reauth_until: "2030-01-01T00:00:00Z",
  totp_reset_required: false,
};
const PID = "11111111-1111-4111-8111-111111111111";
const RID = "22222222-2222-4222-8222-222222222222";
const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

function run(over: Partial<WorkflowRun> = {}): WorkflowRun {
  return {
    execution_id: RID,
    project_id: PID,
    workflow_version_id: "v1",
    title: "Tela de pedidos",
    task: "",
    workspace: "proj",
    kind: "manual",
    issue_key: "",
    status: "running",
    outcome: "",
    failure: "",
    pending: [],
    trace: [],
    wake_at: null,
    created_at: "2026-11-17T10:00:00Z",
    updated_at: "2026-11-17T10:00:00Z",
    ...over,
  };
}

class FakeEventSource {
  static all: FakeEventSource[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((m: MessageEvent<string>) => void) | null = null;
  onerror: (() => void) | null = null;
  constructor(public url: string) {
    FakeEventSource.all.push(this);
  }
  addEventListener() {}
  close() {}
  emit(seq: number, type: string, payload: unknown) {
    this.onmessage?.({
      data: JSON.stringify({ seq, type, payload, created_at: "2026-11-17T10:00:00Z" }),
    } as MessageEvent<string>);
  }
}

type Call = { method: string; url: string; body?: Record<string, unknown> };
function stub(routes: Record<string, (c: Call) => Response>) {
  const calls: Call[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const method = (init.method ?? "GET").toUpperCase();
      const url = String(input).replace(/^\/api/, "");
      const call: Call = {
        method,
        url,
        body: init.body ? JSON.parse(String(init.body)) : undefined,
      };
      calls.push(call);
      const all: Record<string, (c: Call) => Response> = {
        "GET /session": () => json(session),
        ...routes,
      };
      const h = all[`${method} ${url}`];
      return h
        ? h(call)
        : json(
            {
              error: {
                code: "not_found",
                message: "Rota inexistente.",
                action: "x",
                correlation_id: "c",
              },
            },
            404,
          );
    }),
  );
  return calls;
}

function mount() {
  return render(
    <MemoryRouter>
      <SessionProvider>
        <FactoryRuns projectId={PID} />
      </SessionProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  FakeEventSource.all = [];
  vi.stubGlobal("EventSource", FakeEventSource);
});
afterEach(() => vi.unstubAllGlobals());

describe("Executar a fábrica", () => {
  it("inicia uma execução com as variáveis do fluxo e mostra o erro de fluxo não publicado", async () => {
    const calls = stub({
      [`GET /projects/${PID}/runs`]: () => json({ items: [] }),
      [`POST /projects/${PID}/runs`]: () =>
        json(
          {
            error: {
              code: "not_published",
              message: "o workflow não tem versão publicada: rascunho não executa",
              action: "Publique o workflow antes de usá-lo.",
              correlation_id: "c",
            },
          },
          409,
        ),
    });
    mount();
    await userEvent.type(await screen.findByLabelText("Título da tarefa"), "Tela de pedidos");
    await userEvent.type(screen.getByLabelText("O que a fábrica deve fazer"), "Criar a tela.");
    await userEvent.click(screen.getByLabelText(/É a correção de uma Pull Request/));
    await userEvent.click(screen.getByRole("button", { name: "Iniciar execução" }));
    expect(await screen.findByText(/rascunho não executa/)).toBeInTheDocument();
    const post = calls.find((c) => c.method === "POST");
    expect(post?.body).toEqual({
      title: "Tela de pedidos",
      task: "Criar a tela.",
      prd_done: true,
      correction: true,
    });
  });

  it("lista as execuções, responde à decisão pendente e cancela", async () => {
    let current = run({ status: "awaiting_decision", pending: ["h"] });
    const calls = stub({
      [`GET /projects/${PID}/runs`]: () => json({ items: [current] }),
      [`POST /projects/${PID}/runs/${RID}/decide`]: () => {
        current = run({ status: "queued" });
        return json(current);
      },
      [`POST /projects/${PID}/runs/${RID}/cancel`]: () => json(run({ status: "cancelled" })),
    });
    mount();
    const item = await screen.findByLabelText("Execução Tela de pedidos");
    expect(within(item).getByText("Esperando uma decisão sua")).toBeInTheDocument();
    await userEvent.click(within(item).getByRole("button", { name: "Aprovar (h)" }));
    expect(calls.find((c) => c.url.endsWith("/decide"))?.body).toEqual({
      node: "h",
      choice: "approved",
    });
    await waitFor(() => expect(screen.getByText("Na fila")).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Cancelar execução" }));
    expect(calls.some((c) => c.url.endsWith("/cancel"))).toBe(true);
  });

  it("mostra o terminal ao vivo com o trajeto e a saída do agente", async () => {
    stub({ [`GET /projects/${PID}/runs`]: () => json({ items: [run()] }) });
    mount();
    await screen.findByLabelText("Execução Tela de pedidos");
    await waitFor(() => expect(FakeEventSource.all.length).toBeGreaterThan(0));
    const es = FakeEventSource.all[0] as FakeEventSource;
    act(() => {
      es.emit(1, "run_started", { title: "Tela de pedidos" });
      es.emit(2, "node_event", {
        node: "guard_planning",
        kind: "message",
        text: "editei o README",
      });
      es.emit(3, "node_event", { node: "chk", kind: "command", text: "$ make test" });
      es.emit(4, "run_step", { node: "guard_planning", type: "agent", port: "out" });
    });
    const log = await screen.findByRole("log", { name: "Terminal da execução" });
    expect(log).toHaveTextContent("▶ execução iniciada — Tela de pedidos");
    expect(log).toHaveTextContent("[guard_planning] editei o README");
    expect(log).toHaveTextContent("[chk] $ make test");
    expect(log).toHaveTextContent("● guard_planning [agent] → out");
  });

  it("formata as linhas do terminal", () => {
    const ev = (type: string, payload: unknown) => ({ seq: 1, type, payload, created_at: "" });
    expect(terminalLine(ev("run_finished", { status: "completed", outcome: "success" }))).toBe(
      "■ concluída — resultado: success",
    );
    expect(terminalLine(ev("node_event", { node: "n", kind: "error", text: "boom" }))).toBe(
      "  [n] ✖ boom",
    );
    expect(terminalLine(ev("run_cancelled", {}))).toBe("■ execução cancelada");
  });
});
