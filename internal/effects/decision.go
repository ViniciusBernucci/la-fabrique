package effects

import (
	"context"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// NodePrefix marca, no pedido de decisão, qual efeito do ledger ele destrava.
const NodePrefix = "effect:"

// DecisionEscalator devolve um Escalate que abre, na Central de Decisões, um
// pedido de "resultado incerto" com a evidência da verificação. A execução
// precisa estar aguardando (waiting); quem chama é responsável por isso.
func DecisionEscalator(db decisions.DB, rd *security.Redactor, executionID string) func(context.Context, persistence.OutboxEntry, string) error {
	return func(ctx context.Context, e persistence.OutboxEntry, evidence string) error {
		_, _, err := decisions.Create(ctx, db, rd, decisions.NewRequest{
			ProjectID: e.ProjectID, ExecutionID: executionID, NodeID: NodePrefix + e.ID, Kind: decisions.KindUncertainResult,
			Title:   "Efeito externo inconclusivo",
			Problem: fmt.Sprintf("A operação %q foi enviada, mas não foi possível confirmar nem descartar que ela aconteceu no destino. Para não duplicar, a repetição automática está bloqueada.", e.Destination),
			Impact:  "A execução fica parada até você dizer se o efeito existe no destino.",
			Evidence: []decisions.Evidence{
				{Label: "Operação", Text: e.Destination + " · chave " + e.BusinessKey},
				{Label: "Tentativas de envio", Text: fmt.Sprint(e.Attempts)},
				{Label: "Resultado da verificação", Text: evidence},
			},
			Options: []decisions.Option{
				{ID: ResolveApplied, Label: "O efeito existe no destino", Description: "Confirma sem reenviar."},
				{ID: ResolveNotApplied, Label: "O efeito não existe", Description: "Libera a repetição da operação."},
			},
		})
		return err
	}
}

// ResolveFromDecision aplica ao ledger a decisão humana de um pedido de efeito
// inconclusivo. Pedidos que não são de efeito são ignorados (applied=false).
// Repetir a chamada com a mesma decisão é inofensivo.
func (l *Ledger) ResolveFromDecision(ctx context.Context, d decisions.Request) (applied bool, err error) {
	if d.Kind != decisions.KindUncertainResult || !strings.HasPrefix(d.NodeID, NodePrefix) || d.State != decisions.StateResolved {
		return false, nil
	}
	id := strings.TrimPrefix(d.NodeID, NodePrefix)
	e, gerr := persistence.GetOutbox(ctx, l.DB, d.ProjectID, id)
	if gerr != nil {
		return false, gerr
	}
	if e.State != persistence.OutboxUncertain { // já resolvido antes
		return false, nil
	}
	if _, err := l.Resolve(ctx, d.ProjectID, id, d.ResolutionOption, d.ResolutionNote); err != nil {
		return false, err
	}
	return true, nil
}
