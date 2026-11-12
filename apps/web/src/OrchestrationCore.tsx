import type { CSSProperties } from "react";
import { useId } from "react";

export type CoreAgent = { name: string; color: string };
const center = 300;
function point(radius: number, degrees: number) {
  const angle = (degrees * Math.PI) / 180;
  return { x: center + Math.cos(angle) * radius, y: center + Math.sin(angle) * radius };
}
function arc(radius: number, start: number, end: number) {
  const from = point(radius, start);
  const to = point(radius, end);
  return `M${from.x} ${from.y}A${radius} ${radius} 0 0 1 ${to.x} ${to.y}`;
}
// A fixed seed keeps the decorative network stable across clock updates and renders.
let seed = 7;
function random() {
  seed = (seed * 16807) % 2147483647;
  return seed / 2147483647;
}
const neurons: { x: number; y: number }[] = [];
while (neurons.length < 34) {
  const x = (random() * 2 - 1) * 88;
  const y = (random() * 2 - 1) * 88;
  if (x * x + y * y < 88 * 88) neurons.push({ x: center + x, y: center + y });
}
const connections = neurons.flatMap((from, index) =>
  neurons
    .map((to, next) => ({ to, next, distance: (from.x - to.x) ** 2 + (from.y - to.y) ** 2 }))
    .filter(({ next }) => next !== index)
    .sort((a, b) => a.distance - b.distance)
    .slice(0, 2)
    .filter(({ next }) => index < next)
    .map(({ to, next }) => ({ from, to, key: `${index}-${next}` })),
);
function timing(seconds: number, delay = 0): CSSProperties {
  return { animationDuration: `${seconds}s`, animationDelay: `${delay}s` };
}

export function OrchestrationCore({ agents }: { agents: readonly CoreAgent[] }) {
  const id = useId();
  const glow = `${id}-glow`;
  const sphere = `${id}-sphere`;
  const light = `${id}-light`;
  const nodes = agents.filter((agent) => agent.name !== "Núcleos");
  return (
    <svg
      className="orchestration-core"
      viewBox="0 0 600 600"
      role="img"
      aria-label="Núcleos, centro de orquestração da La fabrique"
    >
      <title>Núcleos, centro de orquestração da La fabrique</title>
      <desc>
        Representação demonstrativa dos Núcleos conectados aos agentes{" "}
        {nodes.map((agent) => agent.name).join(", ")}. Não representa execução real.
      </desc>
      <defs>
        <radialGradient id={sphere}>
          <stop stopColor="var(--hud-cyan)" stopOpacity=".28" />
          <stop offset=".8" stopColor="var(--hud-surface-solid)" stopOpacity=".25" />
          <stop offset="1" stopColor="var(--hud-cyan)" stopOpacity=".55" />
        </radialGradient>
        <radialGradient id={light}>
          <stop stopColor="var(--hud-text-strong)" />
          <stop offset=".35" stopColor="var(--hud-cyan)" />
          <stop offset="1" stopColor="var(--hud-cyan)" stopOpacity="0" />
        </radialGradient>
        <filter id={glow} x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="4" result="blur" />
          <feMerge>
            <feMergeNode in="blur" />
            <feMergeNode in="SourceGraphic" />
          </feMerge>
        </filter>
      </defs>
      <g className="core-spin" style={timing(120)} stroke="var(--hud-cyan)">
        {Array.from({ length: 120 }, (_, index) => {
          const outer = point(258, index * 3);
          const inner = point(index % 10 === 0 ? 246 : 252, index * 3);
          return (
            <line
              key={`${outer.x}-${outer.y}`}
              x1={outer.x}
              y1={outer.y}
              x2={inner.x}
              y2={inner.y}
              opacity={index % 10 === 0 ? 0.7 : 0.3}
            />
          );
        })}
      </g>
      <circle
        className="core-spin-reverse"
        cx="300"
        cy="300"
        r="276"
        fill="none"
        stroke="var(--hud-amber)"
        opacity=".35"
        strokeDasharray="2 7"
        style={timing(90)}
      />
      <g
        className="core-spin"
        style={timing(26)}
        fill="none"
        stroke="var(--hud-cyan)"
        strokeWidth="3"
        strokeLinecap="round"
        filter={`url(#${glow})`}
      >
        <path d={arc(208, -30, 60)} />
        <path d={arc(208, 150, 215)} />
      </g>
      <g className="core-spin-reverse" style={timing(18)} fill="none">
        <circle cx="300" cy="300" r="186" stroke="var(--hud-cyan)" opacity=".15" />
        <path d={arc(186, 80, 130)} stroke="var(--hud-teal)" strokeWidth="2" />
        <path d={arc(186, 250, 330)} stroke="var(--hud-teal)" strokeWidth="2" />
      </g>
      {nodes.map((agent, index) => (
        <g
          key={agent.name}
          className={`tone-${agent.color}`}
          transform={`rotate(${index * (360 / nodes.length)} 300 300)`}
        >
          <line x1="300" y1="300" x2="300" y2="64" stroke="var(--tone)" opacity=".18" />
          <line
            className="core-flow"
            x1="300"
            y1="300"
            x2="300"
            y2="64"
            stroke="var(--tone)"
            strokeWidth="1.6"
            strokeDasharray="6 22"
            style={timing(1.6 + index * 0.2)}
          />
          <circle
            className="core-transmission"
            cx="300"
            cy="64"
            r="3.5"
            fill="var(--tone)"
            filter={`url(#${glow})`}
            style={timing(2.4 + index * 0.35)}
          />
        </g>
      ))}
      <circle
        cx="300"
        cy="300"
        r="120"
        fill={`url(#${sphere})`}
        stroke="var(--hud-cyan)"
        strokeOpacity=".6"
      />
      <g className="core-spin" style={timing(14)}>
        {[0, 30, 60, 90, 120, 150].map((angle) => (
          <ellipse
            key={angle}
            cx="300"
            cy="300"
            rx="120"
            ry="34"
            fill="none"
            stroke="var(--hud-cyan)"
            opacity=".22"
            transform={`rotate(${angle} 300 300)`}
          />
        ))}
      </g>
      <g stroke="var(--hud-text-strong)" opacity=".35" strokeWidth=".8">
        {connections.map(({ from, to, key }) => (
          <line key={key} x1={from.x} y1={from.y} x2={to.x} y2={to.y} />
        ))}
      </g>
      {neurons.map((neuron, index) => (
        <circle
          key={`${neuron.x}-${neuron.y}`}
          className="core-blink"
          cx={neuron.x}
          cy={neuron.y}
          r={index % 5 === 0 ? 2.6 : 1.6}
          fill="var(--hud-text-strong)"
          style={timing(1.4 + (index % 7) * 0.4, (index % 9) * 0.2)}
        />
      ))}
      {[-25, 40, 95].map((angle, index) => (
        <g
          key={angle}
          transform={`rotate(${angle} 300 300)`}
          className={["tone-cyan", "tone-green", "tone-purple"][index]}
        >
          <ellipse
            cx="300"
            cy="300"
            rx="158"
            ry="52"
            fill="none"
            stroke="var(--tone)"
            opacity=".5"
          />
          <circle
            className="core-orbit-particle"
            r="4"
            fill="var(--tone)"
            filter={`url(#${glow})`}
            style={timing([9, 13, 11][index] ?? 9)}
          />
        </g>
      ))}
      {[0, 1, 2].map((delay) => (
        <circle
          key={delay}
          className="core-wave"
          cx="300"
          cy="300"
          r="60"
          fill="none"
          stroke="var(--hud-cyan)"
          style={timing(3, delay)}
        />
      ))}
      <circle className="core-pulse" cx="300" cy="300" r="46" fill={`url(#${light})`} />
      <text className="core-name" x="300" y="304" textAnchor="middle">
        NÚCLEOS
      </text>
      {nodes.map((agent, index) => {
        const position = point(236, -90 + index * (360 / nodes.length));
        return (
          <g key={agent.name} className={`core-node tone-${agent.color}`}>
            <circle
              cx={position.x}
              cy={position.y}
              r="27"
              fill="var(--hud-node-bg)"
              stroke="var(--tone)"
              strokeWidth="1.8"
              filter={`url(#${glow})`}
            />
            <circle
              className="core-node-ring"
              cx={position.x}
              cy={position.y}
              r="33"
              fill="none"
              stroke="var(--tone)"
              opacity=".35"
              strokeDasharray="3 5"
              style={{ ...timing(8 + index), transformOrigin: `${position.x}px ${position.y}px` }}
            />
            <text className="core-code" x={position.x} y={position.y + 4} textAnchor="middle">
              {agent.name.slice(0, 3).toLocaleUpperCase("pt-BR")}
            </text>
            <text
              className="core-agent-name"
              x={position.x}
              y={position.y + (position.y > center ? 52 : -42)}
              textAnchor="middle"
            >
              {agent.name}
            </text>
          </g>
        );
      })}
    </svg>
  );
}
