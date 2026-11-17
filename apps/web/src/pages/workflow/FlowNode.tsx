import { Handle, type NodeProps, Position } from "@xyflow/react";
import { createContext, type ReactNode, useContext } from "react";
import type { NodeType } from "../../lib/api";
import { nodeSummary, type WfNode } from "./flowModel";

/** O que o cartão precisa saber além dos dados do nó: o catálogo de tipos e os problemas por nó. */
export interface EditorContextValue {
  types: Map<string, NodeType>;
  issues: Map<string, number>;
}

export const EditorContext = createContext<EditorContextValue>({
  types: new Map(),
  issues: new Map(),
});

const svg = (children: ReactNode) => (
  <svg
    viewBox="0 0 24 24"
    width="20"
    height="20"
    fill="none"
    stroke="currentColor"
    strokeWidth="1.8"
    strokeLinecap="round"
    strokeLinejoin="round"
    aria-hidden="true"
  >
    {children}
  </svg>
);

const ICONS: Record<string, ReactNode> = {
  start: svg(<path d="M7 4.5v15l12-7.5z" />),
  end: svg(
    <>
      <path d="M6 21V4" />
      <path d="M6 4h12l-2.5 4L18 12H6" />
    </>,
  ),
  agent: svg(
    <>
      <rect x="4" y="8" width="16" height="11" rx="3" />
      <path d="M12 8V4.5M9 13.5h.01M15 13.5h.01M9.5 16.5h5" />
      <circle cx="12" cy="3.5" r="1" />
    </>,
  ),
  skill: svg(
    <path d="M10 3.5a2 2 0 1 1 4 0V6h3.5a1 1 0 0 1 1 1v3.5H21a2 2 0 1 1 0 4h-2.5V18a1 1 0 0 1-1 1H14v-2.5a2 2 0 1 0-4 0V19H6.5a1 1 0 0 1-1-1v-3.5H3a2 2 0 1 1 0-4h2.5V7a1 1 0 0 1 1-1H10z" />,
  ),
  script: svg(
    <>
      <path d="M4 5h16v14H4z" />
      <path d="m8 10 3 2-3 2M13 15h3" />
    </>,
  ),
  checks: svg(
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="m8.5 12.2 2.4 2.4 4.6-5" />
    </>,
  ),
  git: svg(
    <>
      <circle cx="6.5" cy="5.5" r="2" />
      <circle cx="6.5" cy="18.5" r="2" />
      <circle cx="17.5" cy="9" r="2" />
      <path d="M6.5 7.5v9M17.5 11c0 3-4 3-9.2 6" />
    </>,
  ),
  docs: svg(
    <>
      <path d="M6 3.5h8l4 4V20.5H6z" />
      <path d="M14 3.5V8h4M9 12.5h6M9 16h6" />
    </>,
  ),
  condition: svg(
    <>
      <path d="M12 3.5v6" />
      <path d="M12 9.5 6 15.5v5M12 9.5l6 6v5" />
    </>,
  ),
  loop: svg(
    <>
      <path d="M4 12a8 8 0 0 1 13.7-5.7L20 8.5" />
      <path d="M20 4v4.5h-4.5M20 12a8 8 0 0 1-13.7 5.7L4 15.5M4 20v-4.5h4.5" />
    </>,
  ),
  wait: svg(
    <>
      <circle cx="12" cy="12" r="8.5" />
      <path d="M12 7.5V12l3 2" />
    </>,
  ),
  human_decision: svg(
    <>
      <circle cx="12" cy="8" r="3.5" />
      <path d="M5 20c.5-4 3.4-6 7-6s6.5 2 7 6" />
    </>,
  ),
  subflow: svg(
    <>
      <rect x="3.5" y="4" width="7" height="6" rx="1.5" />
      <rect x="13.5" y="14" width="7" height="6" rx="1.5" />
      <path d="M7 10v4a2 2 0 0 0 2 2h4.5" />
    </>,
  ),
  converge: svg(<path d="M4 6h5c4 0 4 6 8 6h3M4 18h5c4 0 4-6 8-6" />),
  fork: svg(<path d="M4 12h3c4 0 4-6 8-6h5M7 12c4 0 4 6 8 6h5" />),
};

export function nodeIcon(type: string): ReactNode {
  return ICONS[type] ?? ICONS.agent;
}

const LABELED_HIDDEN = new Set(["in", "out"]);

/** Cartão de um nó, no estilo dos editores de fluxo: ícone por tipo, resumo, portas nomeadas. */
export function FlowNodeCard({ id, data, selected }: NodeProps<WfNode>) {
  const { types, issues } = useContext(EditorContext);
  const t = types.get(data.wfType);
  const inputs = t?.inputs ?? [];
  const outputs = t?.outputs ?? [];
  const problems = issues.get(id) ?? 0;
  const summary = nodeSummary(data.wfType, data.props);
  const rows = Math.max(inputs.length, outputs.length);
  const labelOf = (p: string) => (LABELED_HIDDEN.has(p) ? "" : p);

  return (
    <article
      className={[
        "wf-card",
        `wf-card--${t?.category ?? "execução"}`,
        `wf-card--${data.wfType}`,
        selected ? "wf-card--selected" : "",
        problems > 0 ? "wf-card--invalid" : "",
      ]
        .filter(Boolean)
        .join(" ")}
      aria-label={`Nó ${id} (${t?.label ?? data.wfType})${problems > 0 ? ", com pendências" : ""}`}
    >
      <div className="wf-card-head">
        <span className="wf-card-icon">{nodeIcon(data.wfType)}</span>
        <span className="wf-card-text">
          <strong>{t?.label ?? data.wfType}</strong>
          <small>{id}</small>
        </span>
        {problems > 0 && (
          <span
            className="wf-card-alert"
            title={`${problems} pendência(s)`}
            role="img"
            aria-label={`${problems} pendência(s)`}
          >
            !
          </span>
        )}
      </div>
      {summary && (
        <p className={`wf-card-summary${problems > 0 ? " wf-card-summary--warn" : ""}`}>
          {summary}
        </p>
      )}
      {rows > 0 && (
        <div className="wf-card-ports">
          {Array.from({ length: rows }, (_, i) => {
            const input = inputs[i];
            const output = outputs[i];
            return (
              <div className="wf-port-row" key={`${input ?? ""}|${output ?? ""}`}>
                <span className="wf-port-label wf-port-label--in">
                  {input ? labelOf(input) : ""}
                </span>
                <span className="wf-port-label wf-port-label--out">
                  {output ? labelOf(output) : ""}
                </span>
                {input && (
                  <Handle
                    type="target"
                    id={input}
                    position={Position.Left}
                    className="wf-handle wf-handle--in"
                    aria-label={`Entrada ${input} de ${id}`}
                  />
                )}
                {output && (
                  <Handle
                    type="source"
                    id={output}
                    position={Position.Right}
                    className={`wf-handle wf-handle--out wf-handle--${output}`}
                    aria-label={`Saída ${output} de ${id}`}
                  />
                )}
              </div>
            );
          })}
        </div>
      )}
    </article>
  );
}
