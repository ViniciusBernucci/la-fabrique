package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/clilogin"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// loginView é o que a tela vê de um login de conta de assinatura (ADR-0065). O desafio (link e código)
// só existe enquanto a pessoa precisa dele; nunca há token aqui.
type loginView struct {
	ID              string               `json:"id"`
	Provider        string               `json:"provider"`
	Name            string               `json:"name"`
	State           string               `json:"state"`
	VerificationURI string               `json:"verification_uri,omitempty"`
	UserCode        string               `json:"user_code,omitempty"`
	NeedsCode       bool                 `json:"needs_code"`
	Error           string               `json:"error,omitempty"`
	ExpiresIn       int                  `json:"expires_in"`
	Connection      *security.Connection `json:"connection,omitempty"`
}

func (s *Server) loginView(r *http.Request, l clilogin.Login) loginView {
	v := loginView{ID: l.ID, Provider: string(l.Provider), Name: l.Name, State: l.State, Error: l.Error}
	if left := int(time.Until(l.ExpiresAt).Seconds()); l.Open() && left > 0 {
		v.ExpiresIn = left
	}
	if l.State == clilogin.StateAwaitingUser {
		v.VerificationURI, v.UserCode, v.NeedsCode = l.URL, l.Code, l.NeedsCode
	}
	if l.State == clilogin.StateDone {
		if c, err := s.vault.Get(r.Context(), l.ConnectionID); err == nil {
			v.Connection = &c
		}
	}
	return v
}

func (s *Server) writeLoginError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, clilogin.ErrInProgress):
		writeError(w, r, http.StatusConflict, "login_in_progress", "Já existe uma conexão deste provedor em andamento.", "Conclua ou cancele a que está aberta.")
	case errors.Is(err, clilogin.ErrNameTaken):
		writeError(w, r, http.StatusConflict, "duplicate", "Já existe uma conexão ativa com esse nome.", "Escolha outro nome ou revogue a conexão antiga.")
	case errors.Is(err, clilogin.ErrNotWaiting):
		writeError(w, r, http.StatusConflict, "login_not_waiting", "Este login não está esperando um código.", "Use o link mostrado ou inicie de novo.")
	case errors.Is(err, clilogin.ErrBadCode):
		writeError(w, r, http.StatusBadRequest, "invalid_request", "O código colado não parece válido.", "Copie o código inteiro mostrado na página do Claude.")
	default:
		s.writeDomainError(w, r, err)
	}
}

func (s *Server) loginID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Login inválido.", "Inicie a conexão de novo.")
		return "", false
	}
	return id, true
}

func (s *Server) startCLILogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
		Name     string `json:"name"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	l, err := s.cliLogins.Start(r.Context(), clilogin.Provider(req.Provider), req.Name)
	if err != nil {
		s.writeLoginError(w, r, err)
		return
	}
	sess, _ := sessionFrom(r.Context())
	s.auth.Audit(r.Context(), sess.AdminID, "cli_login_started", s.meta(r), map[string]any{"provider": req.Provider})
	writeJSON(w, http.StatusCreated, s.loginView(r, l))
}

func (s *Server) getCLILogin(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loginID(w, r)
	if !ok {
		return
	}
	l, err := s.cliLogins.Get(r.Context(), id)
	if err != nil {
		s.writeLoginError(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, s.loginView(r, l))
}

func (s *Server) submitCLILoginCode(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loginID(w, r)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.cliLogins.SubmitCode(r.Context(), id, req.Code); err != nil {
		s.writeLoginError(w, r, err)
		return
	}
	l, err := s.cliLogins.Get(r.Context(), id)
	if err != nil {
		s.writeLoginError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, s.loginView(r, l))
}

func (s *Server) cancelCLILogin(w http.ResponseWriter, r *http.Request) {
	id, ok := s.loginID(w, r)
	if !ok {
		return
	}
	l, err := s.cliLogins.Cancel(r.Context(), id)
	if err != nil {
		s.writeLoginError(w, r, err)
		return
	}
	sess, _ := sessionFrom(r.Context())
	s.auth.Audit(r.Context(), sess.AdminID, "cli_login_cancelled", s.meta(r), map[string]any{"provider": string(l.Provider)})
	writeJSON(w, http.StatusOK, s.loginView(r, l))
}
