import { type Edge, MarkerType, type Node } from "@xyflow/react";
import type { NodeProps, NodeType, WorkflowEdge, WorkflowGraph, WorkflowNode } from "../../lib/api";

/** Dados de um nó no canvas: só o que é do fluxo (o resto vem do catálogo de tipos e da validação). */
export interface WfNodeData extends Record<string, unknown> {
  wfType: string;
  props: NodeProps;
}

export type WfNode = Node<WfNodeData, "wf">;
export type WfEdge = Edge;

export const GRID = 20;
export const NODE_WIDTH = 240;

/** Rótulo curto do que o nó faz, para mostrar no cartão sem abrir o painel. */
export function nodeSummary(type: string, props: NodeProps): string {
  const text = (k: string) => (typeof props[k] === "string" ? String(props[k]).trim() : "");
  const first = (s: string) => s.split("\n")[0] ?? "";
  switch (type) {
    case "agent":
      return text("agent") || "Escolha o agente";
    case "skill":
      return text("skill") || "Escolha a skill";
    case "script":
      return first(text("command")) || "Informe o comando";
    case "checks":
      return first(text("commands")) || "Informe os comandos";
    case "git":
      return (
        [text("operation"), text("message")].filter(Boolean).join(" · ") || "Escolha a operação"
      );
    case "docs":
      return text("target") || "Informe o alvo";
    case "condition":
      return text("expression") || "Informe a expressão";
    case "loop":
      return text("condition") || "Informe a condição";
    case "wait":
      return props.minutes ? `${props.minutes} min` : "Informe o tempo";
    case "end":
      return text("outcome") === "failure" ? "Termina com falha" : "Termina com sucesso";
    default:
      return "";
  }
}

const markerEnd = { type: MarkerType.ArrowClosed, width: 18, height: 18 } as const;

export function toFlowNode(n: WorkflowNode): WfNode {
  return {
    id: n.id,
    type: "wf",
    position: { x: n.x, y: n.y },
    data: { wfType: n.type, props: n.props ?? {} },
  };
}

export function toFlowEdge(e: WorkflowEdge): WfEdge {
  return {
    id: e.id,
    source: e.from,
    sourceHandle: e.from_port,
    target: e.to,
    targetHandle: e.to_port,
    markerEnd,
    className: `wf-edge wf-edge--${edgeTone(e.from_port)}`,
  };
}

/** Cor da conexão pela saída de onde sai (erro, ramo verdadeiro/falso, etc.). */
export function edgeTone(port: string): string {
  switch (port) {
    case "error":
    case "false":
    case "rejected":
      return "bad";
    case "true":
    case "approved":
    case "done":
      return "good";
    default:
      return "plain";
  }
}

export function toFlow(graph: WorkflowGraph): { nodes: WfNode[]; edges: WfEdge[] } {
  return { nodes: graph.nodes.map(toFlowNode), edges: graph.edges.map(toFlowEdge) };
}

export function toGraph(nodes: WfNode[], edges: WfEdge[]): WorkflowGraph {
  return {
    nodes: nodes.map((n) => ({
      id: n.id,
      type: n.data.wfType,
      x: Math.round(n.position.x),
      y: Math.round(n.position.y),
      props: n.data.props,
    })),
    edges: edges.map((e) => ({
      id: e.id,
      from: e.source,
      from_port: e.sourceHandle ?? "out",
      to: e.target,
      to_port: e.targetHandle ?? "in",
    })),
  };
}

/** ID livre no formato `<tipo>-<n>` (o mesmo que o editor antigo usava). */
export function nextNodeId(nodes: { id: string }[], type: string): string {
  let n = nodes.filter((x) => x.id.startsWith(`${type}-`)).length + 1;
  while (nodes.some((x) => x.id === `${type}-${n}`)) n++;
  return `${type}-${n}`;
}

export function nextEdgeId(edges: { id: string }[]): string {
  let n = edges.length + 1;
  while (edges.some((e) => e.id === `e${n}`)) n++;
  return `e${n}`;
}

/**
 * Organiza o fluxo da esquerda para a direita: a coluna de cada nó é o caminho mais longo desde o início
 * (ciclos são ignorados), e nós da mesma coluna se empilham na ordem em que aparecem.
 */
export function autoLayout(nodes: WfNode[], edges: WfEdge[]): WfNode[] {
  const depth = new Map<string, number>();
  const incoming = new Map<string, number>();
  for (const n of nodes) incoming.set(n.id, 0);
  for (const e of edges) incoming.set(e.target, (incoming.get(e.target) ?? 0) + 1);
  const queue = nodes.filter((n) => (incoming.get(n.id) ?? 0) === 0).map((n) => n.id);
  for (const id of queue) depth.set(id, 0);
  const remaining = new Map(incoming);
  while (queue.length > 0) {
    const id = queue.shift() as string;
    for (const e of edges.filter((x) => x.source === id)) {
      depth.set(e.target, Math.max(depth.get(e.target) ?? 0, (depth.get(id) ?? 0) + 1));
      remaining.set(e.target, (remaining.get(e.target) ?? 1) - 1);
      if ((remaining.get(e.target) ?? 0) <= 0) queue.push(e.target);
    }
  }
  const columns = new Map<number, string[]>();
  for (const n of nodes) {
    const d = depth.get(n.id) ?? 0; // nós em ciclo ficam na coluna 0
    columns.set(d, [...(columns.get(d) ?? []), n.id]);
  }
  const pos = new Map<string, { x: number; y: number }>();
  for (const [d, ids] of columns) {
    for (const [i, id] of ids.entries())
      pos.set(id, { x: 40 + d * (NODE_WIDTH + 80), y: 40 + i * 180 });
  }
  return nodes.map((n) => ({ ...n, position: pos.get(n.id) ?? n.position }));
}

/** Pode ligar essa saída a essa entrada? (mesmas regras de fácil verificação do servidor) */
export function canConnect(
  edges: WfEdge[],
  c: { source: string; sourceHandle: string | null; target: string; targetHandle: string | null },
): boolean {
  if (c.source === c.target) return false;
  return !edges.some(
    (e) =>
      e.source === c.source &&
      e.sourceHandle === c.sourceHandle &&
      e.target === c.target &&
      e.targetHandle === c.targetHandle,
  );
}

export function typeMapOf(types: NodeType[]): Map<string, NodeType> {
  return new Map(types.map((t) => [t.type, t]));
}

const CARD_H = 130;

/** Dois cartões se sobrepõem? (o cartão tem NODE_WIDTH de largura e ~CARD_H de altura) */
function overlaps(a: { x: number; y: number }, b: { x: number; y: number }, gap = 0) {
  return Math.abs(a.x - b.x) < NODE_WIDTH + gap && Math.abs(a.y - b.y) < CARD_H + gap;
}

/** Algum par de nós se sobrepõe? Fluxos salvos pelo editor antigo (cartões de 190 px) costumam. */
export function hasOverlap(nodes: { position: { x: number; y: number } }[]): boolean {
  return nodes.some((a, i) => nodes.slice(i + 1).some((b) => overlaps(a.position, b.position)));
}

/** Lugar livre perto do ponto pedido (e à vista): tenta o ponto, depois ao redor, e por fim desce. */
export function freeSpot(
  nodes: { position: { x: number; y: number } }[],
  from: { x: number; y: number },
) {
  const busy = (p: { x: number; y: number }) => nodes.some((n) => overlaps(n.position, p, 20));
  const candidates: { x: number; y: number }[] = [];
  for (const dy of [0, 160, -160, 320, -320]) {
    for (const dx of [0, 280, -280]) candidates.push({ x: from.x + dx, y: from.y + dy });
  }
  const free = candidates.find((p) => !busy(p));
  if (free) return free;
  let p = { ...from };
  for (let i = 0; i < 40 && busy(p); i++) p = { x: from.x, y: from.y + (i + 4) * 160 };
  return p;
}
