package techplan

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
)

// Subtask é a unidade interna materializada.
type Subtask struct {
	Key           string     `json:"key"`
	Track         flow.Track `json:"track"`
	Risk          string     `json:"risk"`
	DependsOn     []string   `json:"depends_on"`
	Resources     []string   `json:"resources"`
	ExclusiveBoth bool       `json:"exclusive_both"`
	Hash          string     `json:"content_hash"`
}

// TrackOfPath classifica um caminho: interface web é FRONTEND; o resto (API, banco, infra) BACKEND.
func TrackOfPath(p string) flow.Track {
	if strings.HasPrefix(p, "apps/web/") || strings.HasSuffix(p, ".tsx") || strings.HasSuffix(p, ".css") {
		return flow.Frontend
	}
	return flow.Backend
}

// ResourcesOfPath devolve o recurso compartilhado que o caminho implica ("" se nenhum).
func ResourcesOfPath(p string) string {
	switch {
	case p == "contracts/openapi.yaml" || strings.HasPrefix(p, "contracts/"):
		return "contract:api"
	case strings.HasPrefix(p, "db/migrations/") || strings.HasPrefix(p, "db/schema"):
		return "schema:db"
	case p == "go.mod" || p == "go.sum":
		return "deps:go"
	case p == "apps/web/package.json" || p == "apps/web/package-lock.json":
		return "deps:web"
	case p == ".env.example" || strings.HasPrefix(p, "infra/"):
		return "config:infra"
	}
	return ""
}

func resourcesOf(t planning.Task) []string {
	set := map[string]bool{}
	for _, r := range t.SharedResources {
		if r = strings.TrimSpace(r); r != "" {
			set[r] = true
		}
	}
	for _, p := range t.Paths {
		if r := ResourcesOfPath(p); r != "" {
			set[r] = true
		}
	}
	out := make([]string, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// Validate confere o que a adequação exige além da validação geral do plano. Pura.
func Validate(p planning.Plan) []planning.Problem {
	out := planning.Validate(p)
	add := func(task, field, msg string) {
		out = append(out, planning.Problem{Task: task, Field: field, Message: msg})
	}
	for _, t := range p.Tasks {
		switch flow.Track(t.Track) {
		case flow.Frontend, flow.Backend:
		default:
			add(t.Key, "track", "classifique a subtarefa como FRONTEND ou BACKEND")
			continue
		}
		front, back := 0, 0
		for _, pth := range t.Paths {
			if TrackOfPath(pth) == flow.Frontend {
				front++
			} else {
				back++
			}
		}
		if front > 0 && back > 0 && !t.Indivisible {
			add(t.Key, "track", "tarefa mista: divida em subtarefas FRONTEND e BACKEND ou marque como indivisível")
		}
		if t.Indivisible && front > 0 && back > 0 {
			pred := flow.Backend
			if front > back {
				pred = flow.Frontend
			}
			if flow.Track(t.Track) != pred {
				add(t.Key, "track", "a trilha declarada deve ser a responsabilidade predominante ("+string(pred)+")")
			}
		}
		if !t.Indivisible && front+back > 0 {
			only := flow.Backend
			if front > 0 {
				only = flow.Frontend
			}
			if front == 0 || back == 0 {
				if flow.Track(t.Track) != only {
					add(t.Key, "track", "a trilha não confere com os arquivos ("+string(only)+")")
				}
			}
		}
		if t.ExclusiveBoth && !t.Indivisible {
			add(t.Key, "exclusive_both", "acesso exclusivo às duas trilhas só vale para trabalho indivisível")
		}
	}
	return out
}

// Materialize deriva as subtarefas internas de um plano válido, em ordem estável.
func Materialize(p planning.Plan) []Subtask {
	n := planning.Normalize(p)
	out := make([]Subtask, 0, len(n.Tasks))
	for _, t := range n.Tasks {
		deps := append([]string{}, t.DependsOn...)
		out = append(out, Subtask{Key: t.Key, Track: flow.Track(t.Track), Risk: t.Risk, DependsOn: deps, Resources: resourcesOf(t),
			ExclusiveBoth: t.ExclusiveBoth, Hash: planning.TaskHash(t)})
	}
	return out
}

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Result é o resultado de aceitar uma proposta.
type Result struct {
	PlanVersion int       `json:"plan_version"`
	Unchanged   bool      `json:"unchanged"`
	Subtasks    []Subtask `json:"subtasks"`
}

// ErrRejected indica proposta recusada: nada foi gravado.
var ErrRejected = errors.New("proposta do Tech Lead recusada: nada foi materializado")

// RejectedError carrega os motivos.
type RejectedError struct{ Problems []planning.Problem }

func (e *RejectedError) Error() string {
	ps := make([]string, 0, len(e.Problems))
	for _, p := range e.Problems {
		ps = append(ps, p.String())
	}
	return ErrRejected.Error() + ": " + strings.Join(ps, "; ")
}
func (e *RejectedError) Unwrap() error { return ErrRejected }

// Service valida, versiona o plano e materializa as subtarefas.
type Service struct {
	DB    DB
	Plans *planning.Store
}

// Accept valida a proposta (plano geral + trilhas), grava o plano versionado e materializa as
// subtarefas internas. Idempotente: a mesma proposta não cria versão nem duplica subtarefas.
func (s *Service) Accept(ctx context.Context, projectID, featureID string, p planning.Plan) (Result, error) {
	if ps := Validate(p); len(ps) > 0 {
		return Result{}, &RejectedError{ps}
	}
	o, err := s.Plans.Propose(ctx, projectID, featureID, p)
	if err != nil {
		return Result{}, err
	}
	subs := Materialize(p)
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		for _, st := range subs {
			deps, _ := json.Marshal(st.DependsOn)
			res, _ := json.Marshal(st.Resources)
			tag, err := tx.Exec(ctx, `INSERT INTO subtasks (project_id, feature_id, plan_version, task_key, track, risk, depends_on, resources, exclusive_both, content_hash)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT (feature_id, plan_version, task_key) DO NOTHING`,
				projectID, featureID, o.Stored.Version, st.Key, string(st.Track), st.Risk, deps, res, st.ExclusiveBoth, st.Hash)
			if err != nil {
				return err
			}
			if tag.RowsAffected() == 0 {
				var h string
				if err := tx.QueryRow(ctx, `SELECT content_hash FROM subtasks WHERE feature_id=$1 AND plan_version=$2 AND task_key=$3`, featureID, o.Stored.Version, st.Key).Scan(&h); err != nil {
					return err
				}
				if h != st.Hash {
					return fmt.Errorf("%w: subtarefa %s mudou dentro da mesma versão do plano", persistence.ErrDuplicate, st.Key)
				}
			}
		}
		return nil
	})
	if err != nil {
		return Result{}, persistence.MapError(err)
	}
	return Result{PlanVersion: o.Stored.Version, Unchanged: o.Unchanged, Subtasks: subs}, nil
}

// List devolve as subtarefas materializadas de uma versão do plano.
func (s *Service) List(ctx context.Context, featureID string, version int) ([]Subtask, error) {
	rows, err := s.DB.Query(ctx, `SELECT task_key, track, risk, depends_on, resources, exclusive_both, content_hash FROM subtasks
		WHERE feature_id=$1 AND plan_version=$2 ORDER BY task_key`, featureID, version)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Subtask
	for rows.Next() {
		var st Subtask
		var tr string
		var deps, res []byte
		if err := rows.Scan(&st.Key, &tr, &st.Risk, &deps, &res, &st.ExclusiveBoth, &st.Hash); err != nil {
			return nil, err
		}
		st.Track = flow.Track(tr)
		_ = json.Unmarshal(deps, &st.DependsOn)
		_ = json.Unmarshal(res, &st.Resources)
		out = append(out, st)
	}
	return out, rows.Err()
}
