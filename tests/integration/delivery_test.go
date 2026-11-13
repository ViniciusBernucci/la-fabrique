package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checks"
	"github.com/ViniciusBernucci/le-fabrique/internal/delivery"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// fakeGH é o GitHub de mentira: PRs em memória, com resposta perdida injetável.
type fakeGH struct {
	*httptest.Server
	mu       sync.Mutex
	prs      []map[string]any
	creates  int
	loseNext bool
	bodies   []string
	gets     int
	getFails bool
	comments map[string][]map[string]any // "issues/<n>" e "pulls/<n>"
	reopens  int
	noReopen bool
}

func newFakeGH(t *testing.T) *fakeGH {
	g := &fakeGH{}
	g.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		defer g.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("Authorization") != "Bearer tok-delivery" {
			w.WriteHeader(401)
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/org/app/pulls":
			head := strings.TrimPrefix(r.URL.Query().Get("head"), "org:")
			out := []map[string]any{}
			for _, p := range g.prs {
				if p["head"].(map[string]any)["ref"] == head && p["state"] == "open" {
					out = append(out, p)
				}
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/repos/org/app/"), "/comments")
			out := g.comments[key]
			if out == nil {
				out = []map[string]any{}
			}
			_ = json.NewEncoder(w).Encode(out)
		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/repos/org/app/pulls/"):
			g.reopens++
			if g.noReopen {
				w.WriteHeader(422)
				return
			}
			n := r.URL.Path[len("/repos/org/app/pulls/"):]
			for _, p := range g.prs {
				if fmtNum(p["number"]) == n && p["merged"] != true {
					p["state"] = "open"
				}
			}
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/repos/org/app/pulls/"):
			g.gets++
			if g.getFails {
				w.WriteHeader(502)
				return
			}
			n := r.URL.Path[len("/repos/org/app/pulls/"):]
			for _, p := range g.prs {
				if fmtNum(p["number"]) == n {
					out := map[string]any{}
					for k, v := range p {
						out[k] = v
					}
					_ = json.NewEncoder(w).Encode(out)
					return
				}
			}
			w.WriteHeader(404)
		case r.Method == http.MethodPost && r.URL.Path == "/repos/org/app/pulls":
			var in struct {
				Title, Head, Base, Body string
				Draft                   bool
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			for _, p := range g.prs {
				if p["head"].(map[string]any)["ref"] == in.Head && p["state"] == "open" {
					w.WriteHeader(422)
					_, _ = w.Write([]byte(`{"message":"Validation Failed","errors":[{"message":"A pull request already exists for org:` + in.Head + `."}]}`))
					return
				}
			}
			g.creates++
			n := 100 + g.creates
			g.bodies = append(g.bodies, in.Body)
			g.prs = append(g.prs, map[string]any{"number": n, "html_url": "https://github.com/org/app/pull/" + strconv.Itoa(n), "draft": in.Draft, "state": "open", "merged": false,
				"title": in.Title, "head": map[string]any{"ref": in.Head}, "base": map[string]any{"ref": in.Base}})
			if g.loseNext {
				g.loseNext = false
				w.WriteHeader(502) // a PR foi criada, mas a resposta se perdeu
				return
			}
			w.WriteHeader(201)
			_ = json.NewEncoder(w).Encode(g.prs[len(g.prs)-1])
		default:
			w.WriteHeader(404)
		}
	}))
	t.Cleanup(g.Close)
	return g
}

type fakeOrigin struct{ calls []string }

func (o *fakeOrigin) PublishDraftPR(_ context.Context, _, keyBase, item, url string) error {
	o.calls = append(o.calls, keyBase+"|"+item+"|"+url)
	return nil
}

type dlvEnv struct {
	*qEnv
	p      qProj
	d      *delivery.Deliverer
	gh     *fakeGH
	ck     *checks.Store
	origin *fakeOrigin
}

func newDlvEnv(t *testing.T, slug string) *dlvEnv {
	t.Helper()
	q := newQEnv(t)
	p := q.project(t, slug, planOf("schema", "api"))
	gh := newFakeGH(t)
	api, err := gitops.NewGitHubAPI(security.NewSecret("tok-delivery"), gh.URL, true)
	if err != nil {
		t.Fatal(err)
	}
	pool := q.e.db.Pool
	pub := &gitops.PRPublisher{API: api, Owner: "org", Name: "app", Ledger: &effects.Ledger{DB: pool, Lease: time.Minute, MaxAttempts: 3}}
	d := &dlvEnv{qEnv: q, p: p, d: &delivery.Deliverer{DB: pool, PRs: pub}, gh: gh, ck: &checks.Store{DB: pool, Redactor: security.NewRedactor()}, origin: &fakeOrigin{}}
	return d
}

func (x *dlvEnv) green(t *testing.T, rev string) {
	t.Helper()
	ctx := context.Background()
	sp := []checks.Spec{{Name: "test", Command: "make test"}}
	ex := ckExec{out: map[string]checks.Outcome{"make test": okOut()}}
	_, _ = x.ck.RunSet(ctx, x.p.proj, x.p.feature, checks.RoleBaseline, "base", ex, sp)
	_, _ = x.ck.RunSet(ctx, x.p.proj, x.p.feature, checks.RoleCandidate, rev, ex, sp)
	if _, err := x.ck.Evaluate(ctx, x.p.proj, x.p.feature, rev, "base"); err != nil {
		t.Fatal(err)
	}
}

const shaA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const shaB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func (x *dlvEnv) request(t *testing.T, attempt int, files ...string) delivery.Request {
	t.Helper()
	branch, err := gitops.FeatureBranch(x.p.feature, attempt)
	if err != nil {
		t.Fatal(err)
	}
	return delivery.Request{ProjectID: x.p.proj, FeatureID: x.p.feature, Attempt: attempt, Branch: branch, Base: "main", Revision: shaB,
		Title: "Feature de pedidos", ChangedFiles: files, Gates: []delivery.Gate{delivery.ChecksGate(x.ck, x.p.feature, shaB)},
		Origin: x.origin, OriginKeyBase: "feat:1", OriginItemID: "P1"}
}

// T15-02 — "ausência de documentação bloqueia entrega."
func TestMissingDocumentationBlocksTheDelivery(t *testing.T) {
	x := newDlvEnv(t, "dlv-docs")
	ctx := context.Background()
	x.green(t, shaB)
	if err := x.d.RecordCommit(ctx, x.p.proj, x.p.feature, "schema", shaA, "lf/x", "schema: cria a tabela"); err != nil {
		t.Fatal(err)
	}
	code := []string{"internal/pedidos/pedidos.go", "internal/pedidos/pedidos_test.go", "apps/web/src/pages/Pedidos.tsx", "apps/web/src/pages/Pedidos.test.tsx"}
	out, err := x.d.Deliver(ctx, x.request(t, 1, code...))
	if err != nil || !out.Blocked || out.PR != nil || len(out.DocGaps) != 2 {
		t.Fatalf("%+v %v", out, err)
	}
	joined := strings.Join(out.Reasons, " | ")
	for _, want := range []string{"o módulo internal/pedidos mudou sem documentação", "docs/modulos/pedidos.md", "o módulo apps/web/src/pages", "não está no diff"} {
		if !strings.Contains(joined, want) {
			t.Errorf("faltou %q em: %s", want, joined)
		}
	}
	if x.gh.creates != 0 {
		t.Fatal("nenhuma PR pode ser aberta sem documentação")
	}
	var state string
	_ = x.e.db.Pool.QueryRow(ctx, `SELECT state FROM deliveries WHERE feature_id = $1 AND attempt = 1`, x.p.feature).Scan(&state)
	if state != "blocked" {
		t.Fatalf("a tentativa bloqueada fica registrada: %s", state)
	}
	// Só a documentação do módulo, sem o histórico da feature: ainda bloqueia.
	branch1, _ := gitops.FeatureBranch(x.p.feature, 1)
	_, doc, _ := x.d.Document(ctx, x.p.proj, x.p.feature, shaB, branch1)
	withModuleDoc := append(append([]string{}, code...), "internal/pedidos/doc.go", "docs/modulos/web-pages.md")
	out, _ = x.d.Deliver(ctx, x.request(t, 1, withModuleDoc...))
	if !out.Blocked || len(out.DocGaps) != 0 || !strings.Contains(strings.Join(out.Reasons, " "), "histórico da feature") {
		t.Fatalf("%+v", out)
	}
	// Histórico desatualizado em relação ao registrado: bloqueia.
	path := out.DocPath
	full := append(append([]string{}, withModuleDoc...), path)
	req := x.request(t, 1, full...)
	req.DocContent = doc + "\nlinha a mais"
	if out, _ = x.d.Deliver(ctx, req); !out.Blocked || !strings.Contains(strings.Join(out.Reasons, " "), "desatualizado") {
		t.Fatalf("%+v", out)
	}
	// Tudo certo: abre a Draft PR.
	req.DocContent = doc
	out, err = x.d.Deliver(ctx, req)
	if err != nil || out.Blocked || out.PR == nil || !out.PR.Draft || out.PR.Number != 101 || out.Reused {
		t.Fatalf("%+v %v", out, err)
	}
	if !strings.HasPrefix(path, "docs/features/") || !strings.Contains(doc, "schema: cria a tabela") || !strings.Contains(doc, "test: pass") || !strings.Contains(doc, "Os testes de schema passam") {
		t.Fatalf("o histórico traz commits, checks e critérios:\n%s", doc)
	}
	if len(x.gh.bodies) != 1 || !strings.Contains(x.gh.bodies[0], path) || !strings.Contains(x.gh.bodies[0], "schema: cria a tabela") {
		t.Fatalf("o corpo da PR: %v", x.gh.bodies)
	}
	if len(x.origin.calls) != 1 || !strings.HasPrefix(x.origin.calls[0], "feat:1|P1|https://github.com/org/app/pull/") {
		t.Fatalf("a origem recebe 'Aguardando merge': %v", x.origin.calls)
	}
}

// T15-02 — "Repetição encontra a mesma Draft PR."
func TestRepeatingTheDeliveryFindsTheSameDraftPR(t *testing.T) {
	x := newDlvEnv(t, "dlv-repeat")
	ctx := context.Background()
	x.green(t, shaB)
	path, _, _ := x.d.Document(ctx, x.p.proj, x.p.feature, shaB, "")
	files := []string{"internal/pedidos/pedidos.go", "internal/pedidos/pedidos_test.go", "internal/pedidos/doc.go", path}
	req := x.request(t, 1, files...)
	first, err := x.d.Deliver(ctx, req)
	if err != nil || first.PR == nil {
		t.Fatalf("%+v %v", first, err)
	}
	second, err := x.d.Deliver(ctx, req)
	if err != nil || second.PR == nil || second.PR.Number != first.PR.Number || !second.Reused || x.gh.creates != 1 || second.DeliveryID != first.DeliveryID {
		t.Fatalf("%+v %v creates=%d", second, err, x.gh.creates)
	}
	var n int
	_ = x.e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM deliveries WHERE feature_id = $1`, x.p.feature).Scan(&n)
	if n != 1 {
		t.Fatalf("uma entrega por feature e tentativa: %d", n)
	}
	// A resposta da criação se perde (a PR foi criada): a prova por busca encontra a mesma PR, sem duplicar.
	x.gh.loseNext = true
	req2 := x.request(t, 2, files...)
	third, err := x.d.Deliver(ctx, req2)
	if err != nil || third.PR == nil || x.gh.creates != 2 {
		t.Fatalf("%+v %v creates=%d", third, err, x.gh.creates)
	}
	again, _ := x.d.Deliver(ctx, req2)
	if again.PR.Number != third.PR.Number || x.gh.creates != 2 {
		t.Fatalf("%+v creates=%d", again, x.gh.creates)
	}
	// Uma PR já aberta para a branch (criada por fora ou por uma tentativa anterior) também é encontrada.
	branch3, _ := gitops.FeatureBranch(x.p.feature, 3)
	x.gh.mu.Lock()
	x.gh.prs = append(x.gh.prs, map[string]any{"number": 900, "html_url": "https://github.com/org/app/pull/9", "draft": true, "state": "open", "title": "já existia",
		"head": map[string]any{"ref": branch3}, "base": map[string]any{"ref": "main"}})
	x.gh.mu.Unlock()
	fourth, err := x.d.Deliver(ctx, x.request(t, 3, files...))
	if err != nil || fourth.PR == nil || fourth.PR.Number != 900 || x.gh.creates != 2 {
		t.Fatalf("%+v %v creates=%d", fourth, err, x.gh.creates)
	}
}

func TestOtherGatesAndTheBranchGuardAlsoBlock(t *testing.T) {
	x := newDlvEnv(t, "dlv-gates")
	ctx := context.Background()
	path, _, _ := x.d.Document(ctx, x.p.proj, x.p.feature, shaB, "")
	files := []string{"internal/pedidos/pedidos.go", "internal/pedidos/pedidos_test.go", "internal/pedidos/doc.go", path}
	// Checks da revisão sem evidência: bloqueia, mesmo com a documentação em dia.
	out, err := x.d.Deliver(ctx, x.request(t, 1, files...))
	if err != nil || !out.Blocked || !strings.Contains(strings.Join(out.Reasons, " "), "checks:") || x.gh.creates != 0 {
		t.Fatalf("%+v %v", out, err)
	}
	// Um gate arbitrário (ex.: E2E) com pendência também impede.
	x.green(t, shaB)
	req := x.request(t, 1, files...)
	req.Gates = append(req.Gates, delivery.GateFunc{Label: "e2e", Fn: func(context.Context) ([]string, error) { return []string{"a jornada checkout não tem E2E"}, nil }})
	if out, _ := x.d.Deliver(ctx, req); !out.Blocked || !strings.Contains(strings.Join(out.Reasons, " "), "e2e: a jornada checkout") {
		t.Fatalf("%+v", out)
	}
	// Branch que não é a da feature: recusada pelo publicador (nunca a base).
	bad := x.request(t, 1, files...)
	bad.Branch = "main"
	if _, err := x.d.Deliver(ctx, bad); !errors.Is(err, gitops.ErrBranchForbidden) {
		t.Fatalf("%v", err)
	}
	if x.gh.creates != 0 {
		t.Fatal("nenhuma PR")
	}
	if err := x.d.RecordCommit(ctx, x.p.proj, x.p.feature, "t", "curto", "b", "s"); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
	if _, err := x.d.Deliver(ctx, delivery.Request{ProjectID: x.p.proj, FeatureID: x.p.feature, Attempt: 1}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("%v", err)
	}
}

func fmtNum(v any) string {
	switch n := v.(type) {
	case int:
		return strconv.Itoa(n)
	case float64:
		return strconv.Itoa(int(n))
	}
	return ""
}

// setPR altera o estado de uma PR do GitHub de mentira (o humano fechou ou integrou).
func (g *fakeGH) setPR(number int, state string, merged bool, sha string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, p := range g.prs {
		if fmtNum(p["number"]) == strconv.Itoa(number) {
			p["state"], p["merged"], p["merge_commit_sha"] = state, merged, sha
		}
	}
}

// addComment simula um comentário na PR (conversa ou revisão).
func (g *fakeGH) addComment(kind string, pr int, id int64, login, body string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.comments == nil {
		g.comments = map[string][]map[string]any{}
	}
	key := kind + "/" + strconv.Itoa(pr)
	for i, c := range g.comments[key] {
		if c["id"] == id {
			g.comments[key][i]["body"] = body
			return
		}
	}
	g.comments[key] = append(g.comments[key], map[string]any{"id": id, "body": body, "user": map[string]any{"login": login}})
}
