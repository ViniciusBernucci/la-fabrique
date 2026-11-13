import { type FormEvent, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type MemoryRecord, type MemoryState } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const stateText: Record<MemoryState, string> = {
  pending: "Pendente",
  official: "Oficial",
  invalidated: "Invalidada",
};
const kindText = { fact: "Fato", decision: "Decisão", constraint: "Restrição", lesson: "Lição" };
const confidenceText = { low: "baixa", medium: "média", high: "alta" };

type Filter = MemoryState | "";
const filters: { id: Filter; label: string }[] = [
  { id: "", label: "Todas" },
  { id: "pending", label: "Pendentes" },
  { id: "official", label: "Oficiais" },
  { id: "invalidated", label: "Invalidadas" },
];

/** Memória técnica do projeto: pendente por branch, oficial após o merge (E15). */
export function MemoryPanel({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const [filter, setFilter] = useState<Filter>("");
  const [q, setQ] = useState("");
  const [applied, setApplied] = useState("");
  const { state, reload } = useResource(
    (s) => api.memory(projectId, filter, applied, s),
    `memory:${projectId}:${filter}:${applied}`,
    expire,
  );

  function search(event: FormEvent) {
    event.preventDefault();
    setApplied(q.trim());
  }

  return (
    <section className="panel" aria-label="Memória técnica">
      <div className="panel-head">
        <h2>Memória técnica</h2>
      </div>
      <div className="panel--pad">
        <div className="stepper" role="tablist" aria-label="Estado da memória">
          {filters.map((f) => (
            <button
              key={f.id || "all"}
              type="button"
              role="tab"
              aria-selected={f.id === filter}
              aria-current={f.id === filter ? "step" : undefined}
              className="stepper-item"
              onClick={() => setFilter(f.id)}
            >
              {f.label}
            </button>
          ))}
        </div>
        <form className="inline-form" onSubmit={search}>
          <input
            aria-label="Buscar na memória"
            placeholder="Buscar na memória"
            value={q}
            onChange={(e) => setQ(e.target.value)}
          />
          <button type="submit" className="secondary-action">
            Buscar
          </button>
        </form>
      </div>
      {state.status === "loading" && <LoadingState label="Carregando memória…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && state.data.items.length === 0 && (
        <EmptyState title="Nenhum registro">
          A memória nasce pendente na branch da feature e vira oficial só depois do merge.
        </EmptyState>
      )}
      {state.status === "ready" && state.data.items.length > 0 && (
        <ul className="item-list" aria-label="Registros de memória">
          {state.data.items.map((r) => (
            <MemoryItem key={r.id} projectId={projectId} record={r} onChange={reload} />
          ))}
        </ul>
      )}
    </section>
  );
}

function MemoryItem({
  projectId,
  record: r,
  onChange,
}: {
  projectId: string;
  record: MemoryRecord;
  onChange: () => void;
}) {
  const [asking, setAsking] = useState(false);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function invalidate(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await api.invalidateMemory(projectId, r.id, reason.trim());
      setAsking(false);
      onChange();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <li aria-label={r.key}>
      <div className="item-link">
        <span>
          <strong>{r.key}</strong>
          <small>{r.content}</small>
          <small>
            {kindText[r.kind]} · confiança {confidenceText[r.confidence]} · fonte: {r.source}
          </small>
          <small>
            Evidências: {r.evidence.map((e) => `${e.label} (${e.ref})`).join("; ")} · revisão{" "}
            {r.revision.slice(0, 12)}
          </small>
          {r.state === "pending" && <small>Branch: {r.branch}</small>}
          {r.state === "official" && r.promoted_revision && (
            <small>Promovida no merge {r.promoted_revision.slice(0, 12)}</small>
          )}
          {r.state === "invalidated" && <small>Motivo: {r.invalid_reason}</small>}
        </span>
        <span
          className={`badge ${r.state === "official" ? "badge--active" : r.state === "invalidated" ? "badge--archived" : ""}`}
        >
          {stateText[r.state]}
        </span>
      </div>
      {r.state !== "invalidated" && (
        <div className="panel--pad">
          {!asking ? (
            <button type="button" className="secondary-action" onClick={() => setAsking(true)}>
              Invalidar
            </button>
          ) : (
            <form className="inline-form" onSubmit={invalidate}>
              <input
                aria-label={`Motivo da invalidação de ${r.key}`}
                placeholder="Motivo da invalidação"
                value={reason}
                maxLength={500}
                onChange={(e) => setReason(e.target.value)}
              />
              <button type="submit" disabled={busy || reason.trim() === ""}>
                Confirmar invalidação
              </button>
              <button type="button" className="secondary-action" onClick={() => setAsking(false)}>
                Cancelar
              </button>
            </form>
          )}
          <FormError error={error} />
        </div>
      )}
    </li>
  );
}
