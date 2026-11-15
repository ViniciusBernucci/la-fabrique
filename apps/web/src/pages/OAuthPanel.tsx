import { useEffect, useRef, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import {
  type ApiError,
  api,
  type OAuthProvider,
  type OAuthStarted,
  type OAuthStatus,
} from "../lib/api";
import { useReauth } from "../lib/reauth";
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
export function OAuthPanel({ onConnected }: { onConnected: () => void }) {
  const { expire } = useSession();
  const { runSensitive } = useReauth();
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

  async function start(kind: OAuthProvider["kind"]) {
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const started = await runSensitive(() =>
        api.startOAuth(kind, (name[kind] ?? "").trim() || defaultName[kind]),
      );
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
        A fábrica age no GitHub com a conta que você conectar. Na sua máquina, rode{" "}
        <code>make connect-github</code>: ele usa (ou abre o login do) GitHub CLI e guarda o acesso
        no cofre, sem você digitar token. Se o app OAuth estiver configurado no <code>.env</code>,
        também dá para conectar daqui.
      </p>
      {providers.state.status === "ready" &&
        providers.state.data.items
          .filter((p) => p.available)
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
