package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Gateways de API suportados.
const (
	GatewayOpenRouter = "openrouter"
	GatewayVercel     = "vercel"
)

// ErrIncompatible: o modelo não pode ser usado nesta configuração.
var ErrIncompatible = errors.New("modelo incompatível")

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Params são os parâmetros de geração padrão do modelo (opcionais).
type Params struct {
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
}

// Model é um modelo do catálogo.
type Model struct {
	ID                string    `json:"id"`
	Provider          string    `json:"provider"`
	Gateway           string    `json:"gateway"`
	Name              string    `json:"name"`
	Label             string    `json:"label"`
	SupportsTools     bool      `json:"supports_tools"`
	SupportsStreaming bool      `json:"supports_streaming"`
	ContextWindow     int       `json:"context_window"`
	MaxOutputTokens   int       `json:"max_output_tokens"`
	Params            Params    `json:"params"`
	State             string    `json:"state"`
	Version           int       `json:"version"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Input é o cadastro de um modelo.
type Input struct {
	Provider          string `json:"provider"`
	Gateway           string `json:"gateway"`
	Name              string `json:"name"`
	Label             string `json:"label"`
	SupportsTools     bool   `json:"supports_tools"`
	SupportsStreaming bool   `json:"supports_streaming"`
	ContextWindow     int    `json:"context_window"`
	MaxOutputTokens   int    `json:"max_output_tokens"`
	Params            Params `json:"params"`
}

var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,79}$`)

// Validate normaliza e valida o cadastro, inclusive os parâmetros.
func (in *Input) Validate() error {
	in.Provider, in.Gateway, in.Name, in.Label = strings.TrimSpace(in.Provider), strings.TrimSpace(in.Gateway), strings.TrimSpace(in.Name), strings.TrimSpace(in.Label)
	switch in.Provider {
	case "claude", "codex":
		if in.Gateway != "" {
			return &FieldError{"gateway", "Modelos de CLI não usam gateway."}
		}
	case "api":
		if in.Gateway != GatewayOpenRouter && in.Gateway != GatewayVercel {
			return &FieldError{"gateway", "Informe o gateway: openrouter ou vercel."}
		}
	default:
		return &FieldError{"provider", "Provedor inválido (claude, codex ou api)."}
	}
	if !namePattern.MatchString(in.Name) {
		return &FieldError{"name", "Nome do modelo inválido."}
	}
	if len(in.Label) > 120 {
		return &FieldError{"label", "Rótulo longo demais (até 120)."}
	}
	if in.ContextWindow < 0 || in.ContextWindow > 10_000_000 {
		return &FieldError{"context_window", "Janela de contexto fora da faixa."}
	}
	if in.MaxOutputTokens < 0 || in.MaxOutputTokens > 1_000_000 {
		return &FieldError{"max_output_tokens", "Máximo de saída fora da faixa."}
	}
	if in.ContextWindow > 0 && in.MaxOutputTokens > in.ContextWindow {
		return &FieldError{"max_output_tokens", "A saída máxima não pode exceder a janela de contexto."}
	}
	if p := in.Params; p.Temperature != nil && (*p.Temperature < 0 || *p.Temperature > 2) {
		return &FieldError{"params.temperature", "Temperatura deve estar entre 0 e 2."}
	}
	if p := in.Params; p.TopP != nil && (*p.TopP < 0 || *p.TopP > 1) {
		return &FieldError{"params.top_p", "top_p deve estar entre 0 e 1."}
	}
	if p := in.Params; p.MaxTokens != nil {
		if *p.MaxTokens < 1 || (in.MaxOutputTokens > 0 && *p.MaxTokens > in.MaxOutputTokens) {
			return &FieldError{"params.max_tokens", "max_tokens deve ser positivo e caber na saída máxima do modelo."}
		}
	}
	return nil
}

const columns = `id, provider, gateway, name, label, supports_tools, supports_streaming, context_window, max_output_tokens, params, state, version, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Model, error) {
	var m Model
	var params []byte
	if err := row.Scan(&m.ID, &m.Provider, &m.Gateway, &m.Name, &m.Label, &m.SupportsTools, &m.SupportsStreaming,
		&m.ContextWindow, &m.MaxOutputTokens, &params, &m.State, &m.Version, &m.CreatedAt, &m.UpdatedAt); err != nil {
		return m, persistence.MapError(err)
	}
	_ = json.Unmarshal(params, &m.Params)
	return m, nil
}

// Register cadastra um modelo.
func Register(ctx context.Context, db persistence.DBTX, in Input) (Model, error) {
	if err := in.Validate(); err != nil {
		return Model{}, err
	}
	params, _ := json.Marshal(in.Params)
	return scan(db.QueryRow(ctx, `
		INSERT INTO models (provider, gateway, name, label, supports_tools, supports_streaming, context_window, max_output_tokens, params)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb) RETURNING `+columns,
		in.Provider, in.Gateway, in.Name, in.Label, in.SupportsTools, in.SupportsStreaming, in.ContextWindow, in.MaxOutputTokens, string(params)))
}

// List devolve o catálogo.
func List(ctx context.Context, db persistence.DBTX) ([]Model, error) {
	rows, err := db.Query(ctx, `SELECT `+columns+` FROM models ORDER BY provider, gateway, name`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Model{}
	for rows.Next() {
		m, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, persistence.MapError(rows.Err())
}

// Get lê um modelo.
func Get(ctx context.Context, db persistence.DBTX, id string) (Model, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM models WHERE id = $1`, id))
}

// SetState ativa ou desativa um modelo (controle otimista).
func SetState(ctx context.Context, db persistence.DBTX, id, state string, expectedVersion int) (Model, error) {
	if state != "active" && state != "disabled" {
		return Model{}, &FieldError{"state", "Estado inválido (active ou disabled)."}
	}
	m, err := scan(db.QueryRow(ctx, `
		UPDATE models SET state = $2, version = version + 1, updated_at = now()
		WHERE id = $1 AND version = $3 RETURNING `+columns, id, state, expectedVersion))
	if errors.Is(err, persistence.ErrNotFound) {
		if _, gerr := Get(ctx, db, id); gerr == nil {
			return Model{}, persistence.ErrVersionConflict
		}
	}
	return m, err
}

// Requirement é o que a configuração efetiva de um agente/nó exige do modelo.
type Requirement struct {
	Provider     string
	ConnectionID string // conta efetiva (nil/"" = sem conta)
	Model        string
	NeedsTools   bool
}

// Verdict é o resultado verificável da compatibilidade.
type Verdict struct {
	OK       bool     `json:"ok"`
	Model    *Model   `json:"model,omitempty"`
	Problems []string `json:"problems"`
}

// Check decide se o modelo pode ser usado. Nunca lança erro por incompatibilidade:
// devolve os motivos; use Verdict.Err() quando for preciso bloquear.
func Check(ctx context.Context, db persistence.DBTX, req Requirement) (Verdict, error) {
	v := Verdict{Problems: []string{}}
	add := func(s string) { v.Problems = append(v.Problems, s) }
	req.Model = strings.TrimSpace(req.Model)

	// CLI sem modelo explícito usa o padrão da própria conta; API exige modelo.
	if req.Model == "" {
		if req.Provider == "api" {
			add("O provedor api exige um modelo cadastrado.")
		}
		v.OK = len(v.Problems) == 0
		return v, nil
	}
	gateway := ""
	if req.Provider == "api" {
		if req.ConnectionID == "" {
			add("O provedor api exige uma conta (conexão do tipo api).")
		} else {
			var kind, state, target string
			err := db.QueryRow(ctx, `SELECT kind, state, target FROM connections WHERE id = $1`, req.ConnectionID).Scan(&kind, &state, &target)
			switch {
			case errors.Is(persistence.MapError(err), persistence.ErrNotFound):
				add("A conta não existe.")
			case err != nil:
				return v, persistence.MapError(err)
			case state != "active":
				add("A conta está revogada.")
			case kind != "api":
				add("A conta precisa ser do tipo api.")
			case target != GatewayOpenRouter && target != GatewayVercel:
				add(fmt.Sprintf("O destino da conta (%q) não é um gateway suportado (openrouter ou vercel).", target))
			default:
				gateway = target
			}
		}
	}
	var m Model
	var err error
	if req.Provider != "api" || gateway != "" {
		m, err = scan(db.QueryRow(ctx, `SELECT `+columns+` FROM models WHERE provider = $1 AND gateway = $2 AND name = $3`, req.Provider, gateway, req.Model))
		switch {
		case errors.Is(err, persistence.ErrNotFound):
			add(fmt.Sprintf("O modelo %q não está cadastrado para %s.", req.Model, describe(req.Provider, gateway)))
		case err != nil:
			return v, err
		default:
			v.Model = &m
			if m.State != "active" {
				add("O modelo está desativado.")
			}
			if req.NeedsTools && !m.SupportsTools {
				add("O modelo não suporta ferramentas, mas o agente tem ferramentas concedidas.")
			}
		}
	}
	v.OK = len(v.Problems) == 0
	return v, nil
}

func describe(provider, gateway string) string {
	if gateway != "" {
		return provider + "/" + gateway
	}
	return provider
}

// Err transforma um veredito negativo em erro (nil quando compatível).
func (v Verdict) Err() error {
	if v.OK {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrIncompatible, strings.Join(v.Problems, " "))
}
