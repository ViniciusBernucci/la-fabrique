package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

const columns = `name, kind, description, input, output, permissions, limits, command, operation, version`

func scan(row interface{ Scan(...any) error }) (Tool, error) {
	var t Tool
	var in, out, perms, lim []byte
	if err := row.Scan(&t.Name, &t.Kind, &t.Description, &in, &out, &perms, &lim, &t.Command, &t.Operation, &t.Version); err != nil {
		return t, persistence.MapError(err)
	}
	_ = json.Unmarshal(in, &t.Input)
	_ = json.Unmarshal(out, &t.Output)
	_ = json.Unmarshal(perms, &t.Permissions)
	_ = json.Unmarshal(lim, &t.Limits)
	if t.Input == nil {
		t.Input = []Field{}
	}
	if t.Output == nil {
		t.Output = []Field{}
	}
	return t, nil
}

// ErrNativeName: o nome colide com uma ferramenta nativa.
var ErrNativeName = errors.New("o nome pertence a uma ferramenta nativa")

// ErrPolicy: o comando da ferramenta é proibido pela política.
var ErrPolicy = errors.New("comando fora da política")

// Create cadastra uma ferramenta personalizada. Valida o esquema e confere o
// comando contra a política (o proibido e o que está fora da lista são recusados;
// o sensível é aceito e exigirá aprovação a cada chamada).
func Create(ctx context.Context, db persistence.DBTX, pol *policy.Policy, t Tool) (Tool, error) {
	if t.Kind != KindScript && t.Kind != KindIntegration {
		return Tool{}, &ArgError{"kind", "Ferramentas personalizadas são de script ou de integração."}
	}
	if err := t.Compile(); err != nil {
		return Tool{}, err
	}
	for _, n := range Natives() {
		if n.Name == t.Name {
			return Tool{}, fmt.Errorf("%w: %q", ErrNativeName, t.Name)
		}
	}
	if t.Kind == KindScript {
		d := pol.Evaluate(policy.Action{Tool: policy.ToolScript, Operation: "run", Params: map[string]any{"command": t.Command}})
		if d.Effect == policy.Deny {
			return Tool{}, fmt.Errorf("%w: %s", ErrPolicy, d.Reason)
		}
	}
	in, _ := json.Marshal(t.Input)
	out, _ := json.Marshal(t.Output)
	perms, _ := json.Marshal(t.Permissions)
	lim, _ := json.Marshal(t.Limits)
	return scan(db.QueryRow(ctx, `
		INSERT INTO tools (name, kind, description, input, output, permissions, limits, command, operation)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb, $7::jsonb, $8, $9) RETURNING `+columns,
		t.Name, t.Kind, t.Description, string(in), string(out), string(perms), string(lim), t.Command, t.Operation))
}

// Get lê uma ferramenta personalizada pelo nome.
func Get(ctx context.Context, db persistence.DBTX, name string) (Tool, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM tools WHERE name = $1`, name))
}

// List devolve as ferramentas personalizadas.
func List(ctx context.Context, db persistence.DBTX) ([]Tool, error) {
	rows, err := db.Query(ctx, `SELECT `+columns+` FROM tools ORDER BY name`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Tool{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, persistence.MapError(rows.Err())
}
