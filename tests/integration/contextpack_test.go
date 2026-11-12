package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/contextpack"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type cxEnv struct {
	ckEnv
	cx  *contextpack.Store
	red *security.Redactor
}

func newCxEnv(t *testing.T, name string) cxEnv {
	t.Helper()
	c := newCkEnv(t, name)
	red := security.NewRedactor()
	red.Register("segredo-do-projeto-123")
	return cxEnv{ckEnv: c, red: red, cx: &contextpack.Store{DB: c.e.db.Pool, Checkpoints: c.store, Redactor: red}}
}

// checkpoint grava um checkpoint íntegro sem Docker (manifesto montado à mão).
func (c cxEnv) checkpoint(t *testing.T, label, body string) checkpoints.Checkpoint {
	t.Helper()
	h := sha256.Sum256([]byte(body))
	hash := hex.EncodeToString(h[:])
	files := []checkpoints.FileEntry{{Path: "main.go", Kind: checkpoints.KindFile, Mode: 0o644, Size: int64(len(body)), SHA256: hash}}
	snap := checkpoints.Snapshot{
		Manifest: checkpoints.Manifest{Version: 1, Files: files, TreeHash: checkpoints.TreeHash(files), TotalBytes: int64(len(body))},
		Objects:  map[string][]byte{hash: []byte(body)},
	}
	cp, err := c.store.Save(context.Background(), snap, c.input(label))
	if err != nil {
		t.Fatal(err)
	}
	return cp
}

func (c cxEnv) pack(t *testing.T, extra ...contextpack.Source) contextpack.Pack {
	t.Helper()
	src := append([]contextpack.Source{
		{Kind: contextpack.KindRequirement, Ref: "R1", Content: "Exportar CSV.", Required: true},
		{Kind: contextpack.KindCriterion, Ref: "C1", Content: "Menos de 2s.", Required: true},
		{Kind: contextpack.KindDecision, Ref: "D1", Content: "Usar pgx.", Priority: 3},
	}, extra...)
	p, err := contextpack.Build(src, contextpack.Limits{MaxTokens: 2000}, c.red)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func (c cxEnv) renew(t *testing.T, cpID string, p contextpack.Pack, ms, model, prev string, allow bool) (contextpack.Session, error) {
	return c.cx.Renew(context.Background(), contextpack.RenewInput{ProjectID: c.proj, ExecutionID: c.exec, Milestone: ms,
		CheckpointID: cpID, Pack: p, Model: model, PreviousModel: prev, ModelChangeAllowed: allow})
}

// T12-02 — "Nova sessão retoma com artefatos e contexto verificáveis."
func TestNewSessionResumesWithVerifiableArtifactsAndContext(t *testing.T) {
	c := newCxEnv(t, "cx-resume")
	ctx := context.Background()
	if _, err := c.cx.Resume(ctx, c.proj, c.exec); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("sem sessão: %v", err)
	}
	cp := c.checkpoint(t, "implementação", "package main // v1")
	p := c.pack(t, contextpack.Source{Kind: contextpack.KindFile, Ref: "cfg.env", Content: "K=segredo-do-projeto-123"})
	if strings.Contains(p.Content, "segredo-do-projeto-123") {
		t.Fatal("o segredo não pode entrar no pacote")
	}
	s, err := c.renew(t, cp.ID, p, contextpack.MilestoneSaturation, "modelo-a", "modelo-a", false)
	if err != nil || s.Seq != 1 || s.CheckpointID != cp.ID {
		t.Fatalf("%+v %v", s, err)
	}
	r, err := c.cx.Resume(ctx, c.proj, c.exec)
	if err != nil {
		t.Fatal(err)
	}
	if r.Pack.Content != p.Content || r.Pack.Manifest.ContentHash != p.Manifest.ContentHash || r.Checkpoint.TreeHash != cp.TreeHash ||
		r.Checkpoint.Refs["branch"] != "feature/ck" || r.Session.Milestone != "saturation" || r.NewerCheckpoint {
		t.Fatalf("%+v", r)
	}
	obj, err := c.store.Object(ctx, c.proj, r.Checkpoint.Manifest.Files[0].SHA256)
	if err != nil || string(obj) != "package main // v1" {
		t.Fatalf("artefato: %q %v", obj, err)
	}
	// Segunda renovação: sequência avança e um checkpoint posterior é sinalizado.
	cp2 := c.checkpoint(t, "testes", "package main // v2")
	s2, err := c.renew(t, cp2.ID, p, contextpack.MilestoneTests, "modelo-a", "modelo-a", false)
	if err != nil || s2.Seq != 2 {
		t.Fatalf("%+v %v", s2, err)
	}
	_ = c.checkpoint(t, "revisão", "package main // v3")
	r, _ = c.cx.Resume(ctx, c.proj, c.exec)
	if r.Session.Seq != 2 || !r.NewerCheckpoint {
		t.Fatalf("%+v", r.Session)
	}
}

func TestResumeRefusesTamperedPackOrCheckpoint(t *testing.T) {
	c := newCxEnv(t, "cx-tamper")
	ctx := context.Background()
	cp := c.checkpoint(t, "x", "package main")
	if _, err := c.renew(t, cp.ID, c.pack(t), contextpack.MilestoneReview, "m", "", false); err != nil {
		t.Fatal(err)
	}
	pool := c.e.db.Pool
	if _, err := pool.Exec(ctx, `UPDATE context_packs SET content = replace(content, 'CSV', 'XML') WHERE execution_id = $1`, c.exec); err != nil {
		t.Fatal(err)
	}
	if _, err := c.cx.Resume(ctx, c.proj, c.exec); !errors.Is(err, contextpack.ErrCorrupt) {
		t.Fatalf("pacote adulterado: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE context_packs SET content = replace(content, 'XML', 'CSV') WHERE execution_id = $1`, c.exec); err != nil {
		t.Fatal(err)
	}
	if _, err := c.cx.Resume(ctx, c.proj, c.exec); err != nil {
		t.Fatalf("restaurado o texto, volta a valer: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE checkpoints SET state = '{"status":"outro"}'::jsonb WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.cx.Resume(ctx, c.proj, c.exec); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("checkpoint adulterado: %v", err)
	}
}

func TestRenewRequiresIntactCheckpointOfTheSameExecution(t *testing.T) {
	c := newCxEnv(t, "cx-guard")
	other := newCxEnv(t, "cx-guard-other")
	ctx := context.Background()
	p := c.pack(t)
	// Checkpoint de outra execução/projeto.
	foreign := other.checkpoint(t, "alheio", "package other")
	if _, err := c.renew(t, foreign.ID, p, contextpack.MilestonePlanning, "m", "", false); err == nil {
		t.Fatal("checkpoint de outro projeto deve ser recusado")
	}
	// Checkpoint inexistente.
	if _, err := c.renew(t, "00000000-0000-0000-0000-000000000000", p, contextpack.MilestonePlanning, "m", "", false); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	// Marco inválido.
	cp := c.checkpoint(t, "ok", "package main")
	if _, err := c.renew(t, cp.ID, p, "qualquer", "m", "", false); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	// Pacote adulterado antes de gravar.
	bad := p
	bad.Content += "extra"
	if _, err := c.renew(t, cp.ID, bad, contextpack.MilestonePlanning, "m", "", false); !errors.Is(err, contextpack.ErrCorrupt) {
		t.Fatalf("%v", err)
	}
	// Checkpoint corrompido não serve de base.
	if _, err := c.e.db.Pool.Exec(ctx, `UPDATE checkpoints SET tree_hash = repeat('0', 64) WHERE id = $1`, cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.renew(t, cp.ID, p, contextpack.MilestonePlanning, "m", "", false); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("%v", err)
	}
	var n int
	_ = c.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM agent_sessions WHERE execution_id = $1`, c.exec).Scan(&n)
	if n != 0 {
		t.Fatalf("nenhuma sessão deveria ter sido gravada: %d", n)
	}
}

func TestModelChangeNeedsExplicitAuthorization(t *testing.T) {
	c := newCxEnv(t, "cx-model")
	cp := c.checkpoint(t, "x", "package main")
	p := c.pack(t)
	if _, err := c.renew(t, cp.ID, p, contextpack.MilestoneSessionChange, "modelo-b", "modelo-a", false); !errors.Is(err, contextpack.ErrModelChange) {
		t.Fatalf("troca sem autorização: %v", err)
	}
	s, err := c.renew(t, cp.ID, p, contextpack.MilestoneSessionChange, "modelo-b", "modelo-a", true)
	if err != nil || s.Model != "modelo-b" || s.PreviousModel != "modelo-a" {
		t.Fatalf("%+v %v", s, err)
	}
}

// T12-02 — "Testar estimativa sem métrica oficial": o método fica registrado.
func TestRenewRecordsEstimateWhenThereIsNoOfficialMetric(t *testing.T) {
	c := newCxEnv(t, "cx-estimate")
	cp := c.checkpoint(t, "x", "package main")
	d, err := contextpack.Decide(contextpack.Usage{Estimated: 70_000, Window: 100_000}, contextpack.Policy{})
	if err != nil || !d.Renew || d.Method != contextpack.UsageEstimate {
		t.Fatalf("%+v %v", d, err)
	}
	s, err := c.cx.Renew(context.Background(), contextpack.RenewInput{ProjectID: c.proj, ExecutionID: c.exec, Milestone: contextpack.MilestoneSaturation,
		CheckpointID: cp.ID, Pack: c.pack(t), Model: "m", Decision: &d})
	if err != nil || s.UsageMethod != "estimate" || s.UsageTokens != 87_500 || s.WindowTokens != 100_000 {
		t.Fatalf("%+v %v", s, err)
	}
}

func TestConcurrentRenewalsGetDistinctSequences(t *testing.T) {
	c := newCxEnv(t, "cx-race")
	cp := c.checkpoint(t, "x", "package main")
	p := c.pack(t)
	errs := make(chan error, 6)
	for i := 0; i < 6; i++ {
		go func() { _, err := c.renew(t, cp.ID, p, contextpack.MilestoneTests, "m", "", false); errs <- err }()
	}
	for i := 0; i < 6; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	var n, mx int
	_ = c.e.db.Pool.QueryRow(context.Background(), `SELECT count(*), max(seq) FROM agent_sessions WHERE execution_id = $1`, c.exec).Scan(&n, &mx)
	if n != 6 || mx != 6 {
		t.Fatalf("%d %d", n, mx)
	}
}
