// Local visual/navigation check. Does not authenticate or call an administrative API.
import assert from "node:assert/strict";
import { mkdir, writeFile } from "node:fs/promises";
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const origin = process.env.PREVIEW_URL ?? "http://127.0.0.1:5173";
assert.ok(["127.0.0.1", "localhost", "[::1]"].includes(new URL(origin).hostname));
const output = new URL("./browser/", import.meta.url);
await mkdir(output, { recursive: true });
const browser = await chromium.launch({ headless: true, chromiumSandbox: true });
try {
  const page = await browser.newPage({ viewport: { width: 1536, height: 1024 } });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(origin);
  await page.getByRole("img", { name: "Núcleo de orquestração JARVIS" }).waitFor();
  assert.equal(await page.locator('.home-office img').count(), 0);
  assert.match(await page.locator(".home-demo-note").innerText(), /dados simulados/);
  await page.getByRole("button", { name: "Expandir menu", exact: true }).click();
  assert.equal(await page.locator(".home-sidebar").evaluate((el) => el.getBoundingClientRect().width), 226);
  await page.getByRole("button", { name: "Núcleo IA", exact: true }).click();
  assert.equal(await page.locator("dialog.home-preview").evaluate((el) => el.open), true);
  await page.getByRole("button", { name: "Fechar prévia", exact: true }).click();
  await page.keyboard.press("Control+k");
  assert.equal(await page.getByLabel("Buscar atalhos no sistema").evaluate((el) => el === document.activeElement), true);
  await page.getByLabel("Buscar atalhos no sistema").fill("Agentes IA");
  await page.getByLabel("Buscar atalhos no sistema").fill("");
  await page.screenshot({ path: new URL("home-desktop.png", output).pathname, fullPage: true });
  await page.emulateMedia({ reducedMotion: "reduce" });
  assert.equal(await page.locator(".core-pulse").evaluate((el) => getComputedStyle(el).animationName), "none");
  await page.getByRole("button", { name: "Projetos", exact: true }).click();
  await page.getByLabel("Token administrativo").waitFor();
  assert.equal(await page.locator(".login-core").count(), 1);
  await page.screenshot({ path: new URL("login.png", output).pathname, fullPage: true });
  await page.getByRole("button", { name: "Tarefas", exact: true }).click();
  await page.getByRole("heading", { name: "Tarefas", exact: true }).waitFor();
  assert.equal(await page.locator(".task-card").count(), 9);
  await page.getByRole("button", { name: "Lista", exact: true }).click();
  assert.equal(await page.locator(".tasks-list .task-card").count(), 9);
  for (const width of [1536, 1024, 768, 390, 320]) {
    await page.setViewportSize({ width, height: 1024 });
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  }
  await page.screenshot({ path: new URL("tasks-mobile.png", output).pathname, fullPage: true });
  await page.getByRole("button", { name: "Painel", exact: true }).click();
  assert.equal(await page.locator(".home-sidebar").evaluate((el) => el.getBoundingClientRect().width), 66);
  assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  await page.screenshot({ path: new URL("home-mobile.png", output).pathname, fullPage: true });
  assert.deepEqual(errors, []);
  await writeFile(new URL("results.txt", output), "PASS: home/login/tasks, navigation/preview/search focus, reduced motion and five viewport widths. Settings/operation authenticated flows require separate manual QA.\n");
} finally {
  await browser.close();
}
