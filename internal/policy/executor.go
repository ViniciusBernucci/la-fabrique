package policy

import (
	"context"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// GuardedExecutor protege um workflows.Executor: cada nó de trabalho vira uma
// Action avaliada pela política antes de executar. Ação sensível devolve
// ErrApprovalRequired e proibida, ErrForbidden; em ambos o motor segue a rota de
// erro do nó. (O pedido de aprovação na Central e a retomada são do T13-02.)
type GuardedExecutor struct {
	Inner   workflows.Executor
	Guard   *Guard
	Project string
}

// ActionFor traduz um nó de trabalho na ação que a política avalia.
func ActionFor(project string, n workflows.Node, vars map[string]any) Action {
	a := Action{Project: project, Params: map[string]any{}}
	str := func(k string) string { s, _ := n.Props[k].(string); return strings.TrimSpace(s) }
	switch n.Type {
	case workflows.TScript:
		a.Tool, a.Operation = ToolScript, "run"
		a.Params["command"] = str("command")
	case workflows.TChecks:
		a.Tool, a.Operation = ToolScript, "run"
		a.Params["command"] = str("commands") // uma linha por comando
	case workflows.TGit:
		a.Tool, a.Operation = ToolGit, str("operation")
		if b, ok := vars["branch"].(string); ok {
			a.Params["branch"] = b
		}
	default: // agente, skill e documentação rodam por clientes de IA no modo de escrita do workspace
		a.Tool, a.Operation = ToolCLI, "ai_client.run"
		a.Params["mode"] = "workspace_write"
	}
	a.Resource = n.ID
	return a
}

// Run implementa workflows.Executor.
func (g GuardedExecutor) Run(ctx context.Context, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	var res workflows.ExecResult
	_, err := g.Guard.Do(ctx, ActionFor(g.Project, n, vars), func() error {
		var rerr error
		res, rerr = g.Inner.Run(ctx, n, vars)
		return rerr
	})
	return res, err
}
