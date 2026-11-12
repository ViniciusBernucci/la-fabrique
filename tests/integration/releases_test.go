package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type relEnv struct {
	e    *authEnv
	c    *client
	proj string
}

func newRelEnv(t *testing.T) *relEnv {
	t.Helper()
	e, c, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "releases")
	return &relEnv{e: e, c: c, proj: p.ID}
}

func (r *relEnv) agent(t *testing.T, name string, conn *string) {
	t.Helper()
	b := agentBody(name, conn)
	b["model"], b["tools"] = "", []string{"read_file"}
	r.c.expect(r.c.post("/api/agents", b), 201)
}

func linear(nodes ...workflows.Node) workflows.Graph {
	g := workflows.Graph{Nodes: []workflows.Node{wfNode("s", "start", 0, 0, nil)}}
	prev := "s"
	for i, n := range nodes {
		g.Nodes = append(g.Nodes, n)
		g.Edges = append(g.Edges, wfEdge(string(rune('a'+i)), prev, "out", n.ID, "in"))
		prev = n.ID
	}
	g.Nodes = append(g.Nodes, wfNode("e", "end", 0, 0, map[string]any{"outcome": "success"}))
	g.Edges = append(g.Edges, wfEdge("z", prev, "out", "e", "in"))
	return g
}

func (r *relEnv) draft(t *testing.T, name, kind string, g workflows.Graph) wfResp {
	t.Helper()
	w := createWorkflow(t, r.c, name, kind)
	r.c.expect(saveWorkflow(r.c, w, name, g), 200).JSON(&w)
	return w
}

func (r *relEnv) publish(w wfResp) apiResp {
	return r.c.post("/api/workflows/"+w.ID+"/publish", map[string]any{"expected_version": w.Version})
}

func agentNode(prompt string) workflows.Node {
	return wfNode("dev", "agent", 0, 0, map[string]any{"agent": "dev", "prompt": prompt})
}

func (r *relEnv) feature(t *testing.T, ext string) string {
	t.Helper()
	f, err := persistence.CreateFeature(context.Background(), r.e.db.Pool, r.proj, ext, "Feature "+ext)
	if err != nil {
		t.Fatal(err)
	}
	return f.ID
}

func TestPublishValidatesReferencesAndFreezesASanitizedEffectiveConfig(t *testing.T) {
	r := newRelEnv(t)
	w := r.draft(t, "fluxo", "feature", linear(agentNode("faça")))
	// Agente inexistente bloqueia, com a lista de problemas.
	resp := r.publish(w)
	if resp.StatusCode != 422 || !strings.Contains(string(resp.Body), "agent_not_found") || resp.ErrCode() != "publish_blocked" {
		t.Fatalf("%d %s", resp.StatusCode, resp.Body)
	}
	acct := connWith(t, r.c, "cli", "claude-a", "claude")
	r.agent(t, "dev", &acct.ID)
	var v releases.Version
	r.c.expect(r.publish(w), 201).JSON(&v)
	if v.Number != 1 || v.DraftVersion != w.Version || len(v.Effective) != 1 || v.Effective[0].Agent == nil || v.Effective[0].Agent.Sources["model"] != "agent" || len(v.Effective[0].Tools) != 1 {
		t.Fatalf("%+v", v)
	}
	if body := string(r.c.get("/api/workflow-versions/" + v.ID).Body); strings.Contains(body, "marcador-") {
		t.Fatal("o snapshot não pode conter segredo")
	}
	// Idempotente: o mesmo conteúdo devolve a versão vigente.
	var again releases.Version
	r.c.expect(r.publish(w), 200).JSON(&again)
	if again.ID != v.ID || !again.Unchanged {
		t.Fatalf("%+v", again)
	}
	// Versão de rascunho obsoleta.
	stale := w
	stale.Version = 99
	if rs := r.publish(stale); rs.StatusCode != 409 {
		t.Fatalf("%d", rs.StatusCode)
	}
	// Nova edição → nova versão; a lista as mostra.
	var w2 wfResp
	r.c.expect(saveWorkflow(r.c, w, "fluxo", linear(agentNode("faça de outro jeito"))), 200).JSON(&w2)
	var v2 releases.Version
	r.c.expect(r.publish(w2), 201).JSON(&v2)
	var list struct{ Items []releases.Version }
	r.c.expect(r.c.get("/api/workflows/"+w.ID+"/versions"), 200).JSON(&list)
	if v2.Number != 2 || len(list.Items) != 2 || list.Items[0].Number != 2 {
		t.Fatalf("%+v %+v", v2, list.Items)
	}
	// Modelo incompatível (API sem modelo cadastrado) e rascunho com problema também bloqueiam.
	api := connWith(t, r.c, "api", "gw", "openrouter")
	b := agentBody("api-agent", &api.ID)
	b["provider"], b["model"], b["tools"] = "api", "vendor/nao-cadastrado", []string{}
	r.c.expect(r.c.post("/api/agents", b), 201)
	wm := r.draft(t, "fluxo-api", "feature", linear(wfNode("dev", "agent", 0, 0, map[string]any{"agent": "api-agent", "prompt": "x"})))
	if rs := r.publish(wm); rs.StatusCode != 422 || !strings.Contains(string(rs.Body), "model_incompatible") {
		t.Fatalf("%d %s", rs.StatusCode, rs.Body)
	}
	wb := createWorkflow(t, r.c, "quebrado", "feature")
	bad := workflows.Graph{Nodes: []workflows.Node{wfNode("s", "start", 0, 0, nil)}}
	r.c.expect(saveWorkflow(r.c, wb, "quebrado", bad), 200).JSON(&wb)
	if rs := r.publish(wb); rs.StatusCode != 422 {
		t.Fatalf("grafo inválido: %d", rs.StatusCode)
	}
}

// T07-03 — "Rascunho não executa. Publicação nova não muda execução existente."
func TestDraftDoesNotRunAndANewPublicationDoesNotChangeAnExistingExecution(t *testing.T) {
	r := newRelEnv(t)
	ctx := context.Background()
	acct := connWith(t, r.c, "cli", "claude-a", "claude")
	r.agent(t, "dev", &acct.ID)
	w := r.draft(t, "fluxo", "feature", linear(agentNode("v1")))
	feat := r.feature(t, "F-1")
	if _, _, err := releases.StartExecution(ctx, r.e.db.Pool, r.proj, feat, w.ID); !errors.Is(err, releases.ErrNotPublished) {
		t.Fatalf("rascunho não executa: %v", err)
	}
	var n int
	_ = r.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM executions WHERE feature_id = $1`, feat).Scan(&n)
	if n != 0 {
		t.Fatal("nenhuma execução pode nascer")
	}
	var v1 releases.Version
	r.c.expect(r.publish(w), 201).JSON(&v1)
	ex1, ver, err := releases.StartExecution(ctx, r.e.db.Pool, r.proj, feat, w.ID)
	if err != nil || ver.ID != v1.ID {
		t.Fatalf("%v", err)
	}
	var w2 wfResp
	r.c.expect(saveWorkflow(r.c, w, "fluxo", linear(agentNode("v2"))), 200).JSON(&w2)
	var v2 releases.Version
	r.c.expect(r.publish(w2), 201).JSON(&v2)
	// A execução existente segue na v1; a nova usa a v2.
	if got, err := releases.ExecutionVersion(ctx, r.e.db.Pool, ex1.ID); err != nil || got.ID != v1.ID || got.ContentHash != v1.ContentHash {
		t.Fatalf("%+v %v", got, err)
	}
	ex2, ver2, err := releases.StartExecution(ctx, r.e.db.Pool, r.proj, feat, w.ID)
	if err != nil || ver2.ID != v2.ID {
		t.Fatalf("%v", err)
	}
	if got, _ := releases.ExecutionVersion(ctx, r.e.db.Pool, ex2.ID); got.ID != v2.ID {
		t.Fatalf("%+v", got)
	}
	// Mudar o agente do catálogo depois não muda o que foi congelado.
	var a struct{ ID string }
	for _, it := range func() []struct{ ID, Name string } {
		var l struct{ Items []struct{ ID, Name string } }
		r.c.expect(r.c.get("/api/agents"), 200).JSON(&l)
		return l.Items
	}() {
		a.ID = it.ID
	}
	b := agentBody("dev", &acct.ID)
	b["model"], b["tools"], b["instructions"] = "", []string{"read_file"}, "Instruções NOVAS do catálogo."
	r.c.expect(r.c.do("PUT", "/api/agents/"+a.ID, map[string]any{"expected_version": 1, "name": "dev", "description": "d", "instructions": "Instruções NOVAS do catálogo.", "provider": "claude", "connection_id": acct.ID, "model": "", "skills": []string{}, "tools": []string{"read_file"}, "timeout_minutes": 30}, nil), 200)
	frozen, _ := releases.ExecutionVersion(ctx, r.e.db.Pool, ex1.ID)
	if strings.Contains(frozen.Effective[0].Agent.Instructions, "NOVAS") {
		t.Fatal("a configuração congelada não acompanha o catálogo")
	}
	// Despublicar impede novas execuções e não mexe nas existentes.
	r.c.expect(r.c.post("/api/workflows/"+w.ID+"/unpublish", nil), 204)
	if _, _, err := releases.StartExecution(ctx, r.e.db.Pool, r.proj, feat, w.ID); !errors.Is(err, releases.ErrNotPublished) {
		t.Fatalf("%v", err)
	}
	if got, err := releases.ExecutionVersion(ctx, r.e.db.Pool, ex1.ID); err != nil || got.ID != v1.ID {
		t.Fatalf("%v", err)
	}
}

func TestPublishedVersionsAreImmutableAndSubflowsArePinned(t *testing.T) {
	r := newRelEnv(t)
	ctx := context.Background()
	acct := connWith(t, r.c, "cli", "claude-a", "claude")
	r.agent(t, "dev", &acct.ID)
	sub := r.draft(t, "sub", "subflow", linear(agentNode("sub v1")))
	main := r.draft(t, "main", "feature", linear(wfNode("sf", "subflow", 0, 0, map[string]any{"workflow_id": sub.ID})))
	// Subfluxo não publicado bloqueia.
	if rs := r.publish(main); rs.StatusCode != 422 || !strings.Contains(string(rs.Body), "não está publicado") {
		t.Fatalf("%d %s", rs.StatusCode, rs.Body)
	}
	var sv1 releases.Version
	r.c.expect(r.publish(sub), 201).JSON(&sv1)
	var mv releases.Version
	r.c.expect(r.publish(main), 201).JSON(&mv)
	if mv.Subflows[sub.ID] != sv1.ID {
		t.Fatalf("fixa a versão do subfluxo: %+v", mv.Subflows)
	}
	// Nova versão do subfluxo não altera a versão já publicada do fluxo.
	var sub2 wfResp
	r.c.expect(saveWorkflow(r.c, sub, "sub", linear(agentNode("sub v2"))), 200).JSON(&sub2)
	r.c.expect(r.publish(sub2), 201)
	var again releases.Version
	r.c.expect(r.c.get("/api/workflow-versions/"+mv.ID), 200).JSON(&again)
	if again.ContentHash != mv.ContentHash || again.Subflows[sub.ID] != sv1.ID || len(again.FlatGraph.Nodes) != len(mv.FlatGraph.Nodes) {
		t.Fatal("a versão do fluxo é fixa")
	}
	// Só subfluxos podem ser referenciados.
	feat := r.draft(t, "outra-feature", "feature", linear(agentNode("x")))
	r.c.expect(r.publish(feat), 201)
	bad := r.draft(t, "usa-feature", "feature", linear(wfNode("sf", "subflow", 0, 0, map[string]any{"workflow_id": feat.ID})))
	if rs := r.publish(bad); rs.StatusCode != 422 {
		t.Fatalf("%d", rs.StatusCode)
	}
	// Versões são imutáveis no banco.
	if _, err := r.e.db.Pool.Exec(ctx, `UPDATE workflow_versions SET name = 'x' WHERE id = $1`, mv.ID); err == nil {
		t.Fatal("UPDATE deveria ser recusado")
	}
	if _, err := r.e.db.Pool.Exec(ctx, `DELETE FROM workflow_versions WHERE id = $1`, mv.ID); err == nil {
		t.Fatal("DELETE deveria ser recusado")
	}
}

// T07-03 — "Revalidação em runtime respeita revogação."
func TestRuntimeRevalidationHonorsRevocation(t *testing.T) {
	r := newRelEnv(t)
	ctx := context.Background()
	acct := connWith(t, r.c, "cli", "claude-a", "claude")
	r.agent(t, "dev", &acct.ID)
	w := r.draft(t, "fluxo", "feature", linear(agentNode("x")))
	var v releases.Version
	r.c.expect(r.publish(w), 201).JSON(&v)
	feat := r.feature(t, "F-9")
	var rep struct {
		OK       bool
		Problems []releases.Problem
	}
	r.c.expect(r.c.get("/api/workflow-versions/"+v.ID+"/revalidate"), 200).JSON(&rep)
	if !rep.OK {
		t.Fatalf("%+v", rep)
	}
	r.c.expect(r.c.post("/api/connections/"+acct.ID+"/revoke", nil), 200)
	r.c.expect(r.c.get("/api/workflow-versions/"+v.ID+"/revalidate"), 200).JSON(&rep)
	if rep.OK || len(rep.Problems) != 1 || rep.Problems[0].Code != "connection_revoked" || rep.Problems[0].Node != "dev" {
		t.Fatalf("%+v", rep)
	}
	_, _, err := releases.StartExecution(ctx, r.e.db.Pool, r.proj, feat, w.ID)
	var re *releases.RevalidationError
	if !errors.As(err, &re) {
		t.Fatalf("a conta revogada impede a execução: %v", err)
	}
	var n int
	_ = r.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM executions WHERE feature_id = $1`, feat).Scan(&n)
	if n != 0 {
		t.Fatal("nenhuma execução pode nascer")
	}
}
