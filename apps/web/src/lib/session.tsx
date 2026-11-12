import { createContext, type ReactNode, useCallback, useContext, useEffect, useState } from "react";
import { ApiError, api, type Session } from "./api";

type SessionState =
  | { status: "loading" }
  | { status: "authenticated"; session: Session }
  | { status: "unauthenticated" }
  | { status: "error"; error: ApiError };

interface SessionContextValue {
  state: SessionState;
  /** Marca a sessão como perdida (ex.: um 401 numa chamada qualquer). */
  expire: () => void;
  retry: () => void;
  /** Disponível só em dev (Vite) enquanto o login real (T04-01) não existe. */
  signInDev?: () => Promise<void>;
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<SessionState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  // biome-ignore lint/correctness/useExhaustiveDependencies: attempt força nova verificação da sessão.
  useEffect(() => {
    const controller = new AbortController();
    setState({ status: "loading" });
    api
      .session(controller.signal)
      .then((session) => setState({ status: "authenticated", session }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        if (error instanceof ApiError && error.isUnauthorized)
          setState({ status: "unauthenticated" });
        else
          setState({
            status: "error",
            error:
              error instanceof ApiError
                ? error
                : new ApiError({ status: 0, code: "unexpected", message: "Falha inesperada." }),
          });
      });
    return () => controller.abort();
  }, [attempt]);

  const expire = useCallback(() => setState({ status: "unauthenticated" }), []);
  const retry = useCallback(() => setAttempt((n) => n + 1), []);
  const signInDev = import.meta.env.DEV
    ? async () => {
        const session = await api.devSignIn();
        setState({ status: "authenticated", session });
      }
    : undefined;

  return (
    <SessionContext.Provider value={{ state, expire, retry, signInDev }}>
      {children}
    </SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession fora do SessionProvider");
  return ctx;
}

export function useCurrentSession(): Session {
  const { state } = useSession();
  if (state.status !== "authenticated") throw new Error("sessão não autenticada");
  return state.session;
}
