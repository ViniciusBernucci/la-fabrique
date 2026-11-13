package risk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
)

// Fases da classificação.
const (
	PhasePlanning = "planning" // caminhos previstos pela tarefa
	PhaseDiff     = "diff"     // diff real depois da implementação
	PhaseFix      = "fix"      // diff real depois de uma correção
)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Store guarda a política e o histórico.
type Store struct{ DB persistence.DBTX }

// Policy devolve a política do projeto (a padrão se nunca foi configurada).
func (s *Store) Policy(ctx context.Context, projectID string) (Policy, error) {
	var raw []byte
	err := s.DB.QueryRow(ctx, `SELECT policy FROM risk_policies WHERE project_id = $1`, projectID).Scan(&raw)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		var one int
		if e := s.DB.QueryRow(ctx, `SELECT 1 FROM projects WHERE id = $1`, projectID).Scan(&one); e != nil {
			return Policy{}, persistence.MapError(e)
		}
		return DefaultPolicy(), nil
	}
	if err != nil {
		return Policy{}, persistence.MapError(err)
	}
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf("política de risco ilegível: %w", err)
	}
	return p, nil
}

// SetPolicy grava a política do projeto (substitui a anterior).
func (s *Store) SetPolicy(ctx context.Context, projectID string, p Policy) (Policy, error) {
	if err := p.Validate(); err != nil {
		return Policy{}, &FieldError{"policy", err.Error()}
	}
	if p.Rules == nil {
		p.Rules = []Rule{}
	}
	b, _ := json.Marshal(p)
	_, err := s.DB.Exec(ctx, `INSERT INTO risk_policies (project_id, policy) VALUES ($1, $2::jsonb)
		ON CONFLICT (project_id) DO UPDATE SET policy = $2::jsonb, version = risk_policies.version + 1, updated_at = now()`, projectID, string(b))
	if err != nil {
		return Policy{}, persistence.MapError(err)
	}
	return p, nil
}

// Record é uma classificação gravada.
type Record struct {
	ID       string `json:"id"`
	TaskKey  string `json:"task_key"`
	Phase    string `json:"phase"`
	Revision string `json:"revision"`
	Assessment
	Files []File `json:"files"`
}

// FromTask transforma a tarefa do plano em entrada: a sugestão da IA é o risco inicial da
// tarefa e os caminhos previstos entram como arquivos sem tamanho.
func FromTask(t planning.Task) (string, []File) {
	files := make([]File, 0, len(t.Paths))
	for _, p := range t.Paths {
		files = append(files, File{Path: p})
	}
	return t.Risk, files
}

// Assess classifica e grava. Na fase de planejamento `files` são os caminhos previstos;
// nas fases diff/fix são os arquivos realmente alterados (com o tamanho).
func (s *Store) Assess(ctx context.Context, projectID, featureID, taskKey, phase, revision, suggested string, files []File) (Record, error) {
	if phase != PhasePlanning && phase != PhaseDiff && phase != PhaseFix {
		return Record{}, &FieldError{"phase", "use planning, diff ou fix"}
	}
	if strings.TrimSpace(taskKey) == "" {
		return Record{}, &FieldError{"task_key", "informe a tarefa"}
	}
	if len(files) > 5000 {
		return Record{}, &FieldError{"files", "arquivos demais para classificar"}
	}
	for _, f := range files {
		if strings.TrimSpace(f.Path) == "" || strings.HasPrefix(f.Path, "/") || strings.Contains(f.Path, "..") || f.Added < 0 || f.Removed < 0 {
			return Record{}, &FieldError{"files", fmt.Sprintf("arquivo inválido: %q", f.Path)}
		}
	}
	pol, err := (&Store{DB: s.DB}).Policy(ctx, projectID)
	if err != nil {
		return Record{}, err
	}
	a := Assess(pol, suggested, files)
	rj, _ := json.Marshal(a.Reasons)
	fj, _ := json.Marshal(nonNil(files))
	var id string
	if err := s.DB.QueryRow(ctx, `INSERT INTO risk_assessments (project_id, feature_id, task_key, phase, revision, suggested, final, raised, reasons, files)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb) RETURNING id`,
		projectID, featureID, taskKey, phase, revision, a.Suggested, a.Final, a.Raised, string(rj), string(fj)).Scan(&id); err != nil {
		return Record{}, persistence.MapError(err)
	}
	return Record{ID: id, TaskKey: taskKey, Phase: phase, Revision: revision, Assessment: a, Files: nonNil(files)}, nil
}

func nonNil(f []File) []File {
	if f == nil {
		return []File{}
	}
	return f
}

// History lista as classificações da tarefa, da mais recente para a mais antiga.
func (s *Store) History(ctx context.Context, projectID, featureID, taskKey string) ([]Record, error) {
	rows, err := s.DB.Query(ctx, `SELECT id, task_key, phase, revision, suggested, final, raised, reasons, files FROM risk_assessments
		WHERE project_id = $1 AND feature_id = $2 AND task_key = $3 ORDER BY seq DESC`, projectID, featureID, taskKey)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		var r Record
		var reasons, files []byte
		if err := rows.Scan(&r.ID, &r.TaskKey, &r.Phase, &r.Revision, &r.Suggested, &r.Final, &r.Raised, &reasons, &files); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(reasons, &r.Reasons)
		_ = json.Unmarshal(files, &r.Files)
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}

// Latest devolve a classificação mais recente da tarefa (a que vale agora).
func (s *Store) Latest(ctx context.Context, projectID, featureID, taskKey string) (Record, error) {
	h, err := s.History(ctx, projectID, featureID, taskKey)
	if err != nil {
		return Record{}, err
	}
	if len(h) == 0 {
		return Record{}, persistence.ErrNotFound
	}
	return h[0], nil
}
