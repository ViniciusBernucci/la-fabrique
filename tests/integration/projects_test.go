package integration

import (
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type changesResp struct {
	Items []projects.Change `json:"items"`
}

func newProjectEnv(t *testing.T) (*authEnv, *client, *fakeJira) {
	t.Helper()
	fj := newFakeJira(t)
	e := newAuthEnv(t, nil, httpapi.WithTrackerFactory(func(cfg integrations.Config) (integrations.Tracker, error) {
		cfg.BaseURL = fj.URL // todos os tipos falam com a plataforma falsa
		return integrations.NewTracker(cfg)
	}))
	c := e.newClient()
	c.login(nil)
	return e, c, fj
}

func createProject(t *testing.T, c *client, slug string) projects.Project {
	t.Helper()
	var p projects.Project
	c.expect(c.post("/api/projects", map[string]any{"slug": slug, "name": "Projeto " + slug}), 201).JSON(&p)
	return p
}

func validation(t *testing.T, c *client, id string) projects.Report {
	t.Helper()
	var r projects.Report
	c.expect(c.get("/api/projects/"+id+"/validation"), 200).JSON(&r)
	return r
}

func pending(r projects.Report) map[string]projects.Check {
	m := map[string]projects.Check{}
	for _, c := range r.Checks {
		if c.Status == projects.CheckPending {
			m[c.ID] = c
		}
	}
	return m
}

func conn(t *testing.T, c *client, kind, name string) security.Connection {
	t.Helper()
	var out security.Connection
	secret := "segredo-" + name
	if kind == "jira" {
		secret = "bot@example.com:" + secret // formato exigido pelo adaptador
	}
	c.expect(c.post("/api/connections", map[string]string{"kind": kind, "name": name, "secret": secret}), 201).JSON(&out)
	return out
}

func jiraSource(connID string) map[string]any {
	return map[string]any{
		"kind": "jira", "container": "ABC", "connection_id": connID,
		"eligibility_marker": "lf-ready", "eligible_statuses": []string{"To Do"},
		"priority_map": map[string]int{"high": 1, "medium": 2}, "access": "read",
	}
}

// configure preenche todas as etapas com conexões compatíveis, sem verificar a origem.
func configure(t *testing.T, c *client, p projects.Project, gh, jira security.Connection) projects.Project {
	t.Helper()
	var out projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": p.Version, "wizard_step": "review",
		"repository": map[string]any{"owner": "org-exemplo", "name": "app", "base_branch": "main", "connection_id": gh.ID},
		"source":     jiraSource(jira.ID),
	}, nil), 200).JSON(&out)
	return out
}

// completeConfig configura e verifica a origem (a verificação não muda a versão).
func completeConfig(t *testing.T, c *client, p projects.Project, gh, jira security.Connection) projects.Project {
	t.Helper()
	out := configure(t, c, p, gh, jira)
	var check projects.SourceCheck
	c.expect(c.post("/api/projects/"+p.ID+"/source/check", nil), 200).JSON(&check)
	if !check.OK {
		t.Fatalf("a origem deveria verificar: %+v", check.Steps)
	}
	return out
}

func TestPriorityZeroIsAcceptedOnCreate(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	var p projects.Project
	c.expect(c.post("/api/projects", map[string]any{"slug": "baixa", "name": "Baixa", "priority": 0}), 201).JSON(&p)
	if p.Priority != 0 {
		t.Fatalf("prioridade 0 foi trocada pelo padrão: %d", p.Priority)
	}
}

// T05-01 — "Projeto novo vazio de vínculos persiste e retorna ao assistente."
func TestNewProjectIsDraftWithoutLinksAndResumesWizard(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	p := createProject(t, c, "novo")
	if p.State != projects.StateDraft || p.WizardStep != projects.StepRepository || p.Version != 1 || p.Priority != 2 {
		t.Fatalf("estado inicial: %+v", p)
	}
	if p.Repository.Owner != "" || p.Repository.ConnectionID != nil || p.Source.Kind != "" || p.Source.ConnectionID != nil || p.ActivatedAt != nil {
		t.Fatalf("projeto novo deve nascer sem vínculos: %+v", p)
	}

	// Etapa 2 parcial: nem tudo preenchido, e já persiste.
	var saved projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": 1, "wizard_step": "repository",
		"repository": map[string]any{"owner": "org-exemplo", "name": "", "base_branch": "main"},
	}, nil), 200).JSON(&saved)

	// "Retorna ao assistente": nova leitura (outra sessão) traz etapa e dados.
	c2 := c.e.newClient()
	c2.login(c.secret)
	var got projects.Project
	c2.expect(c2.get("/api/projects/"+p.ID), 200).JSON(&got)
	if got.WizardStep != "repository" || got.Repository.Owner != "org-exemplo" || got.Repository.BaseBranch != "main" || got.Version != 2 {
		t.Fatalf("rascunho não persistiu por etapa: %+v", got)
	}
}

// T05-01 — "Sem preenchimento obrigatório válido, ativação é bloqueada."
func TestActivationBlockedUntilRequiredFieldsAreValid(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	p := createProject(t, c, "bloqueado")

	r := validation(t, c, p.ID)
	if r.OK {
		t.Fatal("projeto vazio não pode estar válido")
	}
	pend := pending(r)
	for _, id := range []string{"repository.owner", "repository.name", "repository.base_branch", "repository.connection",
		"source.kind", "source.connection", "source.container", "source.eligibility", "source.remote"} {
		chk, ok := pend[id]
		if !ok || strings.TrimSpace(chk.Action) == "" {
			t.Fatalf("pendência %s deveria existir com ação concreta: %+v", id, chk)
		}
	}
	if len(r.NotVerified) == 0 {
		t.Fatal("o relatório deve declarar o que ainda não verifica")
	}

	resp := c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": p.Version})
	if resp.StatusCode != 422 || resp.ErrCode() != "validation_failed" {
		t.Fatalf("ativação deveria ser bloqueada: %d %s", resp.StatusCode, resp.Body)
	}
	var after projects.Project
	c.expect(c.get("/api/projects/"+p.ID), 200).JSON(&after)
	if after.State != projects.StateDraft || after.ActivatedAt != nil || after.Version != p.Version {
		t.Fatalf("ativação bloqueada não pode alterar o projeto: %+v", after)
	}

	// Parcialmente preenchido continua bloqueado.
	gh := conn(t, c, "github", "gh-parcial")
	var part projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": 1,
		"repository":       map[string]any{"owner": "org", "name": "app", "base_branch": "main", "connection_id": gh.ID},
	}, nil), 200).JSON(&part)
	if resp := c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": part.Version}); resp.StatusCode != 422 {
		t.Fatalf("sem origem a ativação deve continuar bloqueada: %d", resp.StatusCode)
	}
}

func TestConnectionCompatibilityIsReportedWithReason(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	p := createProject(t, c, "compat")
	gh := conn(t, c, "github", "gh-ok")
	jira := conn(t, c, "jira", "jira-ok")
	clickup := conn(t, c, "clickup", "clickup-ok")

	// Repositório com conexão Jira e origem Jira com conexão ClickUp: incompatíveis.
	var cur projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": 1,
		"repository":       map[string]any{"owner": "org", "name": "app", "base_branch": "main", "connection_id": jira.ID},
		"source":           map[string]any{"kind": "jira", "container": "ABC", "connection_id": clickup.ID, "eligibility_marker": "lf-ready"},
	}, nil), 200).JSON(&cur)
	pend := pending(validation(t, c, p.ID))
	if !strings.Contains(pend["repository.connection"].Action, "GitHub") || !strings.Contains(pend["source.connection"].Action, "Jira") {
		t.Fatalf("incompatibilidade deveria dizer o motivo: %+v", pend)
	}
	if resp := c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": cur.Version}); resp.StatusCode != 422 {
		t.Fatalf("conexão incompatível deve bloquear: %d", resp.StatusCode)
	}

	// Corrige; revoga a do GitHub: volta a bloquear, com motivo.
	cur = completeConfig(t, c, cur, gh, jira)
	if r := validation(t, c, p.ID); !r.OK {
		t.Fatalf("configuração compatível deveria validar: %+v", pending(r))
	}
	c.expect(c.post("/api/connections/"+gh.ID+"/revoke", nil), 200)
	pend = pending(validation(t, c, p.ID))
	if !strings.Contains(pend["repository.connection"].Action, "revogada") {
		t.Fatalf("conexão revogada deve aparecer como pendência: %+v", pend)
	}
	if resp := c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": cur.Version}); resp.StatusCode != 422 {
		t.Fatalf("conexão revogada deve bloquear: %d", resp.StatusCode)
	}
}

func TestInvalidFieldsAreRejectedWithMessage(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	p := createProject(t, c, "campos")
	cases := map[string]map[string]any{
		"dono":    {"repository": map[string]any{"owner": "-ruim-", "name": "a", "base_branch": "main"}},
		"repo":    {"repository": map[string]any{"owner": "o", "name": "a b", "base_branch": "main"}},
		"branch":  {"repository": map[string]any{"owner": "o", "name": "a", "base_branch": "a..b"}},
		"origem":  {"source": map[string]any{"kind": "trello"}},
		"jirakey": {"source": map[string]any{"kind": "jira", "container": "abc"}},
		"nome":    {"name": "   "},
		"prio":    {"priority": 9},
		"etapa":   {"wizard_step": "fim"},
		"conexao": {"repository": map[string]any{"owner": "o", "name": "a", "base_branch": "main", "connection_id": "00000000-0000-4000-8000-000000000000"}},
	}
	for name, body := range cases {
		body["expected_version"] = 1
		if resp := c.do("PATCH", "/api/projects/"+p.ID, body, nil); resp.StatusCode != 400 || resp.ErrCode() != "invalid_request" {
			t.Errorf("%s: esperado 400 invalid_request, veio %d %s", name, resp.StatusCode, resp.Body)
		}
	}
	// Sem expected_version, campo desconhecido e slug inválido.
	if resp := c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"name": "x"}, nil); resp.StatusCode != 400 {
		t.Errorf("sem expected_version: %d", resp.StatusCode)
	}
	if resp := c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": 1, "state": "active"}, nil); resp.StatusCode != 400 {
		t.Errorf("estado não pode ser alterado por PATCH: %d", resp.StatusCode)
	}
	for _, slug := range []string{"Maiusculo", "com espaço", "-x", ""} {
		if resp := c.post("/api/projects", map[string]any{"slug": slug, "name": "x"}); resp.StatusCode != 400 {
			t.Errorf("slug %q: esperado 400, veio %d", slug, resp.StatusCode)
		}
	}
	if resp := c.post("/api/projects", map[string]any{"slug": "campos", "name": "dup"}); resp.StatusCode != 409 || resp.ErrCode() != "duplicate" {
		t.Errorf("slug duplicado: %d %s", resp.StatusCode, resp.Body)
	}
	// Nada disso alterou o projeto.
	var got projects.Project
	c.expect(c.get("/api/projects/"+p.ID), 200).JSON(&got)
	if got.Version != 1 {
		t.Fatalf("falhas não podem mudar a versão: %+v", got)
	}
}

func TestLifecycleActivateSuspendResumeWithTrail(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	p := createProject(t, c, "ciclo")
	gh := conn(t, c, "github", "gh-ciclo")
	jira := conn(t, c, "jira", "jira-ciclo")
	clickup := conn(t, c, "clickup", "clickup-ciclo")
	cur := completeConfig(t, c, p, gh, jira)

	var active projects.Project
	c.expect(c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": cur.Version}), 200).JSON(&active)
	if active.State != projects.StateActive || active.ActivatedAt == nil {
		t.Fatalf("ativação: %+v", active)
	}
	// Ativar de novo, retomar ou suspender com versão errada.
	if resp := c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": active.Version}); resp.StatusCode != 409 || resp.ErrCode() != "invalid_state" {
		t.Errorf("reativar ativo: %d %s", resp.StatusCode, resp.Body)
	}
	if resp := c.post("/api/projects/"+p.ID+"/suspend", map[string]any{"expected_version": active.Version - 1}); resp.StatusCode != 409 || resp.ErrCode() != "version_conflict" {
		t.Errorf("versão obsoleta: %d %s", resp.StatusCode, resp.Body)
	}

	// Ativo: repositório e origem ficam travados; nome continua editável.
	locked := c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": active.Version, "source": map[string]any{"kind": "clickup", "container": "123", "connection_id": clickup.ID},
	}, nil)
	if locked.StatusCode != 409 || locked.ErrCode() != "project_active" {
		t.Fatalf("origem de projeto ativo deveria travar: %d %s", locked.StatusCode, locked.Body)
	}
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": active.Version, "name": "Ciclo renomeado"}, nil), 200).JSON(&active)

	var paused projects.Project
	c.expect(c.post("/api/projects/"+p.ID+"/suspend", map[string]any{"expected_version": active.Version}), 200).JSON(&paused)
	if paused.State != projects.StatePaused {
		t.Fatalf("suspensão: %+v", paused)
	}

	// Suspenso: troca de origem permitida e registrada.
	var changed projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": paused.Version, "source": map[string]any{"kind": "clickup", "container": "123", "connection_id": clickup.ID},
	}, nil), 200).JSON(&changed)

	// Retomar revalida: revogar a conexão da nova origem bloqueia.
	c.expect(c.post("/api/connections/"+clickup.ID+"/revoke", nil), 200)
	if resp := c.post("/api/projects/"+p.ID+"/resume", map[string]any{"expected_version": changed.Version}); resp.StatusCode != 422 {
		t.Fatalf("retomar com conexão revogada deveria bloquear: %d %s", resp.StatusCode, resp.Body)
	}
	var back projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": changed.Version, "source": jiraSource(jira.ID),
	}, nil), 200).JSON(&back)
	c.expect(c.post("/api/projects/"+p.ID+"/resume", map[string]any{"expected_version": back.Version}), 200).JSON(&back)
	if back.State != projects.StateActive {
		t.Fatalf("retomada: %+v", back)
	}

	var trail changesResp
	c.expect(c.get("/api/projects/"+p.ID+"/changes"), 200).JSON(&trail)
	seen := map[string]int{}
	for _, it := range trail.Items {
		seen[it.Action]++
	}
	for action, min := range map[string]int{"created": 1, "updated": 3, "activated": 1, "suspended": 1, "resumed": 1, "source_changed": 2} {
		if seen[action] < min {
			t.Errorf("trilha deveria ter >=%d %q: %v", min, action, seen)
		}
	}
	// A trilha não carrega segredo nem nome de usuário de tentativa.
	if strings.Contains(string(c.get("/api/projects/"+p.ID+"/changes").Body), "segredo-") {
		t.Fatal("trilha não pode conter segredos")
	}
}

// E05 aceite — "Alterar projeto A e demonstrar que projeto B ... preservam suas configurações."
func TestChangingProjectADoesNotTouchProjectB(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	a, b := createProject(t, c, "projeto-a"), createProject(t, c, "projeto-b")
	gh := conn(t, c, "github", "gh-ab")
	var bCfg projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+b.ID, map[string]any{
		"expected_version": 1, "repository": map[string]any{"owner": "org-b", "name": "b", "base_branch": "develop", "connection_id": gh.ID},
	}, nil), 200).JSON(&bCfg)

	c.expect(c.do("PATCH", "/api/projects/"+a.ID, map[string]any{
		"expected_version": 1, "name": "A mudou", "repository": map[string]any{"owner": "org-a", "name": "a", "base_branch": "main"},
	}, nil), 200)

	var gotB projects.Project
	c.expect(c.get("/api/projects/"+b.ID), 200).JSON(&gotB)
	if gotB.Version != bCfg.Version || gotB.Name != bCfg.Name || gotB.Repository.Owner != "org-b" ||
		gotB.Repository.BaseBranch != "develop" || gotB.Repository.ConnectionID == nil || *gotB.Repository.ConnectionID != gh.ID {
		t.Fatalf("projeto B mudou: %+v", gotB)
	}
	var trail changesResp
	c.expect(c.get("/api/projects/"+b.ID+"/changes"), 200).JSON(&trail)
	if len(trail.Items) != 2 { // created + updated do próprio B
		t.Fatalf("a trilha de B só pode ter eventos de B: %+v", trail.Items)
	}
}

func TestConcurrentStepSavesAllowExactlyOneWinner(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	p := createProject(t, c, "corrida")
	const n = 8
	var wg sync.WaitGroup
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp := c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": 1, "description": "desc"}, nil)
			codes <- resp.StatusCode
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
		t.Fatalf("esperado 1 vencedor e %d conflitos, veio %d/%d", n-1, ok, conflict)
	}
}

func TestProjectWritesRequireSessionAndCSRF(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	p := createProject(t, c, "seguro")
	anon := e.newClient()
	for _, r := range []apiResp{
		anon.post("/api/projects", map[string]any{"slug": "x", "name": "x"}),
		anon.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": 1}, nil),
		anon.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": 1}),
		anon.get("/api/projects/" + p.ID + "/validation"),
		anon.get("/api/projects/" + p.ID + "/changes"),
	} {
		if r.StatusCode != 401 {
			t.Errorf("sem sessão: esperado 401, veio %d", r.StatusCode)
		}
	}
	// Com sessão, mas sem token CSRF.
	for _, r := range []apiResp{
		c.do("POST", "/api/projects", map[string]any{"slug": "x", "name": "x"}, map[string]string{"X-CSRF-Token": ""}),
		c.do("PATCH", "/api/projects/"+p.ID, map[string]any{"expected_version": 1, "name": "forjado"}, map[string]string{"X-CSRF-Token": ""}),
		c.do("POST", "/api/projects/"+p.ID+"/suspend", map[string]any{"expected_version": 1}, map[string]string{"X-CSRF-Token": ""}),
	} {
		if r.StatusCode != 403 || r.ErrCode() != "csrf_rejected" {
			t.Errorf("sem CSRF: esperado 403, veio %d %s", r.StatusCode, r.Body)
		}
	}
	var got projects.Project
	c.expect(c.get("/api/projects/"+p.ID), 200).JSON(&got)
	if got.Name != p.Name {
		t.Fatalf("requisição forjada alterou o projeto: %+v", got)
	}
}
