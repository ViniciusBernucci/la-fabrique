import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ExecutionEvent, WorkflowGraph, WorkflowRun } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { FlowTestModal, failureOf, orderNodes, stepsOf } from "./FlowTestModal";

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

const graph: WorkflowGraph = {
  nodes: [
    { id: "inicio", type: "start", x: 0, y: 0, props: {} },
    { id: "dev", type: "agent", x: 0, y: 0, props: { agent: "implementador-backend" } },
    { id: "chk", type: "checks", x: 0, y: 0, props: {} },
    { id: "fim", type: "end", x: 0, y: 0, props: { outcome: "success" } },
    { id: "falha", type: "end", x: 0, y: 0, props: { outcome: "failure" } },
  ],
  edges: [
    { id: "e1", from: "inicio", from_port: "out", to: "dev", to_port: "in" },
    { id: "e2", from: "dev", from_port: "out", to: "chk", to_port: "in" },
    { id: "e3", from: "chk", from_port: "out", to: "fim", to_port: "in" },
    { id: "e4", from: "chk", from_port: "error", to: "falha", to_port: "in" },
  ],
};

function run(over: Partial<WorkflowRun> = {}): WorkflowRun {
  return {
    execution_id: RID,
    project_id: PID,
    workflow_version_id: "v1",
    title: "Teste da fábrica: selo de saúde do sistema",
    task: "",
    workspace: "",
    kind: "test",
    status: "running",
    outcome: "",
    failure: "",
    pending: [],
    trace: [],
    wake_at: null,
    created_at: "2026-11-17T10:00:00Z",
    updated_at: "2026-11-17T10:00:05Z",
    ...over,
  };
}

const ev = (seq: number, type: string, payload: unknown): ExecutionEvent => ({
  seq,
  type,
  payload,
  created_at: "2026-11-17T10:00:00Z",
});

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
  emit(e: ExecutionEvent) {
    this.onmessage?.({ data: JSON.stringify(e) } as MessageEvent<string>);
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
        "GET /workflow-versions/v1": () => json({ flat_graph: graph }),
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

function mount(workflowId?: string) {
  return render(
    <MemoryRouter>
      <SessionProvider>
        <FlowTestModal projectId={PID} workflowId={workflowId} onClose={() => {}} />
      </SessionProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  FakeEventSource.all = [];
  vi.stubGlobal("EventSource", FakeEventSource);
});
afterEach(() => vi.unstubAllGlobals());

describe("lógica do painel de teste", () => {
  it("ordena os nós pelo caminho do fluxo e deduz o estado de cada etapa", () => {
    expect(orderNodes(graph).map((n) => n.id)).toEqual(["inicio", "dev", "chk", "fim", "falha"]);
    const events = [
      ev(1, "run_step", { node: "inicio", type: "start", port: "out" }),
      ev(2, "node_event", { node: "dev", kind: "start" }),
      ev(3, "run_step", { node: "dev", type: "agent", port: "out" }),
      ev(4, "node_event", { node: "chk", kind: "start" }),
    ];
    const s = stepsOf(graph, events, run());
    expect(s.map((x) => x.state)).toEqual(["ok", "ok", "running", "pending", "pending"]);
    expect(s[1]?.label).toBe("Agente: implementador-backend");
    const closed = stepsOf(graph, events, run({ status: "cancelled" }));
    expect(closed.map((x) => x.state)).toEqual(["ok", "ok", "failed", "skipped", "skipped"]);
  });

  it("aponta o nó e o motivo da falha", () => {
    const events = [
      ev(1, "node_event", { node: "chk", kind: "error", text: "falhou (código 1): test -f x" }),
      ev(2, "run_step", { node: "chk", type: "checks", port: "error" }),
    ];
    const r = run({ status: "completed", outcome: "failure" });
    const f = failureOf(stepsOf(graph, events, r), events, r);
    expect(f?.node).toBe("chk");
    expect(f?.label).toContain("Checks");
    expect(f?.reason).toBe("falhou (código 1): test -f x");
    expect(failureOf(stepsOf(graph, [], run()), [], run())).toBeNull();
  });
});

describe("Painel de teste do fluxo", () => {
  it("dispara o teste, mostra etapas, terminal com orientações e o veredito", async () => {
    const calls = stub({
      [`POST /projects/${PID}/runs/test`]: () => json(run(), 201),
      [`GET /projects/${PID}/runs/${RID}`]: () => json(run()),
    });
    mount("33333333-3333-4333-8333-333333333333");
    expect(await screen.findByText(/Teste em andamento/)).toBeInTheDocument();
    expect(calls.find((c) => c.method === "POST")?.body).toEqual({
      workflow_id: "33333333-3333-4333-8333-333333333333",
    });
    await waitFor(() => expect(FakeEventSource.all.length).toBeGreaterThan(0));
    const es = FakeEventSource.all[0] as FakeEventSource;
    act(() => {
      es.emit(ev(1, "run_step", { node: "inicio", type: "start", port: "out" }));
      es.emit(ev(2, "node_event", { node: "dev", kind: "start", text: "iniciando agent" }));
      es.emit(
        ev(3, "node_event", {
          node: "dev",
          kind: "agent",
          agent: "implementador-backend",
          provider: "claude",
          model: "",
          account: "Claude pessoal",
          skills: ["lf-implementar-tarefa"],
        }),
      );
      es.emit(
        ev(4, "node_event", {
          node: "dev",
          kind: "prompt",
          text: "Você é o agente implementador-backend. MODO TESTE DA FÁBRICA",
        }),
      );
      es.emit(
        ev(5, "node_event", { node: "dev", kind: "message", text: "criei o arquivo da trilha" }),
      );
    });
    const etapas = await screen.findByRole("region", { name: "Etapas do fluxo" });
    expect(within(etapas).getByText(/Agente: implementador-backend/)).toBeInTheDocument();
    const term = await screen.findByRole("region", { name: "Terminal dev" });
    expect(term).toHaveTextContent(
      "Agente implementador-backend · claude · modelo padrão da conta · conta Claude pessoal · skills: lf-implementar-tarefa",
    );
    expect(within(term).getByText(/Orientações enviadas ao agente/)).toBeInTheDocument();
    expect(within(term).getByRole("log", { name: "Saída de dev" })).toHaveTextContent(
      "criei o arquivo da trilha",
    );
    expect(within(term).getByText(/MODO TESTE DA FÁBRICA/)).toBeInTheDocument();
  });

  it("mostra onde e por que o teste falhou", async () => {
    stub({
      [`POST /projects/${PID}/runs/test`]: () =>
        json(run({ status: "completed", outcome: "failure" }), 201),
      [`GET /projects/${PID}/runs/${RID}`]: () =>
        json(run({ status: "completed", outcome: "failure" })),
    });
    mount();
    await waitFor(() => expect(FakeEventSource.all.length).toBeGreaterThan(0));
    const es = FakeEventSource.all[0] as FakeEventSource;
    act(() => {
      es.emit(ev(1, "node_event", { node: "chk", kind: "start", text: "iniciando checks" }));
      es.emit(
        ev(2, "node_event", {
          node: "chk",
          kind: "error",
          text: "falhou (código 1): test -f fabrica-teste/backend/saude.md",
        }),
      );
      es.emit(ev(3, "run_step", { node: "chk", type: "checks", port: "error" }));
    });
    const falha = await screen.findByRole("alert", { name: "Falha do teste" });
    expect(falha).toHaveTextContent("Checks (chk)");
    expect(falha).toHaveTextContent("falhou (código 1): test -f fabrica-teste/backend/saude.md");
    expect(screen.getByText("Teste com falha")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Testar de novo" }));
  });

  it("explica o erro quando o fluxo não pode ser preparado para o teste", async () => {
    stub({
      [`POST /projects/${PID}/runs/test`]: () =>
        json(
          {
            error: {
              code: "publish_blocked",
              message: "O fluxo tem pendências e não pôde ser preparado para o teste: x",
              action: "Corrija os problemas listados no editor e teste de novo.",
              correlation_id: "c",
            },
          },
          422,
        ),
    });
    mount();
    expect(await screen.findByText(/não pôde ser preparado para o teste/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Testar de novo" })).toBeInTheDocument();
  });

  it("mostra os problemas do pré-voo, as causas prováveis e copia o relatório de diagnóstico", async () => {
    const report = "# Relatório de diagnóstico da execução\n- Nó: `chk`";
    const calls = stub({
      [`POST /projects/${PID}/runs/test`]: () =>
        json(run({ status: "completed", outcome: "failure" }), 201),
      [`GET /projects/${PID}/runs/${RID}`]: () =>
        json(run({ status: "completed", outcome: "failure" })),
      [`GET /projects/${PID}/runs/${RID}/diagnostics`]: () =>
        json({
          report,
          hints: [
            { cause: "A imagem dos checks não tem o comando.", fix: "Defina FACTORY_TOOL_IMAGE." },
          ],
          failure: { node: "chk", reason: "falhou (código 127): make lint" },
        }),
    });
    const write = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { ...navigator, clipboard: { writeText: write } });
    mount();
    await waitFor(() => expect(FakeEventSource.all.length).toBeGreaterThan(0));
    const es = FakeEventSource.all[0] as FakeEventSource;
    act(() => {
      es.emit(
        ev(1, "run_diag", {
          step: "preflight",
          check: "conta claude",
          ok: false,
          detail: "0 contas ativas",
          dica: "Conecte a conta em Conexões.",
        }),
      );
      es.emit(
        ev(2, "run_diag", {
          node: "chk",
          step: "command.run",
          ms: 120,
          exit_code: 127,
          cmd: "make lint",
        }),
      );
      es.emit(
        ev(3, "node_event", { node: "chk", kind: "error", text: "falhou (código 127): make lint" }),
      );
      es.emit(ev(4, "run_step", { node: "chk", type: "checks", port: "error" }));
    });
    const amb = await screen.findByRole("alert", { name: "Problemas do ambiente" });
    expect(amb).toHaveTextContent("conta claude");
    expect(amb).toHaveTextContent("Conecte a conta em Conexões.");
    const causas = await screen.findByRole("status", { name: "Causas prováveis" });
    expect(causas).toHaveTextContent("A imagem dos checks não tem o comando.");
    expect(causas).toHaveTextContent("Defina FACTORY_TOOL_IMAGE.");
    const tech = screen.getByRole("log", { name: "Diagnóstico técnico" });
    expect(tech).toHaveTextContent("[chk] command.run 120 ms");
    await userEvent.click(screen.getByRole("button", { name: "Copiar relatório de diagnóstico" }));
    await waitFor(() => expect(write).toHaveBeenCalledWith(report));
    expect(await screen.findByText(/Relatório copiado/)).toBeInTheDocument();
    expect(calls.filter((c) => c.url.endsWith("/diagnostics")).length).toBeGreaterThan(0);
  });
});
