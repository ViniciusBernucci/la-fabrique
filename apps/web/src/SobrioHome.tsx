import { type CoreAgent, OrchestrationCore } from "./OrchestrationCore";

type HomeAgent = CoreAgent & { role?: string; status: string; progress: number };
type HomeNotice = { title: string; description?: string; time: string; color: string };
type Metric = { title: string; value: string; detail: string; trend: boolean };

const metrics: Metric[] = [
  { title: "Projetos ativos", value: "5", detail: "de 5", trend: false },
  { title: "Leads · 30 dias", value: "127", detail: "↑ 12%", trend: true },
  { title: "Vendas · 30 dias", value: "R$ 12.450", detail: "↑ 18%", trend: true },
  { title: "Conteúdos gerados", value: "48", detail: "↑ 33%", trend: true },
];

/** Painel inicial da proposta sóbria: decisão pendente, indicadores e núcleo em destaque. */
export function SobrioHome({
  agents,
  notifications,
  greeting,
  dateLabel,
  onNewProject,
  onNewTask,
  onPreview,
}: {
  agents: readonly HomeAgent[];
  notifications: readonly HomeNotice[];
  greeting: string;
  dateLabel: string;
  onNewProject: () => void;
  onNewTask: () => void;
  onPreview: (title: string) => void;
}) {
  const urgent = notifications.find((item) => item.color === "red") ?? notifications[0];
  const working = agents.filter((agent) => agent.status !== "Aguardando fila").length;
  return (
    <main id="home-content" className="sobrio-home" tabIndex={-1}>
      <h1 className="home-sr-only">Central de controle La fabrique</h1>
      <div className="sobrio-head">
        <div>
          <p className="sobrio-label">
            {dateLabel} · {greeting}
          </p>
          <p className="sobrio-title" aria-hidden="true">
            Central de comando
          </p>
          <p className="sobrio-lead">
            {working} agentes trabalhando.{" "}
            {urgent && <span className="sobrio-attention">1 item precisa da sua decisão.</span>}
          </p>
        </div>
        <div className="sobrio-actions">
          <button type="button" className="secondary-action" onClick={onNewProject}>
            Novo projeto
          </button>
          <button type="button" onClick={onNewTask}>
            + Nova tarefa
          </button>
        </div>
      </div>

      {urgent && (
        <section className="sobrio-decision" aria-labelledby="sobrio-decision-title">
          <span className="sobrio-label">Requer sua decisão</span>
          <div>
            <h2 id="sobrio-decision-title">{urgent.title}</h2>
            {urgent.description && <span> · {urgent.description}</span>}
          </div>
          <small>{urgent.time}</small>
          <button
            type="button"
            className="secondary-action"
            onClick={() => onPreview(urgent.title)}
          >
            Revisar agora
          </button>
        </section>
      )}

      <section className="sobrio-metrics" aria-label="Indicadores de demonstração">
        {metrics.map((metric) => (
          <div key={metric.title}>
            <span className="sobrio-label">{metric.title}</span>
            <strong>{metric.value}</strong>
            <small className={metric.trend ? "is-up" : ""}>{metric.detail}</small>
          </div>
        ))}
      </section>

      <section className="sobrio-orchestration" aria-labelledby="sobrio-orchestration-title">
        <div className="sobrio-panel-head">
          <h2 id="sobrio-orchestration-title">Orquestração</h2>
          <span className="sobrio-online">
            {agents.length} de {agents.length} online
          </span>
        </div>
        <div className="sobrio-orchestration-body">
          <div className="sobrio-core">
            <OrchestrationCore agents={agents} />
          </div>
          <ul className="sobrio-agents">
            {agents.map((agent) => (
              <li key={agent.name} className={`tone-${agent.color}`}>
                <div>
                  <i className="home-dot" />
                  <strong>
                    {agent.name}
                    {agent.role}
                  </strong>
                  <small>{agent.progress}%</small>
                </div>
                <span>{agent.status}</span>
                <meter
                  aria-label={`Atividade simulada de ${agent.name}`}
                  min={0}
                  max={100}
                  value={agent.progress}
                />
              </li>
            ))}
          </ul>
        </div>
      </section>
      <p className="home-demo-note">
        Prévia demonstrativa · Indicadores, notificações, agentes e status são dados simulados.
      </p>
    </main>
  );
}
