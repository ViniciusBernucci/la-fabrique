import { type FormEvent, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  type AuthorizationView,
  api,
  type DecisionKind,
  type DecisionRequest,
  type SequenceEntry,
} from "../lib/api";
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

type Tab = "pending" | "resolved" | "stale" | "authorizations";
const tabLabel: Record<Tab, string> = {
  pending: "Pendentes",
  resolved: "Resolvidas",
  stale: "Obsoletas",
  authorizations: "Autorizações",
};

const statusLabel: Record<AuthorizationView["status"], string> = {
  active: "Vigente",
  expired: "Vencida",
  exhausted: "Esgotada",
  revoked: "Revogada",
};

const refusalIds = ["refuse", "deny", "reject"];

const when = (iso: string) => new Date(iso).toLocaleString("pt-BR");

/** Central de Decisões: pedidos humanos das execuções (T13-02). */
export function DecisionsPage() {
  const { expire } = useSession();
  const [tab, setTab] = useState<Tab>("pending");
  const { state, reload } = useResource(
    (s) => (tab === "authorizations" ? Promise.resolve({ items: [] }) : api.listDecisions(tab, s)),
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
      {tab === "authorizations" && <AuthorizationsPanel />}
      <section
        className="panel"
        aria-label={`Decisões ${tabLabel[tab].toLowerCase()}`}
        hidden={tab === "authorizations"}
      >
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

  async function refuse() {
    setBusy(true);
    setError(null);
    try {
      await runSensitive(() => api.refuseDecision(d.id, note));
      onDone();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const refusal = d.options.find((o) => refusalIds.includes(o.id));
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
            {refusal && (
              <button type="button" className="secondary-action" disabled={busy} onClick={refuse}>
                Recusar
              </button>
            )}
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
      <SequenceView executionId={d.execution_id} />
    </article>
  );
}

const seqText = (e: SequenceEntry) => {
  const d = e.detail;
  const extra = [d.title, d.option, d.reason, d.note].find((v) => typeof v === "string" && v);
  return `${e.source} · ${e.type}${extra ? ` — ${String(extra)}` : ""}`;
};

/** Sequência do incidente e da retomada da execução (lida do servidor, sob demanda). */
function SequenceView({ executionId }: { executionId: string }) {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<SequenceEntry[] | null>(null);
  const [error, setError] = useState<ApiError | null>(null);

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next && items === null) {
      try {
        setItems((await api.executionSequence(executionId)).items);
      } catch (e) {
        setError(toApiError(e));
      }
    }
  }

  return (
    <div>
      <button type="button" className="secondary-action" aria-expanded={open} onClick={toggle}>
        {open ? "Ocultar sequência" : "Ver sequência"}
      </button>
      {open && (
        <section aria-label="Sequência do incidente e da retomada">
          <FormError error={error} />
          {items?.length === 0 && <p className="muted">Nenhum registro para esta execução.</p>}
          <ul className="item-list" aria-label="Linhas da sequência">
            {items?.map((e) => (
              <li key={`${e.at}-${e.source}-${e.type}-${JSON.stringify(e.detail)}`}>
                <div className="item-link">
                  <span>
                    <strong>{seqText(e)}</strong>
                  </span>
                  <small>{when(e.at)}</small>
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}

/** Autorizações: validade, uso e revogação (ação sensível). */
function AuthorizationsPanel() {
  const { expire } = useSession();
  const { runSensitive } = useReauth();
  const { state, reload } = useResource(
    (s) => api.listAuthorizations("all", s),
    "authorizations",
    expire,
  );
  const [reason, setReason] = useState("");
  const [error, setError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState("");

  async function revoke(a: AuthorizationView) {
    setError(null);
    setNotice("");
    try {
      const out = await runSensitive(() => api.revokeAuthorization(a.id, reason));
      setNotice(
        out.halt_failed
          ? "Revogada, mas a parada segura da execução falhou: verifique a execução."
          : "Autorização revogada.",
      );
      reload();
    } catch (e) {
      setError(toApiError(e));
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Autorizações">
      {state.status === "loading" && <LoadingState label="Carregando autorizações…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && state.data.items.length === 0 && (
        <EmptyState title="Nenhuma autorização">
          As aprovações de operações sensíveis geram autorizações com prazo e número de usos.
        </EmptyState>
      )}
      {state.status === "ready" && state.data.items.length > 0 && (
        <label>
          Motivo da revogação (opcional)
          <input value={reason} maxLength={500} onChange={(e) => setReason(e.target.value)} />
        </label>
      )}
      <FormError error={error} />
      {notice && (
        <p className="callout" role="status">
          {notice}
        </p>
      )}
      {state.status === "ready" &&
        state.data.items.map((a) => (
          <article key={a.id} className="decision" aria-label={`${a.tool} ${a.operation}`}>
            <header className="actions">
              <h2 className="form-title">
                {a.tool} · {a.operation}
              </h2>
              <span
                className={a.status === "active" ? "badge badge--active" : "badge badge--archived"}
              >
                {statusLabel[a.status]}
              </span>
            </header>
            <dl className="facts">
              <div>
                <dt>Alvo</dt>
                <dd>{a.target || a.resource || "—"}</dd>
              </div>
              <div>
                <dt>Usos</dt>
                <dd>
                  {a.uses} de {a.max_uses} ({a.remaining} restantes
                  {a.in_flight > 0 ? `, ${a.in_flight} em andamento` : ""})
                </dd>
              </div>
              <div>
                <dt>Validade</dt>
                <dd>{when(a.expires_at)}</dd>
              </div>
              {a.revoked_at && (
                <div>
                  <dt>Revogação</dt>
                  <dd>
                    {when(a.revoked_at)}
                    {a.revoke_reason && ` — ${a.revoke_reason}`}
                  </dd>
                </div>
              )}
            </dl>
            {a.status === "active" && (
              <div className="actions">
                <button type="button" onClick={() => revoke(a)}>
                  Revogar
                </button>
              </div>
            )}
            <SequenceView executionId={a.execution_id} />
          </article>
        ))}
    </section>
  );
}
