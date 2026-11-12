package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/models"
)

func (s *Server) listModels(w http.ResponseWriter, r *http.Request) {
	items, err := models.List(r.Context(), s.db)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createModel(w http.ResponseWriter, r *http.Request) {
	var in models.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	m, err := models.Register(r.Context(), s.db, in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (s *Server) setModelState(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		State           string `json:"state"`
		ExpectedVersion int    `json:"expected_version"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	m, err := models.SetState(r.Context(), s.db, id, req.State, req.ExpectedVersion)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// checkModel diz, sem executar nada, se o modelo serve para a configuração.
func (s *Server) checkModel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider     string `json:"provider"`
		ConnectionID string `json:"connection_id"`
		Model        string `json:"model"`
		NeedsTools   bool   `json:"needs_tools"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	v, err := models.Check(r.Context(), s.db, models.Requirement{Provider: req.Provider, ConnectionID: req.ConnectionID, Model: req.Model, NeedsTools: req.NeedsTools})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
