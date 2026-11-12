package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
)

func (s *Server) listDecisions(w http.ResponseWriter, r *http.Request) {
	items, err := decisions.List(r.Context(), s.db, r.URL.Query().Get("state"))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getDecision(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	d, err := decisions.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// resolveDecision registra a decisão. Operação sensível exige reautenticação
// recente (a aprovação de uma operação é uma ação sensível, distinta do parecer
// de revisão de código).
func (s *Server) resolveDecision(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Option string `json:"option"`
		Note   string `json:"note"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	d, err := decisions.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	sess, _ := sessionFrom(r.Context())
	if d.Kind == decisions.KindSensitiveOperation && sess.ReauthUntil == nil {
		writeError(w, r, http.StatusForbidden, "reauth_required", "Esta ação exige confirmar sua identidade.", "Informe senha e código 2FA e repita a ação.")
		return
	}
	out, err := decisions.Resolve(r.Context(), s.db, sess.AdminID, id, req.Option, req.Note)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if !out.Replayed {
		s.auth.Audit(r.Context(), sess.AdminID, "decision_resolved", s.meta(r), map[string]any{"request_id": id, "kind": d.Kind, "option": req.Option})
	}
	w.Header().Set("X-Decision-Replayed", map[bool]string{true: "true", false: "false"}[out.Replayed])
	writeJSON(w, http.StatusOK, out.Request)
}
