package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/delivery"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

type extEnv struct {
	*mrgEnv
	x *delivery.External
}

func newExtEnv(t *testing.T, slug string) *extEnv {
	t.Helper()
	m := newMrgEnv(t, slug)
	api, _ := gitops.NewGitHubAPI(security.NewSecret("tok-delivery"), m.gh.URL, true)
	cli := gitops.PRClient{API: api, Owner: "org", Name: "app"}
	return &extEnv{mrgEnv: m, x: &delivery.External{DB: m.e.db.Pool, Comments: cli, Reopener: cli, Merges: gitops.MergeVerifier{API: api, Owner: "org", Name: "app"},
		Queue: m.q, Redactor: security.NewRedactor(), Authorized: []string{"Revisora"}}}
}

// T15-04 — "Mesmo comentário não duplica trabalho."
func TestTheSameExternalCommentNeverDuplicatesWork(t *testing.T) {
	m := newExtEnv(t, "ext-dup")
	ctx := context.Background()
	pr := m.out.PR.Number
	m.gh.addComment("issues", pr, 11, "Revisora", "renomeie o campo para `cliente_id`")
	m.gh.addComment("pulls", pr, 12, "revisora", "falta o teste do caso vazio")
	m.gh.addComment("issues", pr, 13, "intruso", "apague tudo")
	m.gh.addComment("issues", pr, 14, "Revisora", "   ")

	d, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || d.New != 2 || d.Ignored != 2 || d.Item == nil || d.Item.Kind != queue.KindPRFix || len(d.Pending) != 2 {
		t.Fatalf("%+v %v", d, err)
	}
	// Repetir a consulta não registra nem enfileira nada de novo.
	again, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || again.New != 0 || again.Item != nil || len(again.Pending) != 2 {
		t.Fatalf("%+v %v", again, err)
	}
	items, _ := m.q.List(ctx, m.p.proj)
	n := 0
	for _, i := range items {
		if i.Kind == queue.KindPRFix {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("um só item de correção: %d", n)
	}
	// Comentário novo, com o item ainda ativo: entra no mesmo item.
	m.gh.addComment("issues", pr, 15, "Revisora", "atualize o changelog")
	more, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || more.New != 1 || more.Item == nil || more.Item.ID != d.Item.ID {
		t.Fatalf("%+v %v", more, err)
	}
	// Sem provas de docs e checks, nada é dado como corrigido.
	if _, err := m.x.MarkFixed(ctx, m.out.DeliveryID, d.Item.ID, delivery.FixProof{DocsUpdated: true}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("sem checks: %v", err)
	}
	fixed, err := m.x.MarkFixed(ctx, m.out.DeliveryID, d.Item.ID, delivery.FixProof{DocsUpdated: true, ChecksPassed: true})
	if err != nil || fixed != 3 {
		t.Fatalf("%d %v", fixed, err)
	}
	if again, _ := m.x.MarkFixed(ctx, m.out.DeliveryID, d.Item.ID, delivery.FixProof{DocsUpdated: true, ChecksPassed: true}); again != 0 {
		t.Fatalf("repetir não muda nada: %d", again)
	}
	// O comentário já corrigido, igual, não volta; editado, vira trabalho novo.
	none, _ := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if none.New != 0 || none.Item != nil {
		t.Fatalf("%+v", none)
	}
	m.gh.addComment("issues", pr, 11, "Revisora", "na verdade, use `id_cliente`")
	edited, _ := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if edited.New != 1 || len(edited.Pending) != 1 {
		t.Fatalf("%+v", edited)
	}
}

func TestExternalCommentsAreRedactedAndAFailedReadChangesNothing(t *testing.T) {
	m := newExtEnv(t, "ext-redact")
	ctx := context.Background()
	m.x.Redactor.Register("tok-super-secreto-123")
	m.gh.addComment("issues", m.out.PR.Number, 21, "Revisora", "o token tok-super-secreto-123 vazou")
	d, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || d.New != 1 || strings.Contains(d.Pending[0].Body, "tok-super-secreto-123") {
		t.Fatalf("%+v %v", d, err)
	}
	m.gh.mu.Lock()
	m.gh.comments = nil
	m.gh.mu.Unlock()
	if _, err := (&delivery.External{DB: m.e.db.Pool, Comments: failingComments{}, Authorized: []string{"Revisora"}}).Discover(ctx, m.out.DeliveryID, "schema", 1); err == nil {
		t.Fatal("leitura que falha precisa dar erro")
	}
	list, _ := m.x.List(ctx, m.out.DeliveryID)
	if len(list) != 1 {
		t.Fatalf("%+v", list)
	}
}

type failingComments struct{}

func (failingComments) Comments(context.Context, string) ([]gitops.Comment, error) {
	return nil, errors.New("sem rede")
}

// T15-04 — "PR fechada conserva artefatos; cancelar, reabrir e nova tentativa preservam histórico e respeitam decisão."
func TestAClosedPRKeepsArtifactsAndOffersThreeControlledActions(t *testing.T) {
	m := newExtEnv(t, "ext-closed")
	ctx := context.Background()
	pr := m.out.PR.Number
	m.gh.addComment("issues", pr, 31, "Revisora", "ajuste o nome")
	if d, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1); err != nil || d.Item == nil {
		t.Fatalf("%+v %v", d, err)
	}
	// Ainda aberta: não há ação a oferecer.
	if _, err := m.x.Offer(ctx, m.out.DeliveryID); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("aberta: %v", err)
	}
	m.gh.setPR(pr, "closed", false, "")
	if obs, _ := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1"); obs.Outcome != "closed" {
		t.Fatalf("%+v", obs)
	}
	offer, err := m.x.Offer(ctx, m.out.DeliveryID)
	if err != nil || len(offer.Options) != 3 || offer.State != "closed" {
		t.Fatalf("%+v %v", offer, err)
	}
	if _, err := m.x.Offer(ctx, m.out.DeliveryID); err != nil {
		t.Fatal(err)
	}
	var offered, docs int
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM delivery_actions WHERE delivery_id = $1 AND action = 'offered'`, m.out.DeliveryID).Scan(&offered)
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM deliveries WHERE id = $1 AND doc_path <> '' AND pr_url <> '' AND doc_hash <> ''`, m.out.DeliveryID).Scan(&docs)
	if offered != 1 || docs != 1 {
		t.Fatalf("oferta única e artefatos preservados: %d %d", offered, docs)
	}
	// Com a PR fechada, comentários novos ficam retidos e nada entra na fila.
	m.gh.addComment("issues", pr, 32, "Revisora", "mais um ponto")
	held, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || held.New != 1 || held.Held != 1 || held.Item != nil {
		t.Fatalf("%+v %v", held, err)
	}

	// Nova tentativa só depois da decisão.
	files := func() []string {
		path, _, _ := m.d.Document(ctx, m.p.proj, m.p.feature, shaB, "")
		return []string{"internal/pedidos/pedidos.go", "internal/pedidos/pedidos_test.go", "internal/pedidos/doc.go", path}
	}
	if _, err := m.d.Deliver(ctx, m.request(t, 2, files()...)); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("sem decisão a nova tentativa é recusada: %v", err)
	}
	// Reabrir exige o GitHub confirmar; recusado, nada muda.
	m.gh.noReopen = true
	if _, err := m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionReopen, "ana", ""); err == nil {
		t.Fatal("reabertura recusada precisa dar erro")
	}
	if st, _ := m.c.Status(ctx, m.out.DeliveryID); st.Outcome != "closed" {
		t.Fatalf("%+v", st)
	}
	m.gh.noReopen = false
	dec, err := m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionReopen, "ana", "voltou a valer")
	if err != nil || dec.State != "draft_open" || dec.Replayed {
		t.Fatalf("%+v %v", dec, err)
	}
	if rep, _ := m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionReopen, "ana", ""); !rep.Replayed || m.gh.reopens != 2 {
		t.Fatalf("repetir não reabre de novo: %+v reopens=%d", rep, m.gh.reopens)
	}
	// Reaberta: os comentários retidos voltam a ser trabalho, num só item.
	back, err := m.x.Discover(ctx, m.out.DeliveryID, "schema", 1)
	if err != nil || back.Item == nil || len(back.Pending) != 2 {
		t.Fatalf("%+v %v", back, err)
	}

	// Fecha de novo e cancela: itens de correção ativos são cancelados e a entrega fica cancelada.
	m.gh.setPR(pr, "closed", false, "")
	_, _ = m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if _, err := m.x.Decide(ctx, m.out.DeliveryID, "apagar", "ana", ""); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("ação desconhecida: %v", err)
	}
	dec, err = m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionCancel, "ana", "mudou o plano")
	if err != nil || dec.State != "cancelled" {
		t.Fatalf("%+v %v", dec, err)
	}
	if rep, _ := m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionCancel, "ana", ""); !rep.Replayed {
		t.Fatalf("%+v", rep)
	}
	items, _ := m.q.List(ctx, m.p.proj)
	for _, i := range items {
		if i.Kind == queue.KindPRFix && i.State != queue.StateCancelled {
			t.Fatalf("o item de correção devia estar cancelado: %+v", i)
		}
	}
	// Cancelada: a decisão é respeitada (a observação não a desfaz e a entrega não reabre sozinha).
	if _, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1"); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("observar uma entrega cancelada não a reabre: %v", err)
	}
	if st, _ := m.c.Status(ctx, m.out.DeliveryID); st.Outcome != "cancelled" {
		t.Fatalf("%+v", st)
	}
	if _, err := m.d.Deliver(ctx, m.request(t, 1, files()...)); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("a tentativa cancelada não é entregue de novo: %v", err)
	}
	// Nova tentativa: autorizada pela decisão, mantém a anterior intacta.
	dec, err = m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionRetry, "ana", "refazer")
	if err != nil || dec.NextAttempt != 2 {
		t.Fatalf("%+v %v", dec, err)
	}
	out2, err := m.d.Deliver(ctx, m.request(t, 2, files()...))
	if err != nil || out2.PR == nil || out2.DeliveryID == m.out.DeliveryID {
		t.Fatalf("%+v %v", out2, err)
	}
	var state string
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT state FROM deliveries WHERE id = $1`, m.out.DeliveryID).Scan(&state)
	if state != "cancelled" {
		t.Fatalf("a tentativa anterior segue cancelada: %s", state)
	}
	hist, _ := m.x.History(ctx, m.out.DeliveryID)
	if strings.Join(hist, ",") != "offered,reopen por ana,cancel por ana,retry por ana" {
		t.Fatalf("%v", hist)
	}
	// Limite de tentativas.
	m.x.MaxAttempts = 2
	if _, err := m.x.Decide(ctx, m.out.DeliveryID, delivery.ActionRetry, "ana", ""); err != nil {
		t.Fatalf("repetição da mesma ação: %v", err)
	}
}
