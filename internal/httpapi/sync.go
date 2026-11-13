package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/syncsource"
)

func (s *Server) syncer() *syncsource.Syncer { return &syncsource.Syncer{DB: s.db} }

// sourceTracker monta o adaptador da origem do projeto com a conexão do cofre.
// O segredo não sai do adaptador.
func (s *Server) sourceTracker(r *http.Request, p projects.Project) (integrations.Tracker, error) {
	if p.Source.Kind == "" || p.Source.ConnectionID == nil || p.Source.Container == "" {
		return nil, projects.ErrSourceIncomplete
	}
	secret, conn, err := s.vault.Resolve(r.Context(), *p.Source.ConnectionID)
	if err != nil {
		return nil, err
	}
	if conn.Kind != p.Source.Kind {
		return nil, projects.ErrSourceIncomplete
	}
	return s.trackers(integrations.Config{Kind: conn.Kind, Target: conn.Target, Secret: secret})
}

func syncSource(p projects.Project) syncsource.Source {
	return syncsource.Source{Kind: p.Source.Kind, Container: p.Source.Container, Marker: p.Source.EligibilityMarker,
		EligibleStatuses: p.Source.EligibleStatuses, PriorityMap: p.Source.PriorityMap}
}

// runSync é a sincronização MANUAL: lê a origem uma vez, agora, e registra as diferenças.
// Nada a dispara sozinho.
func (s *Server) runSync(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	tr, err := s.sourceTracker(r, p)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()
	res, err := s.syncer().Run(ctx, id, tr, syncSource(p))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) getSync(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var last *syncsource.Result
	res, err := s.syncer().Last(r.Context(), id)
	switch {
	case err == nil:
		last = &res
	case !errors.Is(err, persistence.ErrNotFound):
		s.writeDomainError(w, r, err)
		return
	}
	items, err := s.syncer().Items(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"last": last, "items": items})
}

func (s *Server) setSyncPriority(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var body struct {
		Priority *int `json:"priority"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	it, err := s.syncer().SetLocalPriority(r.Context(), id, r.PathValue("remoteId"), body.Priority)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

func (s *Server) resolveSyncConflict(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var body struct {
		Choice string `json:"choice"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	it, err := s.syncer().ResolveConflict(r.Context(), id, r.PathValue("remoteId"), body.Choice)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}
