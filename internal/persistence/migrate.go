package persistence

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/ViniciusBernucci/le-fabrique/db/migrations"
)

// Migrator aplica as migrations embutidas usando uma conexão própria
// (identidade e ciclo de vida separados dos serviços de aplicação).
type Migrator struct {
	db       *sql.DB
	provider *goose.Provider
}

// NewMigrator abre conexão database/sql para o goose.
func NewMigrator(databaseURL string) (*Migrator, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("abrir conexão de migração: %w", redactConnErr(err))
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("carregar migrations: %w", err)
	}
	return &Migrator{db: db, provider: provider}, nil
}

// Close libera a conexão.
func (m *Migrator) Close() error { return m.db.Close() }

// Up aplica todas as migrations pendentes.
func (m *Migrator) Up(ctx context.Context) ([]*goose.MigrationResult, error) {
	return m.provider.Up(ctx)
}

// UpTo aplica até a versão informada (inclusive).
func (m *Migrator) UpTo(ctx context.Context, version int64) ([]*goose.MigrationResult, error) {
	return m.provider.UpTo(ctx, version)
}

// Down desfaz a última migration aplicada.
func (m *Migrator) Down(ctx context.Context) (*goose.MigrationResult, error) {
	return m.provider.Down(ctx)
}

// Status lista as migrations e seu estado.
func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	return m.provider.Status(ctx)
}

var _ = stdlib.GetDefaultDriver // garante o registro do driver "pgx"
