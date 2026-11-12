import { renderToStaticMarkup } from "react-dom/server";
import { expect, it } from "vitest";
import { DashboardLayout } from "./DashboardLayout";

it("shares the navigation frame with custom content and marks the active destination", () => {
  const html = renderToStaticMarkup(
    <DashboardLayout onNavigate={() => undefined} activeDestination="settings">
      <h1>Conteúdo administrativo</h1>
    </DashboardLayout>,
  );
  expect(html).toContain("Conteúdo administrativo");
  expect(html).toContain('class="workspace-content"');
  expect(html).toContain('class="home-sidebar"');
  expect(html).toContain('class="home-topbar"');
  expect(html).toMatch(/aria-label="Configurações"[^>]*class="is-active" aria-current="page"/);
  expect(html).not.toContain("Escritório virtual da equipe");
});

it("keeps the command frame and settings rail on every destination", () => {
  for (const destination of ["control", "tasks", "settings"] as const) {
    const html = renderToStaticMarkup(
      <DashboardLayout onNavigate={() => undefined} activeDestination={destination}>
        <p>Área</p>
      </DashboardLayout>,
    );
    expect(html).toContain('aria-label="Buscar atalhos no sistema"');
    expect(html).toContain('aria-label="Expandir menu"');
    expect(html).toContain('aria-label="Atalhos de configurações"');
    expect(html).toContain('aria-label="Configurações: Equipe"');
    expect(html).toContain('aria-labelledby="home-preview-title"');
  }
});

it("exposes the user account entry and a real users destination", () => {
  const html = renderToStaticMarkup(
    <DashboardLayout onNavigate={() => undefined} activeDestination="users">
      <p>Team</p>
    </DashboardLayout>,
  );
  expect(html).toContain('aria-label="Abrir minha conta"');
  expect(html).toContain('aria-label="Usuários"');
  expect(html).not.toContain("Perfil de demonstração");
  expect(html).toContain("Perfil e equipe");
});

it("abre o escritório virtual como destino próprio", () => {
  const html = renderToStaticMarkup(
    <DashboardLayout onNavigate={() => undefined} activeDestination="office">
      <p>Escritório</p>
    </DashboardLayout>,
  );
  expect(html).toMatch(/aria-label="Escritório"[^>]*class="is-active" aria-current="page"/);
});
