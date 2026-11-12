package skills

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// ErrAssertion: uma asserção determinística falhou.
var ErrAssertion = errors.New("asserção falhou")

// ErrNotConfigured: falta o executor necessário (IA ou sandbox).
var ErrNotConfigured = errors.New("executor não configurado")

// AIRunner roda as instruções de uma skill de IA (um agente, por exemplo) e
// devolve o texto final.
type AIRunner func(ctx context.Context, instructions, input string) (string, error)

// Env reúne o que a execução precisa. Scripts SEMPRE passam pela política (Guard).
type Env struct {
	Sandbox *execution.Sandbox
	Guard   *policy.Guard
	AI      AIRunner
}

// StepResult resume um passo.
type StepResult struct {
	Type   string `json:"type"`
	Detail string `json:"detail"`
}

// Result é o resultado da execução.
type Result struct {
	Vars  map[string]any `json:"vars"`
	Steps []StepResult   `json:"steps"`
}

// steps traduz o executor da skill numa lista de passos.
func steps(sk Skill) []Step {
	switch sk.Executor {
	case ExecAI:
		return []Step{{Type: StepAI}}
	case ExecScript:
		out := make([]Step, 0, len(sk.Manifest.Commands))
		for _, c := range sk.Manifest.Commands {
			out = append(out, Step{Type: StepScript, Command: c})
		}
		return out
	}
	return sk.Manifest.Steps
}

// Run executa a skill passo a passo. Variáveis fluem entre os passos (`input`,
// `text` com a saída da IA, `exit_code`, `stdout` do último script e as definidas
// por passos determinísticos).
func Run(ctx context.Context, env Env, sk Skill, vars map[string]any) (Result, error) {
	res := Result{Vars: map[string]any{}}
	for k, v := range vars {
		res.Vars[k] = v
	}
	for i, st := range steps(sk) {
		label := fmt.Sprintf("passo %d", i+1)
		switch st.Type {
		case StepAI:
			if env.AI == nil {
				return res, fmt.Errorf("%w: IA (%s)", ErrNotConfigured, label)
			}
			in, _ := res.Vars["input"].(string)
			text, err := env.AI(ctx, sk.Body, in)
			if err != nil {
				return res, fmt.Errorf("%s (IA): %w", label, err)
			}
			res.Vars["text"] = text
			res.Steps = append(res.Steps, StepResult{StepAI, "ok"})
		case StepScript:
			if env.Sandbox == nil || env.Guard == nil {
				return res, fmt.Errorf("%w: sandbox e política (%s)", ErrNotConfigured, label)
			}
			var out execution.ExecResult
			_, err := env.Guard.Do(ctx, policy.Action{Tool: policy.ToolScript, Operation: "run", Resource: sk.Name,
				Params: map[string]any{"command": st.Command}}, func() error {
				var rerr error
				out, rerr = env.Sandbox.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", st.Command}, Timeout: 10 * time.Minute})
				return rerr
			})
			if err != nil {
				return res, fmt.Errorf("%s (script): %w", label, err)
			}
			res.Vars["exit_code"], res.Vars["stdout"] = float64(out.ExitCode), out.Stdout
			res.Steps = append(res.Steps, StepResult{StepScript, fmt.Sprintf("saída %d", out.ExitCode)})
			if out.ExitCode != 0 {
				return res, fmt.Errorf("%s (script): saiu com código %d", label, out.ExitCode)
			}
		case StepDeterministic:
			if err := deterministic(st, res.Vars); err != nil {
				return res, fmt.Errorf("%s: %w", label, err)
			}
			res.Steps = append(res.Steps, StepResult{StepDeterministic, st.Op})
		default:
			return res, fmt.Errorf("%s: tipo desconhecido", label)
		}
	}
	return res, nil
}

func deterministic(st Step, vars map[string]any) error {
	switch st.Op {
	case OpSet:
		vars[st.Key] = st.Value
	case OpTemplate:
		var missing string
		out := placeholders.ReplaceAllStringFunc(st.Value, func(m string) string {
			name := placeholders.FindStringSubmatch(m)[1]
			var cur any = vars
			for _, p := range strings.Split(name, ".") {
				mm, ok := cur.(map[string]any)
				if !ok {
					missing = name
					return ""
				}
				cur = mm[p]
			}
			if cur == nil {
				missing = name
				return ""
			}
			return fmt.Sprint(cur)
		})
		if missing != "" {
			return fmt.Errorf("variável %q ausente no modelo", missing)
		}
		vars[st.Key] = out
	case OpAssert:
		expr, err := workflows.ParseExpr(st.Expression)
		if err != nil {
			return err
		}
		ok, err := expr.Eval(vars)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: %s", ErrAssertion, st.Expression)
		}
	}
	return nil
}

// Loader localiza uma skill pelo nome.
type Loader func(ctx context.Context, name string) (Skill, error)

// NodeExecutor roda nós `skill` do motor de workflows (e delega os demais tipos).
// É o uso "independente do agente": o nó executa a skill sozinho.
type NodeExecutor struct {
	Load  Loader
	Env   Env
	Other workflows.Executor
}

// Run implementa workflows.Executor.
func (n NodeExecutor) Run(ctx context.Context, node workflows.Node, vars map[string]any) (workflows.ExecResult, error) {
	if node.Type != workflows.TSkill {
		if n.Other == nil {
			return workflows.ExecResult{}, errors.New("sem executor para este tipo de nó")
		}
		return n.Other.Run(ctx, node, vars)
	}
	name, _ := node.Props["skill"].(string)
	sk, err := n.Load(ctx, strings.TrimSpace(name))
	if err != nil {
		return workflows.ExecResult{}, err
	}
	in := map[string]any{}
	for k, v := range vars {
		in[k] = v
	}
	if input, _ := node.Props["input"].(string); strings.TrimSpace(input) != "" {
		in["input"] = strings.TrimSpace(input)
	}
	res, err := Run(ctx, n.Env, sk, in)
	if err != nil {
		return workflows.ExecResult{}, err
	}
	out := map[string]any{}
	for k, v := range res.Vars {
		out[k] = v
	}
	return workflows.ExecResult{Outputs: out}, nil
}
