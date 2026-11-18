import { describe, expect, it } from "vitest";
import type { WorkflowGraph } from "../../lib/api";
import {
  autoLayout,
  canConnect,
  freeSpot,
  hasOverlap,
  nextEdgeId,
  nextNodeId,
  nodeSummary,
  toFlow,
  toGraph,
} from "./flowModel";

const graph: WorkflowGraph = {
  nodes: [
    { id: "inicio", type: "start", x: -40, y: 20, props: {} },
    { id: "ag", type: "agent", x: 300, y: 20, props: { agent: "dev", prompt: "x" } },
    { id: "fim", type: "end", x: 600, y: 20, props: { outcome: "success" } },
    { id: "falha", type: "end", x: 600, y: 200, props: { outcome: "failure" } },
  ],
  edges: [
    { id: "e1", from: "inicio", from_port: "out", to: "ag", to_port: "in" },
    { id: "e2", from: "ag", from_port: "out", to: "fim", to_port: "in" },
    { id: "e3", from: "ag", from_port: "error", to: "falha", to_port: "in" },
  ],
};

describe("modelo do canvas", () => {
  it("converte o grafo para o canvas e de volta sem perder nada (inclusive coordenadas negativas)", () => {
    const f = toFlow(graph);
    expect(toGraph(f.nodes, f.edges)).toEqual(graph);
  });

  it("arredonda as coordenadas ao salvar", () => {
    const f = toFlow(graph);
    f.nodes[0] = { ...f.nodes[0], position: { x: 10.6, y: 3.2 } } as (typeof f.nodes)[number];
    expect(toGraph(f.nodes, f.edges).nodes[0]).toMatchObject({ x: 11, y: 3 });
  });

  it("marca as conexões de erro com o tom de erro", () => {
    const f = toFlow(graph);
    expect(f.edges.find((e) => e.id === "e3")?.className).toContain("wf-edge--bad");
    expect(f.edges.find((e) => e.id === "e1")?.className).toContain("wf-edge--plain");
  });

  it("organiza da esquerda para a direita pelo caminho mais longo", () => {
    const f = toFlow(graph);
    const laid = autoLayout(f.nodes, f.edges);
    const x = (id: string) => laid.find((n) => n.id === id)?.position.x ?? -1;
    expect(x("inicio")).toBeLessThan(x("ag"));
    expect(x("ag")).toBeLessThan(x("fim"));
    expect(x("fim")).toBe(x("falha"));
    const y = (id: string) => laid.find((n) => n.id === id)?.position.y ?? -1;
    expect(y("fim")).not.toBe(y("falha"));
  });

  it("não trava com ciclos", () => {
    const g: WorkflowGraph = {
      nodes: [
        { id: "a", type: "loop", x: 0, y: 0, props: {} },
        { id: "b", type: "agent", x: 0, y: 0, props: {} },
      ],
      edges: [
        { id: "e1", from: "a", from_port: "body", to: "b", to_port: "in" },
        { id: "e2", from: "b", from_port: "out", to: "a", to_port: "in" },
      ],
    };
    const f = toFlow(g);
    expect(autoLayout(f.nodes, f.edges)).toHaveLength(2);
  });

  it("gera IDs livres para nós e conexões", () => {
    expect(nextNodeId([{ id: "agent-1" }, { id: "agent-3" }], "agent")).toBe("agent-4");
    expect(nextNodeId([], "agent")).toBe("agent-1");
    expect(nextEdgeId([{ id: "e1" }, { id: "e2" }])).toBe("e3");
  });

  it("recusa ligar um nó a si mesmo e conexão duplicada", () => {
    const f = toFlow(graph);
    const c = { source: "inicio", sourceHandle: "out", target: "ag", targetHandle: "in" };
    expect(canConnect(f.edges, c)).toBe(false);
    expect(canConnect(f.edges, { ...c, target: "inicio", targetHandle: "in" })).toBe(false);
    expect(canConnect(f.edges, { ...c, target: "fim" })).toBe(true);
  });

  it("resume o que cada nó faz e avisa quando falta preencher", () => {
    expect(nodeSummary("agent", { agent: "tech-lead" })).toBe("tech-lead");
    expect(nodeSummary("agent", {})).toBe("Escolha o agente");
    expect(nodeSummary("checks", { commands: "make lint\nmake test" })).toBe("make lint");
    expect(nodeSummary("wait", { minutes: 1440 })).toBe("1440 min");
    expect(nodeSummary("end", { outcome: "failure" })).toBe("Termina com falha");
  });

  it("o nó novo procura um lugar livre em vez de nascer em cima de outro", () => {
    const nodes = [{ position: { x: 100, y: 100 } }];
    expect(freeSpot([], { x: 100, y: 100 })).toEqual({ x: 100, y: 100 });
    const spot = freeSpot(nodes, { x: 100, y: 100 });
    expect(Math.abs(spot.x - 100) >= 240 || Math.abs(spot.y - 100) >= 130).toBe(true);
  });

  it("detecta cartões sobrepostos e a organização os separa", () => {
    const g: WorkflowGraph = {
      nodes: [
        { id: "a", type: "start", x: 0, y: 0, props: {} },
        { id: "b", type: "end", x: 180, y: 0, props: { outcome: "success" } }, // 180 px: cartão tem 240
      ],
      edges: [{ id: "e1", from: "a", from_port: "out", to: "b", to_port: "in" }],
    };
    const f = toFlow(g);
    expect(hasOverlap(f.nodes)).toBe(true);
    expect(hasOverlap(autoLayout(f.nodes, f.edges))).toBe(false);
  });
});
