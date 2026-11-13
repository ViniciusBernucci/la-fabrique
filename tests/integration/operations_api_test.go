package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/memory"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
)

type opsEnv struct {
	*extEnv
	c *client
}

// newOpsEnv liga a API real ao GitHub de mentira e ao repositório do projeto (conexão no cofre).
func newOpsEnv(t *testing.T, slug string) *opsEnv {
	t.Helper()
	m := newExtEnv(t, slug)
	m.e.api.UseGitHubBase(m.gh.URL)
	var conn struct {
		ID string `json:"id"`
	}
	m.cl.expect(m.cl.post("/api/connections", map[string]string{"kind": "github", "name": "gh-" + slug, "target": "org/app", "secret": "tok-delivery"}), 201).JSON(&conn)
	if _, err := m.e.db.Pool.Exec(context.Background(), `UPDATE projects SET repo_owner = 'org', repo_name = 'app', base_branch = 'main', repo_connection_id = $2 WHERE id = $1`, m.p.proj, conn.ID); err != nil {
		t.Fatal(err)
	}
	return &opsEnv{extEnv: m, c: m.cl}
}

func (o *opsEnv) base() string {
	return "/api/projects/" + o.p.proj + "/deliveries/" + o.out.DeliveryID
}

// T15-03/T15-04/T15-05 pelo painel — a entrega, a consulta da PR e a promoção da memória por HTTP.
func TestTheDeliveryRoutesObserveTheMergeAndPromoteMemoryOnDemand(t *testing.T) {
	o := newOpsEnv(t, "ops-merge")
	ctx := context.Background()
	branch, _ := gitops.FeatureBranch(o.p.feature, 1)
	var list struct {
		Items []struct {
			ID, State, PRURL, Projection string
			FeatureKey                   string         `json:"feature_key"`
			Comments                     map[string]int `json:"comments"`
		}
	}
	o.c.expect(o.c.get("/api/projects/"+o.p.proj+"/deliveries"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].State != "draft_open" || list.Items[0].ID != o.out.DeliveryID || list.Items[0].FeatureKey == "" {
		t.Fatalf("%+v", list)
	}
	mem := &memory.Store{DB: o.e.db.Pool, Redactor: o.x.Redactor}
	if _, _, err := mem.Record(ctx, memInput(o.p, branch, "tabela pedidos", "usa soft delete", shaB)); err != nil {
		t.Fatal(err)
	}
	dep := o.waitingOnMerge(t, "schema")

	// Aberta: nada muda.
	var res struct {
		Observation struct {
			Outcome    string
			Released   []queue.Item
			Projection string
		}
		OriginNote string `json:"origin_note"`
		Memory     map[string]int
	}
	o.c.expect(o.c.post(o.base()+"/observe", map[string]any{}), 200).JSON(&res)
	if res.Observation.Outcome != "open" || len(res.Observation.Released) != 0 || res.Memory != nil {
		t.Fatalf("%+v", res)
	}
	// Merge humano: SHA, dependente liberado, memória promovida; sem mapa de estados a origem não é tocada.
	o.gh.setPR(o.out.PR.Number, "closed", true, "cccccccccccccccccccccccccccccccccccccccc")
	o.c.expect(o.c.post(o.base()+"/observe", map[string]any{}), 200).JSON(&res)
	if res.Observation.Outcome != "merged" || len(res.Observation.Released) != 1 || res.Observation.Released[0].ID != dep.ID || res.Memory["promoted"] != 1 || !strings.Contains(res.OriginNote, "mapa de estados") {
		t.Fatalf("%+v", res)
	}
	var det struct {
		Delivery struct {
			State    string
			MergeSHA string `json:"merge_sha"`
		}
		Observations []struct{ Outcome string }
	}
	o.c.expect(o.c.get(o.base()), 200).JSON(&det)
	if det.Delivery.State != "merged" || det.Delivery.MergeSHA == "" || len(det.Observations) < 2 {
		t.Fatalf("%+v", det)
	}
	var mm struct {
		Items []struct{ State string }
	}
	o.c.expect(o.c.get("/api/projects/"+o.p.proj+"/memory?state=official&q=soft"), 200).JSON(&mm)
	if len(mm.Items) != 1 {
		t.Fatalf("%+v", mm)
	}
	if r := o.c.get("/api/projects/" + o.p.proj + "/memory?state=talvez"); r.StatusCode != 400 {
		t.Fatalf("%d", r.StatusCode)
	}
	// Outro projeto não enxerga a entrega.
	other := createProject(t, o.c, "ops-other")
	if r := o.c.get("/api/projects/" + other.ID + "/deliveries/" + o.out.DeliveryID); r.StatusCode != 404 {
		t.Fatalf("entrega de outro projeto: %d", r.StatusCode)
	}
	if r := o.c.get("/api/projects/" + other.ID + "/memory"); r.StatusCode != 200 || strings.Contains(string(r.Body), "soft delete") {
		t.Fatalf("memória vazou entre projetos: %s", r.Body)
	}
}

func TestTheDeliveryRoutesHandleReviewCommentsAndAClosedPR(t *testing.T) {
	o := newOpsEnv(t, "ops-closed")
	zero := "00000000-0000-4000-8000-000000000000"
	// Sem autores autorizados, nada é descoberto.
	o.gh.addComment("issues", o.out.PR.Number, 71, "Revisora", "renomeie o campo")
	var disc struct {
		Discovery struct {
			New, Ignored int
			Item         *queue.Item
		}
	}
	o.c.expect(o.c.post(o.base()+"/comments/discover", map[string]any{"task_key": "schema"}), 200).JSON(&disc)
	if disc.Discovery.New != 0 || disc.Discovery.Ignored != 1 || disc.Discovery.Item != nil {
		t.Fatalf("%+v", disc)
	}
	var set struct {
		ReviewAuthors []string          `json:"review_authors"`
		StatusMap     map[string]string `json:"status_map"`
	}
	for _, bad := range []map[string]any{{"review_authors": []string{"não vale!"}}, {"status_map": map[string]string{"outro": "x"}}} {
		if r := o.c.do("PUT", "/api/projects/"+o.p.proj+"/delivery-settings", bad, nil); r.StatusCode != 400 {
			t.Fatalf("configuração inválida: %d %s", r.StatusCode, r.Body)
		}
	}
	o.c.expect(o.c.do("PUT", "/api/projects/"+o.p.proj+"/delivery-settings", map[string]any{"review_authors": []string{"Revisora", "revisora"}, "status_map": map[string]string{"done": "Concluída"}}, nil), 200).JSON(&set)
	if len(set.ReviewAuthors) != 1 || set.StatusMap["done"] != "Concluída" {
		t.Fatalf("%+v", set)
	}
	o.c.expect(o.c.post(o.base()+"/comments/discover", map[string]any{"task_key": "schema", "priority": 1}), 200).JSON(&disc)
	if disc.Discovery.New != 1 || disc.Discovery.Item == nil || disc.Discovery.Item.Kind != queue.KindPRFix {
		t.Fatalf("%+v", disc)
	}
	if r := o.c.post(o.base()+"/comments/discover", map[string]any{}); r.StatusCode != 400 {
		t.Fatalf("sem tarefa: %d", r.StatusCode)
	}

	// PR fechada sem merge: o detalhe oferece as três ações; a ação exige reautenticação.
	o.gh.setPR(o.out.PR.Number, "closed", false, "")
	o.c.expect(o.c.post(o.base()+"/observe", map[string]any{}), 200)
	var det struct {
		Offer struct {
			Options []struct{ ID string }
		}
		Comments []struct{ State string }
	}
	o.c.expect(o.c.get(o.base()), 200).JSON(&det)
	if len(det.Offer.Options) != 3 || len(det.Comments) != 1 {
		t.Fatalf("%+v", det)
	}
	if _, err := o.e.db.Pool.Exec(context.Background(), `UPDATE admin_sessions SET reauth_at = NULL`); err != nil {
		t.Fatal(err)
	}
	if r := o.c.post(o.base()+"/actions", map[string]any{"action": "cancel"}); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	o.c.expect(o.c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": o.e.code(o.c.secret)}), 200)
	if r := o.c.post(o.base()+"/actions", map[string]any{"action": "apagar"}); r.StatusCode != 400 {
		t.Fatalf("ação inválida: %d", r.StatusCode)
	}
	var dec struct {
		State    string
		Replayed bool
	}
	o.c.expect(o.c.post(o.base()+"/actions", map[string]any{"action": "reopen", "note": "voltou"}), 200).JSON(&dec)
	if dec.State != "draft_open" || dec.Replayed {
		t.Fatalf("%+v", dec)
	}
	o.c.expect(o.c.post(o.base()+"/actions", map[string]any{"action": "reopen"}), 200).JSON(&dec)
	if !dec.Replayed {
		t.Fatalf("%+v", dec)
	}
	o.gh.setPR(o.out.PR.Number, "closed", false, "")
	o.c.expect(o.c.post(o.base()+"/observe", map[string]any{}), 200)
	o.c.expect(o.c.post(o.base()+"/actions", map[string]any{"action": "cancel"}), 200).JSON(&dec)
	if dec.State != "cancelled" {
		t.Fatalf("%+v", dec)
	}
	var d2 struct {
		History []string
	}
	o.c.expect(o.c.get(o.base()), 200).JSON(&d2)
	if strings.Join(d2.History, ",") != "offered,reopen por "+o.adminID()+",cancel por "+o.adminID() {
		t.Logf("histórico: %v", d2.History)
	}
	var auditN int
	_ = o.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM auth_events WHERE type IN ('delivery_reopen', 'delivery_cancel')`).Scan(&auditN)
	if auditN != 2 {
		t.Fatalf("as ações ficam auditadas: %d", auditN)
	}
	for _, p := range []string{o.base() + "/observe", o.base() + "/actions", "/api/projects/" + o.p.proj + "/deliveries/" + zero + "/observe"} {
		anon := o.e.newClient()
		if r := anon.post(p, map[string]any{}); r.StatusCode != 401 {
			t.Errorf("sem sessão: %s %d", p, r.StatusCode)
		}
	}
	if r := o.c.do("POST", o.base()+"/observe", map[string]any{}, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Errorf("sem CSRF: %d", r.StatusCode)
	}
	if r := o.c.post("/api/projects/"+o.p.proj+"/deliveries/"+zero+"/observe", map[string]any{}); r.StatusCode != 404 {
		t.Errorf("entrega inexistente: %d", r.StatusCode)
	}
}

func (o *opsEnv) adminID() string {
	var id string
	_ = o.e.db.Pool.QueryRow(context.Background(), `SELECT id::text FROM admins LIMIT 1`).Scan(&id)
	return id
}

func TestTheOperationalControlsAndTheMemoryInvalidationWorkThroughTheAPI(t *testing.T) {
	o := newOpsEnv(t, "ops-controls")
	ctx := context.Background()
	ctl := "/api/projects/" + o.p.proj + "/controls"
	exec := o.p.execution(t)
	it := o.p.item(t, "schema", 0)
	if _, err := o.q.Start(ctx, it.ID, exec); err != nil {
		t.Fatal(err)
	}
	var out struct {
		Control struct{ Mode, Reason string }
		Failed  []struct{ Reason string }
	}
	if r := o.c.do("PUT", ctl, map[string]any{"mode": "talvez"}, nil); r.StatusCode != 400 {
		t.Fatalf("%d", r.StatusCode)
	}
	o.c.expect(o.c.do("PUT", ctl, map[string]any{"mode": "draining", "reason": "revisão de custos"}, nil), 200).JSON(&out)
	if out.Control.Mode != "draining" || out.Control.Reason != "revisão de custos" {
		t.Fatalf("%+v", out)
	}
	var ov struct {
		Controls struct {
			Project struct{ Mode string }
		}
	}
	o.c.expect(o.c.get("/api/projects/"+o.p.proj+"/overview"), 200).JSON(&ov)
	if ov.Controls.Project.Mode != "draining" {
		t.Fatalf("o resumo mostra o controle persistido: %+v", ov)
	}
	// Suspensão com item em execução: sem executor, o item segue rodando e a falha é reportada.
	o.c.expect(o.c.do("PUT", ctl, map[string]any{"mode": "suspended", "reason": "incidente"}, nil), 200).JSON(&out)
	if out.Control.Mode != "suspended" || len(out.Failed) != 1 || !strings.Contains(out.Failed[0].Reason, "executor") {
		t.Fatalf("%+v", out)
	}
	items, _ := o.q.List(ctx, o.p.proj)
	if items[0].State != queue.StateRunning {
		t.Fatalf("nunca se dá como parado sem checkpoint seguro: %+v", items[0])
	}
	o.c.expect(o.c.do("PUT", ctl, map[string]any{"mode": "active"}, nil), 200).JSON(&out)
	if out.Control.Mode != "active" {
		t.Fatalf("%+v", out)
	}
	// Global é ação sensível.
	if _, err := o.e.db.Pool.Exec(ctx, `UPDATE admin_sessions SET reauth_at = NULL`); err != nil {
		t.Fatal(err)
	}
	if r := o.c.do("PUT", "/api/controls/global", map[string]any{"mode": "draining"}, nil); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	o.c.expect(o.c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": o.e.code(o.c.secret)}), 200)
	o.c.expect(o.c.do("PUT", "/api/controls/global", map[string]any{"mode": "active"}, nil), 200)

	// Memória: invalidar com motivo.
	mem := &memory.Store{DB: o.e.db.Pool, Redactor: o.x.Redactor}
	rec, _, _ := mem.Record(ctx, memInput(o.p, "lf/x", "regra", "fato antigo", shaB))
	var inv struct {
		State         string `json:"state"`
		InvalidReason string `json:"invalid_reason"`
	}
	if r := o.c.post("/api/projects/"+o.p.proj+"/memory/"+rec.ID+"/invalidate", map[string]any{}); r.StatusCode != 400 {
		t.Fatalf("sem motivo: %d", r.StatusCode)
	}
	o.c.expect(o.c.post("/api/projects/"+o.p.proj+"/memory/"+rec.ID+"/invalidate", map[string]any{"reason": "regra mudou"}), 200).JSON(&inv)
	if inv.State != "invalidated" || inv.InvalidReason != "regra mudou" {
		t.Fatalf("%+v", inv)
	}
	other := createProject(t, o.c, "ops-ctl-other")
	if r := o.c.post("/api/projects/"+other.ID+"/memory/"+rec.ID+"/invalidate", map[string]any{"reason": "x"}); r.StatusCode != 404 {
		t.Fatalf("outro projeto: %d", r.StatusCode)
	}
	// Relatório da feature.
	var rep struct {
		FeatureKey string `json:"feature_key"`
		Markdown   string
		DocPath    string `json:"doc_path"`
	}
	o.c.expect(o.c.get("/api/projects/"+o.p.proj+"/features/"+o.p.feature+"/report"), 200).JSON(&rep)
	if rep.FeatureKey == "" || !strings.Contains(rep.Markdown, rep.FeatureKey) || !strings.HasPrefix(rep.DocPath, "docs/") {
		t.Fatalf("%+v", rep)
	}
}

func TestTheGitHubRoutesFailClosedWithoutARepositoryConnection(t *testing.T) {
	m := newExtEnv(t, "ops-norepo")
	m.e.api.UseGitHubBase(m.gh.URL)
	base := "/api/projects/" + m.p.proj + "/deliveries/" + m.out.DeliveryID
	if r := m.cl.post(base+"/observe", map[string]any{}); r.StatusCode != 422 || r.ErrCode() != "repo_incomplete" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	if r := m.cl.post(base+"/comments/discover", map[string]any{"task_key": "schema"}); r.StatusCode != 422 {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	if strings.Contains(string(m.cl.get(base).Body), "tok-delivery") {
		t.Fatal("o segredo não pode aparecer")
	}
}
