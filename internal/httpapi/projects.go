package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func (s *Server) projectID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return "", false
	}
	return id, true
}

func (s *Server) adminID(r *http.Request) string {
	sess, _ := sessionFrom(r.Context())
	return sess.AdminID
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug        string `json:"slug"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Priority    *int   `json:"priority"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := projects.CreateDraft(r.Context(), s.db, s.adminID(r), projects.NewProject{
		Slug: req.Slug, Name: req.Name, Description: req.Description, Priority: req.Priority,
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	// O cadastro entrega o workflow estrutural (rascunho editável) e o catálogo de status. Falhou aqui?
	// O projeto existe e GET /projects/{id}/workflow provisiona na primeira consulta.
	_ = s.flowStore().EnsureCatalog(r.Context())
	if _, _, perr := stdflow.Provision(r.Context(), s.db, p.ID, p.Name); perr != nil {
		s.log.Warn("workflow padrão não provisionado no cadastro", "project", p.ID)
	}
	writeJSON(w, http.StatusCreated, p)
}

// getProjectWorkflow devolve o workflow estrutural do projeto (provisiona se ainda não existir) e o
// diagnóstico por nó do que falta configurar antes de publicar.
func (s *Server) getProjectWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	wf, created, err := stdflow.Provision(r.Context(), s.db, id, p.Name)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	issues := workflows.Validate(wf.Graph)
	issues = append(issues, workflows.GuardIssues(wf.Graph)...)
	if issues == nil {
		issues = []workflows.Issue{}
	}
	_, perr := releases.Published(r.Context(), s.db, wf.ID)
	writeJSON(w, http.StatusOK, map[string]any{"workflow_id": wf.ID, "name": wf.Name, "version": wf.Version, "created": created,
		"published": perr == nil, "issues": issues})
}

type repositoryBody struct {
	Owner        string  `json:"owner"`
	Name         string  `json:"name"`
	BaseBranch   string  `json:"base_branch"`
	ConnectionID *string `json:"connection_id"`
}

type sourceBody struct {
	Kind              string         `json:"kind"`
	ConnectionID      *string        `json:"connection_id"`
	Container         string         `json:"container"`
	EligibilityMarker string         `json:"eligibility_marker"`
	EligibleStatuses  []string       `json:"eligible_statuses"`
	PriorityMap       map[string]int `json:"priority_map"`
	Access            string         `json:"access"`
}

// updateProject persiste uma etapa do assistente (ou ajuste de dados) com controle
// otimista por expected_version.
func (s *Server) updateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		ExpectedVersion int             `json:"expected_version"`
		Name            *string         `json:"name"`
		Description     *string         `json:"description"`
		Priority        *int            `json:"priority"`
		WizardStep      *string         `json:"wizard_step"`
		Repository      *repositoryBody `json:"repository"`
		Source          *sourceBody     `json:"source"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ExpectedVersion < 1 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "expected_version é obrigatório.", "Envie a versão do projeto que você carregou.")
		return
	}
	patch := projects.Patch{Name: req.Name, Description: req.Description, Priority: req.Priority, WizardStep: req.WizardStep}
	if req.Repository != nil {
		patch.Repository = &projects.Repository{
			Owner: req.Repository.Owner, Name: req.Repository.Name,
			BaseBranch: req.Repository.BaseBranch, ConnectionID: req.Repository.ConnectionID,
		}
	}
	if req.Source != nil {
		patch.Source = &projects.Source{
			Kind: req.Source.Kind, ConnectionID: req.Source.ConnectionID, Container: req.Source.Container,
			EligibilityMarker: req.Source.EligibilityMarker, EligibleStatuses: req.Source.EligibleStatuses,
			PriorityMap: req.Source.PriorityMap, Access: req.Source.Access,
		}
	}
	p, err := projects.Configure(r.Context(), s.db, s.adminID(r), id, req.ExpectedVersion, patch)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) validateProject(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	report, err := projects.Validate(r.Context(), s.db, p)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) lifecycle(op func(r *http.Request, id string, version int) (projects.Project, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := s.projectID(w, r)
		if !ok {
			return
		}
		var req struct {
			ExpectedVersion int `json:"expected_version"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		if req.ExpectedVersion < 1 {
			writeError(w, r, http.StatusBadRequest, "invalid_request", "expected_version é obrigatório.", "Envie a versão do projeto que você carregou.")
			return
		}
		p, err := op(r, id, req.ExpectedVersion)
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func (s *Server) activateProject() http.HandlerFunc {
	return s.lifecycle(func(r *http.Request, id string, v int) (projects.Project, error) {
		return projects.Activate(r.Context(), s.db, s.adminID(r), id, v)
	})
}

func (s *Server) suspendProject() http.HandlerFunc {
	return s.lifecycle(func(r *http.Request, id string, v int) (projects.Project, error) {
		return projects.Suspend(r.Context(), s.db, s.adminID(r), id, v)
	})
}

func (s *Server) resumeProject() http.HandlerFunc {
	return s.lifecycle(func(r *http.Request, id string, v int) (projects.Project, error) {
		return projects.Resume(r.Context(), s.db, s.adminID(r), id, v)
	})
}

func (s *Server) listProjectChanges(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	if _, err := projects.Get(r.Context(), s.db, id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	items, err := projects.Changes(r.Context(), s.db, id, 50)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// checkProjectSource prova o acesso à origem com a conexão do cofre, devolve o
// preview dos itens elegíveis e registra o resultado para a configuração atual.
// Nada é escrito na origem; o segredo não sai do adaptador.
func (s *Server) checkProjectSource(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if p.Source.Kind == "" || p.Source.ConnectionID == nil || p.Source.Container == "" {
		s.writeDomainError(w, r, projects.ErrSourceIncomplete)
		return
	}
	secret, conn, err := s.vault.Resolve(r.Context(), *p.Source.ConnectionID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if conn.Kind != p.Source.Kind {
		s.writeDomainError(w, r, projects.ErrSourceIncomplete)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	var res projects.SourceCheck
	tr, err := s.trackers(integrations.Config{Kind: conn.Kind, Target: conn.Target, Secret: secret})
	if err != nil {
		res = projects.SourceCheck{CheckedAt: time.Now().UTC(), Items: []projects.PreviewItem{}, Steps: []projects.SourceStep{{
			ID: "credential", Label: "Credencial aceita", Status: projects.StepFailed,
			Detail: "O destino ou o formato da credencial da conexão é inválido. Corrija a conexão em Conexões.",
		}}}
	} else {
		res = projects.CheckSource(ctx, tr, p, time.Now().UTC())
	}
	if err := projects.RecordSourceCheck(r.Context(), s.db, s.adminID(r), id, projects.Fingerprint(p, conn.SecretVersion), res); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
