package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Origem de cada valor da configuração efetiva.
const (
	FromAgent = "agent" // veio da base do catálogo
	FromNode  = "node"  // sobrescrito pelo nó do workflow
)

// ErrOverride indica sobrescrita inválida (ex.: ampliar ferramentas).
var ErrOverride = errors.New("sobrescrita inválida")

// ErrAgentNotFound: o nó aponta para um agente que não está no catálogo.
var ErrAgentNotFound = errors.New("agente não encontrado no catálogo")

// NodeOverrides são as sobrescritas declaradas no nó (campos opcionais).
type NodeOverrides struct {
	Model             string
	ConnectionID      string
	ExtraInstructions string
	Tools             []string // nil = sem sobrescrita
	TimeoutMinutes    int      // 0 = sem sobrescrita
}

// OverridesOf lê as sobrescritas das propriedades do nó.
func OverridesOf(n workflows.Node) NodeOverrides {
	str := func(k string) string { s, _ := n.Props[k].(string); return strings.TrimSpace(s) }
	o := NodeOverrides{Model: str("model"), ConnectionID: str("connection"), ExtraInstructions: str("instructions_extra")}
	if t := str("tools"); t != "" {
		o.Tools = []string{}
		for _, p := range strings.Split(t, ",") {
			if p = strings.TrimSpace(p); p != "" {
				o.Tools = append(o.Tools, p)
			}
		}
	}
	if v, ok := n.Props["timeout_minutes"].(float64); ok {
		o.TimeoutMinutes = int(v)
	}
	return o
}

// Effective é a configuração efetiva de um nó: sem segredos (a conta aparece só
// como referência de conexão), com a origem de cada valor.
type Effective struct {
	AgentID        string            `json:"agent_id"`
	AgentName      string            `json:"agent_name"`
	AgentVersion   int               `json:"agent_version"`
	Provider       string            `json:"provider"`
	ConnectionID   *string           `json:"connection_id"`
	Model          string            `json:"model"`
	Instructions   string            `json:"instructions"`
	Prompt         string            `json:"prompt"`
	Skills         []string          `json:"skills"`
	Tools          []string          `json:"tools"`
	Fallback       []FallbackStep    `json:"fallback"`
	TimeoutMinutes int               `json:"timeout_minutes"`
	Sources        map[string]string `json:"sources"`
}

// Resolve combina a base com as sobrescritas do nó. Não altera `base`.
// Ferramentas só podem ser REDUZIDAS (subconjunto da base): o nó não amplia privilégio.
func Resolve(base Agent, n workflows.Node) (Effective, error) {
	o := OverridesOf(n)
	e := Effective{
		AgentID: base.ID, AgentName: base.Name, AgentVersion: base.Version, Provider: base.Provider,
		ConnectionID: base.ConnectionID, Model: base.Model, Instructions: base.Instructions,
		Skills: append([]string{}, base.Skills...), Tools: append([]string{}, base.Tools...),
		Fallback: append([]FallbackStep{}, base.Fallback...), TimeoutMinutes: base.TimeoutMinutes,
		Sources: map[string]string{"provider": FromAgent, "connection": FromAgent, "model": FromAgent,
			"instructions": FromAgent, "skills": FromAgent, "tools": FromAgent, "timeout_minutes": FromAgent},
	}
	if p, _ := n.Props["prompt"].(string); strings.TrimSpace(p) != "" {
		e.Prompt = strings.TrimSpace(p)
	}
	if o.Model != "" {
		if !modelPattern.MatchString(o.Model) {
			return Effective{}, fmt.Errorf("%w: modelo inválido no nó", ErrOverride)
		}
		e.Model, e.Sources["model"] = o.Model, FromNode
	}
	if o.ConnectionID != "" {
		if !uuidPattern.MatchString(o.ConnectionID) {
			return Effective{}, fmt.Errorf("%w: conexão inválida no nó", ErrOverride)
		}
		c := o.ConnectionID
		e.ConnectionID, e.Sources["connection"] = &c, FromNode
	}
	if o.ExtraInstructions != "" {
		e.Instructions += "\n\n" + o.ExtraInstructions
		e.Sources["instructions"] = FromNode
	}
	if o.Tools != nil {
		allowed := map[string]bool{}
		for _, t := range base.Tools {
			allowed[t] = true
		}
		for _, t := range o.Tools {
			if !allowed[t] {
				return Effective{}, fmt.Errorf("%w: o nó não pode ampliar as ferramentas do agente (%q)", ErrOverride, t)
			}
		}
		e.Tools, e.Sources["tools"] = o.Tools, FromNode
	}
	if o.TimeoutMinutes > 0 {
		e.TimeoutMinutes, e.Sources["timeout_minutes"] = o.TimeoutMinutes, FromNode
	}
	return e, nil
}

// EffectiveForNode carrega o agente do nó pelo nome, resolve as sobrescritas e
// confere a conexão efetiva contra o provedor.
func EffectiveForNode(ctx context.Context, db persistence.DBTX, n workflows.Node) (Effective, error) {
	if n.Type != workflows.TAgent {
		return Effective{}, fmt.Errorf("%w: o nó %q não é um agente", ErrOverride, n.ID)
	}
	name, _ := n.Props["agent"].(string)
	base, err := GetByName(ctx, db, strings.TrimSpace(name))
	if errors.Is(err, persistence.ErrNotFound) {
		return Effective{}, fmt.Errorf("%w: %q", ErrAgentNotFound, name)
	}
	if err != nil {
		return Effective{}, err
	}
	e, err := Resolve(base, n)
	if err != nil {
		return Effective{}, err
	}
	if e.ConnectionID != nil {
		if err := CheckConnection(ctx, db, e.Provider, *e.ConnectionID); err != nil {
			return Effective{}, err
		}
	}
	return e, nil
}
