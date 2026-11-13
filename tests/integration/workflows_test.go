package integration

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type wfResp struct {
	workflows.Workflow
	Issues []workflows.Issue `json:"issues"`
}

func wfNode(id, typ string, x, y float64, props map[string]any) workflows.Node {
	if props == nil {
		props = map[string]any{}
	}
	return workflows.Node{ID: id, Type: typ, X: x, Y: y, Props: props}
}

func wfEdge(id, from, fp, to, tp string) workflows.Edge {
	return workflows.Edge{ID: id, From: from, FromPort: fp, To: to, ToPort: tp}
}

// complexWorkflow usa todos os tipos de controle e um loop limitado.
func complexWorkflow() workflows.Graph {
	return workflows.Graph{
		Nodes: []workflows.Node{
			wfNode("s", "start", 40, 40, nil),
			wfNode("plan", "agent", 200, 40, map[string]any{"agent": "planejador", "prompt": "Planeje a feature", "timeout_minutes": 30}),
			wfNode("cond", "condition", 380, 40, map[string]any{"expression": "plan.ok == true"}),
			wfNode("loop", "loop", 560, 20, map[string]any{"max_iterations": 3, "condition": "checks.passed == false"}),
			wfNode("dev", "agent", 740, 20, map[string]any{"agent": "dev", "prompt": "Implemente"}),
			wfNode("chk", "checks", 920, 20, map[string]any{"commands": "make test\nmake lint", "profile": "go"}),
			wfNode("wait", "wait", 560, 200, map[string]any{"minutes": 5}),
			wfNode("hum", "human_decision", 740, 200, map[string]any{"question": "Aprovar a entrega?"}),
			wfNode("docs", "docs", 560, 360, map[string]any{"target": "docs/", "instruction": "Atualize a documentação"}),
			wfNode("git", "git", 740, 360, map[string]any{"operation": "draft_pr", "message": "Entrega"}),
			wfNode("join", "converge", 920, 280, map[string]any{"mode": "any"}),
			wfNode("ok", "end", 1100, 280, map[string]any{"outcome": "success"}),
			wfNode("bad", "end", 920, 200, map[string]any{"outcome": "failure"}),
		},
		Edges: []workflows.Edge{
			wfEdge("1", "s", "out", "plan", "in"), wfEdge("2", "plan", "out", "cond", "in"),
			wfEdge("3", "cond", "true", "loop", "in"), wfEdge("4", "cond", "false", "wait", "in"),
			wfEdge("5", "loop", "body", "dev", "in"), wfEdge("6", "dev", "out", "chk", "in"),
			wfEdge("7", "chk", "out", "loop", "in"), wfEdge("8", "loop", "done", "docs", "in"),
			wfEdge("9", "wait", "out", "hum", "in"), wfEdge("10", "hum", "approved", "join", "in"),
			wfEdge("11", "hum", "rejected", "bad", "in"), wfEdge("12", "docs", "out", "git", "in"),
			wfEdge("13", "git", "out", "join", "in"), wfEdge("14", "join", "out", "ok", "in"),
		},
	}
}

func createWorkflow(t *testing.T, c *client, name, kind string) wfResp {
	t.Helper()
	var w wfResp
	c.expect(c.post("/api/workflows", map[string]any{"name": name, "kind": kind}), 201).JSON(&w)
	return w
}

func saveWorkflow(c *client, w wfResp, name string, g workflows.Graph) apiResp {
	return c.do("PUT", "/api/workflows/"+w.ID, map[string]any{"expected_version": w.Version, "name": name, "description": "d", "graph": g}, nil)
}

func TestWorkflowCatalogAndStarterDraft(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	var cat struct{ Items []workflows.NodeType }
	c.expect(c.get("/api/workflows/node-types"), 200).JSON(&cat)
	if len(cat.Items) != len(workflows.Catalog()) {
		t.Fatalf("catálogo: %d tipos", len(cat.Items))
	}
	w := createWorkflow(t, c, "Feature padrão", "feature")
	if w.Version != 1 || len(w.Graph.Nodes) != 2 || len(w.Graph.Edges) != 1 || len(w.Issues) != 0 {
		t.Fatalf("rascunho inicial deve ser válido: %+v", w)
	}
	var list struct{ Items []workflows.Summary }
	c.expect(c.get("/api/workflows"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].Nodes != 2 || list.Items[0].Issues != 0 {
		t.Fatalf("listagem: %+v", list.Items)
	}
	if r := c.post("/api/workflows", map[string]any{"name": "Feature padrão", "kind": "feature"}); r.StatusCode != 409 {
		t.Fatalf("nome duplicado: %d", r.StatusCode)
	}
	for _, body := range []map[string]any{{"name": "", "kind": "feature"}, {"name": "x", "kind": "outro"}} {
		if r := c.post("/api/workflows", body); r.StatusCode != 400 {
			t.Errorf("%v: %d", body, r.StatusCode)
		}
	}
}

// T07-01 — "Montar fluxo complexo, salvar e reabrir mantendo nós, conexões e propriedades."
func TestComplexWorkflowSurvivesSaveAndReopen(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	w := createWorkflow(t, c, "Fluxo complexo", "feature")
	g := complexWorkflow()
	var saved wfResp
	c.expect(saveWorkflow(c, w, "Fluxo complexo", g), 200).JSON(&saved)
	if len(saved.Issues) != 0 || saved.Version != 2 {
		t.Fatalf("o fluxo complexo deveria ser válido: %+v", saved.Issues)
	}

	// Reabre em outra sessão e compara tudo.
	c2 := e.newClient()
	c2.login(c.secret)
	var got wfResp
	c2.expect(c2.get("/api/workflows/"+w.ID), 200).JSON(&got)
	want, _ := json.Marshal(g)
	have, _ := json.Marshal(got.Graph)
	var wantV, haveV any
	_ = json.Unmarshal(want, &wantV)
	_ = json.Unmarshal(have, &haveV)
	if !reflect.DeepEqual(wantV, haveV) {
		t.Fatalf("o grafo mudou ao reabrir:\nquerido: %s\nlido:    %s", want, have)
	}
	if len(got.Graph.Nodes) != 13 || len(got.Graph.Edges) != 14 || got.Graph.Nodes[3].Props["max_iterations"] != float64(3) {
		t.Fatalf("nós, conexões e propriedades: %+v", got.Graph)
	}
}

// T07-01 — "Exibir erro com nó e campo relevantes."
func TestInvalidDraftIsSavedAndReportsNodeAndField(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	w := createWorkflow(t, c, "Rascunho incompleto", "feature")
	g := complexWorkflow()
	g.Nodes[1].Props["prompt"] = ""            // agente sem instrução
	delete(g.Nodes[3].Props, "max_iterations") // loop sem limite
	g.Edges = g.Edges[:len(g.Edges)-1]         // converge sem saída
	var saved wfResp
	c.expect(saveWorkflow(c, w, "Rascunho incompleto", g), 200).JSON(&saved)
	want := map[[3]string]bool{
		{"plan", "prompt", "required"}: true, {"loop", "max_iterations", "required"}: true, {"join", "out", "output_unconnected"}: true,
	}
	for _, is := range saved.Issues {
		delete(want, [3]string{is.Node, is.Field, is.Code})
		if is.Message == "" {
			t.Errorf("pendência sem mensagem: %+v", is)
		}
	}
	if len(want) != 0 {
		t.Fatalf("faltaram problemas %v em %+v", want, saved.Issues)
	}
	// O rascunho inválido foi gravado e os problemas voltam na leitura e na listagem.
	var got wfResp
	c.expect(c.get("/api/workflows/"+w.ID), 200).JSON(&got)
	if !reflect.DeepEqual(got.Issues, saved.Issues) || len(got.Graph.Nodes) != 13 {
		t.Fatalf("leitura difere do salvo: %+v", got.Issues)
	}
	var list struct{ Items []workflows.Summary }
	c.expect(c.get("/api/workflows"), 200).JSON(&list)
	if list.Items[0].Issues != len(saved.Issues) {
		t.Fatalf("listagem: %+v", list.Items[0])
	}
}

func TestWorkflowSaveRejectsStaleBadAndOversizedInput(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	w := createWorkflow(t, c, "Limites", "subflow")
	g := workflows.Starter()
	c.expect(saveWorkflow(c, w, "Limites", g), 200)
	if r := saveWorkflow(c, w, "Limites", g); r.StatusCode != 409 || r.ErrCode() != "version_conflict" {
		t.Fatalf("versão obsoleta: %d %s", r.StatusCode, r.Body)
	}
	cur := w
	cur.Version = 2
	// Sem expected_version, campo desconhecido, nome vazio e grafo grande demais.
	if r := c.do("PUT", "/api/workflows/"+w.ID, map[string]any{"name": "x", "graph": g}, nil); r.StatusCode != 400 {
		t.Errorf("sem versão: %d", r.StatusCode)
	}
	if r := c.do("PUT", "/api/workflows/"+w.ID, map[string]any{"expected_version": 2, "name": "x", "graph": g, "kind": "feature"}, nil); r.StatusCode != 400 {
		t.Errorf("campo desconhecido: %d", r.StatusCode)
	}
	if r := saveWorkflow(c, cur, "  ", g); r.StatusCode != 400 {
		t.Errorf("nome vazio: %d", r.StatusCode)
	}
	big := workflows.Graph{Nodes: make([]workflows.Node, workflows.MaxNodes+1)}
	for i := range big.Nodes {
		big.Nodes[i] = wfNode("n"+strconv.Itoa(i), "wait", 0, 0, map[string]any{"minutes": 1})
	}
	if r := saveWorkflow(c, cur, "Limites", big); r.StatusCode != 400 {
		t.Errorf("grafo grande: %d %s", r.StatusCode, r.Body)
	}
	huge := workflows.Starter()
	huge.Nodes[0].Props = map[string]any{"x": strings.Repeat("a", 2<<20)}
	if r := saveWorkflow(c, cur, "Limites", huge); r.StatusCode != 400 {
		t.Errorf("corpo enorme: %d", r.StatusCode)
	}
	var got wfResp
	c.expect(c.get("/api/workflows/"+w.ID), 200).JSON(&got)
	if got.Version != 2 {
		t.Fatalf("falhas não podem mudar a versão: %d", got.Version)
	}
	if r := c.get("/api/workflows/nao-e-uuid"); r.StatusCode != 404 {
		t.Errorf("id inválido: %d", r.StatusCode)
	}
}

func TestConcurrentWorkflowSavesHaveOneWinner(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	w := createWorkflow(t, c, "Corrida", "feature")
	const n = 8
	var wg sync.WaitGroup
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- saveWorkflow(c, w, "Corrida", workflows.Starter()).StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	ok, conflict := 0, 0
	for code := range codes {
		switch code {
		case 200:
			ok++
		case 409:
			conflict++
		}
	}
	if ok != 1 || conflict != n-1 {
		t.Fatalf("esperado 1 vencedor e %d conflitos: %d/%d", n-1, ok, conflict)
	}
}

func TestWorkflowRoutesRequireSessionAndCSRF(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	w := createWorkflow(t, c, "Seguro", "feature")
	anon := e.newClient()
	for _, r := range []apiResp{
		anon.get("/api/workflows"), anon.get("/api/workflows/node-types"), anon.get("/api/workflows/" + w.ID),
		anon.post("/api/workflows", map[string]any{"name": "x", "kind": "feature"}),
		anon.do("PUT", "/api/workflows/"+w.ID, map[string]any{"expected_version": 1}, nil),
	} {
		if r.StatusCode != 401 {
			t.Errorf("sem sessão: %d", r.StatusCode)
		}
	}
	for _, r := range []apiResp{
		c.do("POST", "/api/workflows", map[string]any{"name": "y", "kind": "feature"}, map[string]string{"X-CSRF-Token": ""}),
		c.do("PUT", "/api/workflows/"+w.ID, map[string]any{"expected_version": 1, "name": "forjado", "graph": workflows.Starter()}, map[string]string{"X-CSRF-Token": ""}),
	} {
		if r.StatusCode != 403 {
			t.Errorf("sem CSRF: %d", r.StatusCode)
		}
	}
	var got wfResp
	c.expect(c.get("/api/workflows/"+w.ID), 200).JSON(&got)
	if got.Name != "Seguro" {
		t.Fatal("requisição forjada alterou o workflow")
	}
}

func TestValidateEndpointDoesNotPersist(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	w := createWorkflow(t, c, "Só valida", "feature")
	g := complexWorkflow()
	g.Nodes[1].Props["agent"] = ""
	var out struct{ Issues []workflows.Issue }
	c.expect(c.post("/api/workflows/validate", map[string]any{"graph": g}), 200).JSON(&out)
	if len(out.Issues) != 1 || out.Issues[0].Node != "plan" || out.Issues[0].Field != "agent" {
		t.Fatalf("problemas: %+v", out.Issues)
	}
	var got wfResp
	c.expect(c.get("/api/workflows/"+w.ID), 200).JSON(&got)
	if got.Version != 1 || len(got.Graph.Nodes) != 2 {
		t.Fatalf("validar não pode gravar: %+v", got)
	}
	if r := c.post("/api/workflows/validate", map[string]any{"graph": g, "x": 1}); r.StatusCode != 400 {
		t.Fatalf("campo extra: %d", r.StatusCode)
	}
}
