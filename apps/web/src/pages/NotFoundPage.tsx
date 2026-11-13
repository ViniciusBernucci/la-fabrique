import { Link } from "react-router-dom";
import { EmptyState } from "../components/States";

export function NotFoundPage() {
  return (
    <div className="page">
      <section className="panel">
        <EmptyState
          title="Página não encontrada"
          action={
            <Link className="button-link secondary-action" to="/">
              Ir para o painel
            </Link>
          }
        >
          O endereço não corresponde a nenhuma área do sistema.
        </EmptyState>
      </section>
    </div>
  );
}
