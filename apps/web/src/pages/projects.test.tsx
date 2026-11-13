import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { Connection, Project, ValidationReport } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { suggestSlug } from "./NewProjectPage";
import { parsePriorityMap, parseStatuses } from "./ProjectSetupPage";

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
  json({ error: { code, message, action: "Tente.", correlation_id: "corr-1" } }, status);

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

const ID = "11111111-1111-4111-8111-111111111111";
const GH = "22222222-2222-4222-8222-222222222222";

function project(over: Partial<Project> = {}): Project {
  return {
    id: ID,
    slug: "meu-app",
    name: "Meu app",
    description: "",
    state: "draft",
    priority: 2,
    wizard_step: "repository",
    repository: { owner: "", name: "", base_branch: "", connection_id: null },
    source: {
      kind: "",
      connection_id: null,
      container: "",
      eligibility_marker: "",
      eligible_statuses: [],
      priority_map: {},
      access: "read",
    },
    version: 2,
    activated_at: null,
    created_at: "2026-11-12T10:00:00Z",
    updated_at: "2026-11-12T10:00:00Z",
    ...over,
  };
}

function connection(over: Partial<Connection>): Connection {
  return {
    id: GH,
    kind: "github",
    name: "github-principal",
    target: "",
    scopes: "",
    state: "active",
    secret_version: 1,
    last_test_status: "not_run",
    last_test_detail: "",
    version: 1,
    created_at: "2026-11-12T10:00:00Z",
    updated_at: "2026-11-12T10:00:00Z",
    ...over,
  };
}

const connections = {
  items: [
    connection({}),
    connection({
      id: "33333333-3333-4333-8333-333333333333",
      name: "github-revogada",
      state: "revoked",
    }),
    connection({ id: "44444444-4444-4444-8444-444444444444", kind: "jira", name: "jira-time" }),
  ],
};

const pendingReport: ValidationReport = {
  ok: false,
  not_verified: ["Acesso remoto ao repositório e à origem de tarefas (T05-02)"],
  checks: [
    { id: "basics.name", area: "basics", label: "Nome do projeto", status: "ok" },
    {
      id: "repository.owner",
      area: "repository",
      label: "Dono do repositório GitHub",
      status: "pending",
      action: "Informe o dono na etapa Repositório.",
    },
  ],
};

const okReport: ValidationReport = {
  ok: true,
  not_verified: pendingReport.not_verified,
  checks: [{ id: "basics.name", area: "basics", label: "Nome do projeto", status: "ok" }],
};

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("suggestSlug", () => {
  it("normaliza acentos, espaços e símbolos", () => {
    expect(suggestSlug("  Meu Ápp — Teste! ")).toBe("meu-app-teste");
    expect(suggestSlug("!!!")).toBe("");
  });
});

describe("critérios da origem", () => {
  it("interpreta estados e mapa de prioridade, recusando valores inválidos", () => {
    expect(parseStatuses(" To Do, Pronto para dev ,, ")).toEqual(["To Do", "Pronto para dev"]);
    expect(parsePriorityMap("High=1, medium = 2")).toEqual({ high: 1, medium: 2 });
    expect(parsePriorityMap("")).toEqual({});
    for (const bad of ["high", "high=9", "high=x", "=1", "a=1=2", "high=1.5"]) {
      expect(parsePriorityMap(bad)).toEqual(expect.stringContaining("inválido"));
    }
  });
});

describe("cadastro de projeto", () => {
  it("cria o rascunho com CSRF e abre o assistente na etapa Repositório", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      "POST /projects": () => json(project(), 201),
      [`GET /projects/${ID}`]: () => json(project()),
      "GET /connections": () => json(connections),
    });
    renderAt("/projects/new");
    await userEvent.type(await screen.findByLabelText("Nome"), "Meu app");
    expect(screen.getByLabelText("Identificador")).toHaveValue("meu-app");
    await userEvent.click(screen.getByRole("button", { name: "Criar rascunho e continuar" }));

    expect(await screen.findByRole("heading", { name: "Repositório GitHub" })).toBeInTheDocument();
    const create = calls.find((c) => c.method === "POST" && c.url === "/projects");
    expect(create?.csrf).toBe("csrf-1");
    expect(create?.body).toMatchObject({ name: "Meu app", slug: "meu-app", priority: 2 });
  });

  it("mostra o motivo quando o servidor recusa o cadastro", async () => {
    stub({
      "GET /session": () => json(session),
      "POST /projects": () => err(409, "duplicate", "Já existe um registro com esses dados."),
    });
    renderAt("/projects/new");
    await userEvent.type(await screen.findByLabelText("Nome"), "Meu app");
    await userEvent.click(screen.getByRole("button", { name: "Criar rascunho e continuar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Já existe um registro");
  });

  it("persiste a etapa Repositório com a versão carregada e só oferece conexões GitHub ativas", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(project()),
      "GET /connections": () => json(connections),
      [`PATCH /projects/${ID}`]: (c) =>
        json(
          project({ version: 3, wizard_step: "source", repository: c.body?.repository as never }),
        ),
    });
    renderAt(`/projects/${ID}/setup`);
    await screen.findByRole("heading", { name: "Repositório GitHub" });

    const select = screen.getByRole("combobox", { name: /Conexão GitHub/ });
    const options = within(select)
      .getAllByRole("option")
      .map((o) => o.textContent);
    expect(options).toContain("github-principal");
    expect(options).not.toContain("github-revogada");
    expect(options).not.toContain("jira-time");

    await userEvent.type(screen.getByLabelText(/Dono/), "org-exemplo");
    await userEvent.type(screen.getByRole("textbox", { name: "Repositório" }), "app");
    await userEvent.type(screen.getByLabelText("Branch base"), "main");
    await userEvent.selectOptions(select, GH);
    await userEvent.click(screen.getByRole("button", { name: "Salvar e continuar" }));

    expect(await screen.findByRole("heading", { name: "Origem de tarefas" })).toBeInTheDocument();
    const patch = calls.find((c) => c.method === "PATCH");
    expect(patch?.csrf).toBe("csrf-1");
    expect(patch?.body).toEqual({
      expected_version: 2,
      wizard_step: "source",
      repository: { owner: "org-exemplo", name: "app", base_branch: "main", connection_id: GH },
    });
  });

  it("retoma o rascunho na etapa em que parou", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(project({ wizard_step: "source" })),
      "GET /connections": () => json(connections),
    });
    renderAt(`/projects/${ID}/setup`);
    expect(await screen.findByRole("heading", { name: "Origem de tarefas" })).toBeInTheDocument();
  });

  it("trocar a plataforma da origem descarta a conexão e o identificador anteriores", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () =>
        json(
          project({
            wizard_step: "source",
            source: {
              kind: "jira",
              connection_id: "44444444-4444-4444-8444-444444444444",
              container: "ABC",
              eligibility_marker: "",
              eligible_statuses: [],
              priority_map: {},
              access: "read",
            },
          }),
        ),
      "GET /connections": () => json(connections),
    });
    renderAt(`/projects/${ID}/setup`);
    await screen.findByRole("heading", { name: "Origem de tarefas" });
    expect(screen.getByLabelText("Projeto ou lista")).toHaveValue("ABC");
    await userEvent.selectOptions(screen.getByLabelText("Plataforma"), "clickup");
    expect(screen.getByLabelText("Projeto ou lista")).toHaveValue("");
    expect(screen.getByRole("combobox", { name: /^Conexão/ })).toHaveValue("");
  });

  it("revisão lista pendências com ação, bloqueia a ativação e leva à etapa certa", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(project({ wizard_step: "review" })),
      "GET /connections": () => json(connections),
      [`GET /projects/${ID}/validation`]: () => json(pendingReport),
    });
    renderAt(`/projects/${ID}/setup`);
    expect(await screen.findByText(/1 pendência impede/)).toBeInTheDocument();
    expect(screen.getByText("Informe o dono na etapa Repositório.")).toBeInTheDocument();
    expect(screen.getByText(/Ainda não verificado/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Ativar projeto" })).toBeDisabled();

    await userEvent.click(screen.getByRole("button", { name: "Ir para a etapa" }));
    expect(await screen.findByRole("heading", { name: "Repositório GitHub" })).toBeInTheDocument();
  });

  it("ativa quando válido e vai para o detalhe", async () => {
    let active = false;
    const calls = stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () =>
        json(
          active ? project({ state: "active", version: 3 }) : project({ wizard_step: "review" }),
        ),
      "GET /connections": () => json(connections),
      [`GET /projects/${ID}/validation`]: () => json(okReport),
      [`GET /projects/${ID}/changes`]: () => json({ items: [] }),
      [`POST /projects/${ID}/activate`]: () => {
        active = true;
        return json(project({ state: "active", version: 3 }));
      },
    });
    renderAt(`/projects/${ID}/setup`);
    const button = await screen.findByRole("button", { name: "Ativar projeto" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);
    expect(await screen.findByRole("button", { name: "Suspender" })).toBeInTheDocument();
    expect(calls.find((c) => c.url.endsWith("/activate"))?.body).toEqual({ expected_version: 2 });
  });

  it("exibe o bloqueio do servidor se a ativação for recusada", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(project({ wizard_step: "review" })),
      "GET /connections": () => json(connections),
      [`GET /projects/${ID}/validation`]: () => json(okReport),
      [`POST /projects/${ID}/activate`]: () =>
        err(422, "validation_failed", "O projeto tem pendências e não pode ser ativado."),
    });
    renderAt(`/projects/${ID}/setup`);
    const button = await screen.findByRole("button", { name: "Ativar projeto" });
    await waitFor(() => expect(button).toBeEnabled());
    await userEvent.click(button);
    expect(await screen.findByRole("alert")).toHaveTextContent("não pode ser ativado");
  });
});

describe("detalhe do projeto", () => {
  it("rascunho mostra pendências, nenhum recurso associado e o caminho para continuar", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(project()),
      "GET /connections": () => json(connections),
      [`GET /projects/${ID}/validation`]: () => json(pendingReport),
      [`GET /projects/${ID}/changes`]: () =>
        json({
          items: [{ id: 1, action: "created", detail: {}, created_at: "2026-11-12T10:00:00Z" }],
        }),
    });
    renderAt(`/projects/${ID}`);
    expect(await screen.findByRole("link", { name: "Continuar cadastro" })).toHaveAttribute(
      "href",
      `/projects/${ID}/setup`,
    );
    expect(await screen.findByText("Informe o dono na etapa Repositório.")).toBeInTheDocument();
    expect(screen.getByText("Nenhum associado")).toBeInTheDocument();
    expect(await screen.findByText("Rascunho criado")).toBeInTheDocument();
  });

  it("suspende um projeto ativo enviando a versão atual", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(project({ state: "active", version: 5 })),
      "GET /connections": () => json(connections),
      [`GET /projects/${ID}/changes`]: () => json({ items: [] }),
      [`POST /projects/${ID}/suspend`]: () => json(project({ state: "paused", version: 6 })),
    });
    renderAt(`/projects/${ID}`);
    await userEvent.click(await screen.findByRole("button", { name: "Suspender" }));
    await waitFor(() =>
      expect(calls.find((c) => c.url.endsWith("/suspend"))?.body).toEqual({ expected_version: 5 }),
    );
  });
});

describe("verificação da origem", () => {
  const JIRA = "44444444-4444-4444-8444-444444444444";
  const sourceProject = (over: Partial<Project["source"]> = {}) =>
    project({
      wizard_step: "source",
      source: {
        kind: "jira",
        connection_id: JIRA,
        container: "ABC",
        eligibility_marker: "lf-ready",
        eligible_statuses: ["To Do"],
        priority_map: { high: 1 },
        access: "read",
        ...over,
      },
    });

  it("salva a origem com os critérios e mostra o preview dos itens elegíveis", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(sourceProject()),
      "GET /connections": () => json(connections),
      [`PATCH /projects/${ID}`]: (c) =>
        json(project({ version: 3, wizard_step: "source", source: c.body?.source as never })),
      [`POST /projects/${ID}/source/check`]: () =>
        json({
          ok: true,
          has_more: true,
          checked_at: "2026-11-12T10:00:00Z",
          steps: [
            { id: "credential", label: "Credencial aceita", status: "ok", detail: "Conta Bot." },
            {
              id: "write",
              label: "Permissão de escrita",
              status: "skipped",
              detail: "Só no primeiro uso.",
            },
          ],
          items: [
            {
              key: "ABC-1",
              title: "Cadastro",
              status: "To Do",
              priority: "High",
              mapped_priority: 1,
              subtask: false,
              url: "",
            },
            {
              key: "ABC-4",
              title: "Outro",
              status: "To Do",
              priority: "Highest",
              mapped_priority: null,
              subtask: false,
              url: "",
            },
          ],
        }),
    });
    renderAt(`/projects/${ID}/setup`);
    await screen.findByRole("heading", { name: "Origem de tarefas" });
    expect(screen.getByLabelText("Marcador de elegibilidade")).toHaveValue("lf-ready");
    await userEvent.clear(screen.getByLabelText("Mapa de prioridade"));
    await userEvent.type(screen.getByLabelText("Mapa de prioridade"), "high=1, medium=2");
    await userEvent.click(screen.getByRole("button", { name: "Salvar e verificar origem" }));

    expect(await screen.findByText("Origem verificada.")).toBeInTheDocument();
    expect(screen.getByText("ABC-4")).toBeInTheDocument();
    expect(screen.getByText(/Highest → sem mapa/)).toBeInTheDocument();
    expect(screen.getByText(/Há mais itens/)).toBeInTheDocument();
    const patch = calls.find((c) => c.method === "PATCH");
    expect(patch?.body).toMatchObject({
      expected_version: 2,
      wizard_step: "source",
      source: {
        eligibility_marker: "lf-ready",
        eligible_statuses: ["To Do"],
        priority_map: { high: 1, medium: 2 },
        access: "read",
      },
    });
    // Salvar antes de verificar: a verificação sempre vale para o que foi salvo.
    const order = calls.map((c) => `${c.method} ${c.url}`);
    expect(order.indexOf(`PATCH /projects/${ID}`)).toBeLessThan(
      order.indexOf(`POST /projects/${ID}/source/check`),
    );
  });

  it("mostra o motivo quando a origem falha e não chama a verificação com mapa inválido", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      [`GET /projects/${ID}`]: () => json(sourceProject()),
      "GET /connections": () => json(connections),
      [`PATCH /projects/${ID}`]: () => json(sourceProject()),
      [`POST /projects/${ID}/source/check`]: () =>
        json({
          ok: false,
          has_more: false,
          checked_at: "2026-11-12T10:00:00Z",
          items: [],
          steps: [
            {
              id: "credential",
              label: "Credencial aceita",
              status: "failed",
              detail: "A credencial não tem permissão para este projeto ou lista.",
            },
          ],
        }),
    });
    renderAt(`/projects/${ID}/setup`);
    await screen.findByRole("heading", { name: "Origem de tarefas" });

    await userEvent.clear(screen.getByLabelText("Mapa de prioridade"));
    await userEvent.type(screen.getByLabelText("Mapa de prioridade"), "high=9");
    await userEvent.click(screen.getByRole("button", { name: "Salvar e verificar origem" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Mapa de prioridade inválido");
    expect(calls.some((c) => c.method === "PATCH")).toBe(false);

    await userEvent.clear(screen.getByLabelText("Mapa de prioridade"));
    await userEvent.type(screen.getByLabelText("Mapa de prioridade"), "high=1");
    await userEvent.click(screen.getByRole("button", { name: "Salvar e verificar origem" }));
    expect(await screen.findByText("A verificação da origem falhou.")).toBeInTheDocument();
    expect(screen.getByText(/não tem permissão/)).toBeInTheDocument();
    expect(screen.queryByText("Itens elegíveis agora")).not.toBeInTheDocument();
  });
});
