import { describe, expect, it, vi } from "vitest";
import { createMockOfficeFeed, demoOfficeAgents } from "./office-feed";
import {
  applyActivities,
  createDefaultLayout,
  createOfficeSim,
  describeActivity,
  findPath,
  isWalkable,
  seededRandom,
  stepOffice,
  tileOf,
} from "./office-model";

const run = (sim: ReturnType<typeof createOfficeSim>, seconds: number) => {
  for (let t = 0; t < seconds; t += 0.05) stepOffice(sim, 0.05);
};

describe("layout e caminhos", () => {
  it("tem 10 mesas com cadeira e ponto de visita caminháveis", () => {
    const layout = createDefaultLayout();
    expect(layout.desks).toHaveLength(10);
    for (const slot of layout.desks) {
      expect(isWalkable(layout, slot.desk)).toBe(false);
      expect(isWalkable(layout, slot.seat)).toBe(true);
      expect(isWalkable(layout, slot.visitor)).toBe(true);
    }
  });

  it("encontra caminho contornando mesas e nunca pisa em tile bloqueado", () => {
    const layout = createDefaultLayout();
    const path = findPath(layout, { x: 2, y: 3 }, { x: 13, y: 2 });
    expect(path).not.toBeNull();
    expect(path?.at(-1)).toEqual({ x: 13, y: 2 });
    for (const tile of path ?? []) expect(isWalkable(layout, tile)).toBe(true);
    expect(findPath(layout, { x: 2, y: 3 }, { x: 2, y: 2 })).toBeNull();
  });
});

describe("comportamento dos agentes", () => {
  it("quem trabalha volta para a própria mesa e digita", () => {
    const layout = createDefaultLayout();
    const sim = createOfficeSim(
      layout,
      [{ id: "a", name: "A", role: "Dev", activity: { kind: "idle" } }],
      seededRandom(1),
    );
    run(sim, 6);
    applyActivities(sim, [
      { id: "a", name: "A", role: "Dev", activity: { kind: "working", task: "X" } },
    ]);
    run(sim, 20);
    const [agent] = sim.agents;
    if (!agent) throw new Error("agente ausente");
    expect(tileOf(agent)).toEqual(layout.desks[0]?.seat);
    expect(agent.pose).toBe("type");
  });

  it("entrega: vai até a mesa do outro agente, entrega e volta", () => {
    const layout = createDefaultLayout();
    const sim = createOfficeSim(
      layout,
      [
        {
          id: "a",
          name: "A",
          role: "Dev",
          activity: { kind: "handoff", toAgentId: "b", task: "T" },
        },
        { id: "b", name: "B", role: "QA", activity: { kind: "working" } },
      ],
      seededRandom(2),
    );
    let reached = false;
    for (let t = 0; t < 30 && !sim.agents[0]?.handoffDelivered; t += 0.05) {
      stepOffice(sim, 0.05);
      if (sim.agents[0]?.pose === "deliver") {
        reached = true;
        expect(tileOf(sim.agents[0])).toEqual(layout.desks[1]?.visitor);
      }
    }
    expect(reached).toBe(true);
    expect(sim.agents[0]?.handoffDelivered).toBe(true);
  });

  it("ocioso nunca fica parado em mesa e alterna entre circular, café e leitura", () => {
    const layout = createDefaultLayout();
    const sim = createOfficeSim(
      layout,
      demoOfficeAgents.map((agent) => ({ ...agent, activity: { kind: "idle" as const } })),
      seededRandom(3),
    );
    const poses = new Set<string>();
    for (let t = 0; t < 240; t += 0.05) {
      stepOffice(sim, 0.05);
      for (const agent of sim.agents) {
        poses.add(agent.pose);
        expect(isWalkable(layout, tileOf(agent))).toBe(true);
      }
    }
    expect(poses.has("walk")).toBe(true);
    expect(poses.has("drink") || poses.has("read")).toBe(true);
    expect(poses.has("type")).toBe(false);
  });

  it("descreve a atividade em português", () => {
    const agents = [
      {
        id: "a",
        name: "Alex",
        role: "Dev",
        activity: { kind: "handoff" as const, toAgentId: "b", task: "T-1" },
      },
      {
        id: "b",
        name: "Sofia",
        role: "Designer",
        activity: { kind: "working" as const, task: "T-2" },
      },
    ];
    const [alex, sofia] = agents;
    if (!alex || !sofia) throw new Error("agentes ausentes");
    expect(describeActivity(alex, agents)).toBe("Entregando T-1 para Sofia");
    expect(describeActivity(sofia, agents)).toBe("Trabalhando em T-2");
  });
});

describe("fonte simulada", () => {
  it("só agenda timer enquanto houver assinante e muda atividades", () => {
    vi.useFakeTimers();
    const feed = createMockOfficeFeed(demoOfficeAgents, { intervalMs: 100, seed: 5 });
    const before = JSON.stringify(feed.snapshot());
    const listener = vi.fn();
    const stop = feed.subscribe(listener);
    vi.advanceTimersByTime(1000);
    expect(listener).toHaveBeenCalled();
    expect(JSON.stringify(feed.snapshot())).not.toBe(before);
    stop();
    expect(vi.getTimerCount()).toBe(0);
    vi.useRealTimers();
  });
});
