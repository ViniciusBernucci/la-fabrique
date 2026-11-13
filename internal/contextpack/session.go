package contextpack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Marcos que exigem checkpoint ao renovar/retomar.
const (
	MilestonePlanning       = "planning"
	MilestoneImplementation = "implementation"
	MilestoneTests          = "tests"
	MilestoneReview         = "review"
	MilestoneSuspension     = "suspension"
	MilestoneSessionChange  = "session_change"
	MilestoneSaturation     = "saturation"
)

var milestones = map[string]bool{MilestonePlanning: true, MilestoneImplementation: true, MilestoneTests: true,
	MilestoneReview: true, MilestoneSuspension: true, MilestoneSessionChange: true, MilestoneSaturation: true}

// ErrModelChange: a troca de modelo exige autorização explícita.
var ErrModelChange = errors.New("troca de modelo sem autorização")

// DB é o que o Store precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store grava pacotes e sessões e retoma a partir deles.
type Store struct {
	DB          DB
	Checkpoints *checkpoints.Store
	Redactor    *security.Redactor
}

// Session liga um pacote a um checkpoint íntegro.
type Session struct {
	ID            string    `json:"id"`
	ProjectID     string    `json:"project_id"`
	ExecutionID   string    `json:"execution_id"`
	Seq           int       `json:"seq"`
	Milestone     string    `json:"milestone"`
	CheckpointID  string    `json:"checkpoint_id"`
	PackID        string    `json:"pack_id"`
	Model         string    `json:"model"`
	PreviousModel string    `json:"previous_model,omitempty"`
	UsageMethod   string    `json:"usage_method"`
	UsageTokens   int64     `json:"usage_tokens"`
	WindowTokens  int64     `json:"window_tokens"`
	CreatedAt     time.Time `json:"created_at"`
}

// RenewInput é o pedido de uma nova sessão.
type RenewInput struct {
	ProjectID          string
	ExecutionID        string
	Milestone          string
	CheckpointID       string // criado antes pelo chamador (CaptureAndRelease)
	Pack               Pack
	Model              string
	PreviousModel      string
	ModelChangeAllowed bool
	Decision           *Decision // opcional: por que renovou
}

// Renew grava o pacote e a nova sessão numa única transação, depois de conferir
// o checkpoint (hashes) e o pacote (hash do texto, tokens e segredos).
func (s *Store) Renew(ctx context.Context, in RenewInput) (Session, error) {
	if !milestones[in.Milestone] {
		return Session{}, &FieldError{"milestone", fmt.Sprintf("marco desconhecido: %q", in.Milestone)}
	}
	if in.PreviousModel != "" && in.Model != in.PreviousModel && !in.ModelChangeAllowed {
		return Session{}, fmt.Errorf("%w: %q → %q", ErrModelChange, in.PreviousModel, in.Model)
	}
	if err := in.Pack.Verify(s.Redactor); err != nil {
		return Session{}, err
	}
	cp, err := s.Checkpoints.Get(ctx, in.ProjectID, in.CheckpointID) // reconfere os hashes
	if err != nil {
		return Session{}, err
	}
	if cp.ExecutionID != in.ExecutionID {
		return Session{}, &FieldError{"checkpoint_id", "o checkpoint é de outra execução"}
	}
	manifest, _ := json.Marshal(in.Pack.Manifest)
	method, used, window := UsageNone, int64(0), int64(0)
	if in.Decision != nil {
		method, used, window = in.Decision.Method, in.Decision.Used, in.Decision.Window
	}
	var out Session
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 13))`, in.ExecutionID); err != nil {
			return persistence.MapError(err)
		}
		var seq int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(max(seq), 0) + 1 FROM agent_sessions WHERE execution_id = $1`, in.ExecutionID).Scan(&seq); err != nil {
			return persistence.MapError(err)
		}
		var packID string
		if err := tx.QueryRow(ctx, `
			INSERT INTO context_packs (project_id, execution_id, seq, manifest, manifest_hash, content, content_hash, estimated_tokens, limit_tokens)
			VALUES ($1, $2, $3, $4::jsonb, $5, $6, $7, $8, $9) RETURNING id`,
			in.ProjectID, in.ExecutionID, seq, string(manifest), in.Pack.Manifest.Hash(), in.Pack.Content,
			in.Pack.Manifest.ContentHash, in.Pack.Manifest.EstimatedTokens, in.Pack.Manifest.LimitTokens).Scan(&packID); err != nil {
			return persistence.MapError(err)
		}
		err := tx.QueryRow(ctx, `
			INSERT INTO agent_sessions (project_id, execution_id, seq, milestone, checkpoint_id, pack_id, model, previous_model, usage_method, usage_tokens, window_tokens)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id, created_at`,
			in.ProjectID, in.ExecutionID, seq, in.Milestone, cp.ID, packID, in.Model, in.PreviousModel, method, used, window).Scan(&out.ID, &out.CreatedAt)
		if err != nil {
			return persistence.MapError(err)
		}
		out = Session{ID: out.ID, ProjectID: in.ProjectID, ExecutionID: in.ExecutionID, Seq: seq, Milestone: in.Milestone,
			CheckpointID: cp.ID, PackID: packID, Model: in.Model, PreviousModel: in.PreviousModel,
			UsageMethod: method, UsageTokens: used, WindowTokens: window, CreatedAt: out.CreatedAt}
		return nil
	})
	return out, err
}

// Resumption é o que a nova sessão recebe: o último estado verificável.
type Resumption struct {
	Session    Session
	Checkpoint checkpoints.Checkpoint
	Pack       Pack
	// NewerCheckpoint: existe checkpoint posterior ao da sessão; quem retoma decide
	// se usa o mais novo (o último íntegro governa a restauração).
	NewerCheckpoint bool
}

// Resume carrega a sessão mais recente da execução e reconfere o checkpoint e o
// pacote. Qualquer hash divergente aborta com ErrCorrupt (nada é usado às cegas).
func (s *Store) Resume(ctx context.Context, projectID, executionID string) (Resumption, error) {
	var r Resumption
	var manifest []byte
	var manifestHash, contentHash string
	err := s.DB.QueryRow(ctx, `
		SELECT s.id, s.project_id, s.execution_id, s.seq, s.milestone, s.checkpoint_id, s.pack_id, s.model, s.previous_model,
		       s.usage_method, s.usage_tokens, s.window_tokens, s.created_at,
		       p.manifest, p.manifest_hash, p.content, p.content_hash
		FROM agent_sessions s JOIN context_packs p ON p.id = s.pack_id
		WHERE s.execution_id = $1 AND s.project_id = $2 ORDER BY s.seq DESC LIMIT 1`, executionID, projectID).
		Scan(&r.Session.ID, &r.Session.ProjectID, &r.Session.ExecutionID, &r.Session.Seq, &r.Session.Milestone,
			&r.Session.CheckpointID, &r.Session.PackID, &r.Session.Model, &r.Session.PreviousModel,
			&r.Session.UsageMethod, &r.Session.UsageTokens, &r.Session.WindowTokens, &r.Session.CreatedAt,
			&manifest, &manifestHash, &r.Pack.Content, &contentHash)
	if err != nil {
		return Resumption{}, persistence.MapError(err)
	}
	if err := json.Unmarshal(manifest, &r.Pack.Manifest); err != nil {
		return Resumption{}, fmt.Errorf("%w: manifesto ilegível", ErrCorrupt)
	}
	if r.Pack.Manifest.Hash() != manifestHash || r.Pack.Manifest.ContentHash != contentHash {
		return Resumption{}, fmt.Errorf("%w: manifesto", ErrCorrupt)
	}
	if err := r.Pack.Verify(s.Redactor); err != nil {
		return Resumption{}, err
	}
	if r.Checkpoint, err = s.Checkpoints.Get(ctx, projectID, r.Session.CheckpointID); err != nil {
		return Resumption{}, err
	}
	if latest, err := s.Checkpoints.Latest(ctx, projectID, executionID); err == nil && latest.Seq > r.Checkpoint.Seq {
		r.NewerCheckpoint = true
	}
	return r, nil
}
