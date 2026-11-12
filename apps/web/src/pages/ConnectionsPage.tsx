import { type FormEvent, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type Connection, type ConnectionKind } from "../lib/api";
import { useReauth } from "../lib/reauth";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const kindLabel: Record<ConnectionKind, string> = {
  github: "GitHub",
  jira: "Jira",
  clickup: "ClickUp",
  cli: "CLI",
  api: "API",
  mcp: "MCP",
};

const kindHint: Record<ConnectionKind, { target: string; secret: string }> = {
  github: { target: "Organização ou repositório (opcional)", secret: "Token com escopo mínimo" },
  jira: { target: "https://<site>.atlassian.net", secret: "email:token-de-api" },
  clickup: { target: "Workspace (opcional)", secret: "Token pessoal da API" },
  cli: { target: "Nome da CLI (ex.: gh, claude)", secret: "Credencial da CLI" },
  api: { target: "URL base (opcional)", secret: "Chave de API" },
  mcp: { target: "Servidor MCP (nome ou URL)", secret: "Token do servidor MCP" },
};

const testLabel = { ok: "Teste OK", failed: "Teste falhou", not_run: "Sem teste" } as const;

const when = (iso?: string) => (iso ? new Date(iso).toLocaleString("pt-BR") : "—");

export function ConnectionsPage() {
  const { expire } = useSession();
  const { runSensitive } = useReauth();
  const { state, reload } = useResource((s) => api.listConnections(s), "connections", expire);
  const [error, setError] = useState<ApiError | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  async function act(id: string, action: () => Promise<unknown>) {
    setBusyId(id);
    setError(null);
    try {
      await action();
      reload();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="page">
      <PageHeader
        label="Conexões"
        title="Conexões e credenciais"
        lead="Cofre de credenciais. O valor é cifrado com chave fora do banco e nunca volta para a tela; workflows e agentes usam apenas a referência."
      />
      <NewConnectionForm onCreated={reload} />
      <FormError error={error} />
      <section className="panel" aria-label="Lista de conexões">
        {state.status === "loading" && <LoadingState label="Carregando conexões…" />}
        {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
        {state.status === "ready" && state.data.items.length === 0 && (
          <EmptyState title="Nenhuma conexão cadastrada">
            Cadastre GitHub, Jira, ClickUp, CLI, API ou MCP acima.
          </EmptyState>
        )}
        {state.status === "ready" && state.data.items.length > 0 && (
          <ul className="item-list">
            {state.data.items.map((c) => (
              <ConnectionRow
                key={c.id}
                connection={c}
                busy={busyId === c.id}
                onTest={() => act(c.id, () => api.testConnection(c.id))}
                onRotate={(secret) =>
                  act(c.id, () => runSensitive(() => api.rotateConnection(c.id, secret, c.version)))
                }
                onRevoke={() => act(c.id, () => runSensitive(() => api.revokeConnection(c.id)))}
              />
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function NewConnectionForm({ onCreated }: { onCreated: () => void }) {
  const { runSensitive } = useReauth();
  const [kind, setKind] = useState<ConnectionKind>("github");
  const [name, setName] = useState("");
  const [target, setTarget] = useState("");
  const [scopes, setScopes] = useState("");
  const [secret, setSecret] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await runSensitive(() => api.createConnection({ kind, name, target, scopes, secret }));
      setName("");
      setTarget("");
      setScopes("");
      setSecret(""); // o valor sai da memória da tela assim que é enviado
      onCreated();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Nova conexão">
      <form className="form form--grid" onSubmit={submit}>
        <h2>Nova conexão</h2>
        <label>
          Tipo
          <select value={kind} onChange={(e) => setKind(e.target.value as ConnectionKind)}>
            {Object.entries(kindLabel).map(([k, label]) => (
              <option key={k} value={k}>
                {label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Nome
          <input value={name} maxLength={80} onChange={(e) => setName(e.target.value)} required />
        </label>
        <label>
          Destino
          <input
            value={target}
            maxLength={200}
            placeholder={kindHint[kind].target}
            onChange={(e) => setTarget(e.target.value)}
          />
        </label>
        <label>
          Escopos mínimos concedidos
          <input
            value={scopes}
            maxLength={200}
            placeholder="ex.: repo:status, read:org"
            onChange={(e) => setScopes(e.target.value)}
          />
        </label>
        <label>
          Segredo
          <input
            type="password"
            autoComplete="off"
            value={secret}
            placeholder={kindHint[kind].secret}
            onChange={(e) => setSecret(e.target.value)}
            required
          />
        </label>
        <FormError error={error} />
        <div className="actions">
          <button type="submit" disabled={busy}>
            Guardar no cofre
          </button>
        </div>
      </form>
    </section>
  );
}

function ConnectionRow({
  connection: c,
  busy,
  onTest,
  onRotate,
  onRevoke,
}: {
  connection: Connection;
  busy: boolean;
  onTest: () => void;
  onRotate: (secret: string) => Promise<void>;
  onRevoke: () => void;
}) {
  const [rotating, setRotating] = useState(false);
  const [secret, setSecret] = useState("");
  const [confirmRevoke, setConfirmRevoke] = useState(false);
  const revoked = c.state === "revoked";

  async function rotate(event: FormEvent) {
    event.preventDefault();
    await onRotate(secret);
    setSecret("");
    setRotating(false);
  }

  return (
    <li className="row row--stack">
      <div className="row">
        <span>
          <strong>
            {c.name} <span className="badge">{kindLabel[c.kind]}</span>{" "}
            <span className={`badge ${revoked ? "badge--archived" : "badge--active"}`}>
              {revoked ? "Revogada" : "Ativa"}
            </span>
          </strong>
          <small>
            {c.target || "sem destino"} · {c.scopes || "escopos não informados"}
          </small>
          <small>
            Referência: <code>connection:{c.id}</code> · segredo v{c.secret_version}
            {c.rotated_at && ` · rotacionado ${when(c.rotated_at)}`}
            {c.revoked_at && ` · revogada ${when(c.revoked_at)}`}
          </small>
          {!revoked && (
            <small className={`test-${c.last_test_status}`}>
              {testLabel[c.last_test_status]}
              {c.last_test_at && ` · ${when(c.last_test_at)}`}
              {c.last_test_detail && ` — ${c.last_test_detail}`}
            </small>
          )}
        </span>
        {!revoked && (
          <span className="actions">
            <button type="button" className="secondary-action" disabled={busy} onClick={onTest}>
              Testar
            </button>
            <button
              type="button"
              className="secondary-action"
              disabled={busy}
              onClick={() => setRotating((v) => !v)}
            >
              Substituir segredo
            </button>
            {confirmRevoke ? (
              <button type="button" disabled={busy} onClick={onRevoke}>
                Confirmar revogação
              </button>
            ) : (
              <button
                type="button"
                className="secondary-action"
                disabled={busy}
                onClick={() => setConfirmRevoke(true)}
              >
                Revogar
              </button>
            )}
          </span>
        )}
      </div>
      {rotating && !revoked && (
        <form className="form form--inline" onSubmit={rotate}>
          <label>
            Novo segredo para {c.name}
            <input
              type="password"
              autoComplete="off"
              value={secret}
              onChange={(e) => setSecret(e.target.value)}
              required
            />
          </label>
          <button type="submit" disabled={busy}>
            Substituir
          </button>
        </form>
      )}
    </li>
  );
}
