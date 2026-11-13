package integration

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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
