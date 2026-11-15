package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
)

// ADR-0061: clonar, usar pasta existente e criar do zero, com GitHub e remotos falsos em disco.
func TestRepositoryModes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não instalado")
	}
	root := t.TempDir()
	remotes := t.TempDir()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer segredo-") {
			w.WriteHeader(401)
			return
		}
		switch {
		case r.Method == "GET" && r.URL.Path == "/user/repos":
			fmt.Fprint(w, `[{"full_name":"dev/existente","name":"existente","private":true,"default_branch":"main","owner":{"login":"dev"},"permissions":{"push":true}}]`)
		case r.Method == "POST" && r.URL.Path == "/user/repos":
			bare := filepath.Join(remotes, "dev", "criado.git")
			_ = os.MkdirAll(bare, 0o755)
			cmd := exec.Command("git", "init", "--bare", "-q")
			cmd.Dir = bare
			_ = cmd.Run()
			w.WriteHeader(201)
			fmt.Fprint(w, `{"full_name":"dev/criado","name":"criado","private":true,"default_branch":"main","owner":{"login":"dev"}}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer api.Close()

	e := newAuthEnv(t, func(c *config.API) { c.WorkspacesDir = root },
		httpapi.WithWorkspaceSetup(func(s *workspace.Service) {
			s.Git.AllowFile = true
			s.GH = workspace.GitHub{BaseURL: api.URL}
			s.WebURL = "file://" + remotes
		}))
	c := e.newClient()
	c.login(nil)
	gh := conn(t, c, "github", "gh-repo")

	// área de workspaces e lista de repositórios da conta
	var l workspace.Listing
	c.expect(c.get("/api/workspace/browse?path="), 200).JSON(&l)
	if !l.Enabled {
		t.Fatalf("workspace deveria estar ligado: %+v", l)
	}
	c.expect(c.get("/api/workspace/browse?path=../x"), 400)
	var repos struct{ Items []workspace.RepoRef }
	c.expect(c.get("/api/github/repos?connection_id="+gh.ID), 200).JSON(&repos)
	if len(repos.Items) != 1 || repos.Items[0].FullName != "dev/existente" {
		t.Fatalf("repos: %+v", repos.Items)
	}
	jira := conn(t, c, "jira", "jira-repo")
	c.expect(c.get("/api/github/repos?connection_id="+jira.ID), 400)

	prepare := func(p projects.Project, body map[string]any) (apiResp, projects.Project) {
		body["expected_version"] = p.Version
		resp := c.post("/api/projects/"+p.ID+"/repository/prepare", body)
		var out struct {
			Project projects.Project `json:"project"`
			Notes   []string         `json:"notes"`
		}
		if resp.StatusCode == 200 {
			resp.JSON(&out)
		}
		return resp, out.Project
	}

	t.Run("criar novo com repositório no GitHub", func(t *testing.T) {
		p := createProject(t, c, "app-novo")
		resp, got := prepare(p, map[string]any{"mode": "new", "create_remote": true, "private": true, "connection_id": gh.ID, "name": "criado"})
		if resp.StatusCode != 200 {
			t.Fatalf("%d %s", resp.StatusCode, resp.Body)
		}
		r := got.Repository
		if r.Mode != "new" || r.Path != "app-novo" || r.Owner != "dev" || r.Name != "criado" || r.BaseBranch != "main" {
			t.Fatalf("repositório: %+v", r)
		}
		if _, err := os.Stat(filepath.Join(root, "app-novo", ".git")); err != nil {
			t.Fatal("pasta não foi criada")
		}
		out, err := exec.Command("git", "-C", filepath.Join(remotes, "dev", "criado.git"), "rev-parse", "--verify", "main").CombinedOutput()
		if err != nil {
			t.Fatalf("push não chegou: %s", out)
		}
		// versão desatualizada é recusada antes de mexer em disco
		stale := map[string]any{"mode": "new", "path": "outra", "expected_version": p.Version}
		c.expect(c.post("/api/projects/"+p.ID+"/repository/prepare", stale), 409)
		if _, err := os.Stat(filepath.Join(root, "outra")); err == nil {
			t.Fatal("não deveria ter criado a pasta com versão velha")
		}
	})

	t.Run("pasta existente", func(t *testing.T) {
		dir := filepath.Join(root, "legado")
		_ = os.MkdirAll(dir, 0o755)
		for _, args := range [][]string{{"init", "-b", "trunk"}, {"remote", "add", "origin", "https://github.com/acme/legado.git"}} {
			cmd := exec.Command("git", args...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("%s", out)
			}
		}
		p := createProject(t, c, "app-legado")
		resp, got := prepare(p, map[string]any{"mode": "local", "path": "legado"})
		if resp.StatusCode != 200 {
			t.Fatalf("%d %s", resp.StatusCode, resp.Body)
		}
		r := got.Repository
		if r.Mode != "local" || r.Path != "legado" || r.Owner != "acme" || r.Name != "legado" || r.BaseBranch != "trunk" {
			t.Fatalf("repositório: %+v", r)
		}
		// pasta fora da raiz e pasta sem git
		// a mesma pasta em outro projeto é recusada
		p3 := createProject(t, c, "app-legado-2")
		if resp, _ := prepare(p3, map[string]any{"mode": "local", "path": "legado"}); resp.StatusCode != 409 || resp.ErrCode() != "workspace_taken" {
			t.Fatalf("pasta repetida: %d %s", resp.StatusCode, resp.Body)
		}
		p2 := createProject(t, c, "app-fora")
		if resp, _ := prepare(p2, map[string]any{"mode": "local", "path": "../etc"}); resp.StatusCode != 400 {
			t.Fatalf("fora da raiz: %d %s", resp.StatusCode, resp.Body)
		}
		_ = os.MkdirAll(filepath.Join(root, "sem-git"), 0o755)
		if resp, _ := prepare(p2, map[string]any{"mode": "local", "path": "sem-git"}); resp.StatusCode != 400 {
			t.Fatalf("sem git: %d %s", resp.StatusCode, resp.Body)
		}
	})

	t.Run("clonar do GitHub", func(t *testing.T) {
		bare := filepath.Join(remotes, "dev", "existente.git")
		_ = os.MkdirAll(bare, 0o755)
		seed := t.TempDir()
		run := func(dir string, args ...string) {
			cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
			cmd.Dir = dir
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %s", args, out)
			}
		}
		run(bare, "init", "--bare", "-q", "-b", "main")
		run(seed, "init", "-q", "-b", "main")
		_ = os.WriteFile(filepath.Join(seed, "a.txt"), []byte("x"), 0o644)
		run(seed, "add", ".")
		run(seed, "commit", "-q", "-m", "i")
		run(seed, "push", "-q", "file://"+bare, "main")

		p := createProject(t, c, "app-clone")
		resp, got := prepare(p, map[string]any{"mode": "github", "connection_id": gh.ID, "owner": "dev", "name": "existente", "base_branch": "main"})
		if resp.StatusCode != 200 {
			t.Fatalf("%d %s", resp.StatusCode, resp.Body)
		}
		if got.Repository.Path != "app-clone" || got.Repository.Owner != "dev" {
			t.Fatalf("repositório: %+v", got.Repository)
		}
		if _, err := os.Stat(filepath.Join(root, "app-clone", "a.txt")); err != nil {
			t.Fatal("clone sem arquivos")
		}
	})

	t.Run("sem sessão e CSRF", func(t *testing.T) {
		p := createProject(t, c, "app-csrf")
		body := map[string]any{"mode": "new", "expected_version": p.Version}
		if resp := e.newClient().post("/api/workspace/browse", nil); resp.StatusCode == 200 {
			t.Fatal("sem sessão não deveria funcionar")
		}
		if r := c.do("POST", "/api/projects/"+p.ID+"/repository/prepare", body, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
			t.Fatalf("sem CSRF: %d", r.StatusCode)
		}
	})
}

// ADR-0062: ativação exige o repositório verificado; mudar a configuração invalida; o mesmo repositório
// não pode estar em dois projetos ativos.
func TestRepositoryVerificationGatesActivation(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	gh := conn(t, c, "github", "gh-v")
	jira := conn(t, c, "jira", "jira-v")

	pending := func(id, check string) bool {
		for _, ch := range validation(t, c, id).Checks {
			if ch.ID == check {
				return ch.Status != "ok"
			}
		}
		t.Fatalf("check %s não existe", check)
		return false
	}

	p := configure(t, c, createProject(t, c, "verif-a"), gh, jira)
	var sc projects.SourceCheck
	c.expect(c.post("/api/projects/"+p.ID+"/source/check", nil), 200).JSON(&sc)
	if !pending(p.ID, "repository.verified") {
		t.Fatal("repositório não verificado deveria estar pendente")
	}
	if resp := c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": p.Version}); resp.StatusCode != 422 {
		t.Fatalf("sem verificação: %d %s", resp.StatusCode, resp.Body)
	}
	var rc projects.RepoCheck
	c.expect(c.post("/api/projects/"+p.ID+"/repository/check", nil), 200).JSON(&rc)
	if !rc.OK || len(rc.Steps) < 3 {
		t.Fatalf("verificação: %+v", rc)
	}
	if pending(p.ID, "repository.verified") {
		t.Fatal("verificação deveria valer")
	}
	var active projects.Project
	c.expect(c.post("/api/projects/"+p.ID+"/activate", map[string]any{"expected_version": p.Version}), 200).JSON(&active)

	// outro projeto com o mesmo repositório: duplicado
	q := createProject(t, c, "verif-b")
	var q2 projects.Project
	c.expect(c.do("PATCH", "/api/projects/"+q.ID, map[string]any{
		"expected_version": q.Version,
		"repository":       map[string]any{"owner": "org-exemplo", "name": "verif-a", "base_branch": "main", "connection_id": gh.ID},
	}, nil), 200).JSON(&q2)
	if !pending(q.ID, "repository.unique") {
		t.Fatal("o mesmo repositório em dois projetos ativos deveria ser recusado")
	}
	// a verificação do repositório de A vale só para a configuração dela: suspender e mudar invalida
	var susp projects.Project
	c.expect(c.post("/api/projects/"+p.ID+"/suspend", map[string]any{"expected_version": active.Version}), 200).JSON(&susp)
	c.expect(c.do("PATCH", "/api/projects/"+p.ID, map[string]any{
		"expected_version": susp.Version,
		"repository":       map[string]any{"owner": "org-exemplo", "name": "verif-a", "base_branch": "develop", "connection_id": gh.ID},
	}, nil), 200)
	if !pending(p.ID, "repository.verified") {
		t.Fatal("mudar a configuração deveria invalidar a verificação")
	}
	// repositório incompleto não verifica
	c.expect(c.post("/api/projects/"+createProject(t, c, "verif-c").ID+"/repository/check", nil), 409)
}
