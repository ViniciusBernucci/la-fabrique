import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { ProjectOverview } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { duration } from "./RunPanel";

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
const EID = "22222222-2222-4222-8222-222222222222";

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

function project() {
  return {
    id: PID,
    slug: "demo",
    name: "Demo",
    description: "",
    state: "active",
    priority: 2,
    wizard_step: "review",
    repository: { owner: "o", name: "r", base_branch: "main", connection_id: null },
    source: {
      kind: "jira",
      connection_id: null,
      container: "ABC",
      eligibility_marker: "",
      eligible_statuses: [],
      priority_map: {},
      access: "read",
    },
    version: 3,
    activated_at: null,
    created_at: "2026-11-01T10:00:00Z",
    updated_at: "2026-11-02T10:00:00Z",
  };
}

const overview: ProjectOverview = {
  project: project() as unknown as ProjectOverview["project"],
  controls: {
    global: { scope: "global", mode: "draining", reason: "fechar antes do deploy", version: 2 },
    project: {
      scope: "project",
      project_id: PID,
      mode: "suspended",
      reason: "incidente",
      version: 1,
    },
  },
  counts: { running: 0, queued: 1, waiting: 1, done: 0 },
  items: [
    {
      id: "i1",
      feature_id: "f",
      task_key: "schema",
      kind: "task",
      priority: 0,
      state: "waiting",
      wait_reason: "decision",
      blocked: "Aguardando uma decisão humana na Central de Decisões",
      enqueued_at: "2026-11-12T10:00:00Z",
      started_at: "2026-11-12T10:01:00Z",
      seconds: 125,
    },
    {
      id: "i2",
      feature_id: "f",
      task_key: "api",
      kind: "task",
      priority: 1,
      state: "queued",
      blocked: 'o pré-requisito "schema" ainda não foi concluído; o projeto está suspenso',
      enqueued_at: "2026-11-12T10:00:00Z",
      seconds: 0,
    },
  ],
  executions: [{ id: EID, feature_id: "f", state: "waiting", attempt: 1, last_seq: 2 }],
  last_sync: {
    outcome: "ok",
    finished_at: "2026-11-12T09:00:00Z",
    new: 1,
    changed: 0,
    gone: 0,
    conflicts: 0,
  },
};

class FakeEventSource {
  static all: FakeEventSource[] = [];
  onopen: (() => void) | null = null;
  onmessage: ((m: MessageEvent<string>) => void) | null = null;
  onerror: (() => void) | null = null;
  listeners: Record<string, () => void> = {};
  closed = false;
  constructor(public url: string) {
    FakeEventSource.all.push(this);
  }
  addEventListener(name: string, fn: () => void) {
    this.listeners[name] = fn;
  }
  close() {
    this.closed = true;
  }
  emit(seq: number, type: string, payload: unknown) {
    this.onmessage?.({
      data: JSON.stringify({ seq, type, payload, created_at: "2026-11-12T10:00:00Z" }),
    } as MessageEvent<string>);
  }
}

function stub(extra: Record<string, () => Response> = {}) {
  const calls: string[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input).replace(/^\/api/, "");
      calls.push(`${(init.method ?? "GET").toUpperCase()} ${url}`);
      const routes: Record<string, () => Response> = {
        "GET /session": () => json(session),
        [`GET /projects/${PID}`]: () => json(project()),
        [`GET /projects/${PID}/validation`]: () => json({ ok: true, checks: [], not_verified: [] }),
        [`GET /projects/${PID}/changes`]: () => json({ items: [] }),
        "GET /connections": () => json({ items: [] }),
        [`GET /projects/${PID}/overview`]: () => json(overview),
        ...extra,
      };
      const h = routes[`${(init.method ?? "GET").toUpperCase()} ${url}`];
      return h
        ? h()
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

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SessionProvider>
        <App />
      </SessionProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  FakeEventSource.all = [];
  vi.stubGlobal("EventSource", FakeEventSource);
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("Execução do projeto", () => {
  it("mostra os controles persistidos e a causa de cada bloqueio", async () => {
    stub();
    renderAt(`/projects/${PID}`);
    expect(
      await screen.findByText(/Pausa após a feature atual: nenhuma feature nova inicia/),
    ).toBeInTheDocument();
    expect(screen.getByText(/Motivo: fechar antes do deploy/)).toBeInTheDocument();
    expect(screen.getByText(/Suspenso: nada novo inicia/)).toBeInTheDocument();
    expect(screen.getByText(/Motivo: incidente/)).toBeInTheDocument();
    expect(
      screen.getByText("Bloqueio: Aguardando uma decisão humana na Central de Decisões"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/ainda não foi concluído; o projeto está suspenso/),
    ).toBeInTheDocument();
    expect(screen.getByText("Tempo: 2 min")).toBeInTheDocument();
    expect(screen.getByText(/ok · /)).toBeInTheDocument();
  });

  it("recupera os eventos pelo cursor ao reconectar e ignora os já vistos", async () => {
    stub();
    renderAt(`/projects/${PID}`);
    const select = await screen.findByRole("combobox", { name: "Execução" });
    await userEvent.selectOptions(select, EID);
    await waitFor(() => expect(FakeEventSource.all).toHaveLength(1));
    const first = FakeEventSource.all[0];
    expect(first.url).toBe(`/api/projects/${PID}/executions/${EID}/stream?after=0`);
    act(() => {
      first.onopen?.();
      first.emit(1, "step", { n: 1 });
      first.emit(2, "step", { n: 2 });
      first.emit(2, "step", { n: "repetido" });
    });
    expect(await screen.findByText(/#2 step/)).toBeInTheDocument();
    expect(screen.getByText("Ao vivo")).toBeInTheDocument();
    expect(screen.queryByText(/repetido/)).not.toBeInTheDocument();
    // A conexão cai: o painel reabre pelo cursor (último seq visto = 2).
    vi.useFakeTimers({ shouldAdvanceTime: true });
    act(() => first.onerror?.());
    expect(screen.getByText("Reconectando pelo cursor…")).toBeInTheDocument();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1100);
    });
    vi.useRealTimers();
    expect(FakeEventSource.all).toHaveLength(2);
    expect(first.closed).toBe(true);
    const second = FakeEventSource.all[1];
    expect(second.url).toBe(`/api/projects/${PID}/executions/${EID}/stream?after=2`);
    act(() => {
      second.onopen?.();
      second.emit(2, "step", { n: "de novo" }); // o servidor não repetiria, mas o cliente também protege
      second.emit(3, "checkpoint", { n: 3 });
    });
    expect(await screen.findByText(/#3 checkpoint/)).toBeInTheDocument();
    expect(screen.queryByText(/de novo/)).not.toBeInTheDocument();
    // O servidor pede reconexão ao fim do tempo máximo: reabre logo, pelo cursor 3.
    await act(async () => {
      second.listeners.reconnect?.();
      await new Promise((r) => setTimeout(r, 20));
    });
    expect(FakeEventSource.all[2].url).toBe(
      `/api/projects/${PID}/executions/${EID}/stream?after=3`,
    );
  });

  it("busca no histórico e volta ao ao vivo", async () => {
    stub({
      [`GET /projects/${PID}/executions/${EID}/events?limit=200&q=erro`]: () =>
        json({
          items: [
            {
              seq: 7,
              type: "error",
              payload: { msg: "falha de rede" },
              created_at: "2026-11-12T10:00:00Z",
            },
          ],
          next_cursor: 7,
        }),
    });
    renderAt(`/projects/${PID}`);
    await userEvent.selectOptions(await screen.findByRole("combobox", { name: "Execução" }), EID);
    await userEvent.type(await screen.findByLabelText("Buscar no histórico"), "erro");
    await userEvent.click(screen.getByRole("button", { name: "Buscar" }));
    expect(await screen.findByText(/#7 error/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Voltar ao ao vivo" }));
    expect(await screen.findByText("Nenhum evento ainda.")).toBeInTheDocument();
  });

  it("formata a duração", () => {
    expect(duration(5)).toBe("5 s");
    expect(duration(125)).toBe("2 min");
    expect(duration(3725)).toBe("1 h 2 min");
  });

  it("rascunho não mostra execução", async () => {
    stub({ [`GET /projects/${PID}`]: () => json({ ...project(), state: "draft" }) });
    renderAt(`/projects/${PID}`);
    expect(await screen.findByRole("heading", { name: "Demo" })).toBeInTheDocument();
    expect(screen.queryByLabelText("Execução", { selector: "section" })).not.toBeInTheDocument();
  });
});
