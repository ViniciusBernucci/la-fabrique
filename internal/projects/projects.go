// Package projects implementa o cadastro de projetos: rascunho do assistente,
// validação local de preenchimento, ativação/suspensão com trilha e leitura
// paginada (E05). A validação de acesso remoto (T05-02) e de recursos (T05-03)
// entra como novos itens do mesmo relatório.
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

// Etapas do assistente de cadastro, na ordem.
const (
	StepBasics     = "basics"
	StepRepository = "repository"
	StepSource     = "source"
	StepReview     = "review"
)

var validSteps = map[string]bool{StepBasics: true, StepRepository: true, StepSource: true, StepReview: true}

// Repository é o repositório GitHub atendido. Vazio = ainda não preenchido.
type Repository struct {
	Owner        string  `json:"owner"`
	Name         string  `json:"name"`
	BaseBranch   string  `json:"base_branch"`
	ConnectionID *string `json:"connection_id"`
}

// Source é a origem oficial de tarefas (uma por projeto): Jira ou ClickUp.
type Source struct {
	Kind         string  `json:"kind"`
	ConnectionID *string `json:"connection_id"`
	Container    string  `json:"container"`
}

func (s Source) empty() bool { return s.Kind == "" && s.ConnectionID == nil && s.Container == "" }

// Project é o agregado de cadastro. Nasce sem vínculos de agentes, workflows
// ou recursos globais; credenciais aparecem só como referência de conexão.
type Project struct {
	ID          string     `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	State       string     `json:"state"`
	Priority    int        `json:"priority"`
	WizardStep  string     `json:"wizard_step"`
	Repository  Repository `json:"repository"`
	Source      Source     `json:"source"`
	Version     int        `json:"version"`
	ActivatedAt *time.Time `json:"activated_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewProject reúne os dados de criação.
type NewProject struct {
	Slug        string
	Name        string
	Description string
	Priority    *int // nil = padrão (2); 0 é uma prioridade válida
}

const columns = `id, slug, name, description, state, priority, wizard_step,
	repo_owner, repo_name, base_branch, repo_connection_id,
	COALESCE(source_kind, ''), source_connection_id, source_container,
	version, activated_at, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Project, error) {
	var p Project
	err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.State, &p.Priority, &p.WizardStep,
		&p.Repository.Owner, &p.Repository.Name, &p.Repository.BaseBranch, &p.Repository.ConnectionID,
		&p.Source.Kind, &p.Source.ConnectionID, &p.Source.Container,
		&p.Version, &p.ActivatedAt, &p.CreatedAt, &p.UpdatedAt)
	return p, persistence.MapError(err)
}

// Create cria um projeto em rascunho, sem vínculos.
func Create(ctx context.Context, db persistence.DBTX, in NewProject) (Project, error) {
	in.Slug = strings.TrimSpace(in.Slug)
	in.Name = strings.TrimSpace(in.Name)
	if !slugPattern.MatchString(in.Slug) || len(in.Slug) > 63 {
		return Project{}, &FieldError{Field: "slug", Message: "O identificador deve usar letras minúsculas, números e hífens (até 63 caracteres)."}
	}
	if in.Name == "" || len(in.Name) > 120 {
		return Project{}, &FieldError{Field: "name", Message: "O nome deve ter entre 1 e 120 caracteres."}
	}
	in.Description = strings.TrimSpace(in.Description)
	if len(in.Description) > 500 {
		return Project{}, &FieldError{Field: "description", Message: "A descrição deve ter até 500 caracteres."}
	}
	priority := 2
	if in.Priority != nil {
		priority = *in.Priority
	}
	if priority < 0 || priority > 4 {
		return Project{}, &FieldError{Field: "priority", Message: "A prioridade deve estar entre 0 e 4."}
	}
	// Criar conclui a etapa "Dados": o assistente retoma em "Repositório".
	return scan(db.QueryRow(ctx,
		`INSERT INTO projects (slug, name, description, priority, wizard_step) VALUES ($1, $2, $3, $4, 'repository') RETURNING `+columns,
		in.Slug, in.Name, in.Description, priority))
}

// Get lê um projeto por ID.
func Get(ctx context.Context, db persistence.DBTX, id string) (Project, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM projects WHERE id = $1`, id))
}

func getForUpdate(ctx context.Context, db persistence.DBTX, id string) (Project, error) {
	return scan(db.QueryRow(ctx, `SELECT `+columns+` FROM projects WHERE id = $1 FOR UPDATE`, id))
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
