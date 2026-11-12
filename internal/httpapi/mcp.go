package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/mcp"
)

func (s *Server) mcpService() *mcp.Service {
	return &mcp.Service{DB: s.db, Vault: s.vault, Redactor: s.redactor, AllowPrivate: s.mcpAllowPrivate}
}

func (s *Server) listMCPServers(w http.ResponseWriter, r *http.Request) {
	items, err := s.mcpService().Servers(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createMCPServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string  `json:"name"`
		URL          string  `json:"url"`
		ConnectionID *string `json:"connection_id"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	sv, err := s.mcpService().RegisterServer(r.Context(), req.Name, req.URL, req.ConnectionID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, sv)
}

// discoverMCPServer lista as ferramentas do servidor. Nada ganha acesso por ser descoberto.
func (s *Server) discoverMCPServer(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	tools, err := s.mcpService().Discover(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tools})
}

func (s *Server) listMCPTools(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	if _, err := s.mcpService().Server(r.Context(), id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	tools, err := s.mcpService().Tools(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": tools})
}

// agentMCP mostra o que o agente tem concedido e o que ele realmente pode usar agora.
func (s *Server) agentMCP(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	if _, err := agents.Get(r.Context(), s.db, id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	svc := s.mcpService()
	grants, err := svc.Grants(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	usable, err := svc.AgentTools(r.Context(), id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"grants": grants, "tools": usable})
}

type grantBody struct {
	ServerID   string `json:"server_id"`
	Tool       string `json:"tool"`
	SchemaHash string `json:"schema_hash"`
}

func (s *Server) grantMCPTool(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req grantBody
	if !decodeJSON(w, r, &req) {
		return
	}
	if _, err := agents.Get(r.Context(), s.db, id); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if err := s.mcpService().Grant(r.Context(), id, req.ServerID, req.Tool, req.SchemaHash); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	sess, _ := sessionFrom(r.Context())
	s.auth.Audit(r.Context(), sess.AdminID, "mcp_tool_granted", s.meta(r), map[string]any{"agent_id": id, "server_id": req.ServerID, "tool": req.Tool})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) revokeMCPTool(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	var req grantBody
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.mcpService().Revoke(r.Context(), id, req.ServerID, req.Tool); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	sess, _ := sessionFrom(r.Context())
	s.auth.Audit(r.Context(), sess.AdminID, "mcp_tool_revoked", s.meta(r), map[string]any{"agent_id": id, "server_id": req.ServerID, "tool": req.Tool})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listMCPCalls(w http.ResponseWriter, r *http.Request) {
	items, err := s.mcpService().Calls(r.Context(), 50)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
