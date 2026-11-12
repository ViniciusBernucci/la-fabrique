import { useEffect, useState } from "react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { api } from "../lib/api";
import { useCurrentSession } from "../lib/session";
import { Icon, type IconName } from "./Icon";

const COLLAPSE_KEY = "le-fabrique.menu-collapsed";

/** Itens do menu. Novas áreas entram aqui conforme os épicos forem entregues. */
export const navigation: { name: string; to: string; icon: IconName; end?: boolean }[] = [
  { name: "Painel", to: "/", icon: "home", end: true },
  { name: "Projetos", to: "/projects", icon: "folder" },
  { name: "Biblioteca", to: "/library", icon: "library" },
  { name: "Decisões", to: "/decisions", icon: "decisions" },
];

function readCollapsed(): boolean {
  try {
    return globalThis.localStorage?.getItem(COLLAPSE_KEY) !== "false";
  } catch {
    return true;
  }
}

type Health = "checking" | "up" | "down";

function useApiHealth(): Health {
  const [health, setHealth] = useState<Health>("checking");
  useEffect(() => {
    const controller = new AbortController();
    const check = () =>
      api
        .ready(controller.signal)
        .then(() => setHealth("up"))
        .catch(() => !controller.signal.aborted && setHealth("down"));
    check();
    const timer = window.setInterval(check, 30_000);
    return () => {
      controller.abort();
      window.clearInterval(timer);
    };
  }, []);
  return health;
}

function Clock() {
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000);
    return () => window.clearInterval(timer);
  }, []);
  return (
    <div className="clock">
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
  );
}

const healthLabel: Record<Health, string> = {
  checking: "Verificando",
  up: "Operacional",
  down: "Indisponível",
};

export function AppShell() {
  const session = useCurrentSession();
  const health = useApiHealth();
  const navigate = useNavigate();
  const location = useLocation();
  const [collapsed, setCollapsed] = useState(readCollapsed);

  // Em telas pequenas o menu abre recolhido a cada navegação.
  // biome-ignore lint/correctness/useExhaustiveDependencies: location.pathname é o gatilho.
  useEffect(() => {
    if (window.matchMedia?.("(max-width: 760px)").matches) setCollapsed(true);
  }, [location.pathname]);

  function toggle() {
    setCollapsed((current) => {
      try {
        globalThis.localStorage?.setItem(COLLAPSE_KEY, String(!current));
      } catch {
        // preferência apenas visual
      }
      return !current;
    });
  }

  return (
    <div className={`app-shell${collapsed ? " is-menu-collapsed" : ""}`}>
      <a className="skip-link" href="#content">
        Ir para o conteúdo
      </a>
      <header className="topbar">
        <button
          type="button"
          className="brand"
          aria-label="La fabrique, painel inicial"
          onClick={() => navigate("/")}
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
        <span className={`status-pill is-${health === "up" ? "up" : health}`} role="status">
          <i />
          {healthLabel[health]}
        </span>
        <Clock />
        <div className="profile">
          <span className="avatar" aria-hidden="true">
            {session.username.slice(0, 1)}
          </span>
          <span>
            <strong>{session.username}</strong>
            <small>Administrador</small>
          </span>
        </div>
      </header>
      <aside id="sidebar" className="sidebar" aria-label="Menu principal">
        <button
          type="button"
          className="menu-toggle"
          aria-label={collapsed ? "Expandir menu" : "Recolher menu"}
          title={collapsed ? "Expandir menu" : "Recolher menu"}
          aria-expanded={!collapsed}
          aria-controls="sidebar"
          onClick={toggle}
        >
          <Icon name="arrow" />
        </button>
        <nav>
          {navigation.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              end={item.end}
              aria-label={item.name}
              title={item.name}
            >
              <Icon name={item.icon} />
              <span>{item.name}</span>
            </NavLink>
          ))}
        </nav>
      </aside>
      <main id="content" className="content" tabIndex={-1}>
        <Outlet />
      </main>
    </div>
  );
}
