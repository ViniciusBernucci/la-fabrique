import { type FormEvent, useEffect, useRef, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { type ApiError, api, type CLILogin, type CLIProvider, type Connection } from "../lib/api";

const providers: { id: CLIProvider; label: string; defaultName: string; hint: string }[] = [
  {
    id: "claude",
    label: "Claude",
    defaultName: "Claude",
    hint: "Usa a sua assinatura do Claude (claude.ai).",
  },
  {
    id: "codex",
    label: "Codex",
    defaultName: "Codex",
    hint: "Usa a sua assinatura do ChatGPT (Codex).",
  },
];

const stepText: Partial<Record<CLILogin["state"], string>> = {
  pending: "Na fila para começar…",
  preparing: "Preparando o ambiente de login (na primeira vez pode levar alguns minutos)…",
  verifying: "Confirmando o login…",
};

const POLL_MS = 2000;
// Sem resposta do worker por tanto tempo, a tela avisa onde olhar.
const WORKER_HINT_MS = 20_000;

/**
 * Conexão das assinaturas de IA (Claude e Codex) por login oficial: a pessoa autoriza na própria conta
 * e a fábrica nunca vê o token (ADR-0065). O login roda num container do worker.
 */
export function CLILoginPanel({
  connections,
  onConnected,
}: {
  connections: Connection[];
  onConnected: () => void;
}) {
  const [names, setNames] = useState<Partial<Record<CLIProvider, string>>>({});
  const [login, setLogin] = useState<CLILogin | null>(null);
  const [startedAt, setStartedAt] = useState(0);
  const [code, setCode] = useState("");
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const connected = useRef(onConnected);
  connected.current = onConnected;

  const open =
    login !== null && !["done", "failed", "cancelled"].includes(login.state) ? login : null;

  const openId = open?.id ?? null;

  // Acompanha o login aberto até ele terminar (cada resposta renova a tela).
  useEffect(() => {
    if (!openId) return;
    const timer = setInterval(async () => {
      try {
        const next = await api.cliLogin(openId);
        setLogin(next);
        if (next.state === "done") connected.current();
      } catch (e) {
        setError(toApiError(e));
      }
    }, POLL_MS);
    return () => clearInterval(timer);
  }, [openId]);

  async function start(provider: CLIProvider, defaultName: string) {
    setBusy(true);
    setError(null);
    setSent(false);
    setCode("");
    try {
      const name = (names[provider] ?? "").trim() || defaultName;
      setLogin(await api.startCLILogin(provider, name));
      setStartedAt(Date.now());
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!login) return;
    setBusy(true);
    setError(null);
    try {
      setLogin(await api.submitCLILoginCode(login.id, code.trim()));
      setSent(true);
      setCode(""); // o código sai da memória da tela assim que é enviado
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  async function cancel() {
    if (!login) return;
    setBusy(true);
    try {
      setLogin(await api.cancelCLILogin(login.id));
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const accounts = connections.filter(
    (c) =>
      c.kind === "cli" && c.state === "active" && (c.target === "claude" || c.target === "codex"),
  );
  const waitingForWorker = open?.state === "pending" && Date.now() - startedAt > WORKER_HINT_MS;

  return (
    <section className="panel panel--pad" aria-label="Assinaturas de IA">
      <h2>Assinaturas de IA</h2>
      <p className="muted">
        Conecte a sua assinatura para os agentes usarem. Você autoriza na sua própria conta, no site
        oficial; a fábrica não recebe o seu token. A credencial fica num volume isolado da conta.
      </p>
      {accounts.length > 0 && (
        <ul className="item-list">
          {accounts.map((c) => (
            <li key={c.id} className="row">
              <span>
                <strong>{c.name}</strong> <span className="badge badge--active">Conectado</span>
                <small>{c.target === "claude" ? "Claude" : "Codex"} · assinatura</small>
              </span>
            </li>
          ))}
        </ul>
      )}
      {providers.map((p) => (
        <div key={p.id} className="form form--inline">
          <label>
            Nome da conexão ({p.label})
            <input
              value={names[p.id] ?? ""}
              maxLength={80}
              placeholder={p.defaultName}
              disabled={open !== null}
              onChange={(e) => setNames({ ...names, [p.id]: e.target.value })}
            />
            <small className="muted">{p.hint}</small>
          </label>
          <button
            type="button"
            disabled={busy || open !== null}
            onClick={() => start(p.id, p.defaultName)}
          >
            Conectar com {p.label}
          </button>
        </div>
      ))}
      <p className="muted">Antigravity ainda não tem login pelo painel.</p>

      {open && (
        <div className="callout" role="status" aria-label="Login em andamento">
          <strong>
            Conectando {open.provider === "claude" ? "Claude" : "Codex"} — {open.name}
          </strong>
          {stepText[open.state] && <p>{stepText[open.state]}</p>}
          {waitingForWorker && (
            <p className="form-error">
              O worker ainda não pegou este login. Confira se ele está rodando com{" "}
              <code>CLI_LOGIN=true</code> e acesso ao Docker (ADR-0065).
            </p>
          )}
          {open.state === "awaiting_user" && open.provider === "codex" && (
            <>
              <p>
                1. Abra{" "}
                <a href={open.verification_uri} target="_blank" rel="noreferrer noopener">
                  {open.verification_uri}
                </a>{" "}
                e entre na sua conta.
              </p>
              <p>2. Informe este código:</p>
              <p>
                <strong className="mono">{open.user_code}</strong>
              </p>
            </>
          )}
          {open.state === "awaiting_user" && open.provider === "claude" && (
            <>
              <p>
                1.{" "}
                <a href={open.verification_uri} target="_blank" rel="noreferrer noopener">
                  Abrir o login do Claude
                </a>{" "}
                e autorizar com a sua conta.
              </p>
              <p>2. O site vai mostrar um código. Copie-o e cole aqui:</p>
              <form className="inline-form" onSubmit={submit}>
                <input
                  aria-label="Código do Claude"
                  autoComplete="off"
                  value={code}
                  placeholder="Cole o código aqui"
                  onChange={(e) => setCode(e.target.value)}
                />
                <button type="submit" disabled={busy || code.trim() === ""}>
                  Enviar código
                </button>
              </form>
              {sent && <small className="muted">Código enviado. Confirmando…</small>}
            </>
          )}
          {open.expires_in > 0 && open.state === "awaiting_user" && (
            <small className="muted">
              Você tem {Math.max(1, Math.round(open.expires_in / 60))} min para autorizar.
            </small>
          )}
          <div className="actions">
            <button type="button" className="secondary-action" disabled={busy} onClick={cancel}>
              Cancelar
            </button>
          </div>
        </div>
      )}
      {login?.state === "done" && (
        <p role="status">
          Conta conectada. O login ficou só no volume da conta; o cofre guarda apenas um marcador.
        </p>
      )}
      {(login?.state === "failed" || login?.state === "cancelled") && (
        <p className="form-error" role="alert">
          {login.error || "O login não foi concluído."}
        </p>
      )}
      <FormError error={error} />
    </section>
  );
}
