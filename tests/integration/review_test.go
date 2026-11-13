package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/checks"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/review"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type rvEnv struct {
	*schedEnv
	rs *review.Store
	ck *checks.Store
	p  qProj
}

func newRvEnv(t *testing.T, slug string) *rvEnv {
	t.Helper()
	s := newSchedEnv(t)
	p := s.project(t, slug, independent("t"))
	red := security.NewRedactor()
	red.Register("segredo-da-revisao-xyz")
	ck := &checks.Store{DB: s.e.db.Pool, Redactor: red}
	rs := &review.Store{DB: s.e.db.Pool, Redactor: red, Queue: s.q}
	rs.Checks = func(ctx context.Context, featureID, revision string) (bool, []string, error) {
		g, err := ck.Gate(ctx, featureID, revision)
		return g.Blocked, g.Reasons, err
	}
	return &rvEnv{schedEnv: s, rs: rs, ck: ck, p: p}
}

func (v *rvEnv) greenChecks(t *testing.T, rev string) {
	t.Helper()
	ctx := context.Background()
	sp := []checks.Spec{{Name: "test", Command: "make test"}}
	ex := ckExec{out: map[string]checks.Outcome{"make test": okOut()}}
	if _, err := v.ck.RunSet(ctx, v.p.proj, v.p.feature, checks.RoleBaseline, "base", ex, sp); err != nil {
		t.Fatal(err)
	}
	if _, err := v.ck.RunSet(ctx, v.p.proj, v.p.feature, checks.RoleCandidate, rev, ex, sp); err != nil {
		t.Fatal(err)
	}
	if _, err := v.ck.Evaluate(ctx, v.p.proj, v.p.feature, rev, "base"); err != nil {
		t.Fatal(err)
	}
}

func (v *rvEnv) redChecks(t *testing.T, rev string) {
	t.Helper()
	ctx := context.Background()
	sp := []checks.Spec{{Name: "test", Command: "make test"}}
	if _, err := v.ck.RunSet(ctx, v.p.proj, v.p.feature, checks.RoleBaseline, "base", ckExec{out: map[string]checks.Outcome{"make test": okOut()}}, sp); err != nil {
		t.Fatal(err)
	}
	if _, err := v.ck.RunSet(ctx, v.p.proj, v.p.feature, checks.RoleCandidate, rev, ckExec{out: map[string]checks.Outcome{"make test": failOut("quebrou")}}, sp); err != nil {
		t.Fatal(err)
	}
	if _, err := v.ck.Evaluate(ctx, v.p.proj, v.p.feature, rev, "base"); err != nil {
		t.Fatal(err)
	}
}

func finding(sev, title string) review.FindingInput {
	return review.FindingInput{Severity: sev, Title: title, Body: "detalhe de " + title, File: "internal/x/x.go"}
}

// T14-03 — "Revisão inicial independente" e configuração do revisor com fallback.
func TestTheInitialReviewMustBeIndependentAndTheReviewerComesFromAConfiguredChain(t *testing.T) {
	v := newRvEnv(t, "rv-indep")
	ctx := context.Background()
	if _, err := v.rs.Pick(ctx, v.p.proj, "dev", nil); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("sem configuração: %v", err)
	}
	cfg, err := v.rs.SetConfig(ctx, v.p.proj, review.Config{Reviewer: "revisor", Fallback: []string{"revisor-b", "dev"}})
	if err != nil || cfg.MaxRounds != 2 {
		t.Fatalf("o limite padrão é de duas rodadas: %+v %v", cfg, err)
	}
	if got, _ := v.rs.Pick(ctx, v.p.proj, "dev", nil); got != "revisor" {
		t.Fatalf("%s", got)
	}
	if got, _ := v.rs.Pick(ctx, v.p.proj, "revisor", nil); got != "revisor-b" {
		t.Fatalf("o implementador nunca revisa a própria obra: %s", got)
	}
	if got, _ := v.rs.Pick(ctx, v.p.proj, "dev", func(a string) bool { return a != "revisor" }); got != "revisor-b" {
		t.Fatalf("fallback quando o principal está indisponível: %s", got)
	}
	if _, err := v.rs.Pick(ctx, v.p.proj, "dev", func(string) bool { return false }); !errors.Is(err, review.ErrNoReviewer) {
		t.Fatalf("%v", err)
	}
	if _, err := v.rs.SetConfig(ctx, v.p.proj, review.Config{Reviewer: "a", Fallback: []string{"a"}}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("repetido: %v", err)
	}
	if _, err := v.rs.SetConfig(ctx, v.p.proj, review.Config{Reviewer: "a", MaxRounds: 99}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("limite: %v", err)
	}
	// O mesmo agente implementou e revisaria: recusado, e o banco também recusa.
	if _, err := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev1", "dev", "dev"); !errors.Is(err, review.ErrNotIndependent) {
		t.Fatalf("%v", err)
	}
	if _, err := v.e.db.Pool.Exec(ctx, `INSERT INTO reviews (project_id, feature_id, task_key, round, revision, implementer, reviewer, status) VALUES ($1, $2, 't', 1, 'r', 'dev', 'dev', 'reviewing')`, v.p.proj, v.p.feature); err == nil {
		t.Fatal("o banco deve garantir a independência da primeira rodada")
	}
	if _, err := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev1", "dev", "revisor"); err != nil {
		t.Fatal(err)
	}
	if _, err := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev1", "dev", "revisor-b"); !errors.Is(err, review.ErrActive) {
		t.Fatalf("uma revisão por vez: %v", err)
	}
}

// T14-03 — "Crítico/relevante bloqueiam. Baixa severidade fica registrada."
func TestCriticalAndRelevantFindingsBlockWhileLowSeverityIsOnlyRecorded(t *testing.T) {
	v := newRvEnv(t, "rv-sev")
	ctx := context.Background()
	// Revisão obrigatória sem revisão feita: bloqueia.
	if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev1", true); !g.Blocked || g.Status != "missing" {
		t.Fatalf("%+v", g)
	}
	if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev1", false); g.Blocked {
		t.Fatalf("%+v", g)
	}
	rv, _ := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev1", "dev", "revisor")
	if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev1", true); !g.Blocked || g.Status != review.StatusReviewing {
		t.Fatalf("em andamento: %+v", g)
	}
	// Entradas inválidas.
	if _, err := v.rs.Submit(ctx, v.p.proj, rv.ID, []review.FindingInput{{Severity: "grave", Title: "x"}}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	// Só baixa severidade: aprova, mas registra.
	done, err := v.rs.Submit(ctx, v.p.proj, rv.ID, []review.FindingInput{
		{Severity: review.Low, Title: "nome de variável pouco claro", Body: "token=segredo-da-revisao-xyz", File: "a.go"},
		{Severity: review.Low, Title: "comentário desatualizado"}})
	if err != nil || done.Status != review.StatusApproved || len(done.Findings) != 2 || done.Findings[0].State != "noted" {
		t.Fatalf("%+v %v", done, err)
	}
	if strings.Contains(done.Findings[0].Body, "segredo-da-revisao-xyz") {
		t.Fatal("o achado não pode guardar segredo")
	}
	g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev1", true)
	if g.Blocked || len(g.Noted) != 2 {
		t.Fatalf("baixa severidade não bloqueia e fica registrada: %+v", g)
	}
	// Crítico bloqueia; relevante também.
	for _, sev := range []string{review.Critical, review.Relevant} {
		rv2, err := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev-"+sev, "dev", "revisor")
		if err != nil {
			t.Fatal(err)
		}
		got, err := v.rs.Submit(ctx, v.p.proj, rv2.ID, []review.FindingInput{finding(sev, "problema "+sev), finding(review.Low, "nota")})
		if err != nil || got.Status != review.StatusChangesRequested {
			t.Fatalf("%s: %+v %v", sev, got, err)
		}
		if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev-"+sev, true); !g.Blocked || len(g.Noted) != 1 {
			t.Fatalf("%s: %+v", sev, g)
		}
		// Fecha a rodada para liberar a próxima (correção e checks verdes).
		if _, err := v.rs.BeginFix(ctx, v.p.proj, rv2.ID, "revisor"); err != nil {
			t.Fatal(err)
		}
		v.greenChecks(t, "fix-"+sev)
		if _, err := v.rs.CompleteFix(ctx, v.p.proj, rv2.ID, "fix-"+sev); err != nil {
			t.Fatal(err)
		}
		// Fecha também a rodada de revalidação.
		hist, _ := v.rs.History(ctx, v.p.proj, v.p.feature, "t")
		if _, err := v.rs.Submit(ctx, v.p.proj, hist[0].ID, nil); err != nil {
			t.Fatal(err)
		}
	}
}

// T14-03 — "Correção do revisor roda checks."
func TestAFixIsOnlyAcceptedWithGreenChecksAndTriggersARevalidationRound(t *testing.T) {
	v := newRvEnv(t, "rv-fix")
	ctx := context.Background()
	rv, _ := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev1", "dev", "revisor")
	if _, err := v.rs.BeginFix(ctx, v.p.proj, rv.ID, "revisor"); !errors.Is(err, review.ErrBadState) {
		t.Fatalf("só depois dos achados: %v", err)
	}
	if _, err := v.rs.Submit(ctx, v.p.proj, rv.ID, []review.FindingInput{finding(review.Critical, "SQL sem parâmetro"), finding(review.Low, "estilo")}); err != nil {
		t.Fatal(err)
	}
	// O revisor pode corrigir em etapa posterior.
	fix, err := v.rs.BeginFix(ctx, v.p.proj, rv.ID, "revisor")
	if err != nil || fix.Status != review.StatusFixing || fix.Fixer != "revisor" {
		t.Fatalf("%+v %v", fix, err)
	}
	// Sem checks da nova revisão, ou com checks vermelhos: recusado.
	if _, err := v.rs.CompleteFix(ctx, v.p.proj, rv.ID, "rev2"); !errors.Is(err, review.ErrChecksNotGreen) {
		t.Fatalf("sem checks: %v", err)
	}
	v.redChecks(t, "rev2")
	if _, err := v.rs.CompleteFix(ctx, v.p.proj, rv.ID, "rev2"); !errors.Is(err, review.ErrChecksNotGreen) {
		t.Fatalf("checks vermelhos: %v", err)
	}
	if _, err := v.rs.CompleteFix(ctx, v.p.proj, rv.ID, "rev1"); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("a correção precisa de revisão nova: %v", err)
	}
	// Checks verdes: aceita, marca os bloqueantes como corrigidos e abre a rodada 2 sobre a nova revisão.
	v.greenChecks(t, "rev3")
	next, err := v.rs.CompleteFix(ctx, v.p.proj, rv.ID, "rev3")
	if err != nil || next.Round != 2 || next.Revision != "rev3" || next.Status != review.StatusReviewing || next.Reviewer != "revisor" {
		t.Fatalf("%+v %v", next, err)
	}
	old, _ := v.rs.Get(ctx, v.p.proj, rv.ID)
	bySev := map[string]review.Finding{}
	for _, f := range old.Findings {
		bySev[f.Severity] = f
	}
	if old.FixRevision != "rev3" || bySev[review.Critical].State != "fixed" || bySev[review.Critical].FixRevision != "rev3" || bySev[review.Low].State != "noted" {
		t.Fatalf("%+v", old)
	}
	// A aprovação da nova revisão vale só para ela; a revisão antiga segue sem aprovação válida.
	ok, err := v.rs.Submit(ctx, v.p.proj, next.ID, nil)
	if err != nil || ok.Status != review.StatusApproved {
		t.Fatalf("%+v %v", ok, err)
	}
	if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev3", true); g.Blocked {
		t.Fatalf("%+v", g)
	}
	if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev4-depois", true); !g.Blocked {
		t.Fatalf("uma alteração depois da aprovação exige nova revisão: %+v", g)
	}
}

// T14-03 — "esgotamento suspende com checkpoint".
func TestExhaustingTheRoundsSuspendsTheTaskWithASafeCheckpoint(t *testing.T) {
	v := newRvEnv(t, "rv-exhaust")
	ctx := context.Background()
	if _, err := v.rs.SetConfig(ctx, v.p.proj, review.Config{Reviewer: "revisor"}); err != nil {
		t.Fatal(err)
	}
	v.p.item(t, "t", 0)
	exec := v.p.execution(t)
	claimed, err := v.q.Claim(ctx, "worker", exec, 60*1e9)
	if err != nil || claimed == nil {
		t.Fatalf("%+v %v", claimed, err)
	}
	var cpSaved string
	v.rs.Checkpointer = func(ctx context.Context, rv review.Review) (string, bool, error) {
		cpSaved = v.p.checkpoint(t, exec)
		return cpSaved, true, nil
	}
	// Rodada 1: crítico → correção; rodada 2: ainda crítico → esgota (limite padrão de duas rodadas).
	r1, _ := v.rs.Start(ctx, v.p.proj, v.p.feature, "t", "rev1", "dev", "revisor")
	_, _ = v.rs.Submit(ctx, v.p.proj, r1.ID, []review.FindingInput{finding(review.Critical, "falha de autorização")})
	_, _ = v.rs.BeginFix(ctx, v.p.proj, r1.ID, "revisor")
	v.greenChecks(t, "rev2")
	r2, err := v.rs.CompleteFix(ctx, v.p.proj, r1.ID, "rev2")
	if err != nil || r2.Round != 2 {
		t.Fatalf("%+v %v", r2, err)
	}
	ex, err := v.rs.Submit(ctx, v.p.proj, r2.ID, []review.FindingInput{finding(review.Relevant, "o problema continua")})
	if err != nil || ex.Status != review.StatusExhausted || ex.CheckpointID == "" || ex.CheckpointID != cpSaved {
		t.Fatalf("%+v %v", ex, err)
	}
	// O item foi suspenso com o checkpoint seguro: vaga devolvida, dono antigo sem direito de gravar.
	items, _ := v.q.List(ctx, v.p.proj)
	if items[0].State != queue.StateWaiting || items[0].WaitReason != queue.WaitSuspended || items[0].SlotHeld || items[0].CheckpointID != cpSaved || items[0].Generation != claimed.Generation+1 {
		t.Fatalf("%+v", items[0])
	}
	if _, err := v.q.Complete(fenced(claimed), claimed.ID); err == nil {
		t.Fatal("o dono antigo não conclui")
	}
	// Abre uma decisão humana e a revisão segue bloqueando.
	decs, _ := decisions.List(ctx, v.e.db.Pool, decisions.StatePending)
	found := false
	for _, d := range decs {
		found = found || (d.ExecutionID == exec && strings.Contains(d.Title, "esgotou"))
	}
	if !found {
		t.Fatalf("%+v", decs)
	}
	if g, _ := v.rs.Gate(ctx, v.p.proj, v.p.feature, "t", "rev2", true); !g.Blocked || g.Status != review.StatusExhausted {
		t.Fatalf("%+v", g)
	}
	// Se o checkpoint não for seguro, a suspensão não acontece às cegas.
	v2 := newRvEnv(t, "rv-exhaust2")
	_, _ = v2.rs.SetConfig(ctx, v2.p.proj, review.Config{Reviewer: "revisor", MaxRounds: 1})
	v2.p.item(t, "t", 0)
	exec2 := v2.p.execution(t)
	_, _ = v2.q.Claim(ctx, "w", exec2, 60*1e9)
	v2.rs.Checkpointer = func(ctx context.Context, rv review.Review) (string, bool, error) {
		return v2.p.checkpoint(t, exec2), false, nil
	}
	r, _ := v2.rs.Start(ctx, v2.p.proj, v2.p.feature, "t", "rev1", "dev", "revisor")
	if _, err := v2.rs.Submit(ctx, v2.p.proj, r.ID, []review.FindingInput{finding(review.Critical, "x")}); err == nil || !strings.Contains(err.Error(), "suspensão não foi segura") {
		t.Fatalf("%v", err)
	}
	if items, _ := v2.q.List(ctx, v2.p.proj); items[0].State != queue.StateRunning {
		t.Fatalf("o item segue em execução sem checkpoint seguro: %+v", items[0])
	}
}
