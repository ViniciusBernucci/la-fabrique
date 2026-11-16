package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/projectsetup"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

func (s *Server) setupService() projectsetup.Service {
	return projectsetup.Service{DB: s.db, Root: s.ws.Root, Git: s.ws.Git}
}

func (s *Server) writeOpsError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, projectsetup.ErrNoScript):
		writeError(w, r, http.StatusNotFound, "setup_not_found", err.Error(), "Crie o arquivo "+projectsetup.ScriptPath+" no repositório e faça commit.")
	case errors.Is(err, projectsetup.ErrChanged):
		writeError(w, r, http.StatusConflict, "setup_changed", err.Error(), "Recarregue e revise o conteúdo atual antes de aprovar.")
	case errors.Is(err, projectsetup.ErrNotApproved):
		writeError(w, r, http.StatusConflict, "setup_not_approved", err.Error(), "Revise e aprove o script.")
	case errors.Is(err, projectsetup.ErrBadScript):
		writeError(w, r, http.StatusUnprocessableEntity, "setup_invalid", err.Error(), "Use um script de texto UTF-8.")
	case errors.Is(err, worktrees.ErrCleanupOff):
		writeError(w, r, http.StatusConflict, "cleanup_off", err.Error(), "Ligue a política de limpeza do projeto.")
	case errors.Is(err, worktrees.ErrPlanChanged):
		writeError(w, r, http.StatusConflict, "cleanup_plan_changed", err.Error(), "Simule de novo e confirme o novo resultado.")
	case errors.Is(err, worktrees.ErrInvalid), errors.Is(err, persistence.ErrInvalidInput):
		writeError(w, r, http.StatusUnprocessableEntity, "invalid_request", err.Error(), "Confira o repositório do projeto e os dados enviados.")
	default:
		s.writeDomainError(w, r, err)
	}
}

// setupState mostra o script de setup de uma ref e se está aprovado para o conteúdo atual.
func (s *Server) setupState(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		p, err := projects.Get(r.Context(), s.db, id)
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
		ref = p.Repository.BaseBranch
	}
	ctx, cancel := ctxTimeout(r, 30*time.Second)
	defer cancel()
	st, err := s.setupService().Inspect(ctx, id, ref)
	if err != nil {
		s.writeOpsError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// approveSetup aprova o script que o usuário revisou (vinculado ao hash enviado).
func (s *Server) approveSetup(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Ref    string `json:"ref"`
		SHA256 string `json:"sha256"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Ref == "" || len(req.SHA256) != 64 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Informe ref e sha256 do script revisado.", "Carregue o script e aprove o conteúdo mostrado.")
		return
	}
	ctx, cancel := ctxTimeout(r, 30*time.Second)
	defer cancel()
	st, err := s.setupService().Approve(ctx, id, s.adminID(r), req.Ref, req.SHA256)
	if err != nil {
		s.writeOpsError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// revokeSetup revoga uma aprovação.
func (s *Server) revokeSetup(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	aid := r.PathValue("approval")
	if !uuidPattern.MatchString(aid) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return
	}
	if err := s.setupService().Revoke(r.Context(), id, s.adminID(r), aid); err != nil {
		s.writeOpsError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// cleanupManager monta o gerenciador de worktrees do projeto e a SHA da branch base.
func (s *Server) cleanupManager(w http.ResponseWriter, r *http.Request, id string) (*worktrees.Manager, string, bool) {
	m, err := worktrees.Provider{DB: s.db, Root: s.ws.Root}.For(r.Context(), id)
	if err != nil {
		s.writeOpsError(w, r, err)
		return nil, "", false
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return nil, "", false
	}
	base, _ := m.ResolveBase(r.Context(), "", p.Repository.BaseBranch) // sem base: nada é removido
	return m, base, true
}

// cleanupPlan devolve a política e a simulação (nada é alterado).
func (s *Server) cleanupPlan(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	m, base, ok := s.cleanupManager(w, r, id)
	if !ok {
		return
	}
	ctx, cancel := ctxTimeout(r, 60*time.Second)
	defer cancel()
	plan, err := m.PlanCleanup(ctx, id, base, time.Now())
	if err != nil {
		s.writeOpsError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, plan)
}

// setCleanupPolicy liga/desliga a limpeza e define os prazos.
func (s *Server) setCleanupPolicy(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req worktrees.Policy
	if !decodeJSON(w, r, &req) {
		return
	}
	m, _, ok := s.cleanupManager(w, r, id)
	if !ok {
		return
	}
	p, err := m.SetPolicy(r.Context(), id, s.adminID(r), req)
	if err != nil {
		s.writeOpsError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// applyCleanup executa exatamente a simulação que o usuário viu (hash).
func (s *Server) applyCleanup(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		Hash string `json:"hash"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if len(req.Hash) != 64 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Informe o hash da simulação.", "Simule a limpeza e confirme o resultado.")
		return
	}
	m, base, ok := s.cleanupManager(w, r, id)
	if !ok {
		return
	}
	ctx, cancel := ctxTimeout(r, 3*time.Minute)
	defer cancel()
	res, err := m.ApplyCleanup(ctx, id, s.adminID(r), base, req.Hash, time.Now())
	if err != nil {
		s.writeOpsError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
