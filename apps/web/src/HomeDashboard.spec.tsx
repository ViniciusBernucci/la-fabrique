import { renderToStaticMarkup } from "react-dom/server";
import { expect, it } from "vitest";
import { HomeDashboard } from "./HomeDashboard";

it("identifies synthetic information and renders the accessible orchestration core without administrative data", () => {
  for (const layout of ["novo", "antigo"] as const) {
    const html = renderToStaticMarkup(
      <HomeDashboard onNavigate={() => undefined} layout={layout} />,
    );
    expect(html).toContain("Central de controle La fabrique");
    expect(html).toContain("dados simulados");
    expect(html).toContain('aria-label="Núcleos, centro de orquestração da La fabrique"');
    expect(html).not.toContain("JARVIS");
    expect(html).not.toContain("/images/control-room-reference.png");
    expect(html).not.toContain("adminToken");
    expect(html).not.toContain("Bearer");
  }
});

it("keeps the HUD home available as the old layout", () => {
  const html = renderToStaticMarkup(<HomeDashboard onNavigate={() => undefined} layout="antigo" />);
  expect(html).toContain("Mensagens importantes");
  expect(html).toContain('data-layout="antigo"');
  expect(html).toMatch(/aria-pressed="true"[^>]*>Antigo</);
});

it("shows the sober home with the pending decision, metrics and orchestration", () => {
  const html = renderToStaticMarkup(<HomeDashboard onNavigate={() => undefined} layout="novo" />);
  expect(html).toContain("Requer sua decisão");
  expect(html).toContain("Pagamento Hotmart não processado");
  expect(html).toContain("Orquestração");
  expect(html).toContain("Indicadores de demonstração");
  expect(html).not.toContain("Mensagens importantes");
  expect(html).toMatch(/aria-pressed="true"[^>]*>Novo</);
});

it("keeps collapsed navigation accessible and distinguishes activity from quota", () => {
  for (const layout of ["novo", "antigo"] as const) {
    const html = renderToStaticMarkup(
      <HomeDashboard onNavigate={() => undefined} layout={layout} />,
    );
    expect(html).toContain('aria-label="Projetos"');
    expect(html).toContain('aria-label="Configurações"');
    expect(html).toContain('aria-label="Núcleos"');
    expect(html).toContain('aria-current="page"');
    expect(html).toContain('aria-label="Atividade simulada de Núcleos"');
    expect(html).toContain('aria-labelledby="home-preview-title"');
  }
});
