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

export type WizardStep = "basics" | "repository" | "source" | "review";

export interface RepositoryConfig {
  owner: string;
  name: string;
  base_branch: string;
  /** Referência de conexão GitHub (nunca o segredo). */
  connection_id: string | null;
}

export interface SourceConfig {
  kind: "" | "jira" | "clickup";
  connection_id: string | null;
  container: string;
}

export interface Project {
  id: string;
  slug: string;
  name: string;
  description: string;
  state: ProjectState;
  priority: number;
  wizard_step: WizardStep;
  repository: RepositoryConfig;
  source: SourceConfig;
  version: number;
  activated_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface NewProject {
  slug: string;
  name: string;
  description?: string;
  priority?: number;
}

export interface ProjectPatch {
  name?: string;
  description?: string;
  priority?: number;
  wizard_step?: WizardStep;
  repository?: RepositoryConfig;
  source?: SourceConfig;
}

export interface ValidationCheck {
  id: string;
  area: "basics" | "repository" | "source";
  label: string;
  status: "ok" | "pending";
  action?: string;
}

export interface ValidationReport {
  ok: boolean;
  checks: ValidationCheck[];
  /** O que o relatório ainda não avalia (T05-02, T05-03). */
  not_verified: string[];
}

export interface ProjectChange {
  id: number;
  action: "created" | "updated" | "source_changed" | "activated" | "suspended" | "resumed";
  detail: Record<string, unknown>;
  created_at: string;
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
  /** Vai em X-CSRF-Token nas requisições que alteram estado. */
  csrf_token: string;
  /** Ausente = ações sensíveis exigem reautenticação. */
  reauth_until?: string;
  /** Entrou com código de recuperação: o 2FA deve ser substituído. */
  totp_reset_required: boolean;
}

export type LoginStatus = "totp_required" | "totp_enrollment_required";

export interface Enrollment {
  secret: string;
  otpauth_uri: string;
}

export interface SessionInfo {
  id: string;
  created_at: string;
  expires_at: string;
  ip: string;
  user_agent: string;
  auth_method: "totp" | "recovery";
  current: boolean;
}

export interface AuthEvent {
  id: number;
  type: string;
  ip: string;
  user_agent: string;
  created_at: string;
}

export type ConnectionKind = "github" | "jira" | "clickup" | "cli" | "api" | "mcp";

export interface Connection {
  id: string;
  kind: ConnectionKind;
  name: string;
  target: string;
  scopes: string;
  state: "active" | "revoked";
  secret_version: number;
  rotated_at?: string;
  revoked_at?: string;
  last_test_at?: string;
  last_test_status: "ok" | "failed" | "not_run";
  last_test_detail: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface NewConnection {
  kind: ConnectionKind;
  name: string;
  target: string;
  scopes: string;
  secret: string;
}

const BASE = "/api";

/** Token CSRF da sessão atual (memória; o servidor o devolve em /session e no login). */
let csrfToken = "";
export function setCsrfToken(token: string): void {
  csrfToken = token;
}

/** Chama a API com cookie de sessão e converte falhas em ApiError. */
export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  let response: Response;
  try {
    const method = (init.method ?? "GET").toUpperCase();
    const unsafe = method !== "GET" && method !== "HEAD";
    response = await fetch(`${BASE}${path}`, {
      credentials: "same-origin",
      ...init,
      headers: {
        Accept: "application/json",
        ...(unsafe && csrfToken ? { "X-CSRF-Token": csrfToken } : {}),
        ...init.headers,
      },
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

function post<T>(path: string, body?: unknown, signal?: AbortSignal): Promise<T> {
  return request<T>(path, {
    method: "POST",
    signal,
    ...(body === undefined
      ? {}
      : { headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }),
  });
}

function patch<T>(path: string, body: unknown): Promise<T> {
  return request<T>(path, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export const api = {
  session: (signal?: AbortSignal) => request<Session>("/session", { signal }),
  ready: (signal?: AbortSignal) => request<{ status: string }>("/readyz", { signal }),
  listProjects: (signal?: AbortSignal) => request<ProjectPage>("/projects?limit=100", { signal }),
  getProject: (id: string, signal?: AbortSignal) =>
    request<Project>(`/projects/${encodeURIComponent(id)}`, { signal }),
  createProject: (input: NewProject) => post<Project>("/projects", input),
  /** Persiste uma etapa do assistente com controle otimista de versão. */
  updateProject: (id: string, expected_version: number, changes: ProjectPatch) =>
    patch<Project>(`/projects/${encodeURIComponent(id)}`, { expected_version, ...changes }),
  validateProject: (id: string, signal?: AbortSignal) =>
    request<ValidationReport>(`/projects/${encodeURIComponent(id)}/validation`, { signal }),
  projectChanges: (id: string, signal?: AbortSignal) =>
    request<{ items: ProjectChange[] }>(`/projects/${encodeURIComponent(id)}/changes`, { signal }),
  activateProject: (id: string, expected_version: number) =>
    post<Project>(`/projects/${encodeURIComponent(id)}/activate`, { expected_version }),
  suspendProject: (id: string, expected_version: number) =>
    post<Project>(`/projects/${encodeURIComponent(id)}/suspend`, { expected_version }),
  resumeProject: (id: string, expected_version: number) =>
    post<Project>(`/projects/${encodeURIComponent(id)}/resume`, { expected_version }),

  // Login em duas etapas.
  login: (username: string, password: string) =>
    post<{ status: LoginStatus }>("/auth/login", { username, password }),
  loginTotp: (code: string) => post<Session>("/auth/totp", { code }),
  loginRecover: (recovery_code: string) => post<Session>("/auth/recover", { recovery_code }),
  totpSetup: () => post<Enrollment>("/auth/totp/setup"),
  totpEnable: (code: string) =>
    post<Session & { recovery_codes: string[] }>("/auth/totp/enable", { code }),

  // Conta e sessões.
  logout: () => post<{ status: string }>("/auth/logout"),
  reauth: (password: string, code: string) => post<Session>("/auth/reauth", { password, code }),
  authEvents: (signal?: AbortSignal) => request<{ items: AuthEvent[] }>("/auth/events", { signal }),
  listSessions: (signal?: AbortSignal) =>
    request<{ items: SessionInfo[] }>("/sessions", { signal }),
  revokeSession: (id: string) =>
    request<{ status: string }>(`/sessions/${encodeURIComponent(id)}`, { method: "DELETE" }),
  revokeOtherSessions: () => post<{ revoked: number }>("/sessions/revoke-others"),
  replaceTotpBegin: () => post<Enrollment>("/auth/totp/replace/begin"),
  replaceTotpConfirm: (code: string) =>
    post<{ recovery_codes: string[] }>("/auth/totp/replace/confirm", { code }),

  // Cofre de conexões (o segredo só é enviado, nunca recebido).
  listConnections: (signal?: AbortSignal) =>
    request<{ items: Connection[] }>("/connections", { signal }),
  createConnection: (input: NewConnection) => post<Connection>("/connections", input),
  testConnection: (id: string) => post<Connection>(`/connections/${encodeURIComponent(id)}/test`),
  rotateConnection: (id: string, secret: string, expected_version: number) =>
    post<Connection>(`/connections/${encodeURIComponent(id)}/rotate`, { secret, expected_version }),
  revokeConnection: (id: string) =>
    post<Connection>(`/connections/${encodeURIComponent(id)}/revoke`),
};
