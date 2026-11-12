package gateways

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
)

// Limites do laço.
const (
	DefaultMaxTurns     = 8
	DefaultMaxToolCalls = 24
	maxToolResultBytes  = 16 << 10
	maxFinalText        = 64 << 10
)

// Erros do laço.
var (
	ErrModelNotEligible = errors.New("modelo não elegível para esta execução")
	ErrTooManyTurns     = errors.New("o modelo excedeu o limite de rodadas")
	ErrTooManyCalls     = errors.New("o modelo excedeu o limite de chamadas de ferramenta")
)

// Executor executa uma ferramenta JÁ autorizada pelo catálogo. É a única porta
// de efeito do modelo.
type Executor interface {
	Call(ctx context.Context, name string, args map[string]any) (map[string]any, error)
}

// RegistryExecutor liga o laço ao catálogo de ferramentas (concessão, escopo,
// validação e política são do Registry/Guard).
type RegistryExecutor struct {
	Registry *tools.Registry
	Env      tools.Env
	Grant    tools.Grant
	Identity tools.Identity // quem faz a chamada (obrigatória nas integrações)
}

// Call implementa Executor.
func (e RegistryExecutor) Call(ctx context.Context, name string, args map[string]any) (map[string]any, error) {
	if e.Identity != (tools.Identity{}) {
		ctx = tools.WithIdentity(ctx, e.Identity)
	}
	res, err := e.Registry.Call(ctx, e.Env, name, args, e.Grant)
	return res.Output, err
}

// SpecOf descreve uma ferramenta do catálogo como esquema para o modelo.
func SpecOf(t tools.Tool) ToolSpec {
	props, required := map[string]any{}, []any{}
	for _, f := range t.Input {
		p := map[string]any{"description": f.Description}
		switch f.Type {
		case tools.TypeInt:
			p["type"] = "integer"
			if f.Min != 0 {
				p["minimum"] = f.Min
			}
			if f.Max != 0 {
				p["maximum"] = f.Max
			}
		case tools.TypeBool:
			p["type"] = "boolean"
		case tools.TypeEnum:
			p["type"] = "string"
			p["enum"] = f.Options
		default:
			p["type"] = "string"
			if f.Max > 0 {
				p["maxLength"] = f.Max
			}
		}
		props[f.Name] = p
		if f.Required {
			required = append(required, f.Name)
		}
	}
	return ToolSpec{Name: t.Name, Description: t.Description, Parameters: map[string]any{
		"type": "object", "properties": props, "required": required, "additionalProperties": false,
	}}
}

// CallRecord registra o que aconteceu com cada pedido do modelo.
type CallRecord struct {
	Name    string `json:"name"`
	Outcome string `json:"outcome"` // ok | error | refused
}

// RunConfig configura uma execução de agente por gateway.
type RunConfig struct {
	Model        models.Model
	System       string
	Prompt       string
	Tools        []ToolSpec // as ferramentas que o agente TEM (nada além)
	Exec         Executor
	MaxTurns     int
	MaxToolCalls int
	Redact       func(string) string // aplicado ao texto de erro devolvido ao modelo
}

// Result é o resultado da execução.
type Result struct {
	Text  string       `json:"text"`
	Turns int          `json:"turns"`
	Calls []CallRecord `json:"calls"`
	Usage Usage        `json:"usage"`
}

// Eligible confere, antes de qualquer pedido ao gateway, se o modelo pode rodar
// esta configuração (defesa em profundidade além de models.Check).
func Eligible(m models.Model, needsTools bool) error {
	switch {
	case m.Provider != "api":
		return fmt.Errorf("%w: não é um modelo de gateway de API", ErrModelNotEligible)
	case m.State != "active":
		return fmt.Errorf("%w: modelo desativado", ErrModelNotEligible)
	case needsTools && !m.SupportsTools:
		return fmt.Errorf("%w: o modelo não suporta ferramentas", ErrModelNotEligible)
	}
	return nil
}

func clipText(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

// Run conduz o laço: o modelo responde ou pede ferramentas; cada pedido só
// executa se a ferramenta é uma das que o agente tem e o Executor autoriza.
func Run(ctx context.Context, gw Chatter, cfg RunConfig) (Result, error) {
	if err := Eligible(cfg.Model, len(cfg.Tools) > 0); err != nil {
		return Result{}, err
	}
	if len(cfg.Tools) > 0 && cfg.Exec == nil {
		return Result{}, fmt.Errorf("%w: executor ausente", ErrModelNotEligible)
	}
	maxTurns, maxCalls := cfg.MaxTurns, cfg.MaxToolCalls
	if maxTurns <= 0 {
		maxTurns = DefaultMaxTurns
	}
	if maxCalls <= 0 {
		maxCalls = DefaultMaxToolCalls
	}
	redact := cfg.Redact
	if redact == nil {
		redact = func(s string) string { return s }
	}
	have := map[string]bool{}
	for _, t := range cfg.Tools {
		have[t.Name] = true
	}
	msgs := []Message{}
	if strings.TrimSpace(cfg.System) != "" {
		msgs = append(msgs, Message{Role: "system", Content: cfg.System})
	}
	msgs = append(msgs, Message{Role: "user", Content: cfg.Prompt})

	var res Result
	for turn := 1; turn <= maxTurns; turn++ {
		resp, err := gw.Chat(ctx, ChatRequest{Model: cfg.Model.Name, Messages: msgs, Tools: cfg.Tools, Params: cfg.Model.Params})
		if err != nil {
			return res, err
		}
		res.Turns = turn
		res.Usage.PromptTokens += resp.Usage.PromptTokens
		res.Usage.CompletionTokens += resp.Usage.CompletionTokens
		if len(resp.ToolCalls) == 0 {
			res.Text = clipText(resp.Text, maxFinalText)
			return res, nil
		}
		msgs = append(msgs, Message{Role: "assistant", Content: resp.Text, ToolCalls: resp.ToolCalls})
		for _, tc := range resp.ToolCalls {
			if len(res.Calls) >= maxCalls {
				return res, ErrTooManyCalls
			}
			content, outcome := "", "ok"
			switch {
			case !have[tc.Name]:
				outcome, content = "refused", "Ferramenta indisponível para este agente."
			case tc.BadArgs:
				outcome, content = "refused", "Argumentos inválidos: envie um objeto JSON."
			default:
				out, err := cfg.Exec.Call(ctx, tc.Name, tc.Args)
				if err != nil {
					outcome, content = "error", "A ferramenta recusou ou falhou: "+clipText(redact(err.Error()), 300)
				} else {
					b, _ := json.Marshal(out)
					content = clipText(redact(string(b)), maxToolResultBytes)
				}
			}
			res.Calls = append(res.Calls, CallRecord{Name: clipText(tc.Name, 64), Outcome: outcome})
			msgs = append(msgs, Message{Role: "tool", ToolCallID: tc.ID, Content: content})
		}
	}
	return res, ErrTooManyTurns
}
