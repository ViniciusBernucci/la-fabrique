package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
)

type fakeSuspender struct {
	p       func(it queue.Item) (string, bool, error)
	stopped []string
}

func (f *fakeSuspender) Suspend(_ context.Context, it queue.Item) (string, bool, error) {
	f.stopped = append(f.stopped, it.TaskKey)
	return f.p(it)
}

func (s *schedEnv) admin(t *testing.T) string {
	t.Helper()
	var id string
	if err := s.e.db.Pool.QueryRow(context.Background(), `SELECT id FROM admins LIMIT 1`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// secondFeature cria outra feature com plano e itens no mesmo projeto.
func (p qProj) secondFeature(t *testing.T, ext string, keys ...string) qProj {
	t.Helper()
	ctx := context.Background()
	f, err := persistence.CreateFeature(ctx, p.e.db.Pool, p.proj, ext, "Feature "+ext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.plans.Propose(ctx, p.proj, f.ID, independent(keys...)); err != nil {
		t.Fatal(err)
	}
	return qProj{qEnv: p.qEnv, proj: p.proj, feature: f.ID}
}

// T10-04 — "As duas ações têm efeitos diferentes ... Projetos não afetados continuam executando."
func TestDrainAndSuspendHaveDifferentEffectsAndOtherProjectsKeepRunning(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	admin := s.admin(t)
	if _, err := s.q.SetSettings(ctx, 8, 30); err != nil {
		t.Fatal(err)
	}
	a := s.project(t, "ctl-a", independent("a1", "a2"))
	a.item(t, "a1", 0)
	a.item(t, "a2", 1)
	a2 := a.secondFeature(t, "NOVA", "c1")
	a2.item(t, "c1", 0)
	b := s.project(t, "ctl-b", independent("b1", "b2"))
	b.item(t, "b1", 2)
	b.item(t, "b2", 2)
	scopeA := queue.Scope{ProjectID: a.proj}

	// A feature atual de A está em curso: a1 roda.
	a1 := s.claim(t, "w-a")
	if a1 == nil || a1.TaskKey != "a1" {
		t.Fatalf("%+v", a1)
	}

	// 1) PAUSAR APÓS A FEATURE ATUAL (draining): o que roda continua e a feature atual termina; feature nova não inicia.
	ctl, err := s.q.SetMode(ctx, scopeA, queue.ModeDraining, admin, "fechar a feature antes do deploy")
	if err != nil || ctl.Mode != queue.ModeDraining || ctl.Version != 1 {
		t.Fatalf("%+v %v", ctl, err)
	}
	if cur, _ := s.q.Complete(fenced(a1), a1.ID); cur.State != queue.StateDone {
		t.Fatalf("o que já rodava termina normalmente: %+v", cur)
	}
	next := s.claim(t, "w-a")
	if next == nil || next.TaskKey != "a2" {
		t.Fatalf("a feature em curso continua até o fim: %+v", next)
	}
	_, _ = s.q.Complete(fenced(next), next.ID)
	reasons, _ := s.q.Explain(ctx, a2item(t, s, a2, "c1"))
	if !strings.Contains(strings.Join(reasons, " "), "pausa após a feature atual") {
		t.Fatalf("feature nova não inicia durante a pausa: %v", reasons)
	}
	if got := s.claim(t, "w-x"); got == nil || got.ProjectID != b.proj {
		t.Fatalf("o projeto B não é afetado: %+v", got)
	}

	// 2) SUSPENDER AGORA: o que roda é parado com checkpoint seguro e a vaga é devolvida.
	runB, _ := s.q.List(ctx, b.proj)
	var b1 queue.Item
	for _, it := range runB {
		if it.State == queue.StateRunning {
			b1 = it
		}
	}
	bExec := b.execution(t)
	if _, err := s.e.db.Pool.Exec(ctx, `UPDATE queue_items SET execution_id = $2 WHERE id = $1`, b1.ID, bExec); err != nil {
		t.Fatal(err)
	}
	sp := &fakeSuspender{p: func(it queue.Item) (string, bool, error) { return b.checkpoint(t, bExec), true, nil }}
	res, err := s.q.SuspendNow(ctx, queue.Scope{ProjectID: b.proj}, admin, "incidente no projeto B", sp)
	if err != nil || len(res.Suspended) != 1 || len(res.Failed) != 0 || res.Control.Mode != queue.ModeSuspended {
		t.Fatalf("%+v %v", res, err)
	}
	st := res.Suspended[0]
	if st.State != queue.StateWaiting || st.WaitReason != queue.WaitSuspended || st.SlotHeld || st.CheckpointID == "" || st.Generation != b1.Generation+1 {
		t.Fatalf("%+v", st)
	}
	// O antigo dono não grava mais; nada novo inicia em B; A (em pausa, não suspenso) segue as suas regras.
	if _, err := s.q.Complete(queue.WithFence(ctx, queue.Fence{Owner: b1.LeaseOwner, Generation: b1.Generation}), b1.ID); !errors.Is(err, queue.ErrStale) && !errors.Is(err, queue.ErrBadState) {
		t.Fatalf("dono antigo: %v", err)
	}
	if got := s.claim(t, "w-b"); got != nil {
		t.Fatalf("nada novo inicia com B suspenso e A em pausa: %+v", got)
	}
	// Diferença entre as ações: a pausa deixou terminar; a suspensão parou e guardou checkpoint.
	// Retomar B: o item volta à fila e roda de novo.
	ctlB, back, err := s.q.ResumeScope(ctx, queue.Scope{ProjectID: b.proj}, admin, "incidente resolvido")
	if err != nil || ctlB.Mode != queue.ModeActive || len(back) != 1 || back[0].State != queue.StateQueued {
		t.Fatalf("%+v %v %v", ctlB, back, err)
	}
	again := s.claim(t, "w-b2")
	if again == nil || again.ProjectID != b.proj || again.Generation != st.Generation+1 {
		t.Fatalf("%+v", again)
	}
	// Reativar A libera a feature nova.
	if _, _, err := s.q.ResumeScope(ctx, scopeA, admin, "deploy concluído"); err != nil {
		t.Fatal(err)
	}
	reasons, _ = s.q.Explain(ctx, a2item(t, s, a2, "c1"))
	if len(reasons) != 0 {
		t.Fatalf("%v", reasons)
	}
}

func a2item(t *testing.T, s *schedEnv, p qProj, key string) string {
	t.Helper()
	items, _ := s.q.List(context.Background(), p.proj)
	for _, it := range items {
		if it.FeatureID == p.feature && it.TaskKey == key {
			return it.ID
		}
	}
	t.Fatalf("item %s não encontrado", key)
	return ""
}

// T10-04 — "persistem no reboot".
func TestControlsPersistAcrossARestartAndAreAudited(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	admin := s.admin(t)
	p := s.project(t, "ctl-persist", independent("x"))
	p.item(t, "x", 0)
	if _, err := s.q.SetMode(ctx, queue.Scope{Global: true}, queue.ModeSuspended, admin, "manutenção da VPS"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.q.SetMode(ctx, queue.Scope{ProjectID: p.proj}, queue.ModeDraining, admin, ""); err != nil {
		t.Fatal(err)
	}
	// "Reboot": outra instância (outro processo) lê o mesmo banco.
	reborn := &queue.Queue{DB: s.e.db.Pool, Checkpoints: s.cps, Now: s.q.Now}
	if m, _ := reborn.Mode(ctx, queue.Scope{Global: true}); m != queue.ModeSuspended {
		t.Fatalf("%s", m)
	}
	if m, _ := reborn.Mode(ctx, queue.Scope{ProjectID: p.proj}); m != queue.ModeDraining {
		t.Fatalf("%s", m)
	}
	if it, err := reborn.Claim(ctx, "w", "", time.Minute); err != nil || it != nil {
		t.Fatalf("suspensão global persiste: %+v %v", it, err)
	}
	ctls, _ := reborn.Controls(ctx)
	if len(ctls) != 2 || ctls[0].Scope != "project" && ctls[0].Scope != "global" {
		t.Fatalf("%+v", ctls)
	}
	// A suspensão global vale também para o início direto.
	items, _ := reborn.List(ctx, p.proj)
	if _, err := reborn.Start(ctx, items[0].ID, ""); err == nil || !strings.Contains(err.Error(), "suspensão global") {
		t.Fatalf("%v", err)
	}
	// Retomar o global não derruba a pausa do projeto, e a trilha guarda quem fez o quê.
	if _, _, err := reborn.ResumeScope(ctx, queue.Scope{Global: true}, admin, "VPS pronta"); err != nil {
		t.Fatal(err)
	}
	if m, _ := reborn.Mode(ctx, queue.Scope{ProjectID: p.proj}); m != queue.ModeDraining {
		t.Fatalf("%s", m)
	}
	var n int
	var who string
	_ = s.e.db.Pool.QueryRow(ctx, `SELECT count(*), max(admin_id::text) FROM control_events WHERE admin_id = $1`, admin).Scan(&n, &who)
	if n != 3 || who != admin {
		t.Fatalf("%d %s", n, who)
	}
	for _, bad := range []queue.Scope{{}, {Global: true, ProjectID: p.proj}} {
		if _, err := reborn.SetMode(ctx, bad, queue.ModeActive, admin, ""); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Errorf("%+v: %v", bad, err)
		}
	}
	if _, err := reborn.SetMode(ctx, queue.Scope{Global: true}, "pausado", admin, ""); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
}

// Suspensão só com checkpoint seguro; global e projeto não se desfazem um ao outro.
func TestSuspendWithoutASafeCheckpointKeepsTheItemRunning(t *testing.T) {
	s := newSchedEnv(t)
	ctx := context.Background()
	admin := s.admin(t)
	p := s.project(t, "ctl-unsafe", independent("u1"))
	p.item(t, "u1", 0)
	exec := p.execution(t)
	got, _ := s.q.Claim(ctx, "w", exec, time.Minute)
	if got == nil {
		t.Fatal("claim")
	}
	cases := map[string]func(queue.Item) (string, bool, error){
		"sem checkpoint":    func(queue.Item) (string, bool, error) { return "", true, nil },
		"sandbox ativo":     func(queue.Item) (string, bool, error) { return p.checkpoint(t, exec), false, nil },
		"checkpoint alheio": func(queue.Item) (string, bool, error) { return p.checkpoint(t, p.execution(t)), true, nil },
		"falha ao parar":    func(queue.Item) (string, bool, error) { return "", false, errors.New("docker indisponível") },
	}
	for name, f := range cases {
		res, err := s.q.SuspendNow(ctx, queue.Scope{ProjectID: p.proj}, admin, name, &fakeSuspender{p: f})
		if err != nil || len(res.Suspended) != 0 || len(res.Failed) != 1 || res.Control.Mode != queue.ModeSuspended {
			t.Fatalf("%s: %+v %v", name, res, err)
		}
		if items, _ := s.q.List(ctx, p.proj); items[0].State != queue.StateRunning || !items[0].SlotHeld {
			t.Fatalf("%s: o item segue em execução: %+v", name, items[0])
		}
	}
	// Com um checkpoint seguro, suspende; o dono antigo não consegue concluir.
	res, _ := s.q.SuspendNow(ctx, queue.Scope{ProjectID: p.proj}, admin, "agora sim",
		&fakeSuspender{p: func(queue.Item) (string, bool, error) { return p.checkpoint(t, exec), true, nil }})
	if len(res.Suspended) != 1 {
		t.Fatalf("%+v", res)
	}
	if _, err := s.q.Complete(fenced(got), got.ID); err == nil {
		t.Fatal("o dono antigo não conclui um item suspenso")
	}
	// Suspensão global + do projeto: retomar só o global mantém o item suspenso.
	if _, err := s.q.SetMode(ctx, queue.Scope{Global: true}, queue.ModeSuspended, admin, ""); err != nil {
		t.Fatal(err)
	}
	if _, back, _ := s.q.ResumeScope(ctx, queue.Scope{Global: true}, admin, ""); len(back) != 0 {
		t.Fatalf("o projeto ainda está suspenso: %+v", back)
	}
	if _, back, _ := s.q.ResumeScope(ctx, queue.Scope{ProjectID: p.proj}, admin, ""); len(back) != 1 {
		t.Fatalf("%+v", back)
	}
	// O item suspenso não volta por um Resume manual (não há porta para "suspended").
	if _, err := s.q.Block(ctx, "00000000-0000-0000-0000-000000000000", queue.WaitSuspended, "", "", false); err == nil {
		t.Fatal("suspended não é motivo de Block")
	}
}
