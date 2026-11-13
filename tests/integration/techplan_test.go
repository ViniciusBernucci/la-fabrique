package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/techplan"
)

func tpTask(key, track string, paths []string, deps ...string) planning.Task {
	return planning.Task{Key: key, Title: "Tarefa " + key, Description: "faz " + key, Risk: "medium", Track: track, Paths: paths, DependsOn: deps,
		Criteria: []planning.Criterion{{Text: "os testes de " + key + " passam", Check: planning.CheckTest, Target: "./" + key}}}
}

// AJ05-02 / CT04, CT05 — Issue mista vira unidades internas classificadas; ciclo não publica; nada remoto.
func TestTechLeadPlanBecomesInternalSubtasksOnly(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "techplan")
	f, err := persistence.CreateFeature(ctx, db.Pool, p.ID, "TP-1", "Feature mista")
	if err != nil {
		t.Fatal(err)
	}
	svc := &techplan.Service{DB: db.Pool, Plans: &planning.Store{DB: db.Pool}}
	good := planning.Plan{Summary: "api e tela", Tasks: []planning.Task{
		tpTask("api", "BACKEND", []string{"internal/x/x.go", "contracts/openapi.yaml"}),
		tpTask("tela", "FRONTEND", []string{"apps/web/src/pages/X.tsx"}, "api"),
	}}
	cyc := planning.Plan{Tasks: []planning.Task{tpTask("a", "BACKEND", []string{"internal/a.go"}, "b"), tpTask("b", "BACKEND", []string{"internal/b.go"}, "a")}}
	var rej *techplan.RejectedError
	if _, err := svc.Accept(ctx, p.ID, f.ID, cyc); !errors.As(err, &rej) {
		t.Fatalf("%v", err)
	}
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM plans WHERE feature_id=$1`, f.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("plano recusado foi gravado: %d", n)
	}
	r, err := svc.Accept(ctx, p.ID, f.ID, good)
	if err != nil || r.PlanVersion != 1 || len(r.Subtasks) != 2 {
		t.Fatalf("%+v %v", r, err)
	}
	again, err := svc.Accept(ctx, p.ID, f.ID, good)
	if err != nil || !again.Unchanged || again.PlanVersion != 1 {
		t.Fatalf("%+v %v", again, err)
	}
	subs, _ := svc.List(ctx, f.ID, 1)
	if len(subs) != 2 || subs[0].Key != "api" || subs[0].Track != "BACKEND" || subs[1].Track != "FRONTEND" || len(subs[0].Resources) != 1 || subs[0].Resources[0] != "contract:api" {
		t.Fatalf("%+v", subs)
	}
	// Nada foi criado fora da fábrica: sem ID remoto nas tarefas e sem efeito de subtarefa na outbox.
	var remote, effects int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM plan_tasks WHERE feature_id=$1 AND remote_id <> ''`, f.ID).Scan(&remote)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE project_id=$1`, p.ID).Scan(&effects)
	if remote != 0 || effects != 0 {
		t.Fatalf("remote=%d outbox=%d", remote, effects)
	}
}
