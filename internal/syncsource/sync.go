package syncsource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Gatilhos.
const (
	TriggerManual    = "manual"
	TriggerPreflight = "preflight"
)

// Tipos de mudança.
const (
	KindNew        = "new"
	KindChanged    = "changed"
	KindGone       = "gone"
	KindReappeared = "reappeared"
	KindConflict   = "conflict"
)

const (
	pageLimit = 100
	maxPages  = 50
)

// Erros.
var (
	ErrNoConflict = errors.New("o item não tem conflito aberto")
	ErrNoItem     = errors.New("item não sincronizado")
)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Syncer sincroniza sob demanda.
type Syncer struct {
	DB  DB
	Now func() time.Time
}

func (s *Syncer) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Change é uma mudança detectada.
type Change struct {
	RemoteID      string                 `json:"remote_id"`
	Key           string                 `json:"key"`
	Kind          string                 `json:"kind"`
	Fields        map[string]FieldChange `json:"fields"`
	BeforeVersion string                 `json:"before_version"`
	AfterVersion  string                 `json:"after_version"`
	Detail        string                 `json:"detail,omitempty"`
}

// Result é o resultado de uma sincronização.
type Result struct {
	SyncID       string    `json:"sync_id"`
	Trigger      string    `json:"trigger"`
	Outcome      string    `json:"outcome"` // ok | failed
	Error        string    `json:"error,omitempty"`
	Retryable    bool      `json:"retryable,omitempty"`
	Read         int       `json:"read"`
	New          int       `json:"new"`
	Changed      int       `json:"changed"`
	Gone         int       `json:"gone"`
	Conflicts    int       `json:"conflicts"`
	Changes      []Change  `json:"changes"`
	SnapshotHash string    `json:"snapshot_hash,omitempty"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
}

// readAll lê a origem inteira (paginada) e deduplica pela identidade remota.
func readAll(ctx context.Context, tr integrations.Tracker, src Source) ([]Item, int, error) {
	seen := map[string]bool{}
	var out []Item
	read, cursor := 0, ""
	for page := 0; page < maxPages; page++ {
		pg, err := tr.ListItems(ctx, src.Container, integrations.ListOptions{Label: src.Marker, IncludeDone: true, Cursor: cursor, Limit: pageLimit})
		if err != nil {
			return nil, read, err
		}
		for _, it := range pg.Items {
			read++
			if it.ID == "" || seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			out = append(out, normalize(it, src, false))
		}
		if pg.NextCursor == "" {
			return out, read, nil
		}
		cursor = pg.NextCursor
	}
	return nil, read, fmt.Errorf("a origem tem mais de %d páginas: leitura incompleta", maxPages)
}

func failMessage(err error) (string, bool) {
	var e *integrations.Error
	if errors.As(err, &e) {
		return e.Error(), e.Retryable()
	}
	return "falha ao ler a origem", false
}

// Run lê a origem e registra o que mudou desde a última leitura.
func (s *Syncer) Run(ctx context.Context, projectID string, tr integrations.Tracker, src Source) (Result, error) {
	started := s.now()
	items, read, err := readAll(ctx, tr, src)
	if err != nil {
		return s.recordFailure(ctx, projectID, TriggerManual, src, started, read, err)
	}
	res := Result{Trigger: TriggerManual, Outcome: "ok", Read: read, StartedAt: started, Changes: []Change{}}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if err := lock(ctx, tx, projectID); err != nil {
			return err
		}
		stored, err := loadAll(ctx, tx, projectID)
		if err != nil {
			return err
		}
		now := s.now()
		present := map[string]bool{}
		for _, cur := range items {
			present[cur.RemoteID] = true
			prev, has := stored[cur.RemoteID]
			var p *Item
			if has {
				p = &prev
			}
			ch, next := reconcile(p, cur, now)
			if err := save(ctx, tx, projectID, next, p == nil, ch != nil, now); err != nil {
				return err
			}
			if ch != nil {
				res.Changes = append(res.Changes, *ch)
			}
		}
		for id, prev := range stored {
			if prev.State == "gone" || present[id] {
				continue
			}
			gone := prev
			gone.State = "gone"
			ch := &Change{RemoteID: id, Key: prev.Key, Kind: KindGone, Fields: map[string]FieldChange{}, BeforeVersion: prev.Version,
				Detail: "saiu da listagem da origem (removido, sem o marcador de elegibilidade ou movido)"}
			if err := save(ctx, tx, projectID, gone, false, true, now); err != nil {
				return err
			}
			res.Changes = append(res.Changes, *ch)
		}
		sortChanges(res.Changes)
		res.SnapshotHash = snapshotHash(items)
		return s.record(ctx, tx, projectID, src, &res)
	})
	return res, err
}

func sortChanges(cs []Change) {
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && cs[j].RemoteID < cs[j-1].RemoteID; j-- {
			cs[j], cs[j-1] = cs[j-1], cs[j]
		}
	}
}

func lock(ctx context.Context, tx pgx.Tx, projectID string) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 6))`, projectID)
	return persistence.MapError(err)
}

// reconcile compara o gravado com o lido. Devolve a mudança (nil = nada novo) e o item a gravar.
func reconcile(prev *Item, cur Item, now time.Time) (*Change, Item) {
	if prev == nil {
		return &Change{RemoteID: cur.RemoteID, Key: cur.Key, Kind: KindNew, Fields: map[string]FieldChange{}, AfterVersion: cur.Version}, cur
	}
	next := cur
	next.LocalPriority, next.Conflict = prev.LocalPriority, prev.Conflict
	fields := diff(*prev, cur)
	if prev.State == "gone" {
		return &Change{RemoteID: cur.RemoteID, Key: cur.Key, Kind: KindReappeared, Fields: fields, BeforeVersion: prev.Version, AfterVersion: cur.Version,
			Detail: "voltou à listagem da origem"}, next
	}
	if len(fields) == 0 {
		return nil, next
	}
	kind, detail := KindChanged, ""
	// Edição humana local não é sobrescrita em silêncio.
	if _, mp := fields["mapped_priority"]; mp && prev.LocalPriority != nil {
		kind = KindConflict
		next.Conflict = &Conflict{Field: "priority", Local: *prev.LocalPriority, Remote: cur.MappedPriority}
		detail = fmt.Sprintf("a origem mudou a prioridade, mas há edição local (%d); o valor local foi mantido até a decisão", *prev.LocalPriority)
	}
	return &Change{RemoteID: cur.RemoteID, Key: cur.Key, Kind: kind, Fields: fields, BeforeVersion: prev.Version, AfterVersion: cur.Version, Detail: detail}, next
}

const itemCols = `remote_id, item_key, title, description, status, category, priority, mapped_priority, labels, dependencies, is_subtask, eligible, version, content_hash, url, state, local_overrides, conflict`

func scanItem(row interface{ Scan(...any) error }) (Item, error) {
	var it Item
	var labels, deps, over, conflict []byte
	if err := row.Scan(&it.RemoteID, &it.Key, &it.Title, &it.Description, &it.Status, &it.Category, &it.Priority, &it.MappedPriority,
		&labels, &deps, &it.Subtask, &it.Eligible, &it.Version, &it.Hash, &it.URL, &it.State, &over, &conflict); err != nil {
		return it, persistence.MapError(err)
	}
	_ = json.Unmarshal(labels, &it.Labels)
	_ = json.Unmarshal(deps, &it.Dependencies)
	if it.Labels == nil {
		it.Labels = []string{}
	}
	if it.Dependencies == nil {
		it.Dependencies = []string{}
	}
	var o struct {
		P *int `json:"mapped_priority"`
	}
	_ = json.Unmarshal(over, &o)
	it.LocalPriority = o.P
	if len(conflict) > 0 && string(conflict) != "null" {
		var c Conflict
		if json.Unmarshal(conflict, &c) == nil {
			it.Conflict = &c
		}
	}
	return it, nil
}

func loadAll(ctx context.Context, q persistence.DBTX, projectID string) (map[string]Item, error) {
	rows, err := q.Query(ctx, `SELECT `+itemCols+` FROM source_items WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := map[string]Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out[it.RemoteID] = it
	}
	return out, persistence.MapError(rows.Err())
}

func save(ctx context.Context, q persistence.DBTX, projectID string, it Item, isNew, changed bool, now time.Time) error {
	labels, _ := json.Marshal(it.Labels)
	deps, _ := json.Marshal(it.Dependencies)
	over, _ := json.Marshal(map[string]*int{"mapped_priority": it.LocalPriority})
	if it.LocalPriority == nil {
		over = []byte(`{}`)
	}
	var conflict any
	if it.Conflict != nil {
		b, _ := json.Marshal(it.Conflict)
		conflict = string(b)
	}
	_, err := q.Exec(ctx, `
		INSERT INTO source_items (project_id, `+itemCols+`, first_seen_at, last_seen_at, last_changed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13, $14, $15, $16, $17, $18::jsonb, $19::jsonb, $20, $20, $20)
		ON CONFLICT (project_id, remote_id) DO UPDATE SET item_key = $3, title = $4, description = $5, status = $6, category = $7,
			priority = $8, mapped_priority = $9, labels = $10::jsonb, dependencies = $11::jsonb, is_subtask = $12, eligible = $13,
			version = $14, content_hash = $15, url = $16, state = $17, local_overrides = $18::jsonb, conflict = $19::jsonb,
			last_seen_at = $20, last_changed_at = CASE WHEN $21 THEN $20 ELSE source_items.last_changed_at END`,
		projectID, it.RemoteID, it.Key, it.Title, it.Description, it.Status, it.Category, it.Priority, it.MappedPriority,
		string(labels), string(deps), it.Subtask, it.Eligible, it.Version, it.Hash, it.URL, it.State, string(over), conflict, now, changed)
	return persistence.MapError(err)
}

// record grava a sincronização e as mudanças dentro da transação.
func (s *Syncer) record(ctx context.Context, tx pgx.Tx, projectID string, src Source, res *Result) error {
	res.FinishedAt = s.now()
	for _, c := range res.Changes {
		switch c.Kind {
		case KindNew:
			res.New++
		case KindChanged, KindReappeared:
			res.Changed++
		case KindGone:
			res.Gone++
		case KindConflict:
			res.Conflicts++
		}
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO source_syncs (project_id, trigger, source_kind, container, outcome, error, read_count, new_count, changed_count, gone_count, conflict_count, snapshot_hash, started_at, finished_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id`,
		projectID, res.Trigger, src.Kind, src.Container, res.Outcome, res.Error, res.Read, res.New, res.Changed, res.Gone, res.Conflicts,
		res.SnapshotHash, res.StartedAt, res.FinishedAt).Scan(&res.SyncID); err != nil {
		return persistence.MapError(err)
	}
	for _, c := range res.Changes {
		f, _ := json.Marshal(c.Fields)
		if _, err := tx.Exec(ctx, `INSERT INTO source_changes (sync_id, project_id, remote_id, item_key, kind, fields, before_version, after_version, detail)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)`, res.SyncID, projectID, c.RemoteID, c.Key, c.Kind, string(f), c.BeforeVersion, c.AfterVersion, c.Detail); err != nil {
			return persistence.MapError(err)
		}
	}
	return nil
}

// recordFailure registra a leitura que falhou, sem tocar em nenhum item.
func (s *Syncer) recordFailure(ctx context.Context, projectID, trigger string, src Source, started time.Time, read int, cause error) (Result, error) {
	msg, retry := failMessage(cause)
	res := Result{Trigger: trigger, Outcome: "failed", Error: msg, Retryable: retry, Read: read, StartedAt: started, Changes: []Change{}}
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error { return s.record(ctx, tx, projectID, src, &res) })
	return res, err
}

// Preflight é a leitura de UM item antes de iniciar uma execução.
type PreflightResult struct {
	Proceed bool     `json:"proceed"`
	Reason  string   `json:"reason"`
	Item    *Item    `json:"item,omitempty"`
	Sync    Result   `json:"sync"`
	Changes []Change `json:"changes"`
}

// Preflight lê só o item que vai executar, registra a mudança (uma vez) e decide
// se a execução pode começar: item existente, elegível e sem conflito aberto.
func (s *Syncer) Preflight(ctx context.Context, projectID string, tr integrations.Tracker, src Source, remoteID string) (PreflightResult, error) {
	if remoteID == "" {
		return PreflightResult{}, &FieldError{"remote_id", "informe o item"}
	}
	started := s.now()
	raw, err := tr.GetItem(ctx, remoteID)
	missing := false
	if err != nil {
		var e *integrations.Error
		if errors.As(err, &e) && e.Kind == integrations.ErrNotFound {
			missing = true
		} else {
			r, rerr := s.recordFailure(ctx, projectID, TriggerPreflight, src, started, 0, err)
			return PreflightResult{Proceed: false, Reason: "não foi possível ler o item na origem: " + r.Error, Sync: r, Changes: []Change{}}, rerr
		}
	}
	out := PreflightResult{Changes: []Change{}}
	res := Result{Trigger: TriggerPreflight, Outcome: "ok", Read: 1, StartedAt: started, Changes: []Change{}}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if err := lock(ctx, tx, projectID); err != nil {
			return err
		}
		stored, err := loadAll(ctx, tx, projectID)
		if err != nil {
			return err
		}
		now := s.now()
		prev, has := stored[remoteID]
		var p *Item
		if has {
			p = &prev
		}
		var final Item
		if missing {
			res.Read = 0
			if p == nil {
				out.Reason = "o item não existe na origem"
				return s.record(ctx, tx, projectID, src, &res)
			}
			final = prev
			if prev.State != "gone" {
				final.State = "gone"
				ch := Change{RemoteID: remoteID, Key: prev.Key, Kind: KindGone, Fields: map[string]FieldChange{}, BeforeVersion: prev.Version, Detail: "o item não existe mais na origem"}
				if err := save(ctx, tx, projectID, final, false, true, now); err != nil {
					return err
				}
				res.Changes = append(res.Changes, ch)
			}
			out.Reason = "o item não existe mais na origem"
		} else {
			cur := normalize(raw, src, true)
			ch, next := reconcile(p, cur, now)
			if err := save(ctx, tx, projectID, next, p == nil, ch != nil, now); err != nil {
				return err
			}
			if ch != nil {
				res.Changes = append(res.Changes, *ch)
			}
			final = next
		}
		if err := s.record(ctx, tx, projectID, src, &res); err != nil {
			return err
		}
		out.Item = &final
		out.Proceed, out.Reason = decide(final, missing, out.Reason)
		return nil
	})
	out.Sync, out.Changes = res, res.Changes
	return out, err
}

func decide(it Item, missing bool, reason string) (bool, string) {
	switch {
	case missing || it.State == "gone":
		return false, "o item não existe mais na origem"
	case it.Conflict != nil:
		return false, fmt.Sprintf("conflito de prioridade pendente (local %d × origem): decida antes de executar", it.Conflict.Local)
	case it.Subtask:
		return false, "o item agora é uma subtarefa"
	case it.Category == integrations.CategoryDone:
		return false, "o item já está concluído na origem"
	case !it.Eligible:
		return false, fmt.Sprintf("o item não está mais elegível (estado %q ou sem o marcador)", it.Status)
	}
	return true, "item elegível e sem conflito"
}

// Last devolve a sincronização mais recente.
func (s *Syncer) Last(ctx context.Context, projectID string) (Result, error) {
	var r Result
	err := s.DB.QueryRow(ctx, `SELECT id, trigger, outcome, error, read_count, new_count, changed_count, gone_count, conflict_count, snapshot_hash, started_at, finished_at
		FROM source_syncs WHERE project_id = $1 ORDER BY seq DESC LIMIT 1`, projectID).
		Scan(&r.SyncID, &r.Trigger, &r.Outcome, &r.Error, &r.Read, &r.New, &r.Changed, &r.Gone, &r.Conflicts, &r.SnapshotHash, &r.StartedAt, &r.FinishedAt)
	if err != nil {
		return Result{}, persistence.MapError(err)
	}
	r.Changes, err = s.Changes(ctx, projectID, r.SyncID)
	return r, err
}

// Changes lista as mudanças de uma sincronização.
func (s *Syncer) Changes(ctx context.Context, projectID, syncID string) ([]Change, error) {
	rows, err := s.DB.Query(ctx, `SELECT remote_id, item_key, kind, fields, before_version, after_version, detail FROM source_changes
		WHERE project_id = $1 AND sync_id = $2 ORDER BY id`, projectID, syncID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Change{}
	for rows.Next() {
		var c Change
		var f []byte
		if err := rows.Scan(&c.RemoteID, &c.Key, &c.Kind, &f, &c.BeforeVersion, &c.AfterVersion, &c.Detail); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(f, &c.Fields)
		if c.Fields == nil {
			c.Fields = map[string]FieldChange{}
		}
		out = append(out, c)
	}
	return out, persistence.MapError(rows.Err())
}

// Items lista os itens gravados do projeto.
func (s *Syncer) Items(ctx context.Context, projectID string) ([]Item, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+itemCols+` FROM source_items WHERE project_id = $1 ORDER BY item_key, remote_id`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, persistence.MapError(rows.Err())
}

// SetLocalPriority registra a edição humana da prioridade (nil remove). Prevalece
// sobre a origem; uma mudança remota no mesmo campo abre conflito.
func (s *Syncer) SetLocalPriority(ctx context.Context, projectID, remoteID string, priority *int) (Item, error) {
	if priority != nil && (*priority < 0 || *priority > 4) {
		return Item{}, &FieldError{"priority", "a prioridade vai de 0 a 4"}
	}
	var out Item
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if err := lock(ctx, tx, projectID); err != nil {
			return err
		}
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+itemCols+` FROM source_items WHERE project_id = $1 AND remote_id = $2 FOR UPDATE`, projectID, remoteID))
		if errors.Is(err, persistence.ErrNotFound) {
			return ErrNoItem
		}
		if err != nil {
			return err
		}
		it.LocalPriority = priority
		if priority == nil {
			it.Conflict = nil
		}
		if err := save(ctx, tx, projectID, it, false, false, s.now()); err != nil {
			return err
		}
		out = it
		return nil
	})
	return out, err
}

// ResolveConflict encerra o conflito: "keep_local" mantém a edição humana;
// "use_remote" a descarta e adota a prioridade da origem.
func (s *Syncer) ResolveConflict(ctx context.Context, projectID, remoteID, choice string) (Item, error) {
	if choice != "keep_local" && choice != "use_remote" {
		return Item{}, &FieldError{"choice", "use keep_local ou use_remote"}
	}
	var out Item
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if err := lock(ctx, tx, projectID); err != nil {
			return err
		}
		it, err := scanItem(tx.QueryRow(ctx, `SELECT `+itemCols+` FROM source_items WHERE project_id = $1 AND remote_id = $2 FOR UPDATE`, projectID, remoteID))
		if errors.Is(err, persistence.ErrNotFound) {
			return ErrNoItem
		}
		if err != nil {
			return err
		}
		if it.Conflict == nil {
			return ErrNoConflict
		}
		if choice == "use_remote" {
			it.LocalPriority = nil
		}
		it.Conflict = nil
		if err := save(ctx, tx, projectID, it, false, true, s.now()); err != nil {
			return err
		}
		out = it
		return nil
	})
	return out, err
}
