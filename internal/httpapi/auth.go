package httpapi

import (
	"net/http"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const challengeCookie = "lf_challenge"

func (s *Server) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cfg.SessionCookieName, Value: token, Path: "/",
		MaxAge:   int(s.auth.SessionTTL().Seconds()),
		HttpOnly: true, Secure: s.cfg.SessionCookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: s.cfg.SessionCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.SessionCookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) setChallengeCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: challengeCookie, Value: token, Path: "/api/auth", MaxAge: 600,
		HttpOnly: true, Secure: s.cfg.SessionCookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: challengeCookie, Value: "", Path: "/api/auth", MaxAge: -1,
		HttpOnly: true, Secure: s.cfg.SessionCookieSecure, SameSite: http.SameSiteStrictMode,
	})
}

func (s *Server) challengeToken(r *http.Request) string {
	c, err := r.Cookie(challengeCookie)
	if err != nil {
		return ""
	}
	return c.Value
}

// login: senha correta NÃO abre sessão; abre um desafio que exige o 2º fator.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	tok, stage, err := s.auth.StartLogin(r.Context(), req.Username, req.Password, s.meta(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.setChallengeCookie(w, tok)
	writeJSON(w, http.StatusOK, map[string]string{"status": stage})
}

func (s *Server) finishLogin(w http.ResponseWriter, token string, sess security.Session, status int, extra any) {
	s.clearChallengeCookie(w)
	s.setSessionCookie(w, token)
	if extra == nil {
		writeJSON(w, status, sess)
		return
	}
	writeJSON(w, status, extra)
}

func (s *Server) loginTOTP(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	token, sess, err := s.auth.VerifyTOTP(r.Context(), s.challengeToken(r), req.Code, s.meta(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.finishLogin(w, token, sess, http.StatusOK, nil)
}

func (s *Server) loginRecover(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RecoveryCode string `json:"recovery_code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	token, sess, err := s.auth.UseRecoveryCode(r.Context(), s.challengeToken(r), req.RecoveryCode, s.meta(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.finishLogin(w, token, sess, http.StatusOK, nil)
}

func (s *Server) totpSetup(w http.ResponseWriter, r *http.Request) {
	enr, err := s.auth.BeginEnrollment(r.Context(), s.challengeToken(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, enr)
}

type sessionWithCodes struct {
	security.Session
	RecoveryCodes []string `json:"recovery_codes"`
}

func (s *Server) totpEnable(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	token, sess, codes, err := s.auth.ConfirmEnrollment(r.Context(), s.challengeToken(r), req.Code, s.meta(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.finishLogin(w, token, sess, http.StatusCreated, sessionWithCodes{Session: sess, RecoveryCodes: codes})
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	writeJSON(w, http.StatusOK, sess)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	if err := s.auth.Logout(r.Context(), sess, s.meta(r)); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	s.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) reauth(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	var req struct {
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := s.auth.Reauthenticate(r.Context(), sess, req.Password, req.Code, s.meta(r)); err != nil {
		// 403 (não 401): a sessão continua válida; só a confirmação falhou.
		if err == security.ErrInvalidCredentials || err == security.ErrInvalidCode {
			writeError(w, r, http.StatusForbidden, "reauth_failed", "Senha ou código incorretos.", "Confira os dados e tente novamente.")
			return
		}
		s.writeDomainError(w, r, err)
		return
	}
	fresh, err := security.Authenticate(r.Context(), s.db, mustCookie(r, s.cfg.SessionCookieName), s.auth.ReauthWindow())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, fresh)
}

func mustCookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func (s *Server) listAuthEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.auth.ListEvents(r.Context(), 50)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": events})
}

func (s *Server) totpReplaceBegin(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	enr, err := s.auth.BeginReplacement(r.Context(), sess)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, enr)
}

func (s *Server) totpReplaceConfirm(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	var req struct {
		Code string `json:"code"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	codes, err := s.auth.ConfirmReplacement(r.Context(), sess, req.Code, s.meta(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recovery_codes": codes})
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	items, err := s.auth.ListSessions(r.Context(), sess.AdminID, sess.ID)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// revokeSession: encerrar a própria sessão é livre; encerrar outra exige reautenticação.
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return
	}
	if id != sess.ID && sess.ReauthUntil == nil {
		writeError(w, r, http.StatusForbidden, "reauth_required", "Esta ação exige confirmar sua identidade.", "Informe senha e código 2FA e repita a ação.")
		return
	}
	if err := s.auth.RevokeSession(r.Context(), sess, id, s.meta(r)); err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if id == sess.ID {
		s.clearSessionCookie(w)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) revokeOtherSessions(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	n, err := s.auth.RevokeOthers(r.Context(), sess, s.meta(r))
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int64{"revoked": n})
}
