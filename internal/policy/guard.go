package policy

import (
	"context"
	"errors"
	"fmt"
)

// Erros devolvidos quando a ação não pode seguir.
var (
	ErrForbidden        = errors.New("ação proibida pela política")
	ErrApprovalRequired = errors.New("a ação exige aprovação humana")
)

// Refusal acompanha a decisão que impediu a ação.
type Refusal struct {
	Err      error
	Decision Decision
}

func (r *Refusal) Error() string {
	return fmt.Sprintf("%v (%s): %s", r.Err, r.Decision.Rule, r.Decision.Reason)
}
func (r *Refusal) Unwrap() error { return r.Err }

// Recorder grava a trilha das decisões.
type Recorder interface {
	Record(ctx context.Context, a Action, d Decision) error
}

// Guard aplica a política antes de qualquer efeito: avalia, registra e só então
// executa. Se a trilha não puder ser gravada, a ação NÃO roda (falha fechada).
type Guard struct {
	Policy *Policy
	Rec    Recorder
}

// Do roda `run` só se a decisão for Allow. Para Approve devolve ErrApprovalRequired
// e para Deny, ErrForbidden, sem chamar `run`.
func (g *Guard) Do(ctx context.Context, a Action, run func() error) (Decision, error) {
	d := g.Policy.Evaluate(a)
	if g.Rec != nil {
		if err := g.Rec.Record(ctx, a, d); err != nil {
			return d, fmt.Errorf("trilha indisponível, ação não executada: %w", err)
		}
	}
	switch d.Effect {
	case Allow:
		return d, run()
	case Approve:
		return d, &Refusal{Err: ErrApprovalRequired, Decision: d}
	}
	return d, &Refusal{Err: ErrForbidden, Decision: d}
}
