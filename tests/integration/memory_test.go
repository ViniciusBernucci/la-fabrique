package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/memory"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func memInput(p qProj, branch, key, content, rev string) memory.Input {
	return memory.Input{ProjectID: p.proj, FeatureID: p.feature, Branch: branch, Key: key, Kind: "fact", Content: content, Source: "execução da tarefa schema",
		Revision: rev, Evidence: []memory.Evidence{{Label: "check", Ref: "make test@" + rev}}}
}

func newMemStore(m *mrgEnv) *memory.Store {
	return &memory.Store{DB: m.e.db.Pool, Redactor: security.NewRedactor()}
}

func TestPendingMemoryIsDeduplicatedLinkedToEvidenceAndRedacted(t *testing.T) {
	m := newMrgEnv(t, "mem-dedup")
	ctx := context.Background()
	s := newMemStore(m)
	s.Redactor.Register("tok-memoria-secreto")
	in := memInput(m.p, "lf/f-1", "  Tabela   PEDIDOS usa soft delete ", "a tabela usa soft delete; token tok-memoria-secreto", shaB)
	a, created, err := s.Record(ctx, in)
	if err != nil || !created || a.State != memory.StatePending || a.Key != "tabela pedidos usa soft delete" || strings.Contains(a.Content, "tok-memoria-secreto") {
		t.Fatalf("%+v %v", a, err)
	}
	// Mesma chave (normalizada) e conteúdo novo: um só registro, evidências somadas, versão mais nova.
	in.Key, in.Content = "tabela pedidos usa SOFT delete", "a tabela usa soft delete com coluna removido_em"
	in.Evidence = []memory.Evidence{{Label: "revisão", Ref: "rev-9"}, {Label: "check", Ref: "make test@" + shaB}}
	b, created, err := s.Record(ctx, in)
	if err != nil || created || b.ID != a.ID || len(b.Evidence) != 2 || b.Version != 2 || !strings.Contains(b.Content, "removido_em") {
		t.Fatalf("%+v %v", b, err)
	}
	var n int
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM memory_records WHERE project_id = $1`, m.p.proj).Scan(&n)
	if n != 1 {
		t.Fatalf("um só registro: %d", n)
	}
	// Sem evidência, sem revisão, tipo inválido ou sem branch: recusado.
	for _, mut := range []func(*memory.Input){
		func(i *memory.Input) { i.Evidence = nil },
		func(i *memory.Input) { i.Revision = "" },
		func(i *memory.Input) { i.Kind = "opinião" },
		func(i *memory.Input) { i.Branch = "" },
		func(i *memory.Input) { i.Evidence = []memory.Evidence{{Label: "x"}} },
	} {
		bad := memInput(m.p, "lf/f-1", "outra", "x", shaB)
		mut(&bad)
		if _, _, err := s.Record(ctx, bad); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Fatalf("devia recusar: %v", err)
		}
	}
}

// T15-05 — "Projeto A não consome memória de B."
func TestProjectAnNeverConsumesProjectBMemory(t *testing.T) {
	m := newMrgEnv(t, "mem-a")
	ctx := context.Background()
	s := newMemStore(m)
	b := m.qEnv.project(t, "mem-b", planOf("x"))
	ra, _, err := s.Record(ctx, memInput(m.p, "lf/a", "porta da api", "a API de A usa a porta 1", shaB))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Record(ctx, memInput(b, "lf/a", "porta da api", "a API de B usa a porta 2", shaB)); err != nil {
		t.Fatal(err)
	}
	for _, p := range []qProj{m.p, b} {
		got, err := s.Relevant(ctx, p.proj, "lf/a", "porta", 10)
		if err != nil || len(got) != 1 || got[0].ProjectID != p.proj {
			t.Fatalf("%+v %v", got, err)
		}
	}
	if _, err := s.Get(ctx, b.proj, ra.ID); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("registro de A visto por B: %v", err)
	}
	if _, err := s.Invalidate(ctx, b.proj, ra.ID, "tentativa de B"); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("B invalidou o registro de A: %v", err)
	}
	if got, _ := s.Get(ctx, m.p.proj, ra.ID); got.State != memory.StatePending {
		t.Fatalf("%+v", got)
	}
	// Promoção de A (com merge) também não toca em B: oficial só no projeto de A.
	m.gh.setPR(m.out.PR.Number, "closed", true, "cccccccccccccccccccccccccccccccccccccccc")
	if obs, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1"); err != nil || obs.Outcome != "merged" {
		t.Fatalf("%+v %v", obs, err)
	}
	branch, _ := gitops.FeatureBranch(m.p.feature, 1)
	if _, _, err := s.Record(ctx, memInput(m.p, branch, "porta da api", "a API de A usa a porta 1", shaB)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Relevant(ctx, b.proj, "lf/outra", "porta", 10); len(got) != 0 {
		t.Fatalf("B não tem memória oficial de A: %+v", got)
	}
	if _, err := s.Promote(ctx, b.proj, m.out.DeliveryID); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("B não promove a entrega de A: %v", err)
	}
}

// T15-05 — "PR fechada não promove. Merge promove só registros válidos para a revisão integrada."
func TestOnlyAMergeConfirmedByTheOriginPromotesValidRecords(t *testing.T) {
	m := newMrgEnv(t, "mem-promo")
	ctx := context.Background()
	s := newMemStore(m)
	branch, _ := gitops.FeatureBranch(m.p.feature, 1)
	if err := m.d.RecordCommit(ctx, m.p.proj, m.p.feature, "schema", shaA, branch, "schema: cria a tabela"); err != nil {
		t.Fatal(err)
	}
	ok1, _, _ := s.Record(ctx, memInput(m.p, branch, "tabela pedidos", "a tabela de pedidos tem soft delete", shaB))
	ok2, _, _ := s.Record(ctx, memInput(m.p, branch, "indice de pedidos", "há índice por cliente", shaA)) // commit da feature: vale
	stale, _, _ := s.Record(ctx, memInput(m.p, branch, "coluna legada", "a coluna x existe", "dddddddddddddddddddddddddddddddddddddddd"))
	other, _, _ := s.Record(ctx, memInput(m.p, "lf/outra-branch", "so da outra branch", "não é desta entrega", shaB))

	// PR aberta e fechada sem merge: nada é promovido.
	if _, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID); !errors.Is(err, memory.ErrNotMerged) {
		t.Fatalf("aberta: %v", err)
	}
	m.gh.setPR(m.out.PR.Number, "closed", false, "")
	_, _ = m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1")
	if _, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID); !errors.Is(err, memory.ErrNotMerged) {
		t.Fatalf("fechada: %v", err)
	}
	var official int
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM memory_records WHERE state <> 'pending'`).Scan(&official)
	if official != 0 {
		t.Fatalf("nada muda sem merge: %d", official)
	}
	// Pendente é visível só na própria branch.
	if got, _ := s.Relevant(ctx, m.p.proj, branch, "", 10); len(got) != 3 {
		t.Fatalf("a branch vê os seus pendentes: %d", len(got))
	}
	if got, _ := s.Relevant(ctx, m.p.proj, "lf/terceira", "", 10); len(got) != 0 {
		t.Fatalf("outra branch não vê pendentes: %+v", got)
	}

	// Merge confirmado: promove só o que vale para a revisão integrada.
	m.gh.setPR(m.out.PR.Number, "closed", true, "cccccccccccccccccccccccccccccccccccccccc")
	if _, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1"); err != nil {
		t.Fatal(err)
	}
	// (a entrega estava `closed`: o merge posterior é observado e passa a `merged`)
	p, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID)
	if err != nil || len(p.Promoted) != 2 || len(p.Rejected) != 1 || p.MergeSHA != "cccccccccccccccccccccccccccccccccccccccc" {
		t.Fatalf("%+v %v", p, err)
	}
	for _, id := range []string{ok1.ID, ok2.ID} {
		r, _ := s.Get(ctx, m.p.proj, id)
		if r.State != memory.StateOfficial || r.PromotedRevision != p.MergeSHA || r.PromotedAt == nil {
			t.Fatalf("%+v", r)
		}
	}
	if r, _ := s.Get(ctx, m.p.proj, stale.ID); r.State != memory.StateInvalidated || !strings.Contains(r.InvalidReason, "não integrou") {
		t.Fatalf("%+v", r)
	}
	if r, _ := s.Get(ctx, m.p.proj, other.ID); r.State != memory.StatePending {
		t.Fatalf("outra branch fica pendente: %+v", r)
	}
	if again, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID); err != nil || len(again.Promoted) != 0 || len(again.Rejected) != 0 {
		t.Fatalf("repetir não muda: %+v %v", again, err)
	}
	// Oficial é visto por qualquer branch do projeto.
	if got, _ := s.Relevant(ctx, m.p.proj, "lf/terceira", "soft delete", 10); len(got) != 1 || got[0].State != memory.StateOfficial {
		t.Fatalf("%+v", got)
	}
}

func TestPromotionDeduplicatesAgainstOfficialAndSupersedesChangedFacts(t *testing.T) {
	m := newMrgEnv(t, "mem-super")
	ctx := context.Background()
	s := newMemStore(m)
	branch, _ := gitops.FeatureBranch(m.p.feature, 1)
	m.gh.setPR(m.out.PR.Number, "closed", true, "cccccccccccccccccccccccccccccccccccccccc")
	if _, err := m.c.Observe(ctx, m.out.DeliveryID, "feat:1", "P1"); err != nil {
		t.Fatal(err)
	}
	first, _, _ := s.Record(ctx, memInput(m.p, branch, "limite de itens", "o pedido aceita 10 itens", shaB))
	if _, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID); err != nil {
		t.Fatal(err)
	}
	// Mesmo fato de novo: duplicata, evidência somada ao oficial.
	dup := memInput(m.p, branch, "limite de itens", "o pedido aceita 10 itens", shaB)
	dup.Evidence = []memory.Evidence{{Label: "revisão", Ref: "rev-1"}}
	if _, _, err := s.Record(ctx, dup); err != nil {
		t.Fatal(err)
	}
	p, err := s.Promote(ctx, m.p.proj, m.out.DeliveryID)
	if err != nil || len(p.Duplicates) != 1 || len(p.Promoted) != 0 {
		t.Fatalf("%+v %v", p, err)
	}
	if off, _ := s.Get(ctx, m.p.proj, first.ID); off.State != memory.StateOfficial || len(off.Evidence) != 2 {
		t.Fatalf("%+v", off)
	}
	// Fato mudado: o novo vira oficial e o anterior fica invalidado, com o motivo.
	chg := memInput(m.p, branch, "limite de itens", "o pedido aceita 50 itens", shaB)
	if _, _, err := s.Record(ctx, chg); err != nil {
		t.Fatal(err)
	}
	p, err = s.Promote(ctx, m.p.proj, m.out.DeliveryID)
	if err != nil || len(p.Promoted) != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	if old, _ := s.Get(ctx, m.p.proj, first.ID); old.State != memory.StateInvalidated || !strings.Contains(old.InvalidReason, "substituída") {
		t.Fatalf("%+v", old)
	}
	got, _ := s.Relevant(ctx, m.p.proj, "", "limite", 10)
	if len(got) != 1 || !strings.Contains(got[0].Content, "50") {
		t.Fatalf("%+v", got)
	}
	// Invalidação manual de um oficial.
	if r, err := s.Invalidate(ctx, m.p.proj, got[0].ID, "regra mudou"); err != nil || r.State != memory.StateInvalidated {
		t.Fatalf("%+v %v", r, err)
	}
	if got, _ := s.Relevant(ctx, m.p.proj, "", "limite", 10); len(got) != 0 {
		t.Fatalf("invalidado não é consumido: %+v", got)
	}
	// O validador opcional também recusa.
	s.Validate = func(r memory.Record) string {
		if strings.Contains(r.Content, "proibido") {
			return "contradiz uma regra do projeto"
		}
		return ""
	}
	_, _, _ = s.Record(ctx, memInput(m.p, branch, "regra x", "isto é proibido", shaB))
	p, _ = s.Promote(ctx, m.p.proj, m.out.DeliveryID)
	if len(p.Rejected) != 1 || p.Rejected[0].InvalidReason != "contradiz uma regra do projeto" {
		t.Fatalf("%+v", p)
	}
}

func TestLessonsAreExtractedFromFixedReviewFindingsWithoutDuplicating(t *testing.T) {
	m := newMrgEnv(t, "mem-extract")
	ctx := context.Background()
	s := newMemStore(m)
	var rid string
	if err := m.e.db.Pool.QueryRow(ctx, `INSERT INTO reviews (project_id, feature_id, task_key, round, revision, implementer, reviewer, status, fix_revision)
		VALUES ($1, $2, 'schema', 1, $3, 'codex', 'claude', 'approved', $4) RETURNING id::text`, m.p.proj, m.p.feature, shaA, shaB).Scan(&rid); err != nil {
		t.Fatal(err)
	}
	for _, f := range []struct{ sev, title, state, fix string }{
		{"critical", "Consulta sem filtro de projeto", "fixed", shaB},
		{"relevant", "Falta teste do caso vazio", "fixed", shaB},
		{"low", "Nome pouco claro", "fixed", shaB},
		{"relevant", "Ainda aberto", "open", ""},
	} {
		if _, err := m.e.db.Pool.Exec(ctx, `INSERT INTO review_findings (review_id, severity, title, body, file, state, fix_revision) VALUES ($1, $2, $3, 'detalhe', 'internal/x.go', $4, $5)`, rid, f.sev, f.title, f.state, f.fix); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ExtractFromDelivery(ctx, m.p.proj, m.out.DeliveryID)
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	for _, r := range got {
		if r.Kind != "lesson" || r.State != memory.StatePending || len(r.Evidence) != 2 || r.Revision != shaB {
			t.Fatalf("%+v", r)
		}
	}
	if _, err := s.ExtractFromDelivery(ctx, m.p.proj, m.out.DeliveryID); err != nil {
		t.Fatal(err)
	}
	var n int
	_ = m.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM memory_records WHERE project_id = $1`, m.p.proj).Scan(&n)
	if n != 2 {
		t.Fatalf("repetir a extração não duplica: %d", n)
	}
}
