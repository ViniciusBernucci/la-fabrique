import { type FormEvent, useState } from "react";
import { FormError, toApiError } from "../components/AuthParts";
import { PageHeader } from "../components/PageHeader";
import { EmptyState, ErrorState, LoadingState } from "../components/States";
import { type ApiError, api, type SkillExecutor, type SkillStep } from "../lib/api";
import { useSession } from "../lib/session";
import { useResource } from "../lib/useResource";

const executorLabel: Record<SkillExecutor, string> = {
  ai: "IA (instruções)",
  script: "Script (comandos)",
  deterministic: "Determinística (passos)",
  hybrid: "Híbrida (IA + script + passos)",
};

/** Formato do pacote a partir do nome do arquivo. */
export function formatOf(name: string): "tar.gz" | "zip" | "md" | null {
  const n = name.toLowerCase();
  if (n.endsWith(".tar.gz") || n.endsWith(".tgz")) return "tar.gz";
  if (n.endsWith(".zip")) return "zip";
  if (n.endsWith(".md")) return "md";
  return null;
}

/** Passos em JSON (lista); devolve a mensagem de erro em português se inválido. */
export function parseSteps(text: string): SkillStep[] | string {
  if (text.trim() === "") return [];
  try {
    const v = JSON.parse(text);
    if (!Array.isArray(v)) return "Os passos devem ser uma lista JSON.";
    return v as SkillStep[];
  } catch {
    return "Os passos não são um JSON válido.";
  }
}

export function SkillsPage() {
  const { expire } = useSession();
  const skills = useResource((s) => api.listSkills(s), "skills", expire);
  const imports = useResource((s) => api.skillImports(s), "skill-imports", expire);
  const reload = () => {
    skills.reload();
    imports.reload();
  };

  return (
    <div className="page">
      <PageHeader
        label="Skills"
        title="Catálogo de skills"
        lead="Cadastre manualmente ou importe um pacote (tar.gz ou zip) ou um SKILL.md. O conteúdo importado é tratado como dado: nada é executado na importação, e scripts fora da política são recusados."
      />
      <ManualForm onSaved={reload} />
      <ImportForm onDone={reload} />
      <section className="panel" aria-label="Lista de skills">
        {skills.state.status === "loading" && <LoadingState label="Carregando skills…" />}
        {skills.state.status === "error" && (
          <ErrorState error={skills.state.error} onRetry={skills.reload} />
        )}
        {skills.state.status === "ready" && skills.state.data.items.length === 0 && (
          <EmptyState title="Nenhuma skill">Cadastre ou importe a primeira acima.</EmptyState>
        )}
        {skills.state.status === "ready" && skills.state.data.items.length > 0 && (
          <ul className="item-list">
            {skills.state.data.items.map((s) => (
              <li key={s.id}>
                <div className="item-link">
                  <span>
                    <strong>{s.name}</strong>
                    <small>
                      {executorLabel[s.executor]} · {s.source_kind}
                      {s.source_ref ? ` (${s.source_ref})` : ""} · versão {s.version} ·{" "}
                      <code>{s.content_hash.slice(0, 10)}</code>
                    </small>
                  </span>
                  {s.needs_approval && <span className="badge badge--paused">Exige aprovação</span>}
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>
      <section className="panel panel--pad" aria-label="Trilha de importações">
        <h2 className="form-title">Importações</h2>
        {imports.state.status === "ready" && imports.state.data.items.length === 0 && (
          <p className="muted">Nenhuma importação ainda.</p>
        )}
        {imports.state.status === "ready" && imports.state.data.items.length > 0 && (
          <ul className="item-list">
            {imports.state.data.items.map((i) => (
              <li key={i.id}>
                <span className="row">
                  <span>
                    {i.outcome === "rejected" ? "Recusada" : "Importada"}
                    {i.name ? `: ${i.name}` : ""} ({i.outcome})
                    {i.reason && <small> — {i.reason}</small>}
                  </span>
                  <small>{new Date(i.created_at).toLocaleString("pt-BR")}</small>
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function ManualForm({ onSaved }: { onSaved: () => void }) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [executor, setExecutor] = useState<SkillExecutor>("ai");
  const [instructions, setInstructions] = useState("");
  const [commands, setCommands] = useState("");
  const [steps, setSteps] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [local, setLocal] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const parsed = parseSteps(steps);
    if (typeof parsed === "string") {
      setLocal(parsed);
      return;
    }
    setLocal(null);
    setBusy(true);
    setError(null);
    try {
      await api.createSkill({
        name,
        description,
        executor,
        instructions,
        commands: commands
          .split("\n")
          .map((c) => c.trim())
          .filter(Boolean),
        steps: parsed,
      });
      setName("");
      setInstructions("");
      setCommands("");
      setSteps("");
      onSaved();
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Nova skill">
      <form className="form form--grid" onSubmit={submit}>
        <h2>Nova skill</h2>
        <label>
          Nome
          <input value={name} maxLength={64} required onChange={(e) => setName(e.target.value)} />
        </label>
        <label>
          Executor
          <select value={executor} onChange={(e) => setExecutor(e.target.value as SkillExecutor)}>
            {Object.entries(executorLabel).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </select>
        </label>
        <label>
          Descrição
          <input
            value={description}
            maxLength={500}
            onChange={(e) => setDescription(e.target.value)}
          />
        </label>
        <label className="form-wide">
          Instruções (SKILL.md)
          <textarea
            rows={4}
            value={instructions}
            onChange={(e) => setInstructions(e.target.value)}
          />
        </label>
        <label className="form-wide">
          Comandos (um por linha; skill de script)
          <textarea rows={2} value={commands} onChange={(e) => setCommands(e.target.value)} />
        </label>
        <label className="form-wide">
          Passos (JSON; skill determinística ou híbrida)
          <textarea rows={3} value={steps} onChange={(e) => setSteps(e.target.value)} />
        </label>
        {local && (
          <p className="form-error" role="alert">
            {local}
          </p>
        )}
        <FormError error={error} />
        <div className="actions">
          <button type="submit" disabled={busy}>
            Cadastrar skill
          </button>
        </div>
      </form>
    </section>
  );
}

function ImportForm({ onDone }: { onDone: () => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [replace, setReplace] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [local, setLocal] = useState<string | null>(null);
  const [done, setDone] = useState<string | null>(null);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setDone(null);
    if (!file) return;
    const format = formatOf(file.name);
    if (!format) {
      setLocal("Use um arquivo .tar.gz, .tgz, .zip ou .md.");
      return;
    }
    setLocal(null);
    setBusy(true);
    setError(null);
    try {
      const skill = await api.importSkill(file, format, replace, file.name);
      setDone(`Skill ${skill.name} importada (versão ${skill.version}).`);
      setFile(null);
    } catch (e) {
      setError(toApiError(e));
    } finally {
      setBusy(false);
      onDone();
    }
  }

  return (
    <section className="panel panel--pad" aria-label="Importar skill">
      <form className="form form--grid" onSubmit={submit}>
        <h2>Importar pacote</h2>
        <label>
          Arquivo (.tar.gz, .zip ou SKILL.md)
          <input
            type="file"
            accept=".gz,.tgz,.zip,.md"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </label>
        <label className="check">
          <input type="checkbox" checked={replace} onChange={(e) => setReplace(e.target.checked)} />
          Substituir se já existir com conteúdo diferente
        </label>
        {local && (
          <p className="form-error" role="alert">
            {local}
          </p>
        )}
        <FormError error={error} />
        {done && (
          <p className="callout" role="status">
            {done}
          </p>
        )}
        <div className="actions">
          <button type="submit" disabled={busy || !file}>
            Importar
          </button>
        </div>
      </form>
    </section>
  );
}
