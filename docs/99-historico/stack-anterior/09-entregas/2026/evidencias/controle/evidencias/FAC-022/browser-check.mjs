import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const browser = await chromium.launch({ headless: true });
const output = fileURLToPath(new URL("./", import.meta.url));
const origin = process.env.PREVIEW_URL ?? "http://127.0.0.1:5174";
const results = [];
try {
  const page = await browser.newPage({ viewport: { width: 1536, height: 1024 } });
  const errors = [];
  const apiRequests = [];
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("request", (request) => { if (request.url().includes("/api/")) apiRequests.push(request.url()); });
  await page.route("**/api/**", (route) => route.fulfill({ status: 401, contentType: "application/json", body: '{"error":"Synthetic unauthenticated test"}' }));
  await page.goto(origin);
  await page.locator(".home-office img").evaluate((img) => img.decode());
  assert.equal(await page.getByRole("heading", { name: "Central de controle La fabrique" }).count(), 1);
  assert.equal(await page.title(), "La fabrique");
  assert.equal(await page.locator(".home-brand").innerText(), "La fabrique");
  const sceneRatio = await page.evaluate(() => document.querySelector(".home-office").getBoundingClientRect().width / document.querySelector(".home-center").getBoundingClientRect().width);
  assert.ok(Math.abs(sceneRatio - 0.6) < 0.001);
  await page.evaluate(() => { window.layoutSidebar = document.querySelector(".home-sidebar"); window.layoutTopbar = document.querySelector(".home-topbar"); });
  assert.equal(apiRequests.length, 0);
  results.push("Correct La fabrique title/brand and central scene at 60% width");
  results.push("Home opens without authentication or API requests");
  for (const width of [1536, 1280, 1024, 768, 390, 320]) {
    await page.setViewportSize({ width, height: 1024 });
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
    assert.equal(overflow, false, `Horizontal overflow at ${width}px`);
    if (width === 1536 || width === 390) await page.screenshot({ path: `${output}${width === 1536 ? "desktop" : "mobile"}.png`, fullPage: true });
    results.push(`No horizontal overflow at ${width}px`);
  }
  await page.setViewportSize({ width: 1536, height: 1024 });
  await page.keyboard.press("Control+k");
  assert.equal(await page.getByRole("textbox", { name: "Buscar atalhos no sistema" }).evaluate((el) => el === document.activeElement), true);
  await page.getByRole("textbox", { name: "Buscar atalhos no sistema" }).fill("configurações");
  assert.equal(await page.locator(".home-shortcuts button").count(), 1);
  await page.getByRole("textbox", { name: "Buscar atalhos no sistema" }).fill("zzzzzz");
  await page.getByRole("status").waitFor();
  await page.getByRole("textbox", { name: "Buscar atalhos no sistema" }).fill("");
  results.push("Keyboard shortcut, search filter and empty state work");
  const finance = page.getByRole("button", { name: "Financeiro", exact: true });
  await finance.click();
  await page.getByRole("dialog").waitFor();
  await page.keyboard.press("Escape");
  assert.equal(await page.getByRole("dialog").count(), 0);
  assert.equal(await finance.evaluate((el) => el === document.activeElement), true);
  await finance.click();
  await page.getByRole("button", { name: "Voltar ao painel", exact: true }).click();
  assert.equal(await page.getByRole("dialog").count(), 0);
  results.push("Future area preview opens, closes with Escape/button and restores focus");
  await page.getByRole("button", { name: "Ver notificações importantes" }).click();
  assert.equal(await page.getByRole("dialog").getByText("Pagamento Hotmart não processado", { exact: true }).count(), 1);
  await page.keyboard.press("Escape");
  await page.getByRole("button", { name: "Projetos", exact: true }).click();
  await page.getByLabel("Token administrativo").waitFor();
  assert.equal(await page.locator(".eyebrow").innerText(), "La fabrique");
  assert.equal(apiRequests.length, 0);
  await page.getByLabel("Token administrativo").fill("synthetic-browser-test-credential-0000");
  await page.getByRole("button", { name: "Entrar", exact: true }).click();
  await page.getByText("Credencial inválida", { exact: true }).waitFor();
  assert.equal(await page.locator(".shell").count(), 0);
  results.push("Projects shortcut preserves login; invalid synthetic credential cannot bypass authentication");
  await page.getByRole("button", { name: "Voltar ao painel inicial" }).click();
  await page.getByRole("button", { name: "Configurações", exact: true }).click();
  await page.getByLabel("Token administrativo").waitFor();
  results.push("Settings shortcut preserves login and home return works");
  assert.equal(await page.evaluate(() => window.layoutSidebar === document.querySelector(".home-sidebar") && window.layoutTopbar === document.querySelector(".home-topbar")), true);
  results.push("Sidebar/topbar DOM nodes persist from home through login and back");
  const timestamp = "2026-10-05T20:40:00.000Z";
  const settings = { version: 1, createdAt: timestamp, updatedAt: timestamp, configuration: {
    installations: [], assignments: ["PLANNER", "DEVELOPER", "REVIEWER", "QA", "DOCUMENTATION", "SECURITY"].map((role) => ({ role, enabled: false, installationId: null, model: null, permissionMode: "READ_ONLY", timeoutMinutes: 10, maxAttempts: 1 })),
    github: { authMode: "GH_CLI", state: "DISCONNECTED", host: "github.com", owner: null, repository: null, baseBranch: "main", pullRequestCreationEnabled: false, mergeEnabled: false },
    financialSafety: { apiEnabled: false, extraUsageEnabled: false, paidCreditsEnabled: false, autoRechargeEnabled: false, paidFallbackEnabled: false }
  }};
  await page.unroute("**/api/**");
  await page.route("**/api/**", (route) => {
    const path = new URL(route.request().url()).pathname;
    assert.equal(route.request().method(), "GET");
    return route.fulfill({ status: path === "/api/operation" ? 503 : 200, contentType: "application/json", body: JSON.stringify(path === "/api/auth/session" ? {authenticated:true} : path === "/api/settings" ? settings : []) });
  });
  await page.getByLabel("Token administrativo").fill("synthetic-browser-test-credential-0000");
  await page.getByRole("button", { name: "Entrar", exact: true }).click();
  await page.getByRole("heading", { name: "Equipe, contas e integrações", exact: true }).waitFor();
  assert.equal(await page.locator(".home-sidebar button[aria-current=page]").getAttribute("aria-label"), "Configurações");
  assert.equal(await page.locator(".workspace-content h1").evaluate((el) => getComputedStyle(el).fontSize), "24px");
  assert.equal(await page.locator(".settings-hero h2").evaluate((el) => getComputedStyle(el).fontSize), "16px");
  await page.screenshot({path: `${output}settings-desktop.png`, fullPage:true});
  await page.locator(".account-list-item").first().click();
  await page.locator(".account-dialog").waitFor();
  assert.equal(await page.locator(".account-dialog h2").evaluate((el) => getComputedStyle(el).fontSize), "16px");
  assert.equal(await page.locator(".account-dialog label").first().evaluate((el) => getComputedStyle(el).fontSize), "12px");
  await page.getByRole("button", {name:"Fechar configuração",exact:true}).click();
  results.push("Settings modal uses compact 16px headings and 12px labels and closes without saving");
  for (const width of [1536, 1024, 768, 390, 320]) {
    await page.setViewportSize({width,height:1024});
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `Settings overflow at ${width}`);
  }
  await page.screenshot({path: `${output}settings-mobile.png`, fullPage:true});
  await page.setViewportSize({width:1536,height:1024});
  await page.keyboard.press("Control+k");
  await page.getByRole("textbox", {name:"Buscar atalhos no sistema"}).fill("Novo Projeto");
  await page.locator(".home-search-results").getByRole("button", {name:"Novo Projeto",exact:true}).click();
  await page.getByRole("heading", {name:"Controle",exact:true}).waitFor();
  assert.equal(await page.locator(".home-sidebar button[aria-current=page]").getAttribute("aria-label"), "Projetos");
  await page.screenshot({path: `${output}control-desktop.png`, fullPage:true});
  await page.evaluate(() => window.scrollTo(0,600));
  assert.equal(await page.locator(".home-topbar").evaluate((el) => el.getBoundingClientRect().top), 0);
  await page.evaluate(() => window.scrollTo(0,0));
  results.push("Global shortcut search works on settings and topbar stays visible while scrolling");
  for (const width of [1536, 1024, 768, 390, 320]) {
    await page.setViewportSize({width,height:1024});
    assert.equal(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth), false, `Control overflow at ${width}`);
  }
  await page.getByRole("button", {name:"Painel",exact:true}).click();
  await page.getByRole("heading", {name:"Central de controle La fabrique",exact:true}).waitFor();
  assert.equal(await page.evaluate(() => window.layoutSidebar === document.querySelector(".home-sidebar") && window.layoutTopbar === document.querySelector(".home-topbar")), true);
  results.push("Synthetic settings/control/home navigation preserves DOM nodes and active menu; compact typography and five viewport widths PASS");
  assert.deepEqual(errors, []);
  results.push("No browser runtime errors");
  console.log(JSON.stringify({ browser: browser.version(), origin, results }, null, 2));
} finally {
  await browser.close();
}
