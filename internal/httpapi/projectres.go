package httpapi

import (
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/projectres"
)

// discoverProfile inspeciona manifestos ENVIADOS pelo painel (nada é lido do
// disco do servidor, nada é executado) e grava as sugestões com a origem.
func (s *Server) discoverProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var body struct {
		Files map[string]string `json:"files"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	total := 0
	for _, c := range body.Files {
		total += len(c)
	}
	if len(body.Files) == 0 || len(body.Files) > 32 || total > 2<<20 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Envie de 1 a 32 manifestos (até 2 MiB no total).", "Envie package.json, go.mod, Makefile, Compose ou .env.example do repositório.")
		return
	}
	d := projectres.Inspect(projectres.MapFiles(body.Files))
	if err := projectres.SaveDiscovery(r.Context(), s.db, id, d, time.Now().UTC()); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) getProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	p, err := projectres.GetProfile(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) putProfile(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var body struct {
		ExpectedVersion int `json:"expected_version"`
		projectres.ProfileInput
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	p, err := projectres.SetProfile(r.Context(), s.db, id, body.ExpectedVersion, body.ProfileInput, s.redactor, time.Now().UTC())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	res, err := projectres.ListResources(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resources": res})
}

func (s *Server) putResources(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var body struct {
		Resources []projectres.ResourceInput `json:"resources"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	res, err := projectres.SetResources(r.Context(), s.db, id, body.Resources)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"resources": res})
}

func (s *Server) readiness(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	rd, err := projectres.Ready(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rd)
}
