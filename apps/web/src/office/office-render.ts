// Desenho isométrico do escritório em Canvas 2D (FAC-042). Fundo estático em cache,
// móveis em sprites pré-renderizados e só os agentes redesenhados a cada quadro.
import type { AgentSim, Furniture, OfficeLayout, Tile } from "./office-model";

const TW = 64;
const TH = 32;
const WALL = 86;
const MARGIN = 28;
const TOP_PAD = 30;

export interface Viewport {
  cssWidth: number;
  cssHeight: number;
  scale: number;
  dpr: number;
  ox: number;
  oy: number;
}

type Point = [number, number];
type Ctx = CanvasRenderingContext2D;

export function computeViewport(layout: OfficeLayout, cssWidth: number, dpr: number): Viewport {
  const span = layout.width + layout.height;
  const worldWidth = (span * TW) / 2 + MARGIN * 2;
  const worldHeight = (span * TH) / 2 + WALL + MARGIN * 2 + TOP_PAD;
  const scale = Math.min(cssWidth / worldWidth, 1.8);
  return {
    cssWidth,
    cssHeight: Math.round(worldHeight * scale),
    scale,
    dpr,
    ox: (layout.height * TW) / 2 + MARGIN + (cssWidth / scale - worldWidth) / 2,
    oy: WALL + MARGIN + TOP_PAD,
  };
}

function iso(vp: Viewport, x: number, y: number, z = 0): Point {
  return [((x - y) * TW) / 2 + vp.ox, ((x + y) * TH) / 2 + vp.oy - z];
}

function poly(ctx: Ctx, points: Point[], fill: string, stroke?: string) {
  ctx.beginPath();
  points.forEach(([px, py], index) => {
    if (index) ctx.lineTo(px, py);
    else ctx.moveTo(px, py);
  });
  ctx.closePath();
  ctx.fillStyle = fill;
  ctx.fill();
  if (stroke) {
    ctx.strokeStyle = stroke;
    ctx.lineWidth = 1;
    ctx.stroke();
  }
}

interface BoxColors {
  top: string;
  left: string;
  right: string;
}

/** Caixa isométrica centrada em (cx, cy) com meia-largura `a` (x) e `b` (y). */
function box(
  ctx: Ctx,
  vp: Viewport,
  cx: number,
  cy: number,
  a: number,
  b: number,
  z: number,
  h: number,
  c: BoxColors,
) {
  const n = iso(vp, cx - a, cy - b, z + h);
  const e = iso(vp, cx + a, cy - b, z + h);
  const s = iso(vp, cx + a, cy + b, z + h);
  const w = iso(vp, cx - a, cy + b, z + h);
  const eb = iso(vp, cx + a, cy - b, z);
  const sb = iso(vp, cx + a, cy + b, z);
  const wb = iso(vp, cx - a, cy + b, z);
  poly(ctx, [w, s, sb, wb], c.left);
  poly(ctx, [s, e, eb, sb], c.right);
  poly(ctx, [n, e, s, w], c.top);
}

function newCanvas(width: number, height: number): HTMLCanvasElement {
  const canvas = document.createElement("canvas");
  canvas.width = Math.max(1, Math.ceil(width));
  canvas.height = Math.max(1, Math.ceil(height));
  return canvas;
}

function logical(ctx: Ctx, vp: Viewport) {
  const k = vp.scale * vp.dpr;
  ctx.setTransform(k, 0, 0, k, 0, 0);
}

// ---------- camada estática: piso e paredes ----------

export function buildStaticLayer(layout: OfficeLayout, vp: Viewport): HTMLCanvasElement {
  const canvas = newCanvas(vp.cssWidth * vp.dpr, vp.cssHeight * vp.dpr);
  const ctx = canvas.getContext("2d");
  if (!ctx) return canvas;
  logical(ctx, vp);
  const W = layout.width;
  const H = layout.height;

  // Paredes do fundo.
  const wallRight = [iso(vp, 0, 0), iso(vp, W, 0), iso(vp, W, 0, WALL), iso(vp, 0, 0, WALL)];
  const wallLeft = [iso(vp, 0, 0), iso(vp, 0, H), iso(vp, 0, H, WALL), iso(vp, 0, 0, WALL)];
  poly(ctx, wallRight as Point[], "#161B25");
  poly(ctx, wallLeft as Point[], "#1B212D");
  // Rodapé e topo.
  ctx.strokeStyle = "#2B3242";
  ctx.lineWidth = 2;
  for (const [a, b] of [
    [iso(vp, 0, 0, WALL), iso(vp, W, 0, WALL)],
    [iso(vp, 0, 0, WALL), iso(vp, 0, H, WALL)],
  ] as [Point, Point][]) {
    ctx.beginPath();
    ctx.moveTo(...a);
    ctx.lineTo(...b);
    ctx.stroke();
  }
  // Janelas na parede direita.
  for (const start of [2, 5.5, 9]) {
    const frame = [
      iso(vp, start, 0, 26),
      iso(vp, start + 2.4, 0, 26),
      iso(vp, start + 2.4, 0, 70),
      iso(vp, start, 0, 70),
    ];
    poly(ctx, frame as Point[], "#203248", "#33445C");
    const mid = start + 1.2;
    ctx.strokeStyle = "#33445C";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.moveTo(...iso(vp, mid, 0, 26));
    ctx.lineTo(...iso(vp, mid, 0, 70));
    ctx.stroke();
    // Reflexo.
    poly(
      ctx,
      [
        iso(vp, start + 0.3, 0, 60),
        iso(vp, start + 0.7, 0, 66),
        iso(vp, start + 0.7, 0, 40),
        iso(vp, start + 0.3, 0, 34),
      ],
      "rgba(150,190,255,0.08)",
    );
  }
  // Letreiro na parede esquerda.
  const [lx, ly] = iso(vp, 0, 4.7, 60);
  ctx.save();
  ctx.translate(lx, ly);
  ctx.transform(1, -0.5, 0, 1, 0, 0);
  ctx.fillStyle = "#6E9BFF";
  ctx.font = "700 13px ui-monospace, SFMono-Regular, Menlo, monospace";
  ctx.fillText("LA FABRIQUE", -2, 0);
  ctx.fillStyle = "#7F8796";
  ctx.font = "500 8px ui-monospace, SFMono-Regular, Menlo, monospace";
  ctx.fillText("ESCRITÓRIO DOS AGENTES", -1, 12);
  ctx.restore();
  // Quadro branco na parede esquerda.
  poly(
    ctx,
    [iso(vp, 0, 5.4, 30), iso(vp, 0, 8.4, 30), iso(vp, 0, 8.4, 64), iso(vp, 0, 5.4, 64)] as Point[],
    "#232A37",
    "#323B4D",
  );
  ctx.strokeStyle = "rgba(110,155,255,0.45)";
  ctx.lineWidth = 1.2;
  for (let i = 0; i < 4; i++) {
    ctx.beginPath();
    ctx.moveTo(...iso(vp, 0, 5.8, 56 - i * 7));
    ctx.lineTo(...iso(vp, 0, 5.8 + 1 + (i % 2) * 1.2, 56 - i * 7));
    ctx.stroke();
  }

  // Piso xadrez sutil.
  for (let y = 0; y < H; y++) {
    for (let x = 0; x < W; x++) {
      const shade = (x + y) % 2 ? "#1A1F29" : "#1C222D";
      poly(
        ctx,
        [iso(vp, x, y), iso(vp, x + 1, y), iso(vp, x + 1, y + 1), iso(vp, x, y + 1)],
        shade,
      );
    }
  }
  // Tapete da área de leitura e da copa.
  poly(
    ctx,
    [iso(vp, 11.6, 7.6), iso(vp, 14.8, 7.6), iso(vp, 14.8, 10.4), iso(vp, 11.6, 10.4)],
    "rgba(174,158,247,0.10)",
    "rgba(174,158,247,0.18)",
  );
  poly(
    ctx,
    [iso(vp, 11.6, 1.1), iso(vp, 14.9, 1.1), iso(vp, 14.9, 3), iso(vp, 11.6, 3)],
    "rgba(227,169,70,0.07)",
    "rgba(227,169,70,0.15)",
  );
  // Vagas livres: contorno tracejado no lugar da cadeira.
  ctx.setLineDash([3, 3]);
  for (const slot of layout.desks) {
    const { x, y } = slot.seat;
    poly(
      ctx,
      [
        iso(vp, x + 0.15, y + 0.15),
        iso(vp, x + 0.85, y + 0.15),
        iso(vp, x + 0.85, y + 0.85),
        iso(vp, x + 0.15, y + 0.85),
      ],
      "rgba(0,0,0,0)",
      "rgba(255,255,255,0.05)",
    );
  }
  ctx.setLineDash([]);
  return canvas;
}

// ---------- sprites de móveis ----------

interface Sprite {
  image: HTMLCanvasElement;
  left: number;
  top: number;
  width: number;
  height: number;
}

const SPRITE_W = 120;
const SPRITE_H = 140;

function makeSprite(vp: Viewport, anchor: Point, draw: (ctx: Ctx) => void): Sprite {
  const left = anchor[0] - SPRITE_W / 2;
  const top = anchor[1] - SPRITE_H + 40;
  const k = vp.scale * vp.dpr;
  const image = newCanvas(SPRITE_W * k, SPRITE_H * k);
  const ctx = image.getContext("2d");
  if (ctx) {
    ctx.setTransform(k, 0, 0, k, -left * k, -top * k);
    draw(ctx);
  }
  return { image, left, top, width: SPRITE_W, height: SPRITE_H };
}

const wood = { top: "#3A3F4D", left: "#2A2E39", right: "#23262F" };

function drawFurniture(ctx: Ctx, vp: Viewport, item: Furniture) {
  const cx = item.x + 0.5;
  const cy = item.y + 0.5;
  switch (item.kind) {
    case "desk": {
      box(ctx, vp, cx, cy, 0.44, 0.32, 0, 17, wood);
      box(ctx, vp, cx, cy - 0.12, 0.05, 0.03, 17, 4, {
        top: "#3B4150",
        left: "#2B303B",
        right: "#22262F",
      });
      box(ctx, vp, cx, cy - 0.14, 0.27, 0.03, 21, 15, {
        top: "#2E3442",
        left: "#0E1218",
        right: "#232834",
      });
      box(ctx, vp, cx, cy + 0.12, 0.18, 0.06, 17, 1.5, {
        top: "#545B6B",
        left: "#3A404D",
        right: "#333845",
      });
      box(ctx, vp, cx + 0.3, cy + 0.08, 0.05, 0.05, 17, 4, {
        top: "#C9CED7",
        left: "#9AA1AE",
        right: "#8A909C",
      });
      break;
    }
    case "coffee": {
      box(ctx, vp, cx, cy, 0.42, 0.34, 0, 20, {
        top: "#3A3F4D",
        left: "#2A2E39",
        right: "#23262F",
      });
      box(ctx, vp, cx - 0.05, cy - 0.05, 0.18, 0.16, 20, 24, {
        top: "#4A505E",
        left: "#30343F",
        right: "#262A33",
      });
      const [lx, ly] = iso(vp, cx - 0.05, cy + 0.11, 36);
      ctx.fillStyle = "#F07A7A";
      ctx.fillRect(lx - 1, ly - 1, 2.5, 2.5);
      break;
    }
    case "cooler": {
      box(ctx, vp, cx, cy, 0.18, 0.18, 0, 26, {
        top: "#3A4050",
        left: "#2B303B",
        right: "#22262F",
      });
      ctx.beginPath();
      const [bx, by] = iso(vp, cx, cy, 36);
      ctx.ellipse(bx, by, 7, 9, 0, 0, Math.PI * 2);
      ctx.fillStyle = "rgba(125,166,255,0.55)";
      ctx.fill();
      break;
    }
    case "shelf": {
      box(ctx, vp, cx, cy, 0.22, 0.44, 0, 58, {
        top: "#3A3F4D",
        left: "#2A2E39",
        right: "#23262F",
      });
      const colors = ["#6E9BFF", "#AE9EF7", "#E3A946", "#4CC38A", "#F07A7A", "#C9CED7"];
      for (let shelf = 0; shelf < 3; shelf++) {
        for (let i = 0; i < 5; i++) {
          const y0 = cy - 0.36 + i * 0.16;
          const [p1x, p1y] = iso(vp, cx - 0.22, y0, 8 + shelf * 17);
          ctx.fillStyle = colors[(i + shelf * 2) % colors.length] as string;
          ctx.globalAlpha = 0.75;
          ctx.fillRect(p1x - 4, p1y - 12, 3.2, 11);
          ctx.globalAlpha = 1;
        }
      }
      break;
    }
    case "plant": {
      box(ctx, vp, cx, cy, 0.16, 0.16, 0, 12, {
        top: "#4A3B33",
        left: "#3A2E28",
        right: "#30261F",
      });
      const [px, py] = iso(vp, cx, cy, 22);
      for (const [dx, dy, r, c] of [
        [-5, 2, 7, "#2F7A57"],
        [5, 1, 7, "#2A6B4D"],
        [0, -6, 8, "#3A9468"],
        [-2, -1, 6, "#46A877"],
      ] as const) {
        ctx.beginPath();
        ctx.arc(px + dx, py + dy, r, 0, Math.PI * 2);
        ctx.fillStyle = c;
        ctx.fill();
      }
      break;
    }
    case "sofa": {
      const fabric = { top: "#4B4470", left: "#3A3459", right: "#312C4B" };
      box(ctx, vp, cx, cy + 0.05, 0.48, 0.34, 0, 9, fabric);
      box(ctx, vp, cx, cy - 0.36, 0.48, 0.08, 0, 24, fabric);
      break;
    }
  }
}

function drawChair(ctx: Ctx, vp: Viewport, seat: Tile, part: "seat" | "back") {
  const cx = seat.x + 0.5;
  const cy = seat.y + 0.5;
  const fabric = { top: "#3B4252", left: "#2C3240", right: "#252A35" };
  if (part === "seat") {
    box(ctx, vp, cx, cy, 0.03, 0.03, 0, 7, { top: "#555", left: "#444", right: "#3A3A3A" });
    box(ctx, vp, cx, cy, 0.2, 0.2, 7, 3, fabric);
  } else box(ctx, vp, cx, cy + 0.2, 0.2, 0.035, 9, 15, fabric);
}

// ---------- agentes ----------

const SHIRTS = [
  "#6E9BFF",
  "#AE9EF7",
  "#4CC38A",
  "#E3A946",
  "#F07A7A",
  "#4CC3B8",
  "#C9A6FF",
  "#7FD1FF",
  "#FFB38A",
  "#9BE38F",
];
const SKINS = ["#F1C7A5", "#D9A27E", "#B97D57", "#8D5A3B", "#E8B896", "#A86F4C"];
const HAIRS = ["#2B2420", "#5A3B26", "#1B1B1F", "#8A5A35", "#C9A16A", "#3A2A40"];

function shade(hex: string, amount: number): string {
  const n = Number.parseInt(hex.slice(1), 16);
  const f = (c: number) => Math.max(0, Math.min(255, Math.round(c + amount)));
  return `rgb(${f(n >> 16)},${f((n >> 8) & 255)},${f(n & 255)})`;
}

function statusColor(agent: AgentSim): string {
  if (agent.activity.kind === "working") return "#6E9BFF";
  if (agent.activity.kind === "handoff" && !agent.handoffDelivered) return "#E3A946";
  return "#4CC38A";
}

function roundRect(ctx: Ctx, x: number, y: number, w: number, h: number, r: number) {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.arcTo(x + w, y, x + w, y + h, r);
  ctx.arcTo(x + w, y + h, x, y + h, r);
  ctx.arcTo(x, y + h, x, y, r);
  ctx.arcTo(x, y, x + w, y, r);
  ctx.closePath();
}

const AGENT_SCALE = 1.35;

function drawAgent(ctx: Ctx, vp: Viewport, agent: AgentSim, index: number, sofa: boolean) {
  const [ax, ay] = iso(vp, agent.x, agent.y);
  ctx.save();
  ctx.translate(ax, ay);
  ctx.scale(AGENT_SCALE, AGENT_SCALE);
  ctx.translate(-ax, -ay);
  drawAgentBody(ctx, ax, ay, agent, index, sofa);
  ctx.restore();
}

function drawAgentBody(
  ctx: Ctx,
  fx: number,
  fy: number,
  agent: AgentSim,
  index: number,
  sofa: boolean,
) {
  const seated = agent.pose === "type" || (agent.pose === "read" && sofa);
  const front = agent.facing === "SE" || agent.facing === "SW";
  const dir = agent.facing === "SE" || agent.facing === "NE" ? 1 : -1;
  const shirt = SHIRTS[index % SHIRTS.length] as string;
  const skin = SKINS[(index * 7 + 2) % SKINS.length] as string;
  const hair = HAIRS[(index * 5 + 1) % HAIRS.length] as string;
  const t = agent.phase;
  const hip = seated ? fy - 15 : fy - 11;

  // Sombra.
  ctx.beginPath();
  ctx.ellipse(fx, fy, 9, 4, 0, 0, Math.PI * 2);
  ctx.fillStyle = "rgba(0,0,0,0.35)";
  ctx.fill();

  // Pernas.
  ctx.fillStyle = "#2A2F3D";
  if (!seated) {
    const swing = agent.pose === "walk" ? Math.sin(t * 11) * 2.4 : 0;
    ctx.fillRect(fx - 4, hip, 3, 11 + swing * 0.3);
    ctx.fillRect(fx + 1, hip, 3, 11 - swing * 0.3);
    ctx.fillStyle = "#14171E";
    ctx.fillRect(fx - 4.5 + swing * 0.4, fy - 1.5 + Math.max(0, swing) * 0.2, 4, 2);
    ctx.fillRect(fx + 0.5 - swing * 0.4, fy - 1.5 + Math.max(0, -swing) * 0.2, 4, 2);
  } else if (front) {
    ctx.fillRect(fx - 4, hip - 1, 3, 6);
    ctx.fillRect(fx + 1, hip - 1, 3, 6);
  }

  // Tronco.
  const bob =
    agent.pose === "walk"
      ? Math.abs(Math.sin(t * 11)) * 1.2
      : agent.pose === "type"
        ? Math.sin(t * 3) * 0.3
        : 0;
  const top = hip - 13 - bob;
  roundRect(ctx, fx - 6.5, top, 13, 14, 4);
  ctx.fillStyle = shirt;
  ctx.fill();
  ctx.fillStyle = shade(shirt, -35);
  ctx.fillRect(fx + (dir > 0 ? 3 : -6.5), top + 3, 3.5, 10);

  // Braços e objetos nas mãos.
  const handY = top + 9;
  ctx.fillStyle = skin;
  if (agent.pose === "type") {
    const tap = Math.sin(t * 22) > 0 ? 1 : 0;
    ctx.fillRect(fx - 7.5, handY - 3 - tap, 2.6, 3);
    ctx.fillRect(fx + 5, handY - 3 - (1 - tap), 2.6, 3);
  } else if (agent.pose === "drink") {
    const sip = t % 3.2 < 1.1;
    const cupX = fx + dir * 7;
    const cupY = sip ? top + 1 : handY;
    ctx.fillRect(cupX - 1, cupY + 2, 2.6, 3);
    ctx.fillStyle = "#EDEFF3";
    ctx.fillRect(cupX - 2, cupY - 3, 5, 5.5);
    ctx.strokeStyle = "rgba(220,225,235,0.5)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    const steam = (t * 8) % 4;
    ctx.moveTo(cupX, cupY - 4 - steam);
    ctx.quadraticCurveTo(cupX + 2, cupY - 7 - steam, cupX, cupY - 10 - steam);
    ctx.stroke();
  } else if (agent.pose === "read") {
    ctx.fillStyle = "#AE9EF7";
    ctx.fillRect(fx - 6, handY - 4, 12, 8);
    ctx.fillStyle = "#EDEFF3";
    ctx.fillRect(fx - 5, handY - 3, 4.6, 6);
    ctx.fillRect(fx + 0.4, handY - 3, 4.6, 6);
    if (Math.sin(t * 0.8) > 0.97) ctx.fillRect(fx - 0.5, handY - 4, 1, 8);
  } else {
    const swing = agent.pose === "walk" ? Math.sin(t * 11) * 2 : 0;
    ctx.fillRect(fx - 8.5, handY - 2 + swing, 2.5, 4);
    ctx.fillRect(fx + 6, handY - 2 - swing, 2.5, 4);
    const carrying = agent.activity.kind === "handoff" && !agent.handoffDelivered;
    if (carrying) {
      const fx2 = fx + dir * (agent.pose === "deliver" ? 10 : 8);
      const lift = agent.pose === "deliver" ? -4 : 0;
      ctx.fillStyle = "#E3A946";
      ctx.fillRect(fx2 - 4, handY - 5 + lift, 8, 7);
      ctx.fillStyle = "#F5D59A";
      ctx.fillRect(fx2 - 2.5, handY - 3 + lift, 5, 1);
      ctx.fillRect(fx2 - 2.5, handY - 1 + lift, 4, 1);
    }
  }

  // Cabeça.
  const headY = top - 5;
  ctx.beginPath();
  ctx.arc(fx, headY, 5.6, 0, Math.PI * 2);
  ctx.fillStyle = skin;
  ctx.fill();
  ctx.fillStyle = hair;
  ctx.beginPath();
  if (front) {
    ctx.arc(fx, headY - 1, 5.9, Math.PI * 1.05, Math.PI * 1.95);
    ctx.fill();
    ctx.fillStyle = "#1A1C22";
    ctx.fillRect(fx + dir * 1.4 - 2.2, headY + 0.2, 1.3, 1.5);
    ctx.fillRect(fx + dir * 1.4 + 1.2, headY + 0.2, 1.3, 1.5);
  } else {
    ctx.arc(fx, headY - 0.4, 5.9, 0, Math.PI * 2);
    ctx.fill();
  }
}

function drawNameTag(ctx: Ctx, vp: Viewport, agent: AgentSim, sofa: boolean) {
  const [fx, fy] = iso(vp, agent.x, agent.y);
  const seated = agent.pose === "type" || (agent.pose === "read" && sofa);
  const y = fy - (seated ? 52 : 48) * AGENT_SCALE + 6;
  ctx.font = "600 9px ui-monospace, SFMono-Regular, Menlo, monospace";
  const label = agent.name;
  const width = ctx.measureText(label).width + 16;
  roundRect(ctx, fx - width / 2, y - 7, width, 13, 6.5);
  ctx.fillStyle = "rgba(11,13,18,0.82)";
  ctx.fill();
  ctx.strokeStyle = "rgba(255,255,255,0.08)";
  ctx.lineWidth = 1;
  ctx.stroke();
  ctx.beginPath();
  ctx.arc(fx - width / 2 + 7, y - 0.5, 2.4, 0, Math.PI * 2);
  ctx.fillStyle = statusColor(agent);
  ctx.fill();
  ctx.fillStyle = "#E7E9EE";
  ctx.textBaseline = "middle";
  ctx.fillText(label, fx - width / 2 + 12, y);
}

function drawScreen(ctx: Ctx, vp: Viewport, desk: Tile, time: number) {
  const cx = desk.x + 0.5;
  const cy = desk.y + 0.5 - 0.14 + 0.03;
  const a = 0.24;
  const p = [
    iso(vp, cx - a, cy, 34),
    iso(vp, cx + a, cy, 34),
    iso(vp, cx + a, cy, 23),
    iso(vp, cx - a, cy, 23),
  ];
  poly(ctx, p, "#18325C");
  const lines = 5;
  for (let i = 0; i < lines; i++) {
    const phase = (Math.floor(time * 2.2) + i * 3) % 7;
    const len = 0.12 + (phase / 7) * 0.3;
    const z = 31.5 - i * 1.9;
    const x0 = cx - a + 0.04 + (i % 2) * 0.03;
    ctx.strokeStyle = i % 3 === 0 ? "rgba(174,158,247,0.85)" : "rgba(125,200,255,0.85)";
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.moveTo(...iso(vp, x0, cy, z));
    ctx.lineTo(...iso(vp, Math.min(x0 + len, cx + a - 0.04), cy, z));
    ctx.stroke();
  }
  // Brilho do monitor no tampo.
  poly(
    ctx,
    [
      iso(vp, cx - 0.3, cy + 0.05, 17.2),
      iso(vp, cx + 0.3, cy + 0.05, 17.2),
      iso(vp, cx + 0.3, cy + 0.42, 17.2),
      iso(vp, cx - 0.3, cy + 0.42, 17.2),
    ],
    "rgba(110,155,255,0.10)",
  );
}

// ---------- cena ----------

export interface OfficeScene {
  layout: OfficeLayout;
  viewport: Viewport;
  background: HTMLCanvasElement;
  furniture: { sprite: Sprite; depth: number; item: Furniture }[];
  chairs: { seat: Tile; depth: number; seatSprite: Sprite; backSprite: Sprite }[];
  sofaTiles: Set<string>;
}

export function buildScene(layout: OfficeLayout, viewport: Viewport): OfficeScene {
  const furniture = layout.furniture.map((item) => ({
    item,
    depth: item.x + item.y + 1 - (item.kind === "sofa" ? 0.05 : 0),
    sprite: makeSprite(viewport, iso(viewport, item.x + 0.5, item.y + 0.5), (ctx) =>
      drawFurniture(ctx, viewport, item),
    ),
  }));
  const chairs = layout.desks.map(({ seat }) => ({
    seat,
    depth: seat.x + seat.y + 1,
    seatSprite: makeSprite(viewport, iso(viewport, seat.x + 0.5, seat.y + 0.5), (ctx) =>
      drawChair(ctx, viewport, seat, "seat"),
    ),
    backSprite: makeSprite(viewport, iso(viewport, seat.x + 0.5, seat.y + 0.5), (ctx) =>
      drawChair(ctx, viewport, seat, "back"),
    ),
  }));
  return {
    layout,
    viewport,
    background: buildStaticLayer(layout, viewport),
    furniture,
    chairs,
    sofaTiles: new Set(
      layout.furniture.filter((item) => item.kind === "sofa").map((item) => `${item.x},${item.y}`),
    ),
  };
}

type Drawable = { depth: number; draw: (ctx: Ctx) => void };

function blit(ctx: Ctx, sprite: Sprite) {
  ctx.drawImage(sprite.image, sprite.left, sprite.top, sprite.width, sprite.height);
}

/** Desenha um quadro completo. Custo: 1 blit de fundo + sprites + ~10 agentes. */
export function drawOffice(ctx: Ctx, scene: OfficeScene, agents: AgentSim[], time: number): void {
  const vp = scene.viewport;
  ctx.setTransform(1, 0, 0, 1, 0, 0);
  ctx.clearRect(0, 0, ctx.canvas.width, ctx.canvas.height);
  ctx.drawImage(scene.background, 0, 0);
  logical(ctx, vp);

  const working = new Set(
    agents
      .filter((agent) => agent.activity.kind === "working" && agent.pose === "type")
      .map((agent) => agent.deskIndex),
  );
  const items: Drawable[] = [];
  for (const entry of scene.furniture) {
    items.push({
      depth: entry.depth,
      draw: (c) => {
        blit(c, entry.sprite);
        if (entry.item.kind === "desk" && working.has(entry.item.deskIndex ?? -1))
          drawScreen(c, vp, entry.item, time);
      },
    });
  }
  for (const chair of scene.chairs) {
    items.push({ depth: chair.depth - 0.02, draw: (c) => blit(c, chair.seatSprite) });
    items.push({ depth: chair.depth + 0.02, draw: (c) => blit(c, chair.backSprite) });
  }
  agents.forEach((agent, index) => {
    const sofa = scene.sofaTiles.has(`${Math.floor(agent.x)},${Math.floor(agent.y)}`);
    items.push({ depth: agent.x + agent.y, draw: (c) => drawAgent(c, vp, agent, index, sofa) });
  });
  items.sort((a, b) => a.depth - b.depth);
  for (const item of items) item.draw(ctx);
  // Etiquetas por cima de tudo, para nunca ficarem escondidas.
  for (const agent of agents) {
    const sofa = scene.sofaTiles.has(`${Math.floor(agent.x)},${Math.floor(agent.y)}`);
    drawNameTag(ctx, vp, agent, sofa);
  }
}
