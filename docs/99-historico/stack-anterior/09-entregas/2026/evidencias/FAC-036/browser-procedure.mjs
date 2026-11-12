// FAC-036 — verificação em browser (Chromium via CDP) contra fixture descartável.
// Variáveis: CDP_PORT, WEB_URL, API_URL, ADMIN_TOKEN, FIXTURES_DIR, DOWNLOAD_DIR, EVIDENCE_DIR.
import { mkdir, readFile, readdir, rm, writeFile } from "node:fs/promises";
import { join } from "node:path";

const env = (name) => {
  const value = process.env[name];
  if (!value) throw new Error(`Variável ausente: ${name}`);
  return value;
};
const [cdpPort, webUrl, apiUrl, token, fixtures, downloads, evidence] = [
  "CDP_PORT", "WEB_URL", "API_URL", "ADMIN_TOKEN", "FIXTURES_DIR", "DOWNLOAD_DIR", "EVIDENCE_DIR",
].map(env);

const pages = await (await fetch(`http://127.0.0.1:${cdpPort}/json/list`)).json();
const ws = new WebSocket(pages.find((page) => page.type === "page").webSocketDebuggerUrl);
await new Promise((resolve, reject) => {
  ws.addEventListener("open", resolve, { once: true });
  ws.addEventListener("error", reject, { once: true });
});
let sequence = 0;
const pending = new Map();
const runtimeErrors = [];
ws.addEventListener("message", (event) => {
  const value = JSON.parse(event.data);
  if (value.id && pending.has(value.id)) {
    const { resolve, reject, timer } = pending.get(value.id);
    clearTimeout(timer);
    pending.delete(value.id);
    value.error ? reject(new Error(value.error.message)) : resolve(value.result);
  } else if (value.method === "Runtime.exceptionThrown") {
    runtimeErrors.push(value.params.exceptionDetails.text);
  }
});
function call(method, params = {}) {
  return new Promise((resolve, reject) => {
    const id = ++sequence;
    const timer = setTimeout(() => reject(new Error(`CDP timeout: ${method}`)), 15000);
    pending.set(id, { resolve, reject, timer });
    ws.send(JSON.stringify({ id, method, params }));
  });
}
async function evaluate(expression) {
  const result = await call("Runtime.evaluate", { expression, awaitPromise: true, returnByValue: true });
  if (result.exceptionDetails) throw new Error(result.exceptionDetails.exception?.description ?? expression);
  return result.result.value;
}
async function until(expression, label = expression) {
  const start = Date.now();
  while (Date.now() - start < 20000) {
    if (await evaluate(expression)) return;
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`Timeout: ${label}`);
}
const dialog = `document.querySelector("dialog[open]")`;
const dialogText = (text) => `!!${dialog}?.innerText.includes(${JSON.stringify(text)})`;
async function click(text, scope = "document", { startsWith = false } = {}) {
  await evaluate(`(() => {
    const root = ${scope};
    const match = [...root.querySelectorAll("button, a")].find((element) => {
      const value = element.textContent.trim();
      return ${startsWith} ? value.startsWith(${JSON.stringify(text)}) : value === ${JSON.stringify(text)};
    });
    if (!match) throw new Error("Elemento ausente: " + ${JSON.stringify(text)});
    match.click();
  })()`);
}
async function fill(label, value) {
  await evaluate(`(() => {
    const field = [...${dialog}.querySelectorAll("label")]
      .find((item) => item.firstChild?.textContent.trim() === ${JSON.stringify(label)})
      ?.querySelector("input, textarea, select");
    if (!field) throw new Error("Campo ausente: " + ${JSON.stringify(label)});
    const prototype = field instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype
      : field instanceof HTMLSelectElement ? HTMLSelectElement.prototype : HTMLInputElement.prototype;
    Object.getOwnPropertyDescriptor(prototype, "value").set.call(field, ${JSON.stringify(value)});
    field.dispatchEvent(new Event("input", { bubbles: true }));
    field.dispatchEvent(new Event("change", { bubbles: true }));
  })()`);
}
async function upload(file) {
  const { result } = await call("Runtime.evaluate", {
    expression: `${dialog}.querySelector("input[type=file]")`,
  });
  await call("DOM.setFileInputFiles", { objectId: result.objectId, files: [join(fixtures, file)] });
}
async function downloadAndCompare(file) {
  await rm(downloads, { recursive: true, force: true });
  await mkdir(downloads, { recursive: true });
  await call("Browser.setDownloadBehavior", { behavior: "allow", downloadPath: downloads });
  await click("Baixar arquivo .md", dialog);
  const start = Date.now();
  while (Date.now() - start < 10000) {
    const names = await readdir(downloads);
    if (names.includes(file)) {
      const [got, expected] = await Promise.all([
        readFile(join(downloads, file)), readFile(join(fixtures, file)),
      ]);
      if (!got.equals(expected)) throw new Error(`Download divergente: ${file}`);
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, 200));
  }
  throw new Error(`Download ausente: ${file}`);
}
async function settings() {
  const response = await fetch(`${apiUrl}/settings`, { headers: { Authorization: `Bearer ${token}` } });
  return (await response.json()).configuration;
}
async function expectPersisted(item, file, label) {
  const text = new TextDecoder("utf-8", { ignoreBOM: true }).decode(await readFile(join(fixtures, file)));
  if (!item || item.instructions !== text || item.instructionsFilename !== file) {
    throw new Error(`Persistência divergente: ${label}`);
  }
}
async function screenshot(name) {
  const shot = await call("Page.captureScreenshot", { format: "png" });
  await writeFile(join(evidence, `${name}.png`), Buffer.from(shot.data, "base64"));
}
async function openSettings() {
  await call("Page.navigate", { url: webUrl });
  await until(`document.readyState === "complete" && !!document.querySelector("button")`);
  await click("Configurações");
  await until(`!!document.querySelector("input[type=password]") || document.body.innerText.includes("Equipes")`);
  if (await evaluate(`!!document.querySelector("input[type=password]")`)) {
    await evaluate(`(() => {
      const input = document.querySelector("input[type=password]");
      Object.getOwnPropertyDescriptor(HTMLInputElement.prototype, "value").set.call(input, ${JSON.stringify(token)});
      input.dispatchEvent(new Event("input", { bubbles: true }));
    })()`);
    await click("Entrar");
    await until(`document.body.innerText.includes("Sessão administrativa ativa")`);
    await click("Configurações");
  }
  await until(`document.body.innerText.includes("Equipes")`);
  await click("Equipes", "document.querySelector('main')");
  await until(`document.body.innerText.includes("agente(s) na equipe")`);
}
async function openDeveloper() {
  await click("DDeveloper", "document.querySelector('main')", { startsWith: true });
  await click("Configurações do agente");
  await until(dialogText("Configurar Developer"));
}

const checks = [];
await call("Runtime.enable");
await call("Page.enable");
await call("Emulation.setDeviceMetricsOverride", { width: 1280, height: 900, deviceScaleFactor: 1, mobile: false });

// Agente pré-configurado: descrição separada, adicionar .md e salvar.
await openSettings();
await openDeveloper();
await fill("Descrição breve do agente", "Resumo curto do Developer.");
await upload("developer.md");
await until(dialogText("developer.md"));
await screenshot("agente-preconfigurado-desktop");
await click("Salvar configuração", dialog);
await until(`!${dialog}`, "modal fechado após salvar");
let configuration = await settings();
const developer = configuration.assignments.find((item) => item.role === "DEVELOPER");
await expectPersisted(developer, "developer.md", "Developer");
if (developer.description !== "Resumo curto do Developer.") throw new Error("Descrição não persistida");
checks.push("Pré-configurado: descrição e .md (BOM/CRLF/espaços/acentos) salvos pela API");

// Reabrir após recarregar: nome e conteúdo baixado idênticos.
await openSettings();
await openDeveloper();
await until(dialogText("developer.md"));
await downloadAndCompare("developer.md");
checks.push("Reabrir após recarregar e baixar bytes idênticos ao arquivo enviado");

// Arquivos inválidos: erro sem substituir o conteúdo anterior.
for (const file of ["vazio.md", "notas.txt"]) {
  await upload(file);
  await until(`!!${dialog}.querySelector("[role=alert]")`, `erro para ${file}`);
  if (!(await evaluate(dialogText("developer.md")))) throw new Error(`Conteúdo substituído por ${file}`);
}
checks.push("Vazio e extensão inválida mostram erro e preservam o arquivo anterior");

// Cancelar descarta a substituição.
await upload("cancelado.md");
await until(dialogText("cancelado.md"));
await click("Cancelar", dialog);
await until(`!${dialog}`);
await openDeveloper();
if (!(await evaluate(dialogText("developer.md")))) throw new Error("Cancelar não descartou o rascunho");
await click("Cancelar", dialog);
configuration = await settings();
await expectPersisted(configuration.assignments.find((item) => item.role === "DEVELOPER"), "developer.md", "Developer após cancelar");
checks.push("Cancelar descarta a substituição na tela e na API");

// Agente personalizado.
await click("+ Adicionar agente");
await until(dialogText("Novo agente"));
await fill("Nome do agente", "Agente Fixture");
await fill("Cargo na empresa", "Revisor de contratos");
await fill("Descrição breve do agente", "Resumo do agente personalizado.");
await upload("personalizado.md");
await until(dialogText("personalizado.md"));
await click("Salvar configuração", dialog);
await until(`!${dialog}`);
configuration = await settings();
await expectPersisted(configuration.digitalAgents.find((item) => item.nickname === "Agente Fixture"), "personalizado.md", "agente personalizado");
checks.push("Agente personalizado: .md salvo pela API");

// Skill do projeto.
await openSettings();
await click("Skills", "document.querySelector('main')");
await evaluate(`(() => {
  const select = [...document.querySelectorAll("main select")].find((item) =>
    item.closest("label")?.innerText.startsWith("Projeto das skills"));
  select.value = select.options[1].value;
  select.dispatchEvent(new Event("change", { bubbles: true }));
})()`);
await click("+ Registrar skill");
await until(`!!${dialog}`);
await fill("Nome da skill", "revisar-contratos");
await fill("Descrição breve da skill", "Revisa contratos antes do merge.");
await upload("SKILL.md");
await until(dialogText("SKILL.md"));
await screenshot("skill-desktop");
await click("Salvar configuração", dialog);
await until(`!${dialog}`);
configuration = await settings();
await expectPersisted(configuration.projectSkills.find((item) => item.name === "revisar-contratos"), "SKILL.md", "skill");
checks.push("Skill: .md salvo pela API");

// Mobile: modal sem overflow horizontal.
await call("Emulation.setDeviceMetricsOverride", { width: 390, height: 844, deviceScaleFactor: 1, mobile: true });
await openSettings();
await openDeveloper();
await screenshot("agente-preconfigurado-mobile");
if (await evaluate("document.documentElement.scrollWidth > innerWidth + 2")) throw new Error("Overflow no mobile");
checks.push("390px sem overflow horizontal com o modal aberto");

if (runtimeErrors.length) throw new Error(`Exceções de runtime: ${runtimeErrors.join(", ")}`);
const result = {
  fixture: true, web: webUrl, api: apiUrl, database: "PostgreSQL descartável", worker: false,
  checks, runtimeExceptions: runtimeErrors, desktop: [1280, 900], mobile: [390, 844],
};
await writeFile(join(evidence, "browser.json"), `${JSON.stringify(result, null, 2)}\n`);
console.log(JSON.stringify({ checks: checks.length, runtimeExceptions: runtimeErrors.length }));
ws.close();
