package gitops

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const featID = "f1a2b3c4"

func needGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
}

func hostGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// remote cria <root>/o/r.git (bare, com reflog) com um commit em main.
type remote struct {
	root, bare string
	main       string
}

func newRemote(t *testing.T) remote {
	t.Helper()
	needGit(t)
	root := t.TempDir()
	bare := filepath.Join(root, "o", "r.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	hostGit(t, bare, "init", "--bare", "-b", "main")
	hostGit(t, bare, "config", "core.logAllRefUpdates", "always")
	work := t.TempDir()
	hostGit(t, work, "init", "-b", "main")
	_ = os.WriteFile(filepath.Join(work, "README.md"), []byte("base\n"), 0o644)
	_ = os.WriteFile(filepath.Join(work, "main.go"), []byte("package main\n"), 0o644)
	hostGit(t, work, "add", "-A")
	hostGit(t, work, "commit", "-m", "base")
	hostGit(t, work, "push", bare, "main:refs/heads/main")
	return remote{root: root, bare: bare, main: hostGit(t, bare, "rev-parse", "refs/heads/main")}
}

func (r remote) repo() Repo {
	return Repo{Owner: "o", Name: "r", BaseBranch: "main", BaseURL: "file://" + r.root, AllowFile: true}
}

func (r remote) rev(t *testing.T, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	cmd.Dir = r.bare
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r remote) reflogLines(t *testing.T, ref string) int {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(r.bare, "logs", ref))
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(string(b)), "\n"))
}

type counter struct {
	mu   sync.Mutex
	subs map[string]int
}

func newGit() (*Git, *counter) {
	c := &counter{subs: map[string]int{}}
	return &Git{OnRun: func(sub string, _ []string) { c.mu.Lock(); c.subs[sub]++; c.mu.Unlock() }}, c
}

func (c *counter) n(sub string) int { c.mu.Lock(); defer c.mu.Unlock(); return c.subs[sub] }

type fileSpec struct {
	body string
	mode int64
	link string
}

func snapOf(files map[string]fileSpec) checkpoints.Snapshot {
	m := checkpoints.Manifest{Version: 1}
	objs := map[string][]byte{}
	for p, f := range files {
		switch {
		case f.link != "":
			m.Files = append(m.Files, checkpoints.FileEntry{Path: p, Kind: checkpoints.KindSymlink, Mode: 0o777, Link: f.link})
		default:
			h := sha256.Sum256([]byte(f.body))
			hash := hex.EncodeToString(h[:])
			mode := f.mode
			if mode == 0 {
				mode = 0o644
			}
			m.Files = append(m.Files, checkpoints.FileEntry{Path: p, Kind: checkpoints.KindFile, Mode: mode, Size: int64(len(f.body)), SHA256: hash})
			objs[hash] = []byte(f.body)
			m.TotalBytes += int64(len(f.body))
		}
	}
	sortFiles(m.Files)
	m.TreeHash = checkpoints.TreeHash(m.Files)
	return checkpoints.Snapshot{Manifest: m, Objects: objs}
}

func sortFiles(fs []checkpoints.FileEntry) {
	for i := 1; i < len(fs); i++ {
		for j := i; j > 0 && fs[j].Path < fs[j-1].Path; j-- {
			fs[j], fs[j-1] = fs[j-1], fs[j]
		}
	}
}

func passed() Checks {
	h := sha256.Sum256([]byte("relatorio"))
	return Checks{Passed: true, Names: []string{"test", "lint"}, ReportHash: hex.EncodeToString(h[:])}
}

func cloneOf(t *testing.T, r remote) (*Workdir, *counter) {
	t.Helper()
	g, c := newGit()
	t.Cleanup(g.Cleanup)
	conn := &GitConnector{Repo: r.repo(), Git: g}
	wd, err := conn.Clone(t.Context(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return wd, c
}

func TestRepoValidation(t *testing.T) {
	good := Repo{Owner: "acme", Name: "app", BaseBranch: "main"}
	if err := good.Validate(); err != nil || good.URL() != "https://github.com/acme/app.git" {
		t.Fatalf("%v %s", err, good.URL())
	}
	bad := map[string]Repo{
		"dono":       {Owner: "-x", Name: "app", BaseBranch: "main"},
		"dono com /": {Owner: "a/b", Name: "app", BaseBranch: "main"},
		"nome ..":    {Owner: "acme", Name: "..", BaseBranch: "main"},
		"nome vazio": {Owner: "acme", Name: "", BaseBranch: "main"},
		"ramo ..":    {Owner: "acme", Name: "app", BaseBranch: "a..b"},
		"ramo opção": {Owner: "acme", Name: "app", BaseBranch: "--upload-pack=x"},
		"http":       {Owner: "acme", Name: "app", BaseBranch: "main", BaseURL: "http://github.com"},
		"ext":        {Owner: "acme", Name: "app", BaseBranch: "main", BaseURL: "ext::sh -c id"},
		"file":       {Owner: "acme", Name: "app", BaseBranch: "main", BaseURL: "file:///tmp/x"},
		"credencial": {Owner: "acme", Name: "app", BaseBranch: "main", BaseURL: "https://u:p@github.com"},
		"ssh":        {Owner: "acme", Name: "app", BaseBranch: "main", BaseURL: "ssh://git@github.com"},
		"parâmetros": {Owner: "acme", Name: "app", BaseBranch: "main", BaseURL: "https://github.com?x=1"},
	}
	for name, r := range bad {
		if r.Validate() == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
}

func TestFeatureBranchNamingAndPushGuardRefuseEverythingElse(t *testing.T) {
	b, err := FeatureBranch(featID, 2)
	if err != nil || b != "lf/f-f1a2b3c4-a2" || ValidFeatureBranch(b) != nil {
		t.Fatalf("%q %v", b, err)
	}
	for _, bad := range []struct {
		id  string
		att int
	}{{"", 1}, {"a/b", 1}, {"a b", 1}, {"-x", 1}, {featID, 0}, {featID, 100}} {
		if _, err := FeatureBranch(bad.id, bad.att); err == nil {
			t.Errorf("%+v deveria falhar", bad)
		}
	}
	sha := strings.Repeat("ab", 20)
	ok := PushRequest{ProjectID: "p", FeatureID: featID, Attempt: 2, Branch: b, SHA: sha}
	if err := ok.Guard("main"); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(*PushRequest){
		"base":             func(r *PushRequest) { r.Branch = "main" },
		"outra feature":    func(r *PushRequest) { r.Branch = "lf/f-outra-a2" },
		"outra tentativa":  func(r *PushRequest) { r.Branch = "lf/f-f1a2b3c4-a1" },
		"refspec embutido": func(r *PushRequest) { r.Branch = b + ":refs/heads/main" },
		"force":            func(r *PushRequest) { r.Branch = "+" + b },
		"HEAD":             func(r *PushRequest) { r.Branch = "HEAD" },
		"ramo de release":  func(r *PushRequest) { r.Branch = "release/1.0" },
		"subcaminho":       func(r *PushRequest) { r.Branch = b + "/x" },
		"SHA curto":        func(r *PushRequest) { r.SHA = "abc123" },
		"SHA com opção":    func(r *PushRequest) { r.SHA = "--force" },
		"SHA maiúsculo":    func(r *PushRequest) { r.SHA = strings.ToUpper(sha) },
		"sem projeto":      func(r *PushRequest) { r.ProjectID = "" },
		"tentativa ruim":   func(r *PushRequest) { r.Attempt = 0 },
	}
	for name, mut := range cases {
		r := ok
		mut(&r)
		if err := r.Guard("main"); err == nil {
			t.Errorf("%s: deveria ser recusado", name)
		}
	}
	// A base pode até ser "lf/…" na configuração; ainda assim nunca é publicada.
	weird := ok
	if err := weird.Guard(b); err == nil {
		t.Error("a base nunca é a branch de destino")
	}
	if ok.BusinessKey() != fmt.Sprintf("p:%s:2:push:%s:%s", featID, b, sha[:16]) {
		t.Fatalf("%s", ok.BusinessKey())
	}
}

func TestConnectionIsValidatedByARealClone(t *testing.T) {
	r := newRemote(t)
	g, c := newGit()
	t.Cleanup(g.Cleanup)
	conn := &GitConnector{Repo: r.repo(), Git: g}
	rep, err := conn.Check(t.Context())
	if err != nil || !rep.OK || !rep.CloneOK || rep.BaseSHA != r.main || len(rep.Problems) != 0 || len(rep.Unverified) != 3 {
		t.Fatalf("%+v %v", rep, err)
	}
	if c.n("clone") != 1 || c.n("push") != 0 {
		t.Fatalf("verificar a conexão clona e nunca escreve: %v", c.subs)
	}
	if sha, ok, err := conn.RemoteRef(t.Context(), "main"); err != nil || !ok || sha != r.main {
		t.Fatalf("%q %v %v", sha, ok, err)
	}
	if _, ok, err := conn.RemoteRef(t.Context(), "nao-existe"); err != nil || ok {
		t.Fatalf("ramo ausente não é erro: %v %v", ok, err)
	}
	if r.rev(t, "refs/heads/main") != r.main {
		t.Fatal("o remoto não pode mudar")
	}

	bad := conn
	bad.Repo.BaseBranch = "develop"
	if rep, err := bad.Check(t.Context()); err != nil || rep.OK || !strings.Contains(strings.Join(rep.Problems, " "), "não existe") {
		t.Fatalf("ramo base ausente: %+v %v", rep, err)
	}
	missing := &GitConnector{Repo: Repo{Owner: "o", Name: "nao-existe", BaseBranch: "main", BaseURL: "file://" + r.root, AllowFile: true}, Git: g}
	if rep, err := missing.Check(t.Context()); err != nil || rep.OK || len(rep.Problems) == 0 {
		t.Fatalf("repositório ausente: %+v %v", rep, err)
	}
	if _, err := (&GitConnector{Repo: Repo{Owner: "o", Name: "r", BaseBranch: "main", BaseURL: "http://x"}, Git: g}).Check(t.Context()); !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("%v", err)
	}
}

type fakeAPI struct {
	info RepoInfo
	err  error
}

func (f fakeAPI) Info(context.Context, string, string, string) (RepoInfo, error) {
	return f.info, f.err
}

func TestPreflightUsesRepoAPIWhenAvailable(t *testing.T) {
	r := newRemote(t)
	g, _ := newGit()
	t.Cleanup(g.Cleanup)
	check := func(api RepoAPI) Report {
		rep, err := (&GitConnector{Repo: r.repo(), Git: g, API: api}).Check(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	if rep := check(fakeAPI{info: RepoInfo{CanPush: true, BaseProtected: true}}); !rep.OK || len(rep.Unverified) != 0 {
		t.Fatalf("%+v", rep)
	}
	for name, info := range map[string]RepoInfo{
		"sem permissão":     {CanPush: false, BaseProtected: true},
		"base sem proteção": {CanPush: true, BaseProtected: false},
		"arquivado":         {CanPush: true, BaseProtected: true, Archived: true},
	} {
		if rep := check(fakeAPI{info: info}); rep.OK || len(rep.Problems) == 0 {
			t.Errorf("%s: %+v", name, rep)
		}
	}
	// API fora do ar: o clone vale, o resto fica "não verificado" (e o relatório não afirma o que não viu).
	if rep := check(fakeAPI{err: ErrAuth}); !rep.OK || len(rep.Unverified) != 3 {
		t.Fatalf("%+v", rep)
	}
}

func TestGitHubAPIRequestsAndFixedErrors(t *testing.T) {
	token := "TOKEN-GH-9c1d-nao-pode-vazar"
	var auth []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = append(auth, r.Header.Get("Authorization"))
		switch r.URL.Path {
		case "/repos/acme/app":
			fmt.Fprint(w, `{"default_branch":"main","archived":false,"permissions":{"push":true}}`)
		case "/repos/acme/app/branches/release/1.0":
			fmt.Fprint(w, `{"protected":true}`)
		case "/repos/acme/vazado":
			w.WriteHeader(401)
			fmt.Fprint(w, `{"message":"Bad credentials `+token+`"}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	api, err := NewGitHubAPI(security.NewSecret(token), srv.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	info, err := api.Info(t.Context(), "acme", "app", "release/1.0")
	if err != nil || !info.CanPush || !info.BaseProtected || info.DefaultBranch != "main" {
		t.Fatalf("%+v %v", info, err)
	}
	for _, a := range auth {
		if a != "Bearer "+token {
			t.Fatalf("credencial: %q", a)
		}
	}
	if _, err := api.Info(t.Context(), "acme", "vazado", "main"); !errors.Is(err, ErrAuth) || strings.Contains(err.Error(), token) {
		t.Fatalf("%v", err)
	}
	if _, err := api.Info(t.Context(), "acme", "ninguem", "main"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("%v", err)
	}
	if _, err := api.Info(t.Context(), "a/b", "app", "main"); !errors.Is(err, ErrInvalidRepo) {
		t.Fatalf("%v", err)
	}
	for name, u := range map[string]string{"http": "http://api.exemplo.com", "interno": "https://127.0.0.1", "credencial": "https://u:p@api.exemplo.com"} {
		if _, err := NewGitHubAPI(security.NewSecret("x"), u, false); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	if _, err := NewGitHubAPI(security.Secret{}, "", false); err == nil {
		t.Error("sem credencial")
	}
}

func TestCommitMatchesTaskAndChecksAndBuildsTheExactTree(t *testing.T) {
	r := newRemote(t)
	wd, _ := cloneOf(t, r)
	base, _ := wd.BaseTip(t.Context())
	branch, _ := FeatureBranch(featID, 1)
	files := map[string]fileSpec{
		"README.md":                {body: "base\n"},
		"main.go":                  {body: "package main\n\nfunc main() {}\n"},
		"bin/run.sh":               {body: "#!/bin/sh\necho oi\n", mode: 0o755},
		"docs/com espaço/\"a\".md": {body: "aspas\n"},
		"docs/ação.txt":            {body: "unicode\n"},
		"link":                     {link: "main.go"},
		"fora":                     {link: "/etc/passwd"},
	}
	snap := snapOf(files)
	at := time.Unix(1_700_000_000, 0)
	req := CommitRequest{Branch: branch, Parent: base, Task: Task{Key: "PROJ-12", Title: "Implementa o login"}, Checks: passed(), Snapshot: snap, At: at}
	info, err := wd.Commit(t.Context(), req)
	if err != nil || info.Parent != base || len(info.Sensitive) != 0 {
		t.Fatalf("%+v %v", info, err)
	}
	// O commit corresponde à tarefa e aos checks.
	tr, err := wd.ReadTrailers(t.Context(), info.SHA)
	if err != nil || !tr.Matches(req.Task, req.Checks, snap.Manifest.TreeHash) {
		t.Fatalf("%+v %v", tr, err)
	}
	other := req.Checks
	other.ReportHash = strings.Repeat("0", 64)
	if tr.Matches(req.Task, other, snap.Manifest.TreeHash) || tr.Matches(Task{Key: "PROJ-13"}, req.Checks, snap.Manifest.TreeHash) || tr.Matches(req.Task, req.Checks, strings.Repeat("1", 64)) {
		t.Fatal("outra tarefa, outros checks ou outro conteúdo não podem casar")
	}
	// A árvore tem exatamente o conteúdo, os modos e os links do snapshot.
	ls := hostGit(t, wd.Path, "ls-tree", "-r", "--full-tree", info.SHA)
	if !strings.Contains(ls, "100755 blob") || !strings.Contains(ls, "\tbin/run.sh") || strings.Count(ls, "120000 blob") != 2 {
		t.Fatalf("modos:\n%s", ls)
	}
	if got := hostGit(t, wd.Path, "show", info.SHA+":fora"); got != "/etc/passwd" {
		t.Fatalf("o link vai como link, sem ser seguido: %q", got)
	}
	if got := hostGit(t, wd.Path, "show", info.SHA+":docs/com espaço/\"a\".md"); got != "aspas" {
		t.Fatalf("%q", got)
	}
	if n := strings.Count(hostGit(t, wd.Path, "ls-tree", "-r", "--name-only", "-z", info.SHA), "\x00"); n != len(files) {
		t.Fatalf("a árvore do commit é a do snapshot: %d entradas", n)
	}
	if hostGit(t, wd.Path, "log", "-1", "--format=%an <%ae>", info.SHA) != "La fabrique <fabrique@users.noreply.invalid>" {
		t.Fatal("identidade da fábrica")
	}
	// Determinístico: repetir dá o mesmo SHA (a prova do push depende disso).
	again, err := wd.Commit(t.Context(), req)
	if err != nil || again.SHA != info.SHA {
		t.Fatalf("%+v %v", again, err)
	}
	// Nada novo → sem commit vazio.
	req2 := req
	req2.Parent = info.SHA
	if _, err := wd.Commit(t.Context(), req2); !errors.Is(err, ErrNothingToCommit) {
		t.Fatalf("%v", err)
	}
	// Segunda tarefa encadeia no commit anterior.
	files["main.go"] = fileSpec{body: "package main\n\nfunc main() { println(1) }\n"}
	next, err := wd.Commit(t.Context(), CommitRequest{Branch: branch, Parent: info.SHA, Task: Task{Key: "PROJ-13"}, Checks: passed(), Snapshot: snapOf(files), At: at.Add(time.Hour)})
	if err != nil || next.Parent != info.SHA || next.SHA == info.SHA {
		t.Fatalf("%+v %v", next, err)
	}
	if hostGit(t, wd.Path, "log", "-1", "--format=%s", next.SHA) != "PROJ-13" {
		t.Fatal("sem título, a mensagem usa a chave da tarefa")
	}
	// O remoto continua intocado: commit é local.
	if r.rev(t, "refs/heads/main") != r.main || r.rev(t, "refs/heads/"+branch) != "" {
		t.Fatal("commit não publica nada")
	}
}

func TestCommitRefusesUnvalidatedWorkAndUnsafeContent(t *testing.T) {
	r := newRemote(t)
	wd, c := cloneOf(t, r)
	base, _ := wd.BaseTip(t.Context())
	branch, _ := FeatureBranch(featID, 1)
	good := snapOf(map[string]fileSpec{"a.txt": {body: "x\n"}})
	mk := func(mut func(*CommitRequest)) error {
		req := CommitRequest{Branch: branch, Parent: base, Task: Task{Key: "T-1"}, Checks: passed(), Snapshot: good, At: time.Unix(1_700_000_000, 0)}
		mut(&req)
		_, err := wd.Commit(t.Context(), req)
		return err
	}
	if err := mk(func(r *CommitRequest) { r.Checks.Passed = false }); !errors.Is(err, ErrChecksNotPassed) {
		t.Fatalf("checks reprovados: %v", err)
	}
	for name, mut := range map[string]func(*CommitRequest){
		"sem relatório": func(r *CommitRequest) { r.Checks.ReportHash = "" },
		"sem checks":    func(r *CommitRequest) { r.Checks.Names = nil },
		"hash curto":    func(r *CommitRequest) { r.Checks.ReportHash = "abc" },
	} {
		if err := mk(mut); !errors.Is(err, ErrChecksNotPassed) {
			t.Errorf("%s: %v", name, err)
		}
	}
	for name, mut := range map[string]func(*CommitRequest){
		"branch base":       func(r *CommitRequest) { r.Branch = "main" },
		"pai desconhecido":  func(r *CommitRequest) { r.Parent = strings.Repeat("c", 40) },
		"pai malformado":    func(r *CommitRequest) { r.Parent = "HEAD" },
		"tarefa vazia":      func(r *CommitRequest) { r.Task.Key = "" },
		"tarefa com quebra": func(r *CommitRequest) { r.Task.Key = "T-1\nLf-Checks: x" },
		"título com quebra": func(r *CommitRequest) { r.Task.Title = "a\nb" },
		"check com vírgula": func(r *CommitRequest) { r.Checks.Names = []string{"a,b"} },
		"sem data":          func(r *CommitRequest) { r.At = time.Time{} },
		"autor com <>":      func(r *CommitRequest) { r.Author = Identity{Name: "x <y>", Email: "a@b"} },
		"snapshot adulterado": func(r *CommitRequest) {
			s := snapOf(map[string]fileSpec{"a.txt": {body: "x\n"}})
			s.Manifest.TotalBytes = 99
			r.Snapshot = s
		},
		"objeto ausente": func(r *CommitRequest) {
			s := snapOf(map[string]fileSpec{"a.txt": {body: "x\n"}})
			s.Objects = map[string][]byte{}
			r.Snapshot = s
		},
	} {
		if err := mk(mut); err == nil {
			t.Errorf("%s: deveria ser recusado", name)
		}
	}
	for _, p := range []string{".git/config", "sub/.git/hooks/pre-commit", ".GIT/HEAD", "a/.Git/x"} {
		snap := snapOf(map[string]fileSpec{p: {body: "x"}})
		if err := mk(func(r *CommitRequest) { r.Snapshot = snap }); err == nil {
			t.Errorf("%q deveria ser recusado", p)
		}
	}
	if c.n("fast-import") != 0 {
		t.Fatalf("nada disso pode chegar ao git: %d", c.n("fast-import"))
	}
	// Mudança em .github/ é sinalizada.
	snap := snapOf(map[string]fileSpec{"a.txt": {body: "x\n"}, ".github/workflows/ci.yml": {body: "on: push\n"}})
	info, err := wd.Commit(t.Context(), CommitRequest{Branch: branch, Parent: base, Task: Task{Key: "T-2"}, Checks: passed(), Snapshot: snap, At: time.Unix(1_700_000_000, 0)})
	if err != nil || len(info.Sensitive) != 1 || info.Sensitive[0] != ".github/workflows/ci.yml" {
		t.Fatalf("%+v %v", info, err)
	}
}

func TestNoFileIsExtractedOnTheHostAndHooksNeverRun(t *testing.T) {
	r := newRemote(t)
	// O remoto tem um hook; o clone local nunca o recebe nem o executa.
	wd, _ := cloneOf(t, r)
	base, _ := wd.BaseTip(t.Context())
	branch, _ := FeatureBranch(featID, 1)
	marker := filepath.Join(t.TempDir(), "executou")
	if err := os.MkdirAll(filepath.Join(wd.Path, "hooks"), 0o755); err != nil {
		t.Fatal(err)
	}
	hook := "#!/bin/sh\ntouch " + marker + "\n"
	for _, h := range []string{"pre-commit", "post-commit", "reference-transaction", "pre-push"} {
		_ = os.WriteFile(filepath.Join(wd.Path, "hooks", h), []byte(hook), 0o755)
	}
	snap := snapOf(map[string]fileSpec{"../fora.txt": {body: "x"}})
	if _, err := wd.Commit(t.Context(), CommitRequest{Branch: branch, Parent: base, Task: Task{Key: "T-1"}, Checks: passed(), Snapshot: snap, At: time.Unix(1, 0)}); err == nil {
		t.Fatal("caminho fora do manifesto válido é recusado")
	}
	good := snapOf(map[string]fileSpec{"x/y.txt": {body: "ok"}, "link": {link: "/etc"}})
	if _, err := wd.Commit(t.Context(), CommitRequest{Branch: branch, Parent: base, Task: Task{Key: "T-2"}, Checks: passed(), Snapshot: good, At: time.Unix(1, 0)}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("um hook do repositório executou")
	}
	entries, _ := os.ReadDir(wd.Path)
	for _, e := range entries {
		if e.Name() == "x" || e.Name() == "link" || e.Name() == "y.txt" {
			t.Fatalf("arquivo extraído no host: %s", e.Name())
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(wd.Path), "fora.txt")); err == nil {
		t.Fatal("escreveu fora do clone")
	}
}

func TestCredentialStaysOutOfArgvLogsAndErrors(t *testing.T) {
	token := "TOKEN-GIT-5e7f-nao-pode-vazar"
	repo := Repo{Owner: "acme", Name: "app", BaseBranch: "main", Token: security.NewSecret(token), BaseURL: "https://example.invalid"}
	// git de mentira: ecoa o ambiente e os argumentos no stderr, como um erro real poderia.
	dir := t.TempDir()
	fake := filepath.Join(dir, "git")
	_ = os.WriteFile(fake, []byte("#!/bin/sh\necho \"fatal: Authentication failed for https://x token=$GIT_CONFIG_VALUE_0 args=$*\" >&2\nexit 128\n"), 0o755)
	var seen []string
	g := &Git{Bin: fake, OnRun: func(_ string, args []string) { seen = append(seen, strings.Join(args, " ")) }}
	t.Cleanup(g.Cleanup)
	conn := &GitConnector{Repo: repo, Git: g}
	_, _, err := conn.RemoteRef(t.Context(), "main")
	if !errors.Is(err, ErrAuth) || strings.Contains(err.Error(), token) {
		t.Fatalf("%v", err)
	}
	for _, a := range seen {
		if strings.Contains(a, token) {
			t.Fatalf("o token foi parar em argumentos do git: %s", a)
		}
	}
	rep, err := conn.Check(t.Context())
	if err != nil || rep.OK || strings.Contains(strings.Join(rep.Problems, " "), token) {
		t.Fatalf("%+v %v", rep, err)
	}
	// Mensagem de uma falha genérica também sai redigida.
	_ = os.WriteFile(fake, []byte("#!/bin/sh\necho \"fatal: algo deu errado com $GIT_CONFIG_VALUE_0\" >&2\nexit 1\n"), 0o755)
	_, _, err = conn.RemoteRef(t.Context(), "main")
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("%v", err)
	}
	b64 := "eC1hY2Nlc3MtdG9rZW46" // "x-access-token:" em base64: prefixo do cabeçalho
	if err != nil && strings.Contains(err.Error(), b64) {
		t.Fatalf("o cabeçalho de autenticação vazou: %v", err)
	}
	// O cabeçalho só existe para https (nunca para file://).
	env, _ := (&Git{}).env(&Repo{Token: security.NewSecret("x"), BaseURL: "file:///tmp", AllowFile: true})
	for _, e := range env {
		if strings.HasPrefix(e, "GIT_CONFIG_VALUE_0=") {
			t.Fatal("cabeçalho de autenticação em remoto file://")
		}
	}
}
