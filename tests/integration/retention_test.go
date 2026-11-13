package integration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/contextpack"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/resume"
	"github.com/ViniciusBernucci/le-fabrique/internal/retention"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type retEnv struct {
	*qEnv
	r   *retention.Retention
	now time.Time
	p   qProj
}

func newRetEnv(t *testing.T) *retEnv {
	t.Helper()
	q := newQEnv(t)
	e := &retEnv{qEnv: q, now: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)}
	e.r = &retention.Retention{DB: q.e.db.Pool, Checkpoints: q.cps, Now: func() time.Time { return e.now }}
	e.p = q.project(t, "ret", independent("t"))
	return e
}

func (e *retEnv) daysAgo(n int) time.Time { return e.now.AddDate(0, 0, -n) }

func (e *retEnv) execution(t *testing.T, state string, ageDays int) string {
	t.Helper()
	id := e.p.execution(t)
	if _, err := e.e.db.Pool.Exec(context.Background(), `UPDATE executions SET state = $2, updated_at = $3 WHERE id = $1`, id, state, e.daysAgo(ageDays)); err != nil {
		t.Fatal(err)
	}
	return id
}

func (e *retEnv) event(t *testing.T, exec, typ string, ageDays int) int64 {
	t.Helper()
	var seq int64
	if err := persistence.WithTx(context.Background(), e.e.db.Pool, func(tx pgx.Tx) error {
		ev, err := persistence.AppendEvent(context.Background(), tx, e.p.proj, exec, typ, map[string]any{"n": 1})
		seq = ev.Seq
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := e.e.db.Pool.Exec(context.Background(), `UPDATE execution_events SET created_at = $3 WHERE execution_id = $1 AND seq = $2`, exec, seq, e.daysAgo(ageDays)); err != nil {
		t.Fatal(err)
	}
	return seq
}

func (e *retEnv) cp(t *testing.T, exec, body string) string {
	t.Helper()
	cp, err := e.cps.Save(context.Background(), e.p.snapshot(t, body), checkpoints.Input{ProjectID: e.p.proj, ExecutionID: exec, Label: body})
	if err != nil {
		t.Fatal(err)
	}
	return cp.ID
}

func (e *retEnv) countEvents(t *testing.T, exec string) (n int) {
	t.Helper()
	_ = e.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM execution_events WHERE execution_id = $1`, exec).Scan(&n)
	return
}

func (e *retEnv) exists(t *testing.T, cpID string) bool {
	t.Helper()
	var n int
	_ = e.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM checkpoints WHERE id = $1`, cpID).Scan(&n)
	return n == 1
}

// T16-04 — "Relógio de teste remove apenas itens elegíveis" (logs: 30 dias, resumo permanente, seq nunca reaproveitado).
func TestDetailedLogsAreRemovedAfter30DaysKeepingAPermanentSummary(t *testing.T) {
	e := newRetEnv(t)
	ctx := context.Background()
	exec := e.execution(t, "succeeded", 1)
	for _, typ := range []string{"step", "step", "error"} {
		e.event(t, exec, typ, 40)
	}
	e.event(t, exec, "step", 2)
	e.event(t, exec, "done", 2)
	// Execução com decisão pendente (incidente): os logs ficam mesmo velhos.
	inc := e.execution(t, "waiting", 40)
	e.event(t, inc, "step", 45)
	if _, _, err := decisions.Create(ctx, e.e.db.Pool, security.NewRedactor(), newRequest(e.p.proj, inc, "t", decisions.KindConflict)); err != nil {
		t.Fatal(err)
	}
	// Antes do prazo (relógio de teste 25 dias atrás): nada é elegível.
	e.now = e.now.AddDate(0, 0, -20)
	rep, err := e.r.Run(ctx, retention.Policy{}, false)
	if err != nil || rep.LogsDeleted != 0 || e.countEvents(t, exec) != 5 {
		t.Fatalf("%+v %v", rep, err)
	}
	e.now = e.now.AddDate(0, 0, 20)
	// Simulação: não remove nada, mas diz o que removeria.
	dry, err := e.r.Run(ctx, retention.Policy{}, true)
	if err != nil || !dry.DryRun || dry.LogsDeleted != 3 || e.countEvents(t, exec) != 5 {
		t.Fatalf("%+v %v", dry, err)
	}
	if _, err := e.r.SummaryOf(ctx, e.p.proj, exec); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("a simulação não grava resumo: %v", err)
	}
	rep, err = e.r.Run(ctx, retention.Policy{}, false)
	if err != nil || rep.LogsDeleted != 3 || rep.ExecutionsSummarized != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	if n := e.countEvents(t, exec); n != 2 {
		t.Fatalf("só os eventos com mais de 30 dias saem: restam %d", n)
	}
	var oldKept int
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM execution_events WHERE execution_id = $1 AND created_at < $2`, inc, e.daysAgo(30)).Scan(&oldKept)
	if oldKept != 1 {
		t.Fatalf("a execução com incidente mantém os logs antigos: %d", oldKept)
	}
	sum, err := e.r.SummaryOf(ctx, e.p.proj, exec)
	if err != nil || sum.Compacted != 3 || sum.ByType["step"] != 2 || sum.ByType["error"] != 1 || sum.LastSeq != 5 || sum.CompactedTil != 3 || sum.FirstAt == nil {
		t.Fatalf("%+v %v", sum, err)
	}
	kept := false
	for _, d := range rep.Decisions {
		kept = kept || (d.Kind == "logs" && d.Ref == inc && d.Action == "kept" && strings.Contains(d.Reason, "decisão humana pendente"))
	}
	if !kept {
		t.Fatalf("a decisão de manter fica auditada: %+v", rep.Decisions)
	}
	// O seq nunca é reaproveitado, nem depois de remover TODOS os eventos da execução.
	if s := e.event(t, exec, "step", 0); s != 6 {
		t.Fatalf("o cursor continua: %d", s)
	}
	old := e.execution(t, "succeeded", 1)
	e.event(t, old, "step", 50)
	e.event(t, old, "step", 50)
	if _, err := e.r.Run(ctx, retention.Policy{}, false); err != nil {
		t.Fatal(err)
	}
	if e.countEvents(t, old) != 0 {
		t.Fatal("todos os eventos antigos saíram")
	}
	if s := e.event(t, old, "step", 0); s != 3 {
		t.Fatalf("o seq 1 e 2 não podem voltar: %d", s)
	}
	// Repetir não remove nem resume de novo.
	again, err := e.r.Run(ctx, retention.Policy{}, false)
	if err != nil || again.LogsDeleted != 0 {
		t.Fatalf("%+v %v", again, err)
	}
	if s2, _ := e.r.SummaryOf(ctx, e.p.proj, exec); s2.Compacted != 3 {
		t.Fatalf("%+v", s2)
	}
}

// T16-04 — "Checkpoint bloqueado ou vinculado a incidente continua íntegro e restaurável."
func TestCheckpointsOfConcludedWorkAreRemovedAfterTheGraceButProtectedOnesStayRestorable(t *testing.T) {
	e := newRetEnv(t)
	ctx := context.Background()
	pool := e.e.db.Pool

	// D: concluída há 20 dias; um checkpoint bloqueado e outro livre (ambos usam o MESMO objeto).
	d := e.execution(t, "succeeded", 20)
	dFree := e.cp(t, d, "conteudo-compartilhado")
	dBlocked := e.cp(t, d, "conteudo-compartilhado")
	admin := ""
	_ = pool.QueryRow(ctx, `SELECT id FROM admins LIMIT 1`).Scan(&admin)
	if err := e.r.Protect(ctx, e.p.proj, dBlocked, "evidência do incidente 42", "INC-42", admin); err != nil {
		t.Fatal(err)
	}
	// H: cancelada há 30 dias, com objeto próprio: sai por inteiro.
	h := e.execution(t, "cancelled", 30)
	hCp := e.cp(t, h, "conteudo-exclusivo-de-h")
	// E: concluída há 5 dias (carência); F: em execução; J: falhou (não é "concluída").
	eExec := e.execution(t, "succeeded", 5)
	eCp := e.cp(t, eExec, "conteudo-e")
	f := e.execution(t, "running", 40)
	fCp := e.cp(t, f, "conteudo-f")
	j := e.execution(t, "failed", 40)
	jCp := e.cp(t, j, "conteudo-j")
	// G: concluída há 30 dias, mas com efeito externo sem confirmação (incidente).
	g := e.execution(t, "succeeded", 30)
	gCp := e.cp(t, g, "conteudo-g")
	if _, err := pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1, $2, 'github', 'uncertain')`, e.p.proj, resume.EffectPrefix(g, "x")+"create_pr"); err != nil {
		t.Fatal(err)
	}
	// I: concluída há 30 dias, mas o checkpoint é a base de uma sessão do agente.
	i := e.execution(t, "succeeded", 30)
	iCp := e.cp(t, i, "conteudo-i")
	pack, _ := contextpack.Build([]contextpack.Source{{Kind: contextpack.KindRequirement, Ref: "R1", Content: "x", Required: true}}, contextpack.Limits{MaxTokens: 100}, nil)
	cx := &contextpack.Store{DB: pool, Checkpoints: e.cps}
	if _, err := cx.Renew(ctx, contextpack.RenewInput{ProjectID: e.p.proj, ExecutionID: i, Milestone: contextpack.MilestoneReview, CheckpointID: iCp, Pack: pack, Model: "m"}); err != nil {
		t.Fatal(err)
	}
	// K: concluída há 30 dias com uma decisão pendente (incidente).
	k := e.execution(t, "succeeded", 30)
	kCp := e.cp(t, k, "conteudo-k")
	if _, err := pool.Exec(ctx, `UPDATE executions SET state = 'waiting' WHERE id = $1`, k); err != nil {
		t.Fatal(err)
	}
	if _, _, err := decisions.Create(ctx, pool, security.NewRedactor(), newRequest(e.p.proj, k, "t", decisions.KindConflict)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE executions SET state = 'succeeded' WHERE id = $1`, k); err != nil {
		t.Fatal(err)
	}
	var objsBefore int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM checkpoint_objects WHERE project_id = $1`, e.p.proj).Scan(&objsBefore)

	// Simulação: nada muda.
	dry, err := e.r.Run(ctx, retention.Policy{}, true)
	if err != nil || dry.CheckpointsDeleted != 2 || !dry.DryRun {
		t.Fatalf("%+v %v", dry, err)
	}
	for _, id := range []string{dFree, dBlocked, hCp} {
		if !e.exists(t, id) {
			t.Fatal("a simulação não remove nada")
		}
	}
	rep, err := e.r.Run(ctx, retention.Policy{}, false)
	if err != nil || rep.CheckpointsDeleted != 2 {
		t.Fatalf("%+v %v", rep, err)
	}
	// Só os elegíveis saíram: o livre de D e o de H.
	if e.exists(t, dFree) || e.exists(t, hCp) {
		t.Fatal("os elegíveis deveriam ter saído")
	}
	for name, id := range map[string]string{"bloqueado": dBlocked, "carência": eCp, "em execução": fCp, "falhou": jCp, "efeito incerto": gCp, "sessão do agente": iCp, "decisão pendente": kCp} {
		if !e.exists(t, id) {
			t.Fatalf("%s não podia sair", name)
		}
	}
	// O que ficou continua íntegro e restaurável: manifesto, estado e TODOS os objetos.
	for _, id := range []string{dBlocked, eCp, fCp, jCp, gCp, iCp, kCp} {
		cp, err := e.cps.Get(ctx, e.p.proj, id)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		for _, f := range cp.Manifest.Files {
			if _, err := e.cps.Object(ctx, e.p.proj, f.SHA256); err != nil {
				t.Fatalf("%s: objeto %s: %v", id, f.Path, err)
			}
		}
	}
	// Objetos: saiu só o exclusivo de H; o compartilhado de D segue (o bloqueado o usa).
	var objsAfter int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM checkpoint_objects WHERE project_id = $1`, e.p.proj).Scan(&objsAfter)
	if rep.ObjectsDeleted != 1 || objsAfter != objsBefore-1 {
		t.Fatalf("objetos removidos %d (antes %d, depois %d)", rep.ObjectsDeleted, objsBefore, objsAfter)
	}
	// Auditoria: execução registrada e uma decisão por item, com o motivo.
	var runs, decs int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM retention_runs WHERE id = $1 AND NOT dry_run AND checkpoints_deleted = 2`, rep.RunID).Scan(&runs)
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM retention_decisions WHERE run_id = $1 AND kind = 'checkpoint'`, rep.RunID).Scan(&decs)
	if runs != 1 || decs != 9 {
		t.Fatalf("auditoria: %d execução(ões), %d decisões", runs, decs)
	}
	why := map[string]string{}
	for _, dd := range rep.Decisions {
		if dd.Kind == "checkpoint" {
			why[dd.Ref] = dd.Action + ": " + dd.Reason
		}
	}
	for ref, want := range map[string]string{dBlocked: "bloqueado explicitamente", eCp: "carência", fCp: "execução ainda não concluída", jCp: "não está concluída", gCp: "efeito externo sem confirmação", iCp: "sessão do agente", kCp: "decisão humana pendente"} {
		if !strings.Contains(why[ref], "kept") || !strings.Contains(why[ref], want) {
			t.Errorf("%s: %q não menciona %q", ref, why[ref], want)
		}
	}
	// Liberada a proteção, o checkpoint passa a ser elegível numa próxima execução.
	if err := e.r.Unprotect(ctx, e.p.proj, dBlocked); err != nil {
		t.Fatal(err)
	}
	next, err := e.r.Run(ctx, retention.Policy{}, false)
	if err != nil || e.exists(t, dBlocked) || next.CheckpointsDeleted != 1 {
		t.Fatalf("%+v %v", next, err)
	}
	// Com a carência maior que a idade, nada sai.
	if _, err := e.r.Run(ctx, retention.Policy{GraceDays: 60}, false); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{eCp, gCp, iCp} {
		if !e.exists(t, id) {
			t.Fatal("nada mais podia sair")
		}
	}
}

func TestRetentionValidatesThePolicyAndTheProtection(t *testing.T) {
	e := newRetEnv(t)
	ctx := context.Background()
	for _, p := range []retention.Policy{{LogDays: -1}, {GraceDays: 99999}} {
		if _, err := e.r.Run(ctx, p, true); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Errorf("%+v: %v", p, err)
		}
	}
	exec := e.execution(t, "succeeded", 1)
	cp := e.cp(t, exec, "x")
	if err := e.r.Protect(ctx, e.p.proj, cp, "  ", "", ""); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	if err := e.r.Protect(ctx, e.p.proj, "00000000-0000-0000-0000-000000000000", "motivo", "", ""); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("%v", err)
	}
	other := newRetEnv(t)
	if err := other.r.Protect(ctx, other.p.proj, cp, "motivo", "", ""); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("o checkpoint é do projeto: %v", err)
	}
}
