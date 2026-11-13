package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type ctxKey int

const (
	keyCorrelation ctxKey = iota
	keySession
)

var correlationPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{8,64}$`)

func correlationID(ctx context.Context) string {
	id, _ := ctx.Value(keyCorrelation).(string)
	return id
}

func sessionFrom(ctx context.Context) (security.Session, bool) {
	s, ok := ctx.Value(keySession).(security.Session)
	return s, ok
}

func newCorrelationID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Flush e Unwrap deixam respostas em fluxo (SSE) e o controle de prazos por requisição
// atravessarem os invólucros do middleware.
func (s *statusRecorder) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

func isProbe(path string) bool { return path == "/api/healthz" || path == "/api/readyz" }

// clientIP usa X-Forwarded-For só quando a API está atrás do proxy do painel.
func (s *Server) clientIP(r *http.Request) string {
	if s.cfg.TrustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			if ip := strings.TrimSpace(first); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (s *Server) meta(r *http.Request) security.Meta {
	return security.Meta{IP: s.clientIP(r), UserAgent: r.UserAgent(), CorrelationID: correlationID(r.Context())}
}

// isHTTPS considera TLS direto ou, atrás do proxy confiável, X-Forwarded-Proto.
func (s *Server) isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return s.cfg.TrustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// sameOrigin recusa requisições que alteram estado vindas de outra origem. Sem
// Origin nem Sec-Fetch-Site (clientes não-navegador) a decisão fica com o token
// CSRF, exigido em toda rota autenticada.
func (s *Server) sameOrigin(r *http.Request) bool {
	if isSafeMethod(r.Method) {
		return true
	}
	if origin := r.Header.Get("Origin"); origin != "" {
		u, err := url.Parse(origin)
		if err != nil || u.Host == "" {
			return false
		}
		host := r.Host
		if s.cfg.TrustProxy {
			if fh := r.Header.Get("X-Forwarded-Host"); fh != "" {
				host = fh
			}
		}
		return strings.EqualFold(u.Host, host)
	}
	if site := r.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	return true
}

// redactWriter segura respostas JSON e remove segredos conhecidos antes de
// enviá-las. Outros tipos (ex.: SSE futuro) passam direto.
type redactWriter struct {
	http.ResponseWriter
	red     *security.Redactor
	status  int
	buf     bytes.Buffer
	decided bool
	isJSON  bool
}

func (w *redactWriter) decide() {
	w.decided = true
	w.isJSON = strings.HasPrefix(w.Header().Get("Content-Type"), "application/json")
}

func (w *redactWriter) WriteHeader(code int) {
	if w.decided {
		return
	}
	w.status = code
	w.decide()
	if !w.isJSON {
		w.ResponseWriter.WriteHeader(code)
	}
}

func (w *redactWriter) Write(b []byte) (int, error) {
	if !w.decided {
		w.WriteHeader(http.StatusOK)
	}
	if w.isJSON {
		return w.buf.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

func (w *redactWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok && (!w.decided || !w.isJSON) {
		f.Flush()
	}
}

// Unwrap expõe o ResponseWriter de baixo (http.ResponseController).
func (w *redactWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (w *redactWriter) finish() {
	if !w.decided || !w.isJSON {
		return
	}
	out := w.red.RedactBytes(w.buf.Bytes())
	w.Header().Set("Content-Length", strconv.Itoa(len(out)))
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = w.ResponseWriter.Write(out)
}

// withMiddleware aplica, de fora para dentro: correlação, cabeçalhos de segurança,
// HTTPS obrigatório, checagem de origem, limite de corpo, redação de respostas,
// recuperação de panic e log estruturado por requisição.
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id := r.Header.Get("X-Request-ID")
		if !correlationPattern.MatchString(id) {
			id = newCorrelationID()
		}
		ctx := context.WithValue(r.Context(), keyCorrelation, id)
		r = r.WithContext(ctx)
		w.Header().Set("X-Request-ID", id)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if s.cfg.RequireHTTPS {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		rw := &redactWriter{ResponseWriter: rec, red: s.redactor}
		panicked := false
		defer func() {
			if p := recover(); p != nil {
				panicked = true
				s.log.ErrorContext(ctx, "panic na requisição", "panic", p, "stack", string(debug.Stack()), "correlation_id", id)
				writeError(rec, r, http.StatusInternalServerError, "internal_error", "Erro interno.", "Informe o ID de correlação.")
			}
			level := slog.LevelInfo
			if isProbe(r.URL.Path) {
				level = slog.LevelDebug // sondas de saúde não poluem o log padrão
			}
			s.log.Log(ctx, level, "requisição", "method", r.Method, "path", r.URL.Path,
				"status", rec.status, "duration_ms", time.Since(start).Milliseconds(), "correlation_id", id)
		}()

		switch {
		case s.cfg.RequireHTTPS && !isProbe(r.URL.Path) && !s.isHTTPS(r):
			writeError(rw, r, http.StatusBadRequest, "https_required", "Esta API só aceita HTTPS.", "Acesse pelo endereço seguro do painel.")
		case !s.sameOrigin(r):
			writeError(rw, r, http.StatusForbidden, "csrf_rejected", "Requisição de origem não permitida.", "Recarregue o painel e tente novamente.")
		default:
			next.ServeHTTP(rw, r)
		}
		if !panicked {
			rw.finish()
		}
	})
}

// protected exige sessão válida e, em métodos que alteram estado, o token CSRF
// da sessão. Sem sessão no banco (inexistente, expirada, revogada) a resposta é 401.
func (s *Server) protected(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.cfg.SessionCookieName)
		if err != nil {
			s.unauthorized(w, r)
			return
		}
		sess, err := security.Authenticate(r.Context(), s.db, cookie.Value, s.auth.ReauthWindow())
		if err != nil {
			if err == security.ErrNoSession {
				s.unauthorized(w, r)
				return
			}
			s.writeDomainError(w, r, err)
			return
		}
		if !isSafeMethod(r.Method) {
			got := r.Header.Get("X-CSRF-Token")
			if sess.CSRFToken == "" || subtle.ConstantTimeCompare([]byte(got), []byte(sess.CSRFToken)) != 1 {
				s.auth.Audit(r.Context(), sess.AdminID, "csrf_rejected", s.meta(r), map[string]any{"path": r.URL.Path})
				writeError(w, r, http.StatusForbidden, "csrf_rejected", "Token CSRF ausente ou inválido.", "Recarregue o painel e tente novamente.")
				return
			}
		}
		next(w, r.WithContext(context.WithValue(r.Context(), keySession, sess)))
	}
}

// sensitive = protected + reautenticação recente (senha + 2FA nos últimos minutos).
func (s *Server) sensitive(next http.HandlerFunc) http.HandlerFunc {
	return s.protected(func(w http.ResponseWriter, r *http.Request) {
		sess, _ := sessionFrom(r.Context())
		if sess.ReauthUntil == nil {
			writeError(w, r, http.StatusForbidden, "reauth_required", "Esta ação exige confirmar sua identidade.", "Informe senha e código 2FA e repita a ação.")
			return
		}
		next(w, r)
	})
}

func (s *Server) unauthorized(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusUnauthorized, "unauthorized", "Sessão ausente, expirada ou revogada.", "Entre novamente.")
}
