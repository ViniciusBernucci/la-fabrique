// Package persistence reúne pool, transações, migrations e os repositórios do
// núcleo de execução (features, execuções, eventos, outbox e artefatos).
//
// Regras: toda consulta recebe o escopo de projeto validado; mutações usam
// controle otimista por `version`; erros do PostgreSQL são traduzidos para os
// erros de domínio abaixo.
package persistence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Erros de domínio estáveis para as camadas superiores.
var (
	ErrNotFound        = errors.New("registro não encontrado")
	ErrVersionConflict = errors.New("versão obsoleta: o registro foi alterado por outra operação")
	ErrDuplicate       = errors.New("registro duplicado")
	ErrInvalidScope    = errors.New("referência fora do escopo do projeto")
	ErrInvalidInput    = errors.New("dados inválidos")
)

// DBTX é implementado por *pgxpool.Pool e pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Connect abre o pool e confirma conectividade em até timeout.
func Connect(ctx context.Context, databaseURL string, timeout time.Duration) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("criar pool PostgreSQL: %w", redactConnErr(err))
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("conectar ao PostgreSQL: %w", redactConnErr(err))
	}
	return pool, nil
}

// redactConnErr evita que mensagens de driver carreguem a URL completa.
func redactConnErr(err error) error {
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		return fmt.Errorf("SQLSTATE %s: %s", pe.Code, pe.Message)
	}
	var ce *pgconn.ConnectError
	if errors.As(err, &ce) {
		return fmt.Errorf("falha de conexão: %v", ce.Unwrap())
	}
	return errors.New("falha de conexão ou configuração do pool")
}

// WithTx executa fn em uma transação e faz commit se fn retornar nil.
// Qualquer erro ou panic desfaz a transação.
func WithTx(ctx context.Context, pool *pgxpool.Pool, fn func(tx pgx.Tx) error) (err error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("iniciar transação: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback(context.WithoutCancel(ctx))
		}
	}()
	if err = fn(tx); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// MapError traduz erros do PostgreSQL para os erros de domínio.
func MapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(err, &pe) {
		switch pe.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", ErrDuplicate, pe.ConstraintName)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", ErrInvalidScope, pe.ConstraintName)
		case "23514", "23502", "22P02": // check, not null, invalid text representation
			return fmt.Errorf("%w: %s", ErrInvalidInput, pe.Message)
		}
	}
	return err
}
