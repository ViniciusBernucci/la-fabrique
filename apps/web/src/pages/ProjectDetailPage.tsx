import { useState } from "react";
import { Link, useParams } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type Connection, type Project, type ProjectChange } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";
import { StateBadge } from "./ProjectsPage";

const dateTime = new Intl.DateTimeFormat("pt-BR", { dateStyle: "medium", timeStyle: "short" });

export function ProjectDetailPage() {
  const { id = "" } = useParams();
  const { expire } = useSession();
  const { state, reload } = useResource(
    (signal) => api.getProject(id, signal),
    `project:${id}`,
    expire,
  );

  if (state.status === "loading") return <LoadingState label="Carregando projeto…" />;
  if (state.status === "error") {
    if (state.error.isNotFound) {
      return (
        <div className="page">
          <section className="panel">
            <EmptyState
              title="Projeto não encontrado"
              action={
                <Link className="button-link secondary-action" to="/projects">
                  Voltar para projetos
                </Link>
              }
            >
              O projeto não existe ou o endereço está incorreto.
            </EmptyState>
          </section>
        </div>
      );
    }
    return <ErrorState error={state.error} onRetry={reload} />;
  }

  return <ProjectDetail project={state.data} reload={reload} />;
}

const changeLabel: Record<ProjectChange["action"], string> = {
  created: "Rascunho criado",
  updated: "Dados atualizados",
  source_changed: "Origem de tarefas alterada",
  activated: "Projeto ativado",
  suspended: "Projeto suspenso",
  resumed: "Projeto retomado",
};

function connectionName(id: string | null, connections: Connection[]): string {
  if (!id) return "—";
  return connections.find((c) => c.id === id)?.name ?? "Conexão indisponível";
}

function ProjectDetail({ project, reload }: { project: Project; reload: () => void }) {
  const { expire } = useSession();
  const connections = useResource((s) => api.listConnections(s), "connections", expire);
  const validation = useResource(
    (s) => api.validateProject(project.id, s),
    `validation:${project.id}:${project.version}`,
    expire,
  );
  const changes = useResource(
    (s) => api.projectChanges(project.id, s),
    `changes:${project.id}:${project.version}`,
    expire,
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const names = connections.state.status === "ready" ? connections.state.data.items : [];

  async function run(action: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      reload();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const actions = (
    <div className="actions">
      <StateBadge state={project.state} />
      {project.state === "draft" && (
        <Link className="button-link" to={`/projects/${project.id}/setup`}>
          Continuar cadastro
        </Link>
      )}
      {project.state === "active" && (
        <button
          type="button"
          className="secondary-action"
          disabled={busy}
          onClick={() => run(() => api.suspendProject(project.id, project.version))}
        >
          Suspender
        </button>
      )}
      {project.state === "paused" && (
        <>
          <Link className="button-link secondary-action" to={`/projects/${project.id}/setup`}>
            Editar configuração
          </Link>
          <button
            type="button"
            disabled={busy}
            onClick={() => run(() => api.resumeProject(project.id, project.version))}
          >
            Retomar
          </button>
        </>
      )}
    </div>
  );

  return (
    <div className="page">
      <nav className="breadcrumb" aria-label="Trilha">
        <Link to="/projects">Projetos</Link>
        <span aria-hidden="true">/</span>
        <span>{project.slug}</span>
      </nav>
      <PageHeader
        label="Projeto"
        title={project.name}
        lead={project.description || undefined}
        actions={actions}
      />
      <FormError error={error} />
      <section className="panel" aria-label="Resumo do projeto">
        <dl className="facts">
          <div>
            <dt>Identificador</dt>
            <dd>{project.slug}</dd>
          </div>
          <div>
            <dt>Prioridade</dt>
            <dd>{project.priority} de 4</dd>
          </div>
          <div>
            <dt>Versão</dt>
            <dd>{project.version}</dd>
          </div>
          <div>
            <dt>Criado em</dt>
            <dd>{dateTime.format(new Date(project.created_at))}</dd>
          </div>
          <div>
            <dt>Atualizado em</dt>
            <dd>{dateTime.format(new Date(project.updated_at))}</dd>
          </div>
        </dl>
      </section>
      <section className="panel panel--pad" aria-label="Configuração">
        <h2 className="form-title">Configuração</h2>
        <dl className="facts">
          <div>
            <dt>Repositório</dt>
            <dd>
              {project.repository.owner && project.repository.name
                ? `${project.repository.owner}/${project.repository.name}`
                : "Não configurado"}
            </dd>
          </div>
          <div>
            <dt>Branch base</dt>
            <dd>{project.repository.base_branch || "—"}</dd>
          </div>
          <div>
            <dt>Conexão GitHub</dt>
            <dd>{connectionName(project.repository.connection_id, names)}</dd>
          </div>
          <div>
            <dt>Origem de tarefas</dt>
            <dd>
              {project.source.kind
                ? `${project.source.kind === "jira" ? "Jira" : "ClickUp"} · ${project.source.container || "—"}`
                : "Não configurada"}
            </dd>
          </div>
          <div>
            <dt>Conexão da origem</dt>
            <dd>{connectionName(project.source.connection_id, names)}</dd>
          </div>
          <div>
            <dt>Agentes, workflows e recursos</dt>
            <dd>Nenhum associado</dd>
          </div>
        </dl>
      </section>
      {project.state !== "active" && validation.state.status === "ready" && (
        <section className="panel panel--pad" aria-label="Pendências">
          <h2 className="form-title">Pendências</h2>
          {validation.state.data.checks.filter((c) => c.status === "pending").length === 0 ? (
            <p className="muted">Nenhuma pendência de preenchimento.</p>
          ) : (
            <ul className="checklist">
              {validation.state.data.checks
                .filter((c) => c.status === "pending")
                .map((c) => (
                  <li key={c.id} className="checklist-item checklist-item--pending">
                    <span className="checklist-mark" aria-hidden="true">
                      !
                    </span>
                    <span>
                      <strong>{c.label}</strong>
                      <small>{c.action}</small>
                    </span>
                  </li>
                ))}
            </ul>
          )}
        </section>
      )}
      <section className="panel panel--pad" aria-label="Histórico">
        <h2 className="form-title">Histórico</h2>
        {changes.state.status === "loading" && <LoadingState label="Carregando histórico…" />}
        {changes.state.status === "error" && (
          <ErrorState error={changes.state.error} onRetry={changes.reload} />
        )}
        {changes.state.status === "ready" && (
          <ul className="item-list">
            {changes.state.data.items.map((c) => (
              <li key={c.id}>
                <span className="row">
                  <span>{changeLabel[c.action]}</span>
                  <small>{dateTime.format(new Date(c.created_at))}</small>
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
