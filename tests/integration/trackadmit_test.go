package integration

import (
	"context"
	"sync"
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackadmit"
)

func seedSubtask(t *testing.T, db testDB, project, feature, key, track, deps, resources string) {
	t.Helper()
	if _, err := db.Pool.Exec(context.Background(), `INSERT INTO subtasks (project_id, feature_id, plan_version, task_key, track, risk, depends_on, resources, content_hash)
		VALUES ($1,$2,1,$3,$4,'low',$5::jsonb,$6::jsonb,'h')`, project, feature, key, track, deps, resources); err != nil {
		t.Fatal(err)
	}
}

func reasons(w []trackadmit.Waiting) map[string]string {
	m := map[string]string{}
	for _, x := range w {
		m[x.TaskKey+"/"+x.Stage] = x.Reason
	}
	return m
}

// AJ06-02 / CT05, CT06 — independentes ocupam as duas vagas; mesma trilha, dependência e recurso
// compartilhado aguardam; review inicia cedo na vaga da própria trilha; o agregado segue a trilha atrasada.
func TestTrackAdmissionUsesDependenciesResourcesAndSlots(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "tk")
	var feature string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'TK-1','f') RETURNING id::text`, p.ID).Scan(&feature); err != nil {
		t.Fatal(err)
	}
	seedSubtask(t, db, p.ID, feature, "front-a", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, p.ID, feature, "front-b", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, p.ID, feature, "back-a", "BACKEND", `[]`, `["contracts/openapi.yaml"]`)
	seedSubtask(t, db, p.ID, feature, "front-dep", "FRONTEND", `["back-a"]`, `[]`)
	svc := &trackadmit.Service{DB: db.Pool}
	lim := sentinel.Limits{PlanningSlots: 1, GlobalCapacity: 5}

	started, waiting, err := svc.Tick(ctx, p.ID, lim)
	if err != nil {
		t.Fatal(err)
	}
	if len(started) != 2 || started[0].TaskKey != "front-a" || started[1].TaskKey != "back-a" {
		t.Fatalf("independentes ocupam as duas vagas: %+v", started)
	}
	w := reasons(waiting)
	if w["front-b/implementation"] != sentinel.WaitSlot || w["front-dep/implementation"] != sentinel.WaitDependency {
		t.Fatalf("segunda tarefa Front espera a vaga; a dependente espera a API: %v", w)
	}
	// Um novo Tick não admite nada nem duplica reserva.
	if s2, _, err := svc.Tick(ctx, p.ID, lim); err != nil || len(s2) != 0 {
		t.Fatalf("%+v %v", s2, err)
	}
	// O Front termina a implementação: o review começa cedo, na vaga do Front, com o Back ainda implementando.
	if err := svc.Complete(ctx, p.ID, feature, "front-a", trackadmit.StageImplementation); err != nil {
		t.Fatal(err)
	}
	s3, w3, err := svc.Tick(ctx, p.ID, lim)
	if err != nil {
		t.Fatal(err)
	}
	if len(s3) != 1 || s3[0].Stage != trackadmit.StageReview || s3[0].TaskKey != "front-a" {
		t.Fatalf("review antecipado ocupa a vaga do Front: %+v waiting=%v", s3, reasons(w3))
	}
	// Front-a já está em review, mas front-b ainda nem começou e o Back implementa: o agregado não avança.
	if st, behind, tr, _ := svc.Progress(ctx, feature); st != statusflow.Implementacao || behind == "" || tr[statusflow.Backend] != statusflow.StageImplementing {
		t.Fatalf("o agregado segue a etapa obrigatória mais atrasada: %s %s %v", st, behind, tr)
	}
	if err := svc.Complete(ctx, p.ID, feature, "front-a", trackadmit.StageReview); err != nil {
		t.Fatal(err)
	}
	// O Back conclui: a vaga e o recurso são liberados e a tarefa dependente entra.
	if err := svc.Complete(ctx, p.ID, feature, "back-a", trackadmit.StageImplementation); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM track_reservations WHERE project_id=$1 AND slot='RES:contracts/openapi.yaml'`, p.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("o recurso deve ser liberado: %d", n)
	}
	s4, _, err := svc.Tick(ctx, p.ID, lim)
	if err != nil || len(s4) < 2 {
		t.Fatalf("a dependência satisfeita e o review do Back entram: %+v %v", s4, err)
	}
}

// AJ06-02 — conflito real de recurso impede a concorrência mesmo com as duas vagas livres; duas passagens
// simultâneas não admitem a mesma vaga duas vezes.
func TestTrackAdmissionBlocksSharedResourceAndSurvivesConcurrentTicks(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "tk2")
	var feature string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'TK-2','f') RETURNING id::text`, p.ID).Scan(&feature); err != nil {
		t.Fatal(err)
	}
	seedSubtask(t, db, p.ID, feature, "back-a", "BACKEND", `[]`, `["db/schema"]`)
	seedSubtask(t, db, p.ID, feature, "front-a", "FRONTEND", `[]`, `["db/schema"]`)
	svc := &trackadmit.Service{DB: db.Pool}
	lim := sentinel.Limits{PlanningSlots: 1, GlobalCapacity: 5}
	started, waiting, err := svc.Tick(ctx, p.ID, lim)
	if err != nil || len(started) != 1 || reasons(waiting)["front-a/implementation"] != sentinel.WaitResource {
		t.Fatalf("conflito real: uma espera: %+v %v %v", started, reasons(waiting), err)
	}
	// Corrida: duas passagens ao mesmo tempo sobre tarefas livres.
	p2 := mustProject(t, db.Pool, "tk3")
	var f2 string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'TK-3','f') RETURNING id::text`, p2.ID).Scan(&f2)
	seedSubtask(t, db, p2.ID, f2, "front-a", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, p2.ID, f2, "front-b", "FRONTEND", `[]`, `[]`)
	var wg sync.WaitGroup
	var mu sync.Mutex
	total := 0
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, _, err := svc.Tick(ctx, p2.ID, lim)
			if err != nil {
				t.Error(err)
			}
			mu.Lock()
			total += len(s)
			mu.Unlock()
		}()
	}
	wg.Wait()
	if total != 1 {
		t.Fatalf("a vaga do Front só pode ser admitida uma vez: %d", total)
	}
}
