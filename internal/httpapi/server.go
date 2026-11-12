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

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/logging"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Store é o que a API precisa do banco; *pgxpool.Pool satisfaz.
type Store interface {
	persistence.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
}

// Server agrupa dependências dos handlers.
type Server struct {
	cfg      config.API
	db       Store
	log      *slog.Logger
	auth     *security.Auth
	vault    *security.Vault
	redactor *security.Redactor
	tester   *integrations.Tester
}

// Option personaliza o servidor (usado em testes).
type Option func(*Server)

// WithTester troca o testador de conexões.
func WithTester(t *integrations.Tester) Option { return func(s *Server) { s.tester = t } }

// WithClock troca o relógio dos códigos TOTP (testes).
func WithClock(now func() time.Time) Option { return func(s *Server) { s.auth.SetClock(now) } }

// NewServer monta o servidor. A chave do cofre (config) é validada aqui.
func NewServer(cfg config.API, db Store, log *slog.Logger, opts ...Option) (*Server, error) {
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		return nil, err
	}
	redactor := security.NewRedactor()
	s := &Server{
		cfg: cfg, db: db, log: log, redactor: redactor,
		vault:  security.NewVault(db, sealer, redactor),
		tester: integrations.NewTester(),
		auth: security.NewAuth(db, sealer, security.AuthConfig{
			SessionTTL: cfg.SessionTTL,
		}, log),
	}
	for _, o := range opts {
		o(s)
	}
	return s, nil
}

// Prime carrega para o redator os segredos já gravados e o instala nos logs.
// Chamar uma vez na subida, com o banco disponível. Erro de banco impede a API
// de subir (sem isso valores gravados antes do restart não seriam redigidos).
// Conexões que não decifram (VAULT_KEY errada) geram só um aviso: o login por
// código de recuperação não depende da chave.
func (s *Server) Prime(ctx context.Context) error {
	n, err := s.vault.LoadRedactor(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		s.log.Warn("conexões ativas não puderam ser decifradas: VAULT_KEY incorreta?", "quantidade", n)
	}
	logging.SetRedactor(s.redactor.Redact)
	return nil
}

// New cria o handler HTTP completo (sem Prime). Atalho para testes.
func New(cfg config.API, db Store, log *slog.Logger, opts ...Option) (http.Handler, error) {
	s, err := NewServer(cfg, db, log, opts...)
	if err != nil {
		return nil, err
	}
	return s.Handler(), nil
}

// Handler devolve rotas + middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", s.healthz)
	mux.HandleFunc("GET /api/readyz", s.readyz)

	// Login em duas etapas (sem sessão; protegido por origem, throttle e desafio).
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/totp", s.loginTOTP)
	mux.HandleFunc("POST /api/auth/recover", s.loginRecover)
	mux.HandleFunc("POST /api/auth/totp/setup", s.totpSetup)
	mux.HandleFunc("POST /api/auth/totp/enable", s.totpEnable)

	// Sessão e ações de conta.
	mux.HandleFunc("GET /api/session", s.protected(s.getSession))
	mux.HandleFunc("POST /api/auth/logout", s.protected(s.logout))
	mux.HandleFunc("POST /api/auth/reauth", s.protected(s.reauth))
	mux.HandleFunc("GET /api/auth/events", s.protected(s.listAuthEvents))
	mux.HandleFunc("POST /api/auth/totp/replace/begin", s.sensitive(s.totpReplaceBegin))
	mux.HandleFunc("POST /api/auth/totp/replace/confirm", s.sensitive(s.totpReplaceConfirm))
	mux.HandleFunc("GET /api/sessions", s.protected(s.listSessions))
	mux.HandleFunc("DELETE /api/sessions/{id}", s.protected(s.revokeSession))
	mux.HandleFunc("POST /api/sessions/revoke-others", s.sensitive(s.revokeOtherSessions))

	// Cofre de conexões.
	mux.HandleFunc("GET /api/connections", s.protected(s.listConnections))
	mux.HandleFunc("POST /api/connections", s.sensitive(s.createConnection))
	mux.HandleFunc("GET /api/connections/{id}", s.protected(s.getConnection))
	mux.HandleFunc("POST /api/connections/{id}/test", s.protected(s.testConnection))
	mux.HandleFunc("POST /api/connections/{id}/rotate", s.sensitive(s.rotateConnection))
	mux.HandleFunc("POST /api/connections/{id}/revoke", s.sensitive(s.revokeConnection))

	mux.HandleFunc("GET /api/projects", s.protected(s.listProjects))
	mux.HandleFunc("GET /api/projects/{id}", s.protected(s.getProject))
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
