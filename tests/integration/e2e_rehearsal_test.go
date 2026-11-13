package integration

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/delivery"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/memory"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
)

// T17-02 (ensaio de software) — percorre a feature até depois do merge humano nos dobles
// (GitHub e origem simulados, PostgreSQL real): entrega, correção externa, merge observado,
// liberação do dependente, conclusão na origem e promoção da memória, tudo ligado à mesma
// revisão. NÃO substitui a execução no ambiente e no projeto reais (NOT_RUN).
func TestFeatureRehearsalFromDraftPRToPostMergeCycleLinkedToTheIntegratedRevision(t *testing.T) {
	m := newExtEnv(t, "e2e-rehearsal")
	ctx := context.Background()
	branch, _ := gitops.FeatureBranch(m.p.feature, 1)
	mem := &memory.Store{DB: m.e.db.Pool, Redactor: m.x.Redactor}

	// 1. Commits por tarefa, memória técnica pendente e achado de revisão corrigido.
	if err := m.d.RecordCommit(ctx, m.p.proj, m.p.feature, "schema", shaA, branch, "schema: cria a tabela de pedidos"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := mem.Record(ctx, memInput(m.p, branch, "pedidos usam soft delete", "a tabela de pedidos usa a coluna removido_em", shaB)); err != nil {
		t.Fatal(err)
	}

	// 2. A Draft PR já existe (newMrgEnv entregou); um revisor autorizado pede uma correção.
	pr := m.out.PR.Number
	if m.out.PR == nil || !m.out.PR.Draft {
		t.Fatalf("a PR precisa ser rascunho: %+v", m.out.PR)
	}
	m.gh.addComment("issues", pr, 501, "Revisora", "documente a coluna removido_em")
	d, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || d.Item == nil || d.Item.Kind != queue.KindPRFix || d.New != 1 {
		t.Fatalf("%+v %v", d, err)
	}
	// A correção usa a próxima vaga (à frente da tarefa) e conclui.
	if _, err := m.q.Start(ctx, d.Item.ID, m.p.execution(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := m.q.Complete(ctx, d.Item.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := m.x.MarkFixed(ctx, m.out.DeliveryID, d.Item.ID, delivery.FixProof{DocsUpdated: true, ChecksPassed: true}); err != nil || n != 1 {
		t.Fatalf("%d %v", n, err)
	}

	// 3. Um dependente espera o merge; com a PR aberta nada é liberado, concluído ou promovido.
	dep := m.waitingOnMerge(t, "schema")
	obs, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || obs.Outcome != "open" || len(obs.Released) != 0 || m.org.calls != 0 {
		t.Fatalf("%+v %v", obs, err)
	}
	if _, err := mem.Promote(ctx, m.p.proj, m.out.DeliveryID); err == nil {
		t.Fatal("sem merge não se promove memória")
	}

	// 4. Merge humano: SHA observado, dependente liberado, item concluído na origem, memória promovida.
	const mergeSHA = "1234567890abcdef1234567890abcdef12345678"
	m.gh.setPR(pr, "closed", true, mergeSHA)
	obs, err = m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || obs.Outcome != "merged" || obs.MergeSHA != mergeSHA || len(obs.Released) != 1 || obs.Released[0].ID != dep.ID || obs.Projection != "done" || m.org.done != 1 {
		t.Fatalf("%+v %v", obs, err)
	}
	prom, err := mem.Promote(ctx, m.p.proj, m.out.DeliveryID)
	if err != nil || len(prom.Promoted) != 1 || prom.MergeSHA != mergeSHA {
		t.Fatalf("%+v %v", prom, err)
	}

	// 5. Verificável e ligado à revisão: PR, commits, checks, origem, dependência e memória.
	var prURL, sha, state string
	var commits, checksOK int
	if err := m.e.db.Pool.QueryRow(ctx, `SELECT pr_url, merge_sha, state FROM deliveries WHERE id = $1`, m.out.DeliveryID).Scan(&prURL, &sha, &state); err != nil {
		t.Fatal(err)
	}
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM feature_commits WHERE feature_id = $1 AND branch = $2`, m.p.feature, branch).Scan(&commits)
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM check_runs WHERE feature_id = $1 AND revision = $2 AND status = 'pass'`, m.p.feature, shaB).Scan(&checksOK)
	if state != "merged" || sha != mergeSHA || !strings.HasSuffix(prURL, "/pull/"+strconv.Itoa(pr)) || commits != 1 || checksOK < 1 {
		t.Fatalf("estado=%s sha=%s pr=%s commits=%d checks=%d", state, sha, prURL, commits, checksOK)
	}
	off, _ := mem.Relevant(ctx, m.p.proj, "lf/qualquer-outra-branch", "soft delete", 5)
	if len(off) != 1 || off[0].State != memory.StateOfficial || off[0].PromotedRevision != mergeSHA {
		t.Fatalf("%+v", off)
	}
	items, _ := m.q.List(ctx, m.p.proj)
	for _, it := range items {
		if it.ID == dep.ID && it.State != queue.StateQueued {
			t.Fatalf("o dependente volta à fila: %+v", it)
		}
	}
	// Idempotência do pós-merge: repetir tudo não muda nada.
	again, _ := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	rep, _ := mem.Promote(ctx, m.p.proj, m.out.DeliveryID)
	if m.org.done != 1 || len(again.Released) != 0 || len(rep.Promoted) != 0 {
		t.Fatalf("repetir não muda: done=%d %+v %+v", m.org.done, again, rep)
	}
}
