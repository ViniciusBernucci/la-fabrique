import { useState } from "react";
import { Link } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type Project } from "../lib/api";
import { useReauth } from "../lib/reauth";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

export const stateLabel: Record<Project["state"], string> = {
  draft: "Rascunho",
  active: "Ativo",
  paused: "Pausado",
  archived: "Arquivado",
};

export function StateBadge({ state }: { state: Project["state"] }) {
  return <span className={`badge badge--${state}`}>{stateLabel[state]}</span>;
}

export function ProjectsPage() {
  const { expire } = useSession();
  const { state, reload } = useResource((signal) => api.listProjects(signal), "projects", expire);
  const { runSensitive } = useReauth();
  const [target, setTarget] = useState<Project | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function remove(project: Project) {
    setBusy(true);
    setError(null);
    try {
      await runSensitive(() => api.deleteProject(project.id));
      setTarget(null);
      reload();
    } catch (e) {
      setTarget(null);
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        label="Projetos"
        title="Projetos"
        lead="Os projetos atendidos pela fábrica."
        actions={
          <Link className="button-link" to="/projects/new">
            Novo projeto
          </Link>
        }
      />
      <FormError error={error} />
      {target && (
        <div className="dialog-backdrop">
          <div
            className="dialog panel"
            role="alertdialog"
            aria-modal="true"
            aria-labelledby="delete-title"
            aria-describedby="delete-desc"
          >
            <h2 id="delete-title">Apagar “{target.name}”?</h2>
            <p id="delete-desc" className="muted">
              Atenção: esta ação é permanente e não pode ser desfeita. A configuração do projeto
              será removida. Só rascunhos ou projetos suspensos sem execuções podem ser apagados.
              Será pedida sua senha e o código do 2FA.
            </p>
            <div className="actions">
              <button
                type="button"
                className="secondary-action"
                disabled={busy}
                onClick={() => setTarget(null)}
              >
                Cancelar
              </button>
              <button
                type="button"
                className="danger-action"
                disabled={busy}
                onClick={() => remove(target)}
              >
                Apagar definitivamente
              </button>
            </div>
          </div>
        </div>
      )}
      <section className="panel" aria-label="Lista de projetos">
        {state.status === "loading" && <LoadingState label="Carregando projetos…" />}
        {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
        {state.status === "ready" && state.data.items.length === 0 && (
          <EmptyState title="Nenhum projeto cadastrado">
            Cadastre o primeiro projeto com “Novo projeto”. Ele começa em rascunho, sem agentes ou
            workflows associados.
          </EmptyState>
        )}
        {state.status === "ready" && state.data.items.length > 0 && (
          <ul className="item-list">
            {state.data.items.map((project) => (
              <li key={project.id} className="item-row">
                <Link className="item-link" to={`/projects/${project.id}`}>
                  <span>
                    <strong>{project.name}</strong>
                    <small>{project.slug}</small>
                  </span>
                  <StateBadge state={project.state} />
                </Link>
                {project.state !== "active" && (
                  <button
                    type="button"
                    className="danger-action"
                    aria-label={`Apagar projeto ${project.name}`}
                    disabled={busy}
                    onClick={() => setTarget(project)}
                  >
                    Apagar
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
