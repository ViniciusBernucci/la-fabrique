import { Route, Routes } from "react-router-dom";
import { AppShell } from "./components/AppShell";
import { ErrorState, LoadingState, UnauthorizedState } from "./components/States";
import { useSession } from "./lib/session";
import { DecisionsPage } from "./pages/DecisionsPage";
import { HomePage } from "./pages/HomePage";
import { LibraryPage } from "./pages/LibraryPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { ProjectDetailPage } from "./pages/ProjectDetailPage";
import { ProjectsPage } from "./pages/ProjectsPage";

function Gate({ children }: { children: React.ReactNode }) {
  return (
    <div className="gate">
      <div className="panel">{children}</div>
    </div>
  );
}

/** Sem sessão válida nenhuma rota do painel é renderizada (o servidor também recusa). */
export function App() {
  const { state, retry, signInDev } = useSession();

  if (state.status === "loading") {
    return (
      <Gate>
        <LoadingState label="Verificando sessão…" />
      </Gate>
    );
  }
  if (state.status === "error") {
    return (
      <Gate>
        <ErrorState error={state.error} onRetry={retry} />
      </Gate>
    );
  }
  if (state.status === "unauthenticated") {
    return (
      <Gate>
        <UnauthorizedState
          onSignInDev={signInDev ? () => void signInDev().catch(retry) : undefined}
        />
      </Gate>
    );
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<HomePage />} />
        <Route path="projects" element={<ProjectsPage />} />
        <Route path="projects/:id" element={<ProjectDetailPage />} />
        <Route path="library" element={<LibraryPage />} />
        <Route path="decisions" element={<DecisionsPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
