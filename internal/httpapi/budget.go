package httpapi

import (
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
)

func (s *Server) budgetStore() *budget.Store { return &budget.Store{DB: s.db} }

func (s *Server) listBudgets(w http.ResponseWriter, r *http.Request) {
	items, err := s.budgetStore().List(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) setBudget(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope       string `json:"scope"`
		AgentID     string `json:"agent_id"`
		Period      string `json:"period"`
		LimitMicros int64  `json:"limit_micros"`
		State       string `json:"state"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	b, err := s.budgetStore().Set(r.Context(), budget.SetInput{Scope: req.Scope, AgentID: req.AgentID, Period: req.Period, LimitMicros: req.LimitMicros, State: req.State})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// usageReport resume o uso do período (padrão: mês corrente, UTC), com gasto em
// API e assinatura em seções separadas.
func (s *Server) usageReport(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 1, 0)
	parse := func(name string, dst *time.Time) bool {
		v := r.URL.Query().Get(name)
		if v == "" {
			return true
		}
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", "Data inválida em \""+name+"\" (use RFC 3339).", "Corrija o parâmetro e tente novamente.")
			return false
		}
		*dst = t.UTC()
		return true
	}
	if !parse("from", &from) || !parse("to", &to) {
		return
	}
	if !to.After(from) || to.Sub(from) > 366*24*time.Hour {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Período inválido (to depois de from, no máximo 366 dias).", "Corrija os parâmetros from e to.")
		return
	}
	rep, err := s.budgetStore().Report(r.Context(), from, to)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
