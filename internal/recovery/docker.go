package recovery

import (
	"context"
	"errors"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
)

// DockerInspector inspeciona o ambiente real da tentativa interrompida pelo Docker. Falha fechada:
// sem runtime que responda, a inspeção devolve erro e a recuperação ADIA (nada muda).
type DockerInspector struct {
	Docker *execution.Docker
	Opt    checkpoints.Options
}

// ErrRuntimeUnavailable: o runtime de containers não respondeu.
var ErrRuntimeUnavailable = errors.New("o runtime de containers não está disponível para inspecionar o sandbox")

// Quiescent: sem container vivo da feature → quieto; com container, quieto só se nenhum
// processo do projeto estiver vivo nele.
func (d DockerInspector) Quiescent(ctx context.Context, it queue.Item) (bool, error) {
	if d.Docker == nil || !d.Docker.Available(ctx) {
		return false, ErrRuntimeUnavailable
	}
	names, err := d.Docker.ManagedContainers(ctx, it.ProjectID, it.FeatureID)
	if err != nil {
		return false, err
	}
	for _, n := range names {
		quiet, err := d.Docker.Attach(n).VerifyQuiescent(ctx)
		if err != nil {
			return false, err
		}
		if !quiet {
			return false, nil
		}
	}
	return true, nil
}

// Verify compara o workspace vivo (se existir) com o checkpoint. Sem container, não há o que
// comparar: a próxima execução restaura do checkpoint. Sem checkpoint, não há referência.
func (d DockerInspector) Verify(ctx context.Context, it queue.Item, cp *checkpoints.Checkpoint) ([]string, error) {
	if cp == nil {
		return nil, nil
	}
	if d.Docker == nil || !d.Docker.Available(ctx) {
		return nil, ErrRuntimeUnavailable
	}
	names, err := d.Docker.ManagedContainers(ctx, it.ProjectID, it.FeatureID)
	if err != nil {
		return nil, err
	}
	var problems []string
	for _, n := range names {
		if err := checkpoints.Verify(ctx, *cp, checkpoints.SandboxWorkspace{S: d.Docker.Attach(n)}, d.Opt); err != nil {
			problems = append(problems, fmt.Sprintf("o workspace de %s diverge do checkpoint: %v", n, err))
		}
	}
	return problems, nil
}
