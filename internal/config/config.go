// Package config carrega e valida a configuração por variáveis de ambiente.
//
// A validação acumula todos os problemas e nomeia as variáveis envolvidas, mas
// nunca imprime valores de segredos (ex.: senha da DATABASE_URL).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Lookup abstrai os.LookupEnv para facilitar testes.
type Lookup func(key string) (string, bool)

// Environment identifica o ambiente de execução.
type Environment string

const (
	Development Environment = "development"
	Test        Environment = "test"
	Production  Environment = "production"
)

// Common reúne o que todos os binários precisam.
type Common struct {
	Env         Environment
	LogLevel    string
	DatabaseURL string
}

// API é a configuração do cmd/api.
type API struct {
	Common
	Addr                string
	SessionTTL          time.Duration
	SessionCookieName   string
	SessionCookieSecure bool
	// RequireHTTPS recusa requisições que o proxy não marcou como HTTPS
	// (X-Forwarded-Proto) e envia HSTS. Padrão: ligado em produção.
	RequireHTTPS bool
	// TrustProxy confia em X-Forwarded-For/Host/Proto (API atrás do nginx do painel).
	TrustProxy bool
	// VaultKey é a chave AES-256 (32 bytes) do cofre e do segredo TOTP. Fica
	// fora do banco e nunca é logada.
	VaultKey   []byte
	VaultKeyID string
}

// Worker é a configuração do cmd/worker.
type Worker struct {
	Common
	HealthAddr        string
	HeartbeatInterval time.Duration
	// JobsInterval é o intervalo das tarefas leves (liberar decisões, recuperar tentativas).
	JobsInterval time.Duration
	// RetentionInterval é o intervalo da retenção; RetentionApply liga a remoção real (padrão:
	// só simula e registra).
	RetentionInterval time.Duration
	RetentionApply    bool
	// IssueSync liga o sync matinal das Issues pelo worker (SENTINEL_ISSUE_SYNC=true). Desligado por
	// padrão: exige VAULT_KEY no worker para abrir as conexões do cofre, o que é uma decisão de
	// implantação. Hora e fuso definem a janela diária.
	IssueSync     bool
	IssueSyncHour int
	IssueSyncTZ   string
	// CorrectionsPollInterval é o intervalo da consulta direcionada das correções aguardando liberação.
	CorrectionsPollInterval time.Duration
	// PRSync liga a consulta das PRs gerenciadas (SENTINEL_PR_SYNC=true), com agenda própria (PR_POLL_INTERVAL).
	PRSync         bool
	PRPollInterval time.Duration
	VaultKey       []byte
	VaultKeyID     string
}

// Migrate é a configuração do cmd/migrate.
type Migrate struct {
	Common
}

// Error agrega todos os problemas de configuração encontrados.
type Error struct {
	Problems []string
}

func (e *Error) Error() string {
	return "configuração inválida:\n  - " + strings.Join(e.Problems, "\n  - ")
}

// IsConfigError informa se err é um *Error de configuração.
func IsConfigError(err error) bool {
	var target *Error
	return errors.As(err, &target)
}

type loader struct {
	lookup   Lookup
	problems []string
}

func (l *loader) fail(format string, args ...any) {
	l.problems = append(l.problems, fmt.Sprintf(format, args...))
}

func (l *loader) result() error {
	if len(l.problems) == 0 {
		return nil
	}
	sort.Strings(l.problems)
	return &Error{Problems: l.problems}
}

func (l *loader) str(key, def string) string {
	if v, ok := l.lookup(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

func (l *loader) required(key string) string {
	v := l.str(key, "")
	if v == "" {
		l.fail("%s é obrigatória e não foi definida", key)
	}
	return v
}

func (l *loader) boolean(key string, def bool) bool {
	raw, ok := l.lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def
	}
	b, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		l.fail("%s deve ser true ou false", key)
		return def
	}
	return b
}

func (l *loader) duration(key string, def, min time.Duration) time.Duration {
	raw, ok := l.lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return def
	}
	d, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil {
		l.fail("%s deve ser uma duração como 15s ou 12h", key)
		return def
	}
	if d < min {
		l.fail("%s deve ser pelo menos %s", key, min)
		return def
	}
	return d
}

func (l *loader) addr(key, def string) string {
	v := l.str(key, def)
	if _, port, err := net.SplitHostPort(v); err != nil {
		l.fail("%s deve ter o formato host:porta (ex.: 127.0.0.1:8480)", key)
	} else if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		l.fail("%s tem porta fora de 1-65535", key)
	}
	return v
}

func (l *loader) common() Common {
	c := Common{}
	switch env := Environment(l.str("APP_ENV", string(Development))); env {
	case Development, Test, Production:
		c.Env = env
	default:
		l.fail("APP_ENV deve ser development, test ou production")
		c.Env = Development
	}
	c.LogLevel = strings.ToLower(l.str("LOG_LEVEL", "info"))
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		l.fail("LOG_LEVEL deve ser debug, info, warn ou error")
	}
	c.DatabaseURL = l.required("DATABASE_URL")
	if c.DatabaseURL != "" {
		if err := validateDatabaseURL(c.DatabaseURL); err != nil {
			l.fail("DATABASE_URL %s", err.Error())
		}
	}
	return c
}

// validateDatabaseURL nunca inclui o valor original na mensagem.
func validateDatabaseURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return errors.New("não é uma URL válida")
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return errors.New("deve usar o esquema postgres:// ou postgresql://")
	}
	if u.Hostname() == "" {
		return errors.New("deve informar o host do PostgreSQL")
	}
	if strings.Trim(u.Path, "/") == "" {
		return errors.New("deve informar o nome do banco")
	}
	return nil
}

// LoadAPI valida a configuração da API.
func LoadAPI(lookup Lookup) (API, error) {
	l := &loader{lookup: lookup}
	cfg := API{Common: l.common()}
	cfg.Addr = l.addr("API_ADDR", "127.0.0.1:8480")
	cfg.SessionTTL = l.duration("SESSION_TTL", 12*time.Hour, time.Minute)
	cfg.SessionCookieName = l.str("SESSION_COOKIE_NAME", "lf_session")
	cfg.SessionCookieSecure = l.boolean("SESSION_COOKIE_SECURE", cfg.Env == Production)
	if _, ok := l.lookup("DEV_AUTH_ENABLED"); ok && l.boolean("DEV_AUTH_ENABLED", false) {
		l.fail("DEV_AUTH_ENABLED foi removida no T04-01 (ADR-0005); crie a conta com `make create-admin`")
	}
	cfg.RequireHTTPS = l.boolean("REQUIRE_HTTPS", cfg.Env == Production)
	cfg.TrustProxy = l.boolean("TRUST_PROXY", false)
	cfg.VaultKeyID = l.str("VAULT_KEY_ID", "k1")
	cfg.VaultKey = l.vaultKey()
	if cfg.Env == Production && !cfg.RequireHTTPS {
		l.fail("REQUIRE_HTTPS deve ser true com APP_ENV=production")
	}
	if cfg.Env == Production && !cfg.SessionCookieSecure {
		l.fail("SESSION_COOKIE_SECURE deve ser true com APP_ENV=production")
	}
	return cfg, l.result()
}

// vaultKey decodifica VAULT_KEY (base64 de 32 bytes). Nunca inclui o valor na mensagem.
func (l *loader) vaultKey() []byte {
	raw := l.required("VAULT_KEY")
	if raw == "" {
		return nil
	}
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		if b, err := enc.DecodeString(raw); err == nil {
			if len(b) != 32 {
				break
			}
			return b
		}
	}
	l.fail("VAULT_KEY deve ser base64 de exatamente 32 bytes (ex.: openssl rand -base64 32)")
	return nil
}

// LoadWorker valida a configuração do worker.
func LoadWorker(lookup Lookup) (Worker, error) {
	l := &loader{lookup: lookup}
	cfg := Worker{Common: l.common()}
	cfg.HealthAddr = l.addr("WORKER_HEALTH_ADDR", "127.0.0.1:8481")
	cfg.HeartbeatInterval = l.duration("WORKER_HEARTBEAT_INTERVAL", 15*time.Second, time.Second)
	cfg.JobsInterval = l.duration("WORKER_JOBS_INTERVAL", 30*time.Second, time.Second)
	cfg.RetentionInterval = l.duration("RETENTION_INTERVAL", 24*time.Hour, time.Minute)
	cfg.RetentionApply = l.boolean("RETENTION_APPLY", false)
	cfg.IssueSync = l.boolean("SENTINEL_ISSUE_SYNC", false)
	cfg.PRSync = l.boolean("SENTINEL_PR_SYNC", false)
	if cfg.PRSync {
		cfg.PRPollInterval = l.duration("PR_POLL_INTERVAL", 3*time.Minute, 30*time.Second)
	}
	if cfg.IssueSync || cfg.PRSync {
		cfg.VaultKeyID = l.str("VAULT_KEY_ID", "k1")
		cfg.VaultKey = l.vaultKey()
	}
	if cfg.IssueSync {
		cfg.IssueSyncTZ = l.str("ISSUE_SYNC_TZ", "America/Sao_Paulo")
		if _, err := time.LoadLocation(cfg.IssueSyncTZ); err != nil {
			l.fail("ISSUE_SYNC_TZ não é um fuso válido")
		}
		hour, err := strconv.Atoi(l.str("ISSUE_SYNC_HOUR", "7"))
		if err != nil || hour < 0 || hour > 23 {
			l.fail("ISSUE_SYNC_HOUR deve ser de 0 a 23")
		}
		cfg.IssueSyncHour = hour
		cfg.CorrectionsPollInterval = l.duration("CORRECTIONS_POLL_INTERVAL", 5*time.Minute, 30*time.Second)
	}
	return cfg, l.result()
}

// LoadMigrate valida a configuração do migrador.
func LoadMigrate(lookup Lookup) (Migrate, error) {
	l := &loader{lookup: lookup}
	cfg := Migrate{Common: l.common()}
	return cfg, l.result()
}
