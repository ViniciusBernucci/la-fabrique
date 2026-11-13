package httpapi

import (
	"errors"
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
)

func (s *Server) planStore() *planning.Store { return &planning.Store{DB: s.db} }

// planFeature valida {id} e {featureId} e confere que a feature é do projeto.
func (s *Server) planFeature(w http.ResponseWriter, r *http.Request) (project, feature string, ok bool) {
	project, ok = s.projectID(w, r)
	if !ok {
		return "", "", false
	}
	feature = r.PathValue("featureId")
	var one int
	err := s.db.QueryRow(r.Context(), `SELECT 1 FROM features WHERE project_id = $1 AND id::text = $2`, project, feature).Scan(&one)
	if err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return "", "", false
	}
	return project, feature, true
}

// proposePlan recebe o plano do planejador, valida e versiona. Plano inválido ou
// conflitante volta com 422 e a lista de problemas; a versão vigente não muda.
func (s *Server) proposePlan(w http.ResponseWriter, r *http.Request) {
	project, feature, ok := s.planFeature(w, r)
	if !ok {
		return
	}
	var plan planning.Plan
	if !decodeJSON(w, r, &plan) {
		return
	}
	out, err := s.planStore().Propose(r.Context(), project, feature, plan)
	var rej *planning.RejectedError
	if errors.As(err, &rej) {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": ErrorDetail{Code: "plan_rejected", Message: "O plano foi rejeitado; a versão anterior foi preservada.",
				Action: "Corrija os problemas listados e envie o plano de novo.", CorrelationID: correlationID(r.Context())},
			"problems": rej.Problems, "version": out.Stored.Version,
		})
		return
	}
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) getPlan(w http.ResponseWriter, r *http.Request) {
	project, feature, ok := s.planFeature(w, r)
	if !ok {
		return
	}
	st := s.planStore()
	cur, err := st.Current(r.Context(), project, feature)
	var current *planning.Stored
	switch {
	case err == nil:
		current = &cur
	case !errors.Is(err, planning.ErrNoPlan):
		s.writeDomainError(w, r, err)
		return
	}
	tasks, err := st.Tasks(r.Context(), project, feature)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	versions, err := st.Versions(r.Context(), project, feature)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": current, "tasks": tasks, "versions": versions})
}
