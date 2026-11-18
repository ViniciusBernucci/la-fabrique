import { type FormEvent, useEffect, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { type ApiError, api, type ExecutionEvent, type WorkflowRun } from "../lib/api";
import { useSession } from "../lib/session";
import { useEventStream } from "../lib/useEventStream";
import { useResource } from "../lib/useResource";
import { FlowTestModal } from "./FlowTestModal";

const statusLabel: Record<WorkflowRun["status"], string> = {
  queued: "Na fila",
  running: "Em execução",
  waiting: "Dormindo (espera programada)",
  awaiting_decision: "Esperando uma decisão sua",
  completed: "Concluída",
  failed: "Falhou",
  cancelled: "Cancelada",
};

const open = (s: WorkflowRun["status"]) => !["completed", "failed", "cancelled"].includes(s);

type Payload = Record<string, unknown>;
const text = (v: unknown) => (typeof v === "string" ? v : "");

/** Uma linha de "terminal" por evento da execução: o trajeto do fluxo e o que o agente e os comandos disseram. */
export function terminalLine(e: ExecutionEvent): string {
  const p = (typeof e.payload === "object" && e.payload ? e.payload : {}) as Payload;
  switch (e.type) {
    case "run_started":
      return `▶ execução iniciada — ${text(p.title)}`;
    case "run_step": {
      const port = text(p.port) ? ` → ${text(p.port)}` : "";
      const note = text(p.note) ? ` (${text(p.note)})` : "";
      return `● ${text(p.node)} [${text(p.type)}]${port}${note}`;
    }
    case "node_event": {
      const node = `  [${text(p.node)}]`;
      switch (text(p.kind)) {
        case "command":
          return `${node} $ ${text(p.text).replace(/^\$ /, "")}`;
        case "tool":
          return `${node} ⚙ ferramenta ${text(p.tool)}`;
        case "error":
          return `${node} ✖ ${text(p.text)}`;
        default:
          return `${node} ${text(p.text)}`;
      }
    }
    case "run_diag": {
      const ok = p.ok === false ? "✖" : "·";
      const ms = typeof p.ms === "number" ? ` ${p.ms} ms` : "";
      const detail = text(p.detail) || text(p.erro) || text(p.text);
      return `${ok} diag ${text(p.node) ? `[${text(p.node)}] ` : ""}${text(p.step)}${text(p.check) ? ` ${text(p.check)}` : ""}${ms}${detail ? ` — ${detail}` : ""}`;
    }
    case "run_decision":
      return `✔ decisão em ${text(p.node)}: ${text(p.choice)}`;
    case "run_finished":
      return `■ ${text(p.status) === "completed" ? "concluída" : "falhou"} — resultado: ${text(p.outcome) || "falha"}${text(p.failure) ? ` (${text(p.failure)})` : ""}`;
    case "run_cancelled":
      return "■ execução cancelada";
    default:
      return `#${e.seq} ${e.type}`;
  }
}

/** O terminal ao vivo de uma execução: o que a fábrica fez e o que o CLI do agente imprimiu. */
export function Terminal({ projectId, executionId }: { projectId: string; executionId: string }) {
  const { events, status } = useEventStream(projectId, executionId);
  const lines = events.filter(
    (e) => (e.type.startsWith("run_") && e.type !== "run_diag") || e.type === "node_event",
  );
  return (
    <div>
      <p className="state-meta" role="status">
        {status === "live" ? "Ao vivo" : status === "connecting" ? "Conectando…" : "Reconectando…"}
      </p>
      {lines.length === 0 ? (
        <p className="state-meta">Nenhuma linha ainda.</p>
      ) : (
        <pre className="log terminal" role="log" aria-label="Terminal da execução">
          {lines.map(terminalLine).join("\n")}
        </pre>
      )}
    </div>
  );
}

function RunItem({
  projectId,
  run,
  onChange,
}: {
  projectId: string;
  run: WorkflowRun;
  onChange: () => void;
}) {
  const [showTerminal, setShowTerminal] = useState(open(run.status));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function act(fn: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      onChange();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const badge =
    run.status === "completed" && run.outcome === "success"
      ? "badge badge--active"
      : run.status === "failed" || run.outcome === "failure"
        ? "badge badge--draft"
        : "badge badge--paused";

  return (
    <li className="run-item" aria-label={`Execução ${run.title}`}>
      <div className="run-head">
        <span>
          <strong>{run.title}</strong>
          <small className="muted">
            {" "}
            · {run.issue_key ? `Issue ${run.issue_key} · ` : ""}
            {new Date(run.created_at).toLocaleString("pt-BR")}
          </small>
        </span>
        <span className={badge}>
          {run.status === "completed" && run.outcome === "failure"
            ? "Concluída com falha"
            : statusLabel[run.status]}
        </span>
      </div>
      {run.failure && <p className="form-error">{run.failure}</p>}
      {run.status === "awaiting_decision" && (
        <section className="callout" aria-label="Decisão pendente">
          <p>
            O fluxo parou em <strong>{run.pending.join(", ")}</strong> esperando a sua resposta.
          </p>
          <div className="actions">
            {run.pending.map((node) => (
              <span key={node} className="actions">
                <button
                  type="button"
                  disabled={busy}
                  onClick={() =>
                    act(() => api.decideRun(projectId, run.execution_id, node, "approved"))
                  }
                >
                  Aprovar ({node})
                </button>
                <button
                  type="button"
                  className="secondary-action"
                  disabled={busy}
                  onClick={() =>
                    act(() => api.decideRun(projectId, run.execution_id, node, "rejected"))
                  }
                >
                  Recusar ({node})
                </button>
              </span>
            ))}
          </div>
        </section>
      )}
      <div className="actions">
        <button
          type="button"
          className="secondary-action"
          onClick={() => setShowTerminal((v) => !v)}
        >
          {showTerminal ? "Ocultar terminal" : "Ver terminal"}
        </button>
        {open(run.status) && (
          <button
            type="button"
            className="secondary-action"
            disabled={busy}
            onClick={() => act(() => api.cancelRun(projectId, run.execution_id))}
          >
            Cancelar execução
          </button>
        )}
      </div>
      <FormError error={error} />
      {showTerminal && <Terminal projectId={projectId} executionId={run.execution_id} />}
    </li>
  );
}

/** Executar a fábrica: inicia o workflow publicado do projeto e acompanha cada execução ao vivo. */
export function FactoryRuns({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.listRuns(projectId, s),
    `runs:${projectId}`,
    expire,
  );
  const [title, setTitle] = useState("");
  const [task, setTask] = useState("");
  const [prdDone, setPrdDone] = useState(true);
  const [correction, setCorrection] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [testing, setTesting] = useState(false);

  const anyOpen = state.status === "ready" && state.data.items.some((r) => open(r.status));
  useEffect(() => {
    if (!anyOpen) return;
    const t = setInterval(reload, 3000);
    return () => clearInterval(t);
  }, [anyOpen, reload]);

  async function start(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.startRun(projectId, { title, task, prd_done: prdDone, correction });
      setTitle("");
      setTask("");
      reload();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Executar a fábrica">
      <div className="actions">
        <h2 className="form-title">Executar a fábrica</h2>
        <button type="button" className="secondary-action" onClick={() => setTesting(true)}>
          Testar o fluxo
        </button>
      </div>
      {testing && (
        <FlowTestModal
          projectId={projectId}
          onClose={() => {
            setTesting(false);
            reload();
          }}
        />
      )}
      <p className="muted">
        Inicia o <strong>workflow publicado</strong> deste projeto. O worker conduz o fluxo: cada
        agente roda como o CLI oficial (Claude ou Codex) num terminal isolado, com a skill e o
        comando montados pela fábrica. Se o fluxo ainda não foi publicado, publique-o no editor de
        workflows.
      </p>
      <form className="form" onSubmit={start}>
        <label>
          Título da tarefa
          <input
            value={title}
            maxLength={200}
            required
            onChange={(e) => setTitle(e.target.value)}
          />
        </label>
        <label>
          O que a fábrica deve fazer
          <textarea
            rows={4}
            value={task}
            maxLength={20000}
            onChange={(e) => setTask(e.target.value)}
          />
        </label>
        <label className="checkbox">
          <input type="checkbox" checked={prdDone} onChange={(e) => setPrdDone(e.target.checked)} />
          O PRD já está concluído (<code>prd.concluido</code>)
        </label>
        <label className="checkbox">
          <input
            type="checkbox"
            checked={correction}
            onChange={(e) => setCorrection(e.target.checked)}
          />
          É a correção de uma Pull Request (<code>fabrica.correcao</code>)
        </label>
        <FormError error={error} />
        <div className="actions">
          <button type="submit" disabled={busy || title.trim() === ""}>
            Iniciar execução
          </button>
        </div>
      </form>
      {state.status === "ready" &&
        (state.data.items.length === 0 ? (
          <p className="muted">Nenhuma execução ainda.</p>
        ) : (
          <ul className="run-list" aria-label="Execuções">
            {state.data.items.map((r) => (
              <RunItem key={r.execution_id} projectId={projectId} run={r} onChange={reload} />
            ))}
          </ul>
        ))}
    </section>
  );
}
