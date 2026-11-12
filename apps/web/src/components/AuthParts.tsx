import { type ReactNode, useState } from "react";
import { ApiError, type Enrollment } from "../lib/api";

export function toApiError(e: unknown): ApiError {
  return e instanceof ApiError
    ? e
    : new ApiError({ status: 0, code: "unexpected", message: "Falha inesperada." });
}

export function FormError({ error }: { error: ApiError | null }) {
  if (!error) return null;
  return (
    <p className="form-error" role="alert">
      {error.message} {error.action}
      {error.correlationId && <small> (ID {error.correlationId})</small>}
    </p>
  );
}

/** Dados para cadastrar o app autenticador (entrada manual). */
export function EnrollmentPanel({ enrollment }: { enrollment: Enrollment }) {
  return (
    <div className="callout">
      <p>
        No app autenticador (Google Authenticator, 1Password, Authy…) adicione uma conta{" "}
        <strong>por chave</strong> e informe:
      </p>
      <label>
        Chave secreta
        <input readOnly value={enrollment.secret} onFocus={(e) => e.currentTarget.select()} />
      </label>
      <label>
        URI otpauth (para importar por link)
        <input readOnly value={enrollment.otpauth_uri} onFocus={(e) => e.currentTarget.select()} />
      </label>
      <p className="muted">Tipo: baseado em tempo · 6 dígitos · 30 s · SHA-1.</p>
    </div>
  );
}

/** Códigos de recuperação: mostrados uma única vez, com confirmação explícita. */
export function RecoveryCodesPanel({
  codes,
  onConfirm,
  confirmLabel = "Continuar",
}: {
  codes: string[];
  onConfirm: () => void;
  confirmLabel?: ReactNode;
}) {
  const [saved, setSaved] = useState(false);
  return (
    <div className="form">
      <h2>Códigos de recuperação</h2>
      <p className="muted">
        Guarde estes códigos em local seguro (gerenciador de senhas). Cada um vale{" "}
        <strong>uma única vez</strong> e <strong>não será mostrado de novo</strong>. Eles permitem
        entrar se você perder o app autenticador.
      </p>
      <ul className="codes" aria-label="Códigos de recuperação">
        {codes.map((c) => (
          <li key={c}>{c}</li>
        ))}
      </ul>
      <label className="check">
        <input type="checkbox" checked={saved} onChange={(e) => setSaved(e.target.checked)} />
        Guardei os códigos em local seguro
      </label>
      <div className="actions">
        <button type="button" disabled={!saved} onClick={onConfirm}>
          {confirmLabel}
        </button>
      </div>
    </div>
  );
}
