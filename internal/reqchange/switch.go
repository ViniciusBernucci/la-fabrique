package reqchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
)

// Estados da troca.
const (
	SwitchValidated = "validated"
	SwitchRejected  = "rejected"
	SwitchActivated = "activated"
)

// Erros.
var (
	ErrSwitchState = errors.New("a troca não está validada")
	ErrSwitchStale = errors.New("o projeto mudou desde a validação da troca: valide de novo")
)

// Mapping liga a identidade remota ANTIGA de cada feature à NOVA, na nova origem.
type Mapping map[string]string

// SwitchRecord é uma tentativa de troca de origem.
type SwitchRecord struct {
	ID          string          `json:"id"`
	State       string          `json:"state"`
	Problems    []string        `json:"problems"`
	From        projects.Source `json:"from"`
	To          projects.Source `json:"to"`
	Mapping     Mapping         `json:"mapping"`
	CreatedAt   time.Time       `json:"created_at"`
	ActivatedAt *time.Time      `json:"activated_at,omitempty"`
}

// Switcher valida e ativa a troca de plataforma de origem.
type Switcher struct {
	DB  DB
	Now func() time.Time
}

func (s *Switcher) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

const swCols = `id, state, problems, from_source, to_source, mapping, created_at, activated_at`

func scanSwitch(row interface{ Scan(...any) error }) (SwitchRecord, error) {
	var r SwitchRecord
	var problems, from, to, mapping []byte
	if err := row.Scan(&r.ID, &r.State, &problems, &from, &to, &mapping, &r.CreatedAt, &r.ActivatedAt); err != nil {
		return r, persistence.MapError(err)
	}
	r.Problems = []string{}
	_ = json.Unmarshal(problems, &r.Problems)
	_ = json.Unmarshal(from, &r.From)
	_ = json.Unmarshal(to, &r.To)
	_ = json.Unmarshal(mapping, &r.Mapping)
	return r, nil
}

func sourceEq(a, b projects.Source) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// Plan valida a troca (sem alterar nada no projeto) e registra o resultado:
// "validated" quando pode ser ativada, "rejected" com os problemas quando não.
func (s *Switcher) Plan(ctx context.Context, projectID string, to projects.Source, mapping Mapping, tr integrations.Tracker) (SwitchRecord, error) {
	p, err := projects.Get(ctx, s.DB, projectID)
	if err != nil {
		return SwitchRecord{}, err
	}
	var problems []string
	bad := func(format string, a ...any) { problems = append(problems, fmt.Sprintf(format, a...)) }

	if p.State == projects.StateActive {
		bad("o projeto está ativo: suspenda-o antes de trocar a origem")
	}
	if to.Kind != "jira" && to.Kind != "clickup" {
		bad("informe a plataforma da nova origem (jira ou clickup)")
	}
	if to.ConnectionID == nil || strings.TrimSpace(to.Container) == "" {
		bad("informe a conexão e o projeto/lista da nova origem")
	}
	if sourceEq(p.Source, to) {
		bad("a nova origem é igual à atual")
	}
	if to.Kind == p.Source.Kind && to.Container == p.Source.Container && sameConn(to.ConnectionID, p.Source.ConnectionID) {
		bad("a nova origem aponta para a mesma plataforma, conexão e lista da atual")
	}

	rows, err := s.DB.Query(ctx, `SELECT id, external_id FROM features WHERE project_id = $1 ORDER BY external_id`, projectID)
	if err != nil {
		return SwitchRecord{}, persistence.MapError(err)
	}
	old := map[string]string{} // external_id → feature id
	for rows.Next() {
		var fid, ext string
		if err := rows.Scan(&fid, &ext); err != nil {
			rows.Close()
			return SwitchRecord{}, persistence.MapError(err)
		}
		old[ext] = fid
	}
	rows.Close()

	oldIDs := make([]string, 0, len(old))
	for k := range old {
		oldIDs = append(oldIDs, k)
	}
	sort.Strings(oldIDs)
	for _, k := range oldIDs {
		if strings.TrimSpace(mapping[k]) == "" {
			bad("a feature %s não tem identidade na nova origem: mapeie todas para preservar o histórico", k)
		}
	}
	keys := make([]string, 0, len(mapping))
	for k := range mapping {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	seen := map[string]string{}
	for _, k := range keys {
		nv := strings.TrimSpace(mapping[k])
		if _, ok := old[k]; !ok {
			bad("%q não é uma feature do projeto", k)
			continue
		}
		if nv == "" {
			continue
		}
		if prev, dup := seen[nv]; dup {
			bad("a identidade %q foi atribuída a %s e a %s: uma identidade remota por item", nv, prev, k)
		}
		seen[nv] = k
		if other, taken := old[nv]; taken && other != old[k] {
			bad("a identidade nova %q colide com a identidade atual de outra feature", nv)
		}
	}
	if len(problems) == 0 {
		for _, nv := range sortedKeys(seen) {
			if _, err := tr.GetItem(ctx, nv); err != nil {
				var ie *integrations.Error
				if errors.As(err, &ie) && ie.Kind == integrations.ErrNotFound {
					bad("o item %q não existe (ou não é visível) na nova origem", nv)
				} else {
					bad("não foi possível ler %q na nova origem: %s", nv, err.Error())
				}
			}
		}
	}
	state := SwitchValidated
	if len(problems) > 0 {
		state = SwitchRejected
	}
	fj, _ := json.Marshal(p.Source)
	tj, _ := json.Marshal(to)
	mj, _ := json.Marshal(mapping)
	pj, _ := json.Marshal(nonNilStr(problems))
	return scanSwitch(s.DB.QueryRow(ctx, `INSERT INTO source_switches (project_id, from_source, to_source, mapping, state, problems, created_at)
		VALUES ($1, $2::jsonb, $3::jsonb, $4::jsonb, $5, $6::jsonb, $7) RETURNING `+swCols,
		projectID, string(fj), string(tj), string(mj), state, string(pj), s.now()))
}

func sameConn(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func nonNilStr(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Activate aplica uma troca VALIDADA, com o projeto suspenso: a configuração da
// origem muda, as features passam à nova identidade e a antiga fica no histórico.
// Tudo numa transação; a origem do projeto continua única.
func (s *Switcher) Activate(ctx context.Context, adminID, projectID, switchID string) (projects.Project, error) {
	var out projects.Project
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		rec, err := scanSwitch(tx.QueryRow(ctx, `SELECT `+swCols+` FROM source_switches WHERE project_id = $1 AND id = $2 FOR UPDATE`, projectID, switchID))
		if err != nil {
			return err
		}
		if rec.State != SwitchValidated {
			return fmt.Errorf("%w (estado %s)", ErrSwitchState, rec.State)
		}
		var version int
		if err := tx.QueryRow(ctx, `SELECT version FROM projects WHERE id = $1 FOR UPDATE`, projectID).Scan(&version); err != nil {
			return persistence.MapError(err)
		}
		cur, err := projects.Get(ctx, tx, projectID)
		if err != nil {
			return err
		}
		if cur.State == projects.StateActive {
			return projects.ErrActiveLocked
		}
		if !sourceEq(cur.Source, rec.From) {
			return ErrSwitchStale
		}
		rows, err := tx.Query(ctx, `SELECT id, external_id FROM features WHERE project_id = $1 FOR UPDATE`, projectID)
		if err != nil {
			return persistence.MapError(err)
		}
		type feat struct{ id, ext string }
		var feats []feat
		for rows.Next() {
			var f feat
			if err := rows.Scan(&f.id, &f.ext); err != nil {
				rows.Close()
				return persistence.MapError(err)
			}
			feats = append(feats, f)
		}
		rows.Close()
		if len(feats) != len(rec.Mapping) {
			return ErrSwitchStale
		}
		for _, f := range feats {
			if _, ok := rec.Mapping[f.ext]; !ok {
				return ErrSwitchStale
			}
		}
		// Libera as identidades antigas antes de gravar as novas (a unicidade é por projeto).
		for _, f := range feats {
			if _, err := tx.Exec(ctx, `INSERT INTO feature_identity_history (project_id, feature_id, source_kind, remote_id, superseded_by, switch_id)
				VALUES ($1, $2, $3, $4, $5, $6)`, projectID, f.id, rec.From.Kind, f.ext, rec.Mapping[f.ext], rec.ID); err != nil {
				return persistence.MapError(err)
			}
			if _, err := tx.Exec(ctx, `UPDATE features SET external_id = '~' || id::text WHERE id = $1`, f.id); err != nil {
				return persistence.MapError(err)
			}
		}
		for _, f := range feats {
			if _, err := tx.Exec(ctx, `UPDATE features SET external_id = $2, updated_at = $3 WHERE id = $1`, f.id, strings.TrimSpace(rec.Mapping[f.ext]), s.now()); err != nil {
				return persistence.MapError(err)
			}
		}
		// A leitura do estado da origem antiga não vale para a nova; o histórico de sincronizações fica.
		if _, err := tx.Exec(ctx, `DELETE FROM source_items WHERE project_id = $1`, projectID); err != nil {
			return persistence.MapError(err)
		}
		to := rec.To
		updated, err := projects.Update(ctx, tx, projectID, version, projects.Patch{Source: &to})
		if err != nil {
			return err
		}
		out = updated
		if err := projects.RecordChange(ctx, tx, projectID, adminID, projects.ChangeSourceChanged, map[string]any{
			"from": projects.SourceDetail(rec.From), "to": projects.SourceDetail(rec.To), "switch": rec.ID, "features": len(feats),
		}); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE source_switches SET state = 'activated', activated_at = $2 WHERE id = $1`, rec.ID, s.now())
		return persistence.MapError(err)
	})
	return out, err
}

// Switches lista as tentativas de troca do projeto.
func (s *Switcher) Switches(ctx context.Context, projectID string) ([]SwitchRecord, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+swCols+` FROM source_switches WHERE project_id = $1 ORDER BY created_at DESC, id`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []SwitchRecord{}
	for rows.Next() {
		r, err := scanSwitch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}

// IdentityLink é uma identidade remota antiga de uma feature.
type IdentityLink struct {
	SourceKind   string    `json:"source_kind"`
	RemoteID     string    `json:"remote_id"`
	SupersededBy string    `json:"superseded_by"`
	SwitchID     string    `json:"switch_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// IdentityHistory lista os vínculos antigos da feature (do mais antigo ao mais novo).
func (s *Switcher) IdentityHistory(ctx context.Context, projectID, featureID string) ([]IdentityLink, error) {
	rows, err := s.DB.Query(ctx, `SELECT source_kind, remote_id, superseded_by, switch_id, created_at FROM feature_identity_history
		WHERE project_id = $1 AND feature_id = $2 ORDER BY id`, projectID, featureID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []IdentityLink{}
	for rows.Next() {
		var l IdentityLink
		if err := rows.Scan(&l.SourceKind, &l.RemoteID, &l.SupersededBy, &l.SwitchID, &l.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, l)
	}
	return out, persistence.MapError(rows.Err())
}
