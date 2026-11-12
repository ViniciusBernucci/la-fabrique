import { useEffect, useRef, useState } from "react";
import { createMockOfficeFeed, type OfficeFeed } from "./office-feed";
import {
  applyActivities,
  createDefaultLayout,
  createOfficeSim,
  describeActivity,
  type OfficeAgentInput,
  seededRandom,
  settleOffice,
  stepOffice,
} from "./office-model";
import { buildScene, computeViewport, drawOffice, type OfficeScene } from "./office-render";

const FRAME_MS = 1000 / 30;
const DESKS = 10;

/**
 * Escritório isométrico somente visual. O laço de desenho só existe enquanto o
 * componente está montado, visível na tela e com a aba ativa; fora disso, nada roda.
 */
export function VirtualOffice({ feed }: { feed?: OfficeFeed }) {
  const wrapRef = useRef<HTMLDivElement>(null);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [roster, setRoster] = useState<OfficeAgentInput[]>(() => feed?.snapshot() ?? []);

  useEffect(() => {
    const wrap = wrapRef.current;
    const canvas = canvasRef.current;
    const ctx = canvas?.getContext("2d");
    if (!wrap || !canvas || !ctx) return;
    const source = feed ?? createMockOfficeFeed();
    const layout = createDefaultLayout();
    const sim = createOfficeSim(layout, source.snapshot(), seededRandom(7));
    const reduceMotion = window.matchMedia?.("(prefers-reduced-motion: reduce)").matches ?? false;
    let scene: OfficeScene | null = null;
    let raf = 0;
    let last = 0;
    let lastDraw = 0;
    let elapsed = 0;
    let onScreen = true;
    setRoster(source.snapshot());

    const paint = () => {
      if (scene) drawOffice(ctx, scene, sim.agents, elapsed);
    };
    const resize = () => {
      const width = wrap.clientWidth;
      if (!width) return;
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      const viewport = computeViewport(layout, width, dpr);
      canvas.width = Math.round(viewport.cssWidth * dpr);
      canvas.height = Math.round(viewport.cssHeight * dpr);
      canvas.style.height = `${viewport.cssHeight}px`;
      scene = buildScene(layout, viewport);
      paint();
    };
    const frame = (now: number) => {
      raf = requestAnimationFrame(frame);
      if (now - lastDraw < FRAME_MS) return;
      const dt = last ? Math.min((now - last) / 1000, 0.1) : 0;
      last = now;
      lastDraw = now;
      elapsed += dt;
      stepOffice(sim, dt);
      paint();
    };
    const start = () => {
      if (raf || reduceMotion || !onScreen || document.hidden) return;
      last = 0;
      raf = requestAnimationFrame(frame);
    };
    const stop = () => {
      cancelAnimationFrame(raf);
      raf = 0;
    };
    const visibility = () => (document.hidden ? stop() : start());

    if (reduceMotion) settleOffice(sim);
    resize();
    const unsubscribe = source.subscribe((agents) => {
      applyActivities(sim, agents);
      setRoster(agents);
      if (reduceMotion) {
        settleOffice(sim);
        paint();
      }
    });
    const resizeObserver = new ResizeObserver(resize);
    resizeObserver.observe(wrap);
    const intersection = new IntersectionObserver(([entry]) => {
      onScreen = entry?.isIntersecting ?? true;
      if (onScreen) start();
      else stop();
    });
    intersection.observe(canvas);
    document.addEventListener("visibilitychange", visibility);
    start();
    return () => {
      stop();
      unsubscribe();
      resizeObserver.disconnect();
      intersection.disconnect();
      document.removeEventListener("visibilitychange", visibility);
    };
  }, [feed]);

  const free = Math.max(0, DESKS - roster.length);
  return (
    <div className="office-stage">
      <div className="office-canvas" ref={wrapRef}>
        <canvas
          ref={canvasRef}
          role="img"
          aria-label={`Escritório virtual com ${roster.length} agentes e ${free} mesas livres`}
        />
      </div>
      <ul className="office-roster" aria-label="Agentes no escritório">
        {roster.map((agent) => (
          <li key={agent.id} data-activity={agent.activity.kind}>
            <span className="office-dot" aria-hidden="true" />
            <strong>{agent.name}</strong>
            <small>{agent.role}</small>
            <span>{describeActivity(agent, roster)}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
