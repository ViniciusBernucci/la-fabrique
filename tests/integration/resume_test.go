package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/resume"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type rsExec struct {
	calls map[string]int
	out   map[string]any
}

func (x *rsExec) Run(_ context.Context, n workflows.Node, _ map[string]any) (workflows.ExecResult, error) {
	x.calls[n.ID]++
	return workflows.ExecResult{Outputs: map[string]any{"v": x.out[n.ID]}}, nil
}

type rsEnv struct {
	*relEnv
	ex      string
	ver     releases.Version
	exec    *rsExec
	store   *checkpoints.Store
	res     *resume.Resumer
	cp      checkpoints.Checkpoint
	feature string
	wf      wfResp
}

// newRsEnv publica um workflow n1 → n2, inicia a execução presa à versão, roda até o fim e grava o checkpoint.
func newRsEnv(t *testing.T) *rsEnv {
	t.Helper()
	r := newRelEnv(t)
	acct := connWith(t, r.c, "cli", "claude-rs", "claude")
	r.agent(t, "dev", &acct.ID)
	n := func(id string) workflows.Node {
		return wfNode(id, "agent", 0, 0, map[string]any{"agent": "dev", "prompt": "faça " + id})
	}
	w := r.draft(t, "fluxo-resume", "feature", linear(n("n1"), n("n2")))
	r.c.expect(r.publish(w), 201)
	feat := r.feature(t, "RS-1")
	ctx := context.Background()
	ex, ver, err := releases.StartExecution(ctx, r.e.db.Pool, r.proj, feat, w.ID)
	if err != nil {
		t.Fatal(err)
	}
	e := &rsEnv{relEnv: r, ex: ex.ID, ver: ver, feature: feat, wf: w, exec: &rsExec{calls: map[string]int{}, out: map[string]any{"n1": 1, "n2": 1}},
		store: &checkpoints.Store{DB: r.e.db.Pool}}
	e.res = &resume.Resumer{DB: r.e.db.Pool, Checkpoints: e.store, Redactor: security.NewRedactor(), Exec: e.exec}
	st, err := workflows.NewRun(ver.FlatGraph, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := (&workflows.Engine{Exec: e.exec}).Advance(ctx, ver.FlatGraph, st); err != nil || st.Status != workflows.RunCompleted {
		t.Fatalf("%v %s", err, st.Status)
	}
	e.cp = e.saveCheckpoint(t, st)
	e.exec.calls = map[string]int{}
	return e
}

func (e *rsEnv) saveCheckpoint(t *testing.T, st *workflows.RunState) checkpoints.Checkpoint {
	t.Helper()
	body := "package main"
	h := sha256.Sum256([]byte(body))
	hash := hex.EncodeToString(h[:])
	files := []checkpoints.FileEntry{{Path: "main.go", Kind: checkpoints.KindFile, Mode: 0o644, Size: int64(len(body)), SHA256: hash}}
	snap := checkpoints.Snapshot{Manifest: checkpoints.Manifest{Version: 1, Files: files, TreeHash: checkpoints.TreeHash(files), TotalBytes: int64(len(body))},
		Objects: map[string][]byte{hash: []byte(body)}}
	state, _ := json.Marshal(st)
	cp, err := e.store.Save(context.Background(), snap, checkpoints.Input{ProjectID: e.proj, ExecutionID: e.ex, Label: "fim", State: state, Refs: map[string]string{"branch": "feat/rs"}})
	if err != nil {
		t.Fatal(err)
	}
	return cp
}

func baseInputs() resume.Inputs {
	return resume.Inputs{Refs: map[string]string{"branch": "feat/rs", "head": "aaa111"},
		Requirement: resume.Requirement{Title: "Exportar CSV", Description: "Exportar relatórios", Status: "To Do", Priority: "High", Labels: []string{"lf-ready"}},
		ConfigHash:  "cfg-1"}
}

func (e *rsEnv) request(node string, cur resume.Inputs) resume.Request {
	return resume.Request{ProjectID: e.proj, ExecutionID: e.ex, Node: node, Current: cur,
		Options: resume.Options{FastForward: func(_, before, after string) (bool, error) { return before == "aaa111" && after == "bbb222", nil }}}
}

func (e *rsEnv) events(t *testing.T) []string {
	t.Helper()
	rows, err := e.e.db.Pool.Query(context.Background(), `SELECT action FROM resume_events WHERE execution_id = $1 ORDER BY id`, e.ex)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var a string
		_ = rows.Scan(&a)
		out = append(out, a)
	}
	return out
}

// T12-04 — "Reexecutar nó altera só o conjunto afetado ... o workflow original continua fixado."
func TestResumingANodeRerunsOnlyTheAffectedSetOnTheOriginalWorkflow(t *testing.T) {
	e := newRsEnv(t)
	ctx := context.Background()
	if err := e.res.RecordBaseline(ctx, e.proj, e.ex, e.cp.ID, baseInputs()); err != nil {
		t.Fatal(err)
	}
	// Efeito já confirmado do nó: fica registrado como reaproveitado (o ledger não o repete).
	key := resume.EffectPrefix(e.ex, "n1") + "create_subtask"
	if _, err := e.e.db.Pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1, $2, 'jira', 'confirmed')`, e.proj, key); err != nil {
		t.Fatal(err)
	}
	// Alguém republica o workflow depois: a execução segue na versão original.
	var w2 wfResp
	e.c.expect(saveWorkflow(e.c, e.wf, "fluxo-resume", linear(wfNode("n1", "agent", 0, 0, map[string]any{"agent": "dev", "prompt": "outro"}))), 200).JSON(&w2)
	e.c.expect(e.publish(w2), 201)

	e.exec.out["n1"] = 2 // a saída de n1 muda ao reexecutar
	out, err := e.res.Resume(ctx, e.request("n1", baseInputs()))
	if err != nil || out.Status != resume.StatusResumed || out.State == nil {
		t.Fatalf("%+v %v", out, err)
	}
	if out.VersionID != e.ver.ID {
		t.Fatalf("o workflow original deve seguir fixado: %s ≠ %s", out.VersionID, e.ver.ID)
	}
	if v, _ := releases.ExecutionVersion(ctx, e.e.db.Pool, e.ex); v.ID != e.ver.ID {
		t.Fatal("a execução trocou de versão")
	}
	if e.exec.calls["n1"] != 1 || e.exec.calls["n2"] != 0 {
		t.Fatalf("só o nó escolhido roda agora: %v", e.exec.calls)
	}
	if !out.Rerun.OutputChanged || strings.Join(out.Rerun.Invalidated, ",") != "e,n2" || len(out.KeptEffects) != 1 || out.KeptEffects[0] != key {
		t.Fatalf("%+v %v", out.Rerun, out.KeptEffects)
	}
	if out.State.Visited["n2"] || !out.State.Visited["n1"] || !out.State.Visited["s"] {
		t.Fatalf("%+v", out.State.Visited)
	}
	// O checkpoint original segue intacto (a retomada não o reescreve).
	if cur, _ := e.store.Latest(ctx, e.proj, e.ex); cur.ID != e.cp.ID || cur.StateHash != e.cp.StateHash {
		t.Fatal("o checkpoint mudou")
	}
	// O motor completa só o que foi invalidado.
	e.exec.calls = map[string]int{}
	if err := (&workflows.Engine{Exec: e.exec}).Advance(ctx, e.ver.FlatGraph, out.State); err != nil || out.State.Status != workflows.RunCompleted || e.exec.calls["n2"] != 1 || e.exec.calls["n1"] != 0 {
		t.Fatalf("%v %s %v", err, out.State.Status, e.exec.calls)
	}
	if ev := e.events(t); len(ev) != 1 || ev[0] != "resumed" {
		t.Fatalf("%v", ev)
	}
	// Com a mesma saída, nada é invalidado.
	e.exec.calls = map[string]int{}
	out, err = e.res.Resume(ctx, e.request("n1", baseInputs()))
	if err != nil || out.Rerun.OutputChanged == false && len(out.Rerun.Invalidated) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
}

// T12-04 — "Incorporar mudanças compatíveis".
func TestCompatibleInputChangesAreIncorporatedIntoTheBaseline(t *testing.T) {
	e := newRsEnv(t)
	ctx := context.Background()
	_ = e.res.RecordBaseline(ctx, e.proj, e.ex, e.cp.ID, baseInputs())
	cur := baseInputs()
	cur.Refs["head"] = "bbb222" // fast-forward comprovado
	cur.Requirement.Priority = "Low"
	cur.Requirement.Labels = []string{"lf-ready", "urgente"}
	out, err := e.res.Resume(ctx, e.request("n1", cur))
	if err != nil || out.Status != resume.StatusResumed || len(out.Incorporated) != 3 || out.Decision != nil {
		t.Fatalf("%+v %v", out, err)
	}
	if ev := e.events(t); len(ev) != 2 || ev[0] != "incorporated" || ev[1] != "resumed" {
		t.Fatalf("%v", ev)
	}
	// O baseline passou a ser o estado atual: uma nova retomada não vê diferença.
	out, err = e.res.Resume(ctx, e.request("n2", cur))
	if err != nil || len(out.Diffs) != 0 || len(out.Incorporated) != 0 {
		t.Fatalf("%+v %v", out, err)
	}
}

// T12-04 — "Conflito preserva edições humanas".
func TestConflictingInputChangesOpenADecisionAndChangeNothing(t *testing.T) {
	e := newRsEnv(t)
	ctx := context.Background()
	_ = e.res.RecordBaseline(ctx, e.proj, e.ex, e.cp.ID, baseInputs())
	var baselineBefore string
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT inputs_hash FROM resume_baselines WHERE execution_id = $1`, e.ex).Scan(&baselineBefore)
	cur := baseInputs()
	cur.Requirement.Description = "Exportar relatórios em XML (mudou)"
	cur.Refs["head"] = "zzz999" // ref divergente
	out, err := e.res.Resume(ctx, e.request("n1", cur))
	if err != nil || out.Status != resume.StatusBlocked || out.State != nil || out.Decision == nil || out.Decision.Kind != decisions.KindConflict || out.Decision.State != decisions.StatePending {
		t.Fatalf("%+v %v", out, err)
	}
	if len(out.Decision.Evidence) != 2 {
		t.Fatalf("a evidência lista cada diferença: %+v", out.Decision.Evidence)
	}
	if e.exec.calls["n1"] != 0 {
		t.Fatal("nada pode ser reexecutado com conflito")
	}
	var baselineAfter string
	_ = e.e.db.Pool.QueryRow(ctx, `SELECT inputs_hash FROM resume_baselines WHERE execution_id = $1`, e.ex).Scan(&baselineAfter)
	if baselineAfter != baselineBefore {
		t.Fatal("o baseline (e o que o humano editou) não pode mudar")
	}
	if cur, _ := e.store.Latest(ctx, e.proj, e.ex); cur.ID != e.cp.ID {
		t.Fatal("o checkpoint mudou")
	}
	// Repetir não abre outra decisão.
	out2, _ := e.res.Resume(ctx, e.request("n1", cur))
	if out2.Decision == nil || out2.Decision.ID != out.Decision.ID {
		t.Fatalf("decisão duplicada: %+v", out2.Decision)
	}
	if ev := e.events(t); len(ev) != 2 || ev[0] != "blocked" {
		t.Fatalf("%v", ev)
	}
}

func TestAnUncertainEffectInTheAffectedSetBlocksTheResume(t *testing.T) {
	e := newRsEnv(t)
	ctx := context.Background()
	key := resume.EffectPrefix(e.ex, "n2") + "create_pr"
	if _, err := e.e.db.Pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1, $2, 'github', 'uncertain')`, e.proj, key); err != nil {
		t.Fatal(err)
	}
	out, err := e.res.Resume(ctx, e.request("n1", baseInputs()))
	if err != nil || out.Status != resume.StatusBlocked || out.Decision == nil || out.Decision.Kind != decisions.KindUncertainResult || !strings.Contains(out.Decision.Problem, "n2") {
		t.Fatalf("%+v %v", out, err)
	}
	if e.exec.calls["n1"] != 0 || e.exec.calls["n2"] != 0 {
		t.Fatal("nada pode rodar")
	}
	// Efeito incerto de um nó FORA do conjunto afetado não bloqueia.
	e2 := newRsEnv(t)
	if _, err := e2.e.db.Pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1, $2, 'github', 'uncertain')`, e2.proj, resume.EffectPrefix(e2.ex, "n1")+"x"); err != nil {
		t.Fatal(err)
	}
	if out, err := e2.res.Resume(ctx, e2.request("n2", baseInputs())); err != nil || out.Status != resume.StatusResumed || e2.exec.calls["n2"] != 1 {
		t.Fatalf("%+v %v", out, err)
	}
}

func TestResumeValidatesNodeCheckpointAndPublishedConfig(t *testing.T) {
	e := newRsEnv(t)
	ctx := context.Background()
	if _, err := e.res.Resume(ctx, e.request("fantasma", baseInputs())); !errors.Is(err, resume.ErrUnknownNode) {
		t.Fatalf("%v", err)
	}
	if _, err := e.res.Resume(ctx, e.request("s", baseInputs())); !errors.Is(err, resume.ErrNotRerunnable) {
		t.Fatalf("%v", err)
	}
	// Conta do agente revogada: a versão publicada deixou de ser executável → bloqueia, sem rodar.
	acct := connWith(t, e.c, "cli", "claude-rs2", "claude")
	_ = acct
	var conns []security.Connection
	e.c.expect(e.c.get("/api/connections"), 200).JSON(&struct {
		Items *[]security.Connection `json:"items"`
	}{&conns})
	for _, c := range conns {
		if c.Name == "claude-rs" {
			e.c.expect(e.c.post("/api/connections/"+c.ID+"/revoke", nil), 200)
		}
	}
	out, err := e.res.Resume(ctx, e.request("n1", baseInputs()))
	if err != nil || out.Status != resume.StatusBlocked || len(out.Reasons) < 2 || e.exec.calls["n1"] != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	// Checkpoint corrompido: nada é usado às cegas.
	if _, err := e.e.db.Pool.Exec(ctx, `UPDATE checkpoints SET state_hash = repeat('0', 64) WHERE id = $1`, e.cp.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := e.res.Resume(ctx, e.request("n1", baseInputs())); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("%v", err)
	}
}
