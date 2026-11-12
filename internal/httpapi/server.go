// Package httpapi expõe a API interna da fábrica consumida pelo painel web.
// Contrato: contracts/openapi.yaml.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Store é o que a API precisa do banco; *pgxpool.Pool satisfaz.
type Store interface {
	persistence.DBTX
	Ping(ctx context.Context) error
}

// Server agrupa dependências dos handlers.
type Server struct {
	cfg config.API
	db  Store
	log *slog.Logger
}

// New cria o handler HTTP completo (rotas + middleware).
func New(cfg config.API, db Store, log *slog.Logger) http.Handler {
	s := &Server{cfg: cfg, db: db, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", s.healthz)
	mux.HandleFunc("GET /api/readyz", s.readyz)
	mux.HandleFunc("GET /api/session", s.requireSession(s.getSession))
	mux.HandleFunc("GET /api/projects", s.requireSession(s.listProjects))
	mux.HandleFunc("GET /api/projects/{id}", s.requireSession(s.getProject))
	if cfg.DevAuthEnabled {
		mux.HandleFunc("POST /api/dev/session", s.createDevSession)
	}
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, r, http.StatusNotFound, "not_found", "Rota inexistente.", "Consulte contracts/openapi.yaml.")
	})
	return s.withMiddleware(mux)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		s.log.WarnContext(r.Context(), "banco indisponível no readyz", "correlation_id", correlationID(r.Context()))
		writeError(w, r, http.StatusServiceUnavailable, "database_unavailable", "Banco de dados indisponível.", "Verifique o PostgreSQL e tente novamente.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	sess, _ := sessionFrom(r.Context())
	writeJSON(w, http.StatusOK, sess)
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (s *Server) getProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return
	}
	p, err := projects.Get(r.Context(), s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	params := projects.ListParams{Cursor: q.Get("cursor"), State: q.Get("state")}
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeError(w, r, http.StatusBadRequest, "invalid_request", "limit deve ser um inteiro positivo.", "Use limit entre 1 e 200.")
			return
		}
		params.Limit = n
	}
	page, err := projects.List(r.Context(), s.db, params)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// createDevSession é a ÚNICA forma de obter sessão enquanto o login (T04-01) não
// existe. Só é registrada com DEV_AUTH_ENABLED=true, que a configuração recusa
// fora de APP_ENV=development. Remover quando o T04-01 for entregue.
func (s *Server) createDevSession(w http.ResponseWriter, r *http.Request) {
	var adminID string
	err := s.db.QueryRow(r.Context(), `
		INSERT INTO admins (username, password_hash) VALUES ('dev-admin', 'dev-auth-no-password')
		ON CONFLICT (username) DO UPDATE SET username = EXCLUDED.username
		RETURNING id`).Scan(&adminID)
	if err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	token, sess, err := security.CreateSession(r.Context(), s.db, adminID, s.cfg.SessionTTL)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: s.cfg.SessionCookieName, Value: token, Path: "/",
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
		HttpOnly: true, Secure: s.cfg.SessionCookieSecure, SameSite: http.SameSiteStrictMode,
	})
	writeJSON(w, http.StatusCreated, sess)
}
