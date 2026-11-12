package integrations

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func jiraFake(t *testing.T) *fake {
	issues := map[string]string{
		"10001": `{"id":"10001","key":"ABC-1","fields":{"summary":"Cadastro de clientes","issuetype":{"name":"Story","subtask":false},"description":{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"Permitir cadastrar clientes."}]}]},"status":{"name":"To Do","statusCategory":{"key":"new"}},"priority":{"name":"High"},"labels":["lf-ready","backend"],"updated":"2026-11-10T12:00:00.000+0000","issuelinks":[]}}`,
		"10002": `{"id":"10002","key":"ABC-2","fields":{"summary":"Validar CPF","issuetype":{"name":"Sub-task","subtask":true},"status":{"name":"In Progress","statusCategory":{"key":"indeterminate"}},"parent":{"id":"10001","key":"ABC-1"},"labels":[],"updated":"2026-11-10T13:00:00.000+0000"}}`,
		"10003": `{"id":"10003","key":"ABC-3","fields":{"summary":"Relatório","issuetype":{"name":"Task","subtask":false},"status":{"name":"Done","statusCategory":{"key":"done"}},"labels":[],"updated":"2026-11-11T08:00:00.000+0000"}}`,
		"10004": `{"id":"10004","key":"ABC-4","fields":{"summary":"Nova subtarefa","issuetype":{"name":"Sub-task","subtask":true},"status":{"name":"To Do","statusCategory":{"key":"new"}},"parent":{"id":"10001","key":"ABC-1"},"labels":[],"updated":"2026-11-12T08:00:00.000+0000"}}`,
	}
	byKey := map[string]string{"ABC-1": "10001", "ABC-2": "10002", "ABC-3": "10003", "ABC-4": "10004"}
	w := func(rw http.ResponseWriter, status int, body string) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(status)
		io.WriteString(rw, body)
	}
	return newFake(t, func(rw http.ResponseWriter, c call) {
		p := strings.TrimPrefix(c.Path, "/rest/api/3")
		switch {
		case c.Method == "GET" && p == "/myself":
			w(rw, 200, `{"accountId":"acc-1","displayName":"Fábrica Bot","emailAddress":"bot@example.com"}`)
		case c.Method == "GET" && p == "/search/jql":
			if !strings.Contains(c.Query, "jql=project+%3D+ABC+AND+labels+%3D+%22lf-ready%22+AND+statusCategory+%21%3D+Done+ORDER+BY+created+ASC%2C+key+ASC") {
				w(rw, 400, `{"errorMessages":["jql inesperada"]}`)
				return
			}
			if strings.Contains(c.Query, "nextPageToken=tok-2") {
				w(rw, 200, fixture(t, "jira_search_page2.json"))
			} else {
				w(rw, 200, fixture(t, "jira_search_page1.json"))
			}
		case c.Method == "GET" && strings.HasPrefix(p, "/issue/") && !strings.Contains(strings.TrimPrefix(p, "/issue/"), "/"):
			ref := strings.TrimPrefix(p, "/issue/")
			if id, ok := byKey[ref]; ok {
				ref = id
			}
			if body, ok := issues[ref]; ok {
				w(rw, 200, body)
			} else {
				w(rw, 404, `{"errorMessages":["Issue does not exist or you do not have permission to see it."]}`)
			}
		case c.Method == "GET" && strings.HasSuffix(p, "/transitions"):
			w(rw, 200, `{"transitions":[{"id":"11","name":"Start","to":{"name":"In Progress","statusCategory":{"key":"indeterminate"}}},{"id":"31","name":"Finish","to":{"name":"Done","statusCategory":{"key":"done"}}}]}`)
		case c.Method == "POST" && strings.HasSuffix(p, "/transitions"):
			w(rw, 204, "")
		case c.Method == "GET" && p == "/project/ABC":
			w(rw, 200, `{"key":"ABC","issueTypes":[{"id":"10000","name":"Story","subtask":false},{"id":"10001","name":"Task","subtask":false},{"id":"10002","name":"Sub-task","subtask":true}]}`)
		case c.Method == "GET" && p == "/project/ABC/statuses":
			w(rw, 200, `[{"name":"Story","statuses":[{"name":"To Do","statusCategory":{"key":"new"}},{"name":"In Progress","statusCategory":{"key":"indeterminate"}},{"name":"Done","statusCategory":{"key":"done"}}]},{"name":"Task","statuses":[{"name":"To Do","statusCategory":{"key":"new"}},{"name":"Done","statusCategory":{"key":"done"}}]}]`)
		case c.Method == "GET" && p == "/field":
			w(rw, 200, `[{"id":"summary","name":"Summary","custom":false},{"id":"customfield_10016","name":"Story point estimate","custom":true}]`)
		case c.Method == "GET" && p == "/priority":
			w(rw, 200, `[{"name":"Highest"},{"name":"High"},{"name":"Medium"},{"name":"Low"}]`)
		case c.Method == "POST" && p == "/issue":
			w(rw, 201, `{"id":"10004","key":"ABC-4"}`)
		case c.Method == "POST" && p == "/issueLink":
			w(rw, 201, "")
		case c.Method == "POST" && strings.HasSuffix(p, "/remotelink"):
			w(rw, 201, `{"id":10000}`)
		default:
			w(rw, 404, `{"errorMessages":["rota desconhecida"]}`)
		}
	})
}

func newJiraHarness(t *testing.T) harness {
	f := jiraFake(t)
	tr, err := NewTracker(Config{Kind: "jira", Secret: security.NewSecret("bot@example.com:" + secretSentinel), BaseURL: f.URL})
	if err != nil {
		t.Fatal(err)
	}
	return harness{
		name: "jira", tracker: tr, fake: f, container: "ABC",
		story: "10001", subtask: "10002", dependent: "10003", missing: "ABC-999", okTrans: "11", badTrans: "99",
		wantSubtaskBody: []string{`"parent":{"key":"ABC-1"}`, `"issuetype":{"id":"10002"}`, `"summary":"Nova subtarefa"`, `"type":"doc"`},
		validatesLabel:  true,
		depPath:         "/rest/api/3/issueLink",
		wantDepBody:     []string{`"name":"Blocks"`, `"inwardIssue":{"key":"10003"}`, `"outwardIssue":{"key":"10001"}`},
		linkPath:        "/rest/api/3/issue/10003/remotelink",
		wantLinkBody:    []string{`"globalId":"lf-link:https://github.com/o/r/pull/7"`, `"title":"Draft PR #7"`},
	}
}

func TestJiraContract(t *testing.T) { runContract(t, newJiraHarness(t)) }

func TestJiraUsesBasicAuthAndOnlyAtlassianTargets(t *testing.T) {
	h := newJiraHarness(t)
	if _, err := h.tracker.Identity(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := h.fake.rec.calls[0].Auth; !strings.HasPrefix(got, "Basic ") {
		t.Fatalf("Jira usa Basic: %q", got)
	}
	secret := security.NewSecret("a@b.c:tok")
	for _, target := range []string{"http://x.atlassian.net", "https://exemplo.com", "https://169.254.169.254", "https://atlassian.net.evil.com", "", "::"} {
		if _, err := NewTracker(Config{Kind: "jira", Target: target, Secret: secret}); err == nil {
			t.Errorf("alvo %q deveria ser recusado", target)
		}
	}
	if _, err := NewTracker(Config{Kind: "jira", Target: "https://equipe.atlassian.net", Secret: secret}); err != nil {
		t.Errorf("alvo válido recusado: %v", err)
	}
	for _, bad := range []string{"", "sem-dois-pontos", ":tok", "a@b.c:"} {
		if _, err := NewTracker(Config{Kind: "jira", Target: "https://equipe.atlassian.net", Secret: security.NewSecret(bad)}); err == nil {
			t.Errorf("segredo %q deveria ser recusado", bad)
		}
	}
}

func TestJiraDoesNotFollowRedirects(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("redirecionamento seguido") }))
	defer other.Close()
	f := newFake(t, func(rw http.ResponseWriter, c call) { http.Redirect(rw, &http.Request{}, other.URL, 302) })
	tr, _ := NewTracker(Config{Kind: "jira", Secret: security.NewSecret("a@b.c:tok"), BaseURL: f.URL})
	if _, err := tr.Identity(t.Context()); err == nil {
		t.Fatal("302 não é sucesso")
	}
}

func TestADFTextIsExtracted(t *testing.T) {
	raw := []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"a"},{"type":"text","text":"b"}]},{"type":"paragraph","content":[{"type":"text","text":"c"}]}]}`)
	if got := adfText(raw); got != "ab\nc" {
		t.Fatalf("%q", got)
	}
	if adfText(nil) != "" || adfText([]byte("null")) != "" || adfText([]byte("{")) != "" {
		t.Fatal("entradas vazias ou inválidas devolvem vazio")
	}
}
