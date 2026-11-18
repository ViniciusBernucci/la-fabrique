package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/projectsetup"
	"github.com/ViniciusBernucci/le-fabrique/internal/workspace"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

func commitScript(t *testing.T, repo, content string) {
	t.Helper()
	_ = os.MkdirAll(filepath.Join(repo, ".lafabrique"), 0o755)
	if err := os.WriteFile(filepath.Join(repo, projectsetup.ScriptPath), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "setup")
}

// ADR-0062 fase 3 — o setup só roda com aprovação do hash atual e dentro do sandbox.
func TestProjectSetupNeedsApprovalBoundToTheHash(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "setup")
	root := t.TempDir()
	repo := filepath.Join(root, "app")
	_ = os.MkdirAll(repo, 0o755)
	gitIn(t, repo, "init", "-q", "-b", "main")
	_ = os.WriteFile(filepath.Join(repo, "a.txt"), []byte("a\n"), 0o644)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "base")
	if _, err := db.Pool.Exec(ctx, `UPDATE projects SET repo_path = 'app' WHERE id = $1`, p.ID); err != nil {
		t.Fatal(err)
	}
	docker := &execution.Docker{}
	svc := projectsetup.Service{DB: db.Pool, Root: workspace.Root{Dir: root}, Sandbox: docker}

	st, err := svc.Inspect(ctx, p.ID, "main")
	if err != nil || st.Found {
		t.Fatalf("sem script: %+v %v", st, err)
	}
	if _, err := svc.Approve(ctx, p.ID, "", "main", strings.Repeat("a", 64)); !errors.Is(err, projectsetup.ErrNoScript) {
		t.Fatalf("%v", err)
	}

	commitScript(t, repo, "echo ola > feito.txt\ncat a.txt\n")
	st, err = svc.Inspect(ctx, p.ID, "main")
	if err != nil || !st.Found || st.Approved || len(st.SHA256) != 64 || !strings.Contains(st.Content, "ola") {
		t.Fatalf("%+v %v", st, err)
	}
	// Sem aprovação nada roda (nem consulta o Docker: a recusa vem antes de criar o sandbox).
	if _, err := svc.Run(ctx, projectsetup.RunSpec{ProjectID: p.ID, Ref: "main"}); err != nil && !errors.Is(err, projectsetup.ErrNotApproved) && !errors.Is(err, projectsetup.ErrNoSandbox) {
		t.Fatalf("%v", err)
	}
	// Aprovar um hash que o usuário não viu é recusado.
	if _, err := svc.Approve(ctx, p.ID, "", "main", strings.Repeat("0", 64)); !errors.Is(err, projectsetup.ErrChanged) {
		t.Fatalf("%v", err)
	}
	if st, err = svc.Approve(ctx, p.ID, "", "main", st.SHA256); err != nil || !st.Approved || st.ApprovalID == "" {
		t.Fatalf("%+v %v", st, err)
	}
	approved := st

	if docker.Available(ctx) {
		run, err := svc.Run(ctx, projectsetup.RunSpec{ProjectID: p.ID, Ref: "main", Timeout: 2 * time.Minute})
		if err != nil || run.Status != "ok" || !strings.Contains(run.Output, "a") {
			t.Fatalf("%+v %v", run, err)
		}
		if after, _ := svc.Inspect(ctx, p.ID, "main"); after.LastRun == nil || after.LastRun.Status != "ok" {
			t.Fatalf("%+v", after)
		}
		// O setup não escreve no checkout do host.
		if _, err := os.Stat(filepath.Join(repo, "feito.txt")); err == nil {
			t.Fatal("o setup rodou fora do sandbox")
		}
	} else {
		t.Log("Docker indisponível: execução no sandbox não testada")
	}

	// Conteúdo alterado muda o hash: a aprovação anterior não vale.
	commitScript(t, repo, "echo outro\n")
	st, _ = svc.Inspect(ctx, p.ID, "main")
	if st.Approved || st.SHA256 == approved.SHA256 {
		t.Fatalf("a aprovação deve estar presa ao hash: %+v", st)
	}
	if _, err := svc.Run(ctx, projectsetup.RunSpec{ProjectID: p.ID, Ref: "main"}); !errors.Is(err, projectsetup.ErrNotApproved) && !errors.Is(err, projectsetup.ErrNoSandbox) {
		t.Fatalf("%v", err)
	}
	// Revogar a aprovação antiga e inspecionar um commit antigo.
	if err := svc.Revoke(ctx, p.ID, "", approved.ApprovalID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, p.ID, "", approved.ApprovalID); err == nil {
		t.Fatal("revogar duas vezes deve falhar")
	}
	if _, err := svc.Inspect(ctx, p.ID, "--upload-pack=x"); err == nil {
		t.Fatal("ref que parece opção deve ser recusada")
	}
}

// ADR-0062 fase 3 — limpeza: desligada por padrão, simulação antes de apagar, nunca perde trabalho só local.
func TestWorktreeCleanupPolicyAndDryRun(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "clean")
	repo, base := newRepoWithBase(t)
	m := &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir(), MaxPerProject: -1, MaxPerHost: -1}
	f := seedFeature(t, db, p.ID, "K-1")
	prov := func(n int) worktrees.Attempt {
		a, err := m.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: f, Track: "BACKEND", Kind: "implementation", Attempt: n, BaseSHA: base})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	now := time.Now()

	pol, err := m.GetPolicy(ctx, p.ID)
	if err != nil || pol.Enabled {
		t.Fatalf("desligada por padrão: %+v %v", pol, err)
	}
	a := prov(1) // ativa e parada → arquivar
	b := prov(2) // ativa com escritor vivo → pular
	c := prov(3) // arquivada e já incorporada → remover
	d := prov(4) // arquivada com commit só local → pular
	e := prov(5) // arquivada com arquivo não versionado → pular
	_ = m.SetWriter(ctx, b.ID, true)
	_ = os.WriteFile(filepath.Join(d.Path, "so-local.txt"), []byte("x\n"), 0o644)
	gitIn(t, d.Path, "add", "-A")
	gitIn(t, d.Path, "commit", "-q", "-m", "so local")
	_ = os.WriteFile(filepath.Join(e.Path, "rascunho.txt"), []byte("y\n"), 0o644)
	for _, x := range []worktrees.Attempt{c, d, e} {
		if err := m.Release(ctx, x.ID); err != nil {
			t.Fatal(err)
		}
	}
	old := now.Add(-72 * time.Hour)
	if _, err := db.Pool.Exec(ctx, `UPDATE worktree_attempts SET created_at = $2 WHERE id = $1`, a.ID, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE worktree_attempts SET released_at = $2 WHERE id::text = ANY($1)`, []string{c.ID, d.ID, e.ID}, old); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE worktree_attempts SET created_at = $2 WHERE project_id = $1 AND id <> $3`, p.ID, old, b.ID); err != nil {
		t.Fatal(err)
	}

	// Desligada: simulação vazia e aplicar é recusado.
	plan, err := m.PlanCleanup(ctx, p.ID, base, now)
	if err != nil || len(plan.Items) != 0 {
		t.Fatalf("%+v %v", plan, err)
	}
	if _, err := m.ApplyCleanup(ctx, p.ID, "", base, plan.Hash, now); !errors.Is(err, worktrees.ErrCleanupOff) {
		t.Fatalf("%v", err)
	}
	if _, err := m.SetPolicy(ctx, p.ID, "", worktrees.Policy{Enabled: true, ArchiveIdleDays: 0, RemoveArchivedDays: 1}); !errors.Is(err, worktrees.ErrInvalid) {
		t.Fatalf("%v", err)
	}
	if _, err := m.SetPolicy(ctx, p.ID, "", worktrees.Policy{Enabled: true, ArchiveIdleDays: 1, RemoveArchivedDays: 1}); err != nil {
		t.Fatal(err)
	}

	plan, err = m.PlanCleanup(ctx, p.ID, base, now)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, it := range plan.Items {
		got[it.AttemptID] = it.Action
	}
	want := map[string]string{a.ID: "archive", b.ID: "skip", c.ID: "remove", d.ID: "skip", e.ID: "skip"}
	for id, w := range want {
		if got[id] != w {
			t.Fatalf("%s: esperava %s, veio %s\n%+v", id, w, got[id], plan.Items)
		}
	}
	// A simulação não alterou nada.
	if _, err := os.Stat(a.Path); err != nil {
		t.Fatal("a simulação não pode tirar a pasta")
	}
	// Sem base conhecida nada é removido.
	if np, _ := m.PlanCleanup(ctx, p.ID, "", now); func() bool {
		for _, it := range np.Items {
			if it.Action == "remove" {
				return true
			}
		}
		return false
	}() {
		t.Fatal("sem branch base não há remoção")
	}

	if _, err := m.ApplyCleanup(ctx, p.ID, "", base, strings.Repeat("0", 64), now); !errors.Is(err, worktrees.ErrPlanChanged) {
		t.Fatalf("hash diferente: %v", err)
	}
	res, err := m.ApplyCleanup(ctx, p.ID, "", base, plan.Hash, now)
	if err != nil || res.Archived != 1 || res.Removed != 1 || res.Failed != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := os.Stat(a.Path); !os.IsNotExist(err) {
		t.Fatal("a tentativa parada deveria ter sido arquivada")
	}
	if _, err := os.Stat(b.Path); err != nil {
		t.Fatal("tentativa com escritor vivo não pode ser tocada")
	}
	archived, _ := m.Get(ctx, a.ID)
	if archived.State != "released" || archived.ArchiveRef == "" || gitIn(t, repo, "rev-parse", "--verify", archived.ArchiveRef) == "" {
		t.Fatalf("arquivar preserva o trabalho: %+v", archived)
	}
	removed, _ := m.Get(ctx, c.ID)
	if removed.ArchiveRef != "" {
		t.Fatalf("a incorporada tem a ref apagada: %+v", removed)
	}
	for _, x := range []worktrees.Attempt{d, e} {
		kept, _ := m.Get(ctx, x.ID)
		if kept.ArchiveRef == "" || gitIn(t, repo, "rev-parse", "--verify", kept.ArchiveRef) == "" {
			t.Fatalf("trabalho só local deve permanecer: %+v", kept)
		}
	}
	var audited int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM project_changes WHERE project_id = $1 AND action IN ('cleanup_policy_changed','cleanup_applied')`, p.ID).Scan(&audited)
	if audited != 2 {
		t.Fatalf("auditoria: %d", audited)
	}
}

// ADR-0062 fase 3 — rotas de setup e limpeza (sessão, CSRF e hash).
func TestProjectSetupAndCleanupRoutes(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "legado")
	_ = os.MkdirAll(repo, 0o755)
	gitIn(t, repo, "init", "-q", "-b", "main")
	gitIn(t, repo, "remote", "add", "origin", "https://github.com/acme/legado.git")
	commitScript(t, repo, "echo preparar\n")

	e := newAuthEnv(t, func(c *config.API) { c.WorkspacesDir = root })
	c := e.newClient()
	anon := e.newClient()
	c.login(nil)
	p := createProject(t, c, "app-rotas")
	body := map[string]any{"mode": "local", "path": "legado", "expected_version": p.Version}
	c.expect(c.post("/api/projects/"+p.ID+"/repository/prepare", body), 200)

	anon.expect(anon.get("/api/projects/"+p.ID+"/setup"), 401)

	var st projectsetup.State
	c.expect(c.get("/api/projects/"+p.ID+"/setup"), 200).JSON(&st)
	if !st.Found || st.Approved || st.Ref != "main" {
		t.Fatalf("%+v", st)
	}
	c.expect(c.post("/api/projects/"+p.ID+"/setup/approve", map[string]any{"ref": "main", "sha256": strings.Repeat("1", 64)}), 409)
	c.expect(c.post("/api/projects/"+p.ID+"/setup/approve", map[string]any{"ref": "main"}), 400)
	c.expect(c.post("/api/projects/"+p.ID+"/setup/approve", map[string]any{"ref": "main", "sha256": st.SHA256}), 200).JSON(&st)
	if !st.Approved {
		t.Fatalf("%+v", st)
	}
	c.expect(c.get("/api/projects/"+p.ID+"/setup?ref=nao-existe"), 422)
	c.expect(c.delete("/api/projects/"+p.ID+"/setup/approvals/"+st.ApprovalID), 204)
	c.expect(c.get("/api/projects/"+p.ID+"/setup"), 200).JSON(&st)
	if st.Approved {
		t.Fatal("aprovação revogada")
	}

	// Limpeza: desligada por padrão; aplicar sem política é 409; hash errado é 409.
	var plan worktrees.Plan
	c.expect(c.get("/api/projects/"+p.ID+"/cleanup"), 200).JSON(&plan)
	if plan.Policy.Enabled || len(plan.Hash) != 64 {
		t.Fatalf("%+v", plan)
	}
	c.expect(c.post("/api/projects/"+p.ID+"/cleanup/apply", map[string]any{"hash": plan.Hash}), 409)
	c.expect(c.do("PUT", "/api/projects/"+p.ID+"/cleanup/policy", map[string]any{"enabled": true, "archive_idle_days": 0, "remove_archived_days": 5}, nil), 422)
	c.expect(c.do("PUT", "/api/projects/"+p.ID+"/cleanup/policy", map[string]any{"enabled": true, "archive_idle_days": 3, "remove_archived_days": 30}, nil), 200)
	c.expect(c.post("/api/projects/"+p.ID+"/cleanup/apply", map[string]any{"hash": strings.Repeat("2", 64)}), 409)
	c.expect(c.get("/api/projects/"+p.ID+"/cleanup"), 200).JSON(&plan)
	if !plan.Policy.Enabled || plan.Policy.ArchiveIdleDays != 3 {
		t.Fatalf("%+v", plan)
	}
	c.expect(c.post("/api/projects/"+p.ID+"/cleanup/apply", map[string]any{"hash": plan.Hash}), 200)
}
