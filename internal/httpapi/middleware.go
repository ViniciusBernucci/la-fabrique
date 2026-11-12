package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"regexp"
	"runtime/debug"
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

// withMiddleware aplica, de fora para dentro: correlação, cabeçalhos de segurança,
// recuperação de panic, limite de corpo e log estruturado por requisição.
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
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		defer func() {
			if p := recover(); p != nil {
				s.log.ErrorContext(ctx, "panic na requisição", "panic", p, "stack", string(debug.Stack()), "correlation_id", id)
				writeError(rec, r, http.StatusInternalServerError, "internal_error", "Erro interno.", "Informe o ID de correlação.")
			}
			level := slog.LevelInfo
			if r.URL.Path == "/api/healthz" || r.URL.Path == "/api/readyz" {
				level = slog.LevelDebug // sondas de saúde não poluem o log padrão
			}
			s.log.Log(ctx, level, "requisição", "method", r.Method, "path", r.URL.Path,
				"status", rec.status, "duration_ms", time.Since(start).Milliseconds(), "correlation_id", id)
		}()
		next.ServeHTTP(rec, r)
	})
}

// requireSession é o portão de autenticação aplicado no servidor: sem sessão
// válida no banco (existente, não expirada, não revogada) a resposta é 401.
func (s *Server) requireSession(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(s.cfg.SessionCookieName)
		if err != nil {
			s.unauthorized(w, r)
			return
		}
		sess, err := security.Authenticate(r.Context(), s.db, cookie.Value)
		if err != nil {
			if err == security.ErrNoSession {
				s.unauthorized(w, r)
				return
			}
			s.writeDomainError(w, r, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), keySession, sess)))
	}
}

func (s *Server) unauthorized(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusUnauthorized, "unauthorized", "Sessão ausente, expirada ou revogada.", "Entre novamente.")
}
