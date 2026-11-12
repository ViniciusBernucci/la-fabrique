import { type FormEvent, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type DecisionKind, type DecisionRequest } from "../lib/api";
import { useReauth } from "../lib/reauth";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const kindLabel: Record<DecisionKind, string> = {
  conflict: "Conflito",
  missing_requirement: "Requisito faltando",
  sensitive_operation: "Operação sensível",
  uncertain_result: "Resultado incerto",
  pr_closed: "PR fechada",
  other: "Outro",
};

type Tab = "pending" | "resolved" | "stale";
const tabLabel: Record<Tab, string> = {
  pending: "Pendentes",
  resolved: "Resolvidas",
  stale: "Obsoletas",
};

const when = (iso: string) => new Date(iso).toLocaleString("pt-BR");

/** Central de Decisões: pedidos humanos das execuções (T13-02). */
export function DecisionsPage() {
  const { expire } = useSession();
  const [tab, setTab] = useState<Tab>("pending");
  const { state, reload } = useResource(
    (s) => api.listDecisions(tab, s),
    `decisions:${tab}`,
    expire,
  );

  return (
    <div className="page">
      <PageHeader
        label="Decisões"
        title="Central de Decisões"
        lead="Aprovações e escolhas que dependem de você. Cada decisão afeta somente a execução que a pediu."
      />
      <div className="stepper" role="tablist" aria-label="Estado dos pedidos">
        {(Object.keys(tabLabel) as Tab[]).map((t) => (
          <button
            key={t}
            type="button"
            role="tab"
            aria-selected={t === tab}
            className="stepper-item"
            aria-current={t === tab ? "step" : undefined}
            onClick={() => setTab(t)}
          >
            {tabLabel[t]}
          </button>
        ))}
      </div>
      <section className="panel" aria-label={`Decisões ${tabLabel[tab].toLowerCase()}`}>
        {state.status === "loading" && <LoadingState label="Carregando decisões…" />}
        {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
        {state.status === "ready" && state.data.items.length === 0 && (
          <EmptyState title={tab === "pending" ? "Nenhuma decisão pendente" : "Nada por aqui"}>
            {tab === "pending"
              ? "Quando uma execução precisar de aprovação ou decisão humana, ela aparecerá aqui."
              : "Nenhum pedido neste estado."}
          </EmptyState>
        )}
        {state.status === "ready" &&
          state.data.items.map((d) => <DecisionCard key={d.id} decision={d} onDone={reload} />)}
      </section>
    </div>
  );
}

function DecisionCard({ decision: d, onDone }: { decision: DecisionRequest; onDone: () => void }) {
  const { runSensitive } = useReauth();
  const [option, setOption] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await runSensitive(() => api.resolveDecision(d.id, option, note));
      onDone();
    } catch (e) {
      setError(toApiError(e));
      // Conflito ou pedido obsoleto: a lista mudou no servidor.
      if (e instanceof Error && "code" in e && String((e as ApiError).code).startsWith("decision_"))
        onDone();
    } finally {
      setBusy(false);
    }
  }

  const chosen = d.options.find((o) => o.id === d.resolution_option);
  return (
    <article className="panel panel--pad decision" aria-label={d.title}>
      <header className="actions">
        <h2 className="form-title">{d.title}</h2>
        <span className={d.kind === "sensitive_operation" ? "badge badge--paused" : "badge"}>
          {kindLabel[d.kind]}
        </span>
        <small className="muted">{when(d.created_at)}</small>
      </header>
      <p>{d.problem}</p>
      {d.impact && (
        <p className="muted">
          <strong>Impacto:</strong> {d.impact}
        </p>
      )}
      {d.evidence.length > 0 && (
        <dl className="facts" aria-label="Evidência">
          {d.evidence.map((e) => (
            <div key={`${e.label}-${e.text}`}>
              <dt>{e.label}</dt>
              <dd>{e.text}</dd>
            </div>
          ))}
        </dl>
      )}
      {d.state === "pending" && (
        <form className="form" onSubmit={submit}>
          <fieldset className="fieldset-plain" disabled={busy}>
            <legend className="muted">Alternativas</legend>
            {d.options.map((o) => (
              <label key={o.id} className="check">
                <input
                  type="radio"
                  name={`opt-${d.id}`}
                  value={o.id}
                  checked={option === o.id}
                  onChange={() => setOption(o.id)}
                />
                <span>
                  <strong>{o.label}</strong>
                  {o.description && <small className="muted"> — {o.description}</small>}
                </span>
              </label>
            ))}
          </fieldset>
          <label>
            Observação (opcional)
            <input value={note} maxLength={500} onChange={(e) => setNote(e.target.value)} />
          </label>
          <FormError error={error} />
          <div className="actions">
            <button type="submit" disabled={busy || option === ""}>
              Registrar decisão
            </button>
          </div>
        </form>
      )}
      {d.state === "resolved" && (
        <p className="callout" role="status">
          Resolvida em {d.resolved_at ? when(d.resolved_at) : "—"}:{" "}
          <strong>{chosen?.label ?? d.resolution_option}</strong>
          {d.resolution_note && ` — ${d.resolution_note}`}
        </p>
      )}
      {d.state === "stale" && (
        <p className="callout callout--warn" role="status">
          Obsoleta: a execução mudou depois do pedido e a decisão não foi aplicada.
        </p>
      )}
    </article>
  );
}
