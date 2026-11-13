import { type FormEvent, useEffect, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  api,
  type Delivery,
  type DeliveryDetail,
  type DeliveryState,
  type ObserveResult,
} from "../lib/api";
import { useReauth } from "../lib/reauth";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const stateText: Record<DeliveryState, string> = {
  blocked: "Bloqueada",
  draft_open: "Draft PR aberta",
  merged: "Integrada",
  closed: "PR fechada sem merge",
  cancelled: "Cancelada",
};

const commentText = {
  queued: "na fila",
  held: "retido",
  fixed: "corrigido",
  dismissed: "substituído",
};

const outcomeText: Record<ObserveResult["observation"]["outcome"], string> = {
  open: "A PR segue aberta: nada foi liberado.",
  closed: "A PR foi fechada sem merge: nada foi liberado. Escolha uma ação abaixo.",
  merged: "Merge confirmado.",
  unknown: "Não foi possível confirmar o estado da PR: nada foi alterado.",
};

const projectionText = {
  none: "Origem: sem movimento",
  pending: "Origem: pendente",
  done: "Origem: concluída",
  failed: "Origem: falhou (consulte de novo para reconciliar)",
} as const;

const when = (iso: string) => new Date(iso).toLocaleString("pt-BR");

/** Entregas do projeto: Draft PR, merge observado sob demanda, correções externas e memória (E15). */
export function DeliveryPanel({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.deliveries(projectId, s),
    `deliveries:${projectId}`,
    expire,
  );

  return (
    <>
      <section className="panel" aria-label="Entregas">
        <div className="panel-head">
          <h2>Entregas</h2>
        </div>
        {state.status === "loading" && <LoadingState label="Carregando entregas…" />}
        {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
        {state.status === "ready" && state.data.items.length === 0 && (
          <EmptyState title="Nenhuma entrega ainda">
            Quando uma feature chegar à Draft PR, ela aparecerá aqui com o estado do merge.
          </EmptyState>
        )}
        {state.status === "ready" && state.data.items.length > 0 && (
          <ul className="item-list" aria-label="Entregas do projeto">
            {state.data.items.map((d) => (
              <DeliveryItem key={d.id} projectId={projectId} delivery={d} onChange={reload} />
            ))}
          </ul>
        )}
      </section>
      <DeliverySettingsForm projectId={projectId} />
    </>
  );
}

function DeliveryItem({
  projectId,
  delivery: d,
  onChange,
}: {
  projectId: string;
  delivery: Delivery;
  onChange: () => void;
}) {
  const { runSensitive } = useReauth();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState<string[]>([]);
  const [open, setOpen] = useState(false);
  const [detail, setDetail] = useState<DeliveryDetail | null>(null);
  const [taskKey, setTaskKey] = useState("");
  const [note, setNote] = useState("");
  const [report, setReport] = useState<string | null>(null);

  async function loadDetail() {
    try {
      setDetail(await api.deliveryDetail(projectId, d.id));
    } catch (e) {
      setError(toApiError(e));
    }
  }

  async function run(action: () => Promise<string[]>) {
    setBusy(true);
    setError(null);
    try {
      setNotice(await action());
      onChange();
      if (open) await loadDetail();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const observe = () =>
    run(async () => {
      const r = await api.observeDelivery(projectId, d.id);
      const lines = [outcomeText[r.observation.outcome]];
      if (r.observation.outcome === "merged") {
        lines.push(`Itens devolvidos à fila: ${r.observation.released.length}.`);
        if (r.origin_note) lines.push(r.origin_note);
        if (r.memory)
          lines.push(
            `Memória: ${r.memory.promoted} promovida(s), ${r.memory.rejected} invalidada(s), ${r.memory.duplicates} duplicada(s).`,
          );
        if (r.memory_error) lines.push(r.memory_error);
      }
      if (r.observation.detail && r.observation.outcome === "unknown")
        lines.push(r.observation.detail);
      return lines;
    });

  function discover(event: FormEvent) {
    event.preventDefault();
    return run(async () => {
      const r = await api.discoverComments(projectId, d.id, taskKey.trim());
      const x = r.discovery;
      const lines = [
        `Comentários novos: ${x.new}; ignorados (autor não autorizado ou vazio): ${x.ignored}.`,
      ];
      if (r.authorized === 0)
        lines.push("Nenhum autor está autorizado: configure os autores abaixo.");
      if (x.held > 0) lines.push(`${x.held} comentário(s) retido(s) até a decisão sobre a PR.`);
      if (x.item) lines.push("Correção enfileirada para a próxima vaga.");
      return lines;
    });
  }

  const act = (action: string, label: string) =>
    run(async () => {
      const r = await runSensitive(() => api.deliveryAction(projectId, d.id, action, note));
      return [
        `${label}: ${r.replayed ? "já registrada" : "registrada"}${r.next_attempt ? ` (próxima tentativa: ${r.next_attempt})` : ""}.`,
      ];
    });

  async function toggle() {
    const next = !open;
    setOpen(next);
    if (next && detail === null) await loadDetail();
  }

  async function showReport() {
    try {
      setReport((await api.featureReport(projectId, d.feature_id)).markdown);
    } catch (e) {
      setError(toApiError(e));
    }
  }

  const counts = Object.entries(d.comments)
    .map(([k, n]) => `${n} ${commentText[k as keyof typeof commentText] ?? k}`)
    .join(", ");
  const canObserve = d.state === "draft_open" || d.state === "closed" || d.state === "merged";

  return (
    <li aria-label={`${d.feature_key} tentativa ${d.attempt}`}>
      <div className="item-link">
        <span>
          <strong>
            {d.feature_key} · tentativa {d.attempt}
          </strong>
          <small>{d.feature_title}</small>
          {d.pr_url && (
            <small>
              <a href={d.pr_url} target="_blank" rel="noopener noreferrer">
                {d.pr_url}
              </a>
            </small>
          )}
          {d.merge_sha && <small>Merge: {d.merge_sha.slice(0, 12)}</small>}
          {d.state === "merged" && <small>{projectionText[d.projection]}</small>}
          {d.projection_error && <small>{d.projection_error}</small>}
          {d.reasons.map((r) => (
            <small key={r}>Bloqueio: {r}</small>
          ))}
          {counts && <small>Comentários: {counts}</small>}
          <small>
            {d.observed_at ? `Última consulta: ${when(d.observed_at)}` : "PR ainda não consultada"}
          </small>
        </span>
        <span
          className={`badge ${d.state === "merged" ? "badge--active" : d.state === "draft_open" ? "" : "badge--paused"}`}
        >
          {stateText[d.state]}
        </span>
      </div>
      <div className="panel--pad">
        <FormError error={error} />
        {notice.length > 0 && (
          <div className="callout" role="status">
            {notice.map((n) => (
              <span key={n}>{n}</span>
            ))}
          </div>
        )}
        <div className="actions">
          {canObserve && (
            <button type="button" disabled={busy} onClick={observe}>
              Consultar PR
            </button>
          )}
          <button type="button" className="secondary-action" aria-expanded={open} onClick={toggle}>
            {open ? "Ocultar detalhes" : "Detalhes"}
          </button>
          <button type="button" className="secondary-action" onClick={showReport}>
            Histórico da feature
          </button>
        </div>
        {report !== null && (
          <pre className="log" role="log" aria-label="Histórico da feature">
            {report}
          </pre>
        )}
        {d.state === "draft_open" && (
          <form className="inline-form" onSubmit={discover}>
            <input
              aria-label="Tarefa que recebe a correção"
              placeholder="Tarefa que recebe a correção"
              value={taskKey}
              maxLength={100}
              onChange={(e) => setTaskKey(e.target.value)}
            />
            <button
              type="submit"
              className="secondary-action"
              disabled={busy || taskKey.trim() === ""}
            >
              Buscar comentários da revisão
            </button>
          </form>
        )}
        {open && detail && (
          <DetailBody detail={detail} note={note} setNote={setNote} busy={busy} act={act} />
        )}
      </div>
    </li>
  );
}

function DetailBody({
  detail,
  note,
  setNote,
  busy,
  act,
}: {
  detail: DeliveryDetail;
  note: string;
  setNote: (v: string) => void;
  busy: boolean;
  act: (action: string, label: string) => void;
}) {
  return (
    <div>
      {detail.offer && (
        <div className="callout callout--warn">
          <strong>A PR foi fechada sem merge. O que fazer?</strong>
          <label>
            Observação (opcional)
            <input value={note} maxLength={500} onChange={(e) => setNote(e.target.value)} />
          </label>
          <div className="actions">
            {detail.offer.options.map((o) => (
              <button
                key={o.id}
                type="button"
                className={o.id === "reopen" ? undefined : "secondary-action"}
                title={o.description}
                disabled={busy}
                onClick={() => act(o.id, o.label)}
              >
                {o.label}
              </button>
            ))}
          </div>
          <small className="muted">
            {detail.offer.options.map((o) => `${o.label}: ${o.description}`).join(" · ")}
          </small>
        </div>
      )}
      <h3 className="form-title">Comentários da revisão</h3>
      {detail.comments.length === 0 ? (
        <p className="state-meta">Nenhum comentário de autor autorizado registrado.</p>
      ) : (
        <ul className="item-list" aria-label="Comentários">
          {detail.comments.map((c) => (
            <li key={c.id}>
              <div className="item-link">
                <span>
                  <strong>{c.author}</strong>
                  <small>{c.body}</small>
                </span>
                <span className="badge">{commentText[c.state]}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
      {detail.history.length > 0 && (
        <p className="state-meta">Histórico de ações: {detail.history.join(" → ")}</p>
      )}
      {detail.observations.length > 0 && (
        <ul className="item-list" aria-label="Consultas da PR">
          {detail.observations.map((o) => (
            <li key={`${o.created_at}-${o.outcome}`}>
              <div className="item-link">
                <span>
                  <strong>{o.outcome}</strong>
                  <small>{o.detail}</small>
                </span>
                <small>{when(o.created_at)}</small>
              </div>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/** Quem pode pedir correção na PR e como a origem chama os estados da fábrica. */
function DeliverySettingsForm({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.deliverySettings(projectId, s),
    `delivery-settings:${projectId}`,
    expire,
  );
  const [authors, setAuthors] = useState("");
  const [awaiting, setAwaiting] = useState("");
  const [done, setDone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (state.status !== "ready") return;
    setAuthors(state.data.review_authors.join(", "));
    setAwaiting(state.data.status_map.awaiting_merge ?? "");
    setDone(state.data.status_map.done ?? "");
  }, [state]);

  async function save(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    setSaved(false);
    try {
      await api.saveDeliverySettings(projectId, {
        review_authors: authors
          .split(",")
          .map((a) => a.trim())
          .filter(Boolean),
        status_map: { awaiting_merge: awaiting.trim(), done: done.trim() },
      });
      setSaved(true);
      reload();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Configuração da entrega">
      <h2 className="form-title">Configuração da entrega</h2>
      {state.status === "loading" && <LoadingState label="Carregando configuração…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && (
        <form className="form form--grid" onSubmit={save}>
          <label className="form-wide">
            Autores autorizados a pedir correção (logins do GitHub, separados por vírgula)
            <input value={authors} onChange={(e) => setAuthors(e.target.value)} />
          </label>
          <label>
            Estado "Aguardando merge" na origem
            <input value={awaiting} maxLength={100} onChange={(e) => setAwaiting(e.target.value)} />
          </label>
          <label>
            Estado "Concluída" na origem
            <input value={done} maxLength={100} onChange={(e) => setDone(e.target.value)} />
          </label>
          <FormError error={error} />
          {saved && (
            <p className="form-note" role="status">
              Configuração salva.
            </p>
          )}
          <div className="actions">
            <button type="submit" disabled={busy}>
              Salvar configuração
            </button>
          </div>
          <p className="form-note">
            Sem o estado "Concluída", o merge libera a fila e promove a memória, mas o item da
            origem não é movido.
          </p>
        </form>
      )}
    </section>
  );
}
