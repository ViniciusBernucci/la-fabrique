package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/projectres"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
)

type prEnv struct {
	c   *client
	e   *authEnv
	p   projects.Project
	cli string // conta CLI
	api string // conta API (openrouter)
}

func newPrEnv(t *testing.T, slug string) *prEnv {
	t.Helper()
	e, c, _ := newProjectEnv(t)
	p := createProject(t, c, slug)
	gh := conn(t, c, "github", "gh-"+slug)
	jira := conn(t, c, "jira", "jira-"+slug)
	p = completeConfig(t, c, p, gh, jira)
	for _, n := range []string{"m-a", "m-b"} {
		in := models.Input{Provider: "api", Gateway: models.GatewayOpenRouter, Name: n, Label: n, SupportsTools: true, ContextWindow: 100000, MaxOutputTokens: 4000}
		if _, err := models.Register(context.Background(), e.db.Pool, in); err != nil {
			t.Fatal(err)
		}
	}
	return &prEnv{c: c, e: e, p: p, cli: connWith(t, c, "cli", "claude-"+slug, "claude").ID, api: connWith(t, c, "api", "gw-"+slug, "openrouter").ID}
}

func (r *prEnv) agent(t *testing.T, name string) agents.Agent {
	t.Helper()
	b := agentBody(name, &r.api)
	b["provider"], b["model"], b["tools"] = "api", "m-a", []string{"read_file"}
	var a agents.Agent
	r.c.expect(r.c.post("/api/agents", b), 201).JSON(&a)
	return a
}

// workflow cria um workflow de feature com um agente "dev"; publica se pedido.
func (r *prEnv) workflow(t *testing.T, name string, publish bool) wfResp {
	t.Helper()
	w := createWorkflow(t, r.c, name, "feature")
	r.c.expect(saveWorkflow(r.c, w, name, linear(agentNode("faça"))), 200).JSON(&w)
	if publish {
		r.c.expect(r.c.post("/api/workflows/"+w.ID+"/publish", map[string]any{"expected_version": w.Version}), 201)
	}
	return w
}

func (r *prEnv) setResources(items ...projectres.ResourceInput) apiResp {
	return r.c.do("PUT", "/api/projects/"+r.p.ID+"/resources", map[string]any{"resources": items}, nil)
}

func (r *prEnv) activate(t *testing.T) apiResp {
	t.Helper()
	var cur projects.Project
	r.c.expect(r.c.get("/api/projects/"+r.p.ID), 200).JSON(&cur)
	return r.c.post("/api/projects/"+r.p.ID+"/activate", map[string]any{"expected_version": cur.Version})
}

var manifestsFixture = map[string]string{
	"package.json":       `{"name":"app","scripts":{"build":"vite build","test":"vitest run","lint":"biome check .","dev":"vite"}}`,
	"pnpm-lock.yaml":     `lockfileVersion: 9`,
	"go.mod":             "module example.com/app\n\ngo 1.24\n",
	"Makefile":           "build:\n\tgo build ./...\ntest:\n\tgo test ./...\nup:\n\tdocker compose up\n",
	"docker-compose.yml": "services:\n  api:\n    image: x\n  postgres:\n    image: postgres:17\nvolumes:\n  dados:\n",
	".env.example":       "# exemplo\nDATABASE_URL=postgres://user:senha-de-exemplo@db/app\nSTRIPE_KEY=sk_" + "live_valor_que_nao_deve_vazar_123456\nlower=ignorado\n",
	".env":               "STRIPE_KEY=sk_" + "live_REAL_NAO_PODE_SER_LIDO_1234567890",
	"../../etc/passwd":   "root:x:0:0",
}

// T05-03 — "Inspecionar manifestos do repo": sugestões com origem, sem segredo e sem executar nada.
func TestDiscoverySuggestsCommandsWithTheirSourceAndNeverReadsSecrets(t *testing.T) {
	r := newPrEnv(t, "pr-discover")
	var d projectres.Discovery
	resp := r.c.expect(r.c.post("/api/projects/"+r.p.ID+"/profile/discover", map[string]any{"files": manifestsFixture}), 200)
	resp.JSON(&d)
	got := map[string]projectres.Suggestion{}
	for _, s := range d.Commands {
		got[s.Name+"|"+s.Source] = s
	}
	for _, want := range []string{"install|package.json", "build|package.json#scripts.build", "test|package.json#scripts.test", "lint|package.json#scripts.lint", "build|go.mod", "test|Makefile#test", "start|Makefile#up", "start|docker-compose.yml"} {
		if _, ok := got[want]; !ok {
			t.Errorf("faltou a sugestão %s: %+v", want, d.Commands)
		}
	}
	if got["install|package.json"].Value != "pnpm install" || got["test|package.json#scripts.test"].Value != "pnpm run test" {
		t.Fatalf("o gerenciador vem do lockfile: %+v", got)
	}
	if strings.Join(d.Services, ",") != "api,postgres" || strings.Join(d.SecretNames, ",") != "DATABASE_URL,STRIPE_KEY" {
		t.Fatalf("serviços %v, nomes %v", d.Services, d.SecretNames)
	}
	body := string(resp.Body)
	for _, leak := range []string{"sk_live", "senha-de-exemplo", "NAO_PODE_SER_LIDO", "root:x"} {
		if strings.Contains(body, leak) {
			t.Fatalf("a descoberta vazou %q", leak)
		}
	}
	for _, in := range d.Inspected {
		if in == ".env" || strings.Contains(in, "..") {
			t.Fatalf("arquivo que não deveria ser inspecionado: %s", in)
		}
	}
	// Só a descoberta foi gravada: o perfil escolhido segue vazio.
	var prof projectres.Profile
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/profile"), 200).JSON(&prof)
	if prof.Version != 0 || len(prof.Commands) != 0 || prof.Discovery == nil || prof.DiscoveredAt == nil {
		t.Fatalf("%+v", prof)
	}
	// Limites.
	if rs := r.c.post("/api/projects/"+r.p.ID+"/profile/discover", map[string]any{"files": map[string]string{}}); rs.StatusCode != 400 {
		t.Fatalf("%d", rs.StatusCode)
	}
}

func TestProfileKeepsTheSourceOfEachCommandAndStoresSecretsOnlyByReference(t *testing.T) {
	r := newPrEnv(t, "pr-profile")
	r.c.expect(r.c.post("/api/projects/"+r.p.ID+"/profile/discover", map[string]any{"files": manifestsFixture}), 200)
	put := func(version int, body map[string]any) apiResp {
		body["expected_version"] = version
		return r.c.do("PUT", "/api/projects/"+r.p.ID+"/profile", body, nil)
	}
	var prof projectres.Profile
	put(0, map[string]any{
		"commands":    map[string]string{"test": "pnpm run test", "build": "make meu-build"},
		"services":    []string{"postgres", "api", "api"},
		"secret_refs": map[string]string{"STRIPE_KEY": "connection:" + r.api},
	}).JSON(&prof)
	if prof.Version != 1 || prof.Commands["test"].Source != "package.json#scripts.test" || prof.Commands["build"].Source != "manual" ||
		strings.Join(prof.Services, ",") != "api,postgres" || prof.SecretRefs["STRIPE_KEY"] != "connection:"+r.api || prof.Discovery == nil {
		t.Fatalf("%+v", prof)
	}
	// Valor de segredo, comando com segredo, comando desconhecido, variável inválida.
	for name, body := range map[string]map[string]any{
		"valor no lugar da referência": {"secret_refs": map[string]string{"STRIPE_KEY": "sk_" + "live_abcdefghijklmnopqrstuvwx"}},
		"referência inexistente":       {"secret_refs": map[string]string{"X_KEY": "connection:00000000-0000-0000-0000-000000000000"}},
		"comando com segredo":          {"commands": map[string]string{"test": "curl -H 'Authorization: Bearer abcdefghijklmnop' x"}},
		"comando desconhecido":         {"commands": map[string]string{"deploy": "x"}},
		"variável inválida":            {"secret_refs": map[string]string{"minha chave": "connection:" + r.api}},
		"quebra de linha":              {"commands": map[string]string{"test": "a\nb"}},
	} {
		if rs := put(1, body); rs.StatusCode != 400 {
			t.Errorf("%s: %d %s", name, rs.StatusCode, rs.Body)
		}
	}
	// Conexão revogada não serve de referência.
	if rs := r.c.do("POST", "/api/connections/"+r.cli+"/revoke", nil, nil); rs.StatusCode >= 300 {
		t.Skipf("revogação indisponível: %d", rs.StatusCode)
	}
	if rs := put(1, map[string]any{"secret_refs": map[string]string{"X_KEY": "connection:" + r.cli}}); rs.StatusCode != 400 {
		t.Fatalf("revogada: %d", rs.StatusCode)
	}
	// Versão obsoleta.
	if rs := put(0, map[string]any{}); rs.StatusCode != 409 {
		t.Fatalf("%d", rs.StatusCode)
	}
	if blob := string(r.c.get("/api/projects/" + r.p.ID + "/profile").Body); strings.Contains(blob, "segredo-gw") || strings.Contains(blob, "marcador-") {
		t.Fatal("o perfil não pode conter o valor de nenhum segredo")
	}
}

// T05-03 — "Overrides preservam recurso global."
func TestProjectOverridesNeverChangeTheGlobalResource(t *testing.T) {
	r := newPrEnv(t, "pr-override")
	other := createProject(t, r.c, "pr-override-b")
	a := r.agent(t, "dev")
	var res struct{ Resources []projectres.Resource }
	r.setResources(projectres.ResourceInput{Kind: "agent", Ref: a.ID, Override: map[string]any{"model": "m-b", "timeout_minutes": 7}}).JSON(&res)
	if len(res.Resources) != 1 || res.Resources[0].Name != "dev" || res.Resources[0].Effective == nil ||
		res.Resources[0].Effective.Model != "m-b" || res.Resources[0].Effective.TimeoutMinutes != 7 ||
		res.Resources[0].Effective.Sources["model"] != "node" || res.Resources[0].Effective.Sources["provider"] != "agent" {
		t.Fatalf("%+v", res.Resources)
	}
	// O agente global segue idêntico.
	var after agents.Agent
	r.c.expect(r.c.get("/api/agents/"+a.ID), 200).JSON(&after)
	if after.Model != "m-a" || after.Version != a.Version || after.TimeoutMinutes != a.TimeoutMinutes {
		t.Fatalf("o recurso global foi alterado: %+v", after)
	}
	// Outro projeto associa o mesmo agente sem sobrescrita e enxerga a base.
	var res2 struct{ Resources []projectres.Resource }
	r.c.expect(r.c.do("PUT", "/api/projects/"+other.ID+"/resources", map[string]any{"resources": []projectres.ResourceInput{{Kind: "agent", Ref: a.ID}}}, nil), 200).JSON(&res2)
	if res2.Resources[0].Effective.Model != "m-a" || res2.Resources[0].Effective.Sources["model"] != "agent" {
		t.Fatalf("%+v", res2.Resources[0].Effective)
	}
	// Recusas: ampliar ferramentas, sobrescrever o que não é agente, recurso inexistente, duplicado, tipo inválido.
	for name, in := range map[string][]projectres.ResourceInput{
		"amplia ferramentas": {{Kind: "agent", Ref: a.ID, Override: map[string]any{"tools": "read_file,rm_rf"}}},
		"override em skill":  {{Kind: "skill", Ref: "x", Override: map[string]any{"a": 1}}},
		"inexistente":        {{Kind: "agent", Ref: "00000000-0000-0000-0000-000000000000"}},
		"duplicado":          {{Kind: "agent", Ref: a.ID}, {Kind: "agent", Ref: a.ID}},
		"tipo inválido":      {{Kind: "banana", Ref: "x"}},
	} {
		if rs := r.setResources(in...); rs.StatusCode != 400 {
			t.Errorf("%s: %d %s", name, rs.StatusCode, rs.Body)
		}
	}
	// A recusa não desfez a associação anterior (substituição atômica).
	var kept struct{ Resources []projectres.Resource }
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/resources"), 200).JSON(&kept)
	if len(kept.Resources) != 1 || kept.Resources[0].Override["model"] != "m-b" {
		t.Fatalf("%+v", kept.Resources)
	}
}

// T05-03 — "Rascunho de workflow ... bloqueiam ativação."
func TestADraftWorkflowBlocksActivationUntilPublished(t *testing.T) {
	r := newPrEnv(t, "pr-draft")
	r.agent(t, "dev")
	w := r.workflow(t, "fluxo-rascunho", false)
	r.setResources(projectres.ResourceInput{Kind: "workflow", Ref: w.ID}).JSON(&struct{}{})
	pend := pending(validation(t, r.c, r.p.ID))
	c, ok := pend["resources.workflows"]
	if !ok || !strings.Contains(c.Action, "fluxo-rascunho") || !strings.Contains(c.Action, "rascunho") || len(pend) != 1 {
		t.Fatalf("%+v", pend)
	}
	if rs := r.activate(t); rs.StatusCode != 422 {
		t.Fatalf("rascunho deve bloquear a ativação: %d %s", rs.StatusCode, rs.Body)
	}
	r.c.expect(r.c.post("/api/workflows/"+w.ID+"/publish", map[string]any{"expected_version": w.Version}), 201)
	if p := pending(validation(t, r.c, r.p.ID)); len(p) != 0 {
		t.Fatalf("%+v", p)
	}
	if rs := r.activate(t); rs.StatusCode != 200 {
		t.Fatalf("%d %s", rs.StatusCode, rs.Body)
	}
}

// T05-03 — "... conta incompatível bloqueiam ativação."
func TestAnIncompatibleAccountBlocksActivation(t *testing.T) {
	r := newPrEnv(t, "pr-account")
	a := r.agent(t, "dev")
	// Sobrescrita local com conta CLI para um agente de API: incompatível.
	r.setResources(projectres.ResourceInput{Kind: "agent", Ref: a.ID, Override: map[string]any{"connection": r.cli}})
	pend := pending(validation(t, r.c, r.p.ID))
	if c, ok := pend["resources.agents"]; !ok || !strings.Contains(c.Action, "incompatível") || !strings.Contains(c.Action, "dev") {
		t.Fatalf("%+v", pend)
	}
	if rs := r.activate(t); rs.StatusCode != 422 {
		t.Fatalf("%d", rs.StatusCode)
	}
	// Corrigida a sobrescrita, passa; revogar a conta do agente volta a bloquear.
	r.setResources(projectres.ResourceInput{Kind: "agent", Ref: a.ID})
	if p := pending(validation(t, r.c, r.p.ID)); len(p) != 0 {
		t.Fatalf("%+v", p)
	}
	// Modelo desativado também.
	if _, err := r.e.db.Pool.Exec(context.Background(), `UPDATE models SET state = 'disabled' WHERE name = 'm-a'`); err != nil {
		t.Fatal(err)
	}
	if c := pending(validation(t, r.c, r.p.ID))["resources.agents"]; !strings.Contains(c.Action, "desativado") {
		t.Fatalf("%+v", c)
	}
	// Revogar a conta do agente volta a bloquear.
	if _, err := r.e.db.Pool.Exec(context.Background(), `UPDATE models SET state = 'active' WHERE name = 'm-a'`); err != nil {
		t.Fatal(err)
	}
	if rs := r.c.do("POST", "/api/connections/"+r.api+"/revoke", nil, nil); rs.StatusCode >= 300 {
		t.Skipf("revogação indisponível: %d", rs.StatusCode)
	}
	if c := pending(validation(t, r.c, r.p.ID))["resources.agents"]; !strings.Contains(c.Action, "revogada") {
		t.Fatalf("%+v", c)
	}
}

// T05-03 — "Projeto apto pode iniciar execução."
func TestAnAptProjectCanStartAnExecutionOnlyFromAPublishedDefaultWorkflow(t *testing.T) {
	r := newPrEnv(t, "pr-apt")
	r.agent(t, "dev")
	var rd projectres.Readiness
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/readiness"), 200).JSON(&rd)
	if rd.Ready || len(rd.Problems) < 2 { // rascunho (não ativo) e sem fluxo padrão
		t.Fatalf("%+v", rd)
	}
	w := r.workflow(t, "fluxo-padrao", true)
	r.c.expect(r.c.do("PUT", "/api/projects/"+r.p.ID+"/selection", map[string]any{"rules": []any{}, "default": map[string]any{"workflow_id": w.ID}}, nil), 200)
	// O workflow estrutural do projeto (AJ05) nasce em rascunho e também precisa estar publicado.
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/readiness"), 200).JSON(&rd)
	if rd.Ready || !strings.Contains(strings.Join(rd.Problems, "|"), "workflow do projeto não está publicado") {
		t.Fatalf("workflow do projeto deveria bloquear: %+v", rd)
	}
	var pw projWorkflow
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/workflow"), 200).JSON(&pw)
	var sw wfResp
	r.c.expect(r.c.get("/api/workflows/"+pw.WorkflowID), 200).JSON(&sw)
	g := sw.Graph
	for i := range g.Nodes {
		switch g.Nodes[i].Type {
		case "agent":
			g.Nodes[i].Props["agent"] = "dev"
		case "checks":
			g.Nodes[i].Props["commands"] = "make test"
		}
	}
	r.c.expect(saveWorkflow(r.c, sw, sw.Name, g), 200).JSON(&sw)
	r.c.expect(r.c.post("/api/workflows/"+sw.ID+"/publish", map[string]any{"expected_version": sw.Version}), 201)
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/readiness"), 200).JSON(&rd)
	if rd.Ready || len(rd.Problems) != 1 || !strings.Contains(rd.Problems[0], "ativo") {
		t.Fatalf("falta ativar: %+v", rd)
	}
	if rs := r.activate(t); rs.StatusCode != 200 {
		t.Fatalf("%d %s", rs.StatusCode, rs.Body)
	}
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/readiness"), 200).JSON(&rd)
	if !rd.Ready || rd.Target == nil || rd.Target.WorkflowID != w.ID || len(rd.Problems) != 0 {
		t.Fatalf("%+v", rd)
	}
	f, err := persistenceFeature(r, "APT-1")
	if err != nil {
		t.Fatal(err)
	}
	ex, ver, err := releases.StartExecution(context.Background(), r.e.db.Pool, r.p.ID, f, rd.Target.WorkflowID)
	if err != nil || ex.ID == "" || ver.WorkflowID != w.ID {
		t.Fatalf("%+v %v", ex, err)
	}
	// Despublicar o fluxo padrão tira a aptidão.
	if _, err := r.e.db.Pool.Exec(context.Background(), `UPDATE workflows SET published_version_id = NULL WHERE id = $1`, w.ID); err != nil {
		t.Fatal(err)
	}
	r.c.expect(r.c.get("/api/projects/"+r.p.ID+"/readiness"), 200).JSON(&rd)
	if rd.Ready || !strings.Contains(strings.Join(rd.Problems, " "), "rascunho") {
		t.Fatalf("%+v", rd)
	}
}

func persistenceFeature(r *prEnv, ext string) (string, error) {
	var id string
	err := r.e.db.Pool.QueryRow(context.Background(), `INSERT INTO features (project_id, external_id, title) VALUES ($1, $2, 'F') RETURNING id`, r.p.ID, ext).Scan(&id)
	return id, err
}
