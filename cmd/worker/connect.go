package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// connectWithRetry tolera o banco ainda subindo (ex.: após reboot da VPS).
func connectWithRetry(ctx context.Context, url string, log *slog.Logger) (*pgxpool.Pool, error) {
	var lastErr error
	for attempt := 1; attempt <= 6; attempt++ {
		pool, err := persistence.Connect(ctx, url, 5*time.Second)
		if err == nil {
			return pool, nil
		}
		lastErr = err
		log.Warn("PostgreSQL indisponível; nova tentativa", "attempt", attempt, "error", err.Error())
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}
	return nil, fmt.Errorf("após 6 tentativas: %w", lastErr)
}
