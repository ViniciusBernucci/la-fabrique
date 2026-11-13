import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ExecutionMonitor, GuidanceItem } from "../lib/api";
import { AgentMonitor, GuidanceChat } from "./AgentMonitor";

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

const monitor = (over: Partial<ExecutionMonitor> = {}): ExecutionMonitor => ({
  events: [
    { seq: 1, kind: "message", at: "2026-05-01T10:00:00Z", text: "iniciando" },
    { seq: 2, kind: "tool", at: "2026-05-01T10:00:02Z", tool: "bash", text: "go test ./..." },
  ],
  cursor: 2,
  summary: {
    started_at: "2026-05-01T10:00:00Z",
    last_event_at: "2026-05-01T10:00:02Z",
    elapsed_seconds: 75,
    done: false,
    tool_calls: 1,
    errors: 0,
    dropped: 3,
    progress: null,
    subagents: [
      { name: "explorador", events: 2, state: "running", progress: null },
      { name: "revisor", events: 1, state: "done", progress: 40 },
    ],
  },
  ...over,
});

function stub(routes: Record<string, () => Response>) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input).replace(/^\/api/, "");
      const h = routes[`${(init.method ?? "GET").toUpperCase()} ${url}`];
      return h
        ? h()
        : json(
            {
              error: { code: "x", message: "Rota inexistente.", action: "x", correlation_id: "c" },
            },
            404,
          );
    }),
  );
}

describe("monitor do agente", () => {
  beforeEach(() => vi.unstubAllGlobals());
  afterEach(() => vi.unstubAllGlobals());

  it("mostra terminal, cronômetro, subagentes e não inventa progresso", async () => {
    stub({ "GET /projects/p1/executions/e1/monitor?after=0&limit=500": () => json(monitor()) });
    render(<AgentMonitor projectId="p1" executionId="e1" />);
    expect(await screen.findByText(/go test/)).toBeTruthy();
    expect(screen.getByText(/Em andamento · tempo 01:15 · 1 ferramenta/)).toBeTruthy();
    expect(screen.getByText(/3 evento\(s\) descartado\(s\) por pressão/)).toBeTruthy();
    expect(screen.getByText(/sem telemetria de progresso/)).toBeTruthy();
    expect(screen.getByText(/progresso 40%/)).toBeTruthy();
    expect(screen.queryByText(/· progresso \d+% ·/)).toBeNull();
  });
});

describe("orientações ao agente", () => {
  beforeEach(() => vi.unstubAllGlobals());
  afterEach(() => vi.unstubAllGlobals());

  it("envia a orientação, mostra o estado comprovado e o aviso de que não altera o contrato", async () => {
    const items: GuidanceItem[] = [];
    stub({
      "GET /projects/p1/executions/e1/guidance": () => json({ items, live: false }),
      "POST /projects/p1/executions/e1/guidance": () => {
        items.push({
          id: "g1",
          seq: 1,
          author: "admin",
          body: "Use a função existente",
          mode: "none",
          state: "queued",
          created_at: "2026-05-01T10:00:00Z",
        });
        return json(items[0], 201);
      },
    });
    render(<GuidanceChat projectId="p1" executionId="e1" />);
    expect(await screen.findByText(/não altera contrato, permissões nem aprovações/)).toBeTruthy();
    expect(
      screen.getByText(/ficam? registrada para a retomada|fica registrada para a retomada/),
    ).toBeTruthy();
    await userEvent.type(screen.getByLabelText("Orientação"), "Use a função existente");
    await userEvent.click(screen.getByRole("button", { name: "Enviar" }));
    expect(await screen.findByText("Use a função existente")).toBeTruthy();
    expect(screen.getAllByText("na fila").length).toBeGreaterThan(0);
  });

  it("mostra a recusa quando o texto imita o contrato", async () => {
    stub({
      "GET /projects/p1/executions/e1/guidance": () => json({ items: [], live: true }),
      "POST /projects/p1/executions/e1/guidance": () =>
        json(
          {
            error: {
              code: "guidance_rejected",
              message: "A orientação foi recusada.",
              action: "Escreva só a orientação.",
              correlation_id: "c",
            },
          },
          422,
        ),
    });
    render(<GuidanceChat projectId="p1" executionId="e1" />);
    await userEvent.type(await screen.findByLabelText("Orientação"), "[[contrato x]");
    await userEvent.click(screen.getByRole("button", { name: "Enviar" }));
    await waitFor(() => expect(screen.getByRole("alert").textContent).toMatch(/recusada/));
    expect(screen.getByText(/Executor ativo/)).toBeTruthy();
  });
});
