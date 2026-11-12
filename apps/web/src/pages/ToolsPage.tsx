import { PageHeader } from "../components/PageHeader";
import { ErrorState, LoadingState } from "../components/States";
import { api, type Tool } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const kindLabel: Record<Tool["kind"], string> = {
  native: "Nativa",
  script: "Script",
  integration: "Integração",
};

/** Resumo de uma assinatura: "path*: path, max_bytes: int" (obrigatórios com asterisco). */
export function signature(fields: Tool["input"]): string {
  return fields.length === 0
    ? "sem parâmetros"
    : fields.map((f) => `${f.name}${f.required ? "*" : ""}: ${f.type}`).join(", ");
}

export function ToolsPage() {
  const { expire } = useSession();
  const { state, reload } = useResource((s) => api.listTools(s), "tools", expire);
  return (
    <div className="page">
      <PageHeader
        label="Ferramentas"
        title="Catálogo de ferramentas"
        lead="Ferramentas nativas e personalizadas, com parâmetros, saída tipada, permissões e limites. Uma ferramenta só funciona para um agente quando é concedida a ele; parâmetro ou escopo indevido falha antes de qualquer efeito."
      />
      {state.status === "loading" && <LoadingState label="Carregando ferramentas…" />}
      {state.status === "error" && <ErrorState error={state.error} onRetry={reload} />}
      {state.status === "ready" && (
        <section className="panel" aria-label="Lista de ferramentas">
          <ul className="item-list">
            {state.data.items.map((t) => (
              <li key={t.name}>
                <div className="item-link">
                  <span>
                    <strong>{t.name}</strong>
                    <small>{t.description}</small>
                    <small>
                      Entrada: {signature(t.input)} · Saída: {signature(t.output)}
                    </small>
                    <small>
                      Permissões: {t.permissions.join(", ")} · {t.limits.timeout_seconds} s ·{" "}
                      {Math.round(t.limits.max_output_bytes / 1024)} KiB
                      {t.requires?.length ? ` · exige ${t.requires.join(", ")} na imagem` : ""}
                    </small>
                  </span>
                  <span className="badge">{kindLabel[t.kind]}</span>
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  );
}
