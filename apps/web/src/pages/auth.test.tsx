import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { Connection } from "../lib/api";
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

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status });
}
function err(status: number, code: string, message: string): Response {
  return json({ error: { code, message, action: "Tente.", correlation_id: "corr-9999" } }, status);
}

interface Call {
  method: string;
  url: string;
  body?: Record<string, unknown>;
  csrf?: string;
}

/** Roteia por "MÉTODO /caminho" e registra chamadas. */
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

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

const unauth = () => err(401, "unauthorized", "Sessão ausente.");
const ready = () => json({ status: "ready" });

describe("login em duas etapas", () => {
  it("senha correta NÃO abre o painel: pede o código e só então entra", async () => {
    let loggedIn = false;
    const calls = stub({
      "GET /session": () => (loggedIn ? json(session) : unauth()),
      "POST /auth/login": () => json({ status: "totp_required" }),
      "POST /auth/totp": (c) => {
        if (c.body?.code !== "123456") return err(401, "invalid_code", "Código inválido.");
        loggedIn = true;
        return json(session);
      },
      "GET /readyz": ready,
    });
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Usuário"), "admin");
    await userEvent.type(screen.getByLabelText("Senha"), "uma-senha-longa-123");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));

    expect(
      await screen.findByRole("heading", { name: "Verificação em duas etapas" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    // A senha não fica no campo nem é reenviada.
    expect(calls.filter((c) => c.url === "/auth/login")).toHaveLength(1);

    await userEvent.type(screen.getByLabelText(/Código do autenticador/), "000000");
    await userEvent.click(screen.getByRole("button", { name: "Verificar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Código inválido.");
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();

    await userEvent.clear(screen.getByLabelText(/Código do autenticador/));
    await userEvent.type(screen.getByLabelText(/Código do autenticador/), "123456");
    await userEvent.click(screen.getByRole("button", { name: "Verificar" }));
    expect(await screen.findByRole("heading", { name: "Central de comando" })).toBeInTheDocument();
  });

  it("credenciais inválidas mostram o erro e mantêm o login", async () => {
    stub({
      "GET /session": unauth,
      "POST /auth/login": () =>
        err(401, "invalid_credentials", "Usuário, senha ou código inválidos."),
    });
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Usuário"), "admin");
    await userEvent.type(screen.getByLabelText("Senha"), "errada-errada-123");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Usuário, senha ou código inválidos.",
    );
    expect(screen.getByRole("heading", { name: "Entrar" })).toBeInTheDocument();
  });

  it("bloqueio por tentativas mostra a mensagem do servidor", async () => {
    stub({
      "GET /session": unauth,
      "POST /auth/login": () => err(429, "too_many_attempts", "Muitas tentativas."),
    });
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Usuário"), "admin");
    await userEvent.type(screen.getByLabelText("Senha"), "qualquer-senha-123");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Muitas tentativas.");
  });

  it("primeiro acesso: cadastra o 2FA, mostra os códigos uma vez e só então entra", async () => {
    let loggedIn = false;
    stub({
      "GET /session": () => (loggedIn ? json(session) : unauth()),
      "POST /auth/login": () => json({ status: "totp_enrollment_required" }),
      "POST /auth/totp/setup": () =>
        json({
          secret: "JBSWY3DPEHPK3PXP",
          otpauth_uri: "otpauth://totp/La:admin?secret=JBSWY3DPEHPK3PXP",
        }),
      "POST /auth/totp/enable": () => {
        loggedIn = true;
        return json(
          { ...session, recovery_codes: ["AAAA-BBBB-CCCC-DDDD", "EEEE-FFFF-GGGG-HHHH"] },
          201,
        );
      },
      "GET /readyz": ready,
    });
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Usuário"), "admin");
    await userEvent.type(screen.getByLabelText("Senha"), "uma-senha-longa-123");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));

    expect(await screen.findByDisplayValue("JBSWY3DPEHPK3PXP")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText(/Código do autenticador/), "123456");
    await userEvent.click(screen.getByRole("button", { name: "Verificar" }));

    // Os códigos aparecem e o painel só abre depois da confirmação explícita.
    const list = await screen.findByRole("list", { name: "Códigos de recuperação" });
    expect(within(list).getByText("AAAA-BBBB-CCCC-DDDD")).toBeInTheDocument();
    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    const next = screen.getByRole("button", { name: "Continuar" });
    expect(next).toBeDisabled();
    await userEvent.click(screen.getByLabelText(/Guardei os códigos/));
    await userEvent.click(next);
    expect(await screen.findByRole("heading", { name: "Central de comando" })).toBeInTheDocument();
  });

  it("código de recuperação é uma alternativa ao TOTP", async () => {
    let loggedIn = false;
    stub({
      "GET /session": () => (loggedIn ? json({ ...session, totp_reset_required: true }) : unauth()),
      "POST /auth/login": () => json({ status: "totp_required" }),
      "POST /auth/recover": (c) => {
        expect(c.body).toEqual({ recovery_code: "AAAA-BBBB-CCCC-DDDD" });
        loggedIn = true;
        return json({ ...session, totp_reset_required: true });
      },
      "GET /readyz": ready,
    });
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Usuário"), "admin");
    await userEvent.type(screen.getByLabelText("Senha"), "uma-senha-longa-123");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));
    await userEvent.click(
      await screen.findByRole("button", { name: "Usar código de recuperação" }),
    );
    await userEvent.type(screen.getByLabelText("Código de recuperação"), "AAAA-BBBB-CCCC-DDDD");
    await userEvent.click(screen.getByRole("button", { name: "Verificar" }));
    expect(await screen.findByRole("heading", { name: "Central de comando" })).toBeInTheDocument();
  });

  it("desafio expirado volta para a senha", async () => {
    stub({
      "GET /session": unauth,
      "POST /auth/login": () => json({ status: "totp_required" }),
      "POST /auth/totp": () => err(401, "challenge_expired", "A verificação expirou."),
    });
    renderAt("/");
    await userEvent.type(await screen.findByLabelText("Usuário"), "admin");
    await userEvent.type(screen.getByLabelText("Senha"), "uma-senha-longa-123");
    await userEvent.click(screen.getByRole("button", { name: "Continuar" }));
    await userEvent.type(await screen.findByLabelText(/Código do autenticador/), "123456");
    await userEvent.click(screen.getByRole("button", { name: "Verificar" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("A verificação expirou.");
    expect(screen.getByRole("heading", { name: "Entrar" })).toBeInTheDocument();
  });
});

const baseConn: Connection = {
  id: "11111111-1111-4111-8111-111111111111",
  kind: "github",
  name: "GitHub principal",
  target: "org",
  scopes: "repo",
  state: "active",
  secret_version: 1,
  last_test_status: "not_run",
  last_test_detail: "",
  version: 1,
  created_at: "2026-11-01T10:00:00Z",
  updated_at: "2026-11-01T10:00:00Z",
};

describe("conexões (cofre)", () => {
  const base = { "GET /session": () => json(session), "GET /readyz": ready };

  it("lista sem segredo; criar envia o valor uma vez, limpa o campo e usa o CSRF", async () => {
    let items: Connection[] = [];
    const calls = stub({
      ...base,
      "GET /connections": () => json({ items }),
      "POST /connections": (c) => {
        expect(c.csrf).toBe("csrf-1");
        items = [baseConn];
        return json(baseConn, 201);
      },
    });
    renderAt("/connections");
    expect(await screen.findByText("Nenhuma conexão cadastrada")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText("Nome"), "GitHub principal");
    await userEvent.type(screen.getByLabelText("Segredo"), "ghp_valor-super-secreto");
    await userEvent.click(screen.getByRole("button", { name: "Guardar no cofre" }));
    expect(await screen.findByText("GitHub principal")).toBeInTheDocument();
    expect(screen.getByLabelText("Segredo")).toHaveValue("");
    expect(document.body.textContent).not.toContain("ghp_valor-super-secreto");
    expect(calls.find((c) => c.method === "POST" && c.url === "/connections")?.body).toMatchObject({
      secret: "ghp_valor-super-secreto",
    });
    expect(screen.getByText(`connection:${baseConn.id}`)).toBeInTheDocument();
  });

  it("ação sensível sem reautenticação pede senha + código e repete a ação", async () => {
    let attempts = 0;
    let reauthed = false;
    let conn = baseConn;
    stub({
      ...base,
      "GET /connections": () => json({ items: [conn] }),
      "POST /auth/reauth": (c) => {
        expect(c.body).toEqual({ password: "minha-senha-123456", code: "654321" });
        reauthed = true;
        return json(session);
      },
      [`POST /connections/${baseConn.id}/revoke`]: () => {
        attempts++;
        if (reauthed) conn = { ...baseConn, state: "revoked", revoked_at: "2026-11-12T10:00:00Z" };
        return reauthed
          ? json(conn)
          : err(403, "reauth_required", "Esta ação exige confirmar sua identidade.");
      },
    });
    renderAt("/connections");
    await userEvent.click(await screen.findByRole("button", { name: "Revogar" }));
    await userEvent.click(screen.getByRole("button", { name: "Confirmar revogação" }));

    const dialog = await screen.findByRole("dialog", { name: "Confirme sua identidade" });
    await userEvent.type(within(dialog).getByLabelText("Senha"), "minha-senha-123456");
    await userEvent.type(within(dialog).getByLabelText(/Código do autenticador/), "654321");
    await userEvent.click(within(dialog).getByRole("button", { name: "Confirmar" }));

    expect(await screen.findByText("Revogada")).toBeInTheDocument();
    expect(attempts).toBe(2);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("cancelar a reautenticação mantém a conexão e mostra o erro", async () => {
    stub({
      ...base,
      "GET /connections": () => json({ items: [baseConn] }),
      [`POST /connections/${baseConn.id}/revoke`]: () =>
        err(403, "reauth_required", "Esta ação exige confirmar sua identidade."),
    });
    renderAt("/connections");
    await userEvent.click(await screen.findByRole("button", { name: "Revogar" }));
    await userEvent.click(screen.getByRole("button", { name: "Confirmar revogação" }));
    await userEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", { name: "Cancelar" }),
    );
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(
      screen.getByText("Esta ação exige confirmar sua identidade.", { exact: false }),
    ).toBeInTheDocument();
    expect(screen.getByText("Ativa")).toBeInTheDocument();
  });

  it("testar mostra o resultado; rotacionar envia a versão esperada", async () => {
    let conn = baseConn;
    const calls = stub({
      ...base,
      "GET /connections": () => json({ items: [conn] }),
      [`POST /connections/${baseConn.id}/test`]: () => {
        conn = {
          ...conn,
          last_test_status: "ok",
          last_test_detail: "Credencial aceita pelo GitHub.",
        };
        return json(conn);
      },
      [`POST /connections/${baseConn.id}/rotate`]: () => {
        conn = {
          ...conn,
          secret_version: 2,
          version: 2,
          last_test_status: "not_run",
          last_test_detail: "",
        };
        return json(conn);
      },
    });
    renderAt("/connections");
    await userEvent.click(await screen.findByRole("button", { name: "Testar" }));
    expect(await screen.findByText(/Credencial aceita pelo GitHub/)).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Substituir segredo" }));
    await userEvent.type(screen.getByLabelText(/Novo segredo/), "novo-valor-secreto");
    await userEvent.click(screen.getByRole("button", { name: "Substituir" }));
    await waitFor(() => expect(screen.getByText(/segredo v2/)).toBeInTheDocument());
    const rotate = calls.find((c) => c.url.endsWith("/rotate"));
    expect(rotate?.body).toEqual({ secret: "novo-valor-secreto", expected_version: 1 });
    expect(document.body.textContent).not.toContain("novo-valor-secreto");
  });
});

describe("segurança da conta", () => {
  const sessions = {
    items: [
      {
        id: "s1",
        created_at: "2026-11-12T10:00:00Z",
        expires_at: "2026-11-12T22:00:00Z",
        ip: "10.0.0.1",
        user_agent: "Firefox",
        auth_method: "totp",
        current: true,
      },
      {
        id: "s2",
        created_at: "2026-11-11T10:00:00Z",
        expires_at: "2026-11-12T20:00:00Z",
        ip: "10.0.0.2",
        user_agent: "Chrome",
        auth_method: "recovery",
        current: false,
      },
    ],
  };
  const events = {
    items: [
      {
        id: 1,
        type: "login_success",
        ip: "10.0.0.1",
        user_agent: "Firefox",
        created_at: "2026-11-12T10:00:00Z",
      },
    ],
  };

  it("lista sessões e eventos; encerrar outra sessão usa DELETE com CSRF", async () => {
    const calls = stub({
      "GET /session": () => json(session),
      "GET /readyz": ready,
      "GET /sessions": () => json(sessions),
      "GET /auth/events": () => json(events),
      "DELETE /sessions/s2": () => json({ status: "ok" }),
    });
    renderAt("/security");
    expect(await screen.findByText("Esta sessão")).toBeInTheDocument();
    expect(screen.getByText("Login concluído")).toBeInTheDocument();
    expect(screen.getByText("recuperação")).toBeInTheDocument();
    await userEvent.click(screen.getByRole("button", { name: "Encerrar" }));
    await waitFor(() =>
      expect(calls.some((c) => c.method === "DELETE" && c.url === "/sessions/s2")).toBe(true),
    );
    expect(calls.find((c) => c.method === "DELETE")?.csrf).toBe("csrf-1");
  });

  it("após recuperação avisa para substituir o 2FA", async () => {
    stub({
      "GET /session": () => json({ ...session, totp_reset_required: true }),
      "GET /readyz": ready,
      "GET /sessions": () => json(sessions),
      "GET /auth/events": () => json(events),
    });
    renderAt("/security");
    expect(await screen.findByText(/Substitua o segundo fator/)).toBeInTheDocument();
  });

  it("substituir o 2FA: mostra a nova chave e os novos códigos uma única vez", async () => {
    stub({
      "GET /session": () => json(session),
      "GET /readyz": ready,
      "GET /sessions": () => json(sessions),
      "GET /auth/events": () => json(events),
      "POST /auth/totp/replace/begin": () =>
        json({ secret: "NOVASECRET234567", otpauth_uri: "otpauth://totp/x" }),
      "POST /auth/totp/replace/confirm": () => json({ recovery_codes: ["ZZZZ-YYYY-XXXX-WWWW"] }),
    });
    renderAt("/security");
    await userEvent.click(await screen.findByRole("button", { name: "Substituir segundo fator" }));
    expect(await screen.findByDisplayValue("NOVASECRET234567")).toBeInTheDocument();
    await userEvent.type(screen.getByLabelText(/NOVO autenticador/), "123456");
    await userEvent.click(screen.getByRole("button", { name: "Confirmar substituição" }));
    expect(await screen.findByText("ZZZZ-YYYY-XXXX-WWWW")).toBeInTheDocument();
  });

  it("sair encerra a sessão no servidor e volta ao login", async () => {
    let out = false;
    const calls = stub({
      "GET /session": () => (out ? unauth() : json(session)),
      "GET /readyz": ready,
      "GET /sessions": () => json(sessions),
      "GET /auth/events": () => json(events),
      "POST /auth/logout": () => {
        out = true;
        return json({ status: "ok" });
      },
    });
    renderAt("/security");
    const [header] = await screen.findAllByRole("button", { name: "Sair" });
    await userEvent.click(header as HTMLElement);
    expect(await screen.findByRole("heading", { name: "Entrar" })).toBeInTheDocument();
    expect(calls.some((c) => c.url === "/auth/logout" && c.csrf === "csrf-1")).toBe(true);
  });
});
