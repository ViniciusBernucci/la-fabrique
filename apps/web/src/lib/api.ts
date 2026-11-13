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
  /** Rótulo (Jira) ou tag (ClickUp) que marca o item como elegível. */
  eligibility_marker: string;
  eligible_statuses: string[];
  /** Prioridade da origem (minúsculas) → faixa 0..4 da fábrica. */
  priority_map: Record<string, number>;
  access: "read" | "read_write";
}

export interface SourceStep {
  id: "credential" | "container" | "statuses" | "priorities" | "items" | "write";
  label: string;
  status: "ok" | "failed" | "skipped";
  detail: string;
}

export interface PreviewItem {
  key: string;
  title: string;
  status: string;
  priority: string;
  mapped_priority: number | null;
  subtask: boolean;
  url: string;
}

export interface SourceCheck {
  ok: boolean;
  steps: SourceStep[];
  items: PreviewItem[];
  has_more: boolean;
  checked_at: string;
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
  action:
    | "created"
    | "updated"
    | "source_changed"
    | "source_checked"
    | "activated"
    | "suspended"
    | "resumed";
  detail: Record<string, unknown>;
  created_at: string;
}

export interface IssueRecord {
  id: string;
  source: string;
  remote_id: string;
  key: string;
  title: string;
  raw_status: string;
  status_code: string;
  revision: string;
  kind: "feature" | "correction";
  local_priority?: number;
}

export interface IssueSyncRun {
  id: number;
  trigger: "daily" | "manual" | "targeted";
  scope: string;
  outcome: "ok" | "partial" | "failed";
  read: number;
  new: number;
  updated: number;
  ignored: number;
  error?: string;
  started_at: string;
  finished_at: string;
}

export interface IssuesView {
  issues: IssueRecord[];
  runs: IssueSyncRun[];
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

export interface FieldSpec {
  name: string;
  label: string;
  kind: "text" | "textarea" | "int" | "enum";
  required: boolean;
  options?: string[];
  min?: number;
  max?: number;
  hint?: string;
}

export interface NodeType {
  type: string;
  label: string;
  category: "fluxo" | "execução" | "controle";
  inputs: string[] | null;
  outputs: string[];
  multi_in: boolean;
  fields: FieldSpec[] | null;
}

export type NodeProps = Record<string, string | number>;

export interface WorkflowNode {
  id: string;
  type: string;
  x: number;
  y: number;
  props: NodeProps;
}

export interface WorkflowEdge {
  id: string;
  from: string;
  from_port: string;
  to: string;
  to_port: string;
}

export interface WorkflowGraph {
  nodes: WorkflowNode[];
  edges: WorkflowEdge[];
}

export interface WorkflowIssue {
  /** Vazio = problema do grafo inteiro. */
  node: string;
  /** Propriedade ou porta. */
  field: string;
  code: string;
  message: string;
}

export interface Workflow {
  id: string;
  name: string;
  kind: "feature" | "subflow";
  description: string;
  graph: WorkflowGraph;
  version: number;
  issues: WorkflowIssue[];
  created_at: string;
  updated_at: string;
}

export interface WorkflowSummary {
  id: string;
  name: string;
  kind: "feature" | "subflow";
  description: string;
  version: number;
  nodes: number;
  issues: number;
  updated_at: string;
}

export interface FallbackStep {
  connection_id: string;
  model: string;
}

export type AgentProvider = "claude" | "codex" | "api";

export interface Agent {
  id: string;
  name: string;
  description: string;
  instructions: string;
  provider: AgentProvider;
  connection_id: string | null;
  model: string;
  skills: string[];
  tools: string[];
  fallback: FallbackStep[];
  timeout_minutes: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export type AgentInput = Omit<Agent, "id" | "version" | "created_at" | "updated_at">;

export interface EffectiveAgent {
  agent_id: string;
  agent_name: string;
  agent_version: number;
  provider: AgentProvider;
  connection_id: string | null;
  model: string;
  instructions: string;
  prompt: string;
  skills: string[];
  tools: string[];
  fallback: FallbackStep[];
  timeout_minutes: number;
  /** Origem de cada valor: base do catálogo ou sobrescrito pelo nó. */
  sources: Record<string, "agent" | "node">;
}

export type SkillExecutor = "ai" | "script" | "deterministic" | "hybrid";

export interface SkillStep {
  type: "ai" | "script" | "deterministic";
  command?: string;
  op?: "set" | "template" | "assert";
  key?: string;
  value?: string;
  expression?: string;
}

export interface Skill {
  id: string;
  name: string;
  description: string;
  executor: SkillExecutor;
  source_kind: "manual" | "file" | "package" | "git";
  source_ref: string;
  revision: string;
  content_hash: string;
  body: string;
  manifest: {
    permissions: string[];
    resources: string[];
    commands: string[];
    steps: SkillStep[];
  };
  needs_approval: boolean;
  version: number;
  files?: { path: string; sha256: string; size: number }[];
  created_at: string;
  updated_at: string;
}

export interface SkillInput {
  name: string;
  description?: string;
  executor: SkillExecutor;
  instructions?: string;
  commands?: string[];
  steps?: SkillStep[];
}

export interface SkillImport {
  id: number;
  name: string;
  source_kind: string;
  source_ref: string;
  outcome: "imported" | "unchanged" | "replaced" | "rejected";
  reason: string;
  content_hash: string;
  created_at: string;
}

export interface ToolField {
  name: string;
  type: "string" | "int" | "bool" | "path" | "enum" | "list";
  required: boolean;
  description?: string;
  max?: number;
  min?: number;
  options?: string[];
}

export interface Tool {
  name: string;
  kind: "native" | "script" | "integration";
  description: string;
  input: ToolField[];
  output: ToolField[];
  permissions: string[];
  limits: { timeout_seconds: number; max_output_bytes: number };
  command?: string;
  operation?: string;
  requires?: string[];
  version?: number;
}

export type DecisionKind =
  | "conflict"
  | "missing_requirement"
  | "sensitive_operation"
  | "uncertain_result"
  | "pr_closed"
  | "other";

// Consumo, orçamento e provedores (T16-03). Valores em micro-USD (1e-6).
export interface BudgetStatus {
  id: string;
  scope: "global" | "agent";
  agent_id: string | null;
  period: "day" | "month";
  limit_micros: number;
  state: "active" | "disabled";
  version: number;
  spent_micros: number;
  reserved_micros: number;
  remaining_micros: number;
}

export interface UsageReport {
  From: string;
  To: string;
  api: {
    spent_micros: number;
    reserved_micros: number;
    by_source: Record<string, number>;
    by_agent: Record<string, number>;
    overrun_micros: number;
  };
  subscription: { connection_id: string; unit: string; units: number; calls: number }[];
}

export type ProviderState = "available" | "auth" | "quota" | "unavailable" | "billing";

export interface ProviderView {
  connection_id: string;
  name: string;
  kind: "cli" | "api";
  target: string;
  state: "active" | "revoked";
  health: {
    connection_id: string;
    state: ProviderState;
    source: "official" | "history";
    reason: string;
    until?: string | null;
    consecutive_failures: number;
  };
  quota: {
    used: number;
    limit: number;
    unit: string;
    resets_at?: string | null;
    source: string;
    observed_at: string;
  } | null;
}

// Operação ao vivo (T16-01).
export type ControlMode = "active" | "draining" | "suspended";

export interface ControlState {
  scope: "global" | "project";
  project_id?: string;
  mode: ControlMode;
  reason: string;
  version: number;
}

export interface QueueItemView {
  id: string;
  feature_id: string;
  task_key: string;
  kind: "task" | "pr_fix";
  priority: number;
  state: "queued" | "running" | "waiting" | "done" | "cancelled";
  wait_reason?: string;
  wait_ref?: string;
  blocked?: string;
  execution_id?: string;
  enqueued_at: string;
  started_at?: string | null;
  finished_at?: string | null;
  seconds: number;
}

export interface ProjectOverview {
  project: Project;
  controls: { global: ControlState; project: ControlState };
  counts: Record<string, number>;
  items: QueueItemView[];
  executions: {
    id: string;
    feature_id: string;
    state: string;
    attempt: number;
    last_seq: number;
  }[];
  last_sync: {
    outcome: "ok" | "failed";
    finished_at: string;
    new: number;
    changed: number;
    gone: number;
    conflicts: number;
    error?: string;
  } | null;
}

export interface ExecutionEvent {
  seq: number;
  type: string;
  payload: unknown;
  created_at: string;
}

export interface DecisionRequest {
  id: string;
  project_id: string;
  execution_id: string;
  node_id: string;
  kind: DecisionKind;
  title: string;
  problem: string;
  impact: string;
  evidence: { label: string; text: string }[];
  options: { id: string; label: string; description: string }[];
  state: "pending" | "resolved" | "stale";
  execution_version: number;
  resolution_option: string;
  resolution_note: string;
  resolved_at: string | null;
  version: number;
  created_at: string;
}

export type AuthorizationStatus = "active" | "expired" | "exhausted" | "revoked";

export interface AuthorizationView {
  id: string;
  project_id: string;
  execution_id: string;
  node_id: string;
  decision_id: string | null;
  tool: string;
  operation: string;
  resource: string;
  target: string;
  max_uses: number;
  uses: number;
  expires_at: string;
  state: "active" | "revoked";
  revoked_at: string | null;
  revoke_reason: string;
  status: AuthorizationStatus;
  remaining: number;
  in_flight: number;
  expires_in_seconds: number;
}

export interface SequenceEntry {
  at: string;
  source: "execution" | "decision" | "authorization" | "recovery" | "resume";
  type: string;
  detail: Record<string, unknown>;
}

export interface RevokeResult {
  authorization: AuthorizationView;
  in_flight: number;
  halted: boolean;
  halt_failed: boolean;
  already_revoked: boolean;
}

// Entrega, PR e memória técnica.
export type DeliveryState = "blocked" | "draft_open" | "merged" | "closed" | "cancelled";

export interface Delivery {
  id: string;
  feature_id: string;
  feature_key: string;
  feature_title: string;
  attempt: number;
  branch: string;
  revision: string;
  state: DeliveryState;
  reasons: string[];
  pr_url: string;
  merge_sha: string;
  projection: "none" | "pending" | "done" | "failed";
  projection_error: string;
  observed_at: string | null;
  created_at: string;
  comments: Record<string, number>;
}

export interface PRComment {
  id: number;
  comment_id: number;
  author: string;
  body: string;
  state: "queued" | "held" | "fixed" | "dismissed";
  queue_item_id?: string;
}

export interface DeliveryDetail {
  delivery: Delivery;
  comments: PRComment[];
  offer: {
    options: { id: "cancel" | "reopen" | "retry"; label: string; description: string }[];
    max_attempts: number;
  } | null;
  history: string[];
  observations: { outcome: string; detail: string; created_at: string }[];
}

export interface DeliverySettings {
  review_authors: string[];
  status_map: { awaiting_merge?: string; done?: string };
  version: number;
}

export interface ObserveResult {
  observation: {
    outcome: "open" | "closed" | "merged" | "unknown";
    merge_sha?: string;
    projection: string;
    projection_error?: string;
    released: unknown[];
    detail?: string;
  };
  origin_note: string;
  memory?: { promoted: number; rejected: number; duplicates: number };
  memory_error?: string;
}

export interface DiscoverResult {
  discovery: { new: number; ignored: number; held: number; item?: { id: string } | null };
  authorized: number;
}

export type MemoryState = "pending" | "official" | "invalidated";

export interface MemoryRecord {
  id: string;
  branch: string;
  key: string;
  kind: "fact" | "decision" | "constraint" | "lesson";
  content: string;
  source: string;
  evidence: { label: string; ref: string }[];
  revision: string;
  confidence: "low" | "medium" | "high";
  state: MemoryState;
  promoted_revision?: string;
  invalid_reason?: string;
}

export interface FeatureReport {
  feature_key: string;
  title: string;
  revision: string;
  doc_path: string;
  markdown: string;
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

function put<T>(path: string, body: unknown): Promise<T> {
  return request<T>(path, {
    method: "PUT",
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
  /** Prova o acesso à origem e devolve o preview dos itens elegíveis (não escreve na origem). */
  checkSource: (id: string) =>
    post<SourceCheck>(`/projects/${encodeURIComponent(id)}/source/check`),
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

  // Workflows (rascunhos).
  workflowNodeTypes: (signal?: AbortSignal) =>
    request<{ items: NodeType[] }>("/workflows/node-types", { signal }),
  listWorkflows: (signal?: AbortSignal) =>
    request<{ items: WorkflowSummary[] }>("/workflows", { signal }),
  createWorkflow: (input: { name: string; kind: "feature" | "subflow"; description?: string }) =>
    post<Workflow>("/workflows", input),
  getWorkflow: (id: string, signal?: AbortSignal) =>
    request<Workflow>(`/workflows/${encodeURIComponent(id)}`, { signal }),
  saveWorkflow: (
    id: string,
    input: { expected_version: number; name: string; description: string; graph: WorkflowGraph },
  ) => put<Workflow>(`/workflows/${encodeURIComponent(id)}`, input),
  validateWorkflowGraph: (graph: WorkflowGraph, signal?: AbortSignal) =>
    post<{ issues: WorkflowIssue[] }>("/workflows/validate", { graph }, signal),

  // Entrega, PR e memória técnica.
  deliveries: (projectId: string, signal?: AbortSignal) =>
    request<{ items: Delivery[] }>(`/projects/${encodeURIComponent(projectId)}/deliveries`, {
      signal,
    }),
  deliveryDetail: (projectId: string, id: string, signal?: AbortSignal) =>
    request<DeliveryDetail>(
      `/projects/${encodeURIComponent(projectId)}/deliveries/${encodeURIComponent(id)}`,
      { signal },
    ),
  observeDelivery: (projectId: string, id: string) =>
    post<ObserveResult>(
      `/projects/${encodeURIComponent(projectId)}/deliveries/${encodeURIComponent(id)}/observe`,
      {},
    ),
  discoverComments: (projectId: string, id: string, taskKey: string) =>
    post<DiscoverResult>(
      `/projects/${encodeURIComponent(projectId)}/deliveries/${encodeURIComponent(id)}/comments/discover`,
      { task_key: taskKey },
    ),
  deliveryAction: (projectId: string, id: string, action: string, note: string) =>
    post<{ state: DeliveryState; replayed: boolean; next_attempt?: number }>(
      `/projects/${encodeURIComponent(projectId)}/deliveries/${encodeURIComponent(id)}/actions`,
      { action, note },
    ),
  deliverySettings: (projectId: string, signal?: AbortSignal) =>
    request<DeliverySettings>(`/projects/${encodeURIComponent(projectId)}/delivery-settings`, {
      signal,
    }),
  saveDeliverySettings: (
    projectId: string,
    input: { review_authors: string[]; status_map: { awaiting_merge?: string; done?: string } },
  ) => put<DeliverySettings>(`/projects/${encodeURIComponent(projectId)}/delivery-settings`, input),
  featureReport: (projectId: string, featureId: string, signal?: AbortSignal) =>
    request<FeatureReport>(
      `/projects/${encodeURIComponent(projectId)}/features/${encodeURIComponent(featureId)}/report`,
      { signal },
    ),
  memory: (projectId: string, state: MemoryState | "", q: string, signal?: AbortSignal) =>
    request<{ items: MemoryRecord[] }>(
      `/projects/${encodeURIComponent(projectId)}/memory?state=${state}&q=${encodeURIComponent(q)}`,
      { signal },
    ),
  invalidateMemory: (projectId: string, id: string, reason: string) =>
    post<MemoryRecord>(
      `/projects/${encodeURIComponent(projectId)}/memory/${encodeURIComponent(id)}/invalidate`,
      { reason },
    ),
  setProjectControl: (projectId: string, mode: ControlMode, reason: string) =>
    put<{ control: ControlState; failed?: { reason: string }[] }>(
      `/projects/${encodeURIComponent(projectId)}/controls`,
      { mode, reason },
    ),

  // Operação ao vivo.
  projectOverview: (id: string, signal?: AbortSignal) =>
    request<ProjectOverview>(`/projects/${encodeURIComponent(id)}/overview`, { signal }),
  executionEvents: (projectId: string, executionId: string, q: string, signal?: AbortSignal) =>
    request<{ items: ExecutionEvent[]; next_cursor: number }>(
      `/projects/${encodeURIComponent(projectId)}/executions/${encodeURIComponent(executionId)}/events?limit=200${q ? `&q=${encodeURIComponent(q)}` : ""}`,
      { signal },
    ),

  // Issues da origem oficial (sync geral).
  listIssues: (projectId: string, signal?: AbortSignal) =>
    request<IssuesView>(`/projects/${encodeURIComponent(projectId)}/issues`, { signal }),
  syncIssues: (projectId: string) =>
    post<IssueSyncRun>(`/projects/${encodeURIComponent(projectId)}/issues/sync`),

  // Consumo, orçamento e provedores.
  listBudgets: (signal?: AbortSignal) => request<{ items: BudgetStatus[] }>("/budgets", { signal }),
  usageReport: (signal?: AbortSignal) => request<UsageReport>("/usage", { signal }),
  listProviders: (signal?: AbortSignal) =>
    request<{ items: ProviderView[] }>("/providers", { signal }),

  // Central de decisões.
  listDecisions: (state: "pending" | "resolved" | "stale", signal?: AbortSignal) =>
    request<{ items: DecisionRequest[] }>(`/decisions?state=${state}`, { signal }),
  resolveDecision: (id: string, option: string, note: string) =>
    post<DecisionRequest>(`/decisions/${encodeURIComponent(id)}/resolve`, { option, note }),

  refuseDecision: (id: string, note: string) =>
    post<DecisionRequest>(`/decisions/${encodeURIComponent(id)}/refuse`, { note }),
  listAuthorizations: (status: AuthorizationStatus | "all", signal?: AbortSignal) =>
    request<{ items: AuthorizationView[] }>(`/authorizations?status=${status}`, { signal }),
  revokeAuthorization: (id: string, reason: string) =>
    post<RevokeResult>(`/authorizations/${encodeURIComponent(id)}/revoke`, { reason }),
  executionSequence: (executionId: string, signal?: AbortSignal) =>
    request<{ items: SequenceEntry[] }>(`/executions/${encodeURIComponent(executionId)}/sequence`, {
      signal,
    }),

  // Catálogo de ferramentas (leitura; o cadastro de personalizadas é por API).
  listTools: (signal?: AbortSignal) => request<{ items: Tool[] }>("/tools", { signal }),

  // Catálogo de skills. O conteúdo importado é dado: nada executa na importação.
  listSkills: (signal?: AbortSignal) => request<{ items: Skill[] }>("/skills", { signal }),
  createSkill: (input: SkillInput) => post<Skill>("/skills", input),
  skillImports: (signal?: AbortSignal) =>
    request<{ items: SkillImport[] }>("/skills/imports", { signal }),
  importSkill: (file: Blob, format: "tar.gz" | "zip" | "md", replace: boolean, ref: string) =>
    request<Skill>(
      `/skills/import?format=${format}&replace=${replace}&ref=${encodeURIComponent(ref)}`,
      { method: "POST", headers: { "Content-Type": "application/octet-stream" }, body: file },
    ),

  // Catálogo de agentes.
  listAgents: (signal?: AbortSignal) => request<{ items: Agent[] }>("/agents", { signal }),
  createAgent: (input: AgentInput) => post<Agent>("/agents", input),
  updateAgent: (id: string, expected_version: number, input: AgentInput) =>
    put<Agent>(`/agents/${encodeURIComponent(id)}`, { ...input, expected_version }),
  nodeEffective: (workflowId: string, nodeId: string) =>
    request<EffectiveAgent>(
      `/workflows/${encodeURIComponent(workflowId)}/nodes/${encodeURIComponent(nodeId)}/effective`,
    ),

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
