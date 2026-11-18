// Package opslog grava os eventos operacionais da fábrica: jobs de fundo e rotinas agendadas do
// worker (sentinela, admissão, retenção, sincronização, cli-login) e os erros de configuração que
// acontecem uma vez no boot do worker, depois que o banco já está disponível. Serve o painel
// "Operação" (T16-05), fora do escopo por projeto já coberto por execution_events.
package opslog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Níveis e origens aceitos (espelham o CHECK de db/migrations/00073_operational_events.sql).
const (
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"

	SourceSentinela     = "sentinela"
	SourceAdmissao      = "admissao"
	SourceRetencao      = "retencao"
	SourceSincronizacao = "sincronizacao"
	SourceCLILogin      = "clilogin"
	SourceFabrica       = "fabrica"
)

var levels = map[string]bool{LevelInfo: true, LevelWarn: true, LevelError: true}

var sources = map[string]bool{
	SourceSentinela: true, SourceAdmissao: true, SourceRetencao: true,
	SourceSincronizacao: true, SourceCLILogin: true, SourceFabrica: true,
}

// Emit grava um evento operacional. Nível/origem fora do enum é erro de quem chamou (bug de
// wiring, não entrada de usuário): o chamador deve logar um aviso e seguir, nunca propagar isso
// como falha do job/rotina que originou o evento.
func Emit(ctx context.Context, db persistence.DBTX, level, source, message string, payload map[string]any) error {
	if !levels[level] {
		return fmt.Errorf("%w: nível de evento operacional desconhecido: %q", persistence.ErrInvalidInput, level)
	}
	if !sources[source] {
		return fmt.Errorf("%w: origem de evento operacional desconhecida: %q", persistence.ErrInvalidInput, source)
	}
	if payload == nil {
		payload = map[string]any{}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("payload do evento operacional: %w", err)
	}
	_, err = db.Exec(ctx, `INSERT INTO operational_events (level, source, message, payload) VALUES ($1, $2, $3, $4::jsonb)`,
		level, source, message, body)
	return persistence.MapError(err)
}
