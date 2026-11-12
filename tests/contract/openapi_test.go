// Package contract verifica que o contrato (contracts/openapi.yaml) e a API
// implementada não divergem: toda operação declarada precisa estar roteada.
package contract

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
)

type errRow struct{}

func (errRow) Scan(...any) error { return errors.New("sem banco") }

// stubStore faz toda consulta falhar: basta para provar que a rota existe.
type stubStore struct{}

func (stubStore) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("sem banco")
}
func (stubStore) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("sem banco")
}
func (stubStore) QueryRow(context.Context, string, ...any) pgx.Row { return errRow{} }
func (stubStore) Ping(context.Context) error                       { return errors.New("sem banco") }

var (
	pathLine   = regexp.MustCompile(`^  (/[^:\s]*):\s*$`)
	methodLine = regexp.MustCompile(`^    (get|post|put|patch|delete):\s*$`)
)

type operation struct{ method, path string }

func declaredOperations(t *testing.T) []operation {
	t.Helper()
	raw, err := os.ReadFile("../../contracts/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var ops []operation
	current := ""
	inPaths := false
	for _, line := range strings.Split(string(raw), "\n") {
		switch {
		case strings.HasPrefix(line, "paths:"):
			inPaths = true
		case inPaths && line != "" && !strings.HasPrefix(line, " "):
			inPaths = false
		case inPaths:
			if m := pathLine.FindStringSubmatch(line); m != nil {
				current = m[1]
			} else if m := methodLine.FindStringSubmatch(line); m != nil && current != "" {
				ops = append(ops, operation{strings.ToUpper(m[1]), current})
			}
		}
	}
	if len(ops) == 0 {
		t.Fatal("nenhuma operação encontrada no contrato")
	}
	return ops
}

func TestEveryDeclaredOperationIsRouted(t *testing.T) {
	cfg := config.API{
		Common: config.Common{Env: config.Development}, Addr: "127.0.0.1:0",
		SessionTTL: time.Hour, SessionCookieName: "lf_session", DevAuthEnabled: true,
	}
	h := httpapi.New(cfg, stubStore{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	for _, op := range declaredOperations(t) {
		path := "/api" + strings.ReplaceAll(op.path, "{id}", "00000000-0000-4000-8000-000000000000")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(op.method, path, nil))
		if rec.Code == http.StatusNotFound && strings.Contains(rec.Body.String(), "Rota inexistente") {
			t.Errorf("%s %s está no contrato mas não está roteado", op.method, op.path)
		}
		if rec.Code == http.StatusMethodNotAllowed {
			t.Errorf("%s %s: método não implementado", op.method, op.path)
		}
	}
}
