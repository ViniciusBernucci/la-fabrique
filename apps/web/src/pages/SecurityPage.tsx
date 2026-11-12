import { type FormEvent, useState } from "react";
import {
  EnrollmentPanel,
  FormError,
  RecoveryCodesPanel,
  toApiError,
} from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type Enrollment } from "../lib/api";
import { useReauth } from "../lib/reauth";
import { useCurrentSession, useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const when = (iso: string) => new Date(iso).toLocaleString("pt-BR");

const eventLabel: Record<string, string> = {
  admin_created: "Conta criada",
  password_ok: "Senha validada",
  login_success: "Login concluído",
  login_failed: "Login recusado",
  login_blocked: "Login bloqueado (muitas tentativas)",
  totp_failed: "Código 2FA incorreto",
  totp_replay: "Código 2FA reutilizado",
  totp_enrolled: "2FA ativado",
  totp_enroll_failed: "Falha ao ativar o 2FA",
  totp_replaced: "2FA substituído",
  totp_replace_failed: "Falha ao substituir o 2FA",
  recovery_used: "Código de recuperação usado",
  recovery_failed: "Código de recuperação recusado",
  reauth_ok: "Reautenticação concluída",
  reauth_failed: "Reautenticação recusada",
  logout: "Sessão encerrada",
  session_revoked: "Sessão revogada",
  sessions_revoked_others: "Outras sessões revogadas",
  csrf_rejected: "Requisição forjada recusada",
  connection_created: "Conexão criada",
  connection_tested: "Conexão testada",
  connection_rotated: "Segredo de conexão substituído",
  connection_revoked: "Conexão revogada",
};

export function SecurityPage() {
  const { expire, signOut } = useSession();
  const session = useCurrentSession();
  const { runSensitive } = useReauth();
  const sessions = useResource((s) => api.listSessions(s), "sessions", expire);
  const events = useResource((s) => api.authEvents(s), "auth-events", expire);
  const [actionError, setActionError] = useState<ApiError | null>(null);

  async function act(action: () => Promise<unknown>) {
    setActionError(null);
    try {
      await action();
      sessions.reload();
      events.reload();
    } catch (e) {
      setActionError(toApiError(e));
    }
  }

  return (
    <div className="page">
      <PageHeader
        label="Segurança"
        title="Segurança da conta"
        lead={`Conta ${session.username}. Sessões, segundo fator e trilha de autenticação.`}
        actions={
          <button type="button" className="secondary-action" onClick={() => void signOut()}>
            Sair
          </button>
        }
      />

      {session.totp_reset_required && (
        <p className="callout callout--warn" role="alert">
          Você entrou com um código de recuperação. Substitua o segundo fator abaixo para voltar ao
          fluxo normal.
        </p>
      )}
      <FormError error={actionError} />

      <section className="panel panel--pad" aria-label="Sessões ativas">
        <div className="panel-head">
          <h2>Sessões ativas</h2>
          <button
            type="button"
            className="secondary-action"
            onClick={() => void act(() => runSensitive(() => api.revokeOtherSessions()))}
          >
            Encerrar todas as outras
          </button>
        </div>
        {sessions.state.status === "loading" && <LoadingState label="Carregando sessões…" />}
        {sessions.state.status === "error" && (
          <ErrorState error={sessions.state.error} onRetry={sessions.reload} />
        )}
        {sessions.state.status === "ready" && (
          <ul className="item-list">
            {sessions.state.data.items.map((s) => (
              <li key={s.id} className="row">
                <span>
                  <strong>
                    {s.current ? "Esta sessão" : "Outra sessão"}{" "}
                    {s.auth_method === "recovery" && <span className="badge">recuperação</span>}
                  </strong>
                  <small>
                    {s.ip || "IP desconhecido"} · {s.user_agent || "navegador desconhecido"}
                  </small>
                  <small>
                    Iniciada {when(s.created_at)} · expira {when(s.expires_at)}
                  </small>
                </span>
                <button
                  type="button"
                  className="secondary-action"
                  onClick={() =>
                    void act(async () => {
                      if (s.current) {
                        await signOut();
                      } else {
                        await runSensitive(() => api.revokeSession(s.id));
                      }
                    })
                  }
                >
                  {s.current ? "Sair" : "Encerrar"}
                </button>
              </li>
            ))}
          </ul>
        )}
      </section>

      <ReplaceSecondFactor onDone={events.reload} />

      <section className="panel panel--pad" aria-label="Trilha de autenticação">
        <div className="panel-head">
          <h2>Trilha de autenticação</h2>
        </div>
        {events.state.status === "loading" && <LoadingState label="Carregando trilha…" />}
        {events.state.status === "error" && (
          <ErrorState error={events.state.error} onRetry={events.reload} />
        )}
        {events.state.status === "ready" && events.state.data.items.length === 0 && (
          <EmptyState title="Sem eventos" />
        )}
        {events.state.status === "ready" && events.state.data.items.length > 0 && (
          <ul className="item-list">
            {events.state.data.items.map((ev) => (
              <li key={ev.id} className="row">
                <span>
                  <strong>{eventLabel[ev.type] ?? ev.type}</strong>
                  <small>
                    {when(ev.created_at)} · {ev.ip || "sem IP (linha de comando)"}
                  </small>
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

type Replace =
  | { name: "idle" }
  | { name: "pending"; enrollment: Enrollment }
  | { name: "codes"; codes: string[] };

function ReplaceSecondFactor({ onDone }: { onDone: () => void }) {
  const { runSensitive } = useReauth();
  const [state, setState] = useState<Replace>({ name: "idle" });
  const [code, setCode] = useState("");
  const [error, setError] = useState<ApiError | null>(null);
  const [busy, setBusy] = useState(false);

  async function begin() {
    setBusy(true);
    setError(null);
    try {
      setState({ name: "pending", enrollment: await runSensitive(() => api.replaceTotpBegin()) });
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  async function confirm(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const { recovery_codes } = await runSensitive(() => api.replaceTotpConfirm(code));
      setCode("");
      setState({ name: "codes", codes: recovery_codes });
      onDone();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Segundo fator">
      <div className="panel-head">
        <h2>Segundo fator (TOTP)</h2>
      </div>
      {state.name === "idle" && (
        <div className="form">
          <p className="muted">
            Troque o autenticador (celular novo, dispositivo perdido). Os códigos de recuperação
            atuais deixam de valer, as demais sessões são encerradas e novos códigos são emitidos.
          </p>
          <FormError error={error} />
          <div className="actions">
            <button type="button" disabled={busy} onClick={() => void begin()}>
              Substituir segundo fator
            </button>
          </div>
        </div>
      )}
      {state.name === "pending" && (
        <form className="form" onSubmit={confirm}>
          <EnrollmentPanel enrollment={state.enrollment} />
          <label>
            Código do NOVO autenticador
            <input
              inputMode="numeric"
              autoComplete="one-time-code"
              maxLength={7}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
            />
          </label>
          <FormError error={error} />
          <div className="actions">
            <button type="submit" disabled={busy}>
              Confirmar substituição
            </button>
            <button
              type="button"
              className="secondary-action"
              onClick={() => {
                setState({ name: "idle" });
                setError(null);
              }}
            >
              Cancelar
            </button>
          </div>
        </form>
      )}
      {state.name === "codes" && (
        <RecoveryCodesPanel codes={state.codes} onConfirm={() => setState({ name: "idle" })} />
      )}
    </section>
  );
}
