import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { Delivery, DeliveryDetail, MemoryRecord, ProjectOverview } from "../lib/api";
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
const PID = "11111111-1111-4111-8111-111111111111";
const DID = "33333333-3333-4333-8333-333333333333";
const FID = "44444444-4444-4444-8444-444444444444";

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });
const apiError = (status: number, code: string, message: string) =>
  json({ error: { code, message, action: "Tente de novo.", correlation_id: "c-1" } }, status);

const project = {
  id: PID,
  slug: "demo",
  name: "Demo",
  description: "",
  state: "active",
  priority: 2,
  wizard_step: "review",
  repository: { owner: "org", name: "app", base_branch: "main", connection_id: null },
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

const overview = (mode: "active" | "draining" | "suspended"): ProjectOverview => ({
  project: project as unknown as ProjectOverview["project"],
  controls: {
    global: { scope: "global", mode: "active", reason: "", version: 1 },
    project: { scope: "project", project_id: PID, mode, reason: "", version: 1 },
  },
  counts: {},
  items: [],
  executions: [],
  last_sync: null,
});

function delivery(over: Partial<Delivery> = {}): Delivery {
  return {
    id: DID,
    feature_id: FID,
    feature_key: "PED-7",
    feature_title: "Pedidos com soft delete",
    attempt: 1,
    branch: "lf/ped-7-a1",
    revision: "b".repeat(40),
    state: "draft_open",
    reasons: [],
    pr_url: "https://github.com/org/app/pull/12",
    merge_sha: "",
    projection: "none",
    projection_error: "",
    observed_at: null,
    created_at: "2026-11-12T10:00:00Z",
    comments: {},
    ...over,
  };
}

function record(over: Partial<MemoryRecord> = {}): MemoryRecord {
  return {
    id: "55555555-5555-4555-8555-555555555555",
    branch: "lf/ped-7-a1",
    key: "pedidos usam soft delete",
    kind: "fact",
    content: "a tabela de pedidos usa a coluna removido_em",
    source: "revisão da tarefa schema",
    evidence: [{ label: "check", ref: "make test" }],
    revision: "b".repeat(40),
    confidence: "high",
    state: "pending",
    ...over,
  };
}

interface Call {
  method: string;
  url: string;
  body?: Record<string, unknown>;
  csrf?: string;
}

function stub(extra: Record<string, (call: Call) => Response> = {}) {
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
      const routes: Record<string, (c: Call) => Response> = {
        "GET /session": () => json(session),
        [`GET /projects/${PID}`]: () => json(project),
        [`GET /projects/${PID}/validation`]: () => json({ ok: true, checks: [], not_verified: [] }),
        [`GET /projects/${PID}/changes`]: () => json({ items: [] }),
        "GET /connections": () => json({ items: [] }),
        [`GET /projects/${PID}/overview`]: () => json(overview("active")),
        [`GET /projects/${PID}/deliveries`]: () => json({ items: [] }),
        [`GET /projects/${PID}/delivery-settings`]: () =>
          json({ review_authors: [], status_map: {}, version: 1 }),
        [`GET /projects/${PID}/memory?state=&q=`]: () => json({ items: [] }),
        ...extra,
      };
      const h = routes[`${method} ${url}`];
      return h ? h(call) : apiError(404, "not_found", "Rota inexistente.");
    }),
  );
  return calls;
}

function renderProject() {
  return render(
    <MemoryRouter initialEntries={[`/projects/${PID}`]}>
      <SessionProvider>
        <App />
      </SessionProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("Entregas do projeto", () => {
  it("mostra a PR, o estado e consulta o merge sob demanda, explicando o resultado", async () => {
    const calls = stub({
      [`GET /projects/${PID}/deliveries`]: () =>
        json({ items: [delivery({ comments: { queued: 1 } })] }),
      [`POST /projects/${PID}/deliveries/${DID}/observe`]: () =>
        json({
          observation: {
            outcome: "merged",
            merge_sha: "c".repeat(40),
            projection: "done",
            released: [{}, {}],
          },
          origin_note: "mapa de estados da origem não configurado: o item não foi movido",
          memory: { promoted: 2, rejected: 1, duplicates: 0 },
        }),
    });
    renderProject();
    const item = await screen.findByRole("listitem", { name: "PED-7 tentativa 1" });
    expect(within(item).getByText("Draft PR aberta")).toBeInTheDocument();
    expect(within(item).getByRole("link", { name: /pull\/12/ })).toHaveAttribute(
      "href",
      "https://github.com/org/app/pull/12",
    );
    expect(within(item).getByText(/Comentários: 1 na fila/)).toBeInTheDocument();
    expect(within(item).getByText("PR ainda não consultada")).toBeInTheDocument();
    expect(calls.some((c) => c.url.endsWith("/observe"))).toBe(false); // nada consulta sozinho
    await userEvent.click(within(item).getByRole("button", { name: "Consultar PR" }));
    expect(await screen.findByText("Merge confirmado.")).toBeInTheDocument();
    // A lista é recarregada depois da consulta, mas a mensagem e o estado local não somem.
    const listLoads = () =>
      calls.filter((c) => c.method === "GET" && c.url === `/projects/${PID}/deliveries`).length;
    await waitFor(() => expect(listLoads()).toBe(2));
    expect(screen.getByText("Merge confirmado.")).toBeInTheDocument();
    expect(screen.getByText("Itens devolvidos à fila: 2.")).toBeInTheDocument();
    expect(screen.getByText(/Memória: 2 promovida\(s\), 1 invalidada\(s\)/)).toBeInTheDocument();
    expect(screen.getByText(/item não foi movido/)).toBeInTheDocument();
    const post = calls.find((c) => c.url.endsWith("/observe"));
    expect(post?.method).toBe("POST");
    expect(post?.csrf).toBe("csrf-1");
  });

  it("oferece as três ações para a PR fechada e registra a escolhida com a observação", async () => {
    const detail: DeliveryDetail = {
      delivery: delivery({ state: "closed" }),
      comments: [
        { id: 1, comment_id: 9, author: "revisora", body: "renomeie o campo", state: "held" },
      ],
      offer: {
        max_attempts: 3,
        options: [
          { id: "cancel", label: "Cancelar", description: "encerra a entrega" },
          { id: "reopen", label: "Reabrir", description: "reabre a mesma PR" },
          { id: "retry", label: "Nova tentativa", description: "outra branch" },
        ],
      },
      history: ["offered"],
      observations: [
        {
          outcome: "closed",
          detail: "a PR foi fechada sem merge",
          created_at: "2026-11-12T11:00:00Z",
        },
      ],
    };
    const calls = stub({
      [`GET /projects/${PID}/deliveries`]: () => json({ items: [delivery({ state: "closed" })] }),
      [`GET /projects/${PID}/deliveries/${DID}`]: () => json(detail),
      [`POST /projects/${PID}/deliveries/${DID}/actions`]: () =>
        json({ state: "draft_open", replayed: false }),
    });
    renderProject();
    const item = await screen.findByRole("listitem", { name: "PED-7 tentativa 1" });
    expect(within(item).getByText("PR fechada sem merge")).toBeInTheDocument();
    await userEvent.click(within(item).getByRole("button", { name: "Detalhes" }));
    expect(
      await within(item).findByText("A PR foi fechada sem merge. O que fazer?"),
    ).toBeInTheDocument();
    expect(within(item).getByText("renomeie o campo")).toBeInTheDocument();
    for (const name of ["Cancelar", "Reabrir", "Nova tentativa"]) {
      expect(within(item).getByRole("button", { name })).toBeInTheDocument();
    }
    await userEvent.type(within(item).getByLabelText("Observação (opcional)"), "voltou a valer");
    await userEvent.click(within(item).getByRole("button", { name: "Reabrir" }));
    expect(await screen.findByText("Reabrir: registrada.")).toBeInTheDocument();
    const post = calls.find((c) => c.url.endsWith("/actions"));
    expect(post?.body).toEqual({ action: "reopen", note: "voltou a valer" });
    expect(post?.csrf).toBe("csrf-1");
  });

  it("busca os comentários da revisão e explica quando nenhum autor está autorizado", async () => {
    const calls = stub({
      [`GET /projects/${PID}/deliveries`]: () => json({ items: [delivery()] }),
      [`POST /projects/${PID}/deliveries/${DID}/comments/discover`]: () =>
        json({ discovery: { new: 0, ignored: 2, held: 0, item: null }, authorized: 0 }),
    });
    renderProject();
    const item = await screen.findByRole("listitem", { name: "PED-7 tentativa 1" });
    const button = within(item).getByRole("button", { name: "Buscar comentários da revisão" });
    expect(button).toBeDisabled();
    await userEvent.type(within(item).getByLabelText("Tarefa que recebe a correção"), "schema");
    await userEvent.click(button);
    expect(await screen.findByText(/Comentários novos: 0; ignorados/)).toBeInTheDocument();
    expect(screen.getByText(/Nenhum autor está autorizado/)).toBeInTheDocument();
    expect(calls.find((c) => c.url.endsWith("/comments/discover"))?.body).toEqual({
      task_key: "schema",
    });
  });

  it("mostra o erro do servidor em linguagem do painel quando o GitHub falha", async () => {
    stub({
      [`GET /projects/${PID}/deliveries`]: () => json({ items: [delivery()] }),
      [`POST /projects/${PID}/deliveries/${DID}/observe`]: () =>
        apiError(502, "github_unavailable", "Não foi possível falar com o GitHub."),
    });
    renderProject();
    const item = await screen.findByRole("listitem", { name: "PED-7 tentativa 1" });
    await userEvent.click(within(item).getByRole("button", { name: "Consultar PR" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Não foi possível falar com o GitHub.",
    );
  });

  it("carrega o histórico vivo da feature e salva a configuração da entrega", async () => {
    const calls = stub({
      [`GET /projects/${PID}/deliveries`]: () => json({ items: [delivery()] }),
      [`GET /projects/${PID}/features/${FID}/report`]: () =>
        json({
          feature_key: "PED-7",
          title: "Pedidos",
          revision: "b",
          doc_path: "docs/features/ped-7.md",
          markdown: "# PED-7 — Pedidos",
        }),
      [`PUT /projects/${PID}/delivery-settings`]: () =>
        json({ review_authors: ["revisora"], status_map: { done: "Concluída" }, version: 2 }),
    });
    renderProject();
    const item = await screen.findByRole("listitem", { name: "PED-7 tentativa 1" });
    await userEvent.click(within(item).getByRole("button", { name: "Histórico da feature" }));
    expect(await screen.findByRole("log", { name: "Histórico da feature" })).toHaveTextContent(
      "# PED-7 — Pedidos",
    );
    const form = screen.getByRole("region", { name: "Configuração da entrega" });
    await userEvent.type(
      await within(form).findByLabelText(/Autores autorizados/),
      "revisora, outra-pessoa",
    );
    await userEvent.type(within(form).getByLabelText(/Concluída/), "Concluída");
    await userEvent.click(within(form).getByRole("button", { name: "Salvar configuração" }));
    expect(await within(form).findByText("Configuração salva.")).toBeInTheDocument();
    expect(calls.find((c) => c.method === "PUT")?.body).toEqual({
      review_authors: ["revisora", "outra-pessoa"],
      status_map: { awaiting_merge: "", done: "Concluída" },
    });
  });
});

describe("Memória técnica", () => {
  it("lista pendente e oficial com evidência, filtra por estado e busca por termos", async () => {
    const official = record({
      id: "66666666-6666-4666-8666-666666666666",
      key: "porta da api",
      content: "a API usa a porta 8480",
      state: "official",
      promoted_revision: "c".repeat(40),
    });
    const calls = stub({
      [`GET /projects/${PID}/memory?state=&q=`]: () => json({ items: [record(), official] }),
      [`GET /projects/${PID}/memory?state=official&q=`]: () => json({ items: [official] }),
      [`GET /projects/${PID}/memory?state=official&q=porta`]: () => json({ items: [official] }),
    });
    renderProject();
    const panel = await screen.findByRole("region", { name: "Memória técnica" });
    expect(
      await within(panel).findByText("a tabela de pedidos usa a coluna removido_em"),
    ).toBeInTheDocument();
    expect(within(panel).getAllByText(/Evidências: check \(make test\)/)).toHaveLength(2);
    expect(within(panel).getByText("Branch: lf/ped-7-a1")).toBeInTheDocument();
    expect(within(panel).getByText(/Promovida no merge cccccccccccc/)).toBeInTheDocument();
    await userEvent.click(within(panel).getByRole("tab", { name: "Oficiais" }));
    await within(panel).findByText("porta da api");
    expect(within(panel).queryByText("pedidos usam soft delete")).not.toBeInTheDocument();
    await userEvent.type(within(panel).getByLabelText("Buscar na memória"), "porta");
    await userEvent.click(within(panel).getByRole("button", { name: "Buscar" }));
    await within(panel).findByText("porta da api");
    expect(calls.some((c) => c.url === `/projects/${PID}/memory?state=official&q=porta`)).toBe(
      true,
    );
  });

  it("invalida um registro só com motivo e não oferece invalidar o que já está invalidado", async () => {
    let invalidated = false;
    const calls = stub({
      [`GET /projects/${PID}/memory?state=&q=`]: () =>
        json({
          items: [
            invalidated
              ? record({ state: "invalidated", invalid_reason: "regra mudou" })
              : record(),
          ],
        }),
      [`POST /projects/${PID}/memory/${record().id}/invalidate`]: () => {
        invalidated = true;
        return json(record({ state: "invalidated", invalid_reason: "regra mudou" }));
      },
    });
    renderProject();
    const panel = await screen.findByRole("region", { name: "Memória técnica" });
    await userEvent.click(await within(panel).findByRole("button", { name: "Invalidar" }));
    const confirm = within(panel).getByRole("button", { name: "Confirmar invalidação" });
    expect(confirm).toBeDisabled();
    await userEvent.type(within(panel).getByLabelText(/Motivo da invalidação/), "regra mudou");
    await userEvent.click(confirm);
    expect(await within(panel).findByText("Motivo: regra mudou")).toBeInTheDocument();
    expect(within(panel).queryByRole("button", { name: "Invalidar" })).not.toBeInTheDocument();
    expect(calls.find((c) => c.url.endsWith("/invalidate"))?.body).toEqual({
      reason: "regra mudou",
    });
  });
});

describe("Controles do projeto", () => {
  it("pausa após a feature atual e retoma, com o motivo", async () => {
    let mode: "active" | "draining" = "active";
    const calls = stub({
      [`GET /projects/${PID}/overview`]: () => json(overview(mode)),
      [`PUT /projects/${PID}/controls`]: (c) => {
        mode = c.body?.mode === "draining" ? "draining" : "active";
        return json({ control: { mode } });
      },
    });
    renderProject();
    const controls = await screen.findByRole("region", { name: "Controles do projeto" });
    await userEvent.type(within(controls).getByLabelText("Motivo (opcional)"), "revisão de custos");
    await userEvent.click(
      within(controls).getByRole("button", { name: "Pausar após a feature atual" }),
    );
    expect(
      await within(controls).findByRole("button", { name: "Retomar projeto" }),
    ).toBeInTheDocument();
    expect(calls.find((c) => c.method === "PUT")?.body).toEqual({
      mode: "draining",
      reason: "revisão de custos",
    });
    await userEvent.click(within(controls).getByRole("button", { name: "Retomar projeto" }));
    expect(
      await within(controls).findByRole("button", { name: "Pausar após a feature atual" }),
    ).toBeInTheDocument();
  });

  it("avisa quando uma tarefa não pôde ser parada com checkpoint seguro", async () => {
    stub({
      [`PUT /projects/${PID}/controls`]: () =>
        json({
          control: { mode: "suspended" },
          failed: [{ reason: "o executor de sandbox não está ligado" }],
        }),
    });
    renderProject();
    const controls = await screen.findByRole("region", { name: "Controles do projeto" });
    await userEvent.click(within(controls).getByRole("button", { name: "Suspender agora" }));
    expect(await within(controls).findByRole("status")).toHaveTextContent("seguem em execução");
  });
});
