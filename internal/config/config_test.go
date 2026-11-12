package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) Lookup {
	return func(k string) (string, bool) { v, ok := m[k]; return v, ok }
}

const goodDB = "postgres://u:s3cr3t-pass@db:5432/app"

func TestLoadAPIDefaults(t *testing.T) {
	cfg, err := LoadAPI(env(map[string]string{"DATABASE_URL": goodDB}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Addr != "127.0.0.1:8480" || cfg.Env != Development || cfg.SessionTTL != 12*time.Hour || cfg.SessionCookieSecure {
		t.Fatalf("defaults inesperados: %+v", cfg)
	}
}

func TestLoadAPIReportsEveryProblemWithoutSecrets(t *testing.T) {
	_, err := LoadAPI(env(map[string]string{
		"DATABASE_URL": "mysql://u:s3cr3t-pass@db/app",
		"API_ADDR":     "8480",
		"APP_ENV":      "staging",
		"SESSION_TTL":  "abc",
	}))
	if !IsConfigError(err) {
		t.Fatalf("esperava *Error, veio %v", err)
	}
	msg := err.Error()
	for _, want := range []string{"DATABASE_URL", "API_ADDR", "APP_ENV", "SESSION_TTL"} {
		if !strings.Contains(msg, want) {
			t.Errorf("mensagem deveria citar %s: %s", want, msg)
		}
	}
	if strings.Contains(msg, "s3cr3t-pass") {
		t.Fatalf("mensagem vazou segredo: %s", msg)
	}
}

func TestLoadAPIRequiresDatabaseURL(t *testing.T) {
	_, err := LoadAPI(env(nil))
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL é obrigatória") {
		t.Fatalf("erro inesperado: %v", err)
	}
}

func TestDevAuthForbiddenOutsideDevelopment(t *testing.T) {
	_, err := LoadAPI(env(map[string]string{
		"DATABASE_URL": goodDB, "APP_ENV": "production", "DEV_AUTH_ENABLED": "true", "SESSION_COOKIE_SECURE": "true",
	}))
	if err == nil || !strings.Contains(err.Error(), "DEV_AUTH_ENABLED") {
		t.Fatalf("produção deve recusar DEV_AUTH_ENABLED: %v", err)
	}
}

func TestProductionRequiresSecureCookie(t *testing.T) {
	_, err := LoadAPI(env(map[string]string{
		"DATABASE_URL": goodDB, "APP_ENV": "production", "SESSION_COOKIE_SECURE": "false",
	}))
	if err == nil || !strings.Contains(err.Error(), "SESSION_COOKIE_SECURE") {
		t.Fatalf("produção deve exigir cookie seguro: %v", err)
	}
}

func TestLoadWorker(t *testing.T) {
	cfg, err := LoadWorker(env(map[string]string{"DATABASE_URL": goodDB, "WORKER_HEARTBEAT_INTERVAL": "5s"}))
	if err != nil || cfg.HeartbeatInterval != 5*time.Second || cfg.HealthAddr != "127.0.0.1:8481" {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
	if _, err := LoadWorker(env(map[string]string{"DATABASE_URL": goodDB, "WORKER_HEARTBEAT_INTERVAL": "100ms"})); err == nil {
		t.Fatal("intervalo abaixo do mínimo deveria falhar")
	}
}
