import { Link } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { api, type Project } from "../lib/api";
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

  return (
    <div className="page">
      <PageHeader label="Projetos" title="Projetos" lead="Os projetos atendidos pela fábrica." />
      <section className="panel" aria-label="Lista de projetos">
        {state.status === "loading" && <LoadingState label="Carregando projetos…" />}
        {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
        {state.status === "ready" && state.data.items.length === 0 && (
          <EmptyState title="Nenhum projeto cadastrado">
            O cadastro de projetos chega com o épico E05. Até lá, a lista permanece vazia.
          </EmptyState>
        )}
        {state.status === "ready" && state.data.items.length > 0 && (
          <ul className="item-list">
            {state.data.items.map((project) => (
              <li key={project.id}>
                <Link className="item-link" to={`/projects/${project.id}`}>
                  <span>
                    <strong>{project.name}</strong>
                    <small>{project.slug}</small>
                  </span>
                  <StateBadge state={project.state} />
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
