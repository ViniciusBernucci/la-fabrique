import { useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  api,
  type CleanupPlan,
  type CleanupPolicy,
  type SetupState,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const actionLabel = { archive: "Arquivar", remove: "Remover", skip: "Manter" } as const;

/** Setup do repositório (aprovado por hash, roda só no sandbox) e limpeza das worktrees (ADR-0062). */
export function ProjectOpsPanel({ projectId }: { projectId: string }) {
  return (
    <>
      <SetupSection projectId={projectId} />
      <CleanupSection projectId={projectId} />
    </>
  );
}

function SetupSection({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.setupState(projectId, s),
    `setup:${projectId}`,
    expire,
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function act(fn: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      reload();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel" aria-label="Setup do repositório">
      <div className="panel-head">
        <h2>Setup do repositório</h2>
      </div>
      {state.status === "loading" && <LoadingState label="Carregando setup…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && (
        <SetupBody
          data={state.data}
          busy={busy}
          onApprove={(d) => act(() => api.approveSetup(projectId, d.ref, d.sha256))}
          onRevoke={(d) =>
            d.approval_id && act(() => api.revokeSetup(projectId, d.approval_id ?? ""))
          }
        />
      )}
      <FormError error={error} />
    </section>
  );
}

function SetupBody({
  data,
  busy,
  onApprove,
  onRevoke,
}: {
  data: SetupState;
  busy: boolean;
  onApprove: (d: SetupState) => void;
  onRevoke: (d: SetupState) => void;
}) {
  if (!data.found) {
    return (
      <p className="muted">
        Sem setup. Para preparar o ambiente das tentativas, versione{" "}
        <code>.lafabrique/setup.sh</code> na branch <code>{data.ref}</code>. Ele só roda dentro do
        sandbox, sem rede, e só depois da sua aprovação.
      </p>
    );
  }
  return (
    <>
      <p className="muted">
        Branch <code>{data.ref}</code> · hash <code>{data.sha256.slice(0, 12)}</code>. Roda só no
        sandbox, sem rede e sem segredos. Mudou o conteúdo, a aprovação perde a validade.
      </p>
      <h3>Conteúdo do script</h3>
      <pre className="code-block">{data.content}</pre>
      {data.approved ? (
        <div className="actions">
          <span role="status">Aprovado para este conteúdo.</span>
          <button
            type="button"
            className="secondary-action"
            disabled={busy}
            onClick={() => onRevoke(data)}
          >
            Revogar aprovação
          </button>
        </div>
      ) : (
        <div className="actions">
          <span role="status">Aguardando aprovação.</span>
          <button type="button" disabled={busy} onClick={() => onApprove(data)}>
            Aprovar este script
          </button>
        </div>
      )}
      {data.last_run && (
        <p className="state-meta">
          Última execução: {data.last_run.status} (código {data.last_run.exit_code})
        </p>
      )}
    </>
  );
}

function CleanupSection({ projectId }: { projectId: string }) {
  const { expire } = useSession();
  const { state, reload } = useResource(
    (s) => api.cleanupPlan(projectId, s),
    `cleanup:${projectId}`,
    expire,
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [done, setDone] = useState<string | null>(null);

  async function act(fn: () => Promise<unknown>) {
    setBusy(true);
    setError(null);
    setDone(null);
    try {
      await fn();
      reload();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel" aria-label="Limpeza das worktrees">
      <div className="panel-head">
        <h2>Limpeza das worktrees</h2>
      </div>
      {state.status === "loading" && <LoadingState label="Carregando limpeza…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && (
        <CleanupBody
          plan={state.data}
          busy={busy}
          onPolicy={(p) => act(() => api.setCleanupPolicy(projectId, p))}
          onApply={(hash) =>
            act(async () => {
              const r = await api.applyCleanup(projectId, hash);
              setDone(`Arquivadas: ${r.archived} · removidas: ${r.removed} · falhas: ${r.failed}`);
            })
          }
        />
      )}
      {done && <p role="status">{done}</p>}
      <FormError error={error} />
    </section>
  );
}

function CleanupBody({
  plan,
  busy,
  onPolicy,
  onApply,
}: {
  plan: CleanupPlan;
  busy: boolean;
  onPolicy: (p: CleanupPolicy) => void;
  onApply: (hash: string) => void;
}) {
  const [idle, setIdle] = useState(plan.policy.archive_idle_days);
  const [keep, setKeep] = useState(plan.policy.remove_archived_days);
  const todo = plan.items.filter((i) => i.action !== "skip").length;
  return (
    <>
      <p className="muted">
        Desligada por padrão. Arquivar preserva tudo e tira só a pasta; remover apaga o arquivo de
        trabalho já incorporado na branch base (irreversível). Nunca toca tentativa em execução.
      </p>
      <div className="form form--grid">
        <label className="check">
          <input
            type="checkbox"
            checked={plan.policy.enabled}
            disabled={busy}
            onChange={(e) =>
              onPolicy({
                enabled: e.target.checked,
                archive_idle_days: idle,
                remove_archived_days: keep,
              })
            }
          />
          Limpeza ligada
        </label>
        <label>
          Arquivar após (dias parada)
          <input
            type="number"
            min={1}
            max={365}
            value={idle}
            onChange={(e) => setIdle(Number(e.target.value))}
          />
        </label>
        <label>
          Remover arquivadas após (dias)
          <input
            type="number"
            min={1}
            max={3650}
            value={keep}
            onChange={(e) => setKeep(Number(e.target.value))}
          />
        </label>
        <div className="actions">
          <button
            type="button"
            className="secondary-action"
            disabled={busy}
            onClick={() =>
              onPolicy({
                enabled: plan.policy.enabled,
                archive_idle_days: idle,
                remove_archived_days: keep,
              })
            }
          >
            Salvar prazos
          </button>
        </div>
      </div>
      {plan.policy.enabled && (
        <>
          <h3>Simulação</h3>
          {plan.items.length === 0 ? (
            <p className="muted">Nenhuma tentativa para avaliar.</p>
          ) : (
            <ul className="item-list" aria-label="Simulação da limpeza">
              {plan.items.map((i) => (
                <li key={i.attempt_id}>
                  <strong>{actionLabel[i.action]}</strong> <code>{i.branch}</code>
                  <small className="muted"> — {i.reason}</small>
                </li>
              ))}
            </ul>
          )}
          <div className="actions">
            <button type="button" disabled={busy || todo === 0} onClick={() => onApply(plan.hash)}>
              Aplicar esta simulação ({todo})
            </button>
          </div>
        </>
      )}
    </>
  );
}
