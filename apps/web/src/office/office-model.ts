// Modelo puro do escritório virtual (FAC-042): layout em grade, caminhos e
// comportamento dos agentes. Sem DOM nem canvas, para ser testável e reaproveitável.

export type OfficeActivity =
  | { kind: "idle" }
  | { kind: "working"; task?: string }
  | { kind: "handoff"; toAgentId: string; task?: string };

/** Entrada que qualquer fonte (mock hoje, eventos reais depois) entrega ao escritório. */
export interface OfficeAgentInput {
  id: string;
  name: string;
  role: string;
  activity: OfficeActivity;
}

export interface Tile {
  x: number;
  y: number;
}

export type FurnitureKind = "desk" | "coffee" | "sofa" | "shelf" | "plant" | "cooler";

export interface Furniture {
  kind: FurnitureKind;
  x: number;
  y: number;
  deskIndex?: number;
}

export interface DeskSlot {
  desk: Tile;
  seat: Tile;
  /** Ponto onde outro agente para ao entregar um trabalho. */
  visitor: Tile;
}

export interface OfficeLayout {
  width: number;
  height: number;
  blocked: boolean[];
  desks: DeskSlot[];
  coffeeSpots: Tile[];
  readingSpots: Tile[];
  furniture: Furniture[];
}

export type Facing = "NE" | "NW" | "SE" | "SW";
export type Pose = "stand" | "walk" | "type" | "read" | "drink" | "deliver";

type Goal =
  | { type: "seat" }
  | { type: "wander"; tile: Tile }
  | { type: "coffee"; tile: Tile }
  | { type: "read"; tile: Tile }
  | { type: "deliver"; toAgentId: string; tile: Tile };

export interface AgentSim {
  id: string;
  name: string;
  role: string;
  deskIndex: number;
  activity: OfficeActivity;
  /** Posição contínua no espaço de tiles (centro do tile = n + 0.5). */
  x: number;
  y: number;
  path: Tile[];
  goal: Goal | null;
  pose: Pose;
  facing: Facing;
  /** Segundos restantes na pose atual (ocioso/entrega). */
  timer: number;
  /** Fase de animação contínua (passos, digitação). */
  phase: number;
  handoffDelivered: boolean;
}

export interface OfficeSim {
  layout: OfficeLayout;
  agents: AgentSim[];
  random: () => number;
}

export const WALK_SPEED = 2.1; // tiles por segundo

const key = (layout: OfficeLayout, t: Tile) => t.y * layout.width + t.x;
const same = (a: Tile, b: Tile) => a.x === b.x && a.y === b.y;

export function isWalkable(layout: OfficeLayout, t: Tile): boolean {
  return (
    t.x >= 0 &&
    t.y >= 0 &&
    t.x < layout.width &&
    t.y < layout.height &&
    !layout.blocked[key(layout, t)]
  );
}

/** Escritório padrão: 10 mesas em duas fileiras, copa, leitura e plantas. */
export function createDefaultLayout(): OfficeLayout {
  const width = 15;
  const height = 11;
  const blocked = new Array<boolean>(width * height).fill(false);
  const furniture: Furniture[] = [];
  const desks: DeskSlot[] = [];
  const block = (x: number, y: number) => {
    blocked[y * width + x] = true;
  };
  // Faixa encostada nas paredes do fundo não é caminhável (evita agentes atrás dos móveis).
  for (let x = 0; x < width; x++) block(x, 0);
  for (let y = 0; y < height; y++) block(0, y);

  for (const row of [2, 6]) {
    for (const x of [2, 4, 6, 8, 10]) {
      block(x, row);
      furniture.push({ kind: "desk", x, y: row, deskIndex: desks.length });
      desks.push({
        desk: { x, y: row },
        seat: { x, y: row + 1 },
        visitor: { x: x + 1, y: row + 1 },
      });
    }
  }
  const place = (kind: FurnitureKind, x: number, y: number) => {
    block(x, y);
    furniture.push({ kind, x, y });
  };
  place("coffee", 13, 1);
  place("cooler", 14, 1);
  place("shelf", 14, 5);
  place("shelf", 14, 6);
  // Sofás são caminháveis: o agente senta neles para ler.
  furniture.push({ kind: "sofa", x: 12, y: 9 }, { kind: "sofa", x: 13, y: 9 });
  place("plant", 1, 1);
  place("plant", 12, 1);
  place("plant", 1, 10);
  place("plant", 14, 10);
  return {
    width,
    height,
    blocked,
    desks,
    furniture,
    coffeeSpots: [
      { x: 13, y: 2 },
      { x: 14, y: 2 },
      { x: 12, y: 2 },
    ],
    readingSpots: [
      { x: 12, y: 9 },
      { x: 13, y: 9 },
      { x: 12, y: 5 },
    ],
  };
}

/** BFS em 4 direções. Retorna os tiles depois de `from` até `to`, ou null. */
export function findPath(layout: OfficeLayout, from: Tile, to: Tile): Tile[] | null {
  if (same(from, to)) return [];
  if (!isWalkable(layout, to)) return null;
  const prev = new Int32Array(layout.width * layout.height).fill(-1);
  const start = key(layout, from);
  prev[start] = start;
  const queue = [start];
  const goal = key(layout, to);
  for (let head = 0; head < queue.length; head++) {
    const current = queue[head] as number;
    if (current === goal) break;
    const cx = current % layout.width;
    const cy = Math.floor(current / layout.width);
    for (const [dx, dy] of [
      [1, 0],
      [-1, 0],
      [0, 1],
      [0, -1],
    ] as const) {
      const next = { x: cx + dx, y: cy + dy };
      if (!isWalkable(layout, next)) continue;
      const k = key(layout, next);
      if (prev[k] !== -1) continue;
      prev[k] = current;
      queue.push(k);
    }
  }
  if (prev[goal] === -1) return null;
  const path: Tile[] = [];
  for (let k = goal; k !== start; k = prev[k] as number) {
    path.push({ x: k % layout.width, y: Math.floor(k / layout.width) });
  }
  return path.reverse();
}

/** Gerador determinístico (mulberry32): mesma semente, mesmo escritório. */
export function seededRandom(seed: number): () => number {
  let a = seed >>> 0;
  return () => {
    a = (a + 0x6d2b79f5) >>> 0;
    let t = a;
    t = Math.imul(t ^ (t >>> 15), t | 1);
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
}

export function tileOf(agent: AgentSim): Tile {
  return { x: Math.floor(agent.x), y: Math.floor(agent.y) };
}

export function createOfficeSim(
  layout: OfficeLayout,
  inputs: OfficeAgentInput[],
  random: () => number = Math.random,
): OfficeSim {
  const agents = inputs.slice(0, layout.desks.length).map((input, index): AgentSim => {
    const seat = (layout.desks[index] as DeskSlot).seat;
    return {
      id: input.id,
      name: input.name,
      role: input.role,
      deskIndex: index,
      activity: input.activity,
      x: seat.x + 0.5,
      y: seat.y + 0.5,
      path: [],
      goal: null,
      pose: "stand",
      facing: "NE",
      timer: random() * 2,
      phase: random() * 10,
      handoffDelivered: false,
    };
  });
  return { layout, agents, random };
}

/** Aplica um novo retrato vindo da fonte de dados, preservando posição/animação. */
export function applyActivities(sim: OfficeSim, inputs: OfficeAgentInput[]): void {
  for (const input of inputs) {
    const agent = sim.agents.find((item) => item.id === input.id);
    if (!agent) continue;
    agent.name = input.name;
    agent.role = input.role;
    if (JSON.stringify(agent.activity) === JSON.stringify(input.activity)) continue;
    agent.activity = input.activity;
    agent.goal = null;
    agent.path = [];
    agent.timer = 0;
    agent.handoffDelivered = false;
  }
}

function seatOf(sim: OfficeSim, agent: AgentSim): Tile {
  return (sim.layout.desks[agent.deskIndex] as DeskSlot).seat;
}

function reservedTiles(sim: OfficeSim, except: AgentSim): Tile[] {
  const tiles: Tile[] = [];
  for (const other of sim.agents) {
    if (other === except) continue;
    if (other.goal && other.goal.type !== "seat") tiles.push(other.goal.tile);
    tiles.push(tileOf(other));
  }
  return tiles;
}

function pickFree(sim: OfficeSim, agent: AgentSim, candidates: Tile[]): Tile | null {
  const taken = reservedTiles(sim, agent);
  const free = candidates.filter((tile) => !taken.some((other) => same(other, tile)));
  if (!free.length) return null;
  return free[Math.floor(sim.random() * free.length)] as Tile;
}

function randomFloorTile(sim: OfficeSim, agent: AgentSim): Tile | null {
  const { layout } = sim;
  const seats = layout.desks.map((slot) => slot.seat);
  for (let attempt = 0; attempt < 30; attempt++) {
    const tile = {
      x: 1 + Math.floor(sim.random() * (layout.width - 1)),
      y: 1 + Math.floor(sim.random() * (layout.height - 1)),
    };
    if (!isWalkable(layout, tile) || seats.some((seat) => same(seat, tile))) continue;
    if (reservedTiles(sim, agent).some((other) => same(other, tile))) continue;
    return tile;
  }
  return null;
}

function setGoal(sim: OfficeSim, agent: AgentSim, goal: Goal | null): void {
  agent.goal = goal;
  if (!goal) {
    agent.path = [];
    return;
  }
  const target = goal.type === "seat" ? seatOf(sim, agent) : goal.tile;
  agent.path = findPath(sim.layout, tileOf(agent), target) ?? [];
}

function arrived(sim: OfficeSim, agent: AgentSim): boolean {
  if (!agent.goal || agent.path.length) return false;
  const target = agent.goal.type === "seat" ? seatOf(sim, agent) : agent.goal.tile;
  return same(tileOf(agent), target);
}

function decide(sim: OfficeSim, agent: AgentSim): void {
  const activity = agent.activity;
  if (activity.kind === "working") {
    if (!agent.goal) setGoal(sim, agent, { type: "seat" });
    if (arrived(sim, agent)) {
      agent.pose = "type";
      agent.facing = "NE";
    }
    return;
  }
  if (activity.kind === "handoff" && !agent.handoffDelivered) {
    const target = sim.agents.find((other) => other.id === activity.toAgentId);
    if (!target) {
      agent.handoffDelivered = true;
      return;
    }
    const visitor = (sim.layout.desks[target.deskIndex] as DeskSlot).visitor;
    if (agent.goal?.type !== "deliver") {
      setGoal(sim, agent, { type: "deliver", toAgentId: target.id, tile: visitor });
    }
    if (arrived(sim, agent)) {
      if (agent.pose !== "deliver") {
        agent.pose = "deliver";
        agent.facing = "NW";
        agent.timer = 2.6;
      } else if (agent.timer <= 0) {
        agent.handoffDelivered = true;
        setGoal(sim, agent, { type: "seat" });
      }
    }
    return;
  }
  // Ocioso (ou entrega concluída): passeia, toma café ou lê.
  if (agent.goal && !arrived(sim, agent)) return;
  if (agent.goal && arrived(sim, agent) && agent.pose !== "walk" && agent.timer > 0) return;
  if (agent.goal && arrived(sim, agent) && agent.pose === "walk") {
    const goal = agent.goal;
    if (goal.type === "coffee") {
      agent.pose = "drink";
      agent.facing = "NE";
      agent.timer = 6 + sim.random() * 5;
    } else if (goal.type === "read") {
      agent.pose = "read";
      agent.facing = "SW";
      agent.timer = 8 + sim.random() * 7;
    } else {
      agent.pose = "stand";
      agent.timer = 1.5 + sim.random() * 3;
    }
    return;
  }
  const roll = sim.random();
  const coffee = roll < 0.35 ? pickFree(sim, agent, sim.layout.coffeeSpots) : null;
  const read = roll >= 0.35 && roll < 0.65 ? pickFree(sim, agent, sim.layout.readingSpots) : null;
  if (coffee) setGoal(sim, agent, { type: "coffee", tile: coffee });
  else if (read) setGoal(sim, agent, { type: "read", tile: read });
  else {
    const tile = randomFloorTile(sim, agent);
    if (tile) setGoal(sim, agent, { type: "wander", tile });
    else agent.timer = 1;
  }
  if (agent.path.length) agent.pose = "walk";
}

function facingFor(dx: number, dy: number, current: Facing): Facing {
  if (dx > 0) return "SE";
  if (dx < 0) return "NW";
  if (dy > 0) return "SW";
  if (dy < 0) return "NE";
  return current;
}

/** Avança a simulação `dt` segundos. */
export function stepOffice(sim: OfficeSim, dt: number): void {
  for (const agent of sim.agents) {
    agent.phase += dt;
    if (agent.timer > 0) agent.timer -= dt;
    decide(sim, agent);
    const next = agent.path[0];
    if (!next) continue;
    agent.pose = "walk";
    const tx = next.x + 0.5;
    const ty = next.y + 0.5;
    const dx = tx - agent.x;
    const dy = ty - agent.y;
    const distance = Math.hypot(dx, dy);
    const stepLength = WALK_SPEED * dt;
    agent.facing = facingFor(
      Math.sign(Math.round(dx * 100)),
      Math.sign(Math.round(dy * 100)),
      agent.facing,
    );
    if (distance <= stepLength) {
      agent.x = tx;
      agent.y = ty;
      agent.path.shift();
    } else {
      agent.x += (dx / distance) * stepLength;
      agent.y += (dy / distance) * stepLength;
    }
  }
}

/** Leva todos ao estado final imediatamente (movimento reduzido). */
export function settleOffice(sim: OfficeSim, maxSteps = 600): void {
  for (let i = 0; i < maxSteps; i++) {
    stepOffice(sim, 0.1);
    if (sim.agents.every((agent) => !agent.path.length)) break;
  }
}

export function describeActivity(input: OfficeAgentInput, all: OfficeAgentInput[]): string {
  const activity = input.activity;
  if (activity.kind === "working")
    return activity.task ? `Trabalhando em ${activity.task}` : "Trabalhando";
  if (activity.kind === "handoff") {
    const target = all.find((agent) => agent.id === activity.toAgentId);
    const what = activity.task ? `${activity.task} ` : "trabalho ";
    return `Entregando ${what}para ${target?.name ?? "outro agente"}`;
  }
  return "Livre · café, leitura ou circulando";
}
