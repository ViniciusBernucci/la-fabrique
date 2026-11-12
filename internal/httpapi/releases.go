package httpapi

import (
	"errors"
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
)

func (s *Server) publisher() *releases.Publisher {
	return &releases.Publisher{DB: s.db, Redactor: s.redactor}
}

// writeProblems responde com o envelope de erro padrão e a lista de problemas.
func writeProblems(w http.ResponseWriter, r *http.Request, status int, code, msg, action string, problems []releases.Problem) {
	writeJSON(w, status, map[string]any{
		"error":    ErrorDetail{Code: code, Message: msg, Action: action, CorrelationID: correlationID(r.Context())},
		"problems": problems,
	})
}

func (s *Server) publishWorkflow(w http.ResponseWriter, r *http.Request) {
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
	sess, _ := sessionFrom(r.Context())
	v, err := s.publisher().Publish(r.Context(), id, req.ExpectedVersion, sess.AdminID)
	var pe *releases.PublishError
	if errors.As(err, &pe) {
		writeProblems(w, r, http.StatusUnprocessableEntity, "publish_blocked", pe.Error(), "Corrija os problemas listados e publique de novo.", pe.Problems)
		return
	}
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	status := http.StatusCreated
	if v.Unchanged {
		status = http.StatusOK
	}
	writeJSON(w, status, v)
}

func (s *Server) unpublishWorkflow(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	if err := s.publisher().Unpublish(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listWorkflowVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	items, err := releases.List(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	v, err := releases.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) revalidateWorkflowVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	v, err := releases.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	p := releases.Revalidate(r.Context(), s.db, v)
	if p == nil {
		p = []releases.Problem{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": len(p) == 0, "problems": p})
}
