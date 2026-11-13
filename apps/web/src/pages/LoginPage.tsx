import { type FormEvent, useState } from "react";
import {
  EnrollmentPanel,
  FormError,
  RecoveryCodesPanel,
  toApiError,
} from "../components/AuthParts";
import { type ApiError, api, type Enrollment, type Session } from "../lib/api";
import { useSession } from "../lib/session";

type Step =
  | { name: "credentials" }
  | { name: "totp" }
  | { name: "recovery" }
  | { name: "enroll"; enrollment: Enrollment }
  | { name: "codes"; session: Session; codes: string[] };

/** Login em duas etapas: senha → TOTP (ou código de recuperação). Senha sozinha não abre sessão. */
export function LoginPage({ notice }: { notice?: string }) {
  const { setSession } = useSession();
  const [step, setStep] = useState<Step>({ name: "credentials" });
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await action();
    } catch (e) {
      const err = toApiError(e);
      // Desafio expirado/esgotado: recomeça pela senha.
      if (err.code === "challenge_expired") setStep({ name: "credentials" });
      setError(err);
    } finally {
      setBusy(false);
    }
  }

  function credentials(event: FormEvent) {
    event.preventDefault();
    void run(async () => {
      const { status } = await api.login(username, password);
      setPassword("");
      setCode("");
      if (status === "totp_enrollment_required") {
        setStep({ name: "enroll", enrollment: await api.totpSetup() });
      } else {
        setStep({ name: "totp" });
      }
    });
  }

  function secondFactor(event: FormEvent) {
    event.preventDefault();
    void run(async () => {
      if (step.name === "enroll") {
        const { recovery_codes, ...session } = await api.totpEnable(code);
        setStep({ name: "codes", session, codes: recovery_codes });
      } else if (step.name === "recovery") {
        setSession(await api.loginRecover(code));
      } else {
        setSession(await api.loginTotp(code));
      }
    });
  }

  if (step.name === "codes") {
    const { session, codes } = step;
    return (
      <div className="gate">
        <div className="panel panel--pad">
          <RecoveryCodesPanel codes={codes} onConfirm={() => setSession(session)} />
        </div>
      </div>
    );
  }

  const heading = {
    credentials: "Entrar",
    totp: "Verificação em duas etapas",
    recovery: "Código de recuperação",
    enroll: "Ative o 2FA",
  }[step.name];

  return (
    <div className="gate">
      <div className="panel panel--pad">
        <form className="form" onSubmit={step.name === "credentials" ? credentials : secondFactor}>
          <span className="page-label">La fabrique</span>
          <h1 className="form-title">{heading}</h1>
          {notice && <p className="muted">{notice}</p>}

          {step.name === "credentials" && (
            <>
              <label>
                Usuário
                <input
                  autoComplete="username"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                />
              </label>
              <label>
                Senha
                <input
                  type="password"
                  autoComplete="current-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </label>
            </>
          )}

          {step.name === "enroll" && (
            <>
              <p className="muted">
                Primeiro acesso: o segundo fator é obrigatório. Cadastre o autenticador e informe o
                primeiro código.
              </p>
              <EnrollmentPanel enrollment={step.enrollment} />
            </>
          )}

          {(step.name === "totp" || step.name === "enroll") && (
            <label>
              Código do autenticador (6 dígitos)
              <input
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={7}
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
              />
            </label>
          )}

          {step.name === "recovery" && (
            <label>
              Código de recuperação
              <input
                autoComplete="off"
                placeholder="XXXX-XXXX-XXXX-XXXX"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                required
              />
            </label>
          )}

          <FormError error={error} />
          <div className="actions">
            <button type="submit" disabled={busy}>
              {step.name === "credentials" ? "Continuar" : "Verificar"}
            </button>
            {step.name === "totp" && (
              <button
                type="button"
                className="secondary-action"
                onClick={() => {
                  setCode("");
                  setError(null);
                  setStep({ name: "recovery" });
                }}
              >
                Usar código de recuperação
              </button>
            )}
            {step.name !== "credentials" && (
              <button
                type="button"
                className="secondary-action"
                onClick={() => {
                  setCode("");
                  setError(null);
                  setStep({ name: "credentials" });
                }}
              >
                Voltar
              </button>
            )}
          </div>
        </form>
      </div>
    </div>
  );
}
