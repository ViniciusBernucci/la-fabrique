import { fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { NodeType, Workflow, WorkflowGraph, WorkflowIssue } from "../lib/api";
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
  json({ error: { code, message, action: "Tente.", correlation_id: "c-1" } }, status);

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

const ID = "55555555-5555-4555-8555-555555555555";

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
    type: "end",
    label: "Fim",
    category: "fluxo",
    inputs: ["in"],
    outputs: [],
    multi_in: true,
    fields: [
      {
        name: "outcome",
        label: "Resultado",
        kind: "enum",
        required: true,
        options: ["success", "failure"],
      },
    ],
  },
  {
    type: "agent",
    label: "Agente (IA)",
    category: "execução",
    inputs: ["in"],
    outputs: ["out", "error"],
    multi_in: false,
    fields: [
      { name: "agent", label: "Agente", kind: "text", required: true },
      { name: "prompt", label: "Instrução", kind: "textarea", required: true },
      {
        name: "timeout_minutes",
        label: "Tempo máximo (min)",
        kind: "int",
        required: false,
        min: 1,
        max: 720,
      },
    ],
  },
];

function workflow(over: Partial<Workflow> = {}): Workflow {
  return {
    id: ID,
    name: "Feature padrão",
    kind: "feature",
    description: "",
    version: 1,
    issues: [],
    created_at: "2026-11-12T10:00:00Z",
    updated_at: "2026-11-12T10:00:00Z",
    graph: {
      nodes: [
        { id: "inicio", type: "start", x: 80, y: 140, props: {} },
        { id: "fim", type: "end", x: 420, y: 140, props: { outcome: "success" } },
      ],
      edges: [],
    },
    ...over,
  };
}

const requiredPrompt = (g: WorkflowGraph): WorkflowIssue[] =>
  g.nodes
    .filter((n) => n.type === "agent" && !n.props.prompt)
    .map((n) => ({
      node: n.id,
      field: "prompt",
      code: "required",
      message: 'O campo "Instrução" é obrigatório.',
    }));

function editorRoutes(wf: Workflow, extra: Record<string, (c: Call) => Response> = {}) {
  return {
    "GET /session": () => json(session),
    [`GET /workflows/${ID}`]: () => json(wf),
    "GET /workflows/node-types": () => json({ items: catalog }),
    "POST /workflows/validate": (c: Call) =>
      json({ issues: requiredPrompt(c.body?.graph as WorkflowGraph) }),
    ...extra,
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

const click = (name: string | RegExp) => userEvent.click(screen.getByRole("button", { name }));

describe("lista de workflows", () => {
  it("mostra pendências e cria um rascunho abrindo o editor", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      "GET /workflows": () =>
        json({
          items: [
            {
              id: ID,
              name: "Feature padrão",
              kind: "feature",
              description: "",
              version: 3,
              nodes: 5,
              issues: 2,
              updated_at: "2026-11-12T10:00:00Z",
            },
          ],
        }),
      "POST /workflows": () => json(workflow({ name: "Novo" }), 201),
      [`GET /workflows/${ID}`]: () => json(workflow({ name: "Novo" })),
      "GET /workflows/node-types": () => json({ items: catalog }),
    });
    renderAt("/workflows");
    expect(await screen.findByText("2 pendência(s)")).toBeInTheDocument();
    expect(screen.getByText(/5 nós · versão 3/)).toBeInTheDocument();

    await userEvent.type(screen.getByLabelText("Nome"), "Novo");
    await click("Criar rascunho");
    expect(await screen.findByRole("group", { name: "Grafo do workflow" })).toBeInTheDocument();
    const create = calls.find((c) => c.method === "POST" && c.url === "/workflows");
    expect(create?.body).toEqual({ name: "Novo", kind: "feature" });
    expect(create?.csrf).toBe("csrf-1");
  });
});

describe("editor de workflows", () => {
  it("monta um fluxo, edita propriedades e salva nós, conexões e propriedades", async () => {
    const calls = stub(
      editorRoutes(workflow(), {
        [`PUT /workflows/${ID}`]: (c) =>
          json(
            workflow({
              version: 2,
              graph: c.body?.graph as WorkflowGraph,
              name: String(c.body?.name),
            }),
          ),
      }),
    );
    renderAt(`/workflows/${ID}`);
    await screen.findByRole("group", { name: "Grafo do workflow" });
    expect(screen.getByRole("button", { name: /^Nó inicio/ })).toBeInTheDocument();
    expect(screen.getByText("Nenhum problema encontrado.")).toBeInTheDocument();

    // Adiciona um agente e conecta início → agente → fim.
    await click("Agente (IA)");
    await click("Saída out de inicio");
    await click("Entrada in de agent-1");
    await click("Saída out de agent-1");
    await click("Entrada in de fim");
    expect(
      screen.getByRole("button", { name: "Conexão de agent-1.out para fim.in" }),
    ).toBeInTheDocument();

    // Propriedades do agente (o nó recém-criado já vem selecionado).
    await userEvent.type(screen.getByLabelText(/^Agente \*/), "dev");
    await userEvent.type(screen.getByLabelText(/^Instrução/), "Implemente a feature");
    await userEvent.type(screen.getByLabelText(/^Tempo máximo/), "30");

    await click("Salvar rascunho");
    expect(await screen.findByText("Salvo")).toBeInTheDocument();
    const put = calls.find((c) => c.method === "PUT");
    expect(put?.csrf).toBe("csrf-1");
    expect(put?.body?.expected_version).toBe(1);
    const graph = put?.body?.graph as WorkflowGraph;
    expect(graph.nodes.map((n) => n.id)).toEqual(["inicio", "fim", "agent-1"]);
    expect(graph.nodes[2].props).toEqual({
      agent: "dev",
      prompt: "Implemente a feature",
      timeout_minutes: 30,
    });
    expect(graph.edges.map((e) => `${e.from}.${e.from_port}>${e.to}.${e.to_port}`)).toEqual([
      "inicio.out>agent-1.in",
      "agent-1.out>fim.in",
    ]);
  });

  it("mostra o erro com o nó e o campo relevantes e permite localizar o nó", async () => {
    stub(editorRoutes(workflow()));
    renderAt(`/workflows/${ID}`);
    await screen.findByRole("group", { name: "Grafo do workflow" });
    await click("Agente (IA)"); // nasce sem instrução
    const panel = await screen.findByRole("region", { name: "Validação" });
    expect(await within(panel).findByText("Nó agent-1 · prompt")).toBeInTheDocument();
    expect(within(panel).getByText('O campo "Instrução" é obrigatório.')).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Nó agent-1 (Agente (IA)), com pendências" }),
    ).toBeInTheDocument();

    // O erro aparece junto do campo no painel de propriedades.
    expect(await screen.findAllByText('O campo "Instrução" é obrigatório.')).toHaveLength(2);

    // Corrigir o campo limpa o problema.
    await userEvent.type(screen.getByLabelText(/^Instrução/), "ok");
    expect(await screen.findByText("Nenhum problema encontrado.")).toBeInTheDocument();
  });

  it("arrasta um nó e remove o nó com suas conexões", async () => {
    const wf = workflow({
      graph: {
        nodes: [
          { id: "inicio", type: "start", x: 80, y: 140, props: {} },
          { id: "fim", type: "end", x: 420, y: 140, props: { outcome: "success" } },
        ],
        edges: [{ id: "e1", from: "inicio", from_port: "out", to: "fim", to_port: "in" }],
      },
    });
    const calls = stub(
      editorRoutes(wf, {
        [`PUT /workflows/${ID}`]: (c) =>
          json(workflow({ version: 2, graph: c.body?.graph as WorkflowGraph })),
      }),
    );
    renderAt(`/workflows/${ID}`);
    const svg = await screen.findByRole("group", { name: "Grafo do workflow" });

    const head = screen.getByRole("button", { name: /^Nó inicio/ });
    fireEvent.pointerDown(head, { clientX: 100, clientY: 150 });
    fireEvent.pointerMove(svg, { clientX: 163, clientY: 211 });
    fireEvent.pointerUp(svg);
    await click("Salvar rascunho");
    await screen.findByText("Salvo");
    const saved = calls.find((c) => c.method === "PUT")?.body?.graph as WorkflowGraph;
    const moved = saved.nodes[0];
    expect(moved).toMatchObject({ id: "inicio", x: 140, y: 200 }); // 80+63 e 140+61, na grade de 10

    // Remover o nó "fim" remove a conexão que chegava nele.
    await userEvent.click(screen.getByRole("button", { name: /^Nó fim/ }));
    await click("Remover seleção");
    expect(screen.queryByRole("button", { name: /^Nó fim/ })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /^Conexão de inicio/ })).not.toBeInTheDocument();
  });

  it("mostra o conflito de versão ao salvar", async () => {
    stub(
      editorRoutes(workflow(), {
        [`PUT /workflows/${ID}`]: () =>
          err(409, "version_conflict", "O recurso foi alterado por outra operação."),
      }),
    );
    renderAt(`/workflows/${ID}`);
    await screen.findByRole("group", { name: "Grafo do workflow" });
    await click("Fim");
    await click("Salvar rascunho");
    expect(await screen.findByRole("alert")).toHaveTextContent("alterado por outra operação");
    expect(screen.getByText("Alterações não salvas")).toBeInTheDocument();
  });
});
