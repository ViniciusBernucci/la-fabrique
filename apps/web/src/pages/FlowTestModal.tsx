import { useEffect, useMemo, useRef, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import {
  type ApiError,
  api,
  type ExecutionEvent,
  type WorkflowGraph,
  type WorkflowRun,
} from "../lib/api";
import { useEventStream } from "../lib/useEventStream";
import { terminalLine } from "./FactoryRuns";

type Diagnostics = {
  report: string;
  hints: { cause: string; fix: string }[];
  failure: { node: string; reason: string };
};

type Payload = Record<string, unknown>;
const txt = (v: unknown) => (typeof v === "string" ? v : "");
const payloadOf = (e: ExecutionEvent): Payload =>
  (typeof e.payload === "object" && e.payload ? e.payload : {}) as Payload;

export type StepState = "pending" | "running" | "ok" | "waiting" | "failed" | "skipped";

export interface Step {
  id: string;
  type: string;
  label: string;
  state: StepState;
}

const typeLabel: Record<string, string> = {
  start: "Início",
  end: "Fim",
  agent: "Agente",
  skill: "Skill",
  script: "Script",
  checks: "Checks",
  git: "Git",
  docs: "Documentação",
  condition: "Condição",
  loop: "Loop",
  wait: "Espera",
  human_decision: "Decisão humana",
  subflow: "Subfluxo",
  fork: "Ramificação",
  converge: "Convergência",
};

/** Os nós do fluxo na ordem em que o fluxo os alcança (busca em largura a partir do início). */
export function orderNodes(g: WorkflowGraph): WorkflowGraph["nodes"] {
  const byId = new Map(g.nodes.map((n) => [n.id, n]));
  const start = g.nodes.find((n) => n.type === "start");
  const seen = new Set<string>();
  const out: WorkflowGraph["nodes"] = [];
  const queue = start ? [start.id] : [];
  while (queue.length > 0) {
    const id = queue.shift() as string;
    if (seen.has(id)) continue;
    seen.add(id);
    const n = byId.get(id);
    if (n) out.push(n);
    for (const e of g.edges.filter((x) => x.from === id)) queue.push(e.to);
  }
  for (const n of g.nodes) if (!seen.has(n.id)) out.push(n);
  return out;
}

/** O estado de cada nó, a partir dos eventos da execução (passos percorridos, erros e o que está rodando). */
export function stepsOf(
  g: WorkflowGraph | null,
  events: ExecutionEvent[],
  run: WorkflowRun | null,
): Step[] {
  if (!g) return [];
  const visited = new Map<string, string>(); // nó → última porta
  const started = new Set<string>();
  const failed = new Set<string>();
  const waiting = new Set<string>();
  for (const e of events) {
    const p = payloadOf(e);
    const node = txt(p.node);
    if (e.type === "run_step") {
      visited.set(node, txt(p.port));
      if (txt(p.port) === "error") failed.add(node);
      if (txt(p.note).includes("aguardando") || txt(p.note).includes("decisão humana pendente")) {
        waiting.add(node);
      }
      if (txt(p.note).includes("concluída")) waiting.delete(node);
    }
    if (e.type === "node_event") {
      if (txt(p.kind) === "start") started.add(node);
      if (txt(p.kind) === "error") failed.add(node);
    }
    if (e.type === "run_decision") waiting.delete(node);
  }
  const closed = !!run && ["completed", "failed", "cancelled"].includes(run.status);
  return orderNodes(g).map((n) => {
    const agentName = typeof n.props.agent === "string" ? n.props.agent : "";
    const skillName = typeof n.props.skill === "string" ? n.props.skill : "";
    const base = typeLabel[n.type] ?? n.type;
    const label = `${base}${agentName ? `: ${agentName}` : skillName ? `: ${skillName}` : ""}`;
    let state: StepState = "pending";
    if (failed.has(n.id)) state = "failed";
    else if (waiting.has(n.id)) state = "waiting";
    else if (visited.has(n.id)) state = "ok";
    else if (started.has(n.id)) state = closed ? "failed" : "running";
    else if (closed) state = "skipped";
    return { id: n.id, type: n.type, label, state };
  });
}

/** Onde e por que o teste falhou (vazio se não falhou). */
export function failureOf(
  steps: Step[],
  events: ExecutionEvent[],
  run: WorkflowRun | null,
): { node: string; label: string; reason: string } | null {
  if (!run) return null;
  const bad = run.status === "failed" || (run.status === "completed" && run.outcome === "failure");
  if (!bad && !steps.some((s) => s.state === "failed")) return null;
  if (!bad && run.status !== "cancelled") {
    // falha tratada por uma rota de erro: mostra mesmo assim enquanto o teste segue
  }
  let node = "";
  let reason = "";
  for (const e of events) {
    const p = payloadOf(e);
    if (e.type === "node_event" && txt(p.kind) === "error") {
      node = txt(p.node);
      reason = txt(p.text);
    }
  }
  if (!node) {
    const f = steps.find((s) => s.state === "failed");
    node = f?.id ?? "";
  }
  if (!reason) reason = run.failure || "O fluxo terminou em um nó de falha.";
  const step = steps.find((s) => s.id === node);
  return { node, label: step ? `${step.label} (${node})` : node || "fluxo", reason };
}

function nodeLine(e: ExecutionEvent): string {
  const p = payloadOf(e);
  switch (txt(p.kind)) {
    case "command":
      return txt(p.text).startsWith("$") ? txt(p.text) : `$ ${txt(p.text)}`;
    case "tool":
      return `⚙ ferramenta ${txt(p.tool)}`;
    case "error":
      return `✖ ${txt(p.text)}`;
    case "note":
      return `ℹ ${txt(p.text)}`;
    case "start":
      return `▶ ${txt(p.text)}`;
    case "done":
      return `✔ ${txt(p.text)}`;
    default:
      return txt(p.text);
  }
}

const icon: Record<StepState, string> = {
  pending: "○",
  running: "◔",
  ok: "●",
  waiting: "⏸",
  failed: "✖",
  skipped: "–",
};
const stateText: Record<StepState, string> = {
  pending: "aguardando a vez",
  running: "rodando",
  ok: "concluído",
  waiting: "pausado",
  failed: "falhou",
  skipped: "não percorrido",
};

function TerminalCard({
  step,
  events,
  active,
}: {
  step: Step;
  events: ExecutionEvent[];
  active: boolean;
}) {
  const ref = useRef<HTMLPreElement>(null);
  const lines = events.filter((e) => {
    const k = txt(payloadOf(e).kind);
    return k !== "prompt" && k !== "agent";
  });
  const cfg = events.map(payloadOf).find((p) => txt(p.kind) === "agent");
  const prompt = events.map(payloadOf).find((p) => txt(p.kind) === "prompt");
  // biome-ignore lint/correctness/useExhaustiveDependencies: rola para o fim quando chegam linhas novas.
  useEffect(() => {
    ref.current?.scrollTo?.({ top: ref.current.scrollHeight });
  }, [lines.length]);
  return (
    <section
      className={`ft-terminal${active ? " ft-terminal--active" : ""}${step.state === "failed" ? " ft-terminal--failed" : ""}`}
      aria-label={`Terminal ${step.id}`}
    >
      <header>
        <strong>
          {icon[step.state]} {step.label}
        </strong>
        <small className="muted">
          {step.id} · {stateText[step.state]}
        </small>
      </header>
      {cfg && (
        <p className="ft-config">
          Agente <strong>{txt(cfg.agent)}</strong> · {txt(cfg.provider)}
          {txt(cfg.model) ? ` · modelo ${txt(cfg.model)}` : " · modelo padrão da conta"} · conta{" "}
          {txt(cfg.account) || "—"}
          {Array.isArray(cfg.skills) && cfg.skills.length > 0
            ? ` · skills: ${cfg.skills.join(", ")}`
            : " · sem skills"}
        </p>
      )}
      {prompt && (
        <details>
          <summary>Orientações enviadas ao agente (o comando montado pela fábrica)</summary>
          <pre className="log ft-prompt">{txt(prompt.text)}</pre>
        </details>
      )}
      <pre className="log terminal" ref={ref} role="log" aria-label={`Saída de ${step.id}`}>
        {lines.map(nodeLine).join("\n") || "(sem saída ainda)"}
      </pre>
    </section>
  );
}

/**
 * Painel de teste do fluxo em tempo real: dispara a feature fictícia padronizada pelo fluxo escolhido e mostra as
 * etapas, os terminais abertos (com as orientações enviadas a cada agente) e, se algo falhar, onde e por quê.
 */
export function FlowTestModal({
  projectId,
  workflowId,
  onClose,
}: {
  projectId: string;
  workflowId?: string;
  onClose: () => void;
}) {
  const [run, setRun] = useState<WorkflowRun | null>(null);
  const [graph, setGraph] = useState<WorkflowGraph | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [starting, setStarting] = useState(true);
  const [busy, setBusy] = useState(false);
  const [round, setRound] = useState(0);
  const [clock, setClock] = useState(0);
  const closeRef = useRef<HTMLButtonElement>(null);
  const [diagnostics, setDiagnostics] = useState<Diagnostics | null>(null);
  const [copied, setCopied] = useState("");
  const { events, status } = useEventStream(projectId, run?.execution_id ?? null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: round reinicia o teste.
  useEffect(() => {
    let alive = true;
    setStarting(true);
    setError(null);
    setRun(null);
    setGraph(null);
    api
      .startTestRun(projectId, workflowId)
      .then((r) => alive && setRun(r))
      .catch((e: unknown) => alive && setError(toApiError(e)))
      .finally(() => alive && setStarting(false));
    return () => {
      alive = false;
    };
  }, [projectId, workflowId, round]);

  const runId = run?.execution_id;
  const versionId = run?.workflow_version_id;
  useEffect(() => {
    if (!versionId) return;
    api
      .workflowVersion(versionId)
      .then((v) => setGraph(v.flat_graph))
      .catch(() => {});
  }, [versionId]);

  const finished = !!run && ["completed", "failed", "cancelled"].includes(run.status);
  useEffect(() => {
    if (!runId || finished) return;
    const t = setInterval(() => {
      api
        .getRun(projectId, runId)
        .then(setRun)
        .catch(() => {});
    }, 1500);
    return () => clearInterval(t);
  }, [projectId, runId, finished]);

  useEffect(() => {
    if (!run || finished) return;
    const t = setInterval(() => setClock((c) => c + 1), 1000);
    return () => clearInterval(t);
  }, [run, finished]);

  useEffect(() => {
    closeRef.current?.focus();
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  // O relatório de diagnóstico é buscado quando o teste termina (e sob demanda, enquanto roda).
  const finishedKey = finished ? (run?.status ?? "") : "";
  useEffect(() => {
    if (!runId || !finishedKey) return;
    api
      .runDiagnostics(projectId, runId)
      .then(setDiagnostics)
      .catch(() => {});
  }, [projectId, runId, finishedKey]);

  async function fetchReport(): Promise<Diagnostics | null> {
    if (!runId) return null;
    try {
      const d = await api.runDiagnostics(projectId, runId);
      setDiagnostics(d);
      return d;
    } catch (e) {
      setError(toApiError(e));
      return null;
    }
  }

  async function copyReport() {
    const d = await fetchReport();
    if (!d) return;
    try {
      await navigator.clipboard.writeText(d.report);
      setCopied("Relatório copiado. Cole na conversa de suporte.");
    } catch {
      setCopied("Não consegui copiar sozinho: use “Baixar relatório” ou selecione o texto abaixo.");
    }
  }

  async function downloadReport() {
    const d = await fetchReport();
    if (!d) return;
    const url = URL.createObjectURL(new Blob([d.report], { type: "text/markdown;charset=utf-8" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `diagnostico-${runId}.md`;
    a.click();
    URL.revokeObjectURL(url);
  }

  const preflight = useMemo(
    () =>
      events
        .filter((e) => e.type === "run_diag" && txt(payloadOf(e).step) === "preflight")
        .map((e) => payloadOf(e)),
    [events],
  );
  const preflightProblems = preflight.filter((p) => p.ok === false);
  const diagEvents = useMemo(() => events.filter((e) => e.type === "run_diag"), [events]);

  const steps = useMemo(() => stepsOf(graph, events, run), [graph, events, run]);
  const failure = useMemo(() => failureOf(steps, events, run), [steps, events, run]);
  const byNode = useMemo(() => {
    const m = new Map<string, ExecutionEvent[]>();
    for (const e of events) {
      if (e.type !== "node_event") continue;
      const n = txt(payloadOf(e).node);
      m.set(n, [...(m.get(n) ?? []), e]);
    }
    return m;
  }, [events]);
  const terminals = steps.filter((s) => byNode.has(s.id));
  const elapsed = run
    ? Math.max(
        0,
        Math.round(
          ((finished ? new Date(run.updated_at).getTime() : Date.now() + clock * 0) -
            new Date(run.created_at).getTime()) /
            1000,
        ),
      )
    : 0;
  const passed = steps.filter((s) => s.state === "ok").length;

  async function act(fn: () => Promise<unknown>) {
    setBusy(true);
    try {
      await fn();
      if (run) setRun(await api.getRun(projectId, run.execution_id));
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  const verdict = !run
    ? ""
    : run.status === "completed" && run.outcome === "success"
      ? "Teste concluído com sucesso"
      : failure
        ? "Teste com falha"
        : run.status === "cancelled"
          ? "Teste cancelado"
          : run.status === "awaiting_decision"
            ? "Teste pausado: esperando a sua decisão"
            : "Teste em andamento";

  return (
    <div className="dialog-backdrop">
      <div
        className="manual panel ft-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="ft-title"
      >
        <header className="manual-head">
          <div>
            <p className="page-label">Painel de teste em tempo real</p>
            <h2 id="ft-title">Teste do fluxo</h2>
            <p className="muted">
              Feature fictícia padronizada: <strong>selo de saúde</strong> com uma trilha backend e
              uma frontend. Nada é enviado ao GitHub: o push e a Draft PR são simulados.
            </p>
          </div>
          <div className="actions">
            {run && !finished && (
              <button
                type="button"
                className="secondary-action"
                disabled={busy}
                onClick={() => act(() => api.cancelRun(projectId, run.execution_id))}
              >
                Cancelar teste
              </button>
            )}
            {(finished || error) && (
              <button type="button" onClick={() => setRound((r) => r + 1)}>
                Testar de novo
              </button>
            )}
            <button ref={closeRef} type="button" className="secondary-action" onClick={onClose}>
              Fechar
            </button>
          </div>
        </header>

        <FormError error={error} />
        {starting && (
          <p className="state-meta" role="status">
            Preparando o teste (publicando o fluxo e abrindo a execução)…
          </p>
        )}

        {run && (
          <div className="ft-body">
            <p
              className={`ft-verdict ${failure ? "ft-verdict--bad" : run.status === "completed" ? "ft-verdict--ok" : ""}`}
              role="status"
            >
              <strong>{verdict}</strong> · {statusText(run)} · {elapsed}s · {passed}/
              {steps.length || "…"} etapas · terminal{" "}
              {status === "live" ? "ao vivo" : "reconectando"}
            </p>
            {failure && (
              <section className="ft-failure" aria-label="Falha do teste" role="alert">
                <h3>Onde falhou</h3>
                <p>
                  <strong>{failure.label}</strong>
                </p>
                <p>
                  <strong>Motivo:</strong> {failure.reason}
                </p>
              </section>
            )}
            {run.status === "awaiting_decision" && (
              <section className="callout" aria-label="Decisão pendente">
                <p>
                  O fluxo tem um nó de decisão humana e parou em{" "}
                  <strong>{run.pending.join(", ")}</strong>. Responda para o teste continuar.
                </p>
                <div className="actions">
                  {run.pending.map((node) => (
                    <span key={node} className="actions">
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() =>
                          act(() => api.decideRun(projectId, run.execution_id, node, "approved"))
                        }
                      >
                        Aprovar ({node})
                      </button>
                      <button
                        type="button"
                        className="secondary-action"
                        disabled={busy}
                        onClick={() =>
                          act(() => api.decideRun(projectId, run.execution_id, node, "rejected"))
                        }
                      >
                        Recusar ({node})
                      </button>
                    </span>
                  ))}
                </div>
              </section>
            )}
            <div className="ft-columns">
              <section aria-label="Etapas do fluxo" className="ft-steps">
                <h3>Etapas</h3>
                <ol>
                  {steps.map((s) => (
                    <li key={s.id} className={`ft-step ft-step--${s.state}`}>
                      <span aria-hidden="true">{icon[s.state]}</span>
                      <span>
                        <strong>{s.label}</strong>
                        <small className="muted">
                          {" "}
                          {s.id} · {stateText[s.state]}
                        </small>
                      </span>
                    </li>
                  ))}
                  {steps.length === 0 && <li className="muted">Carregando o fluxo…</li>}
                </ol>
              </section>
              <section aria-label="Terminais" className="ft-terminals">
                <h3>Terminais</h3>
                {terminals.length === 0 && (
                  <p className="muted">
                    Nenhum terminal aberto ainda: o primeiro agente ou comando abre o seu aqui.
                  </p>
                )}
                {terminals.map((s) => (
                  <TerminalCard
                    key={s.id}
                    step={s}
                    events={byNode.get(s.id) ?? []}
                    active={s.state === "running"}
                  />
                ))}
              </section>
            </div>
            {preflightProblems.length > 0 && (
              <section className="ft-failure" aria-label="Problemas do ambiente" role="alert">
                <h3>O ambiente tem problemas (pré-voo)</h3>
                <ul>
                  {preflightProblems.map((p) => (
                    <li key={txt(p.check)}>
                      <strong>{txt(p.check)}</strong>: {txt(p.detail)}
                      {txt(p.dica) && <small className="muted"> — {txt(p.dica)}</small>}
                    </li>
                  ))}
                </ul>
              </section>
            )}
            <section className="ft-diagnostics" aria-label="Diagnóstico">
              <h3>Diagnóstico</h3>
              {diagnostics && diagnostics.hints.length > 0 && failure && (
                <div role="status" aria-label="Causas prováveis">
                  <p>
                    <strong>Causas prováveis e o que fazer:</strong>
                  </p>
                  <ul>
                    {diagnostics.hints.map((h) => (
                      <li key={h.cause}>
                        <strong>{h.cause}</strong> {h.fix}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              <p className="muted">
                O relatório reúne o veredito, o pré-voo do ambiente, a linha do tempo com tempos e o
                detalhe técnico de cada etapa (sandbox, CLI, patch, comandos). Copie e cole na
                conversa para pedir ajuda com o problema.
              </p>
              <div className="actions">
                <button type="button" className="secondary-action" onClick={copyReport}>
                  Copiar relatório de diagnóstico
                </button>
                <button type="button" className="secondary-action" onClick={downloadReport}>
                  Baixar relatório (.md)
                </button>
              </div>
              {copied && (
                <p className="state-meta" role="status">
                  {copied}
                </p>
              )}
              <details>
                <summary>Diagnóstico técnico em tempo real ({diagEvents.length} registros)</summary>
                <pre className="log" role="log" aria-label="Diagnóstico técnico">
                  {diagEvents.map(terminalLine).join("\n") || "(nada ainda)"}
                </pre>
              </details>
            </section>
            <details className="ft-timeline">
              <summary>Linha do tempo completa ({events.length} eventos)</summary>
              <pre className="log" role="log" aria-label="Linha do tempo">
                {events.map(terminalLine).join("\n")}
              </pre>
            </details>
          </div>
        )}
      </div>
    </div>
  );
}

function statusText(run: WorkflowRun): string {
  const m: Record<WorkflowRun["status"], string> = {
    queued: "na fila do worker",
    running: "em execução",
    waiting: "dormindo",
    awaiting_decision: "esperando decisão",
    completed: run.outcome === "success" ? "concluída" : "concluída com falha",
    failed: "falhou",
    cancelled: "cancelada",
  };
  return m[run.status];
}
