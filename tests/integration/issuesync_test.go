package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// fakeTracker implementa só o que o sync usa: listagem paginada (sem fechadas, salvo IncludeDone) e GetItem.
type fakeTracker struct {
	integrations.Tracker
	provider string
	items    map[string]integrations.Item
	order    []string
	failPage int // 1-based; 0 = nunca
	gets     int
}

func (f *fakeTracker) Provider() string { return f.provider }
func (f *fakeTracker) ListItems(_ context.Context, _ string, o integrations.ListOptions) (integrations.ItemPage, error) {
	start := 0
	if o.Cursor != "" {
		fmt.Sscanf(o.Cursor, "%d", &start)
	}
	page := start/2 + 1
	if f.failPage == page {
		return integrations.ItemPage{}, &integrations.Error{Kind: integrations.ErrRateLimited, Op: "list"}
	}
	var vis []integrations.Item
	for _, id := range f.order {
		it := f.items[id]
		if it.Category == integrations.CategoryDone && !o.IncludeDone {
			continue
		}
		vis = append(vis, it)
	}
	// A listagem remota esconde a Issue 'hid' (ex.: arquivada do quadro), mas GetItem a devolve.
	var shown []integrations.Item
	for _, it := range vis {
		if it.ID != "hid" {
			shown = append(shown, it)
		}
	}
	end := start + 2
	if end > len(shown) {
		end = len(shown)
	}
	pg := integrations.ItemPage{Items: shown[start:end]}
	if end < len(shown) {
		pg.NextCursor = fmt.Sprint(end)
	}
	return pg, nil
}
func (f *fakeTracker) GetItem(_ context.Context, id string) (integrations.Item, error) {
	f.gets++
	return f.items[id], nil
}

func newFakeTracker(provider string) *fakeTracker {
	f := &fakeTracker{provider: provider, items: map[string]integrations.Item{}}
	add := func(id, status, cat string) {
		f.items[id] = integrations.Item{ID: id, Key: "K-" + id, Title: "Issue " + id, Description: "desc " + id, Status: status, Category: cat, Version: "v1", Priority: "High"}
		f.order = append(f.order, id)
	}
	add("a", "To Do", "todo")    // PRD-DOC: nova elegível
	add("b", "Fix", "todo")      // CORRECAO_PR: nova elegível
	add("c", "Backlog", "todo")  // desconhecida em outro status: ignorada
	add("d", "To Do", "todo")    // PRD-DOC
	add("hid", "Doing", "doing") // conhecida por seed; fora da listagem
	return f
}

func syncMap(source string) statusflow.ProjectMap {
	return statusflow.ProjectMap{Source: source, Remote: map[string]string{"To Do": statusflow.PRDDoc, "Fix": statusflow.CorrecaoPR,
		"Doing": statusflow.Implementacao, "Done": statusflow.Concluido}}
}

// AJ01-01 / CT01 — descobre os dois estados de entrada, ignora outros, atualiza conhecida fora deles e não duplica.
func TestIssueSyncDiscoversEntryStatesAndUpdatesKnownOnes(t *testing.T) {
	for _, kind := range []string{"jira", "clickup"} {
		t.Run(kind, func(t *testing.T) {
			db := newMigratedDB(t)
			ctx := context.Background()
			if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			p := mustProject(t, db.Pool, "isync-"+kind)
			sy := &issuesync.Syncer{DB: db.Pool, Sentinel: &sentinel.Coordinator{DB: db.Pool}}
			tr := newFakeTracker(kind)
			src := issuesync.Source{Kind: kind, Container: "C", Map: syncMap(kind)}
			// Seed: 'hid' já é conhecida (ex.: importada antes, hoje em Implementação).
			if _, err := db.Pool.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, raw_status, status_code, revision, kind)
				VALUES ($1,$2,'hid','K-hid','To Do','PRD-DOC','v0','feature')`, p.ID, kind); err != nil {
				t.Fatal(err)
			}
			run, err := sy.General(ctx, p.ID, issuesync.TriggerManual, tr, src)
			if err != nil || run.Outcome != "ok" || run.New != 3 || run.Ignored != 1 || run.Updated != 1 {
				t.Fatalf("%+v %v", run, err)
			}
			list, _ := sy.List(ctx, p.ID)
			byKey := map[string]issuesync.Issue{}
			for _, i := range list {
				byKey[i.RemoteID] = i
			}
			if len(list) != 4 || byKey["c"].RemoteID != "" || byKey["b"].Kind != "correction" || byKey["a"].Kind != "feature" {
				t.Fatalf("%+v", list)
			}
			if h := byKey["hid"]; h.StatusCode != statusflow.Implementacao || h.Revision != "v1" || tr.gets != 1 {
				t.Fatalf("conhecida fora da entrada deve ser atualizada por ID: %+v gets=%d", h, tr.gets)
			}
			// Repetir: um registro por Issue, nada novo, nenhuma atualização.
			again, err := sy.General(ctx, p.ID, issuesync.TriggerDaily, tr, src)
			if err != nil || again.New != 0 || again.Updated != 0 {
				t.Fatalf("%+v %v", again, err)
			}
			if l2, _ := sy.List(ctx, p.ID); len(l2) != 4 {
				t.Fatalf("duplicou: %d", len(l2))
			}
			// Mudança em Issue conhecida (qualquer status, inclusive concluída) atualiza; prioridade local é preservada.
			if _, err := db.Pool.Exec(ctx, `UPDATE issue_registry SET local_priority=1 WHERE project_id=$1 AND remote_id='a'`, p.ID); err != nil {
				t.Fatal(err)
			}
			it := tr.items["a"]
			it.Status, it.Category, it.Version = "Done", integrations.CategoryDone, "v2"
			tr.items["a"] = it
			up, err := sy.General(ctx, p.ID, issuesync.TriggerManual, tr, src)
			if err != nil || up.Updated != 1 {
				t.Fatalf("%+v %v", up, err)
			}
			l3, _ := sy.List(ctx, p.ID)
			for _, i := range l3 {
				if i.RemoteID == "a" && (i.StatusCode != statusflow.Concluido || i.LocalPrio == nil || *i.LocalPrio != 1) {
					t.Fatalf("%+v", i)
				}
			}
			// flow_issues foi rastreada para as novas (PRD-DOC e CORRECAO_PR).
			var n int
			_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM flow_issues WHERE project_id=$1`, p.ID).Scan(&n)
			if n != 3 {
				t.Fatalf("flow_issues=%d", n)
			}
			runs, _ := sy.Runs(ctx, p.ID, 10)
			if len(runs) != 3 || runs[0].Trigger != issuesync.TriggerManual {
				t.Fatalf("%+v", runs)
			}
		})
	}
}

// AJ01-02 — falha no meio da paginação preserva o progresso e registra o erro.
func TestIssueSyncKeepsProgressWhenAPageFails(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "isync-partial")
	sy := &issuesync.Syncer{DB: db.Pool}
	tr := newFakeTracker("jira")
	tr.failPage = 2
	src := issuesync.Source{Kind: "jira", Container: "C", Map: syncMap("jira")}
	run, err := sy.General(ctx, p.ID, issuesync.TriggerDaily, tr, src)
	if err != nil || run.Outcome != "partial" || run.New != 2 || run.Error == "" {
		t.Fatalf("%+v %v", run, err)
	}
	tr.failPage = 0
	run, err = sy.General(ctx, p.ID, issuesync.TriggerDaily, tr, src)
	if err != nil || run.Outcome != "ok" {
		t.Fatalf("%+v %v", run, err)
	}
	if l, _ := sy.List(ctx, p.ID); len(l) != 3 {
		t.Fatalf("%d", len(l))
	}
	if _, err := sy.General(ctx, p.ID, "nope", tr, src); err == nil || errors.Is(err, nil) {
		t.Fatal("gatilho inválido")
	}
}

// AJ01-02 / CT01 / CT15 — a rotina matinal dispara uma vez por janela, usa a mesma rotina do manual
// e registra último sucesso e erro por escopo.
func TestMorningIssueSyncRunsOncePerWindowAndKeepsErrorPerScope(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	pa := mustProject(t, db.Pool, "morning-a")
	pb := mustProject(t, db.Pool, "morning-b")
	loc := time.FixedZone("BRT", -3*3600)
	clk := &stepClock{t: time.Date(2026, 11, 13, 7, 5, 0, 0, loc)}
	sy := &issuesync.Syncer{DB: db.Pool, Sentinel: &sentinel.Coordinator{DB: db.Pool}, Now: clk.now}
	trA, trB := newFakeTracker("jira"), newFakeTracker("jira")
	trB.failPage = 1 // o projeto B está com a origem em limite de requisições
	resolve := func(_ context.Context, id string) (issuesync.Source, integrations.Tracker, error) {
		src := issuesync.Source{Kind: "jira", Container: "C", Map: syncMap("jira")}
		if id == pb.ID {
			return src, trB, nil
		}
		return src, trA, nil
	}
	sch := &sentinel.Scheduler{DB: db.Pool, Now: clk.now}
	if err := sch.Register(sy.Routine(resolve, sentinel.DailyAt{Hour: 7, Loc: loc})); err != nil {
		t.Fatal(err)
	}
	res, err := sch.Tick(ctx, []string{pa.ID, pb.ID})
	if err != nil || len(res) != 2 {
		t.Fatalf("%v %v", res, err)
	}
	// Mesma janela: nada roda de novo para A; B está em backoff.
	clk.add(30 * time.Second)
	if res, _ = sch.Tick(ctx, []string{pa.ID, pb.ID}); len(res) != 0 {
		t.Fatalf("reexecutou na mesma janela: %v", res)
	}
	stA, _ := sch.State(ctx, sentinel.RoutineIssueSync, pa.ID)
	stB, _ := sch.State(ctx, sentinel.RoutineIssueSync, pb.ID)
	if stA.LastSuccessAt == nil || stA.LastError != "" || stB.LastSuccessAt != nil || stB.FailCount != 1 || stB.LastError == "" {
		t.Fatalf("A=%+v B=%+v", stA, stB)
	}
	if l, _ := sy.List(ctx, pa.ID); len(l) != 3 {
		t.Fatalf("A importou %d", len(l))
	}
	runsB, _ := sy.Runs(ctx, pb.ID, 5)
	if len(runsB) != 1 || runsB[0].Outcome != "failed" || runsB[0].Trigger != issuesync.TriggerDaily {
		t.Fatalf("%+v", runsB)
	}
	// Dia seguinte: nova janela para A; B volta quando a origem se recupera.
	trB.failPage = 0
	clk.add(24 * time.Hour)
	if res, _ = sch.Tick(ctx, []string{pa.ID, pb.ID}); len(res) != 2 {
		t.Fatalf("%v", res)
	}
	if l, _ := sy.List(ctx, pb.ID); len(l) != 3 {
		t.Fatalf("B importou %d", len(l))
	}
}
