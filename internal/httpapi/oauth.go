package httpapi

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/ViniciusBernucci/le-fabrique/internal/oauth"
)

// WithOAuthSetup ajusta o serviço OAuth (endpoints e relógio em testes).
func WithOAuthSetup(f func(*oauth.Service)) Option { return func(s *Server) { f(s.oauth) } }

func (s *Server) oauthProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.oauth.Providers()})
}

func (s *Server) oauthStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	kind := r.PathValue("kind")
	out, err := s.oauth.Start(r.Context(), kind, req.Name)
	switch {
	case errors.Is(err, oauth.ErrUnavailable):
		writeError(w, r, http.StatusConflict, "oauth_unavailable", "Este provedor ainda não está configurado para OAuth.", "Defina as variáveis do app OAuth no .env e reinicie a API (veja a dica na tela).")
		return
	case err != nil:
		s.writeDomainError(w, r, err)
		return
	}
	sess, _ := sessionFrom(r.Context())
	s.auth.Audit(r.Context(), sess.AdminID, "oauth_started", s.meta(r), map[string]any{"kind": kind})
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) oauthFlow(w http.ResponseWriter, r *http.Request) {
	st, err := s.oauth.Poll(r.Context(), r.PathValue("id"))
	if errors.Is(err, oauth.ErrUnknownFlow) {
		writeError(w, r, http.StatusNotFound, "not_found", "Fluxo desconhecido ou expirado.", "Inicie a conexão de novo.")
		return
	}
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

// oauthCallback recebe o retorno do provedor (navegação de topo, sem cookie de
// sessão por causa do SameSite=Strict). O `state` de uso único autoriza a troca.
func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	dest := url.Values{}
	if q.Get("error") != "" {
		dest.Set("oauth", "failed")
		dest.Set("reason", "Você recusou a autorização.")
	} else if st, err := s.oauth.Callback(r.Context(), q.Get("state"), q.Get("code")); err != nil {
		dest.Set("oauth", "failed")
		dest.Set("reason", "Retorno inválido ou expirado. Inicie a conexão de novo.")
	} else if st.Status == "done" {
		dest.Set("oauth", "ok")
	} else {
		dest.Set("oauth", "failed")
		dest.Set("reason", st.Error)
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, "/connections?"+dest.Encode(), http.StatusSeeOther)
}
