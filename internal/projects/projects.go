// Package projects implementa o cadastro de projetos (estado, versão e leitura
// paginada). O assistente de cadastro e as integrações pertencem ao E05.
package projects

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Estados válidos de um projeto.
const (
	StateDraft    = "draft"
	StateActive   = "active"
	StatePaused   = "paused"
	StateArchived = "archived"
)

var (
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	validStates = map[string]bool{StateDraft: true, StateActive: true, StatePaused: true, StateArchived: true}
)

// Project é o agregado de cadastro.
type Project struct {
	ID        string    `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	State     string    `json:"state"`
	Priority  int       `json:"priority"`
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewProject reúne os dados de criação.
type NewProject struct {
	Slug     string
	Name     string
	Priority int
}

const columns = `id, slug, name, state, priority, version, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.State, &p.Priority, &p.Version, &p.CreatedAt, &p.UpdatedAt)
	return p, persistence.MapError(err)
}

// Create cria um projeto em rascunho, sem vínculos.
func Create(ctx context.Context, db persistence.DBTX, in NewProject) (Project, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	if !slugPattern.MatchString(in.Slug) || len(in.Slug) > 63 {
		return Project{}, fmt.Errorf("%w: slug deve usar a-z, 0-9 e hífens", persistence.ErrInvalidInput)
	}
	if in.Name == "" || len(in.Name) > 120 {
		return Project{}, fmt.Errorf("%w: nome deve ter entre 1 e 120 caracteres", persistence.ErrInvalidInput)
	}
	if in.Priority == 0 {
		in.Priority = 2
	}
	return scan(db.QueryRow(ctx,
		`INSERT INTO projects (slug, name, priority) VALUES ($1, $2, $3) RETURNING `+columns,
		in.Slug, in.Name, in.Priority))
}

// Get lê um projeto por ID.
func Get(ctx context.Context, db persistence.DBTX, id string) (Project, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM projects WHERE id = $1`, id))
}

// Patch descreve alterações; campos nil permanecem.
type Patch struct {
	Name     *string
	State    *string
	Priority *int
}

// Update altera o projeto com controle otimista: se expectedVersion não for a
// atual, devolve persistence.ErrVersionConflict.
func Update(ctx context.Context, db persistence.DBTX, id string, expectedVersion int, p Patch) (Project, error) {
	if p.State != nil && !validStates[*p.State] {
		return Project{}, fmt.Errorf("%w: estado desconhecido", persistence.ErrInvalidInput)
	}
	updated, err := scan(db.QueryRow(ctx, `
		UPDATE projects SET
			name = COALESCE($3, name),
			state = COALESCE($4, state),
			priority = COALESCE($5, priority),
			version = version + 1, updated_at = now()
		WHERE id = $1 AND version = $2
		RETURNING `+columns, id, expectedVersion, p.Name, p.State, p.Priority))
	if err == persistence.ErrNotFound {
		if _, getErr := Get(ctx, db, id); getErr == nil {
			return Project{}, persistence.ErrVersionConflict
		}
	}
	return updated, err
}

// ListParams controla filtro e paginação por cursor.
type ListParams struct {
	Limit  int
	Cursor string
	State  string
}

// Page é uma página de projetos.
type Page struct {
	Items      []Project `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

type cursor struct {
	CreatedAt time.Time `json:"c"`
	ID        string    `json:"i"`
}

func encodeCursor(p Project) string {
	b, _ := json.Marshal(cursor{CreatedAt: p.CreatedAt, ID: p.ID})
	return base64.RawURLEncoding.EncodeToString(b)
}

func decodeCursor(raw string) (cursor, error) {
	var c cursor
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(b, &c) != nil || c.ID == "" {
		return c, fmt.Errorf("%w: cursor inválido", persistence.ErrInvalidInput)
	}
	return c, nil
}

// List devolve projetos do mais novo ao mais antigo, com paginação por cursor.
func List(ctx context.Context, db persistence.DBTX, params ListParams) (Page, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		return Page{}, fmt.Errorf("%w: limit máximo é 200", persistence.ErrInvalidInput)
	}
	if params.State != "" && !validStates[params.State] {
		return Page{}, fmt.Errorf("%w: estado desconhecido", persistence.ErrInvalidInput)
	}
	args := []any{limit + 1}
	where := []string{"true"}
	if params.State != "" {
		args = append(args, params.State)
		where = append(where, fmt.Sprintf("state = $%d", len(args)))
	}
	if params.Cursor != "" {
		c, err := decodeCursor(params.Cursor)
		if err != nil {
			return Page{}, err
		}
		args = append(args, c.CreatedAt, c.ID)
		where = append(where, fmt.Sprintf("(created_at, id) < ($%d, $%d::uuid)", len(args)-1, len(args)))
	}
	rows, err := db.Query(ctx, `SELECT `+columns+` FROM projects WHERE `+strings.Join(where, " AND ")+
		` ORDER BY created_at DESC, id DESC LIMIT $1`, args...)
	if err != nil {
		return Page{}, persistence.MapError(err)
	}
	defer rows.Close()
	page := Page{Items: []Project{}}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, p)
	}
	if err := rows.Err(); err != nil {
		return Page{}, persistence.MapError(err)
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		page.NextCursor = encodeCursor(page.Items[limit-1])
	}
	return page, nil
}
