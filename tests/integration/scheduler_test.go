package integration

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
)

type schedEnv struct {
	*qEnv
	now time.Time
	mu  sync.Mutex
}

func newSchedEnv(t *testing.T) *schedEnv {
	t.Helper()
	s := &schedEnv{qEnv: newQEnv(t), now: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)}
	s.q.Now = func() time.Time { s.mu.Lock(); defer s.mu.Unlock(); return s.now }
	return s
}

func (s *schedEnv) advance(d time.Duration) { s.mu.Lock(); s.now = s.now.Add(d); s.mu.Unlock() }

func (s *schedEnv) claim(t *testing.T, owner string) *queue.Item {
	t.Helper()
	it, err := s.q.Claim(context.Background(), owner, "", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func fenced(it *queue.Item) context.Context {
	return queue.WithFence(context.Background(), queue.Fence{Owner: it.LeaseOwner, Generation: it.Generation})
}

func (s *schedEnv) counts(t *testing.T) (running, maxPerProject int) {
	t.Helper()
	ctx := context.Background()
	_ = s.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM queue_items WHERE state = 'running'`).Scan(&running)
	_ = s.e.db.Pool.QueryRow(ctx, `SELECT COALESCE(max(n), 0) FROM (SELECT count(*) n FROM queue_items WHERE state = 'running' GROUP BY project_id) x`).Scan(&maxPerProject)
	return
}

// T10-03 — "Com jobs concorrentes, um por projeto e total configurado."
func TestConcurrentClaimsRespectOnePerProjectAndTheGlobalLimit(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	if _, err := s.q.SetSettings(ctx, 3, 30); err != nil {
		t.Fatal(err)
	}
	for i, slug := range []string{"s-a", "s-b", "s-c", "s-d", "s-e"} {
		p := s.project(t, slug, independent("x", "y"))
		p.item(t, "x", 2)
		p.item(t, "y", 2)
		_ = i
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var got []*queue.Item
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			it, err := s.q.Claim(ctx, "worker-"+string(rune('a'+i)), "", time.Minute)
			if err != nil {
				t.Error(err)
				return
			}
			if it != nil {
				mu.Lock()
				got = append(got, it)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	projects := map[string]bool{}
	for _, it := range got {
		projects[it.ProjectID] = true
		if it.Generation != 1 || it.LeaseOwner == "" || it.LeaseExpires == nil {
			t.Fatalf("%+v", it)
		}
	}
	running, perProject := s.counts(t)
	if len(got) != 3 || len(projects) != 3 || running != 3 || perProject != 1 {
		t.Fatalf("reclamadas %d em %d projetos; em execução %d; máx. por projeto %d", len(got), len(projects), running, perProject)
	}
	if extra := s.claim(t, "mais-um"); extra != nil {
		t.Fatalf("o limite global já foi atingido: %+v", extra)
	}
	// Concluir libera uma vaga global; o projeto que concluiu pode ser escolhido de novo, mas nunca dois por projeto.
	if _, err := s.q.Complete(fenced(got[0]), got[0].ID); err != nil {
		t.Fatal(err)
	}
	if next := s.claim(t, "depois"); next == nil {
		t.Fatal("sobrou capacidade")
	}
	if running, perProject := s.counts(t); running != 3 || perProject != 1 {
		t.Fatalf("%d %d", running, perProject)
	}
}

func TestGlobalLimitPersistsAndNeverPreemptsRunningWork(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	if _, err := s.q.SetSettings(ctx, 2, 30); err != nil {
		t.Fatal(err)
	}
	var items []*queue.Item
	for _, slug := range []string{"l-a", "l-b", "l-c"} {
		p := s.project(t, slug, independent("x"))
		p.item(t, "x", 2)
	}
	for i := 0; i < 2; i++ {
		items = append(items, s.claim(t, "w"))
	}
	if s.claim(t, "w3") != nil {
		t.Fatal("limite 2")
	}
	// Baixar o limite não derruba quem já roda; só bloqueia novas reclamações.
	st, err := s.q.SetSettings(ctx, 1, 30)
	if err != nil || st.GlobalLimit != 1 {
		t.Fatalf("%+v %v", st, err)
	}
	if running, _ := s.counts(t); running != 2 {
		t.Fatalf("ninguém é preemptado: %d", running)
	}
	if s.claim(t, "w4") != nil {
		t.Fatal("acima do novo limite")
	}
	_, _ = s.q.Complete(fenced(items[0]), items[0].ID)
	if s.claim(t, "w5") != nil {
		t.Fatal("ainda há 1 em execução e o limite é 1")
	}
	_, _ = s.q.Complete(fenced(items[1]), items[1].ID)
	if s.claim(t, "w6") == nil {
		t.Fatal("agora cabe 1")
	}
	// Persistente: outra instância lê a mesma configuração.
	other := &queue.Queue{DB: s.e.db.Pool}
	if got, _ := other.Settings(ctx); got.GlobalLimit != 1 || got.AgingMinutes != 30 {
		t.Fatalf("%+v", got)
	}
	for _, bad := range [][2]int{{0, 30}, {65, 30}, {2, 0}, {2, 100000}} {
		if _, err := s.q.SetSettings(ctx, bad[0], bad[1]); err == nil {
			t.Errorf("%v deveria ser recusado", bad)
		}
	}
}

// T10-03 — "Escritor obsoleto não grava."
func TestAStaleWriterCannotWrite(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	p := s.project(t, "stale", independent("w"))
	it := p.item(t, "w", 2)
	exec := p.execution(t)

	a, err := s.q.Claim(ctx, "worker-A", exec, time.Minute)
	if err != nil || a == nil || a.ID != it.ID || a.Generation != 1 {
		t.Fatalf("%+v %v", a, err)
	}
	fa := queue.Fence{Owner: "worker-A", Generation: a.Generation}
	// O dono grava normalmente com a guarda (checkpoint) e renova a lease.
	guarded := &checkpoints.Store{DB: s.e.db.Pool, Guard: s.q.Guard(a.ID, fa)}
	if _, err := guarded.Save(ctx, p.snapshot(t, "v1"), checkpoints.Input{ProjectID: p.proj, ExecutionID: exec, Label: "a1"}); err != nil {
		t.Fatalf("o dono grava: %v", err)
	}
	if _, err := s.q.Heartbeat(ctx, a.ID, fa, time.Minute); err != nil {
		t.Fatal(err)
	}
	// Sem token em item reclamado pelo scheduler: recusado.
	if _, err := s.q.Complete(ctx, a.ID); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("sem token: %v", err)
	}
	// O worker A some: a lease vence, a vaga volta e B a reclama com geração nova.
	s.advance(2 * time.Minute)
	expired, err := s.q.ExpireLeases(ctx)
	if err != nil || len(expired) != 1 || expired[0].ID != a.ID {
		t.Fatalf("%+v %v", expired, err)
	}
	if tasks, _ := p.plans.Tasks(ctx, p.proj, p.feature); tasks[0].State != "pending" {
		t.Fatalf("%+v", tasks)
	}
	b, err := s.q.Claim(ctx, "worker-B", exec, time.Minute)
	if err != nil || b == nil || b.Generation != 2 || b.LeaseOwner != "worker-B" {
		t.Fatalf("%+v %v", b, err)
	}
	fb := queue.Fence{Owner: "worker-B", Generation: b.Generation}

	// O antigo dono volta e tenta de tudo: nada é gravado.
	if _, err := s.q.Heartbeat(ctx, a.ID, fa, time.Minute); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("heartbeat obsoleto: %v", err)
	}
	var before int
	_ = s.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM checkpoints WHERE execution_id = $1`, exec).Scan(&before)
	if _, err := guarded.Save(ctx, p.snapshot(t, "v-velho"), checkpoints.Input{ProjectID: p.proj, ExecutionID: exec, Label: "a2"}); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("checkpoint obsoleto: %v", err)
	}
	var afterStale int
	_ = s.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM checkpoints WHERE execution_id = $1`, exec).Scan(&afterStale)
	if afterStale != before {
		t.Fatalf("o escritor obsoleto gravou: %d → %d", before, afterStale)
	}
	if _, err := s.q.Complete(queue.WithFence(ctx, fa), a.ID); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("conclusão obsoleta: %v", err)
	}
	if _, err := s.q.Block(queue.WithFence(ctx, fa), a.ID, queue.WaitMerge, "", "", false); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("bloqueio obsoleto: %v", err)
	}
	// O dono atual segue gravando e conclui.
	guardedB := &checkpoints.Store{DB: s.e.db.Pool, Guard: s.q.Guard(b.ID, fb)}
	if _, err := guardedB.Save(ctx, p.snapshot(t, "v2"), checkpoints.Input{ProjectID: p.proj, ExecutionID: exec, Label: "b1"}); err != nil {
		t.Fatalf("o dono atual grava: %v", err)
	}
	// Checkpoint de outra execução também é recusado, mesmo para o dono.
	if _, err := guardedB.Save(ctx, p.snapshot(t, "v3"), checkpoints.Input{ProjectID: p.proj, ExecutionID: p.execution(t), Label: "b2"}); !errors.Is(err, queue.ErrStale) {
		t.Fatalf("outra execução: %v", err)
	}
	if done, err := s.q.Complete(queue.WithFence(ctx, fb), b.ID); err != nil || done.State != queue.StateDone || done.LeaseOwner != "" {
		t.Fatalf("%+v %v", done, err)
	}
	ev, _ := s.q.Events(ctx, a.ID)
	joined := strings.Join(ev, ",")
	for _, want := range []string{"claimed:worker-A:1", "lease_expired", "heartbeat_refused:worker-A:1", "write_refused:worker-A:1", "claimed:worker-B:2"} {
		if !strings.Contains(joined, want) {
			t.Errorf("faltou %q em %s", want, joined)
		}
	}
}

func TestHeartbeatKeepsTheLeaseAlive(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	p := s.project(t, "hb", independent("h"))
	p.item(t, "h", 2)
	it := s.claim(t, "w")
	f := queue.Fence{Owner: "w", Generation: it.Generation}
	for i := 0; i < 3; i++ { // 3 × 40 s com lease de 60 s: só vive porque renova
		s.advance(40 * time.Second)
		if exp, _ := s.q.ExpireLeases(ctx); len(exp) != 0 {
			t.Fatalf("a lease renovada não vence: %+v", exp)
		}
		if _, err := s.q.Heartbeat(ctx, it.ID, f, time.Minute); err != nil {
			t.Fatal(err)
		}
	}
	s.advance(61 * time.Second) // sem heartbeat
	if exp, _ := s.q.ExpireLeases(ctx); len(exp) != 1 {
		t.Fatalf("%+v", exp)
	}
	if _, err := s.q.Claim(ctx, "", "", time.Minute); err == nil {
		t.Fatal("worker obrigatório")
	}
	if _, err := s.q.Claim(ctx, "w", "", time.Millisecond); err == nil {
		t.Fatal("lease mínima")
	}
}

// T10-03 — "Projeto elegível antigo consegue vaga."
func TestAnOldEligibleProjectGetsASlotThroughAging(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	if _, err := s.q.SetSettings(ctx, 1, 30); err != nil {
		t.Fatal(err)
	}
	old := s.project(t, "ag-old", independent("o"))
	oldItem := old.item(t, "o", 4) // pouco urgente
	s.advance(10 * time.Minute)
	n1 := s.project(t, "ag-new1", independent("n"))
	n1.item(t, "n", 0)
	first := s.claim(t, "w")
	if first == nil || first.ProjectID != n1.proj {
		t.Fatalf("a mais urgente passa na frente: %+v", first)
	}
	if _, err := s.q.Complete(fenced(first), first.ID); err != nil {
		t.Fatal(err)
	}
	// Chega outra urgente, mas o item antigo já esperou 4 períodos (prioridade efetiva 0) e é mais velho.
	s.advance(2 * time.Hour)
	n2 := s.project(t, "ag-new2", independent("m"))
	n2.item(t, "m", 0)
	got := s.claim(t, "w")
	if got == nil || got.ID != oldItem.ID {
		t.Fatalf("o projeto antigo deve conseguir a vaga: %+v", got)
	}
	// Sem envelhecimento suficiente a ordem original vale.
	s2 := newSchedEnv(t)
	_, _ = s2.q.SetSettings(ctx, 1, 30)
	o := s2.project(t, "ag2-old", independent("o"))
	o.item(t, "o", 4)
	s2.advance(35 * time.Minute) // 1 período: efetiva 3
	nn := s2.project(t, "ag2-new", independent("n"))
	nn.item(t, "n", 1)
	if g := s2.claim(t, "w"); g == nil || g.ProjectID != nn.proj {
		t.Fatalf("%+v", g)
	}
}

func TestPriorityChangesOnlyAffectTheNextSelection(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	if _, err := s.q.SetSettings(ctx, 1, 30); err != nil {
		t.Fatal(err)
	}
	a := s.project(t, "pr-a", independent("x"))
	b := s.project(t, "pr-b", independent("y"))
	ai := a.item(t, "x", 2)
	bi := b.item(t, "y", 2)
	// Mesma prioridade efetiva: vale a prioridade do projeto (menor primeiro), mesmo com chegada posterior.
	if _, err := s.e.db.Pool.Exec(ctx, `UPDATE projects SET priority = 3 WHERE id = $1`, a.proj); err != nil {
		t.Fatal(err)
	}
	if _, err := s.e.db.Pool.Exec(ctx, `UPDATE projects SET priority = 1 WHERE id = $1`, b.proj); err != nil {
		t.Fatal(err)
	}
	cand, _ := s.q.Candidates(ctx)
	if len(cand) != 2 || cand[0].ID != bi.ID {
		t.Fatalf("%+v", cand)
	}
	// Mudar a prioridade de um item da fila muda a próxima seleção.
	if _, err := s.q.SetPriority(ctx, ai.ID, 0); err != nil {
		t.Fatal(err)
	}
	if cand, _ := s.q.Candidates(ctx); cand[0].ID != ai.ID {
		t.Fatalf("%+v", cand)
	}
	if _, err := s.q.SetPriority(ctx, ai.ID, 9); err == nil {
		t.Fatal("faixa")
	}
	// Quem está em execução não é preemptado nem reprioritizado.
	got := s.claim(t, "w")
	if got == nil || got.ID != ai.ID {
		t.Fatalf("%+v", got)
	}
	if _, err := s.q.SetPriority(ctx, ai.ID, 4); !errors.Is(err, queue.ErrBadState) {
		t.Fatalf("%v", err)
	}
	if running, _ := s.counts(t); running != 1 {
		t.Fatal("nada foi preemptado")
	}
}
