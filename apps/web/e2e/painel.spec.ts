import { type Browser, type BrowserContext, expect, type Page, test } from "@playwright/test";
import { totp } from "./totp";

const user = process.env.E2E_USER ?? "";
const password = process.env.E2E_PASSWORD ?? "";
const github = process.env.E2E_GITHUB ?? "";
const shots = "e2e/.results/telas";

test.describe.configure({ mode: "serial" });

let context: BrowserContext;
let page: Page;
const problems: string[] = [];

async function watch(p: Page) {
  p.on("pageerror", (e) => problems.push(`erro de página: ${e.message}`));
  p.on("response", (r) => {
    if (r.status() >= 500) problems.push(`HTTP ${r.status()} em ${r.url()}`);
  });
}

async function snap(name: string) {
  await page.screenshot({ path: `${shots}/${name}.png`, fullPage: true });
}

async function openProject() {
  await page.goto("/projects");
  await page
    .getByRole("link", { name: /Projeto de ensaio/ })
    .first()
    .click();
  await expect(page.getByRole("region", { name: "Entregas" })).toBeVisible();
}

test.beforeAll(async ({ browser }: { browser: Browser }) => {
  context = await browser.newContext({ locale: "pt-BR", viewport: { width: 1360, height: 900 } });
  page = await context.newPage();
  await watch(page);
});

test.afterAll(async () => {
  await context.close();
});

test("primeiro acesso: senha, cadastro do 2FA, códigos de recuperação e painel", async () => {
  await page.goto("/");
  await expect(page.getByRole("heading", { name: "Entrar" })).toBeVisible();
  await page.getByLabel("Usuário").fill(user);
  await page.getByLabel("Senha").fill(password);
  await page.getByRole("button", { name: "Continuar" }).click();
  await expect(page.getByRole("heading", { name: "Ative o 2FA" })).toBeVisible();
  await snap("01-cadastro-2fa");
  await page.getByLabel("Código do autenticador (6 dígitos)").fill("000000");
  await page.getByRole("button", { name: "Verificar" }).click();
  await expect(page.getByRole("alert")).toBeVisible(); // código errado é recusado
  const secret = await page.getByLabel("Chave secreta").inputValue();
  await page.getByLabel("Código do autenticador (6 dígitos)").fill(totp(secret));
  await page.getByRole("button", { name: "Verificar" }).click();
  await expect(page.getByRole("heading", { name: "Códigos de recuperação" })).toBeVisible();
  await expect(
    page.getByRole("list", { name: "Códigos de recuperação" }).getByRole("listitem"),
  ).not.toHaveCount(0);
  await page.getByLabel("Guardei os códigos em local seguro").check();
  await page.getByRole("button", { name: "Continuar" }).click();
  await expect(page.getByRole("link", { name: "Projetos" }).first()).toBeVisible();
  await snap("02-painel");
});

test("a senha sozinha não abre sessão", async ({ browser }) => {
  const anon = await browser.newContext({ locale: "pt-BR" });
  const p = await anon.newPage();
  await p.goto("/");
  await p.getByLabel("Usuário").fill(user);
  await p.getByLabel("Senha").fill(password);
  await p.getByRole("button", { name: "Continuar" }).click();
  await expect(p.getByRole("heading", { name: "Verificação em duas etapas" })).toBeVisible();
  await p.goto("/projects");
  await expect(p.getByRole("heading", { name: "Entrar" })).toBeVisible();
  await anon.close();
});

test("projeto: execução, entregas e memória no padrão do painel", async () => {
  await openProject();
  await expect(page.getByRole("region", { name: "Execução" })).toBeVisible();
  await expect(page.getByRole("region", { name: "Memória técnica" })).toBeVisible();
  const ped7 = page.getByRole("listitem", { name: "PED-7 tentativa 1" });
  const ped8 = page.getByRole("listitem", { name: "PED-8 tentativa 1" });
  await expect(ped7).toContainText("Draft PR aberta");
  await expect(ped8).toContainText("PR fechada sem merge");
  await expect(ped7).toContainText("PR ainda não consultada");
  await expect(page.getByRole("listitem", { name: "pedidos usam soft delete" })).toContainText(
    "Pendente",
  );
  // Só tokens do design: o fundo do painel vem de --hud-surface (nenhuma cor fixa nas telas novas).
  const bg = await page
    .getByRole("region", { name: "Entregas" })
    .evaluate((el) => window.getComputedStyle(el).backgroundColor);
  expect(bg).not.toBe("rgba(0, 0, 0, 0)");
  await snap("03-projeto");
});

test("a consulta da PR é sob demanda e o merge promove a memória", async () => {
  await openProject();
  const ped7 = page.getByRole("listitem", { name: "PED-7 tentativa 1" });
  await ped7.getByRole("button", { name: "Consultar PR" }).click();
  await expect(page.getByText("A PR segue aberta: nada foi liberado.")).toBeVisible();
  await expect(ped7).toContainText("Draft PR aberta");
  // O humano integra a PR no GitHub (de mentira); só a próxima consulta enxerga.
  await fetch(`${github}/__merge/12`, { method: "POST" });
  await expect(ped7).toContainText("Draft PR aberta");
  await ped7.getByRole("button", { name: "Consultar PR" }).click();
  await expect(page.getByText("Merge confirmado.")).toBeVisible();
  await expect(page.getByText(/Memória: 1 promovida\(s\)/)).toBeVisible();
  await expect(ped7).toContainText("Integrada");
  await expect(ped7).toContainText("Merge: c0ffee012345");
  await snap("04-merge-observado");
  await page.getByRole("tab", { name: "Oficiais" }).click();
  await expect(page.getByRole("listitem", { name: "pedidos usam soft delete" })).toContainText(
    "Promovida no merge c0ffee012345",
  );
  await snap("05-memoria-oficial");
});

test("PR fechada sem merge oferece cancelar, reabrir e nova tentativa", async () => {
  await openProject();
  const ped8 = page.getByRole("listitem", { name: "PED-8 tentativa 1" });
  await ped8.getByRole("button", { name: "Detalhes" }).click();
  await expect(ped8.getByText("A PR foi fechada sem merge. O que fazer?")).toBeVisible();
  for (const name of ["Cancelar", "Reabrir", "Nova tentativa"]) {
    await expect(ped8.getByRole("button", { name })).toBeVisible();
  }
  await snap("06-pr-fechada");
  await ped8.getByLabel("Observação (opcional)").fill("plano mudou");
  await ped8.getByRole("button", { name: "Cancelar", exact: true }).click();
  await expect(page.getByText("Cancelar: registrada.")).toBeVisible();
  await expect(ped8).toContainText("Cancelada");
});

test("comentários da revisão só de autores autorizados viram uma correção", async () => {
  await openProject();
  const ped9 = page.getByRole("listitem", { name: "PED-9 tentativa 1" });
  // Sem autores autorizados, nada é aceito.
  await ped9.getByLabel("Tarefa que recebe a correção").fill("exportar");
  await ped9.getByRole("button", { name: "Buscar comentários da revisão" }).click();
  await expect(
    page.getByText(/Comentários novos: 0; ignorados \(autor não autorizado ou vazio\): 2/),
  ).toBeVisible();
  await expect(page.getByText(/Nenhum autor está autorizado/)).toBeVisible();
  const settings = page.getByRole("region", { name: "Configuração da entrega" });
  await settings.getByLabel(/Autores autorizados/).fill("revisora");
  await settings.getByRole("button", { name: "Salvar configuração" }).click();
  await expect(settings.getByText("Configuração salva.")).toBeVisible();
  await ped9.getByLabel("Tarefa que recebe a correção").fill("exportar");
  await ped9.getByRole("button", { name: "Buscar comentários da revisão" }).click();
  await expect(
    page.getByText(/Comentários novos: 1; ignorados \(autor não autorizado ou vazio\): 1/),
  ).toBeVisible();
  await expect(page.getByText("Correção enfileirada para a próxima vaga.")).toBeVisible();
  // O mesmo comentário não vira outro trabalho.
  await ped9.getByRole("button", { name: "Buscar comentários da revisão" }).click();
  await expect(page.getByText(/Comentários novos: 0/)).toBeVisible();
  await ped9.getByRole("button", { name: "Detalhes" }).click();
  await expect(ped9.getByRole("list", { name: "Comentários" }).getByRole("listitem")).toHaveCount(
    1,
  );
  await expect(ped9).toContainText("documente a coluna removido_em");
  await snap("10-comentarios-da-revisao");
});

test("Central: recusar decisão, revogar autorização e ver a sequência", async () => {
  await page.goto("/decisions");
  const card = page.getByRole("article", { name: "Publicar a branch da feature" });
  await expect(card).toBeVisible();
  await snap("07-central");
  await card.getByRole("button", { name: "Ver sequência" }).click();
  await expect(
    card.getByText("decision · decision_requested — Publicar a branch da feature"),
  ).toBeVisible();
  await card.getByLabel("Observação (opcional)").fill("fora do escopo");
  await card.getByRole("button", { name: "Recusar", exact: true }).click();
  await expect(page.getByText("Nenhuma decisão pendente")).toBeVisible();
  await page.getByRole("tab", { name: "Resolvidas" }).click();
  await expect(page.getByText(/Resolvida em/)).toBeVisible();
  await page.getByRole("tab", { name: "Autorizações" }).click();
  const auth = page.getByRole("article", { name: "git push" });
  await expect(auth).toContainText("Vigente");
  await expect(auth).toContainText("0 de 2");
  await page.getByLabel("Motivo da revogação (opcional)").fill("escopo mudou");
  await auth.getByRole("button", { name: "Revogar" }).click();
  await expect(page.getByText("Autorização revogada.")).toBeVisible();
  await expect(auth).toContainText("Revogada");
  await snap("08-autorizacao-revogada");
});

test("controles do projeto: pausar após a feature atual e retomar", async () => {
  await openProject();
  const controls = page.getByRole("region", { name: "Controles do projeto" });
  await controls.getByRole("button", { name: "Pausar após a feature atual" }).click();
  await expect(controls.getByRole("button", { name: "Retomar projeto" })).toBeVisible();
  await expect(page.getByText(/Pausa após a feature atual/)).toBeVisible();
  await snap("09-pausa");
  await controls.getByRole("button", { name: "Retomar projeto" }).click();
  await expect(controls.getByRole("button", { name: "Pausar após a feature atual" })).toBeVisible();
});

test("fluxo padrão do projeto abre completo, com agentes padrão e sem pendências", async () => {
  await openProject();
  const projectId = page.url().split("/projects/")[1]?.split(/[/?#]/)[0];
  const res = await page.request.get(`/api/projects/${projectId}/workflow`); // provisiona o fluxo padrão
  const { workflow_id: workflowId } = (await res.json()) as { workflow_id: string };
  await page.goto(`/workflows/${workflowId}`);
  await expect(page.getByLabel("Grafo do workflow")).toBeVisible();
  await expect(page.locator(".react-flow__node")).toHaveCount(18);
  await expect(page.getByText("Nenhum problema encontrado.")).toBeVisible();
  await expect(page.locator(".wf-card-summary", { hasText: "tech-lead" })).toBeVisible();
  await page.waitForTimeout(600);
  await snap("12-fluxo-padrao");
});

test("editor de workflows: canvas infinito, arrastar nós, conectar pelas portas e salvar", async () => {
  await page.goto("/workflows");
  await page.getByLabel("Nome").fill("Fluxo do ensaio");
  await page.getByRole("button", { name: "Criar rascunho" }).click();
  const flow = page.getByLabel("Grafo do workflow");
  await expect(flow).toBeVisible();
  await expect(page.locator(".react-flow__node")).toHaveCount(2);
  await expect(page.locator(".react-flow__edge")).toHaveCount(1);
  await page.waitForTimeout(600); // o ajuste inicial do zoom termina

  // Arrastar o nó de início muda a posição e marca o rascunho como alterado.
  const inicio = page.locator('.react-flow__node[data-id="inicio"]');
  const before = await inicio.boundingBox();
  if (!before) throw new Error("nó inicio sem caixa");
  await page.mouse.move(before.x + before.width / 2, before.y + 20);
  await page.mouse.down();
  await page.mouse.move(before.x + before.width / 2 + 40, before.y + 120, { steps: 8 });
  await page.mouse.up();
  const after = await inicio.boundingBox();
  expect(Math.abs((after?.y ?? 0) - before.y)).toBeGreaterThan(40);
  await expect(page.getByText("Alterações não salvas")).toBeVisible();

  // Um agente vem da paleta; puxar a saída do início até a entrada dele cria a conexão.
  await page.getByRole("button", { name: "Agente (IA)", exact: true }).click();
  await expect(page.locator(".react-flow__node")).toHaveCount(3);
  await page.waitForTimeout(300);
  const out = await page
    .locator('.react-flow__handle[data-nodeid="inicio"][data-handleid="out"]')
    .boundingBox();
  const target = await page
    .locator('.react-flow__handle[data-nodeid="agent-1"][data-handleid="in"]')
    .boundingBox();
  if (!out || !target) throw new Error("portas sem caixa");
  await page.mouse.move(out.x + out.width / 2, out.y + out.height / 2);
  await page.mouse.down();
  await page.mouse.move(target.x + target.width / 2, target.y + target.height / 2, { steps: 12 });
  await page.mouse.up();
  await inicio.click({ position: { x: 100, y: 20 } });
  await expect(page.getByLabel("Ligar saída out a")).toHaveValue("agent-1|in");

  await page.getByRole("button", { name: "Salvar rascunho" }).click();
  await expect(page.getByText("Salvo", { exact: true })).toBeVisible();
  await snap("11-editor-workflow");
  await page.reload();
  await expect(page.locator(".react-flow__node")).toHaveCount(3);
});

test("manual de workflows abre, navega pelo índice e fecha", async () => {
  await page.goto("/workflows");
  await page.getByRole("button", { name: "Manual de workflows" }).click();
  const dialog = page.getByRole("dialog", { name: "Como montar um workflow" });
  await expect(dialog).toBeVisible();
  await dialog.getByRole("button", { name: "Os tipos de setas, cores e bolinhas" }).click();
  await expect(
    dialog.getByRole("heading", { name: "Os tipos de setas, cores e bolinhas" }),
  ).toBeInViewport();
  await snap("13-manual-de-workflows");
  await page.keyboard.press("Escape");
  await expect(dialog).toBeHidden();
});

test("nenhum erro de página nem resposta 5xx durante o ensaio", async () => {
  expect(problems).toEqual([]);
});
