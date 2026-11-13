package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/stdflow"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type projWorkflow struct {
	WorkflowID string            `json:"workflow_id"`
	Version    int               `json:"version"`
	Created    bool              `json:"created"`
	Published  bool              `json:"published"`
	Issues     []workflows.Issue `json:"issues"`
}

// AJ05-01 / CT14 — novo projeto ganha o grafo padrão editável; repetir não duplica nem sobrescreve.
func TestNewProjectGetsTheStructuralWorkflowAndKeepsEdits(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	p := createProject(t, c, "wf-padrao")
	ctx := context.Background()
	var pw projWorkflow
	c.expect(c.get("/api/projects/"+p.ID+"/workflow"), 200).JSON(&pw)
	if pw.Created || pw.Published || pw.WorkflowID == "" {
		t.Fatalf("o cadastro já deveria ter criado o rascunho: %+v", pw)
	}
	var w wfResp
	c.expect(c.get("/api/workflows/"+pw.WorkflowID), 200).JSON(&w)
	if len(w.Graph.Nodes) != len(workflows.StandardGraph().Nodes) || w.Kind != "feature" {
		t.Fatalf("%d nós, tipo %s", len(w.Graph.Nodes), w.Kind)
	}
	// Diagnóstico por nó: agentes e comandos ainda sem escolha; nenhuma guarda removida.
	var required int
	for _, is := range pw.Issues {
		if is.Code != "required" {
			t.Fatalf("%+v", is)
		}
		required++
	}
	if required != 6 {
		t.Fatalf("pendências: %v", pw.Issues)
	}
	// O administrador edita o rascunho; o provisionamento repetido preserva a edição.
	g := w.Graph
	for i := range g.Nodes {
		if g.Nodes[i].ID == workflows.GuardPlanning {
			g.Nodes[i].Props["prompt"] = "Instrução editada pelo administrador"
		}
	}
	c.expect(saveWorkflow(c, w, w.Name, g), 200).JSON(&w)
	again, created, err := stdflow.Provision(ctx, e.db.Pool, p.ID, p.Name)
	if err != nil || created || again.ID != pw.WorkflowID || again.Version != w.Version {
		t.Fatalf("%+v %v %v", again, created, err)
	}
	var edited string
	for _, n := range again.Graph.Nodes {
		if n.ID == workflows.GuardPlanning {
			edited, _ = n.Props["prompt"].(string)
		}
	}
	if !strings.Contains(edited, "editada") {
		t.Fatalf("a edição foi sobrescrita: %q", edited)
	}
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM project_workflows WHERE project_id=$1`, p.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("cópias: %d", n)
	}
}

// AJ05-03 / CT14, CT16 — bindings incompletos não publicam; guarda contornada não publica; a execução
// fica presa à versão publicada.
func TestProjectWorkflowPublishNeedsBindingsAndKeepsGuards(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	p := createProject(t, c, "wf-publica")
	ctx := context.Background()
	var pw projWorkflow
	c.expect(c.get("/api/projects/"+p.ID+"/workflow"), 200).JSON(&pw)
	var w wfResp
	c.expect(c.get("/api/workflows/"+pw.WorkflowID), 200).JSON(&w)
	publish := func(w wfResp) apiResp {
		return c.post("/api/workflows/"+w.ID+"/publish", map[string]any{"expected_version": w.Version})
	}
	// Bindings incompletos: publicar é recusado com a lista por nó.
	resp := publish(w)
	if resp.StatusCode != 422 || resp.ErrCode() != "publish_blocked" || !strings.Contains(string(resp.Body), workflows.GuardPlanning) {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	// O administrador escolhe manualmente agente e comandos de cada etapa.
	acct := connWith(t, c, "cli", "claude-wf", "claude")
	body := agentBody("dev", &acct.ID)
	body["model"], body["tools"] = "", []string{"read_file"}
	c.expect(c.post("/api/agents", body), 201)
	g := w.Graph
	for i := range g.Nodes {
		switch g.Nodes[i].Type {
		case "agent":
			g.Nodes[i].Props["agent"] = "dev"
		case "checks":
			g.Nodes[i].Props["commands"] = "make test"
		}
	}
	c.expect(saveWorkflow(c, w, w.Name, g), 200).JSON(&w)
	// Atalho que pula checks, Draft PR e espera do merge: recusado mesmo com tudo configurado.
	bad := w.Graph
	bad.Nodes = append(append([]workflows.Node{}, bad.Nodes...), wfNode("atalho", "end", 0, 0, map[string]any{"outcome": "success"}))
	bad.Edges = append(append([]workflows.Edge{}, bad.Edges...), wfEdge("x", workflows.GuardConsolidate, "out", "atalho", "in"))
	var wbad wfResp
	c.expect(saveWorkflow(c, w, w.Name, bad), 200).JSON(&wbad)
	resp = publish(wbad)
	if resp.StatusCode != 422 || !strings.Contains(string(resp.Body), "guard_bypass") {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	// Restaurado: publica.
	var ok wfResp
	c.expect(saveWorkflow(c, wbad, w.Name, g), 200).JSON(&ok)
	var v1 releases.Version
	c.expect(publish(ok), 201).JSON(&v1)
	f := newRelEnvFor(t, e, p.ID).feature(t, "WF-1")
	ex, ver, err := releases.StartExecution(ctx, e.db.Pool, p.ID, f, pw.WorkflowID)
	if err != nil || ver.ID != v1.ID {
		t.Fatalf("%+v %v", ver, err)
	}
	// Mudança posterior do grafo vale para novos trabalhos; a execução em curso mantém a versão fixada.
	g2 := ok.Graph
	for i := range g2.Nodes {
		if g2.Nodes[i].ID == workflows.GuardPlanning {
			g2.Nodes[i].Props["prompt"] = "Nova instrução de planejamento"
		}
	}
	var w2 wfResp
	c.expect(saveWorkflow(c, ok, w.Name, g2), 200).JSON(&w2)
	var v2 releases.Version
	c.expect(publish(w2), 201).JSON(&v2)
	if v2.Number != 2 {
		t.Fatalf("%+v", v2)
	}
	pinned, err := releases.ExecutionVersion(ctx, e.db.Pool, ex.ID)
	if err != nil || pinned.ID != v1.ID {
		t.Fatalf("%+v %v", pinned, err)
	}
}

func newRelEnvFor(t *testing.T, e *authEnv, projectID string) *relEnv {
	t.Helper()
	return &relEnv{e: e, proj: projectID}
}
