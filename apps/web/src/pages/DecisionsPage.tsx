import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/States";

/** Central de Decisões: pedidos humanos pendentes (épico E13, ticket T13-02). */
export function DecisionsPage() {
  return (
    <div className="page">
      <PageHeader
        label="Decisões"
        title="Central de Decisões"
        lead="Aprovações e escolhas que dependem de você."
      />
      <section className="panel" aria-label="Decisões pendentes">
        <EmptyState title="Nenhuma decisão pendente">
          Quando uma execução precisar de aprovação ou decisão humana, ela aparecerá aqui (T13-02).
        </EmptyState>
      </section>
    </div>
  );
}
