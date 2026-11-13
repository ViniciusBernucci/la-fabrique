import { useEffect, useRef, useState } from "react";
import type { ExecutionEvent } from "./api";

export type StreamStatus = "connecting" | "live" | "reconnecting";

const MAX_KEPT = 1000;

/**
 * Acompanha os eventos de uma execução por SSE. Guarda o cursor (seq do último evento) e, ao
 * cair a conexão ou receber `reconnect`, reabre com `?after=<cursor>`: o servidor devolve só o
 * que faltou e eventos já vistos são ignorados.
 */
export function useEventStream(
  projectId: string,
  executionId: string | null,
): { events: ExecutionEvent[]; status: StreamStatus } {
  const [events, setEvents] = useState<ExecutionEvent[]>([]);
  const [status, setStatus] = useState<StreamStatus>("connecting");
  const cursor = useRef(0);

  useEffect(() => {
    cursor.current = 0;
    setEvents([]);
    if (!executionId || typeof EventSource === "undefined") return;
    let es: EventSource | null = null;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let closed = false;
    let backoff = 1000;

    const open = () => {
      if (closed) return;
      const url = `/api/projects/${encodeURIComponent(projectId)}/executions/${encodeURIComponent(executionId)}/stream?after=${cursor.current}`;
      es = new EventSource(url, { withCredentials: true });
      es.onopen = () => {
        backoff = 1000;
        setStatus("live");
      };
      es.onmessage = (m: MessageEvent<string>) => {
        let ev: ExecutionEvent;
        try {
          ev = JSON.parse(m.data) as ExecutionEvent;
        } catch {
          return;
        }
        if (ev.seq <= cursor.current) return; // já visto
        cursor.current = ev.seq;
        setEvents((prev) => [...prev, ev].slice(-MAX_KEPT));
      };
      const reopen = () => {
        es?.close();
        setStatus("reconnecting");
        timer = setTimeout(open, backoff);
        backoff = Math.min(backoff * 2, 15000);
      };
      es.addEventListener("reconnect", () => {
        es?.close();
        setStatus("reconnecting");
        timer = setTimeout(open, 0);
      });
      es.onerror = reopen;
    };
    setStatus("connecting");
    open();
    return () => {
      closed = true;
      if (timer) clearTimeout(timer);
      es?.close();
    };
  }, [projectId, executionId]);

  return { events, status };
}
