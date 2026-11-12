import { type FormEvent, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { type ApiError, api } from "../lib/api";

/** Sugere um identificador a partir do nome (minúsculas, sem acento, hífens). */
export function suggestSlug(name: string): string {
  return name
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 63)
    .replace(/-+$/g, "");
}

/** Etapa 1 do cadastro: cria o rascunho, vazio de vínculos, e segue para o assistente. */
export function NewProjectPage() {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [description, setDescription] = useState("");
  const [priority, setPriority] = useState(2);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const project = await api.createProject({ name, slug, description, priority });
      navigate(`/projects/${project.id}/setup`);
    } catch (e) {
      setError(toApiError(e));
      setBusy(false);
    }
  }

  return (
    <div className="page">
      <nav className="breadcrumb" aria-label="Trilha">
        <Link to="/projects">Projetos</Link>
        <span aria-hidden="true">/</span>
        <span>Novo</span>
      </nav>
      <PageHeader
        label="Cadastro"
        title="Novo projeto"
        lead="O projeto nasce em rascunho, sem agentes, workflows ou recursos associados. Você escolhe cada vínculo nas próximas etapas."
      />
      <section className="panel panel--pad" aria-label="Dados do projeto">
        <form className="form form--grid" onSubmit={submit}>
          <label>
            Nome
            <input
              value={name}
              maxLength={120}
              required
              onChange={(e) => {
                setName(e.target.value);
                if (!slugEdited) setSlug(suggestSlug(e.target.value));
              }}
            />
          </label>
          <label>
            Identificador
            <input
              value={slug}
              maxLength={63}
              required
              pattern="[a-z0-9]+(-[a-z0-9]+)*"
              title="Letras minúsculas, números e hífens"
              onChange={(e) => {
                setSlug(e.target.value);
                setSlugEdited(true);
              }}
            />
          </label>
          <label>
            Prioridade na fila
            <select value={priority} onChange={(e) => setPriority(Number(e.target.value))}>
              {[0, 1, 2, 3, 4].map((n) => (
                <option key={n} value={n}>
                  {n}
                  {n === 0 ? " (mais baixa)" : n === 4 ? " (mais alta)" : ""}
                </option>
              ))}
            </select>
          </label>
          <label>
            Descrição (opcional)
            <input
              value={description}
              maxLength={500}
              onChange={(e) => setDescription(e.target.value)}
            />
          </label>
          <FormError error={error} />
          <div className="actions">
            <button type="submit" disabled={busy}>
              Criar rascunho e continuar
            </button>
            <Link className="button-link secondary-action" to="/projects">
              Cancelar
            </Link>
          </div>
        </form>
      </section>
    </div>
  );
}
