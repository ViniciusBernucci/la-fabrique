import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const kindLabel = { feature: "Fluxo de feature", subflow: "Subfluxo" } as const;

export function WorkflowsPage() {
  const { expire } = useSession();
  const navigate = useNavigate();
  const { state, reload } = useResource((s) => api.listWorkflows(s), "workflows", expire);
  const [name, setName] = useState("");
  const [kind, setKind] = useState<"feature" | "subflow">("feature");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function create(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const wf = await api.createWorkflow({ name, kind });
      navigate(`/workflows/${wf.id}`);
    } catch (e) {
      setError(toApiError(e));
      setBusy(false);
    }
  }

  return (
    <div className="page">
      <PageHeader
        label="Workflows"
        title="Workflows"
        lead="Fluxos determinísticos reutilizáveis. Rascunhos não executam: publicação e execução chegam nos próximos tickets do épico."
      />
      <section className="panel panel--pad" aria-label="Novo workflow">
        <form className="form form--grid" onSubmit={create}>
          <label>
            Nome
            <input
              value={name}
              maxLength={120}
              required
              onChange={(e) => setName(e.target.value)}
            />
          </label>
          <label>
            Tipo
            <select value={kind} onChange={(e) => setKind(e.target.value as "feature" | "subflow")}>
              <option value="feature">Fluxo de feature</option>
              <option value="subflow">Subfluxo</option>
            </select>
          </label>
          <FormError error={error} />
          <div className="actions">
            <button type="submit" disabled={busy}>
              Criar rascunho
            </button>
          </div>
        </form>
      </section>
      <section className="panel" aria-label="Lista de workflows">
        {state.status === "loading" && <LoadingState label="Carregando workflows…" />}
        {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
        {state.status === "ready" && state.data.items.length === 0 && (
          <EmptyState title="Nenhum workflow">Crie o primeiro rascunho acima.</EmptyState>
        )}
        {state.status === "ready" && state.data.items.length > 0 && (
          <ul className="item-list">
            {state.data.items.map((w) => (
              <li key={w.id}>
                <Link className="item-link" to={`/workflows/${w.id}`}>
                  <span>
                    <strong>{w.name}</strong>
                    <small>
                      {kindLabel[w.kind]} · {w.nodes} nós · versão {w.version}
                    </small>
                  </span>
                  <span className={w.issues > 0 ? "badge badge--paused" : "badge badge--active"}>
                    {w.issues > 0 ? `${w.issues} pendência(s)` : "Válido"}
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
