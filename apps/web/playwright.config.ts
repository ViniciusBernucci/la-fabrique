import { defineConfig } from "@playwright/test";

// Rodado por `make e2e` (e2e/run.mjs sobe a API descartável, o GitHub de mentira e o front e
// preenche estas variáveis). Sem elas não há o que testar.
const baseURL = process.env.E2E_BASE_URL;

export default defineConfig({
  testDir: "./e2e",
  testMatch: "**/*.spec.ts",
  outputDir: "./e2e/.results",
  workers: 1,
  retries: 0,
  timeout: 60_000,
  reporter: [["list"]],
  use: {
    baseURL,
    locale: "pt-BR",
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
  },
});
