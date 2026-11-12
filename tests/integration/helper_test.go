// Package integration executa testes contra um PostgreSQL real.
//
// Defina TEST_DATABASE_URL apontando para um banco administrativo (ex.:
// postgres://user:pass@127.0.0.1:5442/postgres). Cada teste cria e descarta um
// banco próprio; sem a variável, os testes são ignorados e `make test-integration`
// falha explicitamente.
package integration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

type testDB struct {
	URL  string
	Pool *pgxpool.Pool
}

// newDatabase cria um banco vazio (sem migrations) e o remove no cleanup.
func newDatabase(t *testing.T) string {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL não definida; teste de integração ignorado")
	}
	ctx := context.Background()
	admin, err := persistence.Connect(ctx, adminURL, 10*time.Second)
	if err != nil {
		t.Fatalf("conectar ao banco administrativo: %v", err)
	}
	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	name := "lf_test_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		admin.Close()
		t.Fatalf("criar banco de teste: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
		admin.Close()
	})
	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/" + name
	return u.String()
}

func connect(t *testing.T, dbURL string) *pgxpool.Pool {
	t.Helper()
	pool, err := persistence.Connect(context.Background(), dbURL, 10*time.Second)
	if err != nil {
		t.Fatalf("conectar: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// newMigratedDB devolve um banco com todas as migrations aplicadas.
func newMigratedDB(t *testing.T) testDB {
	t.Helper()
	dbURL := newDatabase(t)
	m, err := persistence.NewMigrator(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if _, err := m.Up(context.Background()); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	return testDB{URL: dbURL, Pool: connect(t, dbURL)}
}
