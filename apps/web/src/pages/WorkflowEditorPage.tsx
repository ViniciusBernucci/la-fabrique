import {
  Background,
  BackgroundVariant,
  type Connection,
  ConnectionLineType,
  Controls,
  MiniMap,
  type OnEdgesChange,
  type OnNodesChange,
  Panel,
  ReactFlow,
  ReactFlowProvider,
  useReactFlow,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import { type DragEvent, useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { ErrorState, LoadingState } from "../components/States";
import {
  type ApiError,
  api,
  type EffectiveAgent,
  type FieldSpec,
  type NodeType,
  type Workflow,
  type WorkflowIssue,
} from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";
import { EditorContext, FlowNodeCard, nodeIcon } from "./workflow/FlowNode";
import {
  autoLayout,
  canConnect,
  freeSpot,
  GRID,
  hasOverlap,
  nextEdgeId,
  nextNodeId,
  toFlow,
  toFlowEdge,
  toGraph,
  typeMapOf,
  type WfEdge,
  type WfNode,
} from "./workflow/flowModel";

const DRAG_MIME = "application/x-lf-node-type";
const nodeTypesMap = { wf: FlowNodeCard };

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
  return (
    <ReactFlowProvider>
      <Editor initial={wf.state.data} nodeTypes={types.state.data.items} />
    </ReactFlowProvider>
  );
}

function Editor({ initial, nodeTypes }: { initial: Workflow; nodeTypes: NodeType[] }) {
  const { expire } = useSession();
  const flow = useReactFlow();
  const typeMap = useMemo(() => typeMapOf(nodeTypes), [nodeTypes]);
  // Fluxos salvos pelo editor antigo têm cartões menores e podem chegar sobrepostos: a posição é
  // reorganizada ao abrir e vale como ponto de partida (só a posição; o conteúdo não muda).
  const start = useMemo(() => {
    const f = toFlow(initial.graph);
    return hasOverlap(f.nodes) ? { ...f, nodes: autoLayout(f.nodes, f.edges) } : f;
  }, [initial.graph]);
  const [nodes, setNodes] = useState<WfNode[]>(start.nodes);
  // Fluxos longos abrem no começo (legível), não encolhidos para caber inteiros; "Ajustar à tela" mostra tudo.
  const firstView = useMemo(() => {
    const minX = Math.min(...start.nodes.map((n) => n.position.x));
    const head = start.nodes.filter((n) => n.position.x <= minX + 1000);
    return (head.length > 0 ? head : start.nodes).map((n) => ({ id: n.id }));
  }, [start.nodes]);
  const [edges, setEdges] = useState<WfEdge[]>(start.edges);
  const [name, setName] = useState(initial.name);
  const [description, setDescription] = useState(initial.description);
  const [version, setVersion] = useState(initial.version);
  const [issues, setIssues] = useState<WorkflowIssue[]>(initial.issues);
  const graph = useMemo(() => toGraph(nodes, edges), [nodes, edges]);
  const graphKey = useMemo(() => JSON.stringify(graph), [graph]);
  const [saved, setSaved] = useState({
    key: JSON.stringify(toGraph(start.nodes, start.edges)),
    name: initial.name,
    description: initial.description,
  });
  const [justSaved, setJustSaved] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [agentNames, setAgentNames] = useState<string[]>([]);
  const [skillNames, setSkillNames] = useState<string[]>([]);
  const [effective, setEffective] = useState<{
    node: string;
    value?: EffectiveAgent;
    error?: ApiError;
  } | null>(null);
  const stage = useRef<HTMLDivElement>(null);

  const dirty = graphKey !== saved.key || name !== saved.name || description !== saved.description;
  const selectedNode = nodes.find((n) => n.selected);
  const selectedEdge = selectedNode ? undefined : edges.find((e) => e.selected);
  const selectedType = selectedNode ? typeMap.get(selectedNode.data.wfType) : undefined;
  const issueCount = useMemo(() => {
    const m = new Map<string, number>();
    for (const i of issues) if (i.node) m.set(i.node, (m.get(i.node) ?? 0) + 1);
    return m;
  }, [issues]);
  const ctx = useMemo(() => ({ types: typeMap, issues: issueCount }), [typeMap, issueCount]);

  // Agentes e skills do catálogo viram listas nos campos "Agente" e "Skill"; sem eles o campo segue livre.
  useEffect(() => {
    let alive = true;
    api
      .listAgents()
      .then((r) => alive && setAgentNames(r.items.map((a) => a.name)))
      .catch(() => {});
    api
      .listSkills()
      .then((r) => alive && setSkillNames(r.items.map((x) => x.name)))
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, []);

  // Validação no servidor (mesmas regras do salvamento), sem gravar.
  // biome-ignore lint/correctness/useExhaustiveDependencies: expire é estável; o efeito segue só o grafo.
  useEffect(() => {
    if (graphKey === saved.key) return;
    const controller = new AbortController();
    const timer = window.setTimeout(() => {
      api
        .validateWorkflowGraph(JSON.parse(graphKey), controller.signal)
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
  }, [graphKey]);

  const touch = () => setJustSaved(false);
  const onNodesChange: OnNodesChange<WfNode> = (changes) => {
    setNodes((ns) => changes.applyTo(ns));
    if (changes.toArray().some((c) => c.type === "position" || c.type === "remove")) touch();
  };
  const onEdgesChange: OnEdgesChange<WfEdge> = (changes) => {
    setEdges((es) => changes.applyTo(es));
    if (changes.toArray().some((c) => c.type === "remove")) touch();
  };

  function connectPorts(c: Connection) {
    if (!canConnect(edges, c)) return;
    touch();
    // Cada saída tem uma única conexão (o servidor recusa a bifurcação): ligar de novo substitui.
    setEdges((es) => [
      ...es.filter((e) => !(e.source === c.source && e.sourceHandle === c.sourceHandle)),
      toFlowEdge({
        id: nextEdgeId(es),
        from: c.source,
        from_port: c.sourceHandle ?? "out",
        to: c.target,
        to_port: c.targetHandle ?? "in",
      }),
    ]);
  }

  function addNodeAt(type: NodeType, at?: { x: number; y: number }) {
    touch();
    setNodes((ns) => {
      const box = stage.current?.getBoundingClientRect();
      const center = flow.screenToFlowPosition({
        x: (box?.left ?? 0) + (box?.width ?? 0) / 2,
        y: (box?.top ?? 0) + (box?.height ?? 0) / 2,
      });
      const base = at ?? freeSpot(ns, { x: center.x - 120, y: center.y - 40 });
      const node: WfNode = {
        id: nextNodeId(ns, type.type),
        type: "wf",
        position: { x: Math.round(base.x / GRID) * GRID, y: Math.round(base.y / GRID) * GRID },
        data: { wfType: type.type, props: {} },
        selected: true,
      };
      return [...ns.map((n) => ({ ...n, selected: false })), node];
    });
    setEdges((es) => es.map((e) => ({ ...e, selected: false })));
  }

  function onDrop(event: DragEvent) {
    event.preventDefault();
    const type = typeMap.get(event.dataTransfer.getData(DRAG_MIME));
    if (!type) return;
    const p = flow.screenToFlowPosition({ x: event.clientX, y: event.clientY });
    addNodeAt(type, { x: p.x - 120, y: p.y - 30 });
  }

  function removeSelection() {
    touch();
    const gone = new Set(nodes.filter((n) => n.selected).map((n) => n.id));
    setNodes((ns) => ns.filter((n) => !gone.has(n.id)));
    setEdges((es) => es.filter((e) => !e.selected && !gone.has(e.source) && !gone.has(e.target)));
  }

  function selectNode(id: string) {
    setNodes((ns) => ns.map((n) => ({ ...n, selected: n.id === id })));
    setEdges((es) => es.map((e) => ({ ...e, selected: false })));
    const n = nodes.find((x) => x.id === id);
    if (n)
      flow.setCenter(n.position.x + 120, n.position.y + 40, {
        zoom: flow.getZoom(),
        duration: 300,
      });
  }

  function setProp(nodeId: string, field: FieldSpec, raw: string) {
    touch();
    setNodes((ns) =>
      ns.map((n) => {
        if (n.id !== nodeId) return n;
        const props = { ...n.data.props };
        if (raw === "") delete props[field.name];
        else props[field.name] = field.kind === "int" ? Number(raw) : raw;
        return { ...n, data: { ...n.data, props } };
      }),
    );
  }

  /** Liga (ou desliga, com valor vazio) uma saída do nó escolhido; o alvo vem como "nó|entrada". */
  function wireOutput(nodeId: string, port: string, target: string) {
    touch();
    setEdges((es) => {
      const rest = es.filter((e) => !(e.source === nodeId && e.sourceHandle === port));
      if (!target) return rest;
      const [to, toPort] = target.split("|");
      return [
        ...rest,
        toFlowEdge({
          id: nextEdgeId(es),
          from: nodeId,
          from_port: port,
          to: to ?? "",
          to_port: toPort ?? "in",
        }),
      ];
    });
  }

  function organize() {
    touch();
    setNodes((ns) => autoLayout(ns, edges));
    window.setTimeout(() => flow.fitView({ padding: 0.2, maxZoom: 1, duration: 300 }), 30);
  }

  async function showEffective(nodeId: string) {
    try {
      setEffective({ node: nodeId, value: await api.nodeEffective(initial.id, nodeId) });
    } catch (e) {
      setEffective({ node: nodeId, error: toApiError(e) });
    }
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
      setSaved({ key: graphKey, name, description });
      setJustSaved(true);
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setSaving(false);
    }
  }

  const choicesFor = (type: string, field: string) =>
    type === "agent" && field === "agent"
      ? agentNames
      : type === "skill" && field === "skill"
        ? skillNames
        : undefined;

  const targets = (fromId: string) =>
    nodes
      .filter((n) => n.id !== fromId)
      .flatMap((n) =>
        (typeMap.get(n.data.wfType)?.inputs ?? []).map((p) => ({
          value: `${n.id}|${p}`,
          label: `${typeMap.get(n.data.wfType)?.label ?? n.data.wfType} (${n.id})${p === "in" ? "" : ` · ${p}`}`,
        })),
      );

  return (
    <EditorContext.Provider value={ctx}>
      <div className="page">
        <nav className="breadcrumb" aria-label="Trilha">
          <Link to="/workflows">Workflows</Link>
          <span aria-hidden="true">/</span>
          <span>{initial.name}</span>
        </nav>
        <PageHeader
          label={initial.kind === "feature" ? "Fluxo de feature" : "Subfluxo"}
          title={name}
          lead="Rascunho: não executa. Arraste os nós, puxe uma saída até uma entrada para conectar e clique num nó para editar."
          actions={
            <div className="actions">
              <span className={dirty ? "badge badge--paused" : "badge badge--active"}>
                {dirty ? "Alterações não salvas" : justSaved ? "Salvo" : `Versão ${version}`}
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
                touch();
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
                touch();
              }}
            />
          </label>
        </section>
        <section className="panel wf-stage" aria-label="Canvas do workflow">
          <aside className="wf-palette" aria-label="Paleta de nós">
            <div className="wf-palette-head">
              <strong>Adicionar nó</strong>
            </div>
            <p className="wf-palette-hint">Clique ou arraste para o canvas.</p>
            {(["fluxo", "execução", "controle"] as const).map((cat) => (
              <div key={cat} className="wf-palette-group">
                <p className="wf-palette-cat">{cat}</p>
                {nodeTypes
                  .filter((t) => t.category === cat)
                  .map((t) => (
                    <button
                      key={t.type}
                      type="button"
                      draggable
                      className={`wf-palette-item wf-palette-item--${cat}`}
                      onDragStart={(e) => {
                        e.dataTransfer.setData(DRAG_MIME, t.type);
                        e.dataTransfer.effectAllowed = "move";
                      }}
                      onClick={() => addNodeAt(t)}
                    >
                      <span className="wf-palette-icon">{nodeIcon(t.type)}</span>
                      {t.label}
                    </button>
                  ))}
              </div>
            ))}
          </aside>
          <div className="wf-flow-wrap" ref={stage}>
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypesMap}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={connectPorts}
              isValidConnection={(c) => c.source !== c.target}
              onDragOver={(e) => {
                e.preventDefault();
                e.dataTransfer.dropEffect = "move";
              }}
              onDrop={onDrop}
              fitView
              fitViewOptions={{ padding: 0.15, maxZoom: 1, nodes: firstView }}
              minZoom={0.1}
              maxZoom={2}
              snapToGrid
              snapGrid={[GRID, GRID]}
              panOnScroll
              selectionOnDrag={false}
              deleteKeyCode={["Backspace", "Delete"]}
              connectionRadius={36}
              connectionLineType={ConnectionLineType.Bezier}
              defaultEdgeOptions={{ type: "default" }}
              proOptions={{ hideAttribution: true }}
              className="wf-flow"
              aria-label="Grafo do workflow"
            >
              <Background variant={BackgroundVariant.Dots} gap={GRID} size={1.4} />
              <Controls showInteractive={false} position="bottom-left" />
              <MiniMap
                pannable
                zoomable
                position="bottom-right"
                nodeClassName={(n) =>
                  `wf-mini wf-mini--${typeMap.get(String(n.data?.wfType))?.category ?? "execução"}`
                }
              />
              <Panel position="top-center" className="wf-toolbar">
                <button type="button" className="secondary-action" onClick={organize}>
                  Organizar
                </button>
                <button
                  type="button"
                  className="secondary-action"
                  onClick={() => flow.fitView({ padding: 0.2, maxZoom: 1, duration: 300 })}
                >
                  Ajustar à tela
                </button>
                <button
                  type="button"
                  className="secondary-action"
                  disabled={!selectedNode && !selectedEdge}
                  onClick={removeSelection}
                >
                  Remover seleção
                </button>
              </Panel>
            </ReactFlow>
          </div>
          <aside className="wf-drawer" aria-label="Propriedades">
            {!selectedNode && !selectedEdge && (
              <div className="wf-drawer-empty">
                <h2 className="form-title">Propriedades</h2>
                <p className="muted">
                  Clique num nó para editar. Arraste a bolinha de uma saída até a entrada de outro
                  nó para conectar; role para mover o canvas e use Ctrl + rolagem para o zoom.
                </p>
              </div>
            )}
            {selectedEdge && (
              <div className="form">
                <h2 className="form-title">Conexão</h2>
                <p className="muted">
                  <code>
                    {selectedEdge.source}.{selectedEdge.sourceHandle}
                  </code>{" "}
                  →{" "}
                  <code>
                    {selectedEdge.target}.{selectedEdge.targetHandle}
                  </code>
                </p>
                <p className="muted">Conexão selecionada. Use “Remover seleção” para apagá-la.</p>
              </div>
            )}
            {selectedNode && selectedType && (
              <div className="form">
                <h2 className="form-title">{selectedType.label}</h2>
                <p className="muted">
                  <code>{selectedNode.id}</code>
                </p>
                {(selectedType.fields ?? []).length === 0 && (
                  <p className="muted">Este tipo de nó não tem propriedades.</p>
                )}
                {(selectedType.fields ?? []).map((f) => (
                  <FieldEditor
                    key={f.name}
                    field={f}
                    value={selectedNode.data.props[f.name]}
                    choices={choicesFor(selectedNode.data.wfType, f.name)}
                    issues={issues.filter((i) => i.node === selectedNode.id && i.field === f.name)}
                    onChange={(raw) => setProp(selectedNode.id, f, raw)}
                  />
                ))}
                {(selectedType.outputs ?? []).length > 0 && (
                  <fieldset className="wf-wires">
                    <legend>Saídas</legend>
                    {(selectedType.outputs ?? []).map((p) => {
                      const cur = edges.find(
                        (e) => e.source === selectedNode.id && e.sourceHandle === p,
                      );
                      return (
                        <label key={p} className="wf-field">
                          {p === "out" ? "Segue para" : `Se “${p}”, segue para`}
                          <select
                            aria-label={`Ligar saída ${p} a`}
                            value={cur ? `${cur.target}|${cur.targetHandle}` : ""}
                            onChange={(e) => wireOutput(selectedNode.id, p, e.target.value)}
                          >
                            <option value="">Sem conexão</option>
                            {targets(selectedNode.id).map((t) => (
                              <option key={t.value} value={t.value}>
                                {t.label}
                              </option>
                            ))}
                          </select>
                        </label>
                      );
                    })}
                  </fieldset>
                )}
                {selectedNode.data.wfType === "agent" && (
                  <div className="form">
                    <button
                      type="button"
                      className="secondary-action"
                      disabled={dirty}
                      title={dirty ? "Salve o rascunho para ver a configuração efetiva" : undefined}
                      onClick={() => showEffective(selectedNode.id)}
                    >
                      Ver configuração efetiva
                    </button>
                    {effective?.node === selectedNode.id && effective.error && (
                      <p className="form-error" role="alert">
                        {effective.error.message} {effective.error.action}
                      </p>
                    )}
                    {effective?.node === selectedNode.id && effective.value && (
                      <EffectiveView value={effective.value} />
                    )}
                  </div>
                )}
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
        </section>
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
                  {i.node && nodes.some((n) => n.id === i.node) && (
                    <button
                      type="button"
                      className="secondary-action"
                      onClick={() => selectNode(i.node)}
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
    </EditorContext.Provider>
  );
}

function FieldEditor({
  field,
  value,
  issues,
  choices,
  onChange,
}: {
  field: FieldSpec;
  value: string | number | undefined;
  issues: WorkflowIssue[];
  /** Valores do catálogo; com eles o campo vira uma lista (sem eles segue texto livre). */
  choices?: string[];
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
      ) : choices && choices.length > 0 ? (
        <select id={id} value={v} onChange={(e) => onChange(e.target.value)}>
          <option value="">Selecione…</option>
          {choices.map((c) => (
            <option key={c} value={c}>
              {c}
            </option>
          ))}
          {v !== "" && !choices.includes(v) && <option value={v}>{v} (fora do catálogo)</option>}
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

const sourceLabel = { agent: "do agente", node: "do nó" } as const;

function EffectiveView({ value }: { value: EffectiveAgent }) {
  const row = (label: string, key: string, text: string) => (
    <div>
      <dt>{label}</dt>
      <dd>
        {text || "—"}{" "}
        <small className="muted">({sourceLabel[value.sources[key] ?? "agent"]})</small>
      </dd>
    </div>
  );
  return (
    <dl className="facts" aria-label="Configuração efetiva">
      <div>
        <dt>Agente</dt>
        <dd>
          {value.agent_name} (versão {value.agent_version})
        </dd>
      </div>
      {row("Provedor", "provider", value.provider)}
      {row("Conta", "connection", value.connection_id ?? "")}
      {row("Modelo", "model", value.model)}
      {row("Ferramentas", "tools", value.tools.join(", "))}
      {row("Tempo máximo", "timeout_minutes", `${value.timeout_minutes} min`)}
    </dl>
  );
}
