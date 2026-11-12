import { PageHeader } from "../components/PageHeader";
import { EmptyState } from "../components/States";

function greeting(now: Date): string {
  const hour = now.getHours();
  return hour < 12 ? "Bom dia" : hour < 18 ? "Boa tarde" : "Boa noite";
}

/** Painel inicial: vazio por enquanto; os blocos entram com os épicos de operação (E16). */
export function HomePage() {
  const now = new Date();
  const date = now.toLocaleDateString("pt-BR", {
    weekday: "short",
    day: "numeric",
    month: "short",
  });
  return (
    <div className="page">
      <PageHeader
        label={`${date} · ${greeting(now)}`}
        title="Central de comando"
        lead="Os indicadores e a operação aparecem aqui conforme forem construídos."
      />
      <section className="panel" aria-label="Painel">
        <EmptyState title="Nada para mostrar ainda">
          O painel começa limpo. Cada seção será adicionada junto com a funcionalidade que a
          alimenta.
        </EmptyState>
      </section>
    </div>
  );
}
