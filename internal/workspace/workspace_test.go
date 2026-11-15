package workspace

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não instalado")
	}
}

func TestResolveConfinesToRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "fuga")); err != nil {
		t.Skip("sem symlink")
	}
	r := Root{Dir: root}
	for _, bad := range []string{"..", "a/../..", "../x", "a/./b", "fuga", "fuga/novo", "a\\b"} {
		if _, err := r.Resolve(bad); err == nil {
			t.Fatalf("%q deveria ser recusado", bad)
		}
	}
	got, err := r.Resolve("projeto/novo")
	if err != nil || !strings.HasSuffix(got, filepath.Join("projeto", "novo")) {
		t.Fatalf("%q %v", got, err)
	}
	if _, err := (Root{}).Resolve("x"); !errors.Is(err, ErrDisabled) {
		t.Fatalf("raiz desligada: %v", err)
	}
}

func TestBrowseListsOnlySafeDirs(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "a", ".git"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "vazia"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, ".oculta"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "arquivo.txt"), nil, 0o644)
	_ = os.Symlink(outside, filepath.Join(root, "fuga"))
	l, err := (Root{Dir: root}).Browse("")
	if err != nil || !l.Enabled || len(l.Entries) != 2 {
		t.Fatalf("%+v %v", l, err)
	}
	if l.Entries[0].Name != "a" || !l.Entries[0].IsRepo || l.Entries[1].Name != "vazia" || !l.Entries[1].Empty {
		t.Fatalf("%+v", l.Entries)
	}
	if _, err := (Root{Dir: root}).Browse("nao-existe"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%v", err)
	}
	off, _ := (Root{}).Browse("")
	if off.Enabled {
		t.Fatal("deveria estar desligado")
	}
}

func TestParseOrigin(t *testing.T) {
	for in, want := range map[string]string{
		"https://github.com/acme/app.git":         "acme/app",
		"git@github.com:acme/app.git":             "acme/app",
		"https://x-access-token:t@github.com/a/b": "a/b",
		"https://github.com/acme/app.v2/":         "acme/app.v2",
	} {
		o, n, ok := ParseOrigin(in)
		if !ok || o+"/"+n != want {
			t.Fatalf("%s → %s/%s %v", in, o, n, ok)
		}
	}
	if _, _, ok := ParseOrigin("https://gitlab.com/a/b.git"); ok {
		t.Fatal("só GitHub")
	}
}

func gitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}

func service(t *testing.T) (Service, string) {
	needGit(t)
	root := t.TempDir()
	return Service{Root: Root{Dir: root}, Git: Git{AllowFile: true}}, root
}

func TestCreateNewLocalOnly(t *testing.T) {
	s, root := service(t)
	res, err := s.Prepare(context.Background(), Request{Mode: ModeNew, Slug: "meu-app", Title: "Meu app"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Path != "meu-app" || res.Base != "main" || res.Owner != "" {
		t.Fatalf("%+v", res)
	}
	info, err := s.Git.Inspect(context.Background(), filepath.Join(root, "meu-app"))
	if err != nil || info.Branch != "main" || !info.HasCommits {
		t.Fatalf("%+v %v", info, err)
	}
	// de novo: pasta ocupada
	_, err = s.Prepare(context.Background(), Request{Mode: ModeNew, Slug: "meu-app", Title: "x"})
	var ue *UserError
	if !errors.As(err, &ue) || ue.Status != 409 {
		t.Fatalf("%v", err)
	}
}

func TestCreateNewWithRemoteAndPush(t *testing.T) {
	s, root := service(t)
	// "GitHub" falso: API e remoto bare em disco.
	remotes := t.TempDir()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(401)
			return
		}
		if r.Method == "POST" && r.URL.Path == "/user/repos" {
			_ = os.MkdirAll(filepath.Join(remotes, "dev", "novo.git"), 0o755)
			c := exec.Command("git", "init", "--bare", "-q")
			c.Dir = filepath.Join(remotes, "dev", "novo.git")
			_ = c.Run()
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"full_name":"dev/novo","name":"novo","private":true,"default_branch":"main","owner":{"login":"dev"}}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer api.Close()
	s.GH = GitHub{BaseURL: api.URL}
	s.WebURL = "file://" + remotes
	res, err := s.Prepare(context.Background(), Request{Mode: ModeNew, Slug: "novo", Title: "Novo", CreateRemote: true,
		Private: true, Token: security.NewSecret("tok"), Login: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Owner != "dev" || res.Name != "novo" {
		t.Fatalf("%+v", res)
	}
	c := exec.Command("git", "-C", filepath.Join(remotes, "dev", "novo.git"), "rev-parse", "--verify", "main")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("o push não chegou: %v %s", err, out)
	}
	_ = root
}

func TestCreateRemoteFailureRollsBack(t *testing.T) {
	s, root := service(t)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(422) }))
	defer api.Close()
	s.GH = GitHub{BaseURL: api.URL}
	_, err := s.Prepare(context.Background(), Request{Mode: ModeNew, Slug: "dup", Title: "d", CreateRemote: true, Token: security.NewSecret("tok"), Login: "dev"})
	var ue *UserError
	if !errors.As(err, &ue) || ue.Code != "github_create_failed" {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "dup")); !os.IsNotExist(err) {
		t.Fatal("a pasta criada deveria ter sido removida")
	}
	if _, err := s.Prepare(context.Background(), Request{Mode: ModeNew, Slug: "x", CreateRemote: true}); err == nil {
		t.Fatal("sem conexão deveria falhar")
	}
}

func TestLocalExistingRepo(t *testing.T) {
	s, root := service(t)
	dir := filepath.Join(root, "existente")
	_ = os.MkdirAll(dir, 0o755)
	gitCmd(t, dir, "init", "-b", "develop")
	_ = os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	gitCmd(t, dir, "add", ".")
	gitCmd(t, dir, "commit", "-m", "i")
	gitCmd(t, dir, "remote", "add", "origin", "https://github.com/acme/legado.git")
	res, err := s.Prepare(context.Background(), Request{Mode: ModeLocal, Slug: "x", Path: "existente"})
	if err != nil || res.Owner != "acme" || res.Name != "legado" || res.Base != "develop" || res.Path != "existente" {
		t.Fatalf("%+v %v", res, err)
	}
	// pasta sem git
	_ = os.MkdirAll(filepath.Join(root, "comum"), 0o755)
	if _, err := s.Prepare(context.Background(), Request{Mode: ModeLocal, Slug: "x", Path: "comum"}); err == nil {
		t.Fatal("pasta sem git deveria falhar")
	}
	// fora da raiz
	if _, err := s.Prepare(context.Background(), Request{Mode: ModeLocal, Slug: "x", Path: "../etc"}); err == nil {
		t.Fatal("fora da raiz deveria falhar")
	}
	// sem caminho
	if _, err := s.Prepare(context.Background(), Request{Mode: ModeLocal, Slug: "x"}); err == nil {
		t.Fatal("sem caminho deveria falhar")
	}
}

func TestCloneFromRemoteAndReuse(t *testing.T) {
	s, root := service(t)
	remotes := t.TempDir()
	bare := filepath.Join(remotes, "acme", "app.git")
	_ = os.MkdirAll(bare, 0o755)
	gitCmd(t, bare, "init", "--bare", "-b", "main")
	seed := t.TempDir()
	gitCmd(t, seed, "init", "-b", "main")
	_ = os.WriteFile(filepath.Join(seed, "a"), []byte("x"), 0o644)
	gitCmd(t, seed, "add", ".")
	gitCmd(t, seed, "commit", "-m", "i")
	gitCmd(t, seed, "remote", "add", "origin", "file://"+bare)
	gitCmd(t, seed, "push", "origin", "main")
	s.WebURL = "file://" + remotes
	req := Request{Mode: ModeGitHub, Slug: "app", Owner: "acme", Name: "app", Base: "main"}
	res, err := s.Prepare(context.Background(), req)
	if err != nil || res.Path != "app" {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := os.Stat(filepath.Join(root, "app", "a")); err != nil {
		t.Fatal("o clone não trouxe os arquivos")
	}
	// origin do clone é file://, então não bate com acme/app: tratado como outro repositório.
	if _, err := s.Prepare(context.Background(), req); err == nil {
		t.Fatal("pasta ocupada deveria falhar")
	}
	// sem raiz: só salva
	res, err = (Service{Git: Git{}}).Prepare(context.Background(), req)
	if err != nil || res.Path != "" {
		t.Fatalf("%+v %v", res, err)
	}
	// clone de ramo inexistente falha com mensagem
	req2 := Request{Mode: ModeGitHub, Slug: "app2", Owner: "acme", Name: "app", Base: "nao-existe"}
	var ue *UserError
	if _, err := s.Prepare(context.Background(), req2); !errors.As(err, &ue) || ue.Code != "clone_failed" {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "app2")); !os.IsNotExist(err) {
		t.Fatal("clone falho deveria limpar a pasta")
	}
}

func TestGitHubListAndErrors(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ok" {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`[{"full_name":"a/b","name":"b","private":false,"default_branch":"main","owner":{"login":"a"},"permissions":{"push":true}},{"full_name":"a/old","name":"old","archived":true,"owner":{"login":"a"}}]`))
	}))
	defer api.Close()
	g := GitHub{BaseURL: api.URL}
	repos, err := g.ListRepos(context.Background(), security.NewSecret("ok"))
	if err != nil || len(repos) != 1 || repos[0].FullName != "a/b" || !repos[0].CanPush {
		t.Fatalf("%+v %v", repos, err)
	}
	if _, err := g.ListRepos(context.Background(), security.NewSecret("ruim")); !errors.Is(err, ErrGitHubAuth) {
		t.Fatalf("%v", err)
	}
}

func stepsByID(steps []Step) map[string]Step {
	m := map[string]Step{}
	for _, s := range steps {
		m[s.ID] = s
	}
	return m
}

func fakeRepoAPI(t *testing.T, push, archived bool, branches ...string) GitHub {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.Trim(r.URL.Path, "/")
		switch {
		case p == "repos/acme/app":
			_, _ = w.Write([]byte(`{"full_name":"acme/app","default_branch":"main","archived":` + map[bool]string{true: "true", false: "false"}[archived] +
				`,"permissions":{"push":` + map[bool]string{true: "true", false: "false"}[push] + `}}`))
		case strings.HasPrefix(p, "repos/acme/app/branches/"):
			for _, b := range branches {
				if p == "repos/acme/app/branches/"+b {
					_, _ = w.Write([]byte(`{}`))
					return
				}
			}
			w.WriteHeader(404)
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(api.Close)
	return GitHub{BaseURL: api.URL}
}

func TestCheckRepositoryHappyPathAndFailures(t *testing.T) {
	s, root := service(t)
	dir := filepath.Join(root, "app")
	_ = os.MkdirAll(dir, 0o755)
	gitCmd(t, dir, "init", "-b", "main")
	_ = os.WriteFile(filepath.Join(dir, "a"), []byte("x"), 0o644)
	gitCmd(t, dir, "add", ".")
	gitCmd(t, dir, "commit", "-m", "i")
	gitCmd(t, dir, "remote", "add", "origin", "https://github.com/acme/app.git")
	tok := security.NewSecret("t")
	req := CheckRequest{Path: "app", Owner: "acme", Name: "app", Base: "main", Token: tok}

	s.GH = fakeRepoAPI(t, true, false, "main")
	ok, steps := s.Check(context.Background(), req)
	if !ok {
		t.Fatalf("deveria passar: %+v", steps)
	}
	if got := stepsByID(steps); got["clean"].Status != StepOK || got["push"].Status != StepOK || got["base_remote"].Status != StepOK {
		t.Fatalf("%+v", steps)
	}

	// árvore suja só avisa
	_ = os.WriteFile(filepath.Join(dir, "novo"), []byte("y"), 0o644)
	ok, steps = s.Check(context.Background(), req)
	if !ok || stepsByID(steps)["clean"].Status != StepWarn {
		t.Fatalf("suja: %v %+v", ok, steps)
	}

	// somente leitura, arquivado, branch ausente, origin diferente, sem conexão
	s.GH = fakeRepoAPI(t, false, false, "main")
	if ok, steps = s.Check(context.Background(), req); ok || stepsByID(steps)["push"].Status != StepFailed {
		t.Fatalf("sem push: %+v", steps)
	}
	s.GH = fakeRepoAPI(t, true, true, "main")
	if ok, steps = s.Check(context.Background(), req); ok || !strings.Contains(stepsByID(steps)["push"].Detail, "arquivado") {
		t.Fatalf("arquivado: %+v", steps)
	}
	s.GH = fakeRepoAPI(t, true, false)
	if ok, steps = s.Check(context.Background(), req); ok || stepsByID(steps)["base_remote"].Status != StepFailed {
		t.Fatalf("sem branch no remoto: %+v", steps)
	}
	s.GH = fakeRepoAPI(t, true, false, "main")
	other := req
	other.Owner, other.Name = "acme", "outro"
	if ok, steps = s.Check(context.Background(), other); ok || stepsByID(steps)["origin"].Status != StepFailed {
		t.Fatalf("origin diferente: %+v", steps)
	}
	noTok := req
	noTok.Token = security.Secret{}
	if ok, steps = s.Check(context.Background(), noTok); ok || stepsByID(steps)["account"].Status != StepFailed {
		t.Fatalf("sem conexão: %+v", steps)
	}
	badBase := req
	badBase.Base = "inexistente"
	if ok, steps = s.Check(context.Background(), badBase); ok || stepsByID(steps)["base_local"].Status != StepFailed {
		t.Fatalf("branch base local: %+v", steps)
	}
	// pasta sumiu e pasta fora da raiz
	gone := req
	gone.Path = "nao-existe"
	if ok, steps = s.Check(context.Background(), gone); ok || stepsByID(steps)["folder"].Status != StepFailed {
		t.Fatalf("pasta ausente: %+v", steps)
	}
	gone.Path = "../fora"
	if ok, _ = s.Check(context.Background(), gone); ok {
		t.Fatal("fora da raiz deveria falhar")
	}
	// sem pasta local: só a parte remota
	remote := req
	remote.Path = ""
	if ok, steps = s.Check(context.Background(), remote); !ok || stepsByID(steps)["folder"].Status != StepSkipped {
		t.Fatalf("sem pasta: %+v", steps)
	}
}
