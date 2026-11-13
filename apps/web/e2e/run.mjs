// Orquestra o ensaio com navegador real: compila e sobe a API descartável (banco temporário,
// GitHub de mentira, dados de ensaio), sobe o front em portas livres escolhidas pelo sistema e roda
// o Playwright. Nenhuma porta é fixa e nenhuma credencial vem do repositório: o administrador do
// ensaio nasce com uma senha aleatória, válida só enquanto o banco temporário existe.
import { execFileSync, spawn } from "node:child_process";
import { mkdtempSync } from "node:fs";
import net from "node:net";
import { tmpdir } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const web = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const root = path.resolve(web, "../..");

if (!process.env.TEST_DATABASE_URL) {
  console.error("e2e: defina TEST_DATABASE_URL (use `make e2e`, que lê o .env)");
  process.exit(2);
}

const bin = path.join(mkdtempSync(path.join(tmpdir(), "lf-e2e-")), "e2e-stack");
execFileSync("go", ["build", "-o", bin, "./cmd/e2e-stack"], { cwd: root, stdio: "inherit" });

const children = [];
function stopAll() {
  for (const c of children) if (!c.killed) c.kill("SIGTERM");
}
process.on("SIGINT", () => {
  stopAll();
  process.exit(130);
});

function freePort() {
  return new Promise((resolve, reject) => {
    const s = net.createServer();
    s.once("error", reject);
    s.listen(0, "127.0.0.1", () => {
      const { port } = s.address();
      s.close(() => resolve(port));
    });
  });
}

function startStack() {
  return new Promise((resolve, reject) => {
    const p = spawn(bin, [], { env: process.env, stdio: ["ignore", "pipe", "inherit"] });
    children.push(p);
    let buf = "";
    p.stdout.on("data", (d) => {
      buf += d;
      const line = buf.split("\n")[0];
      if (buf.includes("\n")) resolve(JSON.parse(line));
    });
    p.once("exit", (code) => reject(new Error(`a API de ensaio encerrou (código ${code})`)));
  });
}

async function waitFor(url, ms = 60_000) {
  const end = Date.now() + ms;
  while (Date.now() < end) {
    try {
      if ((await fetch(url)).ok) return;
    } catch {
      // ainda subindo
    }
    await new Promise((r) => setTimeout(r, 300));
  }
  throw new Error(`tempo esgotado esperando ${url}`);
}

let code = 1;
try {
  const stack = await startStack();
  const port = await freePort();
  const vite = spawn(
    "npx",
    ["vite", "--port", String(port), "--strictPort", "--host", "127.0.0.1"],
    {
      cwd: web,
      env: { ...process.env, WEB_PORT: String(port), API_PROXY_TARGET: stack.api },
      stdio: "ignore",
    },
  );
  children.push(vite);
  const base = `http://127.0.0.1:${port}`;
  await waitFor(`${base}/`);
  const pw = spawn("npx", ["playwright", "test", ...process.argv.slice(2)], {
    cwd: web,
    stdio: "inherit",
    env: {
      ...process.env,
      E2E_BASE_URL: base,
      E2E_USER: stack.user,
      E2E_PASSWORD: stack.password,
      E2E_GITHUB: stack.github,
    },
  });
  code = await new Promise((resolve) => pw.once("exit", (c) => resolve(c ?? 1)));
} catch (e) {
  console.error(`e2e: ${e.message}`);
} finally {
  stopAll();
  await new Promise((r) => setTimeout(r, 1500)); // a API remove o banco temporário ao encerrar
}
process.exit(code);
