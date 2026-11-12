package gateways

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
)

const testToken = "TOKEN-GW-4f2a-nao-pode-vazar"

// fakeGateway implementa /chat/completions e responde por roteiro.
type fakeGateway struct {
	*httptest.Server
	mu     sync.Mutex
	reqs   []map[string]any
	auth   []string
	script []func(req map[string]any) (int, any)
}

func newFake(t *testing.T, script ...func(map[string]any) (int, any)) *fakeGateway {
	f := &fakeGateway{script: script}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]any
		_ = json.NewDecoder(r.Body).Decode(&req)
		f.mu.Lock()
		i := len(f.reqs)
		f.reqs = append(f.reqs, req)
		f.auth = append(f.auth, r.Header.Get("Authorization"))
		f.mu.Unlock()
		if r.URL.Path != "/v1/chat/completions" || i >= len(f.script) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		status, body := f.script[i](req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeGateway) base() string { return f.URL + "/v1" }

func toolCalls(calls ...[2]string) func(map[string]any) (int, any) {
	return func(map[string]any) (int, any) {
		var tcs []map[string]any
		for i, c := range calls {
			tcs = append(tcs, map[string]any{"id": fmt.Sprintf("c%d", i), "type": "function", "function": map[string]any{"name": c[0], "arguments": c[1]}})
		}
		return 200, map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": nil, "tool_calls": tcs}, "finish_reason": "tool_calls"}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5}}
	}
}

func say(text string) func(map[string]any) (int, any) {
	return func(map[string]any) (int, any) {
		return 200, map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": text}, "finish_reason": "stop"}}, "usage": map[string]any{"prompt_tokens": 20, "completion_tokens": 7}}
	}
}

func client(t *testing.T, kind string, f *fakeGateway) *Client {
	c, err := New(kind, security.NewSecret(testToken), f.base(), true)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func eligibleModel(gateway string) models.Model {
	temp := 0.2
	return models.Model{Provider: "api", Gateway: gateway, Name: "vendor/model-x", SupportsTools: true, State: "active", Params: models.Params{Temperature: &temp}}
}

type countExec struct {
	n    int
	last string
}

func (c *countExec) Call(_ context.Context, name string, _ map[string]any) (map[string]any, error) {
	c.n++
	c.last = name
	return map[string]any{"ok": true}, nil
}

func TestNewRefusesUnsafeDestinationsAndMissingCredential(t *testing.T) {
	tok := security.NewSecret("x")
	for name, u := range map[string]string{"http": "http://exemplo.com/v1", "loopback": "https://127.0.0.1/v1", "metadados": "https://169.254.169.254/v1", "credenciais": "https://u:p@exemplo.com/v1", "query": "https://exemplo.com/v1?x=1"} {
		if _, err := New(models.GatewayOpenRouter, tok, u, false); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	if _, err := New("outro", tok, "", false); err == nil {
		t.Error("gateway desconhecido")
	}
	if _, err := New(models.GatewayVercel, security.Secret{}, "", false); err == nil {
		t.Error("sem credencial")
	}
	for kind, want := range map[string]string{models.GatewayOpenRouter: "https://openrouter.ai/api/v1", models.GatewayVercel: "https://ai-gateway.vercel.sh/v1"} {
		c, err := New(kind, tok, "", false)
		if err != nil || c.BaseURL != want {
			t.Errorf("%s: %v %+v", kind, err, c)
		}
	}
}

func TestChatSendsCredentialParamsAndNormalizesToolCalls(t *testing.T) {
	f := newFake(t, toolCalls([2]string{"read_file", `{"path":"a.txt"}`}, [2]string{"read_file", `não é json`}))
	c := client(t, models.GatewayOpenRouter, f)
	temp, mt := 0.3, 50
	resp, err := c.Chat(t.Context(), ChatRequest{Model: "vendor/m", Messages: []Message{{Role: "user", Content: "oi"}},
		Tools: []ToolSpec{{Name: "read_file", Description: "d", Parameters: map[string]any{"type": "object"}}}, Params: models.Params{Temperature: &temp, MaxTokens: &mt}})
	if err != nil || len(resp.ToolCalls) != 2 || resp.Usage.PromptTokens != 10 {
		t.Fatalf("%+v %v", resp, err)
	}
	if resp.ToolCalls[0].Args["path"] != "a.txt" || resp.ToolCalls[0].BadArgs || !resp.ToolCalls[1].BadArgs {
		t.Fatalf("argumentos: %+v", resp.ToolCalls)
	}
	if f.auth[0] != "Bearer "+testToken {
		t.Fatalf("credencial: %q", f.auth[0])
	}
	r := f.reqs[0]
	if r["model"] != "vendor/m" || r["temperature"] != 0.3 || r["max_tokens"] != float64(50) || r["top_p"] != nil {
		t.Fatalf("pedido: %v", r)
	}
}

func TestChatErrorsAreFixedAndNeverEchoRemoteBodyOrToken(t *testing.T) {
	leak := func(map[string]any) (int, any) {
		return 401, map[string]any{"error": "token " + testToken + " inválido em https://interno.exemplo"}
	}
	for status, kind := range map[int]string{401: "auth", 403: "auth", 429: "rate_limit", 500: "unavailable", 400: "invalid"} {
		st := status
		f := newFake(t, func(map[string]any) (int, any) { return st, map[string]any{"error": testToken + " detalhe interno"} })
		_, err := client(t, models.GatewayVercel, f).Chat(t.Context(), ChatRequest{Model: "m", Messages: []Message{{Role: "user", Content: "x"}}})
		var ge *Error
		if !errors.As(err, &ge) || ge.Kind != kind || strings.Contains(err.Error(), testToken) || strings.Contains(err.Error(), "detalhe") || strings.Contains(err.Error(), "http") {
			t.Errorf("%d: %v", status, err)
		}
	}
	f := newFake(t, leak)
	_, err := client(t, models.GatewayVercel, f).Chat(t.Context(), ChatRequest{Model: "m"})
	if err == nil || strings.Contains(err.Error(), testToken) {
		t.Fatalf("%v", err)
	}
	// Corpo fora do formato e redirecionamento.
	f = newFake(t, func(map[string]any) (int, any) { return 200, map[string]any{"choices": []any{}} })
	if _, err := client(t, models.GatewayVercel, f).Chat(t.Context(), ChatRequest{Model: "m"}); err == nil || err.(*Error).Kind != "protocol" {
		t.Fatalf("sem choices: %v", err)
	}
	redir := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://127.0.0.1:1/x", http.StatusFound)
	}))
	defer redir.Close()
	rc, _ := New(models.GatewayOpenRouter, security.NewSecret(testToken), redir.URL+"/v1", true)
	if _, err := rc.Chat(t.Context(), ChatRequest{Model: "m"}); err == nil {
		t.Fatal("redirecionamento não é seguido")
	}
}

func TestEligibilityIsCheckedBeforeAnyRequest(t *testing.T) {
	f := newFake(t, say("não deveria chegar"))
	c := client(t, models.GatewayOpenRouter, f)
	specs := []ToolSpec{{Name: "read_file"}}
	cases := map[string]models.Model{
		"desativado": func() models.Model { m := eligibleModel("openrouter"); m.State = "disabled"; return m }(),
		"sem tools":  func() models.Model { m := eligibleModel("openrouter"); m.SupportsTools = false; return m }(),
		"cli":        func() models.Model { m := eligibleModel(""); m.Provider = "claude"; return m }(),
	}
	for name, m := range cases {
		if _, err := Run(t.Context(), c, RunConfig{Model: m, Prompt: "x", Tools: specs, Exec: &countExec{}}); !errors.Is(err, ErrModelNotEligible) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if len(f.reqs) != 0 {
		t.Fatalf("nenhum pedido pode sair: %d", len(f.reqs))
	}
	// Sem ferramentas, modelo sem suporte a tools é elegível.
	m := eligibleModel("openrouter")
	m.SupportsTools = false
	if res, err := Run(t.Context(), c, RunConfig{Model: m, Prompt: "x"}); err != nil || res.Text != "não deveria chegar" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestLoopOnlyRunsToolsTheAgentHasAndFeedsResultsBack(t *testing.T) {
	f := newFake(t,
		toolCalls([2]string{"read_file", `{"path":"a"}`}, [2]string{"delete_everything", `{}`}, [2]string{"read_file", `quebrado`}),
		say("pronto"))
	ex := &countExec{}
	res, err := Run(t.Context(), client(t, models.GatewayOpenRouter, f), RunConfig{
		Model: eligibleModel("openrouter"), System: "seja breve", Prompt: "faça", Tools: []ToolSpec{{Name: "read_file"}}, Exec: ex})
	if err != nil || res.Text != "pronto" || res.Turns != 2 || res.Usage.PromptTokens != 30 || res.Usage.CompletionTokens != 12 {
		t.Fatalf("%+v %v", res, err)
	}
	if ex.n != 1 || ex.last != "read_file" {
		t.Fatalf("só a ferramenta do agente executa: %d %s", ex.n, ex.last)
	}
	got := []string{}
	for _, c := range res.Calls {
		got = append(got, c.Name+":"+c.Outcome)
	}
	if strings.Join(got, ",") != "read_file:ok,delete_everything:refused,read_file:refused" {
		t.Fatalf("%v", got)
	}
	// Segunda rodada leva os resultados de volta, sem ferramentas fora do agente anunciadas.
	second := f.reqs[1]["messages"].([]any)
	if len(second) != 6 { // system, user, assistant, 3 tool
		t.Fatalf("mensagens: %d", len(second))
	}
	first := f.reqs[0]["tools"].([]any)
	if len(first) != 1 || first[0].(map[string]any)["function"].(map[string]any)["name"] != "read_file" {
		t.Fatalf("anúncio: %v", first)
	}
	if f.reqs[0]["temperature"] != 0.2 {
		t.Fatalf("parâmetros do modelo: %v", f.reqs[0])
	}
}

func TestLoopLimitsAndRedaction(t *testing.T) {
	forever := func(map[string]any) (int, any) { return toolCalls([2]string{"read_file", `{}`})(nil) }
	f := newFake(t, forever, forever, forever)
	if _, err := Run(t.Context(), client(t, models.GatewayOpenRouter, f), RunConfig{Model: eligibleModel("openrouter"), Prompt: "x", Tools: []ToolSpec{{Name: "read_file"}}, Exec: &countExec{}, MaxTurns: 3}); !errors.Is(err, ErrTooManyTurns) {
		t.Fatalf("%v", err)
	}
	f = newFake(t, toolCalls([2]string{"read_file", `{}`}, [2]string{"read_file", `{}`}, [2]string{"read_file", `{}`}))
	if _, err := Run(t.Context(), client(t, models.GatewayOpenRouter, f), RunConfig{Model: eligibleModel("openrouter"), Prompt: "x", Tools: []ToolSpec{{Name: "read_file"}}, Exec: &countExec{}, MaxToolCalls: 2}); !errors.Is(err, ErrTooManyCalls) {
		t.Fatalf("%v", err)
	}
	// O erro de uma ferramenta volta ao modelo redigido e limitado.
	f = newFake(t, toolCalls([2]string{"read_file", `{}`}), say("ok"))
	_, err := Run(t.Context(), client(t, models.GatewayOpenRouter, f), RunConfig{Model: eligibleModel("openrouter"), Prompt: "x", Tools: []ToolSpec{{Name: "read_file"}},
		Exec:   failExec{errors.New("falhou com " + testToken)},
		Redact: func(s string) string { return strings.ReplaceAll(s, testToken, "[REDACTED]") }})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(f.reqs[1])
	if strings.Contains(string(b), testToken) || !strings.Contains(string(b), "[REDACTED]") {
		t.Fatalf("erro da ferramenta deve voltar redigido: %s", b)
	}
}

type failExec struct{ err error }

func (f failExec) Call(context.Context, string, map[string]any) (map[string]any, error) {
	return nil, f.err
}

func sandbox(t *testing.T) *execution.Sandbox {
	t.Helper()
	d := &execution.Docker{}
	if !d.Available(t.Context()) {
		t.Skip("Docker indisponível")
	}
	sb, err := d.Provision(t.Context(), execution.Spec{ProjectID: "p", FeatureID: "f", Profile: execution.Profiles["base"]})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sb.Destroy(context.Background()) })
	return sb
}

type countRec struct {
	mu sync.Mutex
	n  int
}

func (c *countRec) Record(context.Context, policy.Action, policy.Decision) error {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return nil
}

// "Cada gateway roda edição e teste com modelo elegível": o modelo escreve um
// arquivo, edita e roda o teste no sandbox, só por ferramentas do catálogo.
func TestEachGatewayRunsEditAndTestWithAnEligibleModel(t *testing.T) {
	sb := sandbox(t)
	runTests := tools.Tool{Name: "run_tests", Kind: tools.KindScript, Description: "Roda o teste do projeto.", Permissions: []string{tools.PermExec},
		Limits: tools.Limits{TimeoutSeconds: 20, MaxOutputBytes: 1 << 16}, Command: `grep -q "valor=2" calc.txt && echo "{\"resultado\":\"passou\"}"`,
		Output: []tools.Field{{Name: "resultado", Type: tools.TypeString, Required: true}}}
	reg := tools.NewRegistry(func(_ context.Context, name string) (tools.Tool, error) {
		if name == "run_tests" {
			return runTests, nil
		}
		return tools.Tool{}, errors.New("não")
	})
	for _, gw := range []string{models.GatewayOpenRouter, models.GatewayVercel} {
		t.Run(gw, func(t *testing.T) {
			rec := &countRec{}
			env := tools.Env{Sandbox: sb, Guard: &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{}), Rec: rec}}
			granted := []string{"write_file", "edit_file", "run_tests"}
			ex := RegistryExecutor{Registry: reg, Env: env, Grant: tools.Grant{Tools: granted, Permissions: []string{tools.PermRead, tools.PermWrite, tools.PermExec}}}
			var specs []ToolSpec
			for _, n := range granted {
				tl, err := reg.Get(t.Context(), n)
				if err != nil {
					t.Fatal(err)
				}
				specs = append(specs, SpecOf(tl))
			}
			file := "calc-" + gw + ".txt"
			_ = file
			f := newFake(t,
				toolCalls([2]string{"write_file", `{"path":"calc.txt","content":"valor=1\n"}`}),
				toolCalls([2]string{"run_tests", `{}`}),
				toolCalls([2]string{"edit_file", `{"path":"calc.txt","old":"valor=1","new":"valor=2"}`}, [2]string{"run_tests", `{}`}),
				say("corrigi o valor e o teste passou"))
			res, err := Run(t.Context(), client(t, gw, f), RunConfig{Model: eligibleModel(gw), System: "corrija o teste", Prompt: "faça o teste passar", Tools: specs, Exec: ex})
			if err != nil || res.Text != "corrigi o valor e o teste passou" || res.Turns != 4 {
				t.Fatalf("%+v %v", res, err)
			}
			outcomes := []string{}
			for _, c := range res.Calls {
				outcomes = append(outcomes, c.Name+":"+c.Outcome)
			}
			// O primeiro teste falha (valor=1): a falha volta ao modelo como erro da ferramenta.
			if strings.Join(outcomes, ",") != "write_file:ok,run_tests:error,edit_file:ok,run_tests:ok" {
				t.Fatalf("%v", outcomes)
			}
			if rec.n < 4 {
				t.Fatalf("toda chamada passa pela política: %d", rec.n)
			}
			for _, a := range f.auth {
				if a != "Bearer "+testToken {
					t.Fatalf("credencial: %q", a)
				}
			}
		})
	}
}

func TestRegistryExecutorStillEnforcesGrantAndScope(t *testing.T) {
	sb := sandbox(t)
	env := tools.Env{Sandbox: sb, Guard: &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{}), Rec: &countRec{}}}
	// O modelo recebeu o anúncio de write_file, mas o Grant só cobre read_file: nada é escrito.
	ex := RegistryExecutor{Registry: tools.NewRegistry(nil), Env: env, Grant: tools.Grant{Tools: []string{"read_file"}, Permissions: []string{tools.PermRead}}}
	f := newFake(t, toolCalls([2]string{"write_file", `{"path":"x.txt","content":"x"}`}), say("fim"))
	wf, _ := tools.NewRegistry(nil).Get(t.Context(), "write_file")
	res, err := Run(t.Context(), client(t, models.GatewayOpenRouter, f), RunConfig{Model: eligibleModel("openrouter"), Prompt: "x", Tools: []ToolSpec{SpecOf(wf)}, Exec: ex})
	if err != nil || res.Calls[0].Outcome != "error" {
		t.Fatalf("%+v %v", res, err)
	}
	if _, err := tools.NewRegistry(nil).Call(t.Context(), env, "read_file", map[string]any{"path": "x.txt"}, tools.Grant{Tools: []string{"read_file"}, Permissions: []string{tools.PermRead}}); err == nil {
		t.Fatal("o arquivo não deveria existir")
	}
}

// T11-02 — modo API: as ferramentas autorizadas funcionam e as fugas (traversal,
// symlink, socket do runtime) falham, e o modelo só recebe o erro.
func TestAPIModeAuthorizedToolsWorkAndEscapesFail(t *testing.T) {
	sb := sandbox(t)
	// Um repositório hostil deixou links no workspace (preparo direto, fora das ferramentas).
	res, err := sb.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", "cd /workspace && ln -s /etc/passwd link_passwd && ln -s /var/run/docker.sock link_sock && ln -s / link_raiz"}})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	reg := tools.NewRegistry(nil)
	env := tools.Env{Sandbox: sb, Guard: &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{}), Rec: &countRec{}}}
	granted := []string{"read_file", "write_file", "list_dir"}
	ex := RegistryExecutor{Registry: reg, Env: env, Identity: tools.Identity{Agent: "ag", Run: "r1"},
		Grant: tools.Grant{Tools: granted, Permissions: []string{tools.PermRead, tools.PermWrite}}}
	var specs []ToolSpec
	for _, n := range granted {
		tl, _ := reg.Get(t.Context(), n)
		specs = append(specs, SpecOf(tl))
	}
	f := newFake(t,
		toolCalls(
			[2]string{"write_file", `{"path":"feature/a.txt","content":"ok\n"}`},
			[2]string{"read_file", `{"path":"feature/a.txt"}`},
			[2]string{"read_file", `{"path":"../etc/passwd"}`},
			[2]string{"read_file", `{"path":"/var/run/docker.sock"}`},
			[2]string{"read_file", `{"path":"link_passwd"}`},
			[2]string{"read_file", `{"path":"link_sock"}`},
			[2]string{"list_dir", `{"path":"link_raiz"}`},
			[2]string{"write_file", `{"path":"link_raiz/tmp/escapou.txt","content":"x"}`},
		),
		say("terminei"))
	out, err := Run(t.Context(), client(t, models.GatewayOpenRouter, f), RunConfig{Model: eligibleModel(models.GatewayOpenRouter), Prompt: "x", Tools: specs, Exec: ex})
	if err != nil || out.Text != "terminei" {
		t.Fatalf("%+v %v", out, err)
	}
	got := []string{}
	for _, c := range out.Calls {
		got = append(got, c.Outcome)
	}
	if strings.Join(got, ",") != "ok,ok,error,error,error,error,error,error" {
		t.Fatalf("%v", got)
	}
	// O que voltou ao modelo não contém conteúdo de fora do workspace.
	b, _ := json.Marshal(f.reqs[1])
	if strings.Contains(string(b), "root:x:0") || !strings.Contains(string(b), "sai do workspace") {
		t.Fatalf("o modelo só recebe o erro: %s", b)
	}
	if r, _ := sb.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", "test ! -e /tmp/escapou.txt"}}); r.ExitCode != 0 {
		t.Fatal("nada foi gravado fora do workspace")
	}
}
