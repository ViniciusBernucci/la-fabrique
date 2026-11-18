package factory

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/runner"
	"github.com/ViniciusBernucci/le-fabrique/internal/skills"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// skill executa o nó Skill: passos de IA rodam no terminal do agente (o CLI da conta) e scripts, num sandbox sem
// rede, sempre pela política. As variáveis que a skill grava viram as saídas do nó.
func (e *Executor) skill(ctx context.Context, info runner.Info, n workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	name := propStr(n, "skill")
	sk, err := skills.GetByName(ctx, e.DB, name)
	if err != nil {
		return fail(info, n, "a skill %q não está no catálogo", name), nil
	}
	repo, _, err := e.ensureRepo(ctx, info.Run)
	if err != nil {
		return fail(info, n, "%v", err), nil
	}
	in := map[string]any{}
	for k, v := range vars {
		in[k] = v
	}
	if input := propStr(n, "input"); input != "" {
		in["input"] = input
	}
	var out skills.Result
	var runErr error
	err = e.withSandbox(ctx, info.Run, "base", nil, repo, func(sb *execution.Sandbox) error {
		env := skills.Env{Sandbox: sb, Guard: &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{})},
			AI: func(ctx context.Context, instructions, input string) (string, error) {
				eff := agents.Effective{AgentName: sk.Name, Provider: "claude", Instructions: instructions, Prompt: input, TimeoutMinutes: 30}
				res, err := e.runAgent(ctx, info, n, eff)
				if err != nil {
					return "", err
				}
				if res.Port == workflows.PortError {
					return "", errors.New(fmt.Sprint(res.Outputs["erro"]))
				}
				text, _ := res.Outputs["resumo"].(string)
				return text, nil
			}}
		skCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		defer cancel()
		out, runErr = skills.Run(skCtx, env, sk, in)
		if runErr == nil {
			_, _ = e.exportPatch(context.WithoutCancel(ctx), sb, repo)
		}
		return nil
	})
	if err != nil {
		if ctx.Err() != nil {
			return workflows.ExecResult{}, ctx.Err()
		}
		return fail(info, n, "%v", err), nil
	}
	if runErr != nil {
		return fail(info, n, "skill %s: %v", name, runErr), nil
	}
	outputs := map[string]any{"ok": true}
	for k, v := range out.Vars {
		if _, reserved := in[k]; !reserved || k == "text" || k == "stdout" || k == "exit_code" {
			outputs[k] = v
		}
	}
	for _, st := range out.Steps {
		info.Emit(runner.EventNode, map[string]any{"node": n.ID, "kind": "command", "text": st.Type + ": " + st.Detail})
	}
	return workflows.ExecResult{Outputs: outputs}, nil
}
