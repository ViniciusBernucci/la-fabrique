package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/activation"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// aptProject devolve um projeto legado apto (ativo, workflow padrão e workflow do projeto publicados) com
// mapa de status, para exercitar a ativação.
func aptProject(t *testing.T, slug string) *prEnv {
	t.Helper()
	r := newPrEnv(t, slug)
	r.agent(t, "dev")
	w := r.workflow(t, "fluxo-padrao", true)
	r.c.expect(r.c.do("PUT", "/api/projects/"+r.p.ID+"/selection", map[string]any{"rules": []any{}, "default": map[string]any{"workflow_id": w.ID}}, nil), 200)
	var pw projWorkflow
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/workflow"), 200).JSON(&pw)
	var sw wfResp
	r.c.expect(r.c.get("/api/workflows/"+pw.WorkflowID), 200).JSON(&sw)
	g := sw.Graph
	for i := range g.Nodes {
		switch g.Nodes[i].Type {
		case "agent":
			g.Nodes[i].Props["agent"] = "dev"
		case "checks":
			g.Nodes[i].Props["commands"] = "make test"
		}
	}
	r.c.expect(saveWorkflow(r.c, sw, sw.Name, g), 200).JSON(&sw)
	r.c.expect(r.c.post("/api/workflows/"+sw.ID+"/publish", map[string]any{"expected_version": sw.Version}), 201)
	if rs := r.activate(t); rs.StatusCode != 200 {
		t.Fatalf("%d %s", rs.StatusCode, rs.Body)
	}
	return r
}

// AJ03-05 / CT16 — prévia sem efeito, ativação só em ponto seguro, repetível, alias normalizado com o bruto
// preservado, um único despachante admite; a reversão drena, reconcilia e preserva os dados.
func TestActivationPreviewSafePointIdempotencyAndRollback(t *testing.T) {
	r := aptProject(t, "act")
	ctx := context.Background()
	db := r.e.db.Pool
	pid := r.p.ID
	svc := &activation.Service{DB: db}
	co := &sentinel.Coordinator{DB: db}
	if err := (&statusflow.Store{DB: db}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	// Dados legados: Issue conhecida com o alias antigo, e uma execução do despachante anterior em curso.
	_, _ = db.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, title, raw_status, status_code, revision, kind)
		VALUES ($1,'jira','9','LEG-1','Legada','Reprovado','REPROVED_PR','v1','correction'), ($1,'jira','10','LEG-2','Feature legada','Doing','IMPLEMENTACAO','v1','feature')`, pid)
	var feature string
	_ = db.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'LEG-2','f') RETURNING id::text`, pid).Scan(&feature)
	_, _ = db.Exec(ctx, `INSERT INTO queue_items (project_id, feature_id, task_key, state) VALUES ($1,$2,'t1','running')`, pid, feature)

	// Sem mapa de status a prévia bloqueia e a ativação recusa.
	pv, err := svc.Preview(ctx, pid)
	if err != nil || len(pv.Blocking) == 0 || pv.AliasToNormalize != 1 || pv.KnownIssues != 2 || pv.Untracked != 2 || pv.Safe || len(pv.LegacyRunning) != 1 {
		t.Fatalf("%+v %v", pv, err)
	}
	if _, err := svc.Activate(ctx, pid, co); !errors.Is(err, activation.ErrBlocked) {
		t.Fatalf("%v", err)
	}
	// A prévia não mudou nada.
	var alias int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM issue_registry WHERE status_code='REPROVED_PR'`).Scan(&alias)
	if alias != 1 {
		t.Fatal("a prévia não pode alterar dados")
	}
	// Mapa configurado, mas há execução do despachante anterior: fora de ponto seguro.
	_, _ = db.Exec(ctx, `INSERT INTO project_status_maps (project_id, source, remote, moves) VALUES ($1,'jira','{"Doing":"IMPLEMENTACAO"}','{}')`, pid)
	if _, err := svc.Activate(ctx, pid, co); !errors.Is(err, activation.ErrNotSafe) {
		t.Fatalf("%v", err)
	}
	if ok, _ := svc.CanAdmit(ctx, pid, activation.Legacy); !ok {
		t.Fatal("antes da ativação só o despachante anterior admite")
	}
	if ok, _ := svc.CanAdmit(ctx, pid, activation.Sentinel); ok {
		t.Fatal("a Sentinela não admite antes da ativação")
	}
	// Ponto seguro: a execução terminou. A ativação normaliza o alias, preserva o bruto e rastreia as Issues.
	_, _ = db.Exec(ctx, `UPDATE queue_items SET state='done' WHERE project_id=$1`, pid)
	if _, err := svc.Activate(ctx, pid, co); err != nil {
		t.Fatal(err)
	}
	var code, raw string
	_ = db.QueryRow(ctx, `SELECT status_code, raw_status FROM issue_registry WHERE issue_key='LEG-1'`).Scan(&code, &raw)
	if code != statusflow.ReprovadoPR || raw != "Reprovado" {
		t.Fatalf("alias normalizado com o bruto preservado: %s %s", code, raw)
	}
	if iss, err := co.Get(ctx, pid, "LEG-1"); err != nil || iss.Status != statusflow.ReprovadoPR {
		t.Fatalf("%+v %v", iss, err)
	}
	for _, who := range []struct {
		d  string
		ok bool
	}{{activation.Sentinel, true}, {activation.Legacy, false}} {
		if ok, _ := svc.CanAdmit(ctx, pid, who.d); ok != who.ok {
			t.Fatalf("só um despachante admite: %s=%v", who.d, ok)
		}
	}
	// Repetir a ativação não duplica nada.
	if _, err := svc.Activate(ctx, pid, co); err != nil {
		t.Fatal(err)
	}
	var tracked int
	_ = db.QueryRow(ctx, `SELECT count(*) FROM flow_issues WHERE project_id=$1`, pid).Scan(&tracked)
	if tracked != 2 {
		t.Fatalf("%d", tracked)
	}
	// Reversão: desabilita novas admissões, drena e reconcilia antes de devolver ao anterior.
	_, _ = db.Exec(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'LEG-3','f')`, pid)
	seedSubtask(t, testDB{Pool: db}, pid, feature, "s1", "BACKEND", `[]`, `[]`)
	_, _ = db.Exec(ctx, `UPDATE subtasks SET impl_state='running' WHERE feature_id=$1`, feature)
	_, _ = db.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1,'k-sending','git','sending')`, pid)
	d, err := svc.Rollback(ctx, pid)
	if err != nil || d.Done || len(d.Running) != 1 || len(d.ToReconcile) != 1 {
		t.Fatalf("%+v %v", d, err)
	}
	for _, who := range []string{activation.Sentinel, activation.Legacy} {
		if ok, _ := svc.CanAdmit(ctx, pid, who); ok {
			t.Fatalf("durante a drenagem ninguém admite: %s", who)
		}
	}
	if err := svc.FinishRollback(ctx, pid); !errors.Is(err, activation.ErrNotDrained) {
		t.Fatalf("%v", err)
	}
	_, _ = db.Exec(ctx, `UPDATE subtasks SET impl_state='done' WHERE feature_id=$1`, feature)
	_, _ = db.Exec(ctx, `UPDATE outbox SET state='confirmed' WHERE project_id=$1`, pid)
	if err := svc.FinishRollback(ctx, pid); err != nil {
		t.Fatal(err)
	}
	if ok, _ := svc.CanAdmit(ctx, pid, activation.Legacy); !ok {
		t.Fatal("depois da reversão o despachante anterior volta")
	}
	// Os dados e os vínculos permanecem.
	var keep int
	_ = db.QueryRow(ctx, `SELECT (SELECT count(*) FROM issue_registry WHERE project_id=$1) + (SELECT count(*) FROM flow_issues WHERE project_id=$1)`, pid).Scan(&keep)
	if keep != 4 {
		t.Fatalf("nada é apagado: %d", keep)
	}
}

// AJ03-05 — com outro despachante ativo (ou em reversão) a fila legada não admite o projeto.
func TestLegacyQueueStopsAdmittingWhenAnotherDispatcherIsActive(t *testing.T) {
	e := newQEnv(t)
	ctx := context.Background()
	p := e.project(t, "gate", independent("a"))
	p.item(t, "a", 2)
	set := func(d string) {
		_, err := e.e.db.Pool.Exec(ctx, `INSERT INTO flow_activation (project_id, state, dispatcher) VALUES ($1,'active',$2)
			ON CONFLICT (project_id) DO UPDATE SET dispatcher = EXCLUDED.dispatcher`, p.proj, d)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{"sentinel", "none"} {
		set(d)
		if c, err := e.q.Candidates(ctx); err != nil || len(c) != 0 {
			t.Fatalf("%s: a fila legada não deve admitir: %+v %v", d, c, err)
		}
	}
	set("legacy")
	if c, err := e.q.Candidates(ctx); err != nil || len(c) != 1 {
		t.Fatalf("com o legado ativo a fila volta: %+v %v", c, err)
	}
}
