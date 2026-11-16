package worktrees

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Erros da limpeza.
var (
	ErrCleanupOff  = errors.New("a limpeza está desligada para este projeto")
	ErrPlanChanged = errors.New("o que seria limpo mudou desde a simulação: simule de novo")
)

// Policy é a política de limpeza do projeto. Desligada por padrão.
type Policy struct {
	Enabled            bool `json:"enabled"`
	ArchiveIdleDays    int  `json:"archive_idle_days"`
	RemoveArchivedDays int  `json:"remove_archived_days"`
}

// DefaultPolicy é a política de um projeto que nunca a configurou.
var DefaultPolicy = Policy{Enabled: false, ArchiveIdleDays: 7, RemoveArchivedDays: 30}

// Ações do plano. Arquivar preserva tudo (checkpoint + ref de arquivo) e tira só o diretório;
// remover apaga as refs de arquivo e é irreversível. São ações distintas.
const (
	ActionArchive = "archive"
	ActionRemove  = "remove"
	ActionSkip    = "skip"
)

// Candidate é uma tentativa avaliada pela política.
type Candidate struct {
	AttemptID string `json:"attempt_id"`
	Branch    string `json:"branch"`
	Action    string `json:"action"`
	Reason    string `json:"reason"`
}

// Plan é a simulação (dry-run). Hash identifica exatamente o que será feito.
type Plan struct {
	Policy      Policy      `json:"policy"`
	Items       []Candidate `json:"items"`
	Hash        string      `json:"hash"`
	GeneratedAt time.Time   `json:"generated_at"`
}

// GetPolicy lê a política (padrão desligada).
func (m *Manager) GetPolicy(ctx context.Context, projectID string) (Policy, error) {
	p := DefaultPolicy
	err := m.DB.QueryRow(ctx, `SELECT enabled, archive_idle_days, remove_archived_days FROM worktree_cleanup_policy WHERE project_id::text = $1`, projectID).
		Scan(&p.Enabled, &p.ArchiveIdleDays, &p.RemoveArchivedDays)
	if err != nil {
		if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
			return DefaultPolicy, nil
		}
		return DefaultPolicy, persistence.MapError(err)
	}
	return p, nil
}

// SetPolicy grava a política.
func (m *Manager) SetPolicy(ctx context.Context, projectID, adminID string, p Policy) (Policy, error) {
	if p.ArchiveIdleDays < 1 || p.ArchiveIdleDays > 365 || p.RemoveArchivedDays < 1 || p.RemoveArchivedDays > 3650 {
		return Policy{}, fmt.Errorf("%w: dias fora do intervalo permitido", ErrInvalid)
	}
	_, err := m.DB.Exec(ctx, `INSERT INTO worktree_cleanup_policy (project_id, enabled, archive_idle_days, remove_archived_days) VALUES ($1,$2,$3,$4)
		ON CONFLICT (project_id) DO UPDATE SET enabled = $2, archive_idle_days = $3, remove_archived_days = $4, updated_at = now()`,
		projectID, p.Enabled, p.ArchiveIdleDays, p.RemoveArchivedDays)
	if err != nil {
		return Policy{}, persistence.MapError(err)
	}
	m.audit(ctx, projectID, adminID, "cleanup_policy_changed", map[string]any{"enabled": p.Enabled, "archive_idle_days": p.ArchiveIdleDays, "remove_archived_days": p.RemoveArchivedDays})
	return p, nil
}

func (m *Manager) audit(ctx context.Context, projectID, adminID, action string, detail map[string]any) {
	raw, _ := json.Marshal(detail)
	var admin any
	if adminID != "" {
		admin = adminID
	}
	_, _ = m.DB.Exec(ctx, `INSERT INTO project_changes (project_id, admin_id, action, detail) VALUES ($1,$2,$3,$4::jsonb)`, projectID, admin, action, string(raw))
}

// PlanCleanup simula a limpeza sem tocar em nada. base é a SHA da branch base do projeto: só serve
// para provar que um trabalho arquivado já foi incorporado (vazio = nada é removido).
func (m *Manager) PlanCleanup(ctx context.Context, projectID, base string, now time.Time) (Plan, error) {
	pol, err := m.GetPolicy(ctx, projectID)
	if err != nil {
		return Plan{}, err
	}
	plan := Plan{Policy: pol, Items: []Candidate{}, GeneratedAt: now.UTC()}
	if pol.Enabled {
		rows, err := m.DB.Query(ctx, `SELECT `+cols+`, created_at, released_at,
			COALESCE((SELECT max(created_at) FROM worktree_checkpoints c WHERE c.attempt_id = worktree_attempts.id), created_at)
			FROM worktree_attempts WHERE project_id::text = $1 ORDER BY created_at, id`, projectID)
		if err != nil {
			return Plan{}, persistence.MapError(err)
		}
		type row struct {
			a            Attempt
			created      time.Time
			released     *time.Time
			lastActivity time.Time
		}
		var all []row
		for rows.Next() {
			var r row
			var p, rs []byte
			if err := rows.Scan(&r.a.ID, &r.a.ProjectID, &r.a.FeatureID, &r.a.IssueKey, &r.a.Track, &r.a.Kind, &r.a.Attempt, &r.a.Branch, &r.a.Path,
				&r.a.BaseSHA, &p, &rs, &r.a.State, &r.a.WriterActive, &r.a.ArchiveRef, &r.created, &r.released, &r.lastActivity); err != nil {
				rows.Close()
				return Plan{}, err
			}
			all = append(all, r)
		}
		rows.Close()
		for _, r := range all {
			c := Candidate{AttemptID: r.a.ID, Branch: r.a.Branch, Action: ActionSkip}
			switch {
			case r.a.State == "active" && r.a.WriterActive:
				c.Reason = "há um processo escritor vivo"
			case r.a.State == "active" && now.Sub(r.lastActivity) < days(pol.ArchiveIdleDays):
				c.Reason = fmt.Sprintf("ativa há menos de %d dias", pol.ArchiveIdleDays)
			case r.a.State == "active":
				c.Action = ActionArchive
				c.Reason = fmt.Sprintf("sem atividade há mais de %d dias; tudo é preservado antes de tirar a pasta", pol.ArchiveIdleDays)
			case r.a.ArchiveRef == "":
				c.Reason = "nada arquivado para remover"
			case r.released == nil || now.Sub(*r.released) < days(pol.RemoveArchivedDays):
				c.Reason = fmt.Sprintf("arquivada há menos de %d dias", pol.RemoveArchivedDays)
			case base == "":
				c.Reason = "sem branch base para provar que o trabalho já foi incorporado"
			default:
				if _, err := git(ctx, m.Repo, nil, "", "merge-base", "--is-ancestor", r.a.ArchiveRef, base); err != nil {
					c.Reason = "tem commits que existem só aqui (não estão na branch base)"
				} else if m.onlyInCheckpoint(ctx, r.a) {
					c.Reason = "o último checkpoint tem arquivos que existem só aqui (não versionados)"
				} else {
					c.Action = ActionRemove
					c.Reason = "já incorporada na branch base; as refs de arquivo serão apagadas (irreversível)"
				}
			}
			plan.Items = append(plan.Items, c)
		}
	}
	h := sha256.New()
	canon, _ := json.Marshal(struct {
		P Policy
		I []Candidate
		B string
	}{pol, plan.Items, base})
	h.Write(canon)
	plan.Hash = hex.EncodeToString(h.Sum(nil))
	return plan, nil
}

func days(n int) time.Duration { return time.Duration(n) * 24 * time.Hour }

// CleanupResult resume o que foi feito.
type CleanupResult struct {
	Archived int `json:"archived"`
	Removed  int `json:"removed"`
	Failed   int `json:"failed"`
}

// ApplyCleanup refaz a simulação e só executa se ela tiver o mesmo hash que o usuário viu. Itens
// "skip" nunca são tocados. Arquivar usa Release (preserva e recusa com escritor ativo).
func (m *Manager) ApplyCleanup(ctx context.Context, projectID, adminID, base, hash string, now time.Time) (CleanupResult, error) {
	var res CleanupResult
	err := m.withLock(ctx, func() error {
		plan, err := m.PlanCleanup(ctx, projectID, base, now)
		if err != nil {
			return err
		}
		if !plan.Policy.Enabled {
			return ErrCleanupOff
		}
		if plan.Hash != hash {
			return ErrPlanChanged
		}
		for _, it := range plan.Items {
			switch it.Action {
			case ActionArchive:
				if err := m.release(ctx, it.AttemptID); err != nil {
					res.Failed++
				} else {
					res.Archived++
				}
			case ActionRemove:
				if err := m.removeArchive(ctx, it.AttemptID); err != nil {
					res.Failed++
				} else {
					res.Removed++
				}
			}
		}
		return nil
	})
	if err != nil {
		return res, err
	}
	m.audit(ctx, projectID, adminID, "cleanup_applied", map[string]any{"archived": res.Archived, "removed": res.Removed, "failed": res.Failed})
	return res, nil
}

// removeArchive apaga as refs de arquivo e de checkpoints de uma tentativa já liberada.
func (m *Manager) removeArchive(ctx context.Context, id string) error {
	a, err := m.Get(ctx, id)
	if err != nil {
		return err
	}
	if a.State != "released" || a.ArchiveRef == "" {
		return fmt.Errorf("%w: tentativa não está arquivada", ErrInvalid)
	}
	if _, err := git(ctx, m.Repo, nil, "", "update-ref", "-d", a.ArchiveRef); err != nil {
		return err
	}
	rows, err := m.DB.Query(ctx, `SELECT ref FROM worktree_checkpoints WHERE attempt_id::text = $1`, id)
	if err != nil {
		return persistence.MapError(err)
	}
	var refs []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err == nil {
			refs = append(refs, r)
		}
	}
	rows.Close()
	for _, r := range refs {
		_, _ = git(ctx, m.Repo, nil, "", "update-ref", "-d", r)
	}
	if _, err := m.DB.Exec(ctx, `DELETE FROM worktree_checkpoints WHERE attempt_id::text = $1`, id); err != nil {
		return persistence.MapError(err)
	}
	_, err = m.DB.Exec(ctx, `UPDATE worktree_attempts SET archive_ref = '' WHERE id::text = $1`, id)
	return persistence.MapError(err)
}

// onlyInCheckpoint diz se o último checkpoint guarda conteúdo que a ponta arquivada não tem
// (arquivos não versionados ou alterações sem commit): remover perderia trabalho só local.
func (m *Manager) onlyInCheckpoint(ctx context.Context, a Attempt) bool {
	var sha string
	if err := m.DB.QueryRow(ctx, `SELECT sha FROM worktree_checkpoints WHERE attempt_id::text = $1 ORDER BY seq DESC LIMIT 1`, a.ID).Scan(&sha); err != nil {
		return false
	}
	cp, e1 := git(ctx, m.Repo, nil, "", "rev-parse", sha+"^{tree}")
	tip, e2 := git(ctx, m.Repo, nil, "", "rev-parse", a.ArchiveRef+"^{tree}")
	return e1 != nil || e2 != nil || cp != tip
}
