package agents

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

// Provedores de agente.
const (
	ProviderClaude = "claude"
	ProviderCodex  = "codex"
	ProviderAPI    = "api"
)

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// FallbackStep é uma alternativa da cadeia de fallback (usada pelo T09-04).
type FallbackStep struct {
	ConnectionID string `json:"connection_id"`
	Model        string `json:"model"`
}

// Agent é um agente do catálogo.
type Agent struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Instructions   string         `json:"instructions"`
	Provider       string         `json:"provider"`
	ConnectionID   *string        `json:"connection_id"`
	Model          string         `json:"model"`
	Skills         []string       `json:"skills"`
	Tools          []string       `json:"tools"`
	Fallback       []FallbackStep `json:"fallback"`
	TimeoutMinutes int            `json:"timeout_minutes"`
	Version        int            `json:"version"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// Input reúne os campos editáveis.
type Input struct {
	Name           string         `json:"name"`
	Description    string         `json:"description"`
	Instructions   string         `json:"instructions"`
	Provider       string         `json:"provider"`
	ConnectionID   *string        `json:"connection_id"`
	Model          string         `json:"model"`
	Skills         []string       `json:"skills"`
	Tools          []string       `json:"tools"`
	Fallback       []FallbackStep `json:"fallback"`
	TimeoutMinutes int            `json:"timeout_minutes"`
}

var (
	namePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	itemPattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,79}$`)
	modelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,79}$`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// ErrConnection indica conexão inexistente, revogada ou incompatível com o provedor.
var ErrConnection = errors.New("conexão incompatível")

func cleanList(field string, in []string, max int) ([]string, error) {
	out := []string{}
	seen := map[string]bool{}
	for _, v := range in {
		v = strings.TrimSpace(v)
		if !itemPattern.MatchString(v) {
			return nil, &FieldError{field, "Nome inválido: use letras, números, ponto, dois-pontos, hífen e sublinhado."}
		}
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	if len(out) > max {
		return nil, &FieldError{field, fmt.Sprintf("No máximo %d itens.", max)}
	}
	return out, nil
}

// normalize valida e limpa os campos.
func (in *Input) normalize() error {
	in.Name, in.Description, in.Instructions, in.Model = strings.TrimSpace(in.Name), strings.TrimSpace(in.Description), strings.TrimSpace(in.Instructions), strings.TrimSpace(in.Model)
	if !namePattern.MatchString(in.Name) {
		return &FieldError{"name", "O nome deve usar minúsculas, números, hífen e sublinhado (até 64)."}
	}
	if len(in.Description) > 500 {
		return &FieldError{"description", "A descrição deve ter até 500 caracteres."}
	}
	if in.Instructions == "" || len(in.Instructions) > 20000 {
		return &FieldError{"instructions", "As instruções devem ter entre 1 e 20000 caracteres."}
	}
	switch in.Provider {
	case ProviderClaude, ProviderCodex, ProviderAPI:
	default:
		return &FieldError{"provider", "O provedor deve ser claude, codex ou api."}
	}
	if in.Model != "" && !modelPattern.MatchString(in.Model) {
		return &FieldError{"model", "Modelo inválido."}
	}
	if in.ConnectionID != nil {
		c := strings.TrimSpace(*in.ConnectionID)
		if c == "" {
			in.ConnectionID = nil
		} else if !uuidPattern.MatchString(c) {
			return &FieldError{"connection_id", "A conexão deve ser identificada por um UUID."}
		} else {
			in.ConnectionID = &c
		}
	}
	var err error
	if in.Skills, err = cleanList("skills", in.Skills, 50); err != nil {
		return err
	}
	if in.Tools, err = cleanList("tools", in.Tools, 50); err != nil {
		return err
	}
	if in.TimeoutMinutes == 0 {
		in.TimeoutMinutes = 30
	}
	if in.TimeoutMinutes < 1 || in.TimeoutMinutes > 720 {
		return &FieldError{"timeout_minutes", "O tempo máximo deve estar entre 1 e 720 minutos."}
	}
	if len(in.Fallback) > 5 {
		return &FieldError{"fallback", "No máximo 5 alternativas."}
	}
	if in.Fallback == nil {
		in.Fallback = []FallbackStep{}
	}
	for i, f := range in.Fallback {
		if !uuidPattern.MatchString(f.ConnectionID) || (f.Model != "" && !modelPattern.MatchString(f.Model)) {
			return &FieldError{"fallback", fmt.Sprintf("A alternativa %d é inválida (conexão UUID e modelo válidos).", i+1)}
		}
	}
	return nil
}

// CheckConnection confere se a conexão existe, está ativa e serve ao provedor:
// claude/codex exigem uma conexão `cli` cujo destino é o próprio cliente; api, uma conexão `api`.
func CheckConnection(ctx context.Context, db persistence.DBTX, provider, connectionID string) error {
	var kind, state, target string
	err := db.QueryRow(ctx, `SELECT kind, state, target FROM connections WHERE id = $1`, connectionID).Scan(&kind, &state, &target)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return fmt.Errorf("%w: a conexão não existe", ErrConnection)
	}
	if err != nil {
		return persistence.MapError(err)
	}
	switch {
	case state != "active":
		return fmt.Errorf("%w: a conexão está revogada", ErrConnection)
	case provider == ProviderAPI && kind != "api":
		return fmt.Errorf("%w: o provedor api exige uma conexão do tipo api", ErrConnection)
	case (provider == ProviderClaude || provider == ProviderCodex) && (kind != "cli" || target != provider):
		return fmt.Errorf("%w: o provedor %s exige uma conexão cli com destino %q", ErrConnection, provider, provider)
	}
	return nil
}

const columns = `id, name, description, instructions, provider, connection_id, model, skills, tools, fallback, timeout_minutes, version, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Agent, error) {
	var a Agent
	var skills, tools, fallback []byte
	if err := row.Scan(&a.ID, &a.Name, &a.Description, &a.Instructions, &a.Provider, &a.ConnectionID, &a.Model,
		&skills, &tools, &fallback, &a.TimeoutMinutes, &a.Version, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return a, persistence.MapError(err)
	}
	a.Skills, a.Tools, a.Fallback = []string{}, []string{}, []FallbackStep{}
	_ = json.Unmarshal(skills, &a.Skills)
	_ = json.Unmarshal(tools, &a.Tools)
	_ = json.Unmarshal(fallback, &a.Fallback)
	return a, nil
}

func prepare(ctx context.Context, db persistence.DBTX, in *Input) (skills, tools, fallback []byte, err error) {
	if err = in.normalize(); err != nil {
		return
	}
	if in.ConnectionID != nil {
		if err = CheckConnection(ctx, db, in.Provider, *in.ConnectionID); err != nil {
			return
		}
	}
	for _, f := range in.Fallback {
		if err = CheckConnection(ctx, db, in.Provider, f.ConnectionID); err != nil {
			return
		}
	}
	skills, _ = json.Marshal(in.Skills)
	tools, _ = json.Marshal(in.Tools)
	fallback, _ = json.Marshal(in.Fallback)
	return
}

// Create cadastra um agente.
func Create(ctx context.Context, db persistence.DBTX, in Input) (Agent, error) {
	skills, tools, fallback, err := prepare(ctx, db, &in)
	if err != nil {
		return Agent{}, err
	}
	return scan(db.QueryRow(ctx, `
		INSERT INTO agents (name, description, instructions, provider, connection_id, model, skills, tools, fallback, timeout_minutes)
		VALUES ($1, $2, $3, $4, $5::uuid, $6, $7::jsonb, $8::jsonb, $9::jsonb, $10) RETURNING `+columns,
		in.Name, in.Description, in.Instructions, in.Provider, in.ConnectionID, in.Model, string(skills), string(tools), string(fallback), in.TimeoutMinutes))
}

// Get lê um agente por ID.
func Get(ctx context.Context, db persistence.DBTX, id string) (Agent, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM agents WHERE id = $1`, id))
}

// GetByName lê um agente por nome.
func GetByName(ctx context.Context, db persistence.DBTX, name string) (Agent, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM agents WHERE name = $1`, name))
}

// List devolve os agentes por nome.
func List(ctx context.Context, db persistence.DBTX) ([]Agent, error) {
	rows, err := db.Query(ctx, `SELECT `+columns+` FROM agents ORDER BY name`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Agent{}
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, persistence.MapError(rows.Err())
}

// Update altera o agente com controle otimista. O nome pode mudar; workflows
// referenciam o agente pelo nome, então renomear exige ajustar os nós.
func Update(ctx context.Context, db persistence.DBTX, id string, expectedVersion int, in Input) (Agent, error) {
	skills, tools, fallback, err := prepare(ctx, db, &in)
	if err != nil {
		return Agent{}, err
	}
	a, err := scan(db.QueryRow(ctx, `
		UPDATE agents SET name = $3, description = $4, instructions = $5, provider = $6, connection_id = $7::uuid, model = $8,
			skills = $9::jsonb, tools = $10::jsonb, fallback = $11::jsonb, timeout_minutes = $12, version = version + 1, updated_at = now()
		WHERE id = $1 AND version = $2 RETURNING `+columns,
		id, expectedVersion, in.Name, in.Description, in.Instructions, in.Provider, in.ConnectionID, in.Model, string(skills), string(tools), string(fallback), in.TimeoutMinutes))
	if errors.Is(err, persistence.ErrNotFound) {
		if _, getErr := Get(ctx, db, id); getErr == nil {
			return Agent{}, persistence.ErrVersionConflict
		}
	}
	return a, err
}
