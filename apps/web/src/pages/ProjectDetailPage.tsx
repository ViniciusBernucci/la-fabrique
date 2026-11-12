import { Link, useParams } from "react-router-dom";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { api } from "../lib/api";
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

  const project = state.data;
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
        actions={<StateBadge state={project.state} />}
      />
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
      <section className="panel" aria-label="Configuração">
        <EmptyState title="Sem configuração ainda">
          Repositório, origem de tarefas, workflows e agentes serão configurados pelos épicos
          E05–E09.
        </EmptyState>
      </section>
    </div>
  );
}
