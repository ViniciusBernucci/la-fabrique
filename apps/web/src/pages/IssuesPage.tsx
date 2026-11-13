import { useState } from "react";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { ApiError, api, type IssueSyncRun } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const outcomeLabel: Record<IssueSyncRun["outcome"], string> = {
  ok: "Concluído",
  partial: "Parcial",
  failed: "Falhou",
};

const triggerLabel: Record<IssueSyncRun["trigger"], string> = {
  daily: "Matinal",
  manual: "Manual",
  targeted: "Correções pendentes",
};

const when = (iso: string) => new Date(iso).toLocaleString("pt-BR");

/** Issues da origem oficial: o que foi importado, em que estado e o histórico de sincronização. */
export function IssuesPage() {
  const { expire } = useSession();
  const projects = useResource((s) => api.listProjects(s), "issues-projects", expire);
  const [chosen, setChosen] = useState("");
  const items = projects.state.status === "ready" ? projects.state.data.items : [];
  const projectId = chosen || items[0]?.id || "";
  const view = useResource(
    (s) => (projectId ? api.listIssues(projectId, s) : Promise.resolve({ issues: [], runs: [] })),
    `issues-${projectId}`,
    expire,
  );
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);

  async function sync() {
    setBusy(true);
    setMessage(null);
    try {
      const run = await api.syncIssues(projectId);
      setMessage({
        ok: run.outcome === "ok",
        text:
          run.outcome === "ok"
            ? `Sincronizado: ${run.new} nova(s), ${run.updated} atualizada(s), ${run.ignored} ignorada(s).`
            : `Sincronização ${outcomeLabel[run.outcome].toLowerCase()}: ${run.error ?? "falha ao ler a origem"}. O progresso gravado foi preservado.`,
      });
      view.reload();
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) expire();
      else
        setMessage({
          ok: false,
          text: e instanceof ApiError ? `${e.message} ${e.action}` : "Falha ao sincronizar.",
        });
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        label="Issues"
        title="Issues da origem oficial"
        lead="Issues novas em PRD-DOC ou CORRECAO_PR entram pelo sync matinal ou manual; as já conhecidas são atualizadas em qualquer estado."
        actions={
          <button
            type="button"
            className="primary-action"
            disabled={busy || !projectId}
            onClick={sync}
          >
            {busy ? "Sincronizando…" : "Sincronizar agora"}
          </button>
        }
      />

      {projects.state.status === "loading" && <LoadingState label="Carregando projetos…" />}
      {projects.state.status === "error" && (
        <ErrorState error={projects.state.error} onRetry={projects.reload} />
      )}
      {projects.state.status === "ready" && items.length === 0 && (
        <EmptyState title="Nenhum projeto cadastrado">
          Cadastre um projeto para acompanhar suas Issues.
        </EmptyState>
      )}
      {items.length > 0 && (
        <label className="field">
          <span>Projeto</span>
          <select value={projectId} onChange={(e) => setChosen(e.target.value)}>
            {items.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
        </label>
      )}
      {message && (
        <p className="state-meta" role={message.ok ? "status" : "alert"}>
          {message.text}
        </p>
      )}

      {projectId && (
        <>
          <section className="panel" aria-label="Issues">
            <div className="panel-head">
              <h2>Issues importadas</h2>
            </div>
            {view.state.status === "loading" && <LoadingState label="Carregando Issues…" />}
            {view.state.status === "error" && (
              <ErrorState error={view.state.error} onRetry={view.reload} />
            )}
            {view.state.status === "ready" &&
              (view.state.data.issues.length === 0 ? (
                <EmptyState title="Nenhuma Issue importada">
                  Configure o mapa de status e sincronize para importar as Issues elegíveis.
                </EmptyState>
              ) : (
                <ul className="item-list">
                  {view.state.data.issues.map((i) => (
                    <li key={i.id}>
                      <div className="item-link">
                        <span>
                          <strong>
                            {i.key} · {i.title}
                          </strong>
                          <small>
                            {i.source === "jira" ? "Jira" : "ClickUp"} · status na origem “
                            {i.raw_status}” ·{" "}
                            {i.kind === "correction" ? "correção de PR" : "feature"}
                          </small>
                        </span>
                        <span className="badge">{i.status_code || "sem mapa"}</span>
                      </div>
                    </li>
                  ))}
                </ul>
              ))}
          </section>

          <section className="panel" aria-label="Histórico de sincronização">
            <div className="panel-head">
              <h2>Histórico de sincronização</h2>
            </div>
            {view.state.status === "ready" &&
              (view.state.data.runs.length === 0 ? (
                <EmptyState title="Nenhuma sincronização ainda">
                  O primeiro sync matinal ou manual aparece aqui.
                </EmptyState>
              ) : (
                <ul className="item-list">
                  {view.state.data.runs.map((r) => (
                    <li key={r.id}>
                      <div className="item-link">
                        <span>
                          <strong>
                            {triggerLabel[r.trigger]} · {when(r.finished_at)}
                          </strong>
                          <small>
                            {r.read} lida(s) · {r.new} nova(s) · {r.updated} atualizada(s) ·{" "}
                            {r.ignored} ignorada(s)
                            {r.error ? ` · ${r.error}` : ""}
                          </small>
                        </span>
                        <span
                          className={`badge ${r.outcome === "ok" ? "badge--active" : "badge--paused"}`}
                        >
                          {outcomeLabel[r.outcome]}
                        </span>
                      </div>
                    </li>
                  ))}
                </ul>
              ))}
          </section>
        </>
      )}
    </div>
  );
}
