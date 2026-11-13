package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/selection"
)

func (s *Server) selectionStore() *selection.Store { return &selection.Store{DB: s.db} }

func (s *Server) getSelection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	cfg, err := s.selectionStore().Get(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

func (s *Server) setSelection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var cfg selection.Config
	if !decodeJSON(w, r, &cfg) {
		return
	}
	out, err := s.selectionStore().Set(r.Context(), id, cfg)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) simulateSelection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Attributes selection.Attrs `json:"attributes"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	sel, err := s.selectionStore().Simulate(r.Context(), id, req.Attributes)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, sel)
}

func (s *Server) selectionTraces(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	items, err := s.selectionStore().Traces(r.Context(), id, 50)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
