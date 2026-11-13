import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { Skill, SkillImport } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { formatOf, parseSteps } from "./SkillsPage";

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
  json({ error: { code, message, action: "Corrija.", correlation_id: "c-1" } }, status);

interface Call {
  method: string;
  url: string;
  body?: unknown;
  headers?: Record<string, string>;
}

function stub(routes: Record<string, (call: Call) => Response>) {
  const calls: Call[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input).replace(/^\/api/, "");
      const method = (init.method ?? "GET").toUpperCase();
      const raw = init.body;
      calls.push({
        method,
        url,
        body: typeof raw === "string" ? JSON.parse(raw) : raw,
        headers: init.headers as Record<string, string>,
      });
      const handler = routes[`${method} ${url.split("?")[0]}`];
      return handler
        ? handler(calls[calls.length - 1] as Call)
        : err(404, "not_found", "Rota inexistente.");
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

const skill: Skill = {
  id: "11111111-1111-4111-8111-111111111111",
  name: "revisao",
  description: "",
  executor: "ai",
  source_kind: "package",
  source_ref: "rev.tar.gz",
  revision: "",
  content_hash: "a".repeat(64),
  body: "x",
  manifest: { permissions: [], resources: [], commands: [], steps: [] },
  needs_approval: true,
  version: 2,
  created_at: "2026-11-12T10:00:00Z",
  updated_at: "2026-11-12T10:00:00Z",
};

const rejected: SkillImport = {
  id: 9,
  name: "",
  source_kind: "package",
  source_ref: "ruim.tar.gz",
  outcome: "rejected",
  reason:
    "O script contém um comando proibido: Ferramentas de rede não rodam em scripts. (scripts/pwn.sh)",
  content_hash: "",
  created_at: "2026-11-12T10:00:00Z",
};

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("auxiliares", () => {
  it("reconhece o formato pelo nome e interpreta passos", () => {
    expect(formatOf("a.tar.gz")).toBe("tar.gz");
    expect(formatOf("A.TGZ")).toBe("tar.gz");
    expect(formatOf("pacote.zip")).toBe("zip");
    expect(formatOf("SKILL.md")).toBe("md");
    expect(formatOf("script.sh")).toBeNull();
    expect(parseSteps("")).toEqual([]);
    expect(parseSteps('[{"type":"ai"}]')).toEqual([{ type: "ai" }]);
    expect(parseSteps("{")).toContain("JSON");
    expect(parseSteps('{"a":1}')).toContain("lista");
  });
});

describe("catálogo de skills", () => {
  const base = (extra: Record<string, (c: Call) => Response> = {}) => ({
    "GET /session": () => json(session),
    "GET /skills": () => json({ items: [skill] }),
    "GET /skills/imports": () => json({ items: [rejected] }),
    ...extra,
  });

  it("lista skills com origem, versão e aviso de aprovação, e mostra as recusas", async () => {
    stub(base());
    renderAt("/skills");
    expect(await screen.findByText("revisao")).toBeInTheDocument();
    expect(screen.getByText(/versão 2/)).toBeInTheDocument();
    expect(screen.getByText("Exige aprovação")).toBeInTheDocument();
    expect(await screen.findByText(/Recusada/)).toBeInTheDocument();
    expect(screen.getByText(/comando proibido/)).toBeInTheDocument();
  });

  it("cadastra uma skill de script enviando os comandos linha a linha", async () => {
    const calls = stub(base({ "POST /skills": () => json(skill, 201) }));
    renderAt("/skills");
    await userEvent.type(await screen.findByLabelText("Nome"), "testar");
    await userEvent.selectOptions(screen.getByLabelText("Executor"), "script");
    await userEvent.type(screen.getByLabelText(/^Comandos/), "make test{enter}make lint");
    await userEvent.click(screen.getByRole("button", { name: "Cadastrar skill" }));
    await screen.findByText("revisao");
    const post = calls.find((c) => c.method === "POST");
    expect(post?.body).toMatchObject({
      name: "testar",
      executor: "script",
      commands: ["make test", "make lint"],
    });
  });

  it("recusa passos que não são JSON sem chamar a API", async () => {
    const calls = stub(base());
    renderAt("/skills");
    await userEvent.type(await screen.findByLabelText("Nome"), "x");
    await userEvent.type(screen.getByLabelText(/^Passos/), "isso não é json");
    await userEvent.click(screen.getByRole("button", { name: "Cadastrar skill" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("JSON");
    expect(calls.some((c) => c.method === "POST")).toBe(false);
  });

  it("importa um pacote com o formato certo e mostra o motivo de uma recusa", async () => {
    let rejectNext = false;
    const calls = stub(
      base({
        "POST /skills/import": () =>
          rejectNext
            ? err(422, "import_rejected", "O script contém um comando proibido (scripts/pwn.sh)")
            : json(skill, 201),
      }),
    );
    renderAt("/skills");
    const input = (await screen.findByLabelText(/^Arquivo/)) as HTMLInputElement;
    await userEvent.upload(
      input,
      new File(["conteudo"], "revisao.tar.gz", { type: "application/gzip" }),
    );
    await userEvent.click(screen.getByRole("button", { name: "Importar" }));
    expect(await screen.findByText(/Skill revisao importada \(versão 2\)/)).toBeInTheDocument();
    const imp = calls.find((c) => c.url.startsWith("/skills/import?"));
    expect(imp?.url).toContain("format=tar.gz");
    expect(imp?.url).toContain("replace=false");
    expect(imp?.url).toContain("ref=revisao.tar.gz");
    expect(imp?.headers?.["Content-Type"]).toBe("application/octet-stream");

    rejectNext = true;
    await userEvent.upload(input, new File(["x"], "ruim.zip"));
    await userEvent.click(screen.getByLabelText(/Substituir se já existir/));
    await userEvent.click(screen.getByRole("button", { name: "Importar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("comando proibido");
    expect(calls.filter((c) => c.url.startsWith("/skills/import?")).at(-1)?.url).toContain(
      "replace=true",
    );
  });

  it("recusa arquivo de formato desconhecido sem chamar a API", async () => {
    const calls = stub(base());
    renderAt("/skills");
    const input = (await screen.findByLabelText(/^Arquivo/)) as HTMLInputElement;
    await userEvent.upload(input, new File(["x"], "script.sh"), { applyAccept: false });
    await userEvent.click(screen.getByRole("button", { name: "Importar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("tar.gz");
    expect(calls.some((c) => c.url.startsWith("/skills/import?"))).toBe(false);
  });
});
