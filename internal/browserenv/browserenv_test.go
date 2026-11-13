package browserenv

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestOriginPolicyBlocksProductionHostAndOtherTargets(t *testing.T) {
	app, err := ParseOrigin("http://webapp:3000")
	if err != nil {
		t.Fatal(err)
	}
	allowed := []Origin{app}
	for _, ok := range []string{"http://webapp:3000/", "http://webapp:3000/login?x=1"} {
		if err := CheckURL(allowed, ok); err != nil {
			t.Fatalf("%s: %v", ok, err)
		}
	}
	for _, bad := range []string{"https://app.producao.com/", "http://localhost:3000", "http://169.254.169.254/latest/meta-data", "http://host.docker.internal:8480", "http://webapp:3001", "https://webapp:3000", "http://user:pw@webapp:3000", "file:///etc/passwd", "http://other-exec-app:3000"} {
		if err := CheckURL(allowed, bad); !errors.Is(err, ErrForbiddenURL) {
			t.Fatalf("%s deveria ser bloqueado: %v", bad, err)
		}
	}
	for _, bad := range []string{"http://127.0.0.1:3000", "http://prod.example.com", "http://webapp:3000/path", "ftp://webapp", "http://u:p@webapp", "http://app:3000"} {
		if _, err := ParseOrigin(bad); err == nil {
			t.Fatalf("origem autorizada inválida aceita: %s", bad)
		}
	}
}

func TestSanitizeRemovesSecretsFromEvidence(t *testing.T) {
	got := Sanitize("Authorization: Bearer abcdef1234567890 e password=hunter2 e sk-abcdefghij12345")
	for _, leak := range []string{"abcdef1234567890", "hunter2", "sk-abcdefghij12345"} {
		if strings.Contains(got, leak) {
			t.Fatalf("vazou %q: %s", leak, got)
		}
	}
}

type fakeRT struct {
	calls   []string
	failRun bool
	listed  map[string]string // filtro → saída
}

func (f *fakeRT) Run(_ context.Context, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	if f.failRun && args[0] == "run" {
		return "", errors.New("imagem inexistente")
	}
	if args[0] == "inspect" {
		return "true", nil
	}
	for k, v := range f.listed {
		if strings.Contains(strings.Join(args, " "), k) {
			return v, nil
		}
	}
	return "", nil
}

type fakeDriver struct{ failNav bool }

func (d fakeDriver) Navigate(context.Context, string) error {
	if d.failNav {
		return errors.New("mcp indisponível")
	}
	return nil
}
func (fakeDriver) Interact(context.Context, string, map[string]string) (string, error) {
	return "ok", nil
}
func (fakeDriver) Screenshot(context.Context, string) ([]byte, error) { return []byte("png"), nil }
func (fakeDriver) ConsoleLogs(context.Context) ([]string, error) {
	return []string{"token=abc12345"}, nil
}

func spec() Spec {
	return Spec{ProjectID: "p1", ExecutionID: "e1", Image: "chromium-mcp:1", Allowed: []string{"http://webapp:3000"}}
}

func TestReviewApprovesWithEvidenceAndAlwaysCleansUp(t *testing.T) {
	rt := &fakeRT{}
	rv, ev := RunReview(context.Background(), rt, spec(), "rev1", time.Second, func(*Env) Driver { return fakeDriver{} }, func(tl *Tools) (bool, error) {
		if err := tl.Navigate(context.Background(), "http://webapp:3000/"); err != nil {
			return false, err
		}
		_ = tl.Screenshot(context.Background(), "home")
		_ = tl.Logs(context.Background())
		if err := tl.Navigate(context.Background(), "https://prod.example.com"); err == nil {
			t.Error("produção deve ser bloqueada")
		}
		return true, nil
	})
	if !rv.Approved || rv.Pending || !rv.Independent || rv.Revision != "rev1" {
		t.Fatalf("%+v", rv)
	}
	var shots, blocked int
	for _, it := range ev {
		if it.Kind == "screenshot" {
			shots++
		}
		if it.Kind == "error" && strings.Contains(it.Name, "bloqueada") {
			blocked++
		}
		if strings.Contains(it.Text, "abc12345") {
			t.Fatalf("segredo vazou na evidência: %+v", it)
		}
	}
	if shots != 1 || blocked != 1 {
		t.Fatalf("%+v", ev)
	}
	joined := strings.Join(rt.calls, "\n")
	if !strings.Contains(joined, "network create --internal") || !strings.Contains(joined, "--cap-drop ALL") || strings.Contains(joined, " -v ") || !strings.Contains(joined, "rm -f -v") && !strings.Contains(joined, "ps -aq") {
		t.Fatalf("rede interna, sem montagens do host, recolhimento ao final:\n%s", joined)
	}
	if !strings.Contains(joined, "label=lf.exec=e1") {
		t.Fatalf("a limpeza é por execução:\n%s", joined)
	}
}

func TestBrowserFailureLeavesGatePendingAndStillCleans(t *testing.T) {
	rt := &fakeRT{failRun: true}
	rv, _ := RunReview(context.Background(), rt, spec(), "rev1", time.Second, func(*Env) Driver { return fakeDriver{} }, func(*Tools) (bool, error) { return true, nil })
	if rv.Approved || !rv.Pending {
		t.Fatalf("falha de provisionamento não aprova: %+v", rv)
	}
	if !strings.Contains(strings.Join(rt.calls, "\n"), "network ls") {
		t.Fatalf("o que foi criado antes da falha é recolhido: %v", rt.calls)
	}
	rt2 := &fakeRT{}
	rv, ev := RunReview(context.Background(), rt2, spec(), "rev1", time.Second, func(*Env) Driver { return fakeDriver{failNav: true} }, func(tl *Tools) (bool, error) {
		return true, tl.Navigate(context.Background(), "http://webapp:3000/")
	})
	if rv.Approved || !rv.Pending || len(ev) == 0 {
		t.Fatalf("falha do MCP deixa o gate pendente: %+v", rv)
	}
	if !strings.Contains(strings.Join(rt2.calls, "\n"), "network ls") {
		t.Fatal("a execução com falha também é recolhida")
	}
}

func TestCollectOrphansRemovesOnlyDeadExecutions(t *testing.T) {
	rt := &fakeRT{listed: map[string]string{"label=lf.kind=browser": "dead1\nlive1\ndead1"}}
	got, err := CollectOrphans(context.Background(), rt, map[string]bool{"live1": true})
	if err != nil || len(got) != 1 || got[0] != "dead1" {
		t.Fatalf("%v %v", got, err)
	}
	for _, c := range rt.calls {
		if strings.Contains(c, "lf.exec=live1") && !strings.Contains(c, "label=lf.kind=browser") {
			t.Fatalf("a execução viva não pode ser tocada: %s", c)
		}
	}
}
