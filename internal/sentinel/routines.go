package sentinel

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Rotinas internas independentes.
const (
	RoutineIssueSync   = "issues.sync"       // sync geral matinal
	RoutineCorrections = "corrections.poll"  // correções aguardando liberação
	RoutinePRs         = "prs.poll"          // PRs vinculadas
	RoutineMaintenance = "maintenance.sweep" // agentes inativos, timeouts, órfãos
)

// DailyAt é um horário diário no fuso do projeto.
type DailyAt struct {
	Hour, Minute int
	Loc          *time.Location
}

// Routine descreve uma rotina. Exatamente um entre Every e Daily.
type Routine struct {
	Name    string
	Every   time.Duration
	Daily   *DailyAt
	Lease   time.Duration // tempo máximo de uma execução antes de outra poder assumir
	Backoff time.Duration // base da retentativa após falha (dobra até 32x)
	// Run executa a rotina no escopo (id do projeto ou ""), a partir do cursor gravado.
	// Devolve o novo cursor. O cursor é gravado mesmo em falha parcial (cursor não vazio com erro).
	Run func(ctx context.Context, scope, cursor string) (string, error)
}

// State é o estado observável de uma rotina em um escopo.
type State struct {
	Routine       string     `json:"routine"`
	Scope         string     `json:"scope"`
	Cursor        string     `json:"cursor"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	LastError     string     `json:"last_error"`
	FailCount     int        `json:"fail_count"`
	NextRunAt     *time.Time `json:"next_run_at"`
	Runs          int        `json:"runs"`
}

// Scheduler agenda as rotinas. Relógio e cursor são independentes por (rotina, escopo): falha em
// uma não suprime nem atrasa as outras.
type Scheduler struct {
	DB       interface{ persistence.DBTX }
	Now      func() time.Time
	routines map[string]Routine
}

// Register adiciona uma rotina.
func (s *Scheduler) Register(r Routine) error {
	if r.Name == "" || r.Run == nil {
		return fmt.Errorf("%w: rotina sem nome ou função", persistence.ErrInvalidInput)
	}
	if (r.Every > 0) == (r.Daily != nil) {
		return fmt.Errorf("%w: %s: informe Every ou Daily", persistence.ErrInvalidInput, r.Name)
	}
	if r.Lease <= 0 {
		r.Lease = 5 * time.Minute
	}
	if r.Backoff <= 0 {
		r.Backoff = time.Minute
	}
	if s.routines == nil {
		s.routines = map[string]Routine{}
	}
	s.routines[r.Name] = r
	return nil
}

func (s *Scheduler) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// windowStart é o início da janela diária vigente em now.
func (d DailyAt) windowStart(now time.Time) time.Time {
	loc := d.Loc
	if loc == nil {
		loc = time.UTC
	}
	l := now.In(loc)
	w := time.Date(l.Year(), l.Month(), l.Day(), d.Hour, d.Minute, 0, 0, loc)
	if w.After(l) {
		w = w.AddDate(0, 0, -1)
	}
	return w.UTC()
}

func (s *Scheduler) state(ctx context.Context, routine, scope string) (State, error) {
	st := State{Routine: routine, Scope: scope}
	err := s.DB.QueryRow(ctx, `SELECT cursor, last_run_at, last_success_at, last_error, fail_count, next_run_at, runs
		FROM sentinel_routine_state WHERE routine=$1 AND scope=$2`, routine, scope).
		Scan(&st.Cursor, &st.LastRunAt, &st.LastSuccessAt, &st.LastError, &st.FailCount, &st.NextRunAt, &st.Runs)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, nil
	}
	return st, err
}

// State devolve o estado gravado (para o painel).
func (s *Scheduler) State(ctx context.Context, routine, scope string) (State, error) {
	return s.state(ctx, routine, scope)
}

func (r Routine) due(st State, now time.Time) bool {
	if st.NextRunAt != nil && st.NextRunAt.After(now) {
		return false // em retentativa controlada
	}
	if r.Daily != nil {
		w := r.Daily.windowStart(now)
		return st.LastSuccessAt == nil || st.LastSuccessAt.Before(w)
	}
	return st.LastRunAt == nil || !st.LastRunAt.Add(r.Every).After(now)
}

// Result é o desfecho de uma execução.
type Result struct {
	Routine string
	Scope   string
	Ran     bool
	Err     error
}

// RunOne executa a rotina no escopo se estiver vencida (ou force). A reserva é atômica: duas
// chamadas concorrentes (manual e agendada) não executam ao mesmo tempo.
func (s *Scheduler) RunOne(ctx context.Context, name, scope string, force bool) (Result, error) {
	r, ok := s.routines[name]
	if !ok {
		return Result{}, fmt.Errorf("%w: rotina %q", persistence.ErrNotFound, name)
	}
	now := s.now()
	st, err := s.state(ctx, name, scope)
	if err != nil {
		return Result{}, err
	}
	if !force && !r.due(st, now) {
		return Result{Routine: name, Scope: scope}, nil
	}
	tag, err := s.DB.Exec(ctx, `INSERT INTO sentinel_routine_state (routine, scope, running_until) VALUES ($1,$2,$3)
		ON CONFLICT (routine, scope) DO UPDATE SET running_until=$3
		WHERE sentinel_routine_state.running_until IS NULL OR sentinel_routine_state.running_until < $4`,
		name, scope, now.Add(r.Lease), now)
	if err != nil {
		return Result{}, err
	}
	if tag.RowsAffected() == 0 {
		return Result{Routine: name, Scope: scope}, nil // outra execução em andamento
	}
	newCursor, runErr := r.Run(ctx, scope, st.Cursor)
	done := s.now()
	if runErr != nil {
		fails := st.FailCount + 1
		shift := fails - 1
		if shift > 5 {
			shift = 5
		}
		next := done.Add(r.Backoff << shift)
		cur := st.Cursor
		if newCursor != "" {
			cur = newCursor // progresso parcial preservado
		}
		_, err = s.DB.Exec(ctx, `UPDATE sentinel_routine_state SET cursor=$3, last_run_at=$4, last_error=$5, fail_count=$6,
			next_run_at=$7, running_until=NULL, runs=runs+1 WHERE routine=$1 AND scope=$2`,
			name, scope, cur, done, truncate(runErr.Error(), 300), fails, next)
		if err != nil {
			return Result{}, err
		}
		return Result{Routine: name, Scope: scope, Ran: true, Err: runErr}, nil
	}
	_, err = s.DB.Exec(ctx, `UPDATE sentinel_routine_state SET cursor=$3, last_run_at=$4, last_success_at=$4, last_error='',
		fail_count=0, next_run_at=NULL, running_until=NULL, runs=runs+1 WHERE routine=$1 AND scope=$2`, name, scope, newCursor, done)
	if err != nil {
		return Result{}, err
	}
	return Result{Routine: name, Scope: scope, Ran: true}, nil
}

// Tick executa todas as rotinas vencidas nos escopos dados. Uma falha não impede as demais.
func (s *Scheduler) Tick(ctx context.Context, scopes []string) ([]Result, error) {
	names := make([]string, 0, len(s.routines))
	for n := range s.routines {
		names = append(names, n)
	}
	sort.Strings(names)
	if len(scopes) == 0 {
		scopes = []string{""}
	}
	var out []Result
	for _, n := range names {
		for _, sc := range scopes {
			res, err := s.RunOne(ctx, n, sc, false)
			if err != nil {
				return out, err
			}
			if res.Ran {
				out = append(out, res)
			}
		}
	}
	return out, nil
}

// ProjectScopes lista os projetos ativos como escopos das rotinas por projeto.
func ProjectScopes(ctx context.Context, db persistence.DBTX) ([]string, error) {
	rows, err := db.Query(ctx, `SELECT id::text FROM projects ORDER BY created_at, id`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
