import { type FormEvent, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import {
  type Agent,
  type AgentInput,
  type AgentProvider,
  type ApiError,
  api,
  type Connection,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const providerLabel: Record<AgentProvider, string> = {
  claude: "Claude Code (assinatura/CLI)",
  codex: "Codex CLI (assinatura/CLI)",
  api: "API (gateway)",
};

const empty: AgentInput = {
  name: "",
  description: "",
  instructions: "",
  provider: "claude",
  connection_id: null,
  model: "",
  skills: [],
  tools: [],
  fallback: [],
  timeout_minutes: 30,
};

/** Conexões que servem ao provedor: contas CLI do próprio cliente, ou conexões de API. */
export function connectionsFor(provider: AgentProvider, all: Connection[]): Connection[] {
  return all.filter(
    (c) =>
      c.state === "active" &&
      (provider === "api" ? c.kind === "api" : c.kind === "cli" && c.target === provider),
  );
}

const list = (text: string) =>
  text
    .split(",")
    .map((s) => s.trim())
    .filter(Boolean);

export function AgentsPage() {
  const { expire } = useSession();
  const agents = useResource((s) => api.listAgents(s), "agents", expire);
  const connections = useResource((s) => api.listConnections(s), "connections", expire);
  const [editing, setEditing] = useState<Agent | null>(null);

  if (agents.state.status === "loading" || connections.state.status === "loading") {
    return <LoadingState label="Carregando agentes…" />;
  }
  if (agents.state.status === "error") {
    return <ErrorState error={agents.state.error} onRetry={agents.reload} />;
  }
  if (connections.state.status === "error") {
    return <ErrorState error={connections.state.error} onRetry={connections.reload} />;
  }
  const items = agents.state.data.items;
  return (
    <div className="page">
      <PageHeader
        label="Agentes"
        title="Catálogo de agentes"
        lead="Agentes reutilizáveis entre workflows. A conta é só uma referência a uma conexão do cofre; cada nó pode sobrescrever modelo, conta e (reduzindo) ferramentas sem alterar o agente."
      />
      <AgentForm
        key={editing?.id ?? "novo"}
        agent={editing}
        connections={connections.state.data.items}
        onSaved={() => {
          setEditing(null);
          agents.reload();
        }}
        onCancel={() => setEditing(null)}
      />
      <section className="panel" aria-label="Lista de agentes">
        {items.length === 0 ? (
          <EmptyState title="Nenhum agente">Cadastre o primeiro agente acima.</EmptyState>
        ) : (
          <ul className="item-list">
            {items.map((a) => (
              <li key={a.id}>
                <div className="item-link">
                  <span>
                    <strong>{a.name}</strong>
                    <small>
                      {providerLabel[a.provider]} · {a.model || "modelo padrão"} · versão{" "}
                      {a.version}
                    </small>
                  </span>
                  <button type="button" className="secondary-action" onClick={() => setEditing(a)}>
                    Editar {a.name}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function AgentForm({
  agent,
  connections,
  onSaved,
  onCancel,
}: {
  agent: Agent | null;
  connections: Connection[];
  onSaved: () => void;
  onCancel: () => void;
}) {
  const [f, setF] = useState<AgentInput>(agent ? { ...agent } : empty);
  const [skills, setSkills] = useState((agent?.skills ?? []).join(", "));
  const [tools, setTools] = useState((agent?.tools ?? []).join(", "));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const options = connectionsFor(f.provider, connections);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    const input: AgentInput = { ...f, skills: list(skills), tools: list(tools) };
    try {
      if (agent) await api.updateAgent(agent.id, agent.version, input);
      else await api.createAgent(input);
      onSaved();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section
      className="panel panel--pad"
      aria-label={agent ? `Editar ${agent.name}` : "Novo agente"}
    >
      <form className="form form--grid" onSubmit={submit}>
        <h2>{agent ? `Editar ${agent.name}` : "Novo agente"}</h2>
        <label>
          Nome
          <input
            value={f.name}
            maxLength={64}
            required
            pattern="[a-z0-9][a-z0-9_\-]*"
            title="minúsculas, números, hífen e sublinhado"
            onChange={(e) => setF({ ...f, name: e.target.value })}
          />
        </label>
        <label>
          Provedor
          <select
            value={f.provider}
            onChange={(e) =>
              setF({ ...f, provider: e.target.value as AgentProvider, connection_id: null })
            }
          >
            {Object.entries(providerLabel).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label>
          Conta (conexão)
          <select
            value={f.connection_id ?? ""}
            onChange={(e) => setF({ ...f, connection_id: e.target.value || null })}
          >
            <option value="">Nenhuma (definida no nó)</option>
            {options.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Modelo
          <input
            value={f.model}
            maxLength={80}
            onChange={(e) => setF({ ...f, model: e.target.value })}
          />
        </label>
        <label>
          Tempo máximo (min)
          <input
            type="number"
            min={1}
            max={720}
            value={f.timeout_minutes}
            onChange={(e) => setF({ ...f, timeout_minutes: Number(e.target.value) })}
          />
        </label>
        <label>
          Skills (separadas por vírgula)
          <input value={skills} onChange={(e) => setSkills(e.target.value)} />
        </label>
        <label>
          Ferramentas (separadas por vírgula)
          <input value={tools} onChange={(e) => setTools(e.target.value)} />
        </label>
        <label>
          Descrição
          <input
            value={f.description}
            maxLength={500}
            onChange={(e) => setF({ ...f, description: e.target.value })}
          />
        </label>
        <label className="form-wide">
          Instruções
          <textarea
            rows={6}
            value={f.instructions}
            required
            onChange={(e) => setF({ ...f, instructions: e.target.value })}
          />
        </label>
        <FormError error={error} />
        <div className="actions">
          <button type="submit" disabled={busy}>
            {agent ? "Salvar alterações" : "Cadastrar agente"}
          </button>
          {agent && (
            <button type="button" className="secondary-action" onClick={onCancel}>
              Cancelar edição
            </button>
          )}
        </div>
      </form>
    </section>
  );
}
