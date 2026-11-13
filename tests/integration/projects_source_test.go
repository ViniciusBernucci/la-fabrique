package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// fakeJira simula o Jira. Em modo de recusa o corpo ecoa a credencial, para
// provar que ela nunca chega ao painel. Qualquer escrita é contada.
type fakeJira struct {
	*httptest.Server
	mode   atomic.Value // "" | "forbidden"
	writes atomic.Int32
}

func newFakeJira(t *testing.T) *fakeJira {
	f := &fakeJira{}
	f.mode.Store("")
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			f.writes.Add(1)
			w.WriteHeader(405)
			return
		}
		if f.mode.Load().(string) == "forbidden" {
			w.WriteHeader(403)
			fmt.Fprintf(w, `{"errorMessages":["negado para %s"]}`, r.Header.Get("Authorization"))
			return
		}
		p := strings.TrimPrefix(r.URL.Path, "/rest/api/3")
		w.Header().Set("Content-Type", "application/json")
		switch p {
		case "/myself":
			io.WriteString(w, `{"accountId":"a1","displayName":"Fábrica Bot"}`)
		case "/project/ABC/statuses":
			io.WriteString(w, `[{"name":"Story","statuses":[{"name":"To Do","statusCategory":{"key":"new"}},{"name":"In Progress","statusCategory":{"key":"indeterminate"}},{"name":"Done","statusCategory":{"key":"done"}}]}]`)
		case "/priority":
			io.WriteString(w, `[{"name":"Highest"},{"name":"High"},{"name":"Medium"},{"name":"Low"}]`)
		case "/search/jql":
			issue := func(id, key, status, cat, prio string) string {
				return fmt.Sprintf(`{"id":"%s","key":"%s","fields":{"summary":"Item %s","issuetype":{"name":"Story","subtask":false},"status":{"name":"%s","statusCategory":{"key":"%s"}},"priority":{"name":"%s"},"labels":["lf-ready"],"updated":"2026-11-10T12:00:00.000+0000"}}`, id, key, key, status, cat, prio)
			}
			fmt.Fprintf(w, `{"issues":[%s,%s,%s,%s],"isLast":true}`,
				issue("1", "ABC-1", "To Do", "new", "High"),
				issue("2", "ABC-2", "In Progress", "indeterminate", "Medium"),
				issue("3", "ABC-3", "Done", "done", "High"),
				issue("4", "ABC-4", "To Do", "new", "Highest"))
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

type sourceFixture struct {
	c    *client
	fj   *fakeJira
	p    projects.Project
	gh   security.Connection
	jira security.Connection
}

func newSourceFixture(t *testing.T, slug string) sourceFixture {
	t.Helper()
	_, c, fj := newProjectEnv(t)
	p := createProject(t, c, slug)
	gh := conn(t, c, "github", "gh-"+slug)
	jira := conn(t, c, "jira", "jira-"+slug)
	return sourceFixture{c: c, fj: fj, p: configure(t, c, p, gh, jira), gh: gh, jira: jira}
}

func (f sourceFixture) check(t *testing.T) projects.SourceCheck {
	t.Helper()
	var res projects.SourceCheck
	f.c.expect(f.c.post("/api/projects/"+f.p.ID+"/source/check", nil), 200).JSON(&res)
	return res
}

func step(res projects.SourceCheck, id string) projects.SourceStep {
	for _, s := range res.Steps {
		if s.ID == id {
			return s
		}
	}
	return projects.SourceStep{}
}

// T05-02 — "Origem válida devolve preview correto de itens elegíveis" e a ativação só vale com a verificação.
func TestSourceCheckReturnsEligiblePreviewAndGatesActivation(t *testing.T) {
	f := newSourceFixture(t, "origem-ok")

	// Antes da verificação: tudo preenchido, mas a ativação é recusada pelo motivo certo.
	pend := pending(validation(t, f.c, f.p.ID))
	if len(pend) != 1 || !strings.Contains(pend["source.remote"].Action, "ainda não foi verificada") {
		t.Fatalf("só a verificação remota deveria faltar: %+v", pend)
	}
	if resp := f.c.post("/api/projects/"+f.p.ID+"/activate", map[string]any{"expected_version": f.p.Version}); resp.StatusCode != 422 {
		t.Fatalf("sem verificação a ativação deve ser bloqueada: %d", resp.StatusCode)
	}

	res := f.check(t)
	if !res.OK {
		t.Fatalf("verificação deveria passar: %+v", res.Steps)
	}
	for _, id := range []string{"credential", "container", "statuses", "priorities", "items"} {
		if step(res, id).Status != projects.StepOK {
			t.Errorf("passo %s: %+v", id, step(res, id))
		}
	}
	// Elegíveis: só "To Do", sem concluídos. ABC-1 (High→1) e ABC-4 (Highest, fora do mapa).
	if len(res.Items) != 2 || res.Items[0].Key != "ABC-1" || res.Items[1].Key != "ABC-4" {
		t.Fatalf("preview: %+v", res.Items)
	}
	if res.Items[0].Mapped == nil || *res.Items[0].Mapped != 1 || res.Items[1].Mapped != nil {
		t.Fatalf("mapa de prioridade: %+v", res.Items)
	}
	if !strings.Contains(step(res, "items").Detail, "1 sem prioridade no mapa") {
		t.Fatalf("o preview deve avisar prioridade sem mapa: %q", step(res, "items").Detail)
	}
	if f.fj.writes.Load() != 0 {
		t.Fatal("a verificação não pode escrever na origem")
	}

	if r := validation(t, f.c, f.p.ID); !r.OK {
		t.Fatalf("depois da verificação não deveria haver pendência: %+v", pending(r))
	}
	var active projects.Project
	f.c.expect(f.c.post("/api/projects/"+f.p.ID+"/activate", map[string]any{"expected_version": f.p.Version}), 200).JSON(&active)
	if active.State != projects.StateActive {
		t.Fatalf("ativação: %+v", active)
	}
	var trail changesResp
	f.c.expect(f.c.get("/api/projects/"+f.p.ID+"/changes"), 200).JSON(&trail)
	found := false
	for _, it := range trail.Items {
		found = found || it.Action == projects.ChangeSourceChecked
	}
	if !found {
		t.Fatal("a verificação deve constar na trilha")
	}
}

func TestSourceCheckFailsWithReasonForInvalidMapping(t *testing.T) {
	f := newSourceFixture(t, "mapa-ruim")
	patch := func(src map[string]any) projects.Project {
		var p projects.Project
		f.c.expect(f.c.do("PATCH", "/api/projects/"+f.p.ID, map[string]any{"expected_version": f.p.Version, "source": src}, nil), 200).JSON(&p)
		f.p = p
		return p
	}
	// Valores fora da faixa e escopo inválido são recusados na gravação, com o campo.
	for name, mut := range map[string]func(map[string]any){
		"prioridade 9":  func(s map[string]any) { s["priority_map"] = map[string]int{"high": 9} },
		"escopo":        func(s map[string]any) { s["access"] = "admin" },
		"marcador":      func(s map[string]any) { s["eligibility_marker"] = `a"b` },
		"estado vazio":  func(s map[string]any) { s["eligible_statuses"] = []string{" "} },
		"marcador jira": func(s map[string]any) { s["eligibility_marker"] = "com espaço" },
	} {
		src := jiraSource(f.jira.ID)
		mut(src)
		if resp := f.c.do("PATCH", "/api/projects/"+f.p.ID, map[string]any{"expected_version": f.p.Version, "source": src}, nil); resp.StatusCode != 400 || resp.ErrCode() != "invalid_request" {
			t.Errorf("%s: esperado 400, veio %d %s", name, resp.StatusCode, resp.Body)
		}
	}

	// Estado e prioridade que não existem na origem passam na gravação mas falham na verificação.
	src := jiraSource(f.jira.ID)
	src["eligible_statuses"] = []string{"To Do", "Pronto para dev"}
	src["priority_map"] = map[string]int{"high": 1, "urgentissima": 0}
	patch(src)
	res := f.check(t)
	if res.OK || step(res, "statuses").Status != projects.StepFailed || !strings.Contains(step(res, "statuses").Detail, "Pronto para dev") ||
		step(res, "priorities").Status != projects.StepFailed || !strings.Contains(step(res, "priorities").Detail, "urgentissima") {
		t.Fatalf("mapa inválido deveria falhar com motivo: %+v", res.Steps)
	}
	if resp := f.c.post("/api/projects/"+f.p.ID+"/activate", map[string]any{"expected_version": f.p.Version}); resp.StatusCode != 422 {
		t.Fatalf("verificação falha deve bloquear a ativação: %d", resp.StatusCode)
	}
	if !strings.Contains(pending(validation(t, f.c, f.p.ID))["source.remote"].Action, "falhou") {
		t.Fatal("a pendência deve dizer que a verificação falhou")
	}

	// Sem critério de elegibilidade não há como ativar.
	src = jiraSource(f.jira.ID)
	src["eligibility_marker"], src["eligible_statuses"] = "", []string{}
	patch(src)
	if _, ok := pending(validation(t, f.c, f.p.ID))["source.eligibility"]; !ok {
		t.Fatal("sem marcador nem estado elegível deve haver pendência")
	}
}

func TestSourceCheckReportsPermissionWithoutLeakingSecret(t *testing.T) {
	f := newSourceFixture(t, "sem-permissao")
	f.fj.mode.Store("forbidden")
	resp := f.c.expect(f.c.post("/api/projects/"+f.p.ID+"/source/check", nil), 200)
	var res projects.SourceCheck
	resp.JSON(&res)
	if res.OK || step(res, "credential").Status != projects.StepFailed {
		t.Fatalf("recusa deveria falhar a verificação: %+v", res.Steps)
	}
	body := string(resp.Body)
	if !strings.Contains(step(res, "credential").Detail, "permissão") {
		t.Fatalf("o motivo deve ser claro: %q", step(res, "credential").Detail)
	}
	for _, leak := range []string{"segredo-jira-", "negado para", "Basic ", "bot@example.com"} {
		if strings.Contains(body, leak) || strings.Contains(f.c.e.logs.String(), leak) {
			t.Fatalf("vazamento de %q", leak)
		}
	}
	if r := validation(t, f.c, f.p.ID); r.OK {
		t.Fatal("verificação falha não pode liberar a ativação")
	}
}

func TestSourceCheckBecomesStaleWhenConfigOrSecretChanges(t *testing.T) {
	f := newSourceFixture(t, "obsoleta")
	f.check(t)
	if r := validation(t, f.c, f.p.ID); !r.OK {
		t.Fatalf("deveria estar válida: %+v", pending(r))
	}

	// Mudar a lista invalida.
	var p projects.Project
	src := jiraSource(f.jira.ID)
	src["eligibility_marker"] = "outro-marcador"
	f.c.expect(f.c.do("PATCH", "/api/projects/"+f.p.ID, map[string]any{"expected_version": f.p.Version, "source": src}, nil), 200).JSON(&p)
	if !strings.Contains(pending(validation(t, f.c, f.p.ID))["source.remote"].Action, "mudou") {
		t.Fatal("mudar o marcador deve invalidar a verificação")
	}

	// Reverificar valida; rotacionar o segredo da conexão invalida de novo.
	f.p = p
	f.check(t)
	if r := validation(t, f.c, f.p.ID); !r.OK {
		t.Fatalf("reverificação deveria validar: %+v", pending(r))
	}
	var cur security.Connection
	f.c.expect(f.c.get("/api/connections/"+f.jira.ID), 200).JSON(&cur)
	f.c.expect(f.c.post("/api/connections/"+f.jira.ID+"/rotate", map[string]any{"secret": "bot@example.com:novo-token-123", "expected_version": cur.Version}), 200)
	if !strings.Contains(pending(validation(t, f.c, f.p.ID))["source.remote"].Action, "mudou") {
		t.Fatal("rotacionar o segredo deve invalidar a verificação")
	}
}

// T05-02 — "duas origens ativas ... falham com motivo".
func TestSameSourceCannotFeedTwoActiveProjects(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	gh := conn(t, c, "github", "gh-dupla")
	jira := conn(t, c, "jira", "jira-dupla")
	a := completeConfig(t, c, createProject(t, c, "projeto-a1"), gh, jira)
	c.expect(c.post("/api/projects/"+a.ID+"/activate", map[string]any{"expected_version": a.Version}), 200)

	b := completeConfig(t, c, createProject(t, c, "projeto-b1"), gh, jira)
	pend := pending(validation(t, c, b.ID))
	if !strings.Contains(pend["source.exclusive"].Action, "Projeto projeto-a1") {
		t.Fatalf("deveria apontar o projeto que já usa a origem: %+v", pend)
	}
	if resp := c.post("/api/projects/"+b.ID+"/activate", map[string]any{"expected_version": b.Version}); resp.StatusCode != 422 {
		t.Fatalf("duas origens ativas: %d", resp.StatusCode)
	}

	// Outra lista na mesma conexão é permitida.
	src := jiraSource(jira.ID)
	src["container"] = "XYZ"
	var moved projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+b.ID, map[string]any{"expected_version": b.Version, "source": src}, nil), 200).JSON(&moved)
	if _, ok := pending(validation(t, c, b.ID))["source.exclusive"]; ok {
		t.Fatal("outra lista não conflita")
	}
}

func TestSourceCheckPreconditionsAndAccess(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	p := createProject(t, c, "pre-condicoes")
	// Origem incompleta.
	if resp := c.post("/api/projects/"+p.ID+"/source/check", nil); resp.StatusCode != 422 || resp.ErrCode() != "source_incomplete" {
		t.Fatalf("origem incompleta: %d %s", resp.StatusCode, resp.Body)
	}
	// Conexão de tipo errado para a plataforma.
	gh := conn(t, c, "github", "gh-errada")
	var cur projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": 1, "source": jiraSource(gh.ID)}, nil), 200).JSON(&cur)
	if resp := c.post("/api/projects/"+p.ID+"/source/check", nil); resp.StatusCode != 422 {
		t.Fatalf("conexão incompatível: %d", resp.StatusCode)
	}
	// Conexão revogada.
	jira := conn(t, c, "jira", "jira-revogada")
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": cur.Version, "source": jiraSource(jira.ID)}, nil), 200)
	c.expect(c.post("/api/connections/"+jira.ID+"/revoke", nil), 200)
	if resp := c.post("/api/projects/"+p.ID+"/source/check", nil); resp.StatusCode != 409 || resp.ErrCode() != "connection_revoked" {
		t.Fatalf("conexão revogada: %d %s", resp.StatusCode, resp.Body)
	}
	// Sessão e CSRF.
	anon := e.newClient()
	if r := anon.post("/api/projects/"+p.ID+"/source/check", nil); r.StatusCode != 401 {
		t.Fatalf("sem sessão: %d", r.StatusCode)
	}
	if r := c.do("POST", "/api/projects/"+p.ID+"/source/check", nil, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Fatalf("sem CSRF: %d", r.StatusCode)
	}
}
