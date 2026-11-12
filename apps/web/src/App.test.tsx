import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "./App";
import type { Project } from "./lib/api";
import { SessionProvider } from "./lib/session";

const session = {
  id: "s1",
  admin_id: "a1",
  username: "admin",
  expires_at: "2030-01-01T00:00:00Z",
  csrf_token: "csrf-1",
  totp_reset_required: false,
};

const project: Project = {
  id: "11111111-1111-4111-8111-111111111111",
  slug: "exemplo-api",
  name: "Exemplo — API",
  description: "",
  state: "active",
  priority: 3,
  wizard_step: "review",
  repository: { owner: "org", name: "api", base_branch: "main", connection_id: null },
  source: { kind: "", connection_id: null, container: "" },
  version: 2,
  activated_at: "2026-11-02T10:00:00Z",
  created_at: "2026-11-01T10:00:00Z",
  updated_at: "2026-11-02T10:00:00Z",
};

type Handler = () => Response | Promise<Response>;

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status });
}
function apiError(status: number, code: string, message: string): Response {
  return json(
    { error: { code, message, action: "Tente de novo.", correlation_id: "corr-1234" } },
    status,
  );
}

function stubApi(routes: Record<string, Handler>) {
  const fn = vi.fn(async (input: RequestInfo | URL) => {
    const url = String(input).replace(/^\/api/, "");
    const key = Object.keys(routes).find((k) => url === k || url.startsWith(`${k}?`));
    if (!key) return apiError(404, "not_found", "Rota inexistente.");
    return (routes[key] as Handler)();
  });
  vi.stubGlobal("fetch", fn);
  return fn;
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

const ready: Handler = () => json({ status: "ready" });

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("sem sessão", () => {
  it("mostra o login e não renderiza o menu nem as rotas do painel", async () => {
    stubApi({ "/session": () => apiError(401, "unauthorized", "Sessão ausente.") });
    renderAt("/projects");
    expect(await screen.findByRole("heading", { name: "Entrar" })).toBeInTheDocument();
    expect(screen.queryByRole("navigation", { name: /menu/i })).not.toBeInTheDocument();
    expect(screen.queryByText("Projetos")).not.toBeInTheDocument();
    // O acesso dev sem senha foi removido (T04-01).
    expect(screen.queryByRole("button", { name: /dev/i })).not.toBeInTheDocument();
  });

  it("falha do servidor na verificação mostra erro com nova tentativa", async () => {
    stubApi({ "/session": () => apiError(500, "internal_error", "Erro interno.") });
    renderAt("/");
    expect(await screen.findByRole("alert")).toHaveTextContent("Erro interno.");
    expect(screen.getByText(/corr-1234/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tentar novamente" })).toBeInTheDocument();
  });
});

describe("com sessão", () => {
  const base = { "/session": () => json(session), "/readyz": ready };

  it("o menu tem apenas as áreas entregues e os links corretos", async () => {
    stubApi(base);
    renderAt("/");
    const nav = await screen.findByRole("navigation");
    const links = within(nav).getAllByRole("link");
    expect(links.map((l) => [l.getAttribute("aria-label"), l.getAttribute("href")])).toEqual([
      ["Painel", "/"],
      ["Projetos", "/projects"],
      ["Biblioteca", "/library"],
      ["Decisões", "/decisions"],
      ["Conexões", "/connections"],
      ["Segurança", "/security"],
    ]);
    expect(screen.getByRole("link", { name: "Painel" })).toHaveAttribute("aria-current", "page");
  });

  it("o painel inicial é vazio (sem dados de demonstração)", async () => {
    stubApi(base);
    renderAt("/");
    expect(await screen.findByText("Nada para mostrar ainda")).toBeInTheDocument();
    expect(screen.queryByText(/demonstra/i)).not.toBeInTheDocument();
  });

  it("navega entre as áreas pelos links do menu", async () => {
    stubApi({ ...base, "/projects": () => json({ items: [] }) });
    renderAt("/");
    await userEvent.click(await screen.findByRole("link", { name: "Biblioteca" }));
    expect(await screen.findByRole("heading", { name: "Biblioteca global" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: "Decisões" }));
    expect(await screen.findByRole("heading", { name: "Central de Decisões" })).toBeInTheDocument();
    await userEvent.click(screen.getByRole("link", { name: "Projetos" }));
    expect(await screen.findByRole("heading", { name: "Projetos", level: 1 })).toBeInTheDocument();
  });

  it("lista de projetos: carregando → itens com link para o detalhe", async () => {
    stubApi({ ...base, "/projects": () => json({ items: [project] }) });
    renderAt("/projects");
    expect(screen.getByText("Verificando sessão…")).toBeInTheDocument();
    const link = await screen.findByRole("link", { name: /Exemplo — API/ });
    expect(link).toHaveAttribute("href", `/projects/${project.id}`);
    expect(screen.getByText("Ativo")).toBeInTheDocument();
  });

  it("lista de projetos: estado vazio", async () => {
    stubApi({ ...base, "/projects": () => json({ items: [] }) });
    renderAt("/projects");
    expect(await screen.findByText("Nenhum projeto cadastrado")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Novo projeto" })).toHaveAttribute(
      "href",
      "/projects/new",
    );
  });

  it("lista de projetos: erro com nova tentativa que recupera", async () => {
    let calls = 0;
    stubApi({
      ...base,
      "/projects": () =>
        ++calls === 1
          ? apiError(500, "internal_error", "Erro interno.")
          : json({ items: [project] }),
    });
    renderAt("/projects");
    expect(await screen.findByRole("alert")).toHaveTextContent("Erro interno.");
    await userEvent.click(screen.getByRole("button", { name: "Tentar novamente" }));
    expect(await screen.findByRole("link", { name: /Exemplo — API/ })).toBeInTheDocument();
  });

  it("um 401 numa página devolve o usuário à tela de sessão", async () => {
    stubApi({ ...base, "/projects": () => apiError(401, "unauthorized", "Sessão revogada.") });
    renderAt("/projects");
    expect(await screen.findByRole("heading", { name: "Entrar" })).toBeInTheDocument();
  });

  it("detalhe do projeto mostra os dados", async () => {
    stubApi({
      ...base,
      [`/projects/${project.id}`]: () => json(project),
      "/connections": () => json({ items: [] }),
      [`/projects/${project.id}/changes`]: () => json({ items: [] }),
    });
    renderAt(`/projects/${project.id}`);
    expect(
      await screen.findByRole("heading", { name: "Exemplo — API", level: 1 }),
    ).toBeInTheDocument();
    expect(screen.getByText("3 de 4")).toBeInTheDocument();
    expect(
      within(screen.getByRole("navigation", { name: "Trilha" })).getByRole("link", {
        name: "Projetos",
      }),
    ).toHaveAttribute("href", "/projects");
  });

  it("detalhe de projeto inexistente mostra 'não encontrado'", async () => {
    stubApi({
      ...base,
      "/projects/nope": () => apiError(404, "not_found", "Recurso não encontrado."),
    });
    renderAt("/projects/nope");
    expect(await screen.findByText("Projeto não encontrado")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Voltar para projetos" })).toHaveAttribute(
      "href",
      "/projects",
    );
  });

  it("rota desconhecida mostra página não encontrada", async () => {
    stubApi(base);
    renderAt("/qualquer-coisa");
    expect(await screen.findByText("Página não encontrada")).toBeInTheDocument();
  });

  it("o indicador do topo reflete a prontidão da API", async () => {
    stubApi({
      ...base,
      "/readyz": () => apiError(503, "database_unavailable", "Banco indisponível."),
    });
    renderAt("/");
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Indisponível"));
  });
});
