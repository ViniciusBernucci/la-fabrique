package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/issuesync"
	"github.com/ViniciusBernucci/le-fabrique/internal/prcycle"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// prFakeGH é o GitHub de teste: estado da PR, revisões e comentários com leitura por cursor.
type prFakeGH struct {
	mu        sync.Mutex
	pr        gitops.PRState
	reviews   []gitops.Review
	conv      []gitops.PRComment
	lines     []gitops.PRComment
	perms     map[string]string
	contains  func(ancestor, head string) bool
	rateLimit error
	afters    []int64
}

func (g *prFakeGH) Repo() string { return "org/app" }
func (g *prFakeGH) ReadPR(context.Context, int) (gitops.PRState, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.pr, g.rateLimit
}
func (g *prFakeGH) ListReviews(_ context.Context, _ int, after int64) ([]gitops.Review, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.afters = append(g.afters, after)
	var out []gitops.Review
	for _, r := range g.reviews {
		if r.ID > after {
			out = append(out, r)
		}
	}
	return out, false, nil
}
func (g *prFakeGH) ListConversation(_ context.Context, _ int, after int64) ([]gitops.PRComment, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []gitops.PRComment
	for _, c := range g.conv {
		if c.ID > after {
			out = append(out, c)
		}
	}
	return out, false, nil
}
func (g *prFakeGH) ListLineComments(_ context.Context, _ int, after int64) ([]gitops.PRComment, bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []gitops.PRComment
	for _, c := range g.lines {
		if c.ID > after {
			out = append(out, c)
		}
	}
	return out, false, nil
}
func (g *prFakeGH) Permission(_ context.Context, login string) (string, error) {
	if p, ok := g.perms[login]; ok {
		return p, nil
	}
	return "read", nil
}
func (g *prFakeGH) Contains(_ context.Context, a, h string) (bool, error) {
	if g.contains == nil {
		return a == h, nil
	}
	return g.contains(a, h), nil
}

// prFakeOrigin é a origem de teste. loseFirst: cria a Issue mas responde com falha (timeout após o efeito).
type prFakeOrigin struct {
	mu        sync.Mutex
	items     []integrations.Item
	loseFirst bool
	creates   int
}

func (o *prFakeOrigin) CreateItem(_ context.Context, _ string, in integrations.ItemInput) (integrations.Item, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.creates++
	it := integrations.Item{Provider: "jira", ID: fmt.Sprintf("r%d", len(o.items)+1), Key: fmt.Sprintf("R-%d", len(o.items)+1), Title: in.Title, Description: in.Description, Status: in.Status, Version: "v1", URL: "https://x/r"}
	o.items = append(o.items, it)
	if o.loseFirst {
		o.loseFirst = false
		return integrations.Item{}, errors.New("timeout")
	}
	return it, nil
}
func (o *prFakeOrigin) ListItems(context.Context, string, integrations.ListOptions) (integrations.ItemPage, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return integrations.ItemPage{Items: append([]integrations.Item{}, o.items...)}, nil
}

type prEnvT struct {
	db  testDB
	p   string
	co  *sentinel.Coordinator
	svc *prcycle.Service
	gh  *prFakeGH
	or  *prFakeOrigin
	tg  prcycle.Target
}

func newPREnv(t *testing.T) *prEnvT {
	t.Helper()
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "prcycle")
	co := &sentinel.Coordinator{DB: db.Pool}
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "F-1", Kind: "feature", Status: statusflow.DraftPR}); err != nil {
		t.Fatal(err)
	}
	gh := &prFakeGH{pr: gitops.PRState{Number: 7, State: "open", Draft: true, HeadSHA: "aaa1111", HeadRef: "feat/f-1", Base: "main"}, perms: map[string]string{"rev": "write", "boss": "admin"}}
	or := &prFakeOrigin{}
	sy := &issuesync.Syncer{DB: db.Pool, Sentinel: co}
	m := syncMap("jira")
	m.Remote["Reprovado"] = statusflow.ReprovadoPR
	src := issuesync.Source{Kind: "jira", Container: "C", Map: m}
	e := &prEnvT{db: db, p: p.ID, co: co, gh: gh, or: or}
	e.svc = &prcycle.Service{DB: db.Pool, Ledger: &effects.Ledger{DB: db.Pool}, Sentinel: co}
	e.tg = prcycle.Target{Reader: gh, Origin: or, Container: "C", Map: m,
		Register: func(ctx context.Context, it integrations.Item, parent string) error {
			return sy.RegisterCorrection(ctx, p.ID, src, it, parent)
		}}
	if err := e.svc.Link(ctx, p.ID, "", 0, "F-1", "x"); err == nil {
		t.Fatal("PR inválida deve ser recusada")
	}
	if err := e.svc.Link(ctx, p.ID, "", 7, "F-1", "feat/f-1"); err != nil {
		t.Fatal(err)
	}
	return e
}

func (e *prEnvT) poll(t *testing.T) prcycle.Report {
	t.Helper()
	r, err := e.svc.PollAll(context.Background(), e.p, e.tg)
	if err != nil {
		t.Fatalf("poll: %v", err)
	}
	return r
}

func at(min int) time.Time { return time.Date(2026, 5, 1, 10, min, 0, 0, time.UTC) }

func (e *prEnvT) view(t *testing.T) prcycle.WatchView {
	t.Helper()
	vs, err := e.svc.List(context.Background(), e.p)
	if err != nil || len(vs) != 1 {
		t.Fatalf("%v %v", vs, err)
	}
	return vs[0]
}

// AJ09-02 / CT11 — rodada por revisão formal: comentário solto, comando sem permissão e repetição
// não criam Issue; a revisão agrupa e cria uma só; o comando posterior sobre o mesmo conjunto
// pertence à mesma rodada; a segunda rodada cria nova Issue.
func TestPRRoundsGroupCommentsAndCreateOneIssuePerRound(t *testing.T) {
	e := newPREnv(t)
	ctx := context.Background()
	// Só comentário solto e um comando de quem não tem permissão.
	e.gh.conv = []gitops.PRComment{
		{ID: 1, Author: "dev", Body: "Troque este nome", URL: "https://gh/c1", At: at(1)},
		{ID: 2, Author: "outsider", Body: "/fabrique fix", URL: "https://gh/c2", At: at(2)},
	}
	e.poll(t)
	if e.or.creates != 0 || len(e.view(t).Rounds) != 0 {
		t.Fatalf("comentário solto e comando sem permissão não criam ticket: %+v", e.view(t))
	}
	// Revisão formal com mudanças + duas linhas.
	e.gh.reviews = []gitops.Review{{ID: 10, Author: "rev", State: "CHANGES_REQUESTED", Body: "Precisa ajustar", URL: "https://gh/r10", CommitID: "aaa1111", At: at(5)}}
	e.gh.lines = []gitops.PRComment{
		{ID: 100, ReviewID: 10, Author: "rev", Body: "falta validação", URL: "https://gh/l100", At: at(4), Line: true},
		{ID: 101, ReviewID: 10, Author: "rev", Body: "teste ausente", URL: "https://gh/l101", At: at(4), Line: true},
	}
	e.poll(t)
	e.poll(t) // repetição
	v := e.view(t)
	if len(v.Rounds) != 1 || v.Rounds[0].Status != "issue_created" || v.Rounds[0].Comments != 3 || e.or.creates != 1 {
		t.Fatalf("uma rodada, uma Issue, três comentários: %+v creates=%d", v.Rounds, e.or.creates)
	}
	body := e.or.items[0].Description
	for _, want := range []string{"https://gh/c1", "https://gh/l100", "https://gh/l101", "Revisão #10", "F-1", "#7", prcycle.Marker(v.Rounds[0].Key)} {
		if !strings.Contains(body, want) {
			t.Fatalf("a Issue deve conter %q:\n%s", want, body)
		}
	}
	// A Issue entra como correção vinculada à original, em REPROVADO_PR (aguardando liberação humana).
	iss, err := e.co.Get(ctx, e.p, v.Rounds[0].IssueKey)
	if err != nil || iss.Status != statusflow.ReprovadoPR || iss.Kind != "correction" {
		t.Fatalf("%+v %v", iss, err)
	}
	// Comando autorizado sobre o mesmo conjunto (nada novo na janela): mesma rodada.
	e.gh.conv = append(e.gh.conv, gitops.PRComment{ID: 3, Author: "boss", Body: "/fabrique fix", URL: "https://gh/c3", At: at(6)})
	e.poll(t)
	if v := e.view(t); len(v.Rounds) != 1 || e.or.creates != 1 {
		t.Fatalf("comando sobre o mesmo conjunto é a mesma rodada: %+v", v.Rounds)
	}
	// Snapshot: edição posterior do comentário não muda o texto auditado.
	e.gh.conv[0].Body = "EDITADO"
	var snap, cur string
	_ = e.db.Pool.QueryRow(ctx, `SELECT body, current_body FROM pr_events WHERE kind='comment' AND remote_id=1`).Scan(&snap, &cur)
	if snap != "Troque este nome" {
		t.Fatalf("snapshot alterado: %q %q", snap, cur)
	}
	// Segunda rodada: novo comentário + comando autorizado → nova Issue.
	e.gh.conv = append(e.gh.conv, gitops.PRComment{ID: 4, Author: "dev", Body: "Mais um ajuste", URL: "https://gh/c4", At: at(10)},
		gitops.PRComment{ID: 5, Author: "rev", Body: "/fabrique fix", URL: "https://gh/c5", At: at(11)})
	e.poll(t)
	if v := e.view(t); len(v.Rounds) != 2 || e.or.creates != 2 || v.Rounds[1].IssueKey == v.Rounds[0].IssueKey {
		t.Fatalf("segunda rodada cria nova Issue: %+v creates=%d", v.Rounds, e.or.creates)
	}
}

// AJ09-02 — timeout depois da criação externa: reconcilia pelo marcador e não duplica.
func TestPRRoundIssueIsReconciledAfterTimeoutWithoutDuplicating(t *testing.T) {
	e := newPREnv(t)
	e.or.loseFirst = true
	e.gh.reviews = []gitops.Review{{ID: 10, Author: "rev", State: "CHANGES_REQUESTED", Body: "Ajustar X", URL: "https://gh/r10", At: at(5)}}
	// O ledger verifica por leitura (marcador) antes de repetir: a Issue criada é reconhecida na própria passagem.
	e.poll(t)
	e.poll(t)
	v := e.view(t)
	if e.or.creates != 1 || len(e.or.items) != 1 || len(v.Rounds) != 1 || v.Rounds[0].Status != "issue_created" || v.Rounds[0].IssueKey == "" {
		t.Fatalf("exatamente uma Issue após o timeout: items=%d %+v", len(e.or.items), v.Rounds)
	}
}

// AJ09-02 — fechamento sem achados fica ambíguo e não dispara correção.
func TestPRClosingWithoutFindingsIsExposedAsAmbiguous(t *testing.T) {
	e := newPREnv(t)
	e.gh.conv = []gitops.PRComment{{ID: 9, Author: "boss", Body: "/fabrique fix", URL: "https://gh/c9", At: at(3)}}
	e.poll(t)
	v := e.view(t)
	if e.or.creates != 0 || len(v.Rounds) != 1 || v.Rounds[0].Status != "ambiguous" || len(v.Pending) != 1 {
		t.Fatalf("%+v creates=%d", v, e.or.creates)
	}
}

// AJ09-01 / CT15 — cursor por PR: a segunda leitura parte do último ID; limite de requisições não perde nem duplica.
func TestPRPollUsesCursorsAndSurvivesRateLimitAndRestart(t *testing.T) {
	e := newPREnv(t)
	e.gh.reviews = []gitops.Review{{ID: 10, Author: "rev", State: "COMMENTED", Body: "ok", At: at(1)}}
	e.poll(t)
	e.gh.rateLimit = &gitops.ErrRateLimited{RetryAfter: time.Second}
	if _, err := e.svc.PollAll(context.Background(), e.p, e.tg); err == nil {
		t.Fatal("limite de requisições deve sair como erro")
	}
	e.gh.rateLimit = nil
	// "Reinício": novo serviço sobre o mesmo banco continua do cursor gravado.
	svc2 := &prcycle.Service{DB: e.db.Pool, Ledger: &effects.Ledger{DB: e.db.Pool}, Sentinel: e.co}
	if _, err := svc2.PollAll(context.Background(), e.p, e.tg); err != nil {
		t.Fatal(err)
	}
	if last := e.gh.afters[len(e.gh.afters)-1]; last != 10 {
		t.Fatalf("a leitura deve partir do cursor gravado: %v", e.gh.afters)
	}
	// A rotina da Sentinela usa o nome e a agenda próprios (independentes de Jira/ClickUp).
	r := e.svc.Routine(func(context.Context, string) (prcycle.Target, error) { return e.tg, nil }, 3*time.Minute)
	if r.Name != sentinel.RoutinePRs || r.Every != 3*time.Minute {
		t.Fatalf("%+v", r)
	}
	if _, err := r.Run(context.Background(), e.p, ""); err != nil {
		t.Fatal(err)
	}
}

// AJ09-04 / CT13 — aprovação, Ready e fechamento sem merge não concluem; o merge verificado conclui a
// feature e a correção incorporada (inclusive squash); a pendente fica aberta; repetir não duplica.
func TestPRMergeConcludesOnlyConfirmedAndIncorporated(t *testing.T) {
	e := newPREnv(t)
	ctx := context.Background()
	// Duas rodadas com Issue.
	e.gh.reviews = []gitops.Review{{ID: 10, Author: "rev", State: "CHANGES_REQUESTED", Body: "Ajuste A", At: at(1)}}
	e.poll(t)
	e.gh.reviews = append(e.gh.reviews, gitops.Review{ID: 11, Author: "rev", State: "CHANGES_REQUESTED", Body: "Ajuste B", At: at(20)})
	e.poll(t)
	v := e.view(t)
	if len(v.Rounds) != 2 {
		t.Fatalf("%+v", v.Rounds)
	}
	k1, k2 := v.Rounds[0].IssueKey, v.Rounds[1].IssueKey
	// A correção 1 publicou commits na PR e está em DRAFT_PR; a 2 não publicou.
	_, _ = e.db.Pool.Exec(ctx, `UPDATE pr_rounds SET published_head='bbb2222' WHERE round_key=$1`, v.Rounds[0].Key)
	_, _ = e.db.Pool.Exec(ctx, `UPDATE flow_issues SET status_id=(SELECT id FROM status_catalog WHERE code=$3) WHERE project_id=$1 AND external_key=$2`, e.p, k1, statusflow.DraftPR)

	status := func(key string) string {
		i, err := e.co.Get(ctx, e.p, key)
		if err != nil {
			t.Fatal(err)
		}
		return i.Status
	}
	// Aprovação e Ready for Review: nada conclui.
	e.gh.reviews = append(e.gh.reviews, gitops.Review{ID: 12, Author: "boss", State: "APPROVED", At: at(30)})
	e.gh.pr.Draft = false
	e.poll(t)
	if status("F-1") != statusflow.DraftPR || e.view(t).Draft {
		t.Fatalf("aprovação/Ready não concluem: %s %+v", status("F-1"), e.view(t))
	}
	// Fechada sem merge: decisão humana, nada conclui.
	e.gh.pr.State = "closed"
	e.poll(t)
	if v := e.view(t); !v.NeedsDecision || status("F-1") != statusflow.DraftPR {
		t.Fatalf("%+v", v)
	}
	// Merge com squash: SHA do merge diferente, cabeça integrada contém a publicada.
	e.gh.pr = gitops.PRState{Number: 7, State: "closed", Merged: true, MergeSHA: "ccc3333", HeadSHA: "bbb2222", HeadRef: "feat/f-1", Base: "main", MergedAt: at(40)}
	e.poll(t)
	e.poll(t) // repetição: sem efeito duplicado
	if status("F-1") != statusflow.Concluido || status(k1) != statusflow.Concluido {
		t.Fatalf("feature e correção incorporada concluem: %s %s", status("F-1"), status(k1))
	}
	if status(k2) != statusflow.ReprovadoPR {
		t.Fatalf("a correção pendente fica aberta: %s", status(k2))
	}
	v = e.view(t)
	if v.MergeSHA != "ccc3333" || !v.Rounds[0].Incorporated || v.Rounds[1].Incorporated || len(v.Pending) != 1 {
		t.Fatalf("%+v", v)
	}
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM flow_transitions t JOIN flow_issues i ON i.id=t.issue_id WHERE i.external_key='F-1' AND t.to_code='CONCLUIDO' AND t.decision='applied'`).Scan(&n)
	if n != 1 {
		t.Fatalf("a conclusão é aplicada uma única vez: %d", n)
	}
}

// AJ09-04 — merge sem SHA de integração não conclui (sem prova).
func TestPRMergeWithoutProofDoesNotConclude(t *testing.T) {
	e := newPREnv(t)
	e.gh.pr = gitops.PRState{Number: 7, State: "closed", Merged: true, HeadSHA: "bbb2222", Base: "main"}
	r := e.poll(t)
	i, _ := e.co.Get(context.Background(), e.p, "F-1")
	if i.Status != statusflow.DraftPR || len(r.Pending) == 0 {
		t.Fatalf("%s %+v", i.Status, r)
	}
}
