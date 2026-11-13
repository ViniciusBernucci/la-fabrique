package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
)

func (s *Server) toolRegistry() *tools.Registry {
	return tools.NewRegistry(func(ctx_ context.Context, name string) (tools.Tool, error) { return tools.Get(ctx_, s.db, name) })
}

func (s *Server) listTools(w http.ResponseWriter, r *http.Request) {
	items := tools.Natives()
	custom, err := tools.List(r.Context(), s.db)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": append(items, custom...)})
}

func (s *Server) createTool(w http.ResponseWriter, r *http.Request) {
	var t tools.Tool
	if !decodeJSON(w, r, &t) {
		return
	}
	t.Version = 0
	created, err := tools.Create(r.Context(), s.db, s.policy, t)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// checkTool valida uma chamada SEM executá-la: ferramenta, concessão, escopo e
// argumentos. É o que o executor faz antes de qualquer efeito.
func (s *Server) checkTool(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req struct {
		Args  map[string]any `json:"args"`
		Grant struct {
			Tools       []string `json:"tools"`
			Permissions []string `json:"permissions"`
		} `json:"grant"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	_, clean, err := s.toolRegistry().Check(r.Context(), strings.TrimSpace(name), req.Args, tools.Grant{Tools: req.Grant.Tools, Permissions: req.Grant.Permissions})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "args": clean})
}
