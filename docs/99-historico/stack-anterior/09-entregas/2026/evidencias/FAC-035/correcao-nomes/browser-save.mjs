// Synthetic browser check: real SettingsService with an in-memory persistence fixture.
// No real credentials, provider actions, database, or production writes.
import assert from "node:assert/strict";
import { createRequire } from "node:module";
import { resolve } from "node:path";
const require = createRequire(resolve("package.json"));
const { chromium } = await import(process.env.PLAYWRIGHT_MODULE ?? "playwright");
const { SettingsService } = require("./apps/api/dist/settings/settings.service.js");
const { createDefaultFactoryConfiguration } = require("./apps/api/dist/settings/settings.defaults.js");
const { updateFactorySettingsSchema } = require("@le-fabrique/contracts");
const origin = process.env.PREVIEW_URL ?? "http://127.0.0.1:5174";
assert.ok(["127.0.0.1", "localhost"].includes(new URL(origin).hostname));
const now = new Date();
const record = {
  id: "global", version: 1, configuration: createDefaultFactoryConfiguration(),
  createdAt: now, updatedAt: now,
};
record.configuration.digitalAgents = [{
  id: "b1234567-1234-4234-8234-123456789012", name: "Designer", description: "",
  instructions: "", projectId: null, installationId: null, model: null, skillIds: [], enabled: true,
}];
const factorySettings = {
  upsert: async () => record,
  findUnique: async () => record,
  findUniqueOrThrow: async () => record,
  updateMany: async ({ where, data }) => {
    if (where.version !== record.version) return { count: 0 };
    record.configuration = structuredClone(data.configuration);
    record.version++;
    return { count: 1 };
  },
};
const service = new SettingsService({
  factorySettings, $transaction: async (fn) => fn({ factorySettings }),
});
const browser = await chromium.launch({ headless: true, chromiumSandbox: true });
try {
  const page = await browser.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.route("**/api/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    let body = [];
    if (path === "/api/auth/session") body = { authenticated: true };
    if (path === "/api/settings") {
      if (request.method() === "PUT") {
        const input = updateFactorySettingsSchema.parse(request.postDataJSON());
        body = await service.update(input.expectedVersion, input.configuration);
      } else body = await service.get();
    }
    await route.fulfill({ json: body });
  });
  async function openTeams() {
    await page.getByRole("button", { name: "Configurações", exact: true }).click();
    await page.getByLabel("Token administrativo").fill("synthetic-browser-token-000000000000");
    await page.getByRole("button", { name: "Entrar", exact: true }).click();
    await page.getByRole("tab", { name: "Equipes", exact: true }).click();
  }
  await page.goto(origin);
  await openTeams();
  await page.getByRole("button", { name: "Ver agente Developer", exact: true }).click();
  await page.getByRole("button", { name: "Configurações do agente", exact: true }).click();
  await page.getByLabel("Nome do agente", { exact: true }).fill("Alex");
  await page.getByRole("button", { name: "Salvar configuração", exact: true }).click();
  await page.getByRole("button", { name: "Ver agente Alex", exact: true }).waitFor();
  assert.equal(record.configuration.assignments.find((a) => a.role === "DEVELOPER").nickname, "Alex");
  await page.getByRole("button", { name: "Editar agente Designer", exact: true }).click();
  await page.getByLabel("Nome do agente", { exact: true }).fill("Sofia");
  await page.getByRole("button", { name: "Salvar configuração", exact: true }).click();
  await page.getByRole("button", { name: "Editar agente Sofia", exact: true }).waitFor();
  assert.equal(record.configuration.digitalAgents[0].nickname, "Sofia");
  await page.reload();
  await openTeams();
  await page.getByRole("button", { name: "Ver agente Alex", exact: true }).waitFor();
  await page.getByRole("button", { name: "Editar agente Sofia", exact: true }).click();
  assert.equal(await page.getByLabel("Nome do agente", { exact: true }).inputValue(), "Sofia");
  assert.equal(record.version, 3);
  assert.deepEqual(errors, []);
  console.log("PASS: browser save and reload for preset/custom names; real compiled SettingsService, synthetic in-memory persistence; no real database.");
} finally {
  await browser.close();
}
