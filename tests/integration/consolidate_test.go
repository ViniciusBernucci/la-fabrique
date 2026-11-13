package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/consolidate"
)

func commitOnBranch(t *testing.T, repo, branch, base, file, content string) {
	t.Helper()
	gitIn(t, repo, "checkout", "-q", "-B", branch, base)
	_ = os.MkdirAll(filepath.Dir(filepath.Join(repo, file)), 0o755)
	if err := os.WriteFile(filepath.Join(repo, file), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "mudança em "+file)
	gitIn(t, repo, "checkout", "-q", "main")
}

func consolidationEnv(t *testing.T) (testDB, string, string, string, *consolidate.Consolidator) {
	t.Helper()
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "cons")
	var feature string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'C-1','f') RETURNING id::text`, p.ID).Scan(&feature); err != nil {
		t.Fatal(err)
	}
	repo, base := newRepoWithBase(t)
	return db, p.ID, feature, base, &consolidate.Consolidator{DB: db.Pool, Repo: repo, Root: t.TempDir()}
}

// AJ06-03 / CT06, CT07 — integra as trilhas em ordem, registra equivalência e cabeça candidata,
// repetir (inclusive sem o registro no banco) não duplica commits.
func TestConsolidationIntegratesInOrderAndIsIdempotent(t *testing.T) {
	db, project, feature, base, c := consolidationEnv(t)
	ctx := context.Background()
	commitOnBranch(t, c.Repo, "lf-tmp/f/frontend-impl-1", base, "apps/web/new.ts", "front\n")
	commitOnBranch(t, c.Repo, "lf-tmp/f/backend-impl-1", base, "internal/api/new.go", "package api\n")
	req := consolidate.Request{ProjectID: project, FeatureID: feature, DeliveryBranch: "feature/c-1", StartSHA: base, Attempts: []consolidate.Attempt{
		{Ref: "front", Branch: "lf-tmp/f/frontend-impl-1", BaseSHA: base}, {Ref: "back", Branch: "lf-tmp/f/backend-impl-1", BaseSHA: base}}}
	res, err := c.Run(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Commits) != 2 || res.Commits[0].Changed || !res.Commits[1].Changed || len(res.Revalidate) != 1 || res.Revalidate[0] != "back" {
		t.Fatalf("o primeiro é fast-forward (SHA igual) e o segundo muda de SHA e exige revalidação: %+v", res)
	}
	if got := gitIn(t, c.Repo, "rev-parse", "refs/heads/feature/c-1"); got != res.Head {
		t.Fatalf("a branch de entrega deve apontar para a cabeça candidata: %s %s", got, res.Head)
	}
	if head, rv, err := c.Candidate(ctx, feature); err != nil || head != res.Head || len(rv) != 1 {
		t.Fatalf("%s %v %v", head, rv, err)
	}
	count := func() string { return gitIn(t, c.Repo, "rev-list", "--count", "refs/heads/feature/c-1") }
	before := count()
	res2, err := c.Run(ctx, req)
	if err != nil || res2.Head != res.Head || count() != before {
		t.Fatalf("repetir não muda a cabeça nem duplica: %+v %v", res2, err)
	}
	// Queda entre a ref e o registro: sem as linhas de equivalência, o histórico ainda prova a integração.
	_, _ = db.Pool.Exec(ctx, `DELETE FROM consolidated_commits`)
	res3, err := c.Run(ctx, req)
	if err != nil || res3.Head != res.Head || count() != before || len(res3.Commits) != 2 {
		t.Fatalf("%+v %v", res3, err)
	}
	if gitIn(t, c.Repo, "log", "-1", "--format=%B", res.Commits[1].Integrated); !strings.Contains(gitIn(t, c.Repo, "log", "-1", "--format=%B", res.Commits[1].Integrated), "cherry picked from commit "+res.Commits[1].Original) {
		t.Fatal("o commit integrado deve guardar a origem")
	}
}

// AJ06-03 — conflito preserva os dois lados e não toca a branch de entrega.
func TestConsolidationConflictPreservesBothSidesAndBlocks(t *testing.T) {
	db, project, feature, base, c := consolidationEnv(t)
	ctx := context.Background()
	commitOnBranch(t, c.Repo, "lf-tmp/f/frontend-impl-1", base, "internal/api/b.go", "package front\n")
	commitOnBranch(t, c.Repo, "lf-tmp/f/backend-impl-1", base, "internal/api/b.go", "package back\n")
	res, err := c.Run(ctx, consolidate.Request{ProjectID: project, FeatureID: feature, DeliveryBranch: "feature/c-1", StartSHA: base, Attempts: []consolidate.Attempt{
		{Ref: "front", Branch: "lf-tmp/f/frontend-impl-1", BaseSHA: base}, {Ref: "back", Branch: "lf-tmp/f/backend-impl-1", BaseSHA: base}}})
	if !errors.Is(err, consolidate.ErrConflict) || res.Conflict == nil || res.Conflict.Attempt != "back" || len(res.Conflict.Files) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	// A branch de entrega não recebeu nada do que conflitou (nem o lado que integrou limpo antes), e as duas tentativas seguem intactas.
	if got := gitIn(t, c.Repo, "rev-parse", "refs/heads/feature/c-1"); got != base {
		t.Fatalf("a publicação fica bloqueada: a entrega permanece na SHA de partida: %s", got)
	}
	for _, b := range []string{"lf-tmp/f/frontend-impl-1", "lf-tmp/f/backend-impl-1"} {
		if gitIn(t, c.Repo, "rev-parse", "--verify", "refs/heads/"+b) == "" {
			t.Fatal("tentativa perdida")
		}
	}
	var state string
	var n int
	_ = db.Pool.QueryRow(ctx, `SELECT state FROM consolidation_runs WHERE id::text=$1`, res.RunID).Scan(&state)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM consolidation_conflicts WHERE run_id::text=$1`, res.RunID).Scan(&n)
	if state != "conflict" || n != 1 {
		t.Fatalf("%s %d", state, n)
	}
	if _, _, err := c.Candidate(ctx, feature); err == nil {
		t.Fatal("conflito não gera cabeça candidata")
	}
}

// AJ06-03 — alteração concorrente da branch impede a atualização obsoleta; consolidação simultânea é recusada.
func TestConsolidationRefusesStaleUpdateAndConcurrentRuns(t *testing.T) {
	db, project, feature, base, c := consolidationEnv(t)
	ctx := context.Background()
	commitOnBranch(t, c.Repo, "lf-tmp/f/backend-impl-1", base, "internal/api/new.go", "package api\n")
	gitIn(t, c.Repo, "branch", "feature/c-1", base)
	req := consolidate.Request{ProjectID: project, FeatureID: feature, DeliveryBranch: "feature/c-1", Attempts: []consolidate.Attempt{{Ref: "back", Branch: "lf-tmp/f/backend-impl-1", BaseSHA: base}}}
	c.OnBeforeUpdate = func() { // alguém (um humano) empurra para a branch no meio da consolidação
		commitOnBranch(t, c.Repo, "feature/c-1", base, "README.md", "humano\n")
	}
	if _, err := c.Run(ctx, req); !errors.Is(err, consolidate.ErrStale) {
		t.Fatalf("%v", err)
	}
	if got := gitIn(t, c.Repo, "show", "feature/c-1:README.md"); got != "humano" {
		t.Fatalf("a alteração humana não pode ser descartada: %q", got)
	}
	c.OnBeforeUpdate = nil
	// Corrida entre duas consolidações da mesma feature.
	_, _ = db.Pool.Exec(ctx, `INSERT INTO consolidation_runs (project_id, feature_id, delivery_branch, lease_until) VALUES ($1,$2,'feature/c-1', now() + interval '10 minutes')`, project, feature)
	if _, err := c.Run(ctx, req); !errors.Is(err, consolidate.ErrBusy) {
		t.Fatalf("%v", err)
	}
	// Lease vencido (queda do consolidador anterior): assume.
	_, _ = db.Pool.Exec(ctx, `UPDATE consolidation_runs SET lease_until = now() - interval '1 minute' WHERE state='running'`)
	if _, err := c.Run(ctx, req); err != nil {
		t.Fatalf("lease vencido deve liberar: %v", err)
	}
}
