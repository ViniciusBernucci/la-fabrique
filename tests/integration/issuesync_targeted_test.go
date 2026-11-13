package integration

import (
	"context"
	"sync"
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// countingTracker conta o que foi lido e nunca lista a origem.
type countingTracker struct {
	integrations.Tracker
	mu    sync.Mutex
	items map[string]integrations.Item
	gets  []string
	lists int
}

func (c *countingTracker) Provider() string { return "jira" }
func (c *countingTracker) ListItems(context.Context, string, integrations.ListOptions) (integrations.ItemPage, error) {
	c.mu.Lock()
	c.lists++
	c.mu.Unlock()
	return integrations.ItemPage{}, nil // o filtro remoto de REPROVADO_PR já não encontra a Issue
}
func (c *countingTracker) GetItem(_ context.Context, id string) (integrations.Item, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gets = append(c.gets, id)
	return c.items[id], nil
}

// AJ01-03 / CT02 — consulta direcionada: só os IDs pendentes, sem lista; uma liberação, uma admissão.
func TestTargetedPollReleasesCorrectionsOnlyForPendingIDs(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "targeted")
	co := &sentinel.Coordinator{DB: db.Pool}
	sy := &issuesync.Syncer{DB: db.Pool, Sentinel: co}
	// Feature já em DRAFT_PR e duas correções criadas pela fábrica (REPROVADO_PR), mais uma conhecida em outro estado.
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "F-1", Kind: "feature", Status: statusflow.DraftPR}); err != nil {
		t.Fatal(err)
	}
	var fid string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'F-1','Feature') RETURNING id::text`, p.ID).Scan(&fid)
	_, _ = db.Pool.Exec(ctx, `UPDATE flow_issues SET feature_id=$2 WHERE project_id=$1 AND external_key='F-1'`, p.ID, fid)
	src := issuesync.Source{Kind: "jira", Container: "C", Map: syncMap("jira")}
	src.Map.Remote["Reprovado"] = "REPROVED_PR" // alias legado na origem
	src.Map.Remote["Corrigir"] = statusflow.CorrecaoPR
	mk := func(id, key, status string) integrations.Item {
		return integrations.Item{Provider: "jira", ID: id, Key: key, Title: "Correção " + key, Status: status, Version: "v1"}
	}
	for _, it := range []integrations.Item{mk("c1", "C-1", "Reprovado"), mk("c2", "C-2", "Reprovado")} {
		if err := sy.RegisterCorrection(ctx, p.ID, src, it, "F-1"); err != nil {
			t.Fatal(err)
		}
	}
	// Uma Issue conhecida que NÃO aguarda liberação (não deve ser consultada).
	_, _ = db.Pool.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, raw_status, status_code, revision, kind)
		VALUES ($1,'jira','other','O-1','Done','CONCLUIDO','v1','feature')`, p.ID)

	tr := &countingTracker{items: map[string]integrations.Item{"c1": mk("c1", "C-1", "Reprovado"), "c2": mk("c2", "C-2", "Reprovado")}}
	run, err := sy.Targeted(ctx, p.ID, tr, src)
	if err != nil || run.Outcome != "ok" || run.Read != 2 || tr.lists != 0 || len(tr.gets) != 2 || run.Trigger != "targeted" {
		t.Fatalf("%+v %v lists=%d gets=%v", run, err, tr.lists, tr.gets)
	}
	for _, k := range []string{"C-1", "C-2"} { // sem ação humana: nenhum despacho
		if iss, _ := co.Get(ctx, p.ID, k); iss.Status != statusflow.ReprovadoPR {
			t.Fatalf("%s avançou sem liberação: %+v", k, iss)
		}
	}
	// Humano libera C-1 na origem (revisão nova). Sync direcionado e geral concorrentes observam a mesma revisão.
	rel := mk("c1", "C-1", "Corrigir")
	rel.Version = "v2"
	tr.items["c1"] = rel
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := sy.Targeted(ctx, p.ID, tr, src); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if _, err := sy.General(ctx, p.ID, issuesync.TriggerManual, &fakeTracker{provider: "jira", items: map[string]integrations.Item{}}, src); err != nil {
		t.Fatal(err)
	}
	c1, _ := co.Get(ctx, p.ID, "C-1")
	c2, _ := co.Get(ctx, p.ID, "C-2")
	if c1.Status != statusflow.CorrecaoPR || c1.Version != 2 || c2.Status != statusflow.ReprovadoPR {
		t.Fatalf("%+v %+v", c1, c2)
	}
	var applied int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM flow_transitions t JOIN flow_issues i ON i.id=t.issue_id WHERE i.external_key='C-1' AND t.decision='applied'`).Scan(&applied)
	if applied != 1 {
		t.Fatalf("liberações aplicadas: %d", applied)
	}
	// O item saiu do conjunto de espera: o próximo ciclo consulta só C-2.
	tr.gets = nil
	if _, err := sy.Targeted(ctx, p.ID, tr, src); err != nil || len(tr.gets) != 1 || tr.gets[0] != "c2" {
		t.Fatalf("%v %v", tr.gets, err)
	}
}
