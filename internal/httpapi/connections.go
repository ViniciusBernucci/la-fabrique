package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func (s *Server) connectionID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return "", false
	}
	return id, true
}

func (s *Server) auditConnection(r *http.Request, typ string, c security.Connection) {
	sess, _ := sessionFrom(r.Context())
	s.auth.Audit(r.Context(), sess.AdminID, typ, s.meta(r), map[string]any{"connection_id": c.ID, "kind": c.Kind})
}

func (s *Server) listConnections(w http.ResponseWriter, r *http.Request) {
	items, err := s.vault.List(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.connectionID(w, r)
	if !ok {
		return
	}
	c, err := s.vault.Get(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) createConnection(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Kind   string `json:"kind"`
		Name   string `json:"name"`
		Target string `json:"target"`
		Scopes string `json:"scopes"`
		Secret string `json:"secret"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.vault.Create(r.Context(), security.NewConnection{
		Kind: req.Kind, Name: req.Name, Target: req.Target, Scopes: req.Scopes, Secret: security.NewSecret(req.Secret),
	})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.auditConnection(r, "connection_created", c)
	writeJSON(w, http.StatusCreated, c)
}

// testConnection decifra o segredo só dentro do processo, chama o destino e grava
// o resultado. Nada do segredo ou do corpo remoto volta na resposta.
func (s *Server) testConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.connectionID(w, r)
	if !ok {
		return
	}
	secret, conn, err := s.vault.Resolve(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	res := s.tester.Test(ctx, conn.Kind, conn.Target, secret)
	updated, err := s.vault.RecordTest(r.Context(), id, res.Status, res.Detail)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.auditConnection(r, "connection_tested", updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) rotateConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.connectionID(w, r)
	if !ok {
		return
	}
	var req struct {
		Secret          string `json:"secret"`
		ExpectedVersion int    `json:"expected_version"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	c, err := s.vault.Rotate(r.Context(), id, security.NewSecret(req.Secret), req.ExpectedVersion)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.auditConnection(r, "connection_rotated", c)
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) revokeConnection(w http.ResponseWriter, r *http.Request) {
	id, ok := s.connectionID(w, r)
	if !ok {
		return
	}
	c, err := s.vault.Revoke(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.auditConnection(r, "connection_revoked", c)
	writeJSON(w, http.StatusOK, c)
}
