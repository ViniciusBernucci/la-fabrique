package httpapi

import (
	"errors"
	"net/http"
	"time"

	"context"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

func (s *Server) issueSyncer() *issuesync.Syncer {
	return &issuesync.Syncer{DB: s.db, Sentinel: &sentinel.Coordinator{DB: s.db}}
}

func (s *Server) flowStore() *flow.Store { return &flow.Store{DB: s.db} }

// getStatusMap devolve o mapa de status do projeto e os códigos canônicos que a fábrica projeta.
func (s *Server) getStatusMap(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	m, found, err := s.flowStore().GetMap(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	var out any
	if found {
		out = m
	}
	writeJSON(w, http.StatusOK, map[string]any{"map": out, "codes": flow.ProjectedCodes()})
}

// putStatusMap valida e grava o mapa. Mapa inválido volta com o campo ou a transição que impede a projeção.
func (s *Server) putStatusMap(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var body struct {
		Version int               `json:"version"`
		Source  string            `json:"source"`
		Remote  map[string]string `json:"remote"`
		Moves   map[string]string `json:"moves"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	if err := s.flowStore().EnsureCatalog(r.Context()); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	m, err := s.flowStore().SaveMap(r.Context(), id, body.Version, flow.ProjectMap{Source: body.Source, Remote: body.Remote, Moves: body.Moves})
	if err != nil {
		var me *flow.MapError
		if asMapError(err, &me) {
			writeError(w, r, http.StatusUnprocessableEntity, "invalid_status_map", me.Error(), "Corrija os campos apontados e salve de novo.")
			return
		}
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// listIssues devolve as Issues registradas e o histórico de sincronização (página Issues).
func (s *Server) listIssues(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	sy := s.issueSyncer()
	issues, err := sy.List(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	runs, err := sy.Runs(r.Context(), id, 20)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if issues == nil {
		issues = []issuesync.Issue{}
	}
	if runs == nil {
		runs = []issuesync.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"issues": issues, "runs": runs})
}

// runIssueSync é o sync geral manual: mesma rotina do horário matinal.
func (s *Server) runIssueSync(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	m, found, err := s.flowStore().GetMap(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, http.StatusUnprocessableEntity, "status_map_missing", "O mapa de status do projeto ainda não foi configurado.", "Configure o mapa de status antes de sincronizar.")
		return
	}
	tr, err := s.sourceTracker(r, p)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	run, err := s.issueSyncer().General(ctx, id, issuesync.TriggerManual, tr, issuesync.Source{Kind: p.Source.Kind, Container: p.Source.Container, Map: m.ProjectMap})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func asMapError(err error, target **flow.MapError) bool { return errors.As(err, target) }
