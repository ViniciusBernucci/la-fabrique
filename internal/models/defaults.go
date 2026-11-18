package models

import (
	"context"
	"errors"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// defaultModels é o catálogo inicial das contas de assinatura (ADR-0067). Sem isso o catálogo nasce
// vazio e a verificação de compatibilidade (Check) reprova qualquer modelo escolhido num agente
// ("não está cadastrado"), e a tela não teria o que listar.
//
// Fontes (verificadas em 2026-11-16, sem executar os clientes):
//   - Claude: IDs da família atual (Fable 5.2, Opus 5.5, Sonnet 5.5, Haiku 5.5). Janela de contexto e saída
//     máxima ficam 0 (não informadas): não se inventa o que não foi observado.
//   - Codex: modelos de visibilidade "list" do cache do próprio cliente (models_cache.json), com a janela
//     de contexto que ele informa. Os ocultos (gpt-reserve, codex-auto-review) ficam de fora.
var defaultModels = []Input{
	{Provider: "claude", Name: "claude-fable-5-2", Label: "Claude Fable 5.2", SupportsTools: true, SupportsStreaming: true},
	{Provider: "claude", Name: "claude-opus-5-5", Label: "Claude Opus 5.5", SupportsTools: true, SupportsStreaming: true},
	{Provider: "claude", Name: "claude-sonnet-5-5", Label: "Claude Sonnet 5.5", SupportsTools: true, SupportsStreaming: true},
	{Provider: "claude", Name: "claude-haiku-5-5", Label: "Claude Haiku 5.5", SupportsTools: true, SupportsStreaming: true},

	{Provider: "codex", Name: "gpt-6.1-sol", Label: "GPT-6.1 Sol", SupportsTools: true, SupportsStreaming: true, ContextWindow: 500000},
	{Provider: "codex", Name: "gpt-6-astra", Label: "GPT-6 Astra", SupportsTools: true, SupportsStreaming: true, ContextWindow: 272000},
	{Provider: "codex", Name: "gpt-6.1-terra", Label: "GPT-6.1 Terra", SupportsTools: true, SupportsStreaming: true, ContextWindow: 500000},
	{Provider: "codex", Name: "gpt-6-sol", Label: "GPT-6 Sol", SupportsTools: true, SupportsStreaming: true, ContextWindow: 272000},
	{Provider: "codex", Name: "gpt-6-luna", Label: "GPT-6 Luna", SupportsTools: true, SupportsStreaming: true, ContextWindow: 272000},
	{Provider: "codex", Name: "gpt-5.6-sol", Label: "GPT-5.6 Sol", SupportsTools: true, SupportsStreaming: true, ContextWindow: 272000},
	{Provider: "codex", Name: "gpt-5.6-terra", Label: "GPT-5.6 Terra", SupportsTools: true, SupportsStreaming: true, ContextWindow: 272000},
	{Provider: "codex", Name: "gpt-5.6-luna", Label: "GPT-5.6 Luna", SupportsTools: true, SupportsStreaming: true, ContextWindow: 272000},
}

// DefaultCount é quantos modelos padrão existem (para testes e logs).
func DefaultCount() int { return len(defaultModels) }

// InstallDefaults cadastra os modelos padrão que ainda não existem. Idempotente e conservador: um modelo
// que já está no catálogo (mesmo desativado ou editado) não é tocado nem recriado.
func InstallDefaults(ctx context.Context, db persistence.DBTX) (installed []string, err error) {
	for _, in := range defaultModels {
		in := in
		if _, err := Register(ctx, db, in); err != nil {
			if errors.Is(err, persistence.ErrDuplicate) {
				continue
			}
			return installed, err
		}
		installed = append(installed, in.Provider+"/"+in.Name)
	}
	return installed, nil
}
