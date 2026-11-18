package integration

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newRepoWithBase(t *testing.T) (repo, sha string) {
	t.Helper()
	repo = t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	_ = os.MkdirAll(filepath.Join(repo, "apps/web"), 0o755)
	_ = os.MkdirAll(filepath.Join(repo, "internal/api"), 0o755)
	_ = os.WriteFile(filepath.Join(repo, "apps/web/a.ts"), []byte("a\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "internal/api/b.go"), []byte("package api\n"), 0o644)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "base")
	return repo, gitIn(t, repo, "rev-parse", "HEAD")
}

// AJ06-01 / CT05, CT07 — worktrees e branches distintos por trilha, checkpoint com arquivo não rastreado,
// escopo por caminhos, limpeza isolada por tentativa e recriação após queda.
func TestWorktreesAreIsolatedPerAttemptAndPreserveWork(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "wt")
	repo, base := newRepoWithBase(t)
	m := &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir()}
	var feature string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'WT-1','f') RETURNING id::text`, p.ID).Scan(&feature); err != nil {
		t.Fatal(err)
	}

	front, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: feature, IssueKey: "WT-1", Track: "FRONTEND", Kind: "implementation", Attempt: 1, BaseSHA: base, Paths: []string{"apps/web"}})
	if err != nil {
		t.Fatal(err)
	}
	back, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: feature, IssueKey: "WT-1", Track: "BACKEND", Kind: "implementation", Attempt: 1, BaseSHA: base, Paths: []string{"internal/api"}})
	if err != nil {
		t.Fatal(err)
	}
	if front.Path == back.Path || front.Branch == back.Branch || front.BaseSHA != back.BaseSHA {
		t.Fatalf("diretórios e branches distintos, mesma SHA de partida: %+v %+v", front, back)
	}
	if again, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: feature, Track: "FRONTEND", Kind: "implementation", Attempt: 1, BaseSHA: base}); err != nil || again.ID != front.ID {
		t.Fatalf("repetir devolve a mesma tentativa: %+v %v", again, err)
	}
	// Alterações: arquivo novo não rastreado + modificação no front; arquivo fora do escopo.
	_ = os.WriteFile(filepath.Join(front.Path, "apps/web/new.ts"), []byte("novo\n"), 0o644)
	_ = os.WriteFile(filepath.Join(front.Path, "apps/web/a.ts"), []byte("alterado\n"), 0o644)
	_ = os.WriteFile(filepath.Join(front.Path, "internal/api/b.go"), []byte("package hack\n"), 0o644)
	bad, err := m.Violations(ctx, front.ID)
	if err != nil || len(bad) != 1 || bad[0] != "internal/api/b.go" {
		t.Fatalf("escopo: %v %v", bad, err)
	}
	cp, err := m.Checkpoint(ctx, front.ID)
	if err != nil || cp.Files != 3 {
		t.Fatalf("%+v %v", cp, err)
	}
	if same, _ := m.Checkpoint(ctx, front.ID); same.Seq != cp.Seq {
		t.Fatalf("sem mudança, o checkpoint não se repete: %+v", same)
	}
	if got := gitIn(t, repo, "show", cp.Ref+":apps/web/new.ts"); got != "novo" {
		t.Fatalf("o arquivo não rastreado deve estar no checkpoint: %q", got)
	}
	// Escritor ativo impede a limpeza.
	_ = m.SetWriter(ctx, front.ID, true)
	if err := m.Release(ctx, front.ID); !errors.Is(err, worktrees.ErrWriterActive) {
		t.Fatalf("%v", err)
	}
	_ = m.SetWriter(ctx, front.ID, false)
	// Limpar o front não toca no back.
	_ = os.WriteFile(filepath.Join(back.Path, "internal/api/c.go"), []byte("package api\n"), 0o644)
	if err := m.Release(ctx, front.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(front.Path); !os.IsNotExist(err) {
		t.Fatal("a worktree do front deveria ter sido removida")
	}
	if _, err := os.Stat(filepath.Join(back.Path, "internal/api/c.go")); err != nil {
		t.Fatalf("a worktree do back deve permanecer intacta: %v", err)
	}
	released, _ := m.Get(ctx, front.ID)
	if released.State != "released" || released.ArchiveRef == "" {
		t.Fatalf("%+v", released)
	}
	if got := gitIn(t, repo, "show", cp.Ref+":apps/web/a.ts"); got != "alterado" {
		t.Fatalf("o trabalho do front segue recuperável depois da limpeza: %q", got)
	}
	// Queda: o diretório do back some; provisionar de novo o recria sem perder a branch.
	_ = os.RemoveAll(back.Path)
	rec, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: feature, Track: "BACKEND", Kind: "implementation", Attempt: 1, BaseSHA: base})
	if err != nil || rec.ID != back.ID {
		t.Fatalf("%+v %v", rec, err)
	}
	if _, err := os.Stat(rec.Path); err != nil {
		t.Fatal(err)
	}
	// Entrada inválida.
	if _, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: feature, Track: "BACKEND", Kind: "implementation", Attempt: 2, BaseSHA: base, Paths: []string{"../fora"}}); err == nil {
		t.Fatal("caminho fora do repositório deve ser recusado")
	}
}

func seedFeature(t *testing.T, db testDB, project, ext string) string {
	t.Helper()
	var id string
	if err := db.Pool.QueryRow(context.Background(), `INSERT INTO features (project_id, external_id, title) VALUES ($1,$2,'f') RETURNING id::text`, project, ext).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// ADR-0062 fase 2 — limite de tentativas ativas por projeto e por host, e retry com operationId.
func TestWorktreeLimitsAndOperationLedger(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "lim")
	repo, base := newRepoWithBase(t)
	m := &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir(), MaxPerProject: 2, MaxPerHost: 3}
	f := seedFeature(t, db, p.ID, "L-1")
	spec := func(n int, op string) worktrees.Spec {
		return worktrees.Spec{ProjectID: p.ID, FeatureID: f, Track: "BACKEND", Kind: "implementation", Attempt: n, BaseSHA: base, OperationID: op}
	}
	a1, err := m.Provision(ctx, spec(1, "op-1"))
	if err != nil {
		t.Fatal(err)
	}
	// Retry com o mesmo operationId (inclusive após queda do diretório) devolve a mesma tentativa.
	_ = os.RemoveAll(a1.Path)
	again, err := m.Provision(ctx, spec(1, "op-1"))
	if err != nil || again.ID != a1.ID {
		t.Fatalf("%+v %v", again, err)
	}
	if _, err := os.Stat(again.Path); err != nil {
		t.Fatal("o retry deve recriar a worktree")
	}
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM worktree_attempts WHERE project_id = $1`, p.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("o retry não pode duplicar: %d", n)
	}
	// O mesmo operationId em outra tentativa é recusado.
	if _, err := m.Provision(ctx, spec(2, "op-1")); !errors.Is(err, worktrees.ErrInvalid) {
		t.Fatalf("%v", err)
	}
	if _, err := m.Provision(ctx, spec(2, "op-2")); err != nil {
		t.Fatal(err)
	}
	// Terceira no mesmo projeto passa do limite; repetir uma existente continua permitido.
	if _, err := m.Provision(ctx, spec(3, "op-3")); !errors.Is(err, worktrees.ErrLimit) {
		t.Fatalf("limite por projeto: %v", err)
	}
	if _, err := m.Provision(ctx, spec(2, "op-2")); err != nil {
		t.Fatalf("repetir existente: %v", err)
	}
	// Limite do host: outro projeto ocupa a terceira vaga, a quarta é recusada.
	q := mustProject(t, db.Pool, "lim2")
	qr, qbase := newRepoWithBase(t)
	qm := &worktrees.Manager{DB: db.Pool, Repo: qr, Root: t.TempDir(), MaxPerProject: 2, MaxPerHost: 3}
	qf := seedFeature(t, db, q.ID, "L-2")
	qs := func(n int) worktrees.Spec {
		return worktrees.Spec{ProjectID: q.ID, FeatureID: qf, Track: "BACKEND", Kind: "implementation", Attempt: n, BaseSHA: qbase}
	}
	if _, err := qm.Provision(ctx, qs(1)); err != nil {
		t.Fatal(err)
	}
	if _, err := qm.Provision(ctx, qs(2)); !errors.Is(err, worktrees.ErrLimit) {
		t.Fatalf("limite do host: %v", err)
	}
	// Liberar abre vaga.
	if err := m.Release(ctx, a1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := qm.Provision(ctx, qs(2)); err != nil {
		t.Fatalf("depois de liberar: %v", err)
	}
}

// ADR-0062 fase 2 — escritas concorrentes no mesmo repositório são serializadas e nenhuma se perde.
func TestWorktreeProvisionIsSerializedPerRepository(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "conc")
	repo, base := newRepoWithBase(t)
	m := &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir(), MaxPerProject: -1, MaxPerHost: -1}
	f := seedFeature(t, db, p.ID, "C-1")
	errs := make(chan error, 6)
	for i := 1; i <= 6; i++ {
		go func(n int) {
			_, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: f, Track: "FRONTEND", Kind: "implementation", Attempt: n, BaseSHA: base})
			errs <- err
		}(i)
	}
	for i := 0; i < 6; i++ {
		if err := <-errs; err != nil {
			t.Fatal(err)
		}
	}
	if out := gitIn(t, repo, "worktree", "list", "--porcelain"); strings.Count(out, "worktree ") != 7 {
		t.Fatalf("esperava 7 worktrees (principal + 6):\n%s", out)
	}
}

// ADR-0062 fase 2 — base: explícita > branch do projeto > origin/HEAD; links para fora são detectados.
func TestWorktreeBaseResolutionAndEscapes(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "base")
	repo, base := newRepoWithBase(t)
	gitIn(t, repo, "branch", "feature-x")
	_ = os.WriteFile(filepath.Join(repo, "n.txt"), []byte("n\n"), 0o644)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "segundo")
	head := gitIn(t, repo, "rev-parse", "HEAD")
	m := &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir()}
	if got, err := m.ResolveBase(ctx, "feature-x", "main"); err != nil || got != base {
		t.Fatalf("explícita vence: %s %v", got, err)
	}
	if got, err := m.ResolveBase(ctx, "", "main"); err != nil || got != head {
		t.Fatalf("branch do projeto: %s %v", got, err)
	}
	if _, err := m.ResolveBase(ctx, "nao-existe", "main"); !errors.Is(err, worktrees.ErrInvalid) {
		t.Fatalf("explícita inexistente não cai no padrão: %v", err)
	}
	if _, err := m.ResolveBase(ctx, "a..b", ""); !errors.Is(err, worktrees.ErrInvalid) {
		t.Fatalf("nome inválido: %v", err)
	}
	gitIn(t, repo, "update-ref", "refs/remotes/origin/main", head)
	gitIn(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	if got, err := m.ResolveBase(ctx, "", "sumiu"); err != nil || got != head {
		t.Fatalf("origin/HEAD: %s %v", got, err)
	}

	f := seedFeature(t, db, p.ID, "B-1")
	a, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: f, Track: "BACKEND", Kind: "implementation", Attempt: 1, BaseSHA: base})
	if err != nil {
		t.Fatal(err)
	}
	if up := gitIn(t, repo, "for-each-ref", "--format=%(upstream)", "refs/heads/lf-tmp"); up != "" {
		t.Fatalf("a branch da tentativa não pode rastrear nada: %s", up)
	}
	outside := t.TempDir()
	_ = os.Symlink(outside, filepath.Join(a.Path, "fuga"))
	_ = os.Symlink("apps/web", filepath.Join(a.Path, "interno"))
	bad, err := m.Escapes(ctx, a.ID)
	if err != nil || len(bad) != 1 || bad[0] != "fuga" {
		t.Fatalf("%v %v", bad, err)
	}
}

// ADR-0062 fase 2 — o Provider monta o Manager a partir da pasta do projeto, dentro da raiz.
func TestWorktreeProviderUsesTheProjectFolder(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "prov")
	root := t.TempDir()
	repo := filepath.Join(root, "meu-app")
	_ = os.MkdirAll(repo, 0o755)
	gitIn(t, repo, "init", "-q", "-b", "main")
	_ = os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "base")
	base := gitIn(t, repo, "rev-parse", "HEAD")
	prov := worktrees.Provider{DB: db.Pool, Root: workspace.Root{Dir: root}}

	if _, err := prov.For(ctx, p.ID); !errors.Is(err, worktrees.ErrInvalid) {
		t.Fatalf("sem pasta: %v", err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE projects SET repo_path = 'meu-app' WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	m, err := prov.For(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(repo)
	if m.Repo != real {
		t.Fatalf("o checkout principal é a pasta do projeto: %s", m.Repo)
	}
	f := seedFeature(t, db, p.ID, "P-1")
	a, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: f, Track: "BACKEND", Kind: "implementation", Attempt: 1, BaseSHA: base})
	if err != nil || !strings.Contains(a.Path, worktrees.WorktreesDir) {
		t.Fatalf("%+v %v", a, err)
	}
	// O checkout principal não é alterado (branch e árvore) e a pasta oculta não aparece na navegação.
	if br := gitIn(t, repo, "symbolic-ref", "--short", "HEAD"); br != "main" {
		t.Fatalf("branch do checkout principal mudou: %s", br)
	}
	if st := gitIn(t, repo, "status", "--porcelain"); st != "" {
		t.Fatalf("a árvore do checkout principal deve seguir limpa: %q", st)
	}
	l, err := workspace.Root{Dir: root}.Browse("")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range l.Entries {
		if e.Name == worktrees.WorktreesDir {
			t.Fatal("a pasta das worktrees não deve aparecer na navegação")
		}
	}
	// Fora da raiz: link simbólico no caminho do projeto é recusado.
	_ = os.Symlink(t.TempDir(), filepath.Join(root, "fuga"))
	_, _ = db.Pool.Exec(ctx, `UPDATE projects SET repo_path = 'fuga' WHERE id = $1`, p.ID)
	if _, err := prov.For(ctx, p.ID); !errors.Is(err, worktrees.ErrInvalid) {
		t.Fatalf("fora da raiz: %v", err)
	}
}
