/** Formato de erro da API (ver contracts/openapi.yaml, ErrorBody). */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly action?: string;
  readonly correlationId?: string;

  constructor(init: {
    status: number;
    code: string;
    message: string;
    action?: string;
    correlationId?: string;
  }) {
    super(init.message);
    this.name = "ApiError";
    this.status = init.status;
    this.code = init.code;
    this.action = init.action;
    this.correlationId = init.correlationId;
  }

  get isUnauthorized(): boolean {
    return this.status === 401;
  }
  get isNotFound(): boolean {
    return this.status === 404;
  }
}

export type ProjectState = "draft" | "active" | "paused" | "archived";

export interface Project {
  id: string;
  slug: string;
  name: string;
  state: ProjectState;
  priority: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface ProjectPage {
  items: Project[];
  next_cursor?: string;
}

export interface Session {
  id: string;
  admin_id: string;
  username: string;
  expires_at: string;
}

const BASE = "/api";

/** Chama a API com cookie de sessão e converte falhas em ApiError. */
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(`${BASE}${path}`, {
      credentials: "same-origin",
      ...init,
      headers: { Accept: "application/json", ...init.headers },
    });
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === "AbortError") throw cause;
    throw new ApiError({
      status: 0,
      code: "network_error",
      message: "Não foi possível contatar o servidor.",
      action: "Verifique a conexão e tente novamente.",
    });
  }
  if (response.ok) return (await response.json()) as T;

  let body: {
    error?: { code?: string; message?: string; action?: string; correlation_id?: string };
  } = {};
  try {
    body = await response.json();
  } catch {
    // corpo não-JSON (ex.: erro de proxy): cai no erro genérico abaixo
  }
  throw new ApiError({
    status: response.status,
    code: body.error?.code ?? "http_error",
    message: body.error?.message ?? `O servidor respondeu com erro ${response.status}.`,
    action: body.error?.action,
    correlationId: body.error?.correlation_id ?? response.headers.get("X-Request-ID") ?? undefined,
  });
}

export const api = {
  session: (signal?: AbortSignal) => request<Session>("/session", { signal }),
  devSignIn: () => request<Session>("/dev/session", { method: "POST" }),
  ready: (signal?: AbortSignal) => request<{ status: string }>("/readyz", { signal }),
  listProjects: (signal?: AbortSignal) => request<ProjectPage>("/projects?limit=100", { signal }),
  getProject: (id: string, signal?: AbortSignal) =>
    request<Project>(`/projects/${encodeURIComponent(id)}`, { signal }),
};
