package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/prcycle"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

func (s *Server) prService() *prcycle.Service {
	co := &sentinel.Coordinator{DB: s.db}
	return &prcycle.Service{DB: s.db, Ledger: &effects.Ledger{DB: s.db}, Sentinel: co}
}

// listPullRequests devolve as PRs gerenciadas do projeto com as rodadas e as pendências.
func (s *Server) listPullRequests(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	svc := s.prService()
	if _, err := svc.SyncDeliveries(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	views, err := svc.List(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if views == nil {
		views = []prcycle.WatchView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"pull_requests": views})
}

// syncPullRequests é a consulta manual: a mesma passagem da rotina, sem esperar a agenda.
func (s *Server) syncPullRequests(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	res := &prcycle.Resolver{DB: s.db, Vault: s.vault, Syncer: s.issueSyncer(), GitHubBase: s.githubBase, AllowPrivate: s.githubPrivate}
	t, err := res.Target(r.Context(), id)
	if errors.Is(err, prcycle.ErrNoTarget) {
		writeError(w, r, http.StatusUnprocessableEntity, "pr_target_missing", "O projeto precisa estar ativo, com repositório GitHub, origem e mapa de status configurados.", "Complete o cadastro do projeto.")
		return
	}
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()
	rep, err := s.prService().PollAll(ctx, id, t)
	var rl *gitops.ErrRateLimited
	if errors.As(err, &rl) {
		writeError(w, r, http.StatusTooManyRequests, "github_rate_limited", "O GitHub limitou as requisições.", "Tente de novo em "+rl.RetryAfter.Round(time.Second).String()+".")
		return
	}
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rep)
}
