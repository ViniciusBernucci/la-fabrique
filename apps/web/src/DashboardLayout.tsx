import type { ReactNode } from "react";
import { useEffect, useRef, useState } from "react";
import { type LayoutChoice, useLayoutPreference } from "./layout-preference";
import { OrchestrationCore } from "./OrchestrationCore";
import { SobrioHome } from "./SobrioHome";
import "./home-dashboard.css";

export type HomeDestination = "control" | "settings" | "tasks" | "account" | "users" | "office";
type IconName =
  | "home"
  | "folder"
  | "robot"
  | "content"
  | "chart"
  | "users"
  | "calendar"
  | "star"
  | "money"
  | "settings"
  | "search"
  | "bell"
  | "help"
  | "plus"
  | "arrow"
  | "clock"
  | "cloud"
  | "message";
const paths: Record<IconName, string> = {
  home: "M3 11 12 3l9 8M5 10v11h5v-7h4v7h5V10",
  folder: "M3 7V4h7l3 3h8v14H3V7Zm0 3h18",
  robot: "M12 3v3m-2-3h4M5 8h14v12H5V8ZM2 12v5m20-5v5M9 12v2m6-2v2m-6 3h6",
  content: "M5 3h14v18H5V3Zm4 5h6m-6 4h6m-6 4h4",
  chart: "M4 20v-6h3v6m4 0V9h3v11m4 0V4h3v16",
  users:
    "M9 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-7 9v-3c0-5 14-5 14 0v3H2ZM17 4a4 4 0 0 1 0 8m2 3c3 0 3 3 3 6",
  calendar: "M4 5h16v16H4V5ZM8 2v6m8-6v6M4 10h16",
  star: "m12 2 3 6 7 1-5 5 1 8-6-4-6 4 1-8-5-5 7-1 3-6Z",
  money: "M16 6c-7-5-13 4-4 6s3 11-4 6m4-16v20",
  settings:
    "m9 3 6 0 1 4 4 1 2 5-3 3 0 4-5 2-3-3-4 0-2-5 3-3 0-4 4-2Zm3 6a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z",
  search: "M10 3a7 7 0 1 0 0 14 7 7 0 0 0 0-14Zm5 12 6 6",
  bell: "M5 17h14l-2-4V9a5 5 0 0 0-10 0v4l-2 4Zm5 3h4M12 2v2",
  help: "M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm-3 6c0-4 7-4 6 0-1 3-3 2-3 6m0 3v1",
  plus: "M12 4v16M4 12h16",
  arrow: "M4 12h16m-6-6 6 6-6 6",
  clock: "M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20Zm0 4v6l4 3",
  cloud: "M6 19h12c7-2 3-10-2-9-1-9-13-7-12 1-5 1-4 8 2 8Z",
  message: "M3 4h18v14H8l-5 4V4Zm4 5h10m-10 4h7",
};
function Icon({ name }: { name: IconName }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={paths[name]} />
    </svg>
  );
}
function Avatar({ robot = false }: { robot?: boolean }) {
  return (
    <svg className="home-avatar" viewBox="0 0 48 48" aria-hidden="true">
      <defs>
        <linearGradient id={robot ? "robot-bg" : "person-bg"} x2="1" y2="1">
          <stop stopColor={robot ? "var(--hud-teal)" : "var(--hud-amber)"} />
          <stop offset="1" stopColor="var(--hud-surface-solid)" />
        </linearGradient>
      </defs>
      <circle
        cx="24"
        cy="24"
        r="22"
        fill={`url(#${robot ? "robot-bg" : "person-bg"})`}
        stroke="var(--hud-text-strong)"
        strokeWidth="2"
      />
      {robot ? (
        <>
          <path d="M16 38v-8h16v8" fill="var(--hud-text-strong)" />
          <rect x="9" y="10" width="30" height="24" rx="10" fill="var(--hud-text-strong)" />
          <rect x="13" y="14" width="22" height="15" rx="6" fill="var(--hud-surface-solid)" />
          <circle cx="19" cy="21" r="3" fill="var(--hud-teal)" />
          <circle cx="29" cy="21" r="3" fill="var(--hud-teal)" />
          <path d="M24 6v4" stroke="var(--hud-text-strong)" strokeWidth="3" />
        </>
      ) : (
        <>
          <path d="M10 45c1-17 27-17 28 0" fill="var(--hud-text-strong)" />
          <ellipse cx="24" cy="24" rx="11" ry="13" fill="var(--hud-amber)" />
          <path d="M12 25C4 4 39 2 37 26l-6-10-8 2-7-3-4 10" fill="var(--hud-surface-solid)" />
          <circle cx="20" cy="25" r="1.6" fill="var(--hud-surface-solid)" />
          <circle cx="29" cy="25" r="1.6" fill="var(--hud-surface-solid)" />
          <path d="M21 32q4 3 7-1" stroke="var(--hud-text-strong)" strokeWidth="1.5" fill="none" />
        </>
      )}
    </svg>
  );
}
const navigation: { name: string; icon: IconName; destination?: HomeDestination }[] = [
  { name: "Painel", icon: "home" },
  { name: "Projetos", icon: "folder", destination: "control" },
  { name: "Tarefas", icon: "content", destination: "tasks" },
  { name: "Agentes de IA", icon: "robot", destination: "settings" },
  { name: "Núcleos", icon: "robot" },
  { name: "Pipeline", icon: "chart" },
  { name: "Revisões", icon: "content" },
  { name: "Repositórios", icon: "folder" },
  { name: "Histórico", icon: "clock" },
  { name: "Escritório", icon: "home", destination: "office" },
  { name: "Usuários", icon: "users", destination: "users" },
  { name: "Configurações", icon: "settings", destination: "settings" },
];
const shortcuts: {
  name: string;
  description: string;
  icon: IconName;
  color: string;
  destination?: HomeDestination;
}[] = [
  {
    name: "Novo Projeto",
    description: "Criar workspace",
    icon: "plus",
    color: "blue",
    destination: "control",
  },
  {
    name: "Agentes IA",
    description: "Gerenciar agentes",
    icon: "robot",
    color: "silver",
    destination: "settings",
  },
  { name: "Calendário", description: "Eventos e agendas", icon: "calendar", color: "red" },
  { name: "Conteúdos", description: "Vídeos e posts", icon: "content", color: "purple" },
  { name: "CRM / Leads", description: "Pipeline de vendas", icon: "users", color: "cyan" },
  { name: "Financeiro", description: "Receitas e despesas", icon: "money", color: "green" },
  {
    name: "Oportunidades",
    description: "Rádios, playlists, eventos",
    icon: "star",
    color: "yellow",
  },
  { name: "Relatórios", description: "Desempenho geral", icon: "chart", color: "teal" },
  {
    name: "Configurações",
    description: "Sistema e integrações",
    icon: "settings",
    color: "silver",
    destination: "settings",
  },
];
const notifications: {
  title: string;
  description?: string;
  time: string;
  icon: IconName;
  color: string;
}[] = [
  {
    title: "Pagamento Hotmart não processado",
    time: "Agora há pouco",
    icon: "clock",
    color: "red",
  },
  {
    title: "Novo lead qualificado",
    description: "João Silva · Workshop Produção Musical",
    time: "1 hora atrás",
    icon: "calendar",
    color: "orange",
  },
  {
    title: "Geração de conteúdo concluída",
    description: "5 posts prontos para revisão",
    time: "2 horas atrás",
    icon: "cloud",
    color: "blue",
  },
  {
    title: "Relatório semanal disponível",
    description: "Desempenho dos projetos",
    time: "3 horas atrás",
    icon: "chart",
    color: "green",
  },
];
const agents = [
  {
    name: "Núcleos",
    role: " (Orquestrador)",
    status: "Online",
    progress: 100,
    color: "green",
    robot: true,
  },
  { name: "Marketing", status: "Executando tarefas", progress: 75, color: "purple" },
  { name: "Financeiro", status: "Aguardando fila", progress: 25, color: "orange" },
  { name: "Conteúdo", status: "Processando vídeos", progress: 60, color: "yellow" },
  { name: "Desenvolvimento", status: "Online", progress: 90, color: "green" },
];
const recentProjects = [
  {
    name: "Meu Estúdio",
    description: "Produção musical e cursos",
    icon: "star" as const,
    color: "orange",
    status: "Ativo",
  },
  {
    name: "Agência Artística",
    description: "Divulgação e eventos",
    icon: "star" as const,
    color: "orange",
    status: "Ativo",
  },
  {
    name: "La fabrique",
    description: "Desenvolvimento do sistema",
    icon: "robot" as const,
    color: "purple",
    status: "Em desenvolvimento",
  },
  {
    name: "IA ENEM",
    description: "Plataforma educacional",
    icon: "folder" as const,
    color: "blue",
    status: "Pausado",
  },
];

export function DashboardLayout({
  onNavigate,
  onHome,
  activeDestination,
  children,
  layout: layoutOverride,
}: {
  onNavigate: (destination: HomeDestination) => void;
  onHome?: () => void;
  activeDestination?: HomeDestination;
  children?: ReactNode;
  /** Força um visual (testes/demonstração); por padrão usa a escolha salva no seletor. */
  layout?: LayoutChoice;
}) {
  const [storedLayout, setLayout] = useLayoutPreference();
  const layout = layoutOverride ?? storedLayout;
  const [sidebarCollapsed, setSidebarCollapsed] = useState(true);
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  const greeting =
    now.getHours() < 12 ? "Bom dia" : now.getHours() < 18 ? "Boa tarde" : "Boa noite";
  useEffect(() => {
    const smallScreen = window.matchMedia("(max-width: 760px)");
    const collapseOnSmallScreen = () => {
      if (smallScreen.matches) setSidebarCollapsed(true);
    };
    smallScreen.addEventListener("change", collapseOnSmallScreen);
    return () => smallScreen.removeEventListener("change", collapseOnSmallScreen);
  }, []);
  function toggleSidebar() {
    setSidebarCollapsed((collapsed) => !collapsed);
  }
  const [query, setQuery] = useState("");
  const [preview, setPreview] = useState<string | null>(null);
  const dialog = useRef<HTMLDialogElement>(null);
  const search = useRef<HTMLInputElement>(null);
  useEffect(() => {
    if (preview) dialog.current?.showModal();
  }, [preview]);
  useEffect(() => {
    function shortcut(event: KeyboardEvent) {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        search.current?.focus();
      }
    }
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, []);
  const filtered = shortcuts.filter((item) =>
    `${item.name} ${item.description}`
      .toLocaleLowerCase("pt-BR")
      .includes(query.toLocaleLowerCase("pt-BR")),
  );
  function open(name: string, destination?: HomeDestination) {
    if (destination) {
      setQuery("");
      onNavigate(destination);
    } else setPreview(name);
  }
  function panelHeading(title: string, action: () => void) {
    return (
      <div className="home-panel-heading">
        <h2>{title}</h2>
        <button type="button" onClick={action}>
          Ver {title === "Notificações importantes" ? "todas" : "todos"}
          <Icon name="arrow" />
        </button>
      </div>
    );
  }
  function goHome() {
    setQuery("");
    onHome?.();
  }
  function isActive(name: string) {
    return (
      name ===
      (activeDestination === "control"
        ? "Projetos"
        : activeDestination === "settings"
          ? "Configurações"
          : activeDestination === "users"
            ? "Usuários"
            : activeDestination === "account"
              ? "Minha conta"
              : activeDestination === "tasks"
                ? "Tarefas"
                : activeDestination === "office"
                  ? "Escritório"
                  : "Painel")
    );
  }
  return (
    <div
      className={`home-dashboard${sidebarCollapsed ? " is-menu-collapsed" : ""}`}
      data-layout={layout}
    >
      <a className="home-skip" href="#home-content">
        Ir para o conteúdo
      </a>
      <aside id="home-sidebar" className="home-sidebar" aria-label="Menu principal">
        <button
          type="button"
          className="home-menu-toggle"
          aria-label={sidebarCollapsed ? "Expandir menu" : "Recolher menu"}
          title={sidebarCollapsed ? "Expandir menu" : "Recolher menu"}
          aria-expanded={!sidebarCollapsed}
          aria-controls="home-sidebar"
          onClick={toggleSidebar}
        >
          <Icon name="arrow" />
        </button>
        <nav>
          {navigation.map((item) => (
            <button
              type="button"
              key={item.name}
              aria-label={item.name}
              title={item.name}
              className={isActive(item.name) ? "is-active" : ""}
              aria-current={isActive(item.name) ? "page" : undefined}
              onClick={() =>
                item.name === "Painel" ? goHome() : open(item.name, item.destination)
              }
            >
              <Icon name={item.icon} />
              <span>{item.name}</span>
              {item.name === "Revisões" && (
                <small className="home-nav-badge" title="Prévia demonstrativa">
                  Prévia
                </small>
              )}
            </button>
          ))}
        </nav>
        <div className="home-workspace">
          <span>Workspace de demonstração</span>
          <div>
            <button type="button" onClick={() => setPreview("Workspaces")}>
              Meu Estúdio <span>⌄</span>
            </button>
            <button
              type="button"
              aria-label="Configurar workspace"
              onClick={() => onNavigate("settings")}
            >
              <Icon name="settings" />
            </button>
          </div>
          <button type="button" className="home-all-projects" onClick={() => onNavigate("control")}>
            Ver todos os projetos <Icon name="arrow" />
          </button>
        </div>
      </aside>
      <div className="home-main">
        <header className="home-topbar">
          <button
            type="button"
            className="home-brand"
            aria-label="La fabrique, painel inicial"
            onClick={goHome}
          >
            <svg viewBox="0 0 38 38" fill="none" aria-hidden="true">
              <circle
                cx="19"
                cy="19"
                r="17"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeDasharray="4 3"
              />
              <circle cx="19" cy="19" r="10" stroke="currentColor" strokeWidth="2" />
              <circle cx="19" cy="19" r="4" fill="currentColor" />
            </svg>
            <span>
              <strong>La fabrique</strong>
              <small>Centro de Comando</small>
            </span>
          </button>
          <fieldset className="layout-switch">
            <legend className="home-sr-only">Visual do sistema</legend>
            {(
              [
                ["novo", "Novo"],
                ["antigo", "Antigo"],
              ] as const
            ).map(([value, label]) => (
              <button
                type="button"
                key={value}
                aria-pressed={layout === value}
                onClick={() => setLayout(value)}
              >
                {label}
              </button>
            ))}
          </fieldset>
          <span className="home-operational">
            <i />
            OPERACIONAL <small>· demonstração</small>
          </span>
          <div className="home-clock">
            <span>
              {now.toLocaleDateString("pt-BR", {
                weekday: "short",
                day: "numeric",
                month: "short",
                year: "numeric",
              })}
            </span>
            <time dateTime={now.toISOString()}>{now.toLocaleTimeString("pt-BR")}</time>
          </div>

          <label className="home-search">
            <Icon name="search" />
            <input
              ref={search}
              aria-label="Buscar atalhos no sistema"
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="Buscar algo no sistema..."
            />
            <kbd>⌘ K</kbd>
          </label>
          <span className="home-demo-tag">Demonstração</span>
          <div className="home-top-actions">
            <button
              type="button"
              aria-label="Ver notificações importantes"
              onClick={() => setPreview("Notificações importantes")}
            >
              <Icon name="bell" />
              <span className="home-notification-count">4</span>
            </button>
            <button type="button" aria-label="Ajuda" onClick={() => setPreview("Ajuda")}>
              <Icon name="help" />
            </button>
            <button
              type="button"
              className="home-profile"
              aria-label="Abrir minha conta"
              onClick={() => onNavigate("account")}
            >
              <Avatar />
              <span>
                <strong>Minha conta</strong>
                <small>Perfil e equipe</small>
              </span>
              <span>⌄</span>
            </button>
          </div>
        </header>
        {children && query && (
          <section className="home-search-results" aria-label="Resultados da busca de atalhos">
            {filtered.map((item) => (
              <button
                type="button"
                key={item.name}
                onClick={() => open(item.name, item.destination)}
              >
                {item.name}
              </button>
            ))}
            {filtered.length === 0 && <p role="status">Nenhum atalho encontrado.</p>}
          </section>
        )}
        {children ? (
          <section id="home-content" className="workspace-content" tabIndex={-1}>
            {children}
          </section>
        ) : layout === "novo" ? (
          <SobrioHome
            agents={agents}
            notifications={notifications}
            greeting={greeting}
            dateLabel={now.toLocaleDateString("pt-BR", {
              weekday: "short",
              day: "numeric",
              month: "short",
            })}
            onNewProject={() => onNavigate("control")}
            onNewTask={() => onNavigate("tasks")}
            onPreview={setPreview}
          />
        ) : (
          <main id="home-content" className="home-columns" tabIndex={-1}>
            <h1 className="home-sr-only">Central de controle La fabrique</h1>
            <p className="home-demo-note">
              Prévia demonstrativa · Indicadores, notificações, mensagens, projetos e status são
              dados simulados.
            </p>
            <div className="home-command-row">
              <div className="home-core-status">
                <section className="home-card">
                  <h2>Status do núcleo</h2>
                  {agents.map((agent) => (
                    <div className={`home-status-row tone-${agent.color}`} key={agent.name}>
                      <i className="home-dot" />
                      <strong>{agent.name}</strong>
                      <small>{agent.status}</small>
                    </div>
                  ))}
                </section>
                <section className="home-card home-diagnostics">
                  <h2>Diagnóstico demonstrativo</h2>
                  <div className="home-gauges">
                    {agents.slice(1, 4).map((agent) => (
                      <div className={`home-gauge tone-${agent.color}`} key={agent.name}>
                        <svg viewBox="0 0 66 66" aria-hidden="true">
                          <circle cx="33" cy="33" r="27" />
                          <circle
                            cx="33"
                            cy="33"
                            r="27"
                            strokeDasharray={`${agent.progress * 1.696} 169.6`}
                          />
                        </svg>
                        <strong>{agent.progress}%</strong>
                        <small>{agent.name}</small>
                      </div>
                    ))}
                  </div>
                  <p>Atividade simulada · não representa cota ou consumo.</p>
                </section>
              </div>
              <section className="home-office" aria-label="Escritório virtual da equipe">
                <h2>Núcleo de orquestração</h2>
                <span className="home-core-label">Demonstração</span>
                <OrchestrationCore agents={agents} />
                <div className="home-greeting">
                  <h2>{greeting}, Administrador</h2>
                  <p>Núcleos · Workspace de demonstração</p>
                  <button
                    type="button"
                    className="home-command"
                    onClick={() => onNavigate("tasks")}
                  >
                    <span aria-hidden="true">&gt;</span>
                    <span>Descreva uma nova tarefa…</span>
                    <strong>ENVIAR</strong>
                  </button>
                </div>
              </section>
              <section className="home-card home-notifications">
                {panelHeading("Notificações importantes", () =>
                  setPreview("Notificações importantes"),
                )}
                {notifications.map((item) => (
                  <button
                    type="button"
                    className="home-notice"
                    key={item.title}
                    onClick={() => setPreview(item.title)}
                  >
                    <span className={`home-notice-icon tone-${item.color}`}>
                      <Icon name={item.icon} />
                    </span>
                    <span>
                      <strong>{item.title}</strong>
                      {item.description && <small>{item.description}</small>}
                      <small>{item.time}</small>
                    </span>
                  </button>
                ))}
              </section>
            </div>
            <div className="home-bottom-row">
              <section className="home-card home-agents">
                {panelHeading("Status dos Agentes", () => onNavigate("settings"))}
                {agents.map((agent) => (
                  <button
                    type="button"
                    className="home-agent"
                    key={agent.name}
                    onClick={() => onNavigate("settings")}
                  >
                    <Avatar robot={agent.robot} />
                    <span>
                      <strong>
                        {agent.name}
                        <span>{agent.role}</span>
                      </strong>
                      <small>
                        <i className={`home-dot tone-${agent.color}`} />
                        {agent.status}
                      </small>
                    </span>
                    <span className="home-agent-progress">
                      <meter
                        aria-label={`Atividade simulada de ${agent.name}`}
                        min={0}
                        max={100}
                        value={agent.progress}
                      ></meter>
                      <small>{agent.progress}%</small>
                    </span>
                  </button>
                ))}
              </section>
              <section className="home-card home-projects">
                {panelHeading("Projetos recentes", () => onNavigate("control"))}
                {recentProjects.map((project) => (
                  <button
                    type="button"
                    className="home-project"
                    key={project.name}
                    onClick={() => onNavigate("control")}
                  >
                    <span className={`home-project-icon tone-${project.color}`}>
                      <Icon name={project.icon} />
                    </span>
                    <span>
                      <strong>{project.name}</strong>
                      <small>{project.description}</small>
                    </span>
                    <small className="home-project-status">
                      <i
                        className={`home-dot tone-${project.status === "Ativo" ? "green" : project.status === "Pausado" ? "orange" : "blue"}`}
                      />
                      {project.status}
                    </small>
                  </button>
                ))}
              </section>
              <section className="home-metrics" aria-label="Indicadores de demonstração">
                {[
                  {
                    title: "Projetos Ativos",
                    value: "5",
                    detail: "de 5 disponíveis",
                    icon: "folder" as const,
                    color: "blue",
                  },
                  {
                    title: "Leads (30 dias)",
                    value: "127",
                    detail: "↑ 12%",
                    icon: "users" as const,
                    color: "purple",
                  },
                  {
                    title: "Vendas (30 dias)",
                    value: "R$ 12.450",
                    detail: "↑ 18%",
                    icon: "money" as const,
                    color: "green",
                  },
                  {
                    title: "Conteúdos Gerados",
                    value: "48",
                    detail: "↑ 33%",
                    icon: "content" as const,
                    color: "purple",
                  },
                ].map((metric, index) => (
                  <article className="home-card home-metric" key={metric.title}>
                    <span className={`home-icon-tile tone-${metric.color}`}>
                      <Icon name={metric.icon} />
                    </span>
                    <div>
                      <span>{metric.title}</span>
                      <strong>{metric.value}</strong>
                      <small className={index > 0 ? "home-growth" : ""}>{metric.detail}</small>
                    </div>
                    {index === 0 ? (
                      <div className="home-metric-line" />
                    ) : (
                      <div className={`home-bars tone-${metric.color}`} aria-hidden="true">
                        {[35, 63, 45, 72, 88, 100].map((height) => (
                          <i key={height} style={{ height: `${height}%` }} />
                        ))}
                      </div>
                    )}
                  </article>
                ))}
              </section>
            </div>
            <div className="home-support-row">
              <section className="home-card home-quick">
                <h2>Acesso rápido</h2>
                <div className="home-shortcuts">
                  {filtered.map((item) => (
                    <button
                      type="button"
                      key={item.name}
                      onClick={() => open(item.name, item.destination)}
                    >
                      <span className={`home-shortcut-icon tone-${item.color}`}>
                        <Icon name={item.icon} />
                      </span>
                      <span>
                        <strong>{item.name}</strong>
                        <small>{item.description}</small>
                      </span>
                    </button>
                  ))}
                </div>
                {filtered.length === 0 && (
                  <p className="home-empty" role="status">
                    Nenhum atalho encontrado. Tente “projetos” ou “configurações”.
                  </p>
                )}
              </section>
              <section className="home-card home-important-message">
                <span className="home-shortcut-icon tone-blue">
                  <Icon name="message" />
                </span>
                <div>
                  <h2>Mensagens importantes</h2>
                  <p>
                    Seu escritório está pronto para ganhar vida. Revise as contas e a equipe para
                    começar.
                  </p>
                </div>
                <button type="button" onClick={() => onNavigate("settings")}>
                  Configurar equipe <Icon name="arrow" />
                </button>
              </section>
            </div>
          </main>
        )}
      </div>
      <aside className="home-settings-rail" aria-label="Atalhos de configurações">
        <span>CONFIG</span>
        {(
          [
            ["Geral", "settings"],
            ["Modelos", "robot"],
            ["Integrações", "cloud"],
            ["Acessos", "users"],
            ["Alertas", "bell"],
            ["Equipe", "users"],
          ] as const
        ).map(([label, icon]) => (
          <button
            type="button"
            key={label}
            aria-label={`Configurações: ${label}`}
            onClick={() => onNavigate("settings")}
          >
            <Icon name={icon} />
            <span>{label}</span>
          </button>
        ))}
      </aside>
      <dialog
        ref={dialog}
        className="home-preview"
        onClose={() => setPreview(null)}
        aria-labelledby="home-preview-title"
      >
        <div className="home-preview-heading">
          <span className="home-demo-tag">Demonstração</span>
          <form method="dialog">
            <button type="submit" aria-label="Fechar prévia">
              ×
            </button>
          </form>
        </div>
        <h2 id="home-preview-title">{preview}</h2>
        {preview === "Notificações importantes" ? (
          notifications.map((item) => (
            <p key={item.title}>
              <strong>{item.title}</strong>
              <br />
              {item.description ?? item.time}
            </p>
          ))
        ) : (
          <p>
            Esta é uma prévia da central de controle. Os dados desta área são simulados; a
            integração será disponibilizada em uma próxima etapa.
          </p>
        )}
        <form method="dialog">
          <button type="submit">Voltar ao painel</button>
        </form>
      </dialog>
    </div>
  );
}
