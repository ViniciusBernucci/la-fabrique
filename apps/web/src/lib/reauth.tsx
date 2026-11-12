import {
  createContext,
  type FormEvent,
  type ReactNode,
  useCallback,
  useContext,
  useRef,
  useState,
} from "react";
import { ApiError, api } from "./api";
import { useSession } from "./session";

interface ReauthContextValue {
  /**
   * Executa uma ação sensível. Se o servidor responder `reauth_required`, pede
   * senha + código 2FA e repete a ação uma vez. Cancelar propaga o erro original.
   */
  runSensitive: <T>(action: () => Promise<T>) => Promise<T>;
}

const ReauthContext = createContext<ReauthContextValue | null>(null);

export function ReauthProvider({ children }: { children: ReactNode }) {
  const { setSession } = useSession();
  const [open, setOpen] = useState(false);
  const resolver = useRef<((ok: boolean) => void) | null>(null);

  const ask = useCallback(
    () =>
      new Promise<boolean>((resolve) => {
        resolver.current = resolve;
        setOpen(true);
      }),
    [],
  );

  const close = (ok: boolean) => {
    setOpen(false);
    resolver.current?.(ok);
    resolver.current = null;
  };

  const runSensitive = useCallback(
    async <T,>(action: () => Promise<T>): Promise<T> => {
      try {
        return await action();
      } catch (error) {
        if (error instanceof ApiError && error.code === "reauth_required" && (await ask())) {
          return await action();
        }
        throw error;
      }
    },
    [ask],
  );

  return (
    <ReauthContext.Provider value={{ runSensitive }}>
      {children}
      {open && (
        <ReauthDialog
          onCancel={() => close(false)}
          onDone={(session) => {
            setSession(session);
            close(true);
          }}
        />
      )}
    </ReauthContext.Provider>
  );
}

export function useReauth(): ReauthContextValue {
  const ctx = useContext(ReauthContext);
  if (!ctx) throw new Error("useReauth fora do ReauthProvider");
  return ctx;
}

function ReauthDialog({
  onCancel,
  onDone,
}: {
  onCancel: () => void;
  onDone: (session: Awaited<ReturnType<typeof api.reauth>>) => void;
}) {
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      onDone(await api.reauth(password, code));
    } catch (e) {
      setError(
        e instanceof ApiError
          ? e
          : new ApiError({ status: 0, code: "unexpected", message: "Falha inesperada." }),
      );
      setBusy(false);
    }
  }

  return (
    <div className="dialog-backdrop">
      <div className="dialog panel" role="dialog" aria-modal="true" aria-labelledby="reauth-title">
        <form className="form" onSubmit={submit}>
          <h2 id="reauth-title">Confirme sua identidade</h2>
          <p className="muted">Esta ação é sensível. Informe a senha e o código atual do 2FA.</p>
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
          <label>
            Código do autenticador
            <input
              inputMode="numeric"
              autoComplete="one-time-code"
              pattern="[0-9 ]*"
              maxLength={7}
              value={code}
              onChange={(e) => setCode(e.target.value)}
              required
            />
          </label>
          {error && (
            <p className="form-error" role="alert">
              {error.message} {error.action}
            </p>
          )}
          <div className="actions">
            <button type="submit" disabled={busy}>
              Confirmar
            </button>
            <button type="button" className="secondary-action" onClick={onCancel}>
              Cancelar
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
