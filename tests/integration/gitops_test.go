package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

func ghGit(t *testing.T, dir string, args ...string) string {
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

type ghRemote struct{ root, bare, main string }

func newGHRemote(t *testing.T) ghRemote {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git indisponível")
	}
	root := t.TempDir()
	bare := filepath.Join(root, "o", "r.git")
	if err := os.MkdirAll(bare, 0o755); err != nil {
		t.Fatal(err)
	}
	ghGit(t, bare, "init", "--bare", "-b", "main")
	ghGit(t, bare, "config", "core.logAllRefUpdates", "always")
	work := t.TempDir()
	ghGit(t, work, "init", "-b", "main")
	_ = os.WriteFile(filepath.Join(work, "README.md"), []byte("base\n"), 0o644)
	ghGit(t, work, "add", "-A")
	ghGit(t, work, "commit", "-m", "base")
	ghGit(t, work, "push", bare, "main:refs/heads/main")
	return ghRemote{root: root, bare: bare, main: ghGit(t, bare, "rev-parse", "refs/heads/main")}
}

func (r ghRemote) rev(t *testing.T, ref string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", ref)
	cmd.Dir = r.bare
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (r ghRemote) updates(ref string) int {
	b, err := os.ReadFile(filepath.Join(r.bare, "logs", ref))
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(string(b)), "\n"))
}

type ghEnv struct {
	remote ghRemote
	proj   string
	feat   string
	branch string
	wd     *gitops.Workdir
	pub    *gitops.Publisher
	pushes *int
	mu     *sync.Mutex
	e      *authEnv
	escal  *[]string
}

// newGHEnv monta clone bare, ledger e publicador. intercept (opcional) envolve o `git push`.
func newGHEnv(t *testing.T, intercept func(run func() error, n int) error) ghEnv {
	t.Helper()
	remote := newGHRemote(t)
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "gitops")
	var mu sync.Mutex
	pushes := 0
	g := &gitops.Git{
		OnRun: func(sub string, _ []string) {
			if sub == "push" {
				mu.Lock()
				pushes++
				mu.Unlock()
			}
		},
		Intercept: func(sub string, run func() error) error {
			if sub == "push" && intercept != nil {
				mu.Lock()
				n := pushes
				mu.Unlock()
				return intercept(run, n)
			}
			return run()
		},
	}
	t.Cleanup(g.Cleanup)
	repo := gitops.Repo{Owner: "o", Name: "r", BaseBranch: "main", BaseURL: "file://" + remote.root, AllowFile: true}
	wd, err := (&gitops.GitConnector{Repo: repo, Git: g}).Clone(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var escal []string
	ledger := &effects.Ledger{DB: e.db.Pool, Lease: time.Minute, MaxAttempts: 3,
		Escalate: func(_ context.Context, _ persistence.OutboxEntry, evidence string) error {
			escal = append(escal, evidence)
			return nil
		}}
	feat := "f1a2b3c4"
	branch, _ := gitops.FeatureBranch(feat, 1)
	return ghEnv{remote: remote, proj: p.ID, feat: feat, branch: branch, wd: wd, pub: &gitops.Publisher{Work: wd, Ledger: ledger}, pushes: &pushes, mu: &mu, e: e, escal: &escal}
}

func ghSnap(files map[string]string) checkpoints.Snapshot {
	m := checkpoints.Manifest{Version: 1}
	objs := map[string][]byte{}
	for p, body := range files {
		h := sha256.Sum256([]byte(body))
		hash := hex.EncodeToString(h[:])
		m.Files = append(m.Files, checkpoints.FileEntry{Path: p, Kind: checkpoints.KindFile, Mode: 0o644, Size: int64(len(body)), SHA256: hash})
		objs[hash] = []byte(body)
		m.TotalBytes += int64(len(body))
	}
	for i := 1; i < len(m.Files); i++ {
		for j := i; j > 0 && m.Files[j].Path < m.Files[j-1].Path; j-- {
			m.Files[j], m.Files[j-1] = m.Files[j-1], m.Files[j]
		}
	}
	m.TreeHash = checkpoints.TreeHash(m.Files)
	return checkpoints.Snapshot{Manifest: m, Objects: objs}
}

func (g ghEnv) commit(t *testing.T, parent, key string, files map[string]string) gitops.CommitInfo {
	t.Helper()
	h := sha256.Sum256([]byte("relatorio-" + key))
	info, err := g.wd.Commit(context.Background(), gitops.CommitRequest{
		Branch: g.branch, Parent: parent, Task: gitops.Task{Key: key, Title: "Tarefa " + key},
		Checks:   gitops.Checks{Passed: true, Names: []string{"lint", "test"}, ReportHash: hex.EncodeToString(h[:])},
		Snapshot: ghSnap(files), At: time.Unix(1_700_000_000, 0)})
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func (g ghEnv) req(sha string) gitops.PushRequest {
	return gitops.PushRequest{ProjectID: g.proj, FeatureID: g.feat, Attempt: 1, Branch: g.branch, SHA: sha}
}

func (g ghEnv) pushCount() int { g.mu.Lock(); defer g.mu.Unlock(); return *g.pushes }

func (g ghEnv) outboxRows(t *testing.T, key string) (n int, state string) {
	t.Helper()
	ctx := context.Background()
	_ = g.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE project_id = $1 AND business_key = $2`, g.proj, key).Scan(&n)
	_ = g.e.db.Pool.QueryRow(ctx, `SELECT COALESCE(max(state), '') FROM outbox WHERE project_id = $1 AND business_key = $2`, g.proj, key).Scan(&state)
	return
}

// T15-01 — branch por feature, commit por tarefa validada e push confirmado por ref.
func TestFeatureBranchIsPublishedAndConfirmedByRef(t *testing.T) {
	g := newGHEnv(t, nil)
	ctx := context.Background()
	baseUpdates := g.remote.updates("refs/heads/main")
	info := g.commit(t, g.remote.main, "PROJ-12", map[string]string{"README.md": "base\n", "feature.go": "package x\n"})
	res, err := g.pub.Push(ctx, g.req(info.SHA))
	if err != nil || res.Outcome != effects.OutcomeApplied {
		t.Fatalf("%+v %v", res, err)
	}
	if got := g.remote.rev(t, "refs/heads/"+g.branch); got != info.SHA {
		t.Fatalf("a ref remota deve apontar para o commit: %q", got)
	}
	proof, ok := gitops.ProofOf(res)
	if !ok || proof.SHA != info.SHA || proof.Ref != "refs/heads/"+g.branch {
		t.Fatalf("prova no ledger: %+v %v", proof, ok)
	}
	// O commit publicado corresponde à tarefa e aos checks.
	tr, err := g.wd.ReadTrailers(ctx, info.SHA)
	if err != nil || tr.Task != "PROJ-12" || tr.Checks != "lint,test" {
		t.Fatalf("%+v %v", tr, err)
	}
	// A base não foi tocada.
	if g.remote.rev(t, "refs/heads/main") != g.remote.main || g.remote.updates("refs/heads/main") != baseUpdates {
		t.Fatal("a branch base não pode mudar")
	}
	// Segunda tarefa na mesma branch: avança a ref (fast-forward) com outra chave.
	next := g.commit(t, info.SHA, "PROJ-13", map[string]string{"README.md": "base\n", "feature.go": "package x\n// 2\n"})
	if res, err := g.pub.Push(ctx, g.req(next.SHA)); err != nil || res.Outcome != effects.OutcomeApplied || g.remote.rev(t, "refs/heads/"+g.branch) != next.SHA {
		t.Fatalf("%+v %v", res, err)
	}
	if g.remote.updates("refs/heads/"+g.branch) != 2 || g.remote.rev(t, "refs/heads/main") != g.remote.main {
		t.Fatal("uma atualização por tarefa e a base intacta")
	}
}

// T15-01 — "Simular timeout de push e prova por ref sem duplicidade ou alteração direta da base."
func TestPushTimeoutIsSettledByRefProofWithoutDuplicates(t *testing.T) {
	// A resposta se perde DEPOIS de o remoto ter recebido o push.
	g := newGHEnv(t, func(run func() error, n int) error {
		if err := run(); err != nil {
			return err
		}
		return gitops.ErrTimeout // o remoto aplicou; o cliente só viu o tempo esgotar
	})
	ctx := context.Background()
	info := g.commit(t, g.remote.main, "PROJ-20", map[string]string{"README.md": "base\n", "a.go": "package a\n"})
	res, err := g.pub.Push(ctx, g.req(info.SHA))
	if err != nil || res.Outcome != effects.OutcomeReconciled {
		t.Fatalf("a prova pela ref reconcilia: %+v %v", res, err)
	}
	if g.pushCount() != 1 || g.remote.updates("refs/heads/"+g.branch) != 1 || g.remote.rev(t, "refs/heads/"+g.branch) != info.SHA {
		t.Fatalf("um único push: %d execuções, %d atualizações", g.pushCount(), g.remote.updates("refs/heads/"+g.branch))
	}
	// Repetir o mesmo efeito não reenvia nada.
	again, err := g.pub.Push(ctx, g.req(info.SHA))
	if err != nil || again.Outcome != effects.OutcomeAlreadyConfirmed || g.pushCount() != 1 {
		t.Fatalf("%+v %v pushes=%d", again, err, g.pushCount())
	}
	if n, st := g.outboxRows(t, g.req(info.SHA).BusinessKey()); n != 1 || st != "confirmed" {
		t.Fatalf("um único efeito: %d %s", n, st)
	}
	// A base nunca foi alvo.
	if g.remote.rev(t, "refs/heads/main") != g.remote.main || g.remote.updates("refs/heads/main") > 1 {
		t.Fatal("alteração direta da base")
	}
}

func TestPushTimeoutBeforeTheRemoteReceivedItIsRetriedSafely(t *testing.T) {
	// A primeira tentativa morre antes de chegar ao remoto; a prova mostra a ref ausente e a repetição vale.
	g := newGHEnv(t, func(run func() error, n int) error {
		if n == 1 {
			return gitops.ErrTimeout
		}
		return run()
	})
	info := g.commit(t, g.remote.main, "PROJ-21", map[string]string{"README.md": "base\n", "b.go": "package b\n"})
	res, err := g.pub.Push(context.Background(), g.req(info.SHA))
	if err != nil || res.Outcome != effects.OutcomeApplied {
		t.Fatalf("%+v %v", res, err)
	}
	if g.pushCount() != 2 || g.remote.updates("refs/heads/"+g.branch) != 1 || g.remote.rev(t, "refs/heads/"+g.branch) != info.SHA {
		t.Fatalf("duas execuções, uma atualização no remoto: %d / %d", g.pushCount(), g.remote.updates("refs/heads/"+g.branch))
	}
}

func TestPushIsRestrictedToTheFeatureBranchAndNeverTouchesTheBase(t *testing.T) {
	g := newGHEnv(t, nil)
	ctx := context.Background()
	info := g.commit(t, g.remote.main, "PROJ-30", map[string]string{"README.md": "base\n", "c.go": "package c\n"})
	for name, mut := range map[string]func(*gitops.PushRequest){
		"base":                func(r *gitops.PushRequest) { r.Branch = "main" },
		"outra feature":       func(r *gitops.PushRequest) { r.Branch = "lf/f-outra-a1" },
		"force":               func(r *gitops.PushRequest) { r.Branch = "+" + g.branch },
		"refspec para a base": func(r *gitops.PushRequest) { r.Branch = g.branch + ":refs/heads/main" },
		"sha inválido":        func(r *gitops.PushRequest) { r.SHA = "--delete" },
	} {
		req := g.req(info.SHA)
		mut(&req)
		if _, err := g.pub.Push(ctx, req); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	if g.pushCount() != 0 {
		t.Fatalf("nenhum git push pode ter rodado: %d", g.pushCount())
	}
	var rows int
	_ = g.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE project_id = $1`, g.proj).Scan(&rows)
	if rows != 0 {
		t.Fatalf("pedido recusado nem vira intenção: %d", rows)
	}
	if g.remote.rev(t, "refs/heads/main") != g.remote.main || g.remote.rev(t, "refs/heads/"+g.branch) != "" {
		t.Fatal("o remoto não pode mudar")
	}
}

func TestDivergedRemoteBlocksWithEvidenceAndKeepsTheForeignCommit(t *testing.T) {
	// Sem resposta do push (timeout antes de chegar) e a ref do remoto em outro SHA.
	lost := func(run func() error, n int) error { return gitops.ErrTimeout }
	g := newGHEnv(t, lost)
	ctx := context.Background()
	mine := g.commit(t, g.remote.main, "PROJ-40", map[string]string{"README.md": "base\n", "d.go": "package d\n"})

	// Alguém publicou OUTRO commit na branch da feature, fora da fábrica.
	work := t.TempDir()
	ghGit(t, work, "init", "-b", "main")
	_ = os.WriteFile(filepath.Join(work, "x.txt"), []byte("de fora\n"), 0o644)
	ghGit(t, work, "add", "-A")
	ghGit(t, work, "commit", "-m", "de fora")
	ghGit(t, work, "push", g.remote.bare, "HEAD:refs/heads/"+g.branch)
	foreign := g.remote.rev(t, "refs/heads/"+g.branch)

	_, err := g.pub.Push(ctx, g.req(mine.SHA))
	if !errors.Is(err, effects.ErrBlocked) || len(*g.escal) != 1 || !strings.Contains((*g.escal)[0], "divergiu") {
		t.Fatalf("divergência bloqueia com evidência: %v %v", err, *g.escal)
	}
	if g.remote.rev(t, "refs/heads/"+g.branch) != foreign {
		t.Fatal("a fábrica não pode sobrescrever o que não é dela")
	}
	if _, err := g.pub.Push(ctx, g.req(mine.SHA)); !errors.Is(err, effects.ErrBlocked) {
		t.Fatalf("a repetição automática segue bloqueada: %v", err)
	}
	if n, st := g.outboxRows(t, g.req(mine.SHA).BusinessKey()); n != 1 || st != "uncertain" {
		t.Fatalf("%d %s", n, st)
	}
}

// O push chegou e, antes da verificação, alguém empilhou um commit SOBRE o nosso:
// a prova pela ref continua confirmando (o nosso commit é ancestral da ponta).
func TestRefAdvancedAfterOurPushIsStillConfirmed(t *testing.T) {
	var g ghEnv
	g = newGHEnv(t, func(run func() error, n int) error {
		if err := run(); err != nil {
			return err
		}
		work := t.TempDir()
		ghGit(t, work, "init", "-b", "main")
		ghGit(t, work, "fetch", g.remote.bare, g.branch)
		ghGit(t, work, "checkout", "-b", "tmp", "FETCH_HEAD")
		ghGit(t, work, "commit", "--allow-empty", "-m", "empilhado por fora")
		ghGit(t, work, "push", g.remote.bare, "HEAD:refs/heads/"+g.branch)
		return gitops.ErrTimeout
	})
	info := g.commit(t, g.remote.main, "PROJ-41", map[string]string{"README.md": "base\n", "e.go": "package e\n"})
	res, err := g.pub.Push(context.Background(), g.req(info.SHA))
	if err != nil || res.Outcome != effects.OutcomeReconciled {
		t.Fatalf("%+v %v", res, err)
	}
	proof, ok := gitops.ProofOf(res)
	tip := g.remote.rev(t, "refs/heads/"+g.branch)
	if !ok || proof.SHA != tip || tip == info.SHA || !strings.Contains(proof.Note, "avançou") {
		t.Fatalf("a prova registra a ponta atual: %+v tip=%s ours=%s", proof, tip, info.SHA)
	}
	if g.pushCount() != 1 {
		t.Fatalf("sem reenvio: %d", g.pushCount())
	}
}

func TestChangesUnderGithubRequireExplicitApproval(t *testing.T) {
	g := newGHEnv(t, nil)
	ctx := context.Background()
	info := g.commit(t, g.remote.main, "PROJ-50", map[string]string{"README.md": "base\n", ".github/workflows/ci.yml": "on: push\n"})
	if len(info.Sensitive) != 1 {
		t.Fatalf("%+v", info)
	}
	if _, err := g.pub.Push(ctx, g.req(info.SHA)); !errors.Is(err, gitops.ErrWorkflowChange) || g.pushCount() != 0 {
		t.Fatalf("%v pushes=%d", err, g.pushCount())
	}
	approved := g.req(info.SHA)
	approved.AllowWorkflows = true
	if res, err := g.pub.Push(ctx, approved); err != nil || res.Outcome != effects.OutcomeApplied {
		t.Fatalf("com aprovação: %+v %v", res, err)
	}
}
