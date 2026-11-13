package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Tipos de workflow.
const (
	KindFeature = "feature"
	KindSubflow = "subflow"
)

// Workflow é um rascunho persistido.
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Description string    `json:"description"`
	Graph       Graph     `json:"graph"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Summary é a linha da listagem.
type Summary struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Kind        string    `json:"kind"`
	Description string    `json:"description"`
	Version     int       `json:"version"`
	Nodes       int       `json:"nodes"`
	Issues      int       `json:"issues"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const columns = `id, name, kind, description, graph, version, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Workflow, error) {
	var w Workflow
	var raw []byte
	if err := row.Scan(&w.ID, &w.Name, &w.Kind, &w.Description, &raw, &w.Version, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return w, persistence.MapError(err)
	}
	if err := json.Unmarshal(raw, &w.Graph); err != nil {
		return w, fmt.Errorf("grafo corrompido: %w", err)
	}
	normalize(&w.Graph)
	return w, nil
}

func normalize(g *Graph) {
	if g.Nodes == nil {
		g.Nodes = []Node{}
	}
	if g.Edges == nil {
		g.Edges = []Edge{}
	}
	for i := range g.Nodes {
		if g.Nodes[i].Props == nil {
			g.Nodes[i].Props = map[string]any{}
		}
	}
}

// Starter é o grafo inicial: início ligado a um fim.
func Starter() Graph {
	return Graph{
		Nodes: []Node{
			{ID: "inicio", Type: TStart, X: 80, Y: 140, Props: map[string]any{}},
			{ID: "fim", Type: TEnd, X: 420, Y: 140, Props: map[string]any{"outcome": "success"}},
		},
		Edges: []Edge{{ID: "e1", From: "inicio", FromPort: PortOut, To: "fim", ToPort: PortIn}},
	}
}

func checkMeta(name, kind, description string) (string, string, *FieldError) {
	name, description = strings.TrimSpace(name), strings.TrimSpace(description)
	if name == "" || len(name) > 120 {
		return "", "", &FieldError{"name", "O nome deve ter entre 1 e 120 caracteres."}
	}
	if len(description) > 500 {
		return "", "", &FieldError{"description", "A descrição deve ter até 500 caracteres."}
	}
	return name, description, nil
}

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Create cria um rascunho com o grafo inicial.
func Create(ctx context.Context, db persistence.DBTX, name, kind, description string) (Workflow, error) {
	name, description, ferr := checkMeta(name, kind, description)
	if ferr != nil {
		return Workflow{}, ferr
	}
	if kind != KindFeature && kind != KindSubflow {
		return Workflow{}, &FieldError{"kind", "O tipo deve ser feature ou subflow."}
	}
	raw, _ := json.Marshal(Starter())
	return scan(db.QueryRow(ctx, `INSERT INTO workflows (name, kind, description, graph) VALUES ($1, $2, $3, $4::jsonb) RETURNING `+columns,
		name, kind, description, string(raw)))
}

// Get lê um workflow.
func Get(ctx context.Context, db persistence.DBTX, id string) (Workflow, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM workflows WHERE id = $1`, id))
}

// List devolve os workflows com contagem de nós e de problemas.
func List(ctx context.Context, db persistence.DBTX) ([]Summary, error) {
	rows, err := db.Query(ctx, `SELECT `+columns+` FROM workflows ORDER BY updated_at DESC, id`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Summary{}
	for rows.Next() {
		w, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, Summary{ID: w.ID, Name: w.Name, Kind: w.Kind, Description: w.Description, Version: w.Version,
			Nodes: len(w.Graph.Nodes), Issues: len(Validate(w.Graph)), UpdatedAt: w.UpdatedAt})
	}
	return out, persistence.MapError(rows.Err())
}

// Save grava nome, descrição e grafo com controle otimista. Rascunho inválido é
// aceito (os problemas voltam na leitura); só limites estruturais são recusados.
func Save(ctx context.Context, db persistence.DBTX, id string, expectedVersion int, name, description string, g Graph) (Workflow, error) {
	name, description, ferr := checkMeta(name, "", description)
	if ferr != nil {
		return Workflow{}, ferr
	}
	normalize(&g)
	if len(g.Nodes) > MaxNodes || len(g.Edges) > MaxEdges {
		return Workflow{}, &FieldError{"graph", "O fluxo excede o tamanho máximo."}
	}
	raw, err := json.Marshal(g)
	if err != nil {
		return Workflow{}, &FieldError{"graph", "Grafo inválido."}
	}
	w, err := scan(db.QueryRow(ctx, `
		UPDATE workflows SET name = $3, description = $4, graph = $5::jsonb, version = version + 1, updated_at = now()
		WHERE id = $1 AND version = $2 RETURNING `+columns, id, expectedVersion, name, description, string(raw)))
	if err == persistence.ErrNotFound {
		if _, getErr := Get(ctx, db, id); getErr == nil {
			return Workflow{}, persistence.ErrVersionConflict
		}
	}
	return w, err
}
