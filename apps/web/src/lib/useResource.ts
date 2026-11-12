import { useCallback, useEffect, useRef, useState } from "react";
import { ApiError } from "./api";

export type ResourceState<T> =
  | { status: "loading" }
  | { status: "ready"; data: T }
  | { status: "error"; error: ApiError };

/**
 * Carrega um recurso com cancelamento. `onUnauthorized` é chamado quando o
 * servidor responde 401, para a aplicação voltar à tela de sessão.
 */
export function useResource<T>(
  load: (signal: AbortSignal) => Promise<T>,
  key: string,
  onUnauthorized?: () => void,
): { state: ResourceState<T>; reload: () => void } {
  const [state, setState] = useState<ResourceState<T>>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const loadRef = useRef(load);
  const unauthorizedRef = useRef(onUnauthorized);
  loadRef.current = load;
  unauthorizedRef.current = onUnauthorized;

  // biome-ignore lint/correctness/useExhaustiveDependencies: key e attempt identificam o recurso; load/onUnauthorized ficam em refs.
  useEffect(() => {
    const controller = new AbortController();
    setState({ status: "loading" });
    loadRef
      .current(controller.signal)
      .then((data) => setState({ status: "ready", data }))
      .catch((error: unknown) => {
        if (controller.signal.aborted) return;
        const apiError =
          error instanceof ApiError
            ? error
            : new ApiError({ status: 0, code: "unexpected", message: "Falha inesperada." });
        if (apiError.isUnauthorized) unauthorizedRef.current?.();
        setState({ status: "error", error: apiError });
      });
    return () => controller.abort();
  }, [key, attempt]);

  const reload = useCallback(() => setAttempt((n) => n + 1), []);
  return { state, reload };
}
