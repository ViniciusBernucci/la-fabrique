import { type KeyboardEvent, type PointerEvent, useEffect, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  api,
  type FieldSpec,
  type NodeProps,
  type NodeType,
  type Workflow,
  type WorkflowEdge,
  type WorkflowGraph,
  type WorkflowIssue,
  type WorkflowNode,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const NODE_W = 190;
const HEAD = 32;
const ROW = 24;
const GRID = 10;

const snap = (n: number) => Math.max(0, Math.round(n / GRID) * GRID);

function nodeHeight(t: NodeType | undefined): number {
  const rows = Math.max(t?.inputs?.length ?? 0, t?.outputs.length ?? 0, 1);
  return HEAD + rows * ROW + 8;
}

function portY(node: WorkflowNode, i: number): number {
  return node.y + HEAD + ROW * i + ROW / 2;
}

/** Posição de uma porta no canvas (entrada à esquerda, saída à direita). */
function portPoint(
  graph: WorkflowGraph,
  types: Map<string, NodeType>,
  nodeId: string,
  port: string,
  side: "in" | "out",
): { x: number; y: number } | null {
  const node = graph.nodes.find((n) => n.id === nodeId);
  const type = node && types.get(node.type);
  if (!node || !type) return null;
  const list = side === "in" ? (type.inputs ?? []) : type.outputs;
  const i = list.indexOf(port);
  if (i < 0) return null;
  return { x: side === "in" ? node.x : node.x + NODE_W, y: portY(node, i) };
}

export function WorkflowEditorPage() {
  const { id = "" } = useParams();
  const { expire } = useSession();
  const wf = useResource((s) => api.getWorkflow(id, s), `workflow:${id}`, expire);
  const types = useResource((s) => api.workflowNodeTypes(s), "workflow-node-types", expire);

  if (wf.state.status === "loading" || types.state.status === "loading") {
    return <LoadingState label="Carregando editor…" />;
  }
  if (wf.state.status === "error") return <ErrorState error={wf.state.error} onRetry={wf.reload} />;
  if (types.state.status === "error") {
    return <ErrorState error={types.state.error} onRetry={types.reload} />;
  }
  return <Editor initial={wf.state.data} nodeTypes={types.state.data.items} />;
}

function Editor({ initial, nodeTypes }: { initial: Workflow; nodeTypes: NodeType[] }) {
  const { expire } = useSession();
  const typeMap = new Map(nodeTypes.map((t) => [t.type, t]));
  const [graph, setGraph] = useState<WorkflowGraph>(initial.graph);
  const [name, setName] = useState(initial.name);
  const [description, setDescription] = useState(initial.description);
  const [version, setVersion] = useState(initial.version);
  const [issues, setIssues] = useState<WorkflowIssue[]>(initial.issues);
  const [dirty, setDirty] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [selected, setSelected] = useState<{ kind: "node" | "edge"; id: string } | null>(null);
  const [pending, setPending] = useState<{ node: string; port: string } | null>(null);
  const drag = useRef<{ id: string; dx: number; dy: number } | null>(null);

  const change = (fn: (g: WorkflowGraph) => WorkflowGraph) => {
    setGraph(fn);
    setDirty(true);
    setSaved(false);
  };

  // Validação no servidor (mesmas regras do salvamento), sem gravar.
  // biome-ignore lint/correctness/useExhaustiveDependencies: expire é estável; o efeito segue só o grafo.
  useEffect(() => {
    if (!dirty) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      api
        .validateWorkflowGraph(graph, controller.signal)
        .then((r) => setIssues(r.issues))
        .catch((e: unknown) => {
          if (e instanceof Error && e.name !== "AbortError" && "isUnauthorized" in e) {
            if ((e as ApiError).isUnauthorized) expire();
          }
        });
    }, 350);
    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [graph, dirty]);

  function addNode(type: NodeType) {
    change((g) => {
      let n = g.nodes.filter((x) => x.type === type.type).length + 1;
      while (g.nodes.some((x) => x.id === `${type.type}-${n}`)) n++;
      const count = g.nodes.length;
      const node: WorkflowNode = {
        id: `${type.type}-${n}`,
        type: type.type,
        x: 60 + (count % 5) * 220,
        y: 40 + Math.floor(count / 5) * 160,
        props: {},
      };
      setSelected({ kind: "node", id: node.id });
      return { ...g, nodes: [...g.nodes, node] };
    });
  }

  function connect(toNode: string, toPort: string) {
    if (!pending) return;
    const from = pending;
    setPending(null);
    change((g) => {
      if (
        g.edges.some(
          (e) =>
            e.from === from.node &&
            e.from_port === from.port &&
            e.to === toNode &&
            e.to_port === toPort,
        )
      ) {
        return g;
      }
      let n = g.edges.length + 1;
      while (g.edges.some((e) => e.id === `e${n}`)) n++;
      const edge: WorkflowEdge = {
        id: `e${n}`,
        from: from.node,
        from_port: from.port,
        to: toNode,
        to_port: toPort,
      };
      return { ...g, edges: [...g.edges, edge] };
    });
  }

  function removeSelected() {
    if (!selected) return;
    change((g) =>
      selected.kind === "node"
        ? {
            nodes: g.nodes.filter((n) => n.id !== selected.id),
            edges: g.edges.filter((e) => e.from !== selected.id && e.to !== selected.id),
          }
        : { ...g, edges: g.edges.filter((e) => e.id !== selected.id) },
    );
    setSelected(null);
  }

  function setProp(nodeId: string, field: FieldSpec, raw: string) {
    change((g) => ({
      ...g,
      nodes: g.nodes.map((n) => {
        if (n.id !== nodeId) return n;
        const props: NodeProps = { ...n.props };
        if (raw === "") delete props[field.name];
        else props[field.name] = field.kind === "int" ? Number(raw) : raw;
        return { ...n, props };
      }),
    }));
  }

  function onPointerMove(event: PointerEvent<SVGSVGElement>) {
    const d = drag.current;
    if (!d) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const x = snap(event.clientX - rect.left - d.dx);
    const y = snap(event.clientY - rect.top - d.dy);
    change((g) => ({ ...g, nodes: g.nodes.map((n) => (n.id === d.id ? { ...n, x, y } : n)) }));
  }

  async function save() {
    setSaving(true);
    setError(null);
    try {
      const wf = await api.saveWorkflow(initial.id, {
        expected_version: version,
        name,
        description,
        graph,
      });
      setVersion(wf.version);
      setIssues(wf.issues);
      setDirty(false);
      setSaved(true);
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setSaving(false);
    }
  }

  const issueNodes = new Set(issues.filter((i) => i.node).map((i) => i.node));
  const selectedNode =
    selected?.kind === "node" ? graph.nodes.find((n) => n.id === selected.id) : null;
  const selectedType = selectedNode ? typeMap.get(selectedNode.type) : undefined;
  const width = Math.max(1200, ...graph.nodes.map((n) => n.x + NODE_W + 80));
  const height = Math.max(
    640,
    ...graph.nodes.map((n) => n.y + nodeHeight(typeMap.get(n.type)) + 80),
  );
  const onKey = (action: () => void) => (e: KeyboardEvent) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      action();
    }
  };

  return (
    <div className="page">
      <nav className="breadcrumb" aria-label="Trilha">
        <Link to="/workflows">Workflows</Link>
        <span aria-hidden="true">/</span>
        <span>{initial.name}</span>
      </nav>
      <PageHeader
        label={initial.kind === "feature" ? "Fluxo de feature" : "Subfluxo"}
        title={name}
        lead="Rascunho: não executa. Arraste os nós, clique numa saída e depois numa entrada para conectar."
        actions={
          <div className="actions">
            <span className={dirty ? "badge badge--paused" : "badge badge--active"}>
              {dirty ? "Alterações não salvas" : saved ? "Salvo" : `Versão ${version}`}
            </span>
            <button type="button" disabled={saving || !dirty} onClick={save}>
              Salvar rascunho
            </button>
          </div>
        }
      />
      <FormError error={error} />
      <section className="panel panel--pad wf-meta" aria-label="Dados do workflow">
        <label>
          Nome
          <input
            value={name}
            maxLength={120}
            onChange={(e) => {
              setName(e.target.value);
              setDirty(true);
            }}
          />
        </label>
        <label>
          Descrição
          <input
            value={description}
            maxLength={500}
            onChange={(e) => {
              setDescription(e.target.value);
              setDirty(true);
            }}
          />
        </label>
      </section>
      <div className="wf-layout">
        <aside className="panel panel--pad wf-palette" aria-label="Paleta de nós">
          <h2 className="form-title">Adicionar nó</h2>
          {(["fluxo", "execução", "controle"] as const).map((cat) => (
            <div key={cat}>
              <p className="muted">{cat}</p>
              <div className="wf-palette-group">
                {nodeTypes
                  .filter((t) => t.category === cat)
                  .map((t) => (
                    <button
                      key={t.type}
                      type="button"
                      className="secondary-action"
                      onClick={() => addNode(t)}
                    >
                      {t.label}
                    </button>
                  ))}
              </div>
            </div>
          ))}
        </aside>
        <section className="panel wf-canvas" aria-label="Canvas do workflow">
          <svg
            width={width}
            height={height}
            role="group"
            aria-label="Grafo do workflow"
            onPointerMove={onPointerMove}
            onPointerUp={() => {
              drag.current = null;
            }}
          >
            <rect
              width={width}
              height={height}
              className="wf-bg"
              onClick={() => {
                setSelected(null);
                setPending(null);
              }}
            />
            {graph.edges.map((e) => {
              const a = portPoint(graph, typeMap, e.from, e.from_port, "out");
              const b = portPoint(graph, typeMap, e.to, e.to_port, "in");
              if (!a || !b) return null;
              const d = `M ${a.x} ${a.y} C ${a.x + 60} ${a.y}, ${b.x - 60} ${b.y}, ${b.x} ${b.y}`;
              const isSel = selected?.kind === "edge" && selected.id === e.id;
              return (
                <g
                  key={e.id}
                  role="button"
                  tabIndex={0}
                  aria-label={`Conexão de ${e.from}.${e.from_port} para ${e.to}.${e.to_port}`}
                  aria-pressed={isSel}
                  onClick={() => setSelected({ kind: "edge", id: e.id })}
                  onKeyDown={onKey(() => setSelected({ kind: "edge", id: e.id }))}
                >
                  <path d={d} className="wf-edge-hit" />
                  <path d={d} className={isSel ? "wf-edge wf-edge--selected" : "wf-edge"} />
                </g>
              );
            })}
            {graph.nodes.map((n) => {
              const t = typeMap.get(n.type);
              const invalid = issueNodes.has(n.id);
              const isSel = selected?.kind === "node" && selected.id === n.id;
              return (
                <g key={n.id} transform={`translate(${n.x} ${n.y})`}>
                  <rect
                    width={NODE_W}
                    height={nodeHeight(t)}
                    rx={8}
                    className={[
                      "wf-node",
                      `wf-node--${t?.category ?? "execução"}`,
                      invalid ? "wf-node--invalid" : "",
                      isSel ? "wf-node--selected" : "",
                    ].join(" ")}
                  />
                  <g
                    role="button"
                    tabIndex={0}
                    aria-label={`Nó ${n.id} (${t?.label ?? n.type})${invalid ? ", com pendências" : ""}`}
                    aria-pressed={isSel}
                    className="wf-node-head"
                    onClick={() => setSelected({ kind: "node", id: n.id })}
                    onKeyDown={onKey(() => setSelected({ kind: "node", id: n.id }))}
                    onPointerDown={(ev) => {
                      const svg = ev.currentTarget.ownerSVGElement;
                      const rect = svg?.getBoundingClientRect();
                      drag.current = {
                        id: n.id,
                        dx: ev.clientX - (rect?.left ?? 0) - n.x,
                        dy: ev.clientY - (rect?.top ?? 0) - n.y,
                      };
                      setSelected({ kind: "node", id: n.id });
                    }}
                  >
                    <rect width={NODE_W} height={HEAD} rx={8} className="wf-node-title" />
                    <text x={10} y={21} className="wf-text-strong">
                      {t?.label ?? n.type}
                    </text>
                    <text x={NODE_W - 8} y={21} textAnchor="end" className="wf-text-muted">
                      {n.id}
                    </text>
                  </g>
                  {(t?.inputs ?? []).map((p, i) => (
                    <g key={`in-${p}`}>
                      <circle
                        cx={0}
                        cy={HEAD + ROW * i + ROW / 2}
                        r={7}
                        role="button"
                        tabIndex={0}
                        aria-label={`Entrada ${p} de ${n.id}`}
                        className={pending ? "wf-port wf-port--target" : "wf-port"}
                        onClick={() => connect(n.id, p)}
                        onKeyDown={onKey(() => connect(n.id, p))}
                      />
                      <text x={12} y={HEAD + ROW * i + ROW / 2 + 4} className="wf-text-muted">
                        {p}
                      </text>
                    </g>
                  ))}
                  {(t?.outputs ?? []).map((p, i) => {
                    const active = pending?.node === n.id && pending.port === p;
                    return (
                      <g key={`out-${p}`}>
                        <text
                          x={NODE_W - 12}
                          y={HEAD + ROW * i + ROW / 2 + 4}
                          textAnchor="end"
                          className="wf-text-muted"
                        >
                          {p}
                        </text>
                        <circle
                          cx={NODE_W}
                          cy={HEAD + ROW * i + ROW / 2}
                          r={7}
                          role="button"
                          tabIndex={0}
                          aria-label={`Saída ${p} de ${n.id}`}
                          aria-pressed={active}
                          className={active ? "wf-port wf-port--active" : "wf-port"}
                          onClick={() => setPending({ node: n.id, port: p })}
                          onKeyDown={onKey(() => setPending({ node: n.id, port: p }))}
                        />
                      </g>
                    );
                  })}
                </g>
              );
            })}
          </svg>
        </section>
        <aside className="panel panel--pad wf-side" aria-label="Propriedades">
          <div className="actions">
            <h2 className="form-title">Propriedades</h2>
            <button
              type="button"
              className="secondary-action"
              disabled={!selected}
              onClick={removeSelected}
            >
              Remover seleção
            </button>
          </div>
          {pending && (
            <p className="callout" role="status">
              Escolha a entrada de destino para a saída {pending.port} de {pending.node}.
            </p>
          )}
          {!selectedNode && (
            <p className="muted">
              {selected?.kind === "edge"
                ? "Conexão selecionada."
                : "Selecione um nó para editar suas propriedades."}
            </p>
          )}
          {selectedNode && selectedType && (
            <div className="form">
              <p className="muted">
                {selectedType.label} · <code>{selectedNode.id}</code>
              </p>
              {(selectedType.fields ?? []).length === 0 && (
                <p className="muted">Este tipo de nó não tem propriedades.</p>
              )}
              {(selectedType.fields ?? []).map((f) => (
                <FieldEditor
                  key={f.name}
                  field={f}
                  value={selectedNode.props[f.name]}
                  issues={issues.filter((i) => i.node === selectedNode.id && i.field === f.name)}
                  onChange={(raw) => setProp(selectedNode.id, f, raw)}
                />
              ))}
              {issues
                .filter(
                  (i) =>
                    i.node === selectedNode.id &&
                    !(selectedType.fields ?? []).some((f) => f.name === i.field),
                )
                .map((i) => (
                  <p key={`${i.field}-${i.code}`} className="form-error">
                    {i.field ? `${i.field}: ` : ""}
                    {i.message}
                  </p>
                ))}
            </div>
          )}
        </aside>
      </div>
      <section className="panel panel--pad" aria-label="Validação">
        <h2 className="form-title">Validação</h2>
        {issues.length === 0 ? (
          <p className="callout" role="status">
            Nenhum problema encontrado.
          </p>
        ) : (
          <ul className="checklist">
            {issues.map((i) => (
              <li
                key={`${i.node}-${i.field}-${i.code}`}
                className="checklist-item checklist-item--pending"
              >
                <span className="checklist-mark" aria-hidden="true">
                  !
                </span>
                <span>
                  <strong>
                    {i.node ? `Nó ${i.node}` : "Fluxo"}
                    {i.field ? ` · ${i.field}` : ""}
                  </strong>
                  <small>{i.message}</small>
                </span>
                {i.node && graph.nodes.some((n) => n.id === i.node) && (
                  <button
                    type="button"
                    className="secondary-action"
                    onClick={() => setSelected({ kind: "node", id: i.node })}
                  >
                    Localizar
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function FieldEditor({
  field,
  value,
  issues,
  onChange,
}: {
  field: FieldSpec;
  value: string | number | undefined;
  issues: WorkflowIssue[];
  onChange: (raw: string) => void;
}) {
  const v = value === undefined ? "" : String(value);
  const label = `${field.label}${field.required ? " *" : ""}`;
  const id = `wf-field-${field.name}`;
  return (
    <div className="wf-field">
      <label htmlFor={id}>{label}</label>
      {field.kind === "textarea" ? (
        <textarea id={id} rows={3} value={v} onChange={(e) => onChange(e.target.value)} />
      ) : field.kind === "enum" ? (
        <select id={id} value={v} onChange={(e) => onChange(e.target.value)}>
          <option value="">Selecione…</option>
          {(field.options ?? []).map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
      ) : (
        <input
          id={id}
          type={field.kind === "int" ? "number" : "text"}
          value={v}
          min={field.min}
          max={field.max}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
      {field.hint && <small className="muted">{field.hint}</small>}
      {issues.map((i) => (
        <small key={i.code} className="form-error" role="alert">
          {i.message}
        </small>
      ))}
    </div>
  );
}
