import type { ReactNode } from "react";
import type { ApiError } from "../lib/api";

export function LoadingState({ label = "Carregando…" }: { label?: string }) {
  return (
    <div className="state" role="status" aria-live="polite">
      <div className="spinner" aria-hidden="true" />
      <p>{label}</p>
    </div>
  );
}

export function EmptyState({
  title,
  children,
  action,
}: {
  title: string;
  children?: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="state">
      <span className="page-label">Vazio</span>
      <h2>{title}</h2>
      {children && <p>{children}</p>}
      {action}
    </div>
  );
}

export function ErrorState({ error, onRetry }: { error: ApiError; onRetry?: () => void }) {
  return (
    <div className="state state--error" role="alert">
      <span className="page-label">Erro</span>
      <h2>{error.message}</h2>
      {error.action && <p>{error.action}</p>}
      {error.correlationId && (
        <span className="state-meta">ID de correlação: {error.correlationId}</span>
      )}
      {onRetry && (
        <button type="button" className="secondary-action" onClick={onRetry}>
          Tentar novamente
        </button>
      )}
    </div>
  );
}

export function UnauthorizedState({ onSignInDev }: { onSignInDev?: () => void }) {
  return (
    <div className="state" role="alert">
      <span className="page-label">Acesso restrito</span>
      <h2>Sessão necessária</h2>
      <p>
        O servidor recusa o acesso sem uma sessão válida.{" "}
        {onSignInDev
          ? "Em desenvolvimento, use o acesso de dev abaixo."
          : "O login com senha e 2FA será disponibilizado no ticket T04-01."}
      </p>
      {onSignInDev && (
        <button type="button" onClick={onSignInDev}>
          Entrar (somente dev)
        </button>
      )}
    </div>
  );
}
