package integration

import (
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
)

type issuesView struct {
	Issues []issuesync.Issue `json:"issues"`
	Runs   []issuesync.Run   `json:"runs"`
}

func fullMoves() map[string]string {
	m := map[string]string{}
	for _, c := range statusflow.ProjectedCodes() {
		m[c] = "t-" + c
	}
	return m
}

// AJ01-02 / AJ02-01 — mapa de status, sync manual e página Issues pela API.
func TestIssuesAPIStatusMapManualSyncAndHistory(t *testing.T) {
	f := newSourceFixture(t, "issues-api")
	base := "/api/projects/" + f.p.ID
	// Sem mapa: sync recusado com motivo.
	if r := f.c.post(base+"/issues/sync", nil); r.StatusCode != 422 || r.ErrCode() != "status_map_missing" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	// Mapa inválido: 422 apontando o campo.
	bad := map[string]any{"version": 0, "source": "jira", "remote": map[string]string{"To Do": "NADA"}, "moves": fullMoves()}
	if r := f.c.do("PUT", base+"/status-map", bad, nil); r.StatusCode != 422 || r.ErrCode() != "invalid_status_map" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	good := map[string]any{"version": 0, "source": "jira", "remote": map[string]string{"To Do": "PRD-DOC", "In Progress": "IMPLEMENTACAO", "Done": "CONCLUIDO"}, "moves": fullMoves()}
	f.c.expect(f.c.do("PUT", base+"/status-map", good, nil), 200)
	if r := f.c.do("PUT", base+"/status-map", good, nil); r.StatusCode != 409 { // versão antiga
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	var view struct {
		Map   *statusflow.SavedMap `json:"map"`
		Codes []string             `json:"codes"`
	}
	f.c.expect(f.c.get(base+"/status-map"), 200).JSON(&view)
	if view.Map == nil || view.Map.Version != 1 || len(view.Codes) != 9 {
		t.Fatalf("%+v", view)
	}
	// Sync manual: as duas Issues em "To Do" entram; as demais são ignoradas.
	var run issuesync.Run
	f.c.expect(f.c.post(base+"/issues/sync", nil), 200).JSON(&run)
	if run.Outcome != "ok" || run.New != 2 || run.Ignored != 2 || run.Trigger != "manual" {
		t.Fatalf("%+v", run)
	}
	f.c.expect(f.c.post(base+"/issues/sync", nil), 200).JSON(&run)
	if run.New != 0 || run.Updated != 0 {
		t.Fatalf("%+v", run)
	}
	var iv issuesView
	f.c.expect(f.c.get(base+"/issues"), 200).JSON(&iv)
	if len(iv.Issues) != 2 || len(iv.Runs) != 2 || iv.Issues[0].StatusCode != "PRD-DOC" || iv.Runs[0].Trigger != "manual" {
		t.Fatalf("%+v", iv)
	}
	// Sem sessão: negado.
	anon := f.c.e.newClient()
	if r := anon.get(base + "/issues"); r.StatusCode != 401 {
		t.Fatalf("%d", r.StatusCode)
	}
}
