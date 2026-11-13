import { type FormEvent, useEffect, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  api,
  type ControlMode,
  type ControlState,
  type ExecutionEvent,
  type QueueItemView,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useEventStream } from "../lib/useEventStream";
import { useResource } from "../lib/useResource";
import { AgentMonitor, GuidanceChat } from "./AgentMonitor";

const stateLabel: Record<QueueItemView["state"], string> = {
  queued: "Na fila",
  running: "Em execução",
  waiting: "Aguardando",
  done: "Concluída",
  cancelled: "Cancelada",
};

const modeText: Record<Exclude<ControlState["mode"], "active">, string> = {
  draining: "Pausa após a feature atual: nenhuma feature nova inicia; a atual termina.",
  suspended: "Suspenso: nada novo inicia e o que rodava foi parado com checkpoint.",
};

export function duration(seconds: number): string {
  if (seconds < 60) return `${seconds} s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min`;
  return `${Math.floor(seconds / 3600)} h ${Math.floor((seconds % 3600) / 60)} min`;
}

function ControlLine({ label, c }: { label: string; c: ControlState }) {
  if (c.mode === "active") return null;
  return (
    <p className="state-meta" role="status">
      <strong>{label}:</strong> {modeText[c.mode]}
      {c.reason ? ` Motivo: ${c.reason}.` : ""}
    </p>
  );
}

/** Controles do projeto: pausar após a feature atual, suspender ou retomar. */
function ProjectControls({
  projectId,
  mode,
  onChange,
}: {
  projectId: string;
  mode: ControlMode;
  onChange: () => void;
}) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [notice, setNotice] = useState("");

  async function apply(next: ControlMode) {
    setBusy(true);
    setError(null);
    setNotice("");
    try {
      const r = await api.setProjectControl(projectId, next, reason.trim());
      const first = r.failed?.[0];
      if (r.failed && first)
        setNotice(
          `${r.failed.length} tarefa(s) não puderam ser paradas com checkpoint seguro e seguem em execução: ${first.reason}`,
        );
      onChange();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel--pad" aria-label="Controles do projeto">
      <label>
        Motivo (opcional)
        <input value={reason} maxLength={500} onChange={(e) => setReason(e.target.value)} />
      </label>
      <div className="actions">
        {mode !== "active" && (
          <button type="button" disabled={busy} onClick={() => apply("active")}>
            Retomar projeto
          </button>
        )}
        {mode === "active" && (
          <button
            type="button"
            className="secondary-action"
            disabled={busy}
            onClick={() => apply("draining")}
          >
            Pausar após a feature atual
          </button>
        )}
        {mode !== "suspended" && (
          <button
            type="button"
            className="secondary-action"
            disabled={busy}
            onClick={() => apply("suspended")}
          >
            Suspender agora
          </button>
        )}
      </div>
      <FormError error={error} />
      {notice && (
        <p className="callout callout--warn" role="status">
          {notice}
        </p>
      )}
    </section>
  );
}

/** Execução do projeto: controles persistidos, tarefas com a causa do bloqueio e logs ao vivo (T16-01). */
export function RunPanel({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.projectOverview(projectId, s),
    `overview:${projectId}`,
    expire,
  );
  const [execution, setExecution] = useState<string | null>(null);

  // O resumo se atualiza sozinho; os logs seguem pelo fluxo.
  useEffect(() => {
    const t = setInterval(reload, 10000);
    return () => clearInterval(t);
  }, [reload]);

  return (
    <section className="panel" aria-label="Execução">
      <div className="panel-head">
        <h2>Execução</h2>
      </div>
      {state.status === "loading" && <LoadingState label="Carregando execução…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && (
        <>
          <ControlLine label="Controle global" c={state.data.controls.global} />
          <ControlLine label="Controle do projeto" c={state.data.controls.project} />
          {state.data.controls.global.mode === "active" &&
            state.data.controls.project.mode === "active" && (
              <p className="state-meta">Controles operacionais: ativos.</p>
            )}
          <ProjectControls
            projectId={projectId}
            mode={state.data.controls.project.mode}
            onChange={reload}
          />
          <dl className="facts">
            {(["running", "queued", "waiting", "done"] as const).map((k) => (
              <div key={k}>
                <dt>{stateLabel[k]}</dt>
                <dd>{state.data.counts[k] ?? 0}</dd>
              </div>
            ))}
            <div>
              <dt>Última sincronização</dt>
              <dd>
                {state.data.last_sync
                  ? `${state.data.last_sync.outcome === "ok" ? "ok" : "falhou"} · ${new Date(state.data.last_sync.finished_at).toLocaleString("pt-BR")}`
                  : "nunca"}
              </dd>
            </div>
          </dl>
          {state.data.items.length === 0 ? (
            <p className="state-meta">Nenhuma tarefa na fila deste projeto.</p>
          ) : (
            <ul className="item-list" aria-label="Tarefas">
              {state.data.items.map((it) => (
                <li key={it.id}>
                  <div className="item-link">
                    <span>
                      <strong>
                        {it.task_key}
                        {it.kind === "pr_fix" ? " (correção de PR)" : ""}
                      </strong>
                      {it.blocked ? <small>Bloqueio: {it.blocked}</small> : null}
                      {it.started_at ? <small>Tempo: {duration(it.seconds)}</small> : null}
                    </span>
                    <span
                      className={`badge ${it.state === "running" ? "badge--active" : it.state === "waiting" || it.blocked ? "badge--paused" : ""}`}
                    >
                      {stateLabel[it.state]}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          )}
          {state.data.executions.length > 0 && (
            <div className="panel--pad">
              <h3 className="form-title">Logs ao vivo</h3>
              <label>
                Execução
                <select
                  value={execution ?? ""}
                  onChange={(e) => setExecution(e.target.value || null)}
                >
                  <option value="">Escolha uma execução</option>
                  {state.data.executions.map((e) => (
                    <option key={e.id} value={e.id}>
                      {`Tentativa ${e.attempt} · ${e.state} · ${e.last_seq} evento(s)`}
                    </option>
                  ))}
                </select>
              </label>
              {execution && <LogStream projectId={projectId} executionId={execution} />}
              {execution && <AgentMonitor projectId={projectId} executionId={execution} />}
              {execution && <GuidanceChat projectId={projectId} executionId={execution} />}
            </div>
          )}
        </>
      )}
    </section>
  );
}

const statusText = {
  connecting: "Conectando…",
  live: "Ao vivo",
  reconnecting: "Reconectando pelo cursor…",
} as const;

function eventLine(e: ExecutionEvent): string {
  const p = typeof e.payload === "string" ? e.payload : JSON.stringify(e.payload);
  return `#${e.seq} ${e.type} ${p}`;
}

/** Fluxo ao vivo + histórico pesquisável de uma execução. */
export function LogStream({ projectId, executionId }: { projectId: string; executionId: string }) {
  const { events, status } = useEventStream(projectId, executionId);
  const [q, setQ] = useState("");
  const [found, setFound] = useState<ExecutionEvent[] | null>(null);
  const [err, setErr] = useState(false);

  async function search(e: FormEvent) {
    e.preventDefault();
    setErr(false);
    if (!q.trim()) {
      setFound(null);
      return;
    }
    try {
      setFound((await api.executionEvents(projectId, executionId, q.trim())).items);
    } catch {
      setErr(true);
    }
  }

  const shown = found ?? events;
  return (
    <section aria-label="Logs">
      <p className="state-meta" role="status">
        {statusText[status]}
      </p>
      <form onSubmit={search} className="inline-form">
        <input
          aria-label="Buscar no histórico"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Buscar no histórico"
        />
        <button type="submit" className="secondary-action">
          Buscar
        </button>
        {found && (
          <button
            type="button"
            className="secondary-action"
            onClick={() => {
              setFound(null);
              setQ("");
            }}
          >
            Voltar ao ao vivo
          </button>
        )}
      </form>
      {err && <p className="state-meta">Não foi possível buscar no histórico.</p>}
      {shown.length === 0 ? (
        <p className="state-meta">{found ? "Nada encontrado." : "Nenhum evento ainda."}</p>
      ) : (
        <pre className="log" role="log" aria-label="Eventos">
          {shown.map(eventLine).join("\n")}
        </pre>
      )}
    </section>
  );
}
