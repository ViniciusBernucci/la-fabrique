package issuesync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Gatilhos.
const (
	TriggerDaily    = "daily"
	TriggerManual   = "manual"
	TriggerTargeted = "targeted"
)

const (
	pageLimit = 100
	maxPages  = 50
	maxKnown  = 500
)

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Source é a origem configurada do projeto.
type Source struct {
	Kind      string // jira | clickup
	Container string
	Map       flow.ProjectMap
}

// Syncer sincroniza.
type Syncer struct {
	DB       DB
	Sentinel *sentinel.Coordinator // opcional: rastreia as Issues novas
	// PRD, quando definido, é chamado para Issue em PRD-DOC (nova ou com revisão nova): lê o PRD da origem e
	// roda o gate. Pendências do PRD não são erro de sync: ficam registradas e a Issue permanece em PRD-DOC.
	PRD func(ctx context.Context, projectID string, it integrations.Item) error
	Now      func() time.Time
}

func (s *Syncer) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Run é o resultado de uma sincronização.
type Run struct {
	ID       int64     `json:"id"`
	Trigger  string    `json:"trigger"`
	Scope    string    `json:"scope"`
	Outcome  string    `json:"outcome"` // ok | partial | failed
	Read     int       `json:"read"`
	New      int       `json:"new"`
	Updated  int       `json:"updated"`
	Ignored  int       `json:"ignored"`
	Error    string    `json:"error,omitempty"`
	Started  time.Time `json:"started_at"`
	Finished time.Time `json:"finished_at"`
}

type outcome int

const (
	oNew outcome = iota
	oUpdated
	oSame
	oIgnored
)

func entryKind(code string) (string, bool) {
	switch code {
	case flow.PRDDoc:
		return "feature", true
	case flow.CorrecaoPR:
		return "correction", true
	}
	return "", false
}

// apply grava um item lido: novo elegível, conhecido atualizado (em qualquer estado) ou ignorado.
func (s *Syncer) apply(ctx context.Context, projectID string, src Source, it integrations.Item) (outcome, error) {
	if it.ID == "" {
		return oIgnored, nil
	}
	code := ""
	if r, ok := src.Map.Resolve(it.Status); ok {
		code = r.Code
	}
	var rev, kindDB string
	var known bool
	err := s.DB.QueryRow(ctx, `SELECT revision, kind FROM issue_registry WHERE project_id=$1 AND source=$2 AND remote_id=$3`, projectID, src.Kind, it.ID).Scan(&rev, &kindDB)
	switch {
	case err == nil:
		known = true
	case errors.Is(err, pgx.ErrNoRows):
	default:
		return oIgnored, persistence.MapError(err)
	}
	if !known {
		kind, eligible := entryKind(code)
		if !eligible || it.Subtask {
			return oIgnored, nil
		}
		if _, err := s.DB.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, title, description, raw_status, status_code, revision, priority, kind, url, last_seen_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT (project_id, source, remote_id) DO NOTHING`,
			projectID, src.Kind, it.ID, it.Key, it.Title, it.Description, it.Status, code, it.Version, it.Priority, kind, it.URL, s.now()); err != nil {
			return oIgnored, persistence.MapError(err)
		}
		if s.Sentinel != nil {
			if _, err := s.Sentinel.Track(ctx, sentinel.NewIssue{ProjectID: projectID, Key: it.Key, Kind: kind, Status: code}); err != nil {
				return oIgnored, err
			}
		}
		if code == flow.PRDDoc && s.PRD != nil {
			if err := s.PRD(ctx, projectID, it); err != nil {
				return oNew, err
			}
		}
		return oNew, nil
	}
	// Liberação de correção (sync geral ou consulta direcionada): a Sentinela recebe a observação ANTES de
	// o registro mudar; se falhar, o item continua aguardando e a próxima consulta repete (deduplicada).
	if kindDB == "correction" && code == flow.CorrecaoPR {
		if err := s.forwardRelease(ctx, projectID, it); err != nil {
			return oIgnored, err
		}
	}
	if rev == it.Version && it.Version != "" {
		_, err := s.DB.Exec(ctx, `UPDATE issue_registry SET last_seen_at=$4 WHERE project_id=$1 AND source=$2 AND remote_id=$3`, projectID, src.Kind, it.ID, s.now())
		return oSame, persistence.MapError(err)
	}
	// Atualiza o conhecido em qualquer status; não toca na prioridade local (edição humana).
	if _, err := s.DB.Exec(ctx, `UPDATE issue_registry SET issue_key=$4, title=$5, description=$6, raw_status=$7, status_code=$8, revision=$9, priority=$10, url=$11, last_seen_at=$12
		WHERE project_id=$1 AND source=$2 AND remote_id=$3`, projectID, src.Kind, it.ID, it.Key, it.Title, it.Description, it.Status, code, it.Version, it.Priority, it.URL, s.now()); err != nil {
		return oIgnored, persistence.MapError(err)
	}
	if code == flow.PRDDoc && s.PRD != nil {
		if err := s.PRD(ctx, projectID, it); err != nil {
			return oUpdated, err
		}
	}
	return oUpdated, nil
}

func (s *Syncer) record(ctx context.Context, projectID, trigger, scope string, r Run) (Run, error) {
	err := s.DB.QueryRow(ctx, `INSERT INTO issue_sync_runs (project_id, trigger, scope, outcome, read_count, new_count, updated, ignored, error, started_at, finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`, projectID, trigger, scope, r.Outcome, r.Read, r.New, r.Updated, r.Ignored, r.Error, r.Started, r.Finished).Scan(&r.ID)
	return r, persistence.MapError(err)
}

func (s *Syncer) known(ctx context.Context, projectID, source string) ([]string, error) {
	rows, err := s.DB.Query(ctx, `SELECT remote_id FROM issue_registry WHERE project_id=$1 AND source=$2 ORDER BY remote_id LIMIT $3`, projectID, source, maxKnown)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// General executa o sync geral (diário ou manual): listagem paginada com Issues fechadas incluídas
// e leitura por ID das conhecidas que a listagem não trouxe. Falha no meio mantém o que já foi
// gravado e devolve o Run com outcome "partial" ou "failed" (sem erro Go: o resultado é o registro).
func (s *Syncer) General(ctx context.Context, projectID, trigger string, tr integrations.Tracker, src Source) (Run, error) {
	if trigger != TriggerDaily && trigger != TriggerManual {
		return Run{}, fmt.Errorf("%w: gatilho %q", persistence.ErrInvalidInput, trigger)
	}
	run := Run{Trigger: trigger, Scope: "all", Started: s.now()}
	seen := map[string]bool{}
	fail := func(err error) (Run, error) {
		msg := "falha ao ler a origem"
		var ie *integrations.Error
		if errors.As(err, &ie) {
			msg = ie.Error()
		}
		run.Error, run.Outcome = msg, "failed"
		if run.Read > 0 {
			run.Outcome = "partial"
		}
		run.Finished = s.now()
		return s.record(ctx, projectID, trigger, "all", run)
	}
	count := func(o outcome) {
		switch o {
		case oNew:
			run.New++
		case oUpdated:
			run.Updated++
		case oIgnored:
			run.Ignored++
		}
	}
	cursor := ""
	for page := 0; ; page++ {
		if page >= maxPages {
			return fail(fmt.Errorf("a origem tem mais de %d páginas: leitura incompleta", maxPages))
		}
		pg, err := tr.ListItems(ctx, src.Container, integrations.ListOptions{IncludeDone: true, Cursor: cursor, Limit: pageLimit})
		if err != nil {
			return fail(err)
		}
		for _, it := range pg.Items {
			run.Read++
			if seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			o, err := s.apply(ctx, projectID, src, it)
			if err != nil {
				return Run{}, err
			}
			count(o)
		}
		if pg.NextCursor == "" {
			break
		}
		cursor = pg.NextCursor
	}
	ids, err := s.known(ctx, projectID, src.Kind)
	if err != nil {
		return Run{}, err
	}
	sort.Strings(ids)
	for _, id := range ids {
		if seen[id] {
			continue
		}
		it, err := tr.GetItem(ctx, id)
		if err != nil {
			return fail(err)
		}
		run.Read++
		o, err := s.apply(ctx, projectID, src, it)
		if err != nil {
			return Run{}, err
		}
		count(o)
	}
	run.Outcome, run.Finished = "ok", s.now()
	return s.record(ctx, projectID, trigger, "all", run)
}

// Issue é a Issue registrada.
type Issue struct {
	ID         string `json:"id"`
	Source     string `json:"source"`
	RemoteID   string `json:"remote_id"`
	Key        string `json:"key"`
	Title      string `json:"title"`
	RawStatus  string `json:"raw_status"`
	StatusCode string `json:"status_code"`
	Revision   string `json:"revision"`
	Kind       string `json:"kind"`
	LocalPrio  *int   `json:"local_priority,omitempty"`
}

// List devolve as Issues registradas do projeto.
func (s *Syncer) List(ctx context.Context, projectID string) ([]Issue, error) {
	rows, err := s.DB.Query(ctx, `SELECT id::text, source, remote_id, issue_key, title, raw_status, status_code, revision, kind, local_priority
		FROM issue_registry WHERE project_id=$1 ORDER BY issue_key`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Issue
	for rows.Next() {
		var i Issue
		if err := rows.Scan(&i.ID, &i.Source, &i.RemoteID, &i.Key, &i.Title, &i.RawStatus, &i.StatusCode, &i.Revision, &i.Kind, &i.LocalPrio); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

// Runs devolve o histórico de sincronizações (mais recentes primeiro).
func (s *Syncer) Runs(ctx context.Context, projectID string, limit int) ([]Run, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.DB.Query(ctx, `SELECT id, trigger, scope, outcome, read_count, new_count, updated, ignored, error, started_at, finished_at
		FROM issue_sync_runs WHERE project_id=$1 ORDER BY id DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		if err := rows.Scan(&r.ID, &r.Trigger, &r.Scope, &r.Outcome, &r.Read, &r.New, &r.Updated, &r.Ignored, &r.Error, &r.Started, &r.Finished); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
