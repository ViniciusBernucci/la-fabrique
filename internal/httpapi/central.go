package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/central"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
)

var refusalOptions = []string{"refuse", "deny", "reject"}

func (s *Server) listAuthorizations(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", "all", central.StatusActive, central.StatusExpired, central.StatusExhausted, central.StatusRevoked:
	default:
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Situação desconhecida.", "Use active, expired, exhausted, revoked ou all.")
		return
	}
	items, err := central.ListAuthorizations(r.Context(), s.db, time.Now().UTC(), status)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// revokeAuthorization revoga (ação sensível: exige reautenticação). A revogação vale no commit; se
// havia uso em andamento e a parada segura falhar, a resposta diz isso (a revogação continua valendo).
func (s *Server) revokeAuthorization(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	project, err := central.AuthorizationProject(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	store := &authorizations.Store{DB: s.db}
	rev, err := store.Revoke(r.Context(), project, id, s.adminID(r), req.Reason)
	haltFailed := errors.Is(err, authorizations.ErrHaltFailed)
	if err != nil && !haltFailed {
		s.writeDomainError(w, r, err)
		return
	}
	if !rev.AlreadyRevoked {
		s.auth.Audit(r.Context(), s.adminID(r), "authorization_revoked", s.meta(r), map[string]any{"authorization_id": id, "in_flight": rev.InFlight, "halted": rev.Halted, "halt_failed": haltFailed})
	}
	views, _ := central.ListAuthorizations(r.Context(), s.db, time.Now().UTC(), "all")
	var view *central.AuthView
	for i := range views {
		if views[i].ID == id {
			view = &views[i]
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"authorization": view, "in_flight": rev.InFlight, "halted": rev.Halted, "halt_failed": haltFailed, "already_revoked": rev.AlreadyRevoked})
}

// refuseDecision recusa o pedido escolhendo a alternativa de recusa (refuse, deny ou reject) que o
// pedido declara; sem ela, a recusa não existe para esse pedido.
func (s *Server) refuseDecision(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Note string `json:"note"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	d, err := decisions.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	option := ""
	for _, ref := range refusalOptions {
		for _, o := range d.Options {
			if o.ID == ref && option == "" {
				option = ref
			}
		}
	}
	if option == "" {
		writeError(w, r, http.StatusUnprocessableEntity, "no_refusal_option", "Este pedido não tem uma alternativa de recusa.", "Escolha uma das alternativas listadas.")
		return
	}
	s.applyDecision(w, r, d, option, req.Note)
}

// executionSequence devolve a sequência do incidente e da retomada da execução.
func (s *Server) executionSequence(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	project, err := central.ExecutionProject(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	items, err := central.Sequence(r.Context(), s.db, project, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"execution_id": id, "project_id": project, "items": items})
}
