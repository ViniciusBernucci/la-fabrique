package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type workflowView struct {
	workflows.Workflow
	Issues []workflows.Issue `json:"issues"`
}

func viewOf(w workflows.Workflow) workflowView {
	issues := workflows.Validate(w.Graph)
	if issues == nil {
		issues = []workflows.Issue{}
	}
	return workflowView{Workflow: w, Issues: issues}
}

func (s *Server) workflowNodeTypes(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": workflows.Catalog()})
}

func (s *Server) listWorkflows(w http.ResponseWriter, r *http.Request) {
	items, err := workflows.List(r.Context(), s.db)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createWorkflow(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Description string `json:"description"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	wf, err := workflows.Create(r.Context(), s.db, req.Name, req.Kind, req.Description)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, viewOf(wf))
}

func (s *Server) getWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r) // mesmo formato de UUID
	if !ok {
		return
	}
	wf, err := workflows.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, viewOf(wf))
}

// saveWorkflow grava o rascunho. Um grafo inválido é aceito: os problemas voltam
// em `issues`, com nó e campo, para o editor exibir.
func (s *Server) saveWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		ExpectedVersion int             `json:"expected_version"`
		Name            string          `json:"name"`
		Description     string          `json:"description"`
		Graph           workflows.Graph `json:"graph"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ExpectedVersion < 1 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "expected_version é obrigatório.", "Envie a versão do workflow que você carregou.")
		return
	}
	wf, err := workflows.Save(r.Context(), s.db, id, req.ExpectedVersion, req.Name, req.Description, req.Graph)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, viewOf(wf))
}

// validateWorkflowGraph valida um grafo sem persistir (o editor chama ao editar).
func (s *Server) validateWorkflowGraph(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Graph workflows.Graph `json:"graph"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	issues := workflows.Validate(req.Graph)
	if issues == nil {
		issues = []workflows.Issue{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues})
}
