import { useEffect, useRef, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import {
  type ApiError,
  api,
  type Connection,
  type OAuthProvider,
  type OAuthStarted,
  type OAuthStatus,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const providerLabel: Record<OAuthProvider["kind"], string> = {
  github: "GitHub",
  clickup: "ClickUp",
  jira: "Jira",
};

const defaultName: Record<OAuthProvider["kind"], string> = {
  github: "GitHub",
  clickup: "ClickUp",
  jira: "Jira",
};

/**
 * Conexão por OAuth: nenhum token passa pela tela. GitHub usa o código de
 * dispositivo (autoriza no navegador); ClickUp e Jira redirecionam ao provedor.
 */
const requiredScopes = ["repo", "read:org", "workflow"];

export function OAuthPanel({
  onConnected,
  connections,
  onRecheck,
}: {
  onConnected: () => void;
  connections: Connection[];
  onRecheck: () => Promise<void>;
}) {
  const { expire } = useSession();
  const providers = useResource((s) => api.listOAuthProviders(s), "oauth-providers", expire);
  const [name, setName] = useState<Partial<Record<OAuthProvider["kind"], string>>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [device, setDevice] = useState<{ kind: string; started: OAuthStarted } | null>(null);
  const [result, setResult] = useState<OAuthStatus | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const connected = useRef(onConnected);
  connected.current = onConnected;

  // Retorno do redirecionamento (?oauth=ok|failed) — mostra o resultado e limpa a URL.
  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    const outcome = q.get("oauth");
    if (!outcome) return;
    setResult(
      outcome === "ok"
        ? { status: "done" }
        : { status: "failed", error: q.get("reason") ?? "A conexão não foi concluída." },
    );
    window.history.replaceState(null, "", window.location.pathname);
    if (outcome === "ok") connected.current();
  }, []);

  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  function poll(flowId: string, seconds: number) {
    timer.current = setTimeout(async () => {
      try {
        const st = await api.oauthFlow(flowId);
        if (st.status === "pending") return poll(flowId, seconds);
        setDevice(null);
        setResult(st);
        if (st.status === "done") connected.current();
      } catch (e) {
        setDevice(null);
        setError(toApiError(e));
      }
    }, Math.max(seconds, 1) * 1000);
  }

  const accounts = connections.filter((c) => c.kind === "github" && c.state === "active");

  async function recheck() {
    setBusy(true);
    setError(null);
    try {
      await onRecheck();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  async function start(kind: OAuthProvider["kind"]) {
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const started = await api.startOAuth(kind, (name[kind] ?? "").trim() || defaultName[kind]);
      if (started.flow === "redirect" && started.authorize_url) {
        window.location.assign(started.authorize_url);
        return;
      }
      setDevice({ kind, started });
      poll(started.flow_id, started.interval ?? 5);
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Conectar com OAuth">
      <h2>GitHub</h2>
      <p className="muted">
        A fábrica age no GitHub com a conta do seu GitHub CLI, como o Orca. Cada projeto escolhe
        qual conta usar na sua configuração.
      </p>
      {accounts.length === 0 ? (
        <div className="callout" role="status">
          <strong>Não conectado</strong>
          <p>
            <strong>Pelo navegador (local ou VPS):</strong> clique em "Conectar com GitHub" abaixo.
            Você recebe um código e autoriza no github.com com a sua conta; nada é digitado aqui.
            Exige o app OAuth do GitHub no <code>.env</code> (uma vez, veja a dica abaixo).
          </p>
          <p>
            <strong>
              Pelo GitHub CLI (só na máquina com o <code>gh</code> logado):
            </strong>{" "}
            no terminal, <code>make connect-github</code>, e depois "Verificar de novo".
          </p>
        </div>
      ) : (
        <ul className="item-list">
          {accounts.map((c) => {
            const missing = requiredScopes.filter((r) => !c.scopes.split(/[\s,]+/).includes(r));
            return (
              <li key={c.id} className="row">
                <span>
                  <strong>{c.target || c.name}</strong>{" "}
                  <span className="badge badge--active">Conectado</span>
                  <small>
                    {c.name} · escopos: {c.scopes || "não informados"}
                  </small>
                  {missing.length > 0 && (
                    <small className="form-error">
                      Faltam escopos: {missing.join(", ")}. Rode{" "}
                      <code>gh auth refresh -s {missing.join(",")}</code> e depois{" "}
                      <code>make connect-github</code>.
                    </small>
                  )}
                </span>
              </li>
            );
          })}
        </ul>
      )}
      <div className="actions">
        <button type="button" className="secondary-action" disabled={busy} onClick={recheck}>
          Verificar de novo
        </button>
      </div>
      {providers.state.status === "ready" &&
        providers.state.data.items
          .filter((p) => p.available || p.kind === "github")
          .map((p) => (
            <div key={p.kind} className="form form--inline">
              <label>
                Nome da conexão ({providerLabel[p.kind]})
                <input
                  value={name[p.kind] ?? ""}
                  maxLength={80}
                  placeholder={defaultName[p.kind]}
                  onChange={(e) => setName({ ...name, [p.kind]: e.target.value })}
                />
              </label>
              <button
                type="button"
                disabled={busy || !p.available || device !== null}
                onClick={() => start(p.kind)}
              >
                Conectar com {providerLabel[p.kind]}
              </button>
              {!p.available && <small className="muted">{p.hint}</small>}
            </div>
          ))}
      {device && (
        <div className="callout" role="status">
          <p>
            Abra{" "}
            <a href={device.started.verification_uri} target="_blank" rel="noreferrer">
              {device.started.verification_uri}
            </a>{" "}
            e informe o código:
          </p>
          <p>
            <strong className="mono">{device.started.user_code}</strong>
          </p>
          <small className="muted">Aguardando a autorização…</small>
        </div>
      )}
      {result?.status === "done" && (
        <p role="status">Conta conectada. A credencial está no cofre.</p>
      )}
      {result?.status === "failed" && <p className="form-error">{result.error}</p>}
      <FormError error={error} />
    </section>
  );
}
