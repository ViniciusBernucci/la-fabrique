import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/States";

/** Biblioteca global: agentes, skills, ferramentas, MCP e modelos (épicos E07–E09). */
export function LibraryPage() {
  return (
    <div className="page">
      <PageHeader
        label="Biblioteca"
        title="Biblioteca global"
        lead="Recursos reutilizáveis entre projetos."
      />
      <section className="panel" aria-label="Biblioteca">
        <EmptyState title="Biblioteca vazia">
          Agentes, skills, ferramentas, servidores MCP e modelos serão cadastrados aqui (épicos E07
          a E09).
        </EmptyState>
      </section>
    </div>
  );
}
