import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { App } from "../App";
import type { BudgetStatus, ProviderView, UsageReport } from "../lib/api";
import { SessionProvider } from "../lib/session";
import { blocks, usd } from "./UsagePage";

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

function stub(routes: Record<string, () => Response>) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init: RequestInit = {}) => {
      const url = String(input).replace(/^\/api/, "");
      const handler = routes[`${(init.method ?? "GET").toUpperCase()} ${url}`];
      return handler
        ? handler()
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

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <SessionProvider>
        <App />
      </SessionProvider>
    </MemoryRouter>,
  );
}

const budget = (over: Partial<BudgetStatus>): BudgetStatus => ({
  id: "b1",
  scope: "global",
  agent_id: null,
  period: "month",
  limit_micros: 10_000_000,
  state: "active",
  version: 1,
  spent_micros: 2_500_000,
  reserved_micros: 500_000,
  remaining_micros: 7_000_000,
  ...over,
});

const usage: UsageReport = {
  From: "2026-11-01T00:00:00Z",
  To: "2026-12-01T00:00:00Z",
  api: {
    spent_micros: 2_500_000,
    reserved_micros: 500_000,
    by_source: { estimated: 1_500_000, provider_reported: 1_000_000 },
    by_agent: { "": 300_000 },
    overrun_micros: 0,
  },
  subscription: [{ connection_id: "c-cli", unit: "mensagens", units: 12, calls: 5 }],
};

const providers: ProviderView[] = [
  {
    connection_id: "c-cli",
    name: "claude-assinatura",
    kind: "cli",
    target: "claude",
    state: "active",
    health: {
      connection_id: "c-cli",
      state: "available",
      source: "history",
      reason: "",
      consecutive_failures: 0,
    },
    quota: null,
  },
  {
    connection_id: "c-api",
    name: "gateway",
    kind: "api",
    target: "openrouter",
    state: "active",
    health: {
      connection_id: "c-api",
      state: "billing",
      source: "official",
      reason: "saldo insuficiente",
      consecutive_failures: 1,
    },
    quota: null,
  },
];

beforeEach(() => {
  vi.stubGlobal("matchMedia", () => ({
    matches: false,
    addEventListener() {},
    removeEventListener() {},
  }));
});
afterEach(() => vi.unstubAllGlobals());

describe("formatação", () => {
  it("converte micro-USD e detecta o teto que bloqueia", () => {
    expect(usd(1_234_500)).toBe("US$ 1,2345");
    expect(blocks(budget({ remaining_micros: 0 }))).toBe(true);
    expect(blocks(budget({ remaining_micros: 1 }))).toBe(false);
    expect(blocks(budget({ remaining_micros: 0, state: "disabled" }))).toBe(false);
  });
});

describe("Consumo, orçamento e provedores", () => {
  const routes = (b: BudgetStatus[]) => ({
    "GET /session": () => json(session),
    "GET /budgets": () => json({ items: b }),
    "GET /usage": () => json(usage),
    "GET /providers": () => json({ items: providers }),
    "GET /agents": () => json({ items: [] }),
  });

  it("mostra teto, reservas, origem do valor, cota não observada e disponibilidade", async () => {
    stub(routes([budget({})]));
    renderAt("/usage");
    expect(await screen.findByText(/Teto US\$ 10,00/)).toBeInTheDocument();
    expect(screen.getByText(/reservado US\$ 0,50/)).toBeInTheDocument();
    expect(screen.getByText("Dentro do teto")).toBeInTheDocument();
    const sources = within(screen.getByLabelText("Gasto por origem do valor"));
    expect(sources.getByText("Estimado (sem valor reportado)")).toBeInTheDocument();
    expect(sources.getByText("Reportado pelo provedor")).toBeInTheDocument();
    expect(screen.getByText("Sem agente associado")).toBeInTheDocument();
    // Valor não observado aparece explicitamente (nunca zero).
    expect(
      screen.getByText("Cota não observada: o provedor ainda não informou."),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        /Consumo no período: 12 mensagens em 5 chamada\(s\)\. Sem limite observado\./,
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Bloqueio financeiro")).toBeInTheDocument();
    expect(
      screen.getByText(/saldo insuficiente\. Sinal oficial; só volta após revalidação/),
    ).toBeInTheDocument();
    expect(screen.getByText("Sem cota de assinatura (conta de API).")).toBeInTheDocument();
  });

  it("reflete o bloqueio de novas chamadas quando o teto acabou e quando não há teto global", async () => {
    stub(routes([budget({ remaining_micros: 0, spent_micros: 10_000_000 })]));
    renderAt("/usage");
    expect(await screen.findByText("Bloqueia novas chamadas")).toBeInTheDocument();
    vi.unstubAllGlobals();
    vi.stubGlobal("matchMedia", () => ({
      matches: false,
      addEventListener() {},
      removeEventListener() {},
    }));
    stub(routes([]));
    renderAt("/usage");
    expect(
      await screen.findByText(/Sem teto global ativo: as chamadas em API são recusadas/),
    ).toBeInTheDocument();
  });
});
