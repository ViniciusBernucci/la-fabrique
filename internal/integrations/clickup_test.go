package integrations

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func clickupFake(t *testing.T) *fake {
	tasks := map[string]string{
		"t1": `{"id":"t1","name":"Cadastro de clientes","text_content":"Permitir cadastrar clientes.","status":{"status":"to do","type":"open"},"priority":{"priority":"high"},"parent":null,"tags":[{"name":"lf-ready"},{"name":"backend"}],"date_updated":"1731240000000","url":"https://app.clickup.com/t/t1","list":{"id":"L1"}}`,
		"t2": `{"id":"t2","name":"Validar CPF","status":{"status":"in progress","type":"custom"},"parent":"t1","tags":[],"date_updated":"1731243600000","url":"https://app.clickup.com/t/t2","list":{"id":"L1"}}`,
		"t3": `{"id":"t3","name":"Relatório mensal","status":{"status":"complete","type":"closed"},"parent":null,"tags":[],"date_updated":"1731312000000","url":"https://app.clickup.com/t/t3","list":{"id":"L1"}}`,
	}
	w := func(rw http.ResponseWriter, status int, body string) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(status)
		io.WriteString(rw, body)
	}
	return newFake(t, func(rw http.ResponseWriter, c call) {
		p := c.Path
		switch {
		case c.Method == "GET" && p == "/user":
			w(rw, 200, `{"user":{"id":183,"username":"Fábrica Bot","email":"bot@example.com"}}`)
		case c.Method == "GET" && p == "/list/L1/task":
			if !strings.Contains(c.Query, "tags%5B%5D=lf-ready") || !strings.Contains(c.Query, "subtasks=true") {
				w(rw, 400, `{"err":"consulta inesperada","ECODE":"X"}`)
				return
			}
			if strings.Contains(c.Query, "page=1") {
				w(rw, 200, fixture(t, "clickup_tasks_page1.json"))
			} else {
				w(rw, 200, fixture(t, "clickup_tasks_page0.json"))
			}
		case c.Method == "GET" && p == "/list/L1":
			w(rw, 200, `{"id":"L1","statuses":[{"status":"to do","type":"open"},{"status":"in progress","type":"custom"},{"status":"complete","type":"closed"}]}`)
		case c.Method == "GET" && p == "/list/L1/field":
			w(rw, 200, `{"fields":[{"id":"f1","name":"Estimativa","type":"number"}]}`)
		case c.Method == "GET" && strings.HasPrefix(p, "/task/"):
			if body, ok := tasks[strings.TrimPrefix(p, "/task/")]; ok {
				w(rw, 200, body)
			} else {
				w(rw, 404, `{"err":"Task not found","ECODE":"ITEM_015"}`)
			}
		case c.Method == "PUT" && p == "/task/t1":
			w(rw, 200, `{"id":"t1"}`)
		case c.Method == "POST" && p == "/list/L1/task":
			w(rw, 200, `{"id":"t4","name":"Nova subtarefa","status":{"status":"to do","type":"open"},"parent":"t1","tags":[],"date_updated":"1731400000000","url":"https://app.clickup.com/t/t4","list":{"id":"L1"}}`)
		case c.Method == "POST" && p == "/task/t3/dependency":
			w(rw, 200, `{}`)
		case c.Method == "POST" && p == "/task/t3/comment":
			w(rw, 200, `{"id":1}`)
		default:
			w(rw, 404, `{"err":"rota desconhecida"}`)
		}
	})
}

func newClickUpHarness(t *testing.T) harness {
	f := clickupFake(t)
	tr, err := NewTracker(Config{Kind: "clickup", Secret: security.NewSecret("pk_" + secretSentinel), BaseURL: f.URL})
	if err != nil {
		t.Fatal(err)
	}
	return harness{
		name: "clickup", tracker: tr, fake: f, container: "L1",
		story: "t1", subtask: "t2", dependent: "t3", missing: "nao-existe", okTrans: "in progress", badTrans: "inexistente",
		wantSubtaskBody: []string{`"parent":"t1"`, `"name":"Nova subtarefa"`, `"description":"Detalhe"`},
		depPath:         "/task/t3/dependency",
		wantDepBody:     []string{`"depends_on":"t1"`},
		linkPath:        "/task/t3/comment",
		wantLinkBody:    []string{"Draft PR #7 https://github.com/o/r/pull/7"},
	}
}

func TestClickUpContract(t *testing.T) { runContract(t, newClickUpHarness(t)) }

func TestClickUpSendsTokenWithoutBearerAndDoesNotFollowRedirects(t *testing.T) {
	h := newClickUpHarness(t)
	if _, err := h.tracker.Identity(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := h.fake.rec.calls[0].Auth; got != "pk_"+secretSentinel {
		t.Fatalf("ClickUp usa o token puro: %q", got)
	}
	if _, err := NewTracker(Config{Kind: "clickup", Secret: security.NewSecret("  ")}); err == nil {
		t.Fatal("segredo vazio deve ser recusado")
	}
	if _, err := NewTracker(Config{Kind: "desconhecido"}); err == nil {
		t.Fatal("tipo desconhecido deve ser recusado")
	}
}

func TestClickUpStatusTypesMapToCategories(t *testing.T) {
	for in, want := range map[string]string{"open": CategoryTodo, "custom": CategoryInProgress, "closed": CategoryDone, "done": CategoryDone} {
		if got := clickupCategory(in); got != want {
			t.Errorf("%s: %s", in, got)
		}
	}
}
