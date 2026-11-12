package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	items, err := agents.List(r.Context(), s.db)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	a, err := agents.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var in agents.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	a, err := agents.Create(r.Context(), s.db, in)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req struct {
		agents.Input
		ExpectedVersion int `json:"expected_version"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.ExpectedVersion < 1 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "expected_version é obrigatório.", "Envie a versão do agente que você carregou.")
		return
	}
	a, err := agents.Update(r.Context(), s.db, id, req.ExpectedVersion, req.Input)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// workflowNodeEffective mostra a configuração efetiva (base do catálogo mais as
// sobrescritas do nó) de um nó de agente do rascunho. Sem segredos.
func (s *Server) workflowNodeEffective(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	wf, err := workflows.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	nodeID := r.PathValue("node")
	for _, n := range wf.Graph.Nodes {
		if n.ID != nodeID {
			continue
		}
		eff, err := agents.EffectiveForNode(r.Context(), s.db, n)
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, eff)
		return
	}
	writeError(w, r, http.StatusNotFound, "not_found", "Nó não encontrado no workflow.", "Verifique o identificador do nó.")
}
