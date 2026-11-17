import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { Agent, Connection, EffectiveAgent, Model, NodeType, Workflow } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { connectionsFor } from "./AgentsPage";

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
  json({ error: { code, message, action: "Tente.", correlation_id: "c-1" } }, status);

interface Call {
  method: string;
  url: string;
  body?: Record<string, unknown>;
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

const CLAUDE_A = "11111111-1111-4111-8111-111111111111";
const CLAUDE_B = "22222222-2222-4222-8222-222222222222";
const CODEX = "33333333-3333-4333-8333-333333333333";
const GATEWAY = "44444444-4444-4444-8444-444444444444";
const AGENT = "55555555-5555-4555-8555-555555555555";
const WF = "66666666-6666-4666-8666-666666666666";

function conn(
  id: string,
  name: string,
  kind: Connection["kind"],
  target: string,
  state: Connection["state"] = "active",
): Connection {
  return {
    id,
    kind,
    name,
    target,
    scopes: "",
    state,
    secret_version: 1,
    last_test_status: "not_run",
    last_test_detail: "",
    version: 1,
    created_at: "2026-11-12T10:00:00Z",
    updated_at: "2026-11-12T10:00:00Z",
  };
}

const connections = {
  items: [
    conn(CLAUDE_A, "claude-conta-a", "cli", "claude"),
    conn(CLAUDE_B, "claude-conta-b", "cli", "claude"),
    conn(CODEX, "codex-conta", "cli", "codex"),
    conn(GATEWAY, "gateway", "api", "openrouter"),
    conn("77777777-7777-4777-8777-777777777777", "claude-revogada", "cli", "claude", "revoked"),
  ],
};

const mdl = (
  provider: Model["provider"],
  name: string,
  label: string,
  over: Partial<Model> = {},
): Model => ({
  id: `m-${provider}-${name}`,
  provider,
  gateway: "",
  name,
  label,
  supports_tools: true,
  supports_streaming: true,
  context_window: 0,
  max_output_tokens: 0,
  state: "active",
  version: 1,
  ...over,
});

const catalogModels: { items: Model[] } = {
  items: [
    mdl("claude", "claude-opus-5-5", "Claude Opus 5.5"),
    mdl("claude", "claude-sonnet-5-5", "Claude Sonnet 5.5"),
    mdl("claude", "claude-velho", "Claude Velho", { state: "disabled" }),
    mdl("codex", "gpt-6.1-sol", "GPT-6.1 Sol"),
    mdl("codex", "gpt-5.6-luna", "GPT-5.6 Luna"),
    mdl("api", "anthropic/claude-sonnet-5-5", "Sonnet via OpenRouter", { gateway: "openrouter" }),
    mdl("api", "vercel/modelo", "Modelo Vercel", { gateway: "vercel" }),
  ],
};

const modelOptions = () =>
  within(screen.getByRole("combobox", { name: "Modelo" }))
    .getAllByRole("option")
    .map((o) => o.textContent);

const agent: Agent = {
  id: AGENT,
  name: "dev",
  description: "",
  instructions: "Seja objetivo.",
  provider: "claude",
  connection_id: CLAUDE_A,
  model: "claude-sonnet-5-5",
  skills: ["revisao"],
  tools: ["Read", "Edit"],
  fallback: [],
  timeout_minutes: 30,
  version: 3,
  created_at: "2026-11-12T10:00:00Z",
  updated_at: "2026-11-12T10:00:00Z",
};

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("catálogo de agentes", () => {
  it("só oferece contas ativas que servem ao provedor", () => {
    const names = (p: "claude" | "codex" | "api") =>
      connectionsFor(p, connections.items).map((c) => c.name);
    expect(names("claude")).toEqual(["claude-conta-a", "claude-conta-b"]);
    expect(names("codex")).toEqual(["codex-conta"]);
    expect(names("api")).toEqual(["gateway"]);
  });

  it("cadastra um agente com a conta escolhida e as listas interpretadas", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      "GET /agents": () => json({ items: [] }),
      "GET /connections": () => json(connections),
      "GET /models": () => json(catalogModels),
      "POST /agents": () => json(agent, 201),
    });
    renderAt("/agents");
    await userEvent.type(await screen.findByLabelText("Nome"), "dev");
    const select = screen.getByRole("combobox", { name: "Conta (conexão)" });
    expect(
      within(select)
        .getAllByRole("option")
        .map((o) => o.textContent),
    ).not.toContain("codex-conta");
    await userEvent.selectOptions(select, CLAUDE_B);
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Modelo" }),
      "claude-sonnet-5-5",
    );
    await userEvent.type(screen.getByLabelText(/^Ferramentas \(separadas/), "Read, Edit ,, Bash");
    await userEvent.type(screen.getByLabelText("Instruções"), "Seja objetivo.");
    await userEvent.click(screen.getByRole("button", { name: "Cadastrar agente" }));

    await screen.findByText("Nenhum agente"); // recarrega a lista
    const post = calls.find((c) => c.method === "POST");
    expect(post?.body).toMatchObject({
      name: "dev",
      provider: "claude",
      connection_id: CLAUDE_B,
      model: "claude-sonnet-5-5",
      tools: ["Read", "Edit", "Bash"],
      instructions: "Seja objetivo.",
    });
  });

  it("carrega a lista de modelos de cada LLM e a troca ao mudar o provedor", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /agents": () => json({ items: [] }),
      "GET /connections": () => json(connections),
      "GET /models": () => json(catalogModels),
    });
    renderAt("/agents");
    await screen.findByRole("combobox", { name: "Modelo" });
    // Claude: só os ativos do Claude (o desativado e os de outros provedores não aparecem)
    expect(modelOptions()).toEqual([
      "Padrão da conta",
      "Claude Opus 5.5 (claude-opus-5-5)",
      "Claude Sonnet 5.5 (claude-sonnet-5-5)",
      "Outro (digitar o nome)…",
    ]);
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Modelo" }),
      "claude-opus-5-5",
    );
    await userEvent.selectOptions(screen.getByLabelText("Provedor"), "codex");
    // Codex: lista própria, e a escolha do Claude foi descartada
    expect(modelOptions()).toEqual([
      "Padrão da conta",
      "GPT-5.6 Luna (gpt-5.6-luna)",
      "GPT-6.1 Sol (gpt-6.1-sol)",
      "Outro (digitar o nome)…",
    ]);
    expect(screen.getByRole("combobox", { name: "Modelo" })).toHaveValue("");
  });

  it("no provedor api lista os modelos do gateway da conta escolhida", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /agents": () => json({ items: [] }),
      "GET /connections": () => json(connections),
      "GET /models": () => json(catalogModels),
    });
    renderAt("/agents");
    await userEvent.selectOptions(await screen.findByLabelText("Provedor"), "api");
    // sem conta escolhida: todos os modelos de API
    expect(modelOptions()).toContain("Sonnet via OpenRouter (anthropic/claude-sonnet-5-5)");
    expect(modelOptions()).toContain("Modelo Vercel (vercel/modelo)");
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Conta (conexão)" }),
      GATEWAY,
    );
    // a conexão "gateway" tem destino openrouter: só os modelos desse gateway
    expect(modelOptions()).toContain("Sonnet via OpenRouter (anthropic/claude-sonnet-5-5)");
    expect(modelOptions()).not.toContain("Modelo Vercel (vercel/modelo)");
  });

  it("permite digitar um modelo fora do catálogo e funciona sem a lista", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      "GET /agents": () => json({ items: [] }),
      "GET /connections": () => json(connections),
      // sem GET /models: a lista falha, o campo continua utilizável
      "POST /agents": () => json(agent, 201),
    });
    renderAt("/agents");
    await userEvent.type(await screen.findByLabelText("Nome"), "dev");
    expect(
      await screen.findByText(/Nenhum modelo cadastrado para este provedor/),
    ).toBeInTheDocument();
    await userEvent.selectOptions(
      screen.getByRole("combobox", { name: "Modelo" }),
      "Outro (digitar o nome)…",
    );
    await userEvent.type(screen.getByLabelText("Nome do modelo"), "modelo-novo");
    await userEvent.type(screen.getByLabelText("Instruções"), "Seja objetivo.");
    await userEvent.click(screen.getByRole("button", { name: "Cadastrar agente" }));
    await screen.findByText("Nenhum agente");
    expect(calls.find((c) => c.method === "POST")?.body).toMatchObject({ model: "modelo-novo" });
  });

  it("trocar o provedor descarta a conta escolhida", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /agents": () => json({ items: [] }),
      "GET /connections": () => json(connections),
    });
    renderAt("/agents");
    const select = await screen.findByRole("combobox", { name: "Conta (conexão)" });
    await userEvent.selectOptions(select, CLAUDE_A);
    expect(select).toHaveValue(CLAUDE_A);
    await userEvent.selectOptions(screen.getByLabelText("Provedor"), "codex");
    expect(screen.getByRole("combobox", { name: "Conta (conexão)" })).toHaveValue("");
  });

  it("edita com a versão carregada e mostra o motivo de uma recusa", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      "GET /agents": () => json({ items: [agent] }),
      "GET /connections": () => json(connections),
      "GET /models": () => json(catalogModels),
      [`PUT /agents/${AGENT}`]: () =>
        err(409, "connection_incompatible", "conexão incompatível: a conexão está revogada"),
    });
    renderAt("/agents");
    await userEvent.click(await screen.findByRole("button", { name: "Editar dev" }));
    const model = screen.getByRole("combobox", { name: "Modelo" });
    expect(model).toHaveValue("claude-sonnet-5-5"); // o modelo salvo vem selecionado
    await userEvent.selectOptions(model, "claude-opus-5-5");
    await userEvent.click(screen.getByRole("button", { name: "Salvar alterações" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("a conexão está revogada");
    const put = calls.find((c) => c.method === "PUT");
    expect(put?.body).toMatchObject({ expected_version: 3, model: "claude-opus-5-5", name: "dev" });
  });
});

describe("configuração efetiva no editor de workflows", () => {
  const catalog: NodeType[] = [
    {
      type: "start",
      label: "Início",
      category: "fluxo",
      inputs: null,
      outputs: ["out"],
      multi_in: false,
      fields: null,
    },
    {
      type: "agent",
      label: "Agente (IA)",
      category: "execução",
      inputs: ["in"],
      outputs: ["out", "error"],
      multi_in: false,
      fields: [{ name: "agent", label: "Agente", kind: "text", required: true }],
    },
  ];
  const wf: Workflow = {
    id: WF,
    name: "Fluxo",
    kind: "feature",
    description: "",
    version: 1,
    issues: [],
    created_at: "2026-11-12T10:00:00Z",
    updated_at: "2026-11-12T10:00:00Z",
    graph: {
      nodes: [
        { id: "inicio", type: "start", x: 40, y: 40, props: {} },
        {
          id: "ag",
          type: "agent",
          x: 240,
          y: 40,
          props: { agent: "dev", model: "claude-haiku-5-5" },
        },
      ],
      edges: [],
    },
  };
  const effective: EffectiveAgent = {
    agent_id: AGENT,
    agent_name: "dev",
    agent_version: 3,
    provider: "claude",
    connection_id: CLAUDE_A,
    model: "claude-haiku-5-5",
    instructions: "Seja objetivo.",
    prompt: "",
    skills: [],
    tools: ["Read"],
    fallback: [],
    timeout_minutes: 30,
    sources: {
      provider: "agent",
      connection: "agent",
      model: "node",
      tools: "agent",
      timeout_minutes: "agent",
    },
  };

  it("mostra a configuração efetiva com a origem de cada valor", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /workflows/${WF}`]: () => json(wf),
      "GET /workflows/node-types": () => json({ items: catalog }),
      "POST /workflows/validate": () => json({ issues: [] }),
      [`GET /workflows/${WF}/nodes/ag/effective`]: () => json(effective),
    });
    renderAt(`/workflows/${WF}`);
    await screen.findByLabelText("Grafo do workflow");
    fireEvent.click(screen.getByLabelText(/^Nó ag/));
    await userEvent.click(screen.getByRole("button", { name: "Ver configuração efetiva" }));
    const view = await screen
      .findByRole("group", { name: "Configuração efetiva" })
      .catch(() => screen.findByLabelText("Configuração efetiva"));
    expect(within(view).getByText(/dev \(versão 3\)/)).toBeInTheDocument();
    expect(within(view).getByText(/claude-haiku-5-5/)).toBeInTheDocument();
    expect(within(view).getByText("(do nó)")).toBeInTheDocument();
    expect(within(view).getAllByText("(do agente)").length).toBeGreaterThanOrEqual(3);
  });

  it("mostra o motivo quando a sobrescrita é inválida", async () => {
    stub({
      "GET /session": () => json(session),
      [`GET /workflows/${WF}`]: () => json(wf),
      "GET /workflows/node-types": () => json({ items: catalog }),
      [`GET /workflows/${WF}/nodes/ag/effective`]: () =>
        err(
          422,
          "invalid_override",
          "sobrescrita inválida: o nó não pode ampliar as ferramentas do agente",
        ),
    });
    renderAt(`/workflows/${WF}`);
    await screen.findByLabelText("Grafo do workflow");
    fireEvent.click(screen.getByLabelText(/^Nó ag/));
    await userEvent.click(screen.getByRole("button", { name: "Ver configuração efetiva" }));
    expect(await screen.findByText(/não pode ampliar as ferramentas/)).toBeInTheDocument();
  });
});
