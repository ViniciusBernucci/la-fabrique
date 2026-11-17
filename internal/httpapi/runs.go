package httpapi

import (
	"errors"
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
)

// Execuções do workflow de um projeto (ADR-0069): quem inicia é a pessoa, quem conduz é o worker.

func (s *Server) runID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("runId")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return "", false
	}
	return id, true
}

func (s *Server) writeRunError(w http.ResponseWriter, r *http.Request, err error) {
	var fe *runner.FieldError
	if errors.As(err, &fe) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", fe.Message, "Corrija o campo \""+fe.Field+"\" e tente novamente.")
		return
	}
	var re *releases.RevalidationError
	if errors.As(err, &re) {
		writeProblems(w, r, http.StatusConflict, "revalidation_failed", re.Error(), "Corrija o que mudou desde a publicação (conta, modelo, skill ou ferramenta) e publique de novo.", re.Problems)
		return
	}
	s.writeDomainError(w, r, err)
}

func (s *Server) startRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Title      string `json:"title"`
		Task       string `json:"task"`
		Correction bool   `json:"correction"`
		PRDDone    bool   `json:"prd_done"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	run, err := runner.Start(r.Context(), s.db, runner.StartInput{
		ProjectID: id, Title: req.Title, Task: req.Task, Workspace: p.Repository.Path,
		CreatedBy: s.adminID(r), Correction: req.Correction, PRDDone: req.PRDDone,
	})
	if err != nil {
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) listRuns(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	items, err := runner.List(r.Context(), s.db, id, 30)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	run, err := runner.Get(r.Context(), s.db, id, runID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) cancelRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	run, err := runner.Cancel(r.Context(), s.db, id, runID, s.adminID(r))
	if err != nil {
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (s *Server) decideRun(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	runID, ok := s.runID(w, r)
	if !ok {
		return
	}
	var req struct {
		Node   string `json:"node"`
		Choice string `json:"choice"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	run, err := runner.Decide(r.Context(), s.db, id, runID, req.Node, req.Choice, s.adminID(r))
	if err != nil {
		if errors.Is(err, persistence.ErrInvalidInput) {
			writeError(w, r, http.StatusConflict, "invalid_decision", err.Error(), "Confira o nó e a escolha (approved ou rejected).")
			return
		}
		s.writeRunError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}
