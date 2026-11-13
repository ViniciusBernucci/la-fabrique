import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { api, type BudgetStatus, type ProviderState, type ProviderView } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

/** micro-USD → "US$ 1,2345". */
export function usd(micros: number): string {
  return `US$ ${(micros / 1_000_000).toLocaleString("pt-BR", { minimumFractionDigits: 2, maximumFractionDigits: 4 })}`;
}

const periodLabel = { day: "dia", month: "mês" } as const;

const sourceLabel: Record<string, string> = {
  provider_reported: "Reportado pelo provedor",
  estimated: "Estimado (sem valor reportado)",
  reconciliation: "Ajuste de reconciliação",
};

export const stateLabel: Record<ProviderState, string> = {
  available: "Disponível",
  auth: "Autenticação recusada",
  quota: "Cota esgotada",
  unavailable: "Indisponível",
  billing: "Bloqueio financeiro",
};

const when = (iso?: string | null) => (iso ? new Date(iso).toLocaleString("pt-BR") : "");

/** Um teto bloqueia novas chamadas quando não sobra saldo (gasto + reservas). */
export function blocks(b: BudgetStatus): boolean {
  return b.state === "active" && b.remaining_micros <= 0;
}

/** Consumo, orçamento e provedores (T16-03). Valor não observado aparece como tal. */
export function UsagePage() {
  const { expire } = useSession();
  const budgets = useResource((s) => api.listBudgets(s), "budgets", expire);
  const usage = useResource((s) => api.usageReport(s), "usage", expire);
  const providers = useResource((s) => api.listProviders(s), "providers", expire);
  const agents = useResource((s) => api.listAgents(s), "usage-agents", expire);
  const agentName = (id: string) =>
    agents.state.status === "ready"
      ? (agents.state.data.items.find((a) => a.id === id)?.name ?? `agente ${id.slice(0, 8)}`)
      : `agente ${id.slice(0, 8)}`;

  return (
    <div className="page">
      <PageHeader
        label="Consumo"
        title="Consumo, orçamento e provedores"
        lead="Gasto em API e uso de assinatura são mostrados separados e nunca somados. O que não foi observado aparece como não observado, nunca como zero."
      />

      <section className="panel" aria-label="Orçamentos">
        <div className="panel-head">
          <h2>Tetos de gasto em API</h2>
        </div>
        {budgets.state.status === "loading" && <LoadingState label="Carregando orçamentos…" />}
        {budgets.state.status === "error" && (
          <ErrorState error={budgets.state.error} onRetry={budgets.reload} />
        )}
        {budgets.state.status === "ready" && (
          <>
            {!budgets.state.data.items.some(
              (b) => b.scope === "global" && b.state === "active",
            ) && (
              <p className="state-meta" role="alert">
                Sem teto global ativo: as chamadas em API são recusadas até haver um orçamento.
              </p>
            )}
            {budgets.state.data.items.length === 0 ? (
              <EmptyState title="Nenhum orçamento definido">
                Defina um teto global para liberar chamadas em API.
              </EmptyState>
            ) : (
              <ul className="item-list">
                {budgets.state.data.items.map((b) => (
                  <li key={b.id}>
                    <div className="item-link">
                      <span>
                        <strong>
                          {b.scope === "global" ? "Global" : agentName(b.agent_id ?? "")} ·{" "}
                          {periodLabel[b.period]}
                        </strong>
                        <small>
                          Teto {usd(b.limit_micros)} · gasto {usd(b.spent_micros)} · reservado{" "}
                          {usd(b.reserved_micros)} · disponível {usd(b.remaining_micros)}
                        </small>
                      </span>
                      <span className={`badge ${blocks(b) ? "badge--paused" : "badge--active"}`}>
                        {b.state === "disabled"
                          ? "Desativado"
                          : blocks(b)
                            ? "Bloqueia novas chamadas"
                            : "Dentro do teto"}
                      </span>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}
      </section>

      <section className="panel" aria-label="Uso do período">
        <div className="panel-head">
          <h2>Uso do mês</h2>
        </div>
        {usage.state.status === "loading" && <LoadingState label="Carregando uso…" />}
        {usage.state.status === "error" && (
          <ErrorState error={usage.state.error} onRetry={usage.reload} />
        )}
        {usage.state.status === "ready" && (
          <>
            <dl className="facts">
              <div>
                <dt>Gasto em API</dt>
                <dd>{usd(usage.state.data.api.spent_micros)}</dd>
              </div>
              <div>
                <dt>Reservado</dt>
                <dd>{usd(usage.state.data.api.reserved_micros)}</dd>
              </div>
              <div>
                <dt>Acima do estimado</dt>
                <dd>{usd(usage.state.data.api.overrun_micros)}</dd>
              </div>
            </dl>
            <ul className="item-list" aria-label="Gasto por origem do valor">
              {Object.entries(usage.state.data.api.by_source).map(([src, v]) => (
                <li key={src}>
                  <div className="item-link">
                    <span>
                      <strong>{sourceLabel[src] ?? src}</strong>
                    </span>
                    <span>{usd(v)}</span>
                  </div>
                </li>
              ))}
              {Object.keys(usage.state.data.api.by_source).length === 0 && (
                <li>
                  <div className="item-link">
                    <span>
                      <strong>Nenhum gasto em API registrado neste período</strong>
                    </span>
                  </div>
                </li>
              )}
            </ul>
            <ul className="item-list" aria-label="Gasto por agente">
              {Object.entries(usage.state.data.api.by_agent).map(([id, v]) => (
                <li key={id || "sem-agente"}>
                  <div className="item-link">
                    <span>
                      <strong>{id ? agentName(id) : "Sem agente associado"}</strong>
                    </span>
                    <span>{usd(v)}</span>
                  </div>
                </li>
              ))}
            </ul>
          </>
        )}
      </section>

      <section className="panel" aria-label="Provedores">
        <div className="panel-head">
          <h2>Provedores e cotas</h2>
        </div>
        {providers.state.status === "loading" && <LoadingState label="Carregando provedores…" />}
        {providers.state.status === "error" && (
          <ErrorState error={providers.state.error} onRetry={providers.reload} />
        )}
        {providers.state.status === "ready" && providers.state.data.items.length === 0 && (
          <EmptyState title="Nenhuma conta de IA cadastrada">
            Cadastre uma conta de assinatura (CLI) ou de API em Conexões.
          </EmptyState>
        )}
        {providers.state.status === "ready" && (
          <ul className="item-list">
            {providers.state.data.items.map((p) => (
              <ProviderRow
                key={p.connection_id}
                provider={p}
                units={
                  usage.state.status === "ready"
                    ? usage.state.data.subscription.filter(
                        (l) => l.connection_id === p.connection_id,
                      )
                    : []
                }
              />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function ProviderRow({
  provider: p,
  units,
}: {
  provider: ProviderView;
  units: { unit: string; units: number; calls: number }[];
}) {
  const ok = p.state === "active" && p.health.state === "available";
  return (
    <li>
      <div className="item-link">
        <span>
          <strong>
            {p.name} · {p.kind === "cli" ? "assinatura (CLI)" : `API (${p.target})`}
          </strong>
          <small>
            {p.state === "revoked"
              ? "Conexão revogada."
              : p.health.state === "available"
                ? "Sem falha registrada."
                : `${(p.health.reason || "Sem detalhe").replace(/\.$/, "")}. Sinal ${p.health.source === "official" ? "oficial" : "do histórico"}${p.health.until ? `; volta a ser tentada após ${when(p.health.until)}` : "; só volta após revalidação"}.`}
          </small>
          <small>
            {p.quota
              ? `Cota observada: ${p.quota.used} de ${p.quota.limit} ${p.quota.unit} (fonte ${p.quota.source}, em ${when(p.quota.observed_at)}${p.quota.resets_at ? `; renova ${when(p.quota.resets_at)}` : ""}).`
              : p.kind === "cli"
                ? "Cota não observada: o provedor ainda não informou."
                : "Sem cota de assinatura (conta de API)."}
          </small>
          {units.map((l) => (
            <small key={l.unit}>
              Consumo no período: {l.units} {l.unit} em {l.calls} chamada(s).
              {p.quota ? "" : " Sem limite observado."}
            </small>
          ))}
        </span>
        <span className={`badge ${ok ? "badge--active" : "badge--paused"}`}>
          {p.state === "revoked" ? "Revogada" : stateLabel[p.health.state]}
        </span>
      </div>
    </li>
  );
}
