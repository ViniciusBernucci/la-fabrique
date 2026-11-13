package queue

import (
	"context"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// DecisionGate libera quando a execução do item não tem pedido de decisão pendente.
type DecisionGate struct{ DB persistence.DBTX }

// Ready implementa Gate.
func (g DecisionGate) Ready(ctx context.Context, it Item) (bool, string, error) {
	if it.ExecutionID == "" {
		return false, "o item não tem execução associada", nil
	}
	var n int
	if err := g.DB.QueryRow(ctx, `SELECT count(*) FROM decision_requests WHERE execution_id = $1 AND state = 'pending'`, it.ExecutionID).Scan(&n); err != nil {
		return false, "", persistence.MapError(err)
	}
	if n > 0 {
		return false, fmt.Sprintf("%d decisão(ões) pendente(s)", n), nil
	}
	return true, "", nil
}

// MergeGate libera quando a PR de `wait_ref` está integrada (merge confirmado com SHA).
type MergeGate struct{ Merges tasksync.MergeVerifier }

// Ready implementa Gate.
func (g MergeGate) Ready(ctx context.Context, it Item) (bool, string, error) {
	if g.Merges == nil || it.WaitRef == "" {
		return false, "sem verificador de merge ou sem PR informada", nil
	}
	st, err := g.Merges.Merged(ctx, it.WaitRef)
	if err != nil {
		return false, "", err
	}
	switch {
	case st.Closed && !st.Merged:
		return false, "a PR foi fechada sem merge", nil
	case !st.Merged || st.SHA == "":
		return false, "a PR ainda não foi integrada", nil
	}
	return true, "", nil
}
