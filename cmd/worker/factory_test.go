package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// recordingDB grava o SQL e os argumentos do último Exec, para conferir o que opslog.Emit grava
// sem precisar de um banco de verdade (T16-05, CA-03).
type recordingDB struct {
	sql  string
	args []any
}

func (r *recordingDB) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	r.sql, r.args = sql, args
	return pgconn.CommandTag{}, nil
}
func (r *recordingDB) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("não usado neste teste")
}
func (r *recordingDB) QueryRow(context.Context, string, ...any) pgx.Row { return nil }

// T16-05, CA-03 — "proxy de saída indisponível" grava um evento operacional error/fabrica.
func TestReportEgressProxyUnavailableLogsAndRecordsAnOperationalEvent(t *testing.T) {
	db := &recordingDB{}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	reportEgressProxyUnavailable(context.Background(), db, log, errors.New("imagem ausente"))
	if !strings.Contains(db.sql, "INSERT INTO operational_events") {
		t.Fatalf("não gravou: %q", db.sql)
	}
	if len(db.args) < 3 || db.args[0] != "error" || db.args[1] != "fabrica" {
		t.Fatalf("args: %+v", db.args)
	}
	if msg, ok := db.args[2].(string); !ok || !strings.Contains(msg, "proxy de saída indisponível") {
		t.Fatalf("mensagem: %+v", db.args[2])
	}
}
