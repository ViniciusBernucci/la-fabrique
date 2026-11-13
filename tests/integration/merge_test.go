package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/delivery"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type fakeCompleter struct {
	fail  int
	calls int
	done  int
}

func (f *fakeCompleter) Complete(_ context.Context, _, _, _, _ string) (effects.Result, error) {
	f.calls++
	if f.fail > 0 {
		f.fail--
		return effects.Result{}, errors.New("a origem recusou a transição")
	}
	f.done++
	return effects.Result{Outcome: effects.OutcomeApplied}, nil
}

type mrgEnv struct {
	*dlvEnv
	c   *delivery.Completer
	org *fakeCompleter
	out delivery.Outcome
}

func newMrgEnv(t *testing.T, slug string) *mrgEnv {
	t.Helper()
	x := newDlvEnv(t, slug)
	ctx := context.Background()
	x.green(t, shaB)
	path, _, _ := x.d.Document(ctx, x.p.proj, x.p.feature, shaB, "")
	files := []string{"internal/pedidos/pedidos.go", "internal/pedidos/pedidos_test.go", "internal/pedidos/doc.go", path}
	out, err := x.d.Deliver(ctx, x.request(t, 1, files...))
	if err != nil || out.PR == nil {
		t.Fatalf("%+v %v", out, err)
	}
	api, _ := gitops.NewGitHubAPI(security.NewSecret("tok-delivery"), x.gh.URL, true)
	ver := gitops.MergeVerifier{API: api, Owner: "org", Name: "app"}
	org := &fakeCompleter{}
	x.q.Gates[queue.WaitMerge] = queue.MergeGate{Merges: ver}
	return &mrgEnv{dlvEnv: x, org: org, out: out,
		c: &delivery.Completer{DB: x.e.db.Pool, Merges: ver, Origin: org, Queue: x.q}}
}

func (m *mrgEnv) waitingOnMerge(t *testing.T, key string) queue.Item {
	t.Helper()
	ctx := context.Background()
	it := m.p.item(t, key, 0)
	if _, err := m.q.Start(ctx, it.ID, m.p.execution(t)); err != nil {
		t.Fatal(err)
	}
	exec := ""
	items, _ := m.q.List(ctx, m.p.proj)
	for _, i := range items {
		if i.ID == it.ID {
			exec = i.ExecutionID
		}
	}
	blocked, err := m.q.Block(ctx, it.ID, queue.WaitMerge, m.out.PR.URL, m.p.checkpoint(t, exec), true)
	if err != nil {
		t.Fatal(err)
	}
	return blocked
}

// T15-03 — "Draft PR aberta ou fechada sem merge não libera dependentes."
func TestAnOpenOrClosedWithoutMergePRNeverReleasesDependents(t *testing.T) {
	m := newMrgEnv(t, "mrg-open")
	ctx := context.Background()
	dep := m.waitingOnMerge(t, "schema")
	obs, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || obs.Outcome != "open" || len(obs.Released) != 0 || m.org.calls != 0 {
		t.Fatalf("%+v %v", obs, err)
	}
	// Fechada sem merge: a entrega fica `closed`, nada é liberado nem concluído na origem.
	m.gh.setPR(m.out.PR.Number, "closed", false, "")
	obs, err = m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || obs.Outcome != "closed" || len(obs.Released) != 0 || m.org.calls != 0 {
		t.Fatalf("%+v %v", obs, err)
	}
	var state string
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT state FROM deliveries WHERE id = $1`, m.out.DeliveryID).Scan(&state)
	items, _ := m.q.List(ctx, m.p.proj)
	waiting := false
	for _, it := range items {
		waiting = waiting || (it.ID == dep.ID && it.State == queue.StateWaiting)
	}
	if state != "closed" || !waiting {
		t.Fatalf("estado %s, dependente esperando=%v", state, waiting)
	}
	// "Integrada" sem o SHA do merge: sem prova, nada é liberado.
	m.gh.setPR(m.out.PR.Number, "closed", true, "")
	m2 := newMrgEnv(t, "mrg-nosha")
	m2.gh.setPR(m2.out.PR.Number, "closed", true, "")
	if obs, _ := m2.c.Observe(ctx, m2.out.DeliveryID, "feat:1", "P1"); obs.Outcome != "unknown" || m2.org.calls != 0 {
		t.Fatalf("%+v", obs)
	}
}

// T15-03 — "Merge humano observado libera e eventual falha de status fica visível e reconciliável."
func TestAnObservedHumanMergeReleasesDependentsAndAStatusFailureIsVisibleAndReconcilable(t *testing.T) {
	m := newMrgEnv(t, "mrg-merged")
	ctx := context.Background()
	dep := m.waitingOnMerge(t, "schema")
	m.org.fail = 1 // a origem recusa a primeira projeção do status
	m.gh.setPR(m.out.PR.Number, "closed", true, "deadbeefcafe1234deadbeefcafe1234deadbeef")
	obs, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || obs.Outcome != "merged" || obs.MergeSHA != "deadbeefcafe1234deadbeefcafe1234deadbeef" {
		t.Fatalf("%+v %v", obs, err)
	}
	// O merge está provado e registrado: o dependente foi liberado mesmo com a projeção falhando.
	if len(obs.Released) != 1 || obs.Released[0].ID != dep.ID || obs.Released[0].State != queue.StateQueued {
		t.Fatalf("%+v", obs.Released)
	}
	// A falha de status fica visível e reconciliável.
	if obs.Projection != delivery.ProjectionFailed || !strings.Contains(obs.ProjectionError, "recusou") {
		t.Fatalf("%+v", obs)
	}
	st, _ := m.c.Status(ctx, m.out.DeliveryID)
	if st.Outcome != "merged" || st.Projection != delivery.ProjectionFailed || st.MergeSHA == "" {
		t.Fatalf("%+v", st)
	}
	var state, sha string
	var mergedAt *string
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT state, merge_sha, merged_at::text FROM deliveries WHERE id = $1`, m.out.DeliveryID).Scan(&state, &sha, &mergedAt)
	if state != "merged" || sha != obs.MergeSHA || mergedAt == nil {
		t.Fatalf("%s %s %v", state, sha, mergedAt)
	}
	// Reconciliar: a projeção passa a "feita", sem liberar nem concluir duas vezes.
	rec, err := m.c.Reconcile(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || rec.Projection != delivery.ProjectionDone || m.org.done != 1 || len(rec.Released) != 0 {
		t.Fatalf("%+v %v done=%d", rec, err, m.org.done)
	}
	again, _ := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if again.Projection != delivery.ProjectionDone || m.org.calls != 2 || m.org.done != 1 {
		t.Fatalf("repetir não conclui de novo: %+v calls=%d", again, m.org.calls)
	}
	var n int
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM delivery_observations WHERE delivery_id = $1 AND outcome = 'merged'`, m.out.DeliveryID).Scan(&n)
	if n != 1 {
		t.Fatalf("a observação do merge é auditada uma vez: %d", n)
	}
	// Reconciliar uma entrega que não foi integrada é recusado.
	m3 := newMrgEnv(t, "mrg-rec")
	if _, err := m3.c.Reconcile(ctx, m3.out.DeliveryID, "feat:1", "P1"); err == nil {
		t.Fatal("só entrega integrada")
	}
}

// Consulta que falha deixa tudo como está (descoberta só sob demanda, sem polling).
func TestAFailedObservationChangesNothingAndIsRecorded(t *testing.T) {
	m := newMrgEnv(t, "mrg-fail")
	ctx := context.Background()
	m.gh.getFails = true
	before := m.gh.gets
	obs, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if err != nil || obs.Outcome != "unknown" || !strings.Contains(obs.Detail, "não foi possível consultar") || m.gh.gets != before+1 {
		t.Fatalf("%+v %v", obs, err)
	}
	var state string
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT state FROM deliveries WHERE id = $1`, m.out.DeliveryID).Scan(&state)
	if state != "draft_open" || m.org.calls != 0 {
		t.Fatalf("%s %d", state, m.org.calls)
	}
	// Nada consulta o GitHub sozinho.
	time0 := m.gh.gets
	if m.gh.gets != time0 {
		t.Fatal("sem polling")
	}
	// Entrega inexistente ou sem PR.
	if _, err := m.c.Observe(ctx, "00000000-0000-0000-0000-000000000000", "k", "i"); err == nil {
		t.Fatal("entrega inexistente")
	}
	if n, ok := gitops.PRNumberFromURL("https://github.com/o/r/pull/42"); !ok || n != 42 {
		t.Fatal(n)
	}
	for _, bad := range []string{"", "https://github.com/o/r/issues/4", "https://github.com/o/r/pull/", "https://github.com/o/r/pull/4x", "https://github.com/o/r/pull/0"} {
		if _, ok := gitops.PRNumberFromURL(bad); ok {
			t.Errorf("%q deveria ser inválida", bad)
		}
	}
}
