import { renderToStaticMarkup } from "react-dom/server";
import { expect, it } from "vitest";
import { OfficePanel } from "./OfficePanel";
import { createMockOfficeFeed } from "./office-feed";

it("renderiza o escritório como imagem acessível e lista os agentes simulados", () => {
  const html = renderToStaticMarkup(<OfficePanel feed={createMockOfficeFeed()} />);
  expect(html).toContain("Escritório virtual");
  expect(html).toContain('role="img"');
  expect(html).toContain("6 agentes e 4 mesas livres");
  expect(html).toContain("Dados simulados");
  expect(html).toContain("Alex");
});
