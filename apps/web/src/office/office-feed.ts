import { type OfficeAgentInput, seededRandom } from "./office-model";

/**
 * Fonte de dados do escritório. Hoje há só a simulada; a fonte real (FAC posterior)
 * deve traduzir eventos de execução em `OfficeActivity` e respeitar este contrato.
 */
export interface OfficeFeed {
  snapshot(): OfficeAgentInput[];
  /** Assina mudanças; o retorno cancela a assinatura e libera timers. */
  subscribe(listener: (agents: OfficeAgentInput[]) => void): () => void;
}

export const demoOfficeAgents: OfficeAgentInput[] = [
  {
    id: "tech-lead",
    name: "Tech Lead",
    role: "Tech Lead",
    activity: { kind: "working", task: "DEMO-003" },
  },
  {
    id: "dev",
    name: "Alex",
    role: "Desenvolvedor",
    activity: { kind: "working", task: "DEMO-004" },
  },
  { id: "revisor", name: "Revisor", role: "Revisor", activity: { kind: "idle" } },
  { id: "qa", name: "QA", role: "QA", activity: { kind: "working", task: "DEMO-005" } },
  { id: "docs", name: "Documentação", role: "Documentação", activity: { kind: "idle" } },
  { id: "designer", name: "Sofia", role: "Designer", activity: { kind: "idle" } },
];

const demoTasks = ["DEMO-002", "DEMO-004", "DEMO-005", "DEMO-006", "DEMO-007", "DEMO-008"];

/**
 * Simulação local: a cada intervalo um agente muda de atividade. Só roda enquanto
 * houver assinante (o escritório aberto), sem timers em segundo plano.
 */
export function createMockOfficeFeed(
  initial: OfficeAgentInput[] = demoOfficeAgents,
  options: { intervalMs?: number; seed?: number } = {},
): OfficeFeed {
  const random = seededRandom(options.seed ?? 42);
  const intervalMs = options.intervalMs ?? 4500;
  let agents = initial.map((agent) => ({ ...agent }));
  const listeners = new Set<(agents: OfficeAgentInput[]) => void>();
  let timer: ReturnType<typeof setInterval> | undefined;
  const pending: { at: number; apply: () => void }[] = [];
  let tick = 0;

  const update = (id: string, activity: OfficeAgentInput["activity"]) => {
    agents = agents.map((agent) => (agent.id === id ? { ...agent, activity } : agent));
  };

  function advance() {
    tick++;
    for (const item of pending.filter((entry) => entry.at <= tick)) item.apply();
    for (let i = pending.length - 1; i >= 0; i--)
      if ((pending[i]?.at ?? 0) <= tick) pending.splice(i, 1);
    const busy = new Set(
      agents
        .filter((agent) => agent.activity.kind === "handoff")
        .flatMap((agent) =>
          agent.activity.kind === "handoff" ? [agent.id, agent.activity.toAgentId] : [],
        ),
    );
    const candidates = agents.filter((agent) => !busy.has(agent.id));
    const agent = candidates[Math.floor(random() * candidates.length)];
    if (!agent) return emit();
    const task = demoTasks[Math.floor(random() * demoTasks.length)] as string;
    const roll = random();
    if (agent.activity.kind === "idle") update(agent.id, { kind: "working", task });
    else if (roll < 0.45) {
      const others = candidates.filter((other) => other.id !== agent.id);
      const target = others[Math.floor(random() * others.length)];
      if (target) {
        const handed = agent.activity.kind === "working" ? (agent.activity.task ?? task) : task;
        update(agent.id, { kind: "handoff", toAgentId: target.id, task: handed });
        pending.push({
          at: tick + 3,
          apply: () => {
            update(agent.id, { kind: "idle" });
            update(target.id, { kind: "working", task: handed });
          },
        });
      }
    } else update(agent.id, { kind: "idle" });
    emit();
  }

  function emit() {
    for (const listener of listeners) listener(agents);
  }

  return {
    snapshot: () => agents,
    subscribe(listener) {
      listeners.add(listener);
      if (!timer) timer = setInterval(advance, intervalMs);
      return () => {
        listeners.delete(listener);
        if (!listeners.size && timer) {
          clearInterval(timer);
          timer = undefined;
        }
      };
    },
  };
}
