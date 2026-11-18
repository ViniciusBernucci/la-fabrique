import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { IssuesView } from "../lib/api";
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

const view = (over: Partial<IssuesView> = {}): IssuesView => ({
  issues: [
    {
      id: "i1",
      source: "jira",
      remote_id: "1",
      key: "ABC-1",
      title: "Cadastro de clientes",
      raw_status: "To Do",
      status_code: "PRD-DOC",
      revision: "v1",
      kind: "feature",
    },
  ],
  runs: [
    {
      id: 1,
      trigger: "daily",
      scope: "all",
      outcome: "partial",
      read: 4,
      new: 1,
      updated: 0,
      ignored: 0,
      error: "limite de requisições da plataforma atingido",
      started_at: "2026-11-13T10:00:00Z",
      finished_at: "2026-11-13T10:00:02Z",
    },
  ],
  pending: [
    {
      issue: "ABC-1",
      version: 2,
      target: "PRD-DOC-CONCLUIDO",
      reason: "mapping",
      detail: "sem status ou transição remota mapeada para PRD-DOC-CONCLUIDO",
      attempts: 1,
      updated_at: "2026-11-13T10:00:05Z",
    },
  ],
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
}

const base = {
  "GET /session": () => json(session),
  "GET /projects?limit=100": () =>
    json({ items: [{ id: "p1", name: "Projeto Um" }], next_cursor: null }),
};

function renderIssues() {
  return render(
    <MemoryRouter initialEntries={["/issues"]}>
      <SessionProvider>
        <App />
      </SessionProvider>
    </MemoryRouter>,
  );
}

describe("página Issues", () => {
  beforeEach(() => vi.unstubAllGlobals());
  afterEach(() => vi.unstubAllGlobals());

  it("mostra Issues, status canônico e histórico com falha parcial", async () => {
    stub({ ...base, "GET /projects/p1/issues": () => json(view()) });
    renderIssues();
    expect(await screen.findByText(/ABC-1 · Cadastro de clientes/)).toBeTruthy();
    expect(screen.getByText("PRD-DOC")).toBeTruthy();
    expect(screen.getByText("Parcial")).toBeTruthy();
    expect(screen.getByText(/limite de requisições/)).toBeTruthy();
    expect(screen.getByText("Mapeamento ausente")).toBeTruthy();
    expect(screen.getByText(/ABC-1 → PRD-DOC-CONCLUIDO/)).toBeTruthy();
  });

  it("mostra o avanço por trilha, o que segura o avanço e a projeção atrasada", async () => {
    stub({
      ...base,
      "GET /projects/p1/issues": () =>
        json(
          view({
            progress: {
              "ABC-1": {
                key: "ABC-1",
                internal_status: "IMPLEMENTACAO",
                aggregate_status: "IMPLEMENTACAO",
                held_by: "BACKEND",
                tracks: [
                  { track: "FRONTEND", stage: "reviewed" },
                  { track: "BACKEND", stage: "implementing" },
                ],
                blocking: [
                  "a trilha backend é a etapa obrigatória mais atrasada e segura o avanço",
                ],
                commits: ["abc123"],
                evidence: [{ track: "FRONTEND", revision: "r1" }],
                publications: [],
                projection: {
                  internal: "IMPLEMENTACAO",
                  external: "PLANEJAMENTO",
                  in_sync: false,
                  pending: "write: falha de escrita",
                },
              },
            },
          }),
        ),
    });
    renderIssues();
    expect(await screen.findByText(/FRONTEND: revisada · BACKEND: implementando/)).toBeTruthy();
    expect(screen.getByText(/segura o avanço: BACKEND/)).toBeTruthy();
    expect(screen.getByText(/Bloqueio: a trilha backend/)).toBeTruthy();
    expect(screen.getByText(/origem mostra PLANEJAMENTO \(projeção atrasada: write/)).toBeTruthy();
    expect(screen.getByText(/1 commit\(s\) · 1 evidência\(s\)/)).toBeTruthy();
  });

  it("sincroniza agora e recarrega a lista", async () => {
    let synced = false;
    stub({
      ...base,
      "GET /projects/p1/issues": () =>
        json(synced ? view() : view({ issues: [], runs: [], pending: [] })),
      "POST /projects/p1/issues/sync": () => {
        synced = true;
        return json({
          id: 2,
          trigger: "manual",
          scope: "all",
          outcome: "ok",
          read: 4,
          new: 1,
          updated: 0,
          ignored: 3,
          started_at: "2026-11-13T10:00:00Z",
          finished_at: "2026-11-13T10:00:01Z",
        });
      },
    });
    renderIssues();
    expect(await screen.findByText("Nenhuma Issue importada")).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Sincronizar agora" }));
    await waitFor(() => expect(screen.getByText(/Sincronizado: 1 nova/)).toBeTruthy());
    expect(await screen.findByText(/ABC-1 · Cadastro de clientes/)).toBeTruthy();
  });

  it("mostra o erro da API ao sincronizar sem mapa de status", async () => {
    stub({
      ...base,
      "GET /projects/p1/issues": () => json(view({ issues: [], runs: [], pending: [] })),
      "POST /projects/p1/issues/sync": () =>
        json(
          {
            error: {
              code: "status_map_missing",
              message: "O mapa de status do projeto ainda não foi configurado.",
              action: "Configure o mapa de status antes de sincronizar.",
              correlation_id: "c",
            },
          },
          422,
        ),
    });
    renderIssues();
    await userEvent.click(await screen.findByRole("button", { name: "Sincronizar agora" }));
    expect(
      await screen.findByText(/mapa de status do projeto ainda não foi configurado/),
    ).toBeTruthy();
  });

  it("só oferece executar Issues liberadas e inicia a execução pela fábrica", async () => {
    const released = view({
      issues: [
        {
          id: "i1",
          source: "jira",
          remote_id: "1",
          key: "ABC-1",
          title: "Em PRD",
          raw_status: "To Do",
          status_code: "PRD-DOC",
          revision: "v1",
          kind: "feature",
        },
        {
          id: "i2",
          source: "jira",
          remote_id: "2",
          key: "ABC-2",
          title: "Pronta",
          raw_status: "Ready",
          status_code: "PRD-DOC-CONCLUIDO",
          revision: "v1",
          kind: "feature",
        },
      ],
    });
    let posted = "";
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
        const url = String(input).replace(/^\/api/, "");
        const key = `${(init.method ?? "GET").toUpperCase()} ${url}`;
        if (key === "GET /session") return json(session);
        if (key === "GET /projects?limit=100")
          return json({ items: [{ id: "p1", name: "Projeto Um" }], next_cursor: null });
        if (key === "GET /projects/p1/issues") return json(released);
        if (key === "POST /projects/p1/issues/ABC-2/run") {
          posted = key;
          return json(
            { execution_id: "e1", issue_key: "ABC-2", title: "x", status: "queued" },
            201,
          );
        }
        return json(
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
    renderIssues();
    await screen.findByText(/ABC-2 · Pronta/);
    expect(screen.queryByRole("button", { name: "Executar ABC-1 na fábrica" })).toBeNull();
    await userEvent.click(screen.getByRole("button", { name: "Executar ABC-2 na fábrica" }));
    expect(await screen.findByText(/Execução de ABC-2 iniciada/)).toBeTruthy();
    expect(posted).toBe("POST /projects/p1/issues/ABC-2/run");
  });
});
