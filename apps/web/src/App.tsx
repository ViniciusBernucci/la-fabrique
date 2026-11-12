import { Route, Routes } from "react-router-dom";
import { AppShell } from "./components/AppShell";
import { ErrorState, LoadingState } from "./components/States";
import { useSession } from "./lib/session";
import { AgentsPage } from "./pages/AgentsPage";
import { ConnectionsPage } from "./pages/ConnectionsPage";
import { DecisionsPage } from "./pages/DecisionsPage";
import { HomePage } from "./pages/HomePage";
import { LibraryPage } from "./pages/LibraryPage";
import { LoginPage } from "./pages/LoginPage";
import { NewProjectPage } from "./pages/NewProjectPage";
import { NotFoundPage } from "./pages/NotFoundPage";
import { ProjectDetailPage } from "./pages/ProjectDetailPage";
import { ProjectSetupPage } from "./pages/ProjectSetupPage";
import { ProjectsPage } from "./pages/ProjectsPage";
import { SecurityPage } from "./pages/SecurityPage";
import { SkillsPage } from "./pages/SkillsPage";
import { WorkflowEditorPage } from "./pages/WorkflowEditorPage";
import { WorkflowsPage } from "./pages/WorkflowsPage";

function Gate({ children }: { children: React.ReactNode }) {
  return (
    <div className="gate">
      <div className="panel">{children}</div>
    </div>
  );
}

/** Sem sessão válida só o login é renderizado (o servidor também recusa qualquer rota de dados). */
export function App() {
  const { state, retry } = useSession();

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
    return <LoginPage />;
  }

  return (
    <Routes>
      <Route element={<AppShell />}>
        <Route index element={<HomePage />} />
        <Route path="projects" element={<ProjectsPage />} />
        <Route path="projects/new" element={<NewProjectPage />} />
        <Route path="projects/:id" element={<ProjectDetailPage />} />
        <Route path="projects/:id/setup" element={<ProjectSetupPage />} />
        <Route path="library" element={<LibraryPage />} />
        <Route path="agents" element={<AgentsPage />} />
        <Route path="skills" element={<SkillsPage />} />
        <Route path="workflows" element={<WorkflowsPage />} />
        <Route path="workflows/:id" element={<WorkflowEditorPage />} />
        <Route path="decisions" element={<DecisionsPage />} />
        <Route path="connections" element={<ConnectionsPage />} />
        <Route path="security" element={<SecurityPage />} />
        <Route path="*" element={<NotFoundPage />} />
      </Route>
    </Routes>
  );
}
