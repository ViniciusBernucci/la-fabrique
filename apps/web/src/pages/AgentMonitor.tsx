import { type FormEvent, useCallback, useEffect, useRef, useState } from "react";
import {
  ApiError,
  api,
  type ExecutionMonitor,
  type GuidanceItem,
  type MonitorEvent,
} from "../lib/api";

function clock(total: number): string {
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const p = (n: number) => String(n).padStart(2, "0");
  return h > 0 ? `${h}:${p(m)}:${p(s)}` : `${p(m)}:${p(s)}`;
}

function line(e: MonitorEvent): string {
  const who = e.subagent ? `[${e.subagent}] ` : "";
  switch (e.kind) {
    case "tool":
      return `#${e.seq} ${who}$ ${e.tool ?? "ferramenta"} ${e.text ?? ""}`.trimEnd();
    case "error":
      return `#${e.seq} ${who}ERRO ${e.text ?? ""}`.trimEnd();
    case "gap":
      return `#${e.seq} ⚠ lacuna: eventos descartados por pressão`;
    case "done":
      return `#${e.seq} ${who}— fim —`;
    default:
      return `#${e.seq} ${who}${e.text ?? ""}`.trimEnd();
  }
}

/** Terminal, cronômetro, ferramentas e subagentes do agente (CLI ou API); reconecta pelo cursor sem repetir. */
export function AgentMonitor({
  projectId,
  executionId,
}: {
  projectId: string;
  executionId: string;
}) {
  const [events, setEvents] = useState<MonitorEvent[]>([]);
  const [summary, setSummary] = useState<ExecutionMonitor["summary"] | null>(null);
  const [failed, setFailed] = useState(false);
  const cursor = useRef(0);

  const poll = useCallback(async () => {
    try {
      const v = await api.executionMonitor(projectId, executionId, cursor.current);
      cursor.current = v.cursor;
      setFailed(false);
      setSummary(v.summary);
      if (v.events.length > 0) setEvents((prev) => [...prev, ...v.events]);
    } catch {
      setFailed(true);
    }
  }, [projectId, executionId]);

  useEffect(() => {
    cursor.current = 0;
    setEvents([]);
    setSummary(null);
    void poll();
    const t = setInterval(() => void poll(), 2000);
    return () => clearInterval(t);
  }, [poll]);

  return (
    <section aria-label="Monitor do agente">
      <h3>Monitor do agente</h3>
      {failed && <p className="state-meta">Reconectando ao monitor pelo cursor…</p>}
      {summary && (
        <p className="state-meta" role="status">
          {summary.done ? "Concluída" : "Em andamento"} · tempo {clock(summary.elapsed_seconds)} ·{" "}
          {summary.tool_calls} ferramenta(s) · {summary.errors} erro(s)
          {summary.progress != null ? ` · progresso ${summary.progress}%` : ""}
          {summary.dropped > 0 ? ` · ${summary.dropped} evento(s) descartado(s) por pressão` : ""}
        </p>
      )}
      {summary && summary.subagents.length > 0 && (
        <ul className="item-list" aria-label="Subagentes">
          {summary.subagents.map((s) => (
            <li key={s.name}>
              <div className="item-link">
                <span>
                  <strong>{s.name}</strong>
                  <small>
                    {s.events} evento(s) ·{" "}
                    {s.progress != null
                      ? `progresso ${s.progress}%`
                      : "sem telemetria de progresso"}
                  </small>
                </span>
                <span className={`badge ${s.state === "done" ? "badge--active" : ""}`}>
                  {s.state === "done" ? "concluído" : "em andamento"}
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
      {events.length === 0 ? (
        <p className="state-meta">Nenhuma saída do agente ainda.</p>
      ) : (
        <pre className="log" role="log" aria-label="Terminal do agente">
          {events.map(line).join("\n")}
        </pre>
      )}
    </section>
  );
}

const stateLabel: Record<GuidanceItem["state"], string> = {
  queued: "na fila",
  sent: "enviada (sem confirmação do cliente)",
  acknowledged: "confirmada pelo cliente",
  uncertain: "entrega incerta",
  undeliverable: "não entregue (execução terminou)",
};

/** Orientações ao agente: o estado mostrado é só o que foi comprovado. */
export function GuidanceChat({
  projectId,
  executionId,
}: {
  projectId: string;
  executionId: string;
}) {
  const [items, setItems] = useState<GuidanceItem[]>([]);
  const [live, setLive] = useState(false);
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await api.listGuidance(projectId, executionId);
      setItems(r.items);
      setLive(r.live);
    } catch {
      /* o histórico volta no próximo ciclo */
    }
  }, [projectId, executionId]);

  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), 4000);
    return () => clearInterval(t);
  }, [load]);

  async function send(e: FormEvent) {
    e.preventDefault();
    if (!text.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await api.sendGuidance(projectId, executionId, text.trim());
      setText("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? `${err.message} ${err.action}` : "Falha ao enviar.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section aria-label="Orientações ao agente">
      <h3>Orientações ao agente</h3>
      <p className="state-meta">
        {live
          ? "Executor ativo: a entrega segue a capacidade do cliente."
          : "Sem executor ativo: a orientação fica registrada para a retomada."}{" "}
        A orientação não altera contrato, permissões nem aprovações.
      </p>
      <form onSubmit={send} className="inline-form">
        <input
          aria-label="Orientação"
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Escreva a orientação para o agente"
          maxLength={4000}
        />
        <button type="submit" className="secondary-action" disabled={busy || !text.trim()}>
          Enviar
        </button>
      </form>
      {error && (
        <p className="state-meta" role="alert">
          {error}
        </p>
      )}
      {items.length === 0 ? (
        <p className="state-meta">Nenhuma orientação enviada.</p>
      ) : (
        <ul className="item-list" aria-label="Histórico de orientações">
          {items.map((g) => (
            <li key={g.id}>
              <div className="item-link">
                <span>
                  <strong>{g.body}</strong>
                  <small>
                    {g.author} · {g.detail || stateLabel[g.state]}
                  </small>
                </span>
                <span
                  className={`badge ${g.state === "acknowledged" ? "badge--active" : "badge--paused"}`}
                >
                  {stateLabel[g.state]}
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
