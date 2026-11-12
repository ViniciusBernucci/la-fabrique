import { type FormEvent, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  api,
  type Connection,
  type Project,
  type ProjectPatch,
  type RepositoryConfig,
  type SourceConfig,
  type WizardStep,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const steps: { id: WizardStep; label: string }[] = [
  { id: "basics", label: "Dados" },
  { id: "repository", label: "Repositório" },
  { id: "source", label: "Origem" },
  { id: "review", label: "Revisão" },
];

const priorityOptions = [0, 1, 2, 3, 4];

/** Assistente de cadastro: cada etapa é persistida; o rascunho volta na etapa em que parou. */
export function ProjectSetupPage() {
  const { id = "" } = useParams();
  const { expire } = useSession();
  const { state, reload } = useResource((s) => api.getProject(id, s), `project:${id}`, expire);
  const connections = useResource((s) => api.listConnections(s), "connections", expire);

  if (state.status === "loading" || connections.state.status === "loading") {
    return <LoadingState label="Carregando assistente…" />;
  }
  if (state.status === "error") return <ErrorState error={state.error} onRetry={reload} />;
  if (connections.state.status === "error") {
    return <ErrorState error={connections.state.error} onRetry={connections.reload} />;
  }
  return <Wizard initial={state.data} connections={connections.state.data.items} />;
}

function Wizard({ initial, connections }: { initial: Project; connections: Connection[] }) {
  const [project, setProject] = useState(initial);
  const [step, setStep] = useState<WizardStep>(initial.wizard_step);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const locked = project.state === "active";

  async function save(changes: ProjectPatch, next: WizardStep) {
    setBusy(true);
    setError(null);
    try {
      const updated = await api.updateProject(project.id, project.version, {
        ...changes,
        wizard_step: next,
      });
      setProject(updated);
      setStep(next);
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const goto = (target: WizardStep) => {
    setError(null);
    setStep(target);
  };

  return (
    <div className="page">
      <nav className="breadcrumb" aria-label="Trilha">
        <Link to="/projects">Projetos</Link>
        <span aria-hidden="true">/</span>
        <Link to={`/projects/${project.id}`}>{project.slug}</Link>
        <span aria-hidden="true">/</span>
        <span>Cadastro</span>
      </nav>
      <PageHeader
        label="Cadastro"
        title={project.name}
        lead="Preencha as etapas na ordem que preferir. Tudo é salvo ao avançar e você pode voltar depois."
      />
      <ol className="stepper" aria-label="Etapas do cadastro">
        {steps.map((s, index) => (
          <li key={s.id}>
            <button
              type="button"
              className="stepper-item"
              aria-current={s.id === step ? "step" : undefined}
              onClick={() => goto(s.id)}
            >
              <span aria-hidden="true">{index + 1}</span> {s.label}
            </button>
          </li>
        ))}
      </ol>
      {locked && step !== "review" && (
        <p className="callout callout--warn" role="status">
          O projeto está ativo: repositório e origem não podem mudar. Suspenda o projeto na tela de
          detalhe para editar.
        </p>
      )}
      <section
        className="panel panel--pad"
        aria-label={`Etapa ${steps.find((s) => s.id === step)?.label}`}
      >
        {step === "basics" && (
          <BasicsStep project={project} busy={busy} onSave={(c) => save(c, "repository")} />
        )}
        {step === "repository" && (
          <RepositoryStep
            project={project}
            connections={connections}
            busy={busy}
            locked={locked}
            onBack={() => goto("basics")}
            onSave={(repository) => save({ repository }, "source")}
          />
        )}
        {step === "source" && (
          <SourceStep
            project={project}
            connections={connections}
            busy={busy}
            locked={locked}
            onBack={() => goto("repository")}
            onSave={(source) => save({ source }, "review")}
          />
        )}
        {step === "review" && (
          <ReviewStep
            project={project}
            onGoto={goto}
            onProject={setProject}
            onBack={() => goto("source")}
          />
        )}
        {step !== "review" && <FormError error={error} />}
      </section>
    </div>
  );
}

function BasicsStep({
  project,
  busy,
  onSave,
}: {
  project: Project;
  busy: boolean;
  onSave: (changes: ProjectPatch) => void;
}) {
  const [name, setName] = useState(project.name);
  const [description, setDescription] = useState(project.description);
  const [priority, setPriority] = useState(project.priority);

  function submit(event: FormEvent) {
    event.preventDefault();
    onSave({ name, description, priority });
  }

  return (
    <form className="form form--grid" onSubmit={submit}>
      <h2>Dados</h2>
      <label>
        Nome
        <input value={name} maxLength={120} required onChange={(e) => setName(e.target.value)} />
      </label>
      <label>
        Identificador
        <input value={project.slug} readOnly aria-readonly="true" />
      </label>
      <label>
        Prioridade na fila
        <select value={priority} onChange={(e) => setPriority(Number(e.target.value))}>
          {priorityOptions.map((n) => (
            <option key={n} value={n}>
              {n}
              {n === 0 ? " (mais baixa)" : n === 4 ? " (mais alta)" : ""}
            </option>
          ))}
        </select>
      </label>
      <label>
        Descrição
        <input
          value={description}
          maxLength={500}
          onChange={(e) => setDescription(e.target.value)}
        />
      </label>
      <div className="actions">
        <button type="submit" disabled={busy}>
          Salvar e continuar
        </button>
      </div>
    </form>
  );
}

function ConnectionSelect({
  label,
  kind,
  connections,
  value,
  onChange,
  disabled,
}: {
  label: string;
  kind: Connection["kind"] | "";
  connections: Connection[];
  value: string | null;
  onChange: (id: string | null) => void;
  disabled?: boolean;
}) {
  const options = connections.filter((c) => c.kind === kind && c.state === "active");
  return (
    <label>
      {label}
      <select
        value={value ?? ""}
        disabled={disabled || kind === ""}
        onChange={(e) => onChange(e.target.value || null)}
      >
        <option value="">Selecione…</option>
        {options.map((c) => (
          <option key={c.id} value={c.id}>
            {c.name}
          </option>
        ))}
        {value && !options.some((c) => c.id === value) && (
          <option value={value}>Conexão indisponível (revogada ou de outro tipo)</option>
        )}
      </select>
      {kind !== "" && options.length === 0 && (
        <small className="muted">
          Nenhuma conexão ativa deste tipo. <Link to="/connections">Cadastre em Conexões</Link>.
        </small>
      )}
    </label>
  );
}

function RepositoryStep({
  project,
  connections,
  busy,
  locked,
  onBack,
  onSave,
}: {
  project: Project;
  connections: Connection[];
  busy: boolean;
  locked: boolean;
  onBack: () => void;
  onSave: (repository: RepositoryConfig) => void;
}) {
  const [repo, setRepo] = useState<RepositoryConfig>(project.repository);
  const set = (patch: Partial<RepositoryConfig>) => setRepo({ ...repo, ...patch });

  function submit(event: FormEvent) {
    event.preventDefault();
    onSave(repo);
  }

  return (
    <form className="form form--grid" onSubmit={submit}>
      <h2>Repositório GitHub</h2>
      <fieldset className="fieldset-plain" disabled={locked}>
        <label>
          Dono (usuário ou organização)
          <input
            value={repo.owner}
            maxLength={39}
            onChange={(e) => set({ owner: e.target.value })}
          />
        </label>
        <label>
          Repositório
          <input
            value={repo.name}
            maxLength={100}
            onChange={(e) => set({ name: e.target.value })}
          />
        </label>
        <label>
          Branch base
          <input
            value={repo.base_branch}
            maxLength={200}
            placeholder="main"
            onChange={(e) => set({ base_branch: e.target.value })}
          />
        </label>
        <ConnectionSelect
          label="Conexão GitHub"
          kind="github"
          connections={connections}
          value={repo.connection_id}
          onChange={(connection_id) => set({ connection_id })}
        />
      </fieldset>
      <div className="actions">
        <button type="button" className="secondary-action" onClick={onBack}>
          Voltar
        </button>
        <button type="submit" disabled={busy || locked}>
          Salvar e continuar
        </button>
      </div>
    </form>
  );
}

const containerHint = {
  jira: "Chave do projeto Jira (ex.: ABC)",
  clickup: "ID da lista do ClickUp",
  "": "Escolha a plataforma primeiro",
} as const;

function SourceStep({
  project,
  connections,
  busy,
  locked,
  onBack,
  onSave,
}: {
  project: Project;
  connections: Connection[];
  busy: boolean;
  locked: boolean;
  onBack: () => void;
  onSave: (source: SourceConfig) => void;
}) {
  const [source, setSource] = useState<SourceConfig>(project.source);

  function submit(event: FormEvent) {
    event.preventDefault();
    onSave(source);
  }

  return (
    <form className="form form--grid" onSubmit={submit}>
      <h2>Origem de tarefas</h2>
      <p className="muted form-note">Cada projeto usa uma única origem oficial: Jira ou ClickUp.</p>
      <fieldset className="fieldset-plain" disabled={locked}>
        <label>
          Plataforma
          <select
            value={source.kind}
            onChange={(e) =>
              setSource({
                // Trocar de plataforma descarta a conexão e o identificador da anterior.
                kind: e.target.value as SourceConfig["kind"],
                connection_id: null,
                container: "",
              })
            }
          >
            <option value="">Selecione…</option>
            <option value="jira">Jira</option>
            <option value="clickup">ClickUp</option>
          </select>
        </label>
        <ConnectionSelect
          label="Conexão"
          kind={source.kind}
          connections={connections}
          value={source.connection_id}
          onChange={(connection_id) => setSource({ ...source, connection_id })}
        />
        <label>
          Projeto ou lista
          <input
            value={source.container}
            maxLength={64}
            disabled={source.kind === ""}
            placeholder={containerHint[source.kind]}
            onChange={(e) => setSource({ ...source, container: e.target.value })}
          />
        </label>
      </fieldset>
      <div className="actions">
        <button type="button" className="secondary-action" onClick={onBack}>
          Voltar
        </button>
        <button type="submit" disabled={busy || locked}>
          Salvar e continuar
        </button>
      </div>
    </form>
  );
}

const areaStep: Record<string, WizardStep> = {
  basics: "basics",
  repository: "repository",
  source: "source",
};

function ReviewStep({
  project,
  onGoto,
  onProject,
  onBack,
}: {
  project: Project;
  onGoto: (step: WizardStep) => void;
  onProject: (p: Project) => void;
  onBack: () => void;
}) {
  const navigate = useNavigate();
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.validateProject(project.id, s),
    `validation:${project.id}:${project.version}`,
    expire,
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function activate() {
    setBusy(true);
    setError(null);
    try {
      const updated = await api.activateProject(project.id, project.version);
      onProject(updated);
      navigate(`/projects/${project.id}`);
    } catch (e) {
      setError(toApiError(e));
      reload();
    } finally {
      setBusy(false);
    }
  }

  if (state.status === "loading") return <LoadingState label="Validando…" />;
  if (state.status === "error") return <ErrorState error={state.error} onRetry={reload} />;
  const report = state.data;
  const pending = report.checks.filter((c) => c.status === "pending");
  const canActivate = report.ok && project.state === "draft";

  return (
    <div className="form">
      <h2>Revisão</h2>
      {report.ok ? (
        <p className="callout" role="status">
          Todos os itens obrigatórios estão preenchidos.
        </p>
      ) : (
        <p className="callout callout--warn" role="status">
          {pending.length} {pending.length === 1 ? "pendência impede" : "pendências impedem"} a
          ativação.
        </p>
      )}
      <ul className="checklist" aria-label="Itens de validação">
        {report.checks.map((c) => (
          <li key={c.id} className={`checklist-item checklist-item--${c.status}`}>
            <span className="checklist-mark" aria-hidden="true">
              {c.status === "ok" ? "✓" : "!"}
            </span>
            <span>
              <strong>{c.label}</strong>
              <small>{c.status === "ok" ? "Preenchido" : c.action}</small>
            </span>
            {c.status === "pending" && (
              <button
                type="button"
                className="secondary-action"
                onClick={() => onGoto(areaStep[c.area])}
              >
                Ir para a etapa
              </button>
            )}
          </li>
        ))}
      </ul>
      {report.not_verified.length > 0 && (
        <div className="callout">
          <strong>Ainda não verificado nesta versão:</strong>
          <ul>
            {report.not_verified.map((n) => (
              <li key={n}>{n}</li>
            ))}
          </ul>
        </div>
      )}
      <FormError error={error} />
      <div className="actions">
        <button type="button" className="secondary-action" onClick={onBack}>
          Voltar
        </button>
        <button type="button" disabled={!canActivate || busy} onClick={activate}>
          Ativar projeto
        </button>
        {project.state !== "draft" && (
          <small className="muted">
            Só rascunhos são ativados aqui; veja o detalhe do projeto.
          </small>
        )}
      </div>
    </div>
  );
}
