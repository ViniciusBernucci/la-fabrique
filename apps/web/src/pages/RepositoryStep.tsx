import { type FormEvent, useEffect, useState } from "react";
import { toApiError } from "../components/AuthParts";
import { ConnectionSelect } from "../components/ConnectionSelect";
import {
  type ApiError,
  api,
  type Connection,
  type GitHubRepo,
  type PrepareRepository,
  type Project,
  type RepoMode,
  type WorkspaceListing,
} from "../lib/api";

const modes: { id: RepoMode; title: string; hint: string }[] = [
  { id: "github", title: "Clonar do GitHub", hint: "Escolha um repositório da conta conectada." },
  { id: "local", title: "Pasta existente", hint: "Use um repositório que já está em uma pasta." },
  { id: "new", title: "Criar novo", hint: "Comece do zero, com ou sem repositório no GitHub." },
];

/** Etapa Repositório: três caminhos para chegar a um repositório (ADR-0061). */
export function RepositoryStep({
  project,
  connections,
  busy,
  locked,
  onBack,
  onPrepare,
}: {
  project: Project;
  connections: Connection[];
  busy: boolean;
  locked: boolean;
  onBack: () => void;
  onPrepare: (body: PrepareRepository) => void;
}) {
  const initial = project.repository;
  const [mode, setMode] = useState<RepoMode>(initial.mode || "github");
  const [owner, setOwner] = useState(initial.owner);
  const [name, setName] = useState(initial.name || (initial.mode === "new" ? "" : ""));
  const [base, setBase] = useState(initial.base_branch);
  const [conn, setConn] = useState<string | null>(initial.connection_id);
  const [path, setPath] = useState(initial.mode === "github" ? "" : initial.path);
  const [createRemote, setCreateRemote] = useState(false);
  const [isPrivate, setIsPrivate] = useState(true);

  function submit(event: FormEvent) {
    event.preventDefault();
    const body: PrepareRepository = { mode, connection_id: conn };
    if (mode === "github") {
      Object.assign(body, { owner, name, base_branch: base });
    } else if (mode === "local") {
      Object.assign(body, { path, owner, name, base_branch: base });
    } else {
      Object.assign(body, {
        path: path || undefined,
        owner: createRemote ? owner : undefined,
        name: name || undefined,
        base_branch: base || undefined,
        create_remote: createRemote,
        private: isPrivate,
      });
    }
    onPrepare(body);
  }

  const needsConn = mode === "github" || (mode === "new" && createRemote);
  const ready =
    mode === "github"
      ? Boolean(owner && name && base)
      : mode === "local"
        ? Boolean(path)
        : !createRemote || Boolean(conn);

  return (
    <form className="form form--grid" onSubmit={submit}>
      <h2>Repositório do projeto</h2>
      <fieldset className="fieldset-plain" disabled={locked}>
        <div className="mode-picker" role="radiogroup" aria-label="Como trazer o repositório">
          {modes.map((m) => (
            <label key={m.id} className={`mode-card${mode === m.id ? " mode-card--on" : ""}`}>
              <input
                type="radio"
                name="repo-mode"
                value={m.id}
                checked={mode === m.id}
                onChange={() => setMode(m.id)}
              />
              <strong>{m.title}</strong>
              <small>{m.hint}</small>
            </label>
          ))}
        </div>

        {mode === "github" && (
          <GitHubPicker
            conn={conn}
            onPick={(r) => {
              setOwner(r.owner);
              setName(r.name);
              setBase(r.default_branch || "main");
            }}
          />
        )}
        {mode === "local" && <FolderPicker value={path} onChange={setPath} />}
        {mode === "new" && (
          <>
            <label>
              Pasta (dentro da área de workspaces)
              <input
                value={path}
                maxLength={300}
                placeholder={project.slug}
                onChange={(e) => setPath(e.target.value)}
              />
              <small className="muted">
                Vazio usa “{project.slug}”. A pasta deve não existir ou estar vazia.
              </small>
            </label>
            <label className="check">
              <input
                type="checkbox"
                checked={createRemote}
                onChange={(e) => setCreateRemote(e.target.checked)}
              />
              Criar também o repositório no GitHub
            </label>
          </>
        )}

        {(mode !== "new" || createRemote) && (
          <ConnectionSelect
            label="Conexão GitHub"
            kind="github"
            connections={connections}
            value={conn}
            onChange={setConn}
          />
        )}
        {needsConn && !conn && (
          <small className="muted">Escolha a conta GitHub que será usada.</small>
        )}

        {mode !== "new" || createRemote ? (
          <>
            <label>
              {mode === "new" ? "Dono (vazio = sua conta)" : "Dono (usuário ou organização)"}
              <input value={owner} maxLength={39} onChange={(e) => setOwner(e.target.value)} />
            </label>
            <label>
              Repositório
              <input
                value={name}
                maxLength={100}
                placeholder={mode === "new" ? project.slug : ""}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
          </>
        ) : null}
        <label>
          Branch base
          <input
            value={base}
            maxLength={200}
            placeholder="main"
            onChange={(e) => setBase(e.target.value)}
          />
          {mode === "local" && <small className="muted">Vazio usa a branch atual da pasta.</small>}
        </label>
        {mode === "new" && createRemote && (
          <label>
            Visibilidade
            <select
              value={isPrivate ? "private" : "public"}
              onChange={(e) => setIsPrivate(e.target.value === "private")}
            >
              <option value="private">Privado</option>
              <option value="public">Público</option>
            </select>
          </label>
        )}
      </fieldset>
      <div className="actions">
        <button type="button" className="secondary-action" onClick={onBack}>
          Voltar
        </button>
        <button type="submit" disabled={busy || locked || !ready || (needsConn && !conn)}>
          {mode === "github"
            ? "Clonar e continuar"
            : mode === "new"
              ? "Criar e continuar"
              : "Usar e continuar"}
        </button>
      </div>
    </form>
  );
}

/** Lista os repositórios da conta conectada e preenche dono, nome e branch. */
function GitHubPicker({ conn, onPick }: { conn: string | null; onPick: (r: GitHubRepo) => void }) {
  const [repos, setRepos] = useState<GitHubRepo[] | null>(null);
  const [error, setError] = useState<ApiError | null>(null);
  const [filter, setFilter] = useState("");

  useEffect(() => {
    setRepos(null);
    setError(null);
    if (!conn) return;
    const ctl = new AbortController();
    api
      .githubRepos(conn, ctl.signal)
      .then((r) => setRepos(r.items))
      .catch((e) => {
        if (!ctl.signal.aborted) setError(toApiError(e));
      });
    return () => ctl.abort();
  }, [conn]);

  if (!conn) {
    return <small className="muted">Escolha a conexão GitHub para listar seus repositórios.</small>;
  }
  if (error) {
    return (
      <p className="callout callout--warn" role="alert">
        Não consegui listar os repositórios: {error.message} Você ainda pode preencher dono e nome
        abaixo.
      </p>
    );
  }
  if (!repos) return <small className="muted">Carregando repositórios…</small>;
  const q = filter.trim().toLowerCase();
  const shown = repos.filter((r) => r.full_name.toLowerCase().includes(q)).slice(0, 30);
  return (
    <div className="repo-picker">
      <label>
        Buscar repositório
        <input value={filter} placeholder="dono/nome" onChange={(e) => setFilter(e.target.value)} />
      </label>
      <ul className="item-list repo-list" aria-label="Repositórios da conta">
        {shown.length === 0 && <li className="muted">Nenhum repositório encontrado.</li>}
        {shown.map((r) => (
          <li key={r.full_name}>
            <button type="button" className="repo-item" onClick={() => onPick(r)}>
              <strong>{r.full_name}</strong>
              <small>
                {r.private ? "privado" : "público"}
                {r.can_push ? "" : " · somente leitura"}
                {r.description ? ` · ${r.description}` : ""}
              </small>
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}

/** Navega pelas pastas da área de workspaces e escolhe um repositório git existente. */
function FolderPicker({ value, onChange }: { value: string; onChange: (p: string) => void }) {
  const [dir, setDir] = useState("");
  const [listing, setListing] = useState<WorkspaceListing | null>(null);
  const [error, setError] = useState<ApiError | null>(null);

  useEffect(() => {
    const ctl = new AbortController();
    setError(null);
    api
      .browseWorkspace(dir, ctl.signal)
      .then(setListing)
      .catch((e) => {
        if (!ctl.signal.aborted) setError(toApiError(e));
      });
    return () => ctl.abort();
  }, [dir]);

  if (error) {
    return (
      <p className="callout callout--warn" role="alert">
        {error.message}
      </p>
    );
  }
  if (!listing) return <small className="muted">Carregando pastas…</small>;
  if (!listing.enabled) {
    return (
      <p className="callout callout--warn" role="status">
        A área de workspaces não está configurada no servidor (WORKSPACES_DIR). Configure-a e monte
        a pasta dos seus projetos para usar repositórios existentes.
      </p>
    );
  }
  return (
    <div className="folder-picker">
      <p className="muted">
        Pasta atual: <code>/{listing.path}</code>
        {listing.parent !== null && (
          <>
            {" "}
            <button
              type="button"
              className="link-button"
              onClick={() => setDir(listing.parent ?? "")}
            >
              ↑ subir
            </button>
          </>
        )}
      </p>
      {listing.is_repo && (
        <button
          type="button"
          className={value === listing.path ? "" : "secondary-action"}
          onClick={() => onChange(listing.path)}
        >
          Usar esta pasta
        </button>
      )}
      <ul className="item-list repo-list" aria-label="Pastas">
        {listing.entries.length === 0 && <li className="muted">Nenhuma subpasta.</li>}
        {listing.entries.map((e) => (
          <li key={e.path} className="repo-row">
            <button type="button" className="repo-item" onClick={() => setDir(e.path)}>
              <strong>{e.name}</strong>
              <small>{e.is_repo ? "repositório git" : e.empty ? "vazia" : "pasta"}</small>
            </button>
            {e.is_repo && (
              <button
                type="button"
                className={value === e.path ? "" : "secondary-action"}
                onClick={() => onChange(e.path)}
              >
                {value === e.path ? "Selecionada" : "Usar"}
              </button>
            )}
          </li>
        ))}
      </ul>
      {value && (
        <p role="status">
          Selecionada: <code>/{value}</code>
        </p>
      )}
    </div>
  );
}
