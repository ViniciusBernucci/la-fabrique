package integration

import (
	"reflect"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func connWith(t *testing.T, c *client, kind, name, target string) security.Connection {
	t.Helper()
	var out security.Connection
	c.expect(c.post("/api/connections", map[string]string{"kind": kind, "name": name, "target": target, "secret": "marcador-" + name}), 201).JSON(&out)
	return out
}

func agentBody(name string, conn *string) map[string]any {
	b := map[string]any{
		"name": name, "description": "Agente de desenvolvimento", "instructions": "Seja objetivo e escreva testes.",
		"provider": "claude", "model": "claude-sonnet-5-5", "skills": []string{"revisao"}, "tools": []string{"Read", "Edit", "Bash"}, "timeout_minutes": 30,
	}
	if conn != nil {
		b["connection_id"] = *conn
	}
	return b
}

func workflowWithAgent(t *testing.T, c *client, name string, props map[string]any) (wfResp, string) {
	t.Helper()
	w := createWorkflow(t, c, name, "feature")
	g := w.Graph
	props["agent"] = "dev"
	props["prompt"] = "Implemente a feature"
	g.Nodes = append(g.Nodes, workflows.Node{ID: "ag", Type: "agent", X: 250, Y: 140, Props: props})
	var saved wfResp
	c.expect(saveWorkflow(c, w, name, g), 200).JSON(&saved)
	return saved, "ag"
}

func TestAgentCatalogCRUDAndConnectionRules(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	acctA := connWith(t, c, "cli", "claude-a", "claude")
	codex := connWith(t, c, "cli", "codex-a", "codex")
	apiConn := connWith(t, c, "api", "gateway", "")

	var a agents.Agent
	c.expect(c.post("/api/agents", agentBody("dev", &acctA.ID)), 201).JSON(&a)
	if a.Version != 1 || a.Provider != "claude" || *a.ConnectionID != acctA.ID || !reflect.DeepEqual(a.Tools, []string{"Read", "Edit", "Bash"}) {
		t.Fatalf("%+v", a)
	}
	var list struct{ Items []agents.Agent }
	c.expect(c.get("/api/agents"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].ID != a.ID {
		t.Fatalf("lista: %+v", list.Items)
	}
	// A API nunca devolve segredo: só a referência da conexão.
	if body := string(c.get("/api/agents/" + a.ID).Body); containsAny(body, "marcador-") {
		t.Fatalf("segredo na resposta: %s", body)
	}

	// Atualização com controle otimista.
	upd := agentBody("dev", &acctA.ID)
	upd["model"], upd["expected_version"] = "claude-opus-5-5", 1
	var a2 agents.Agent
	c.expect(c.do("PUT", "/api/agents/"+a.ID, upd, nil), 200).JSON(&a2)
	if a2.Version != 2 || a2.Model != "claude-opus-5-5" {
		t.Fatalf("%+v", a2)
	}
	if r := c.do("PUT", "/api/agents/"+a.ID, upd, nil); r.StatusCode != 409 || r.ErrCode() != "version_conflict" {
		t.Fatalf("versão obsoleta: %d %s", r.StatusCode, r.Body)
	}
	delete(upd, "expected_version")
	if r := c.do("PUT", "/api/agents/"+a.ID, upd, nil); r.StatusCode != 400 {
		t.Fatalf("sem expected_version: %d", r.StatusCode)
	}

	// Conexão incompatível, de outro provedor, de outro tipo, inexistente ou revogada.
	bad := func(provider string, conn string, name string) apiResp {
		b := agentBody(name, &conn)
		b["provider"] = provider
		return c.post("/api/agents", b)
	}
	for label, r := range map[string]apiResp{
		"codex com conta claude": bad("codex", acctA.ID, "x1"),
		"claude com conta codex": bad("claude", codex.ID, "x2"),
		"claude com conexão api": bad("claude", apiConn.ID, "x3"),
		"conexão inexistente":    bad("claude", "00000000-0000-4000-8000-000000000000", "x4"),
	} {
		if r.StatusCode != 409 || r.ErrCode() != "connection_incompatible" {
			t.Errorf("%s: %d %s", label, r.StatusCode, r.Body)
		}
	}
	c.expect(c.post("/api/connections/"+codex.ID+"/revoke", nil), 200)
	if r := bad("codex", codex.ID, "x5"); r.StatusCode != 409 {
		t.Errorf("conexão revogada: %d", r.StatusCode)
	}
	// Provedor api aceita conexão api; fallback também é conferido.
	b := agentBody("via-api", &apiConn.ID)
	b["provider"] = "api"
	c.expect(c.post("/api/agents", b), 201)
	b = agentBody("com-fallback", &acctA.ID)
	b["fallback"] = []map[string]string{{"connection_id": apiConn.ID, "model": "m"}}
	if r := c.post("/api/agents", b); r.StatusCode != 409 {
		t.Errorf("fallback incompatível: %d %s", r.StatusCode, r.Body)
	}

	// Validação de campos e nome duplicado.
	for label, body := range map[string]map[string]any{
		"nome":          {"name": "Dev Ruim", "instructions": "x", "provider": "claude"},
		"instruções":    {"name": "z1", "instructions": " ", "provider": "claude"},
		"provedor":      {"name": "z2", "instructions": "x", "provider": "gemini"},
		"campo extra":   {"name": "z3", "instructions": "x", "provider": "claude", "x": 1},
		"tool inválida": {"name": "z4", "instructions": "x", "provider": "claude", "tools": []string{"a;b"}},
	} {
		if r := c.post("/api/agents", body); r.StatusCode != 400 {
			t.Errorf("%s: %d %s", label, r.StatusCode, r.Body)
		}
	}
	if r := c.post("/api/agents", agentBody("dev", &acctA.ID)); r.StatusCode != 409 || r.ErrCode() != "duplicate" {
		t.Errorf("nome duplicado: %d %s", r.StatusCode, r.Body)
	}
}

func containsAny(s string, subs ...string) bool {
	for _, x := range subs {
		for i := 0; i+len(x) <= len(s); i++ {
			if s[i:i+len(x)] == x {
				return true
			}
		}
	}
	return false
}

// T08-01 — "Dois workflows escolhem configurações diferentes e catálogo original permanece íntegro. Execução mostra configuração efetiva."
func TestTwoWorkflowsResolveDifferentConfigsAndCatalogStaysIntact(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	acctA := connWith(t, c, "cli", "claude-a", "claude")
	acctB := connWith(t, c, "cli", "claude-b", "claude")
	var base agents.Agent
	c.expect(c.post("/api/agents", agentBody("dev", &acctA.ID)), 201).JSON(&base)

	w1, n1 := workflowWithAgent(t, c, "Fluxo rápido", map[string]any{"model": "claude-haiku-5-5", "timeout_minutes": 10})
	w2, n2 := workflowWithAgent(t, c, "Fluxo cuidadoso", map[string]any{"connection": acctB.ID, "tools": "Read", "instructions_extra": "Revise duas vezes."})

	var e1, e2 agents.Effective
	c.expect(c.get("/api/workflows/"+w1.ID+"/nodes/"+n1+"/effective"), 200).JSON(&e1)
	c.expect(c.get("/api/workflows/"+w2.ID+"/nodes/"+n2+"/effective"), 200).JSON(&e2)

	if e1.Model != "claude-haiku-5-5" || e1.TimeoutMinutes != 10 || *e1.ConnectionID != acctA.ID || e1.Sources["model"] != "node" || e1.Sources["connection"] != "agent" || e1.Prompt != "Implemente a feature" {
		t.Fatalf("workflow 1: %+v", e1)
	}
	if e2.Model != "claude-sonnet-5-5" || *e2.ConnectionID != acctB.ID || !reflect.DeepEqual(e2.Tools, []string{"Read"}) ||
		e2.Instructions != "Seja objetivo e escreva testes.\n\nRevise duas vezes." || e2.Sources["connection"] != "node" || e2.Sources["model"] != "agent" {
		t.Fatalf("workflow 2: %+v", e2)
	}
	if e1.AgentVersion != 1 || e2.AgentVersion != 1 {
		t.Fatal("a versão da base aparece na configuração efetiva")
	}

	// O catálogo original permanece íntegro: mesma versão e mesmos campos.
	var after agents.Agent
	c.expect(c.get("/api/agents/"+base.ID), 200).JSON(&after)
	if !reflect.DeepEqual(base, after) {
		t.Fatalf("o agente do catálogo mudou:\n%+v\n%+v", base, after)
	}

	// Sobrescritas inválidas e referências quebradas.
	w3, n3 := workflowWithAgent(t, c, "Amplia ferramentas", map[string]any{"tools": "Read, Write"})
	if r := c.get("/api/workflows/" + w3.ID + "/nodes/" + n3 + "/effective"); r.StatusCode != 422 || r.ErrCode() != "invalid_override" {
		t.Fatalf("ampliar ferramentas: %d %s", r.StatusCode, r.Body)
	}
	w4, n4 := workflowWithAgent(t, c, "Conta revogada", map[string]any{"connection": acctB.ID})
	c.expect(c.post("/api/connections/"+acctB.ID+"/revoke", nil), 200)
	if r := c.get("/api/workflows/" + w4.ID + "/nodes/" + n4 + "/effective"); r.StatusCode != 409 || r.ErrCode() != "connection_incompatible" {
		t.Fatalf("conta do nó revogada: %d %s", r.StatusCode, r.Body)
	}
	w5 := createWorkflow(t, c, "Agente fantasma", "feature")
	g := w5.Graph
	g.Nodes = append(g.Nodes, workflows.Node{ID: "ag", Type: "agent", Props: map[string]any{"agent": "nao-existe", "prompt": "x"}})
	c.expect(saveWorkflow(c, w5, "Agente fantasma", g), 200)
	if r := c.get("/api/workflows/" + w5.ID + "/nodes/ag/effective"); r.StatusCode != 422 || r.ErrCode() != "agent_not_found" {
		t.Fatalf("agente ausente: %d %s", r.StatusCode, r.Body)
	}
	if r := c.get("/api/workflows/" + w1.ID + "/nodes/fim/effective"); r.StatusCode != 422 {
		t.Fatalf("nó que não é agente: %d", r.StatusCode)
	}
	if r := c.get("/api/workflows/" + w1.ID + "/nodes/nao-existe/effective"); r.StatusCode != 404 {
		t.Fatalf("nó inexistente: %d", r.StatusCode)
	}
	// O validador do workflow acusa sobrescrita malformada com nó e campo.
	bad := w1.Graph
	bad.Nodes[2].Props = map[string]any{"agent": "dev", "prompt": "x", "connection": "nao-e-uuid", "tools": "a;b"}
	var saved wfResp
	w1v2 := w1
	c.expect(saveWorkflow(c, w1v2, "Fluxo rápido", bad), 200).JSON(&saved)
	found := map[string]bool{}
	for _, is := range saved.Issues {
		if is.Node == "ag" && (is.Field == "connection" || is.Field == "tools") {
			found[is.Field] = true
		}
	}
	if !found["connection"] || !found["tools"] {
		t.Fatalf("problemas do nó: %+v", saved.Issues)
	}
}

func TestAgentRoutesRequireSessionAndCSRF(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	var a agents.Agent
	c.expect(c.post("/api/agents", agentBody("seguro", nil)), 201).JSON(&a)
	anon := e.newClient()
	for _, r := range []apiResp{anon.get("/api/agents"), anon.get("/api/agents/" + a.ID), anon.post("/api/agents", agentBody("x", nil))} {
		if r.StatusCode != 401 {
			t.Errorf("sem sessão: %d", r.StatusCode)
		}
	}
	if r := c.do("POST", "/api/agents", agentBody("forjado", nil), map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Errorf("sem CSRF: %d", r.StatusCode)
	}
}
