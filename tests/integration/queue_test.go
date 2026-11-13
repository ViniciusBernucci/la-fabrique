package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/reqchange"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

type qEnv struct {
	e     *authEnv
	q     *queue.Queue
	plans *planning.Store
	cps   *checkpoints.Store
}

func newQEnv(t *testing.T) *qEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	cps := &checkpoints.Store{DB: e.db.Pool}
	return &qEnv{e: e, cps: cps, plans: &planning.Store{DB: e.db.Pool}, q: &queue.Queue{DB: e.db.Pool, Checkpoints: cps, Gates: map[string]queue.Gate{}}}
}

type qProj struct {
	*qEnv
	proj    string
	feature string
}

// project cria um projeto ATIVO com uma feature e o plano dado.
func (e *qEnv) project(t *testing.T, slug string, plan planning.Plan) qProj {
	t.Helper()
	ctx := context.Background()
	p := mustProject(t, e.e.db.Pool, slug)
	if _, err := e.e.db.Pool.Exec(ctx, `UPDATE projects SET state = 'active' WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	f, err := persistence.CreateFeature(ctx, e.e.db.Pool, p.ID, "Q-"+slug, "Feature "+slug)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.plans.Propose(ctx, p.ID, f.ID, plan); err != nil {
		t.Fatal(err)
	}
	return qProj{qEnv: e, proj: p.ID, feature: f.ID}
}

func independent(keys ...string) planning.Plan {
	var ts []planning.Task
	for _, k := range keys {
		ts = append(ts, plTask(k))
	}
	return planning.Plan{Summary: "independentes", Tasks: ts}
}

func (p qProj) item(t *testing.T, key string, prio int) queue.Item {
	t.Helper()
	it, err := p.q.Enqueue(context.Background(), p.proj, p.feature, key, prio)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func (p qProj) execution(t *testing.T) string {
	t.Helper()
	ex, err := persistence.CreateExecution(context.Background(), p.e.db.Pool, p.proj, p.feature)
	if err != nil {
		t.Fatal(err)
	}
	return ex.ID
}

func (p qProj) checkpoint(t *testing.T, execID string) string {
	t.Helper()
	body := "package main"
	h := sha256.Sum256([]byte(body))
	hash := hex.EncodeToString(h[:])
	files := []checkpoints.FileEntry{{Path: "main.go", Kind: checkpoints.KindFile, Mode: 0o644, Size: int64(len(body)), SHA256: hash}}
	snap := checkpoints.Snapshot{Manifest: checkpoints.Manifest{Version: 1, Files: files, TreeHash: checkpoints.TreeHash(files), TotalBytes: int64(len(body))}, Objects: map[string][]byte{hash: []byte(body)}}
	cp, err := p.cps.Save(context.Background(), snap, checkpoints.Input{ProjectID: p.proj, ExecutionID: execID, Label: "q"})
	if err != nil {
		t.Fatal(err)
	}
	return cp.ID
}

func notEligible(t *testing.T, err error, want ...string) {
	t.Helper()
	var ne *queue.NotEligibleError
	if !errors.As(err, &ne) {
		t.Fatalf("esperava inelegível: %v", err)
	}
	all := strings.Join(ne.Reasons, " | ")
	for _, w := range want {
		if !strings.Contains(all, w) {
			t.Fatalf("faltou %q em: %s", w, all)
		}
	}
}

// T10-02 — "Só tarefa elegível inicia."
func TestOnlyEligibleTasksStart(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "q-elig", basePlan()) // schema → api → tela
	items, err := e.q.EnqueuePlan(ctx, p.proj, p.feature, 2)
	if err != nil || len(items) != 3 {
		t.Fatalf("%+v %v", items, err)
	}
	if again, _ := e.q.EnqueuePlan(ctx, p.proj, p.feature, 2); len(again) != 3 {
		t.Fatal("enfileirar de novo é idempotente")
	}
	if all, _ := e.q.List(ctx, p.proj); len(all) != 3 {
		t.Fatalf("sem duplicar: %d", len(all))
	}
	by := map[string]queue.Item{}
	for _, it := range items {
		by[it.TaskKey] = it
	}
	cand, _ := e.q.Candidates(ctx)
	if len(cand) != 1 || cand[0].TaskKey != "schema" {
		t.Fatalf("só schema é elegível: %+v", cand)
	}
	_, err = e.q.Start(ctx, by["api"].ID, "")
	notEligible(t, err, `pré-requisito "schema"`)
	if _, err := e.q.Start(ctx, by["schema"].ID, ""); err != nil {
		t.Fatal(err)
	}
	_, err = e.q.Start(ctx, by["api"].ID, "")
	notEligible(t, err, "vaga do projeto", `pré-requisito "schema"`)
	if tasks, _ := e.plans.Tasks(ctx, p.proj, p.feature); tasks[2].Key != "tela" && tasks[0].State != planning.StateRunning {
		t.Fatalf("%+v", tasks)
	}
	if _, err := e.q.Complete(ctx, by["schema"].ID); err != nil {
		t.Fatal(err)
	}
	if cand, _ := e.q.Candidates(ctx); len(cand) != 1 || cand[0].TaskKey != "api" {
		t.Fatalf("%+v", cand)
	}
	// Projeto pausado não inicia nada.
	if _, err := e.e.db.Pool.Exec(ctx, `UPDATE projects SET state = 'paused' WHERE id = $1`, p.proj); err != nil {
		t.Fatal(err)
	}
	_, err = e.q.Start(ctx, by["api"].ID, "")
	notEligible(t, err, "não está ativo")
	if cand, _ := e.q.Candidates(ctx); len(cand) != 0 {
		t.Fatalf("%+v", cand)
	}
	// Tarefa que não existe no plano não entra na fila.
	if _, err := e.q.Enqueue(ctx, p.proj, p.feature, "fantasma", 2); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
}

func TestOneSlotPerProjectAndParallelismAcrossProjects(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	a := e.project(t, "q-par-a", independent("a1", "a2"))
	b := e.project(t, "q-par-b", independent("b1"))
	a1, a2 := a.item(t, "a1", 0), a.item(t, "a2", 1)
	b1 := b.item(t, "b1", 3)
	cand, _ := e.q.Candidates(ctx)
	if len(cand) != 2 || cand[0].TaskKey != "a1" || cand[1].TaskKey != "b1" {
		t.Fatalf("um candidato por projeto, na ordem global: %+v", cand)
	}
	if _, err := e.q.Start(ctx, a1.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Start(ctx, b1.ID, ""); err != nil {
		t.Fatalf("outro projeto roda em paralelo: %v", err)
	}
	_, err := e.q.Start(ctx, a2.ID, "")
	notEligible(t, err, "vaga do projeto")
}

func TestConcurrentWorkersNeverStartTwoTasksInOneProject(t *testing.T) {
	e := newQEnv(t)
	p := e.project(t, "q-race", independent("r1", "r2", "r3"))
	ids := []string{p.item(t, "r1", 0).ID, p.item(t, "r2", 0).ID, p.item(t, "r3", 0).ID}
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok := 0
	for i := 0; i < 9; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := e.q.Start(context.Background(), ids[i%3], ""); err == nil {
				mu.Lock()
				ok++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	var running int
	_ = e.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM queue_items WHERE project_id = $1 AND state = 'running'`, p.proj).Scan(&running)
	if ok != 1 || running != 1 {
		t.Fatalf("iniciadas %d, em execução %d", ok, running)
	}
}

// T10-02 — "PR corrigida ganha próxima vaga elegível."
func TestAFixedPRGetsTheNextEligibleSlot(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "q-prfix", independent("t1", "t2"))
	t1, t2 := p.item(t, "t1", 0), p.item(t, "t2", 1)
	if _, err := e.q.Start(ctx, t1.ID, ""); err != nil {
		t.Fatal(err)
	}
	fix, err := e.q.EnqueuePRFix(ctx, p.proj, p.feature, "t1", "https://github.com/o/r/pull/7", 4)
	if err != nil || fix.Kind != queue.KindPRFix {
		t.Fatalf("%+v %v", fix, err)
	}
	if again, _ := e.q.EnqueuePRFix(ctx, p.proj, p.feature, "t1", "https://github.com/o/r/pull/7", 4); again.ID != fix.ID {
		t.Fatal("a mesma correção não duplica")
	}
	if _, err := e.q.EnqueuePRFix(ctx, p.proj, p.feature, "t1", "", 4); err == nil {
		t.Fatal("PR obrigatória")
	}
	// A vaga está ocupada: ninguém passa na frente enquanto t1 roda.
	_, err = e.q.Start(ctx, fix.ID, "")
	notEligible(t, err, "vaga do projeto")
	if _, err := e.q.Complete(ctx, t1.ID); err != nil {
		t.Fatal(err)
	}
	// Na próxima vaga, a correção vem antes de t2 (mesmo com prioridade pior).
	cand, _ := e.q.Candidates(ctx)
	if len(cand) != 1 || cand[0].ID != fix.ID {
		t.Fatalf("%+v", cand)
	}
	_, err = e.q.Start(ctx, t2.ID, "")
	notEligible(t, err, "a vez é da tarefa", "correção de PR")
	if _, err := e.q.Start(ctx, fix.ID, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Complete(ctx, fix.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Start(ctx, t2.ID, ""); err != nil {
		t.Fatal(err)
	}
}

// T10-02 — "Estado bloqueado libera espaço após checkpoint seguro."
func TestABlockedItemReleasesTheSlotOnlyAfterASafeCheckpoint(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "q-block", independent("b1", "b2"))
	b1, b2 := p.item(t, "b1", 0), p.item(t, "b2", 1)
	exec := p.execution(t)
	if _, err := e.q.Start(ctx, b1.ID, exec); err != nil {
		t.Fatal(err)
	}
	// Sem checkpoint: espera segurando a vaga.
	w, err := e.q.Block(ctx, b1.ID, queue.WaitMerge, "https://github.com/o/r/pull/9", "", false)
	if err != nil || w.State != queue.StateWaiting || w.WaitReason != "merge" || !w.SlotHeld {
		t.Fatalf("%+v %v", w, err)
	}
	_, err = e.q.Start(ctx, b2.ID, "")
	notEligible(t, err, "vaga do projeto")
	// Checkpoint de outra execução, ou sandbox ainda ativo, não libera.
	other := p.execution(t)
	if _, err := e.q.ReleaseSlot(ctx, b1.ID, p.checkpoint(t, other), true); !errors.Is(err, queue.ErrNotSafe) {
		t.Fatalf("%v", err)
	}
	cp := p.checkpoint(t, exec)
	if _, err := e.q.ReleaseSlot(ctx, b1.ID, cp, false); !errors.Is(err, queue.ErrNotSafe) {
		t.Fatalf("sandbox não quieto: %v", err)
	}
	if _, err := e.q.ReleaseSlot(ctx, b1.ID, "00000000-0000-0000-0000-000000000000", true); !errors.Is(err, queue.ErrNotSafe) {
		t.Fatalf("checkpoint inexistente: %v", err)
	}
	// Checkpoint seguro: a vaga volta e a próxima tarefa elegível inicia.
	rel, err := e.q.ReleaseSlot(ctx, b1.ID, cp, true)
	if err != nil || rel.SlotHeld || rel.CheckpointID != cp || rel.State != queue.StateWaiting {
		t.Fatalf("%+v %v", rel, err)
	}
	if _, err := e.q.Start(ctx, b2.ID, ""); err != nil {
		t.Fatalf("%v", err)
	}
	// Com checkpoint seguro desde o bloqueio, a vaga é devolvida na hora.
	e2 := e.project(t, "q-block2", independent("c1"))
	c1 := e2.item(t, "c1", 0)
	ex2 := e2.execution(t)
	if _, err := e.q.Start(ctx, c1.ID, ex2); err != nil {
		t.Fatal(err)
	}
	w2, err := e.q.Block(ctx, c1.ID, queue.WaitProvider, "api", e2.checkpoint(t, ex2), true)
	if err != nil || w2.SlotHeld {
		t.Fatalf("%+v %v", w2, err)
	}
	// Checkpoint inválido no bloqueio é recusado e o item segue em execução.
	e3 := e.project(t, "q-block3", independent("d1"))
	d1 := e3.item(t, "d1", 0)
	ex3 := e3.execution(t)
	_, _ = e.q.Start(ctx, d1.ID, ex3)
	if _, err := e.q.Block(ctx, d1.ID, queue.WaitResources, "", e3.checkpoint(t, e3.execution(t)), true); !errors.Is(err, queue.ErrNotSafe) {
		t.Fatalf("%v", err)
	}
	if _, err := e.q.Block(ctx, d1.ID, "outro", "", "", false); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
}

func TestWaitingStatesReturnOnlyWhenTheirGateIsReleased(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "q-gates", independent("g1"))
	g1 := p.item(t, "g1", 0)
	exec := p.execution(t)
	merges := &fakeMerges{}
	e.q.Gates[queue.WaitDecision] = queue.DecisionGate{DB: e.e.db.Pool}
	e.q.Gates[queue.WaitMerge] = queue.MergeGate{Merges: merges}
	start := func() { // inicia e bloqueia com checkpoint seguro
		if _, err := e.q.Start(ctx, g1.ID, exec); err != nil {
			t.Fatal(err)
		}
	}
	// Decisão pendente.
	start()
	if _, err := e.q.Block(ctx, g1.ID, queue.WaitDecision, "", p.checkpoint(t, exec), true); err != nil {
		t.Fatal(err)
	}
	ex, _ := persistence.CreateExecution(ctx, e.e.db.Pool, p.proj, p.feature)
	_ = ex
	if _, err := e.e.db.Pool.Exec(ctx, `UPDATE executions SET state = 'waiting' WHERE id = $1`, exec); err != nil {
		t.Fatal(err)
	}
	dec, _, err := decisions.Create(ctx, e.e.db.Pool, security.NewRedactor(), newRequest(p.proj, exec, "g1", decisions.KindConflict))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Resume(ctx, g1.ID); !errors.Is(err, queue.ErrNotReady) {
		t.Fatalf("decisão pendente: %v", err)
	}
	var admin string
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT id FROM admins LIMIT 1`).Scan(&admin)
	if _, err := decisions.Resolve(ctx, e.e.db.Pool, admin, dec.ID, "seguir-a", "ok"); err != nil {
		t.Fatal(err)
	}
	back, err := e.q.Resume(ctx, g1.ID)
	if err != nil || back.State != queue.StateQueued || back.WaitReason != "" || back.SlotHeld {
		t.Fatalf("%+v %v", back, err)
	}
	if tasks, _ := e.plans.Tasks(ctx, p.proj, p.feature); tasks[0].State != planning.StatePending {
		t.Fatalf("%+v", tasks)
	}
	// Merge: só libera com PR integrada e SHA.
	start()
	if _, err := e.q.Block(ctx, g1.ID, queue.WaitMerge, "https://github.com/o/r/pull/3", p.checkpoint(t, exec), true); err != nil {
		t.Fatal(err)
	}
	merges.state = tasksync.MergeState{}
	if _, err := e.q.Resume(ctx, g1.ID); !errors.Is(err, queue.ErrNotReady) {
		t.Fatalf("%v", err)
	}
	merges.state = tasksync.MergeState{Closed: true}
	if _, err := e.q.Resume(ctx, g1.ID); !errors.Is(err, queue.ErrNotReady) {
		t.Fatalf("fechada sem merge: %v", err)
	}
	merges.state = tasksync.MergeState{Merged: true, SHA: "abc123"}
	if _, err := e.q.Resume(ctx, g1.ID); err != nil {
		t.Fatalf("%v", err)
	}
	// Motivo sem porta configurada: recusa (falha fechada).
	start()
	if _, err := e.q.Block(ctx, g1.ID, queue.WaitProvider, "", p.checkpoint(t, exec), true); err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Resume(ctx, g1.ID); !errors.Is(err, queue.ErrNoGate) {
		t.Fatalf("%v", err)
	}
	e.q.Gates[queue.WaitProvider] = gateFunc(func(queue.Item) (bool, string) { return true, "" })
	if _, err := e.q.Resume(ctx, g1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Resume(ctx, g1.ID); !errors.Is(err, queue.ErrBadState) {
		t.Fatalf("não está esperando: %v", err)
	}
}

type gateFunc func(queue.Item) (bool, string)

func (f gateFunc) Ready(_ context.Context, it queue.Item) (bool, string, error) {
	ok, why := f(it)
	return ok, why, nil
}

func TestAPendingRequirementChangeKeepsTheFeatureOutOfTheQueue(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "q-req", independent("r1"))
	r1 := p.item(t, "r1", 0)
	h := &reqchange.Handler{DB: e.e.db.Pool, Redactor: security.NewRedactor()}
	if _, err := h.Handle(ctx, p.proj, p.feature, rcReq()); err != nil {
		t.Fatal(err)
	}
	chg := rcReq()
	chg.Description = "Mudou o conteúdo"
	if _, err := h.Handle(ctx, p.proj, p.feature, chg); err != nil { // nada iniciado → replanejar
		t.Fatal(err)
	}
	_, err := e.q.Start(ctx, r1.ID, "")
	notEligible(t, err, "replanejamento ou decisão pendente")
	if cand, _ := e.q.Candidates(ctx); len(cand) != 0 {
		t.Fatalf("%+v", cand)
	}
	if _, err := h.CompleteReplan(ctx, p.proj, p.feature); err != nil {
		t.Fatal(err)
	}
	if _, err := e.q.Start(ctx, r1.ID, ""); err != nil {
		t.Fatalf("%v", err)
	}
	reasons, err := e.q.Explain(ctx, r1.ID)
	if err != nil || len(reasons) == 0 {
		t.Fatalf("em execução já não é elegível: %v %v", reasons, err)
	}
}
