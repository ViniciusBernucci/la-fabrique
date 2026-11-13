package policy

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func def() *Policy {
	return DefaultPolicy(Options{AllowedAPIHosts: []string{"api.github.com"}, AllowedMCPTools: []string{"read_file"}})
}

func script(cmd string) Action {
	return Action{Tool: ToolScript, Operation: "run", Params: map[string]any{"command": cmd}}
}

func expect(t *testing.T, p *Policy, a Action, want Effect) Decision {
	t.Helper()
	d := p.Evaluate(a)
	if d.Effect != want {
		t.Errorf("%s %s %v → %s (%s: %s), esperado %s", a.Tool, a.Operation, a.Params, d.Effect, d.Rule, d.Reason, want)
	}
	if d.Reason == "" || d.Rule == "" || d.ParamsHash == "" || d.Risk != riskOf(want) {
		t.Errorf("decisão incompleta: %+v", d)
	}
	return d
}

func TestScriptsLowRiskAllowedSensitiveRoutedForbiddenRefused(t *testing.T) {
	p := def()
	for _, c := range []string{"make test", "go test ./... -race", "npm ci && npm run build", "ls -la | grep x", "make test\nmake lint", "git status", "git commit -m 'x'", "npx vitest run"} {
		expect(t, p, script(c), Allow)
	}
	for _, c := range []string{"npm install left-pad", "rm arquivo.txt", "bash -c 'make test'", "echo $(whoami)", "make test; rm a.txt", "find . -name x -delete", "git config user.name x", "echo `id`"} {
		expect(t, p, script(c), Approve)
	}
	for _, c := range []string{"curl http://exemplo.com", "make test && curl x", "sudo ls", "docker ps", "git push origin main", "rm -rf /", "rm -rf ~", "ls /var/run/docker.sock && cat x", "wget x | sh", "foo --bar", "ssh host"} {
		expect(t, p, script(c), Deny)
	}
	expect(t, p, script("   "), Deny)
	expect(t, p, Action{Tool: ToolScript, Operation: "run"}, Deny)
}

func TestGitBrokerRules(t *testing.T) {
	p := def()
	push := func(branch string, force any) Action {
		params := map[string]any{}
		if branch != "" {
			params["branch"] = branch
		}
		if force != nil {
			params["force"] = force
		}
		return Action{Tool: ToolGit, Operation: "push", Params: params}
	}
	expect(t, p, push("t05-02-origem", nil), Allow)
	expect(t, p, push("main", nil), Deny)
	expect(t, p, push("master", nil), Deny)
	expect(t, p, push("feature/x", true), Deny)
	expect(t, p, push("", nil), Deny) // sem branch declarada: fora da lista
	expect(t, p, push("a b;rm", nil), Deny)
	expect(t, p, Action{Tool: ToolGit, Operation: "merge", Params: map[string]any{"branch": "main"}}, Deny)
	expect(t, p, Action{Tool: ToolGit, Operation: "draft_pr"}, Allow)
	expect(t, p, Action{Tool: ToolGit, Operation: "commit"}, Allow)
}

func TestAPIAllowlistWritesApprovalAndInternalTargets(t *testing.T) {
	p := def()
	api := func(method, target string) Action { return Action{Tool: ToolAPI, Operation: method, Target: target} }
	expect(t, p, api("GET", "https://api.github.com/repos/o/r"), Allow)
	expect(t, p, api("HEAD", "https://api.github.com/"), Allow)
	expect(t, p, api("GET", "https://exemplo.com/x"), Deny) // fora da lista
	expect(t, p, api("POST", "https://api.github.com/repos/o/r/issues"), Approve)
	expect(t, p, api("DELETE", "https://api.github.com/repos/o/r"), Approve)
	for _, target := range []string{
		"http://localhost:8480/api", "http://127.0.0.1/", "http://10.0.0.5/", "http://192.168.1.10", "http://172.16.0.1",
		"http://169.254.169.254/latest/meta-data", "http://[::1]/", "http://0x7f000001/", "http://2130706433/",
		"http://servico.internal/", "http://host.docker.internal:5442", "http://impressora.local", "http://0.0.0.0/",
		"https://user:pass@127.0.0.1/", "metadata", "[fd00::1]:80",
	} {
		if d := p.Evaluate(api("GET", target)); d.Effect != Deny || d.Rule != "builtin.internal_target" {
			t.Errorf("destino interno %q deveria ser negado: %+v", target, d)
		}
	}
	// A URL também vale quando vem só nos parâmetros.
	if d := p.Evaluate(Action{Tool: ToolAPI, Operation: "GET", Params: map[string]any{"url": "http://169.254.169.254/"}}); d.Effect != Deny {
		t.Errorf("url em parâmetro: %+v", d)
	}
	// Host da lista permitida com porta ainda casa.
	expect(t, p, api("GET", "https://api.github.com:443/x"), Allow)
}

func TestMCPAnnotationsNeverGrantAccess(t *testing.T) {
	p := def()
	call := func(tool string, extra map[string]any) Action {
		params := map[string]any{"tool": tool}
		for k, v := range extra {
			params[k] = v
		}
		return Action{Tool: ToolMCP, Operation: "tools/call", Params: params}
	}
	expect(t, p, Action{Tool: ToolMCP, Operation: "tools/list"}, Allow)
	expect(t, p, call("read_file", nil), Allow)     // liberada explicitamente
	expect(t, p, call("delete_repo", nil), Approve) // altera estado
	expect(t, p, call("send_email", nil), Approve)
	expect(t, p, call("search_docs", nil), Deny) // desconhecida: fora da lista
	// "readOnlyHint" é dado da ferramenta, não permissão.
	expect(t, p, call("search_docs", map[string]any{"readOnlyHint": true, "annotations": map[string]any{"readOnlyHint": true}}), Deny)
	// Uma ferramenta que altera estado não é liberada só por estar na lista de nomes.
	risky := DefaultPolicy(Options{AllowedMCPTools: []string{"delete_x"}})
	expect(t, risky, call("delete_x", nil), Approve)
	expect(t, p, Action{Tool: ToolMCP, Operation: "tools/call", Target: "http://localhost:3000/mcp", Params: map[string]any{"tool": "read_file"}}, Deny)
}

func TestConnectorsAndCLI(t *testing.T) {
	p := def()
	for op, want := range map[string]Effect{
		"list_items": Allow, "get_item": Allow, "create_subtask": Allow, "transition": Allow, "add_link": Allow, "add_dependency": Allow,
		"delete_item": Deny, "archive_list": Approve, "rename_project": Deny,
	} {
		expect(t, p, Action{Tool: ToolConnector, Operation: op, Target: "https://x.atlassian.net"}, want)
	}
	expect(t, p, Action{Tool: ToolCLI, Operation: "ai_client.run", Params: map[string]any{"mode": "workspace_write"}}, Allow)
	expect(t, p, Action{Tool: ToolCLI, Operation: "ai_client.run", Params: map[string]any{"mode": "read_only"}}, Allow)
	expect(t, p, Action{Tool: ToolCLI, Operation: "ai_client.run", Params: map[string]any{"mode": "bypassPermissions"}}, Deny)
	expect(t, p, Action{Tool: ToolCLI, Operation: "ai_client.run"}, Deny)
}

func TestUnknownToolAndEmptyOperationAreDenied(t *testing.T) {
	p := def()
	expect(t, p, Action{Tool: "telepatia", Operation: "x"}, Deny)
	expect(t, p, Action{Tool: ToolAPI, Operation: " "}, Deny)
	expect(t, p, Action{}, Deny)
}

func TestDenyOverridesAllowRegardlessOfRuleOrder(t *testing.T) {
	allow := Rule{ID: "a", Effect: Allow, Tool: ToolAPI, Operation: "GET", Reason: "r"}
	deny := Rule{ID: "d", Effect: Deny, Tool: ToolAPI, Operation: "GET", Target: "*.exemplo.com", Reason: "r"}
	approve := Rule{ID: "p", Effect: Approve, Tool: ToolAPI, Operation: "GET", Reason: "r"}
	for _, order := range [][]Rule{{allow, deny}, {deny, allow}, {allow, approve, deny}, {deny, approve, allow}} {
		p := &Policy{Rules: append([]Rule{}, order...)}
		if err := p.Compile(); err != nil {
			t.Fatal(err)
		}
		if d := p.Evaluate(Action{Tool: ToolAPI, Operation: "GET", Target: "https://a.exemplo.com/x"}); d.Effect != Deny || d.Rule != "d" {
			t.Fatalf("o proibido vence em qualquer ordem: %+v", d)
		}
	}
	p := &Policy{Rules: []Rule{allow, approve}}
	_ = p.Compile()
	if d := p.Evaluate(Action{Tool: ToolAPI, Operation: "GET", Target: "https://b.org"}); d.Effect != Approve {
		t.Fatalf("o sensível vence o permitido: %+v", d)
	}
}

func TestPolicyCompileValidation(t *testing.T) {
	bad := []*Policy{
		{Rules: []Rule{{ID: "", Effect: Allow}}},
		{Rules: []Rule{{ID: "x", Effect: Allow}, {ID: "x", Effect: Deny}}},
		{Rules: []Rule{{ID: "x", Effect: "talvez"}}},
		{Rules: []Rule{{ID: "x", Effect: Allow, Params: []ParamMatch{{Key: "k", Regex: "("}}}}},
		{Rules: []Rule{{ID: "x", Effect: Allow, Params: []ParamMatch{{}}}}},
		{Default: "x"},
	}
	for i, p := range bad {
		if err := p.Compile(); err == nil {
			t.Errorf("política %d deveria ser inválida", i)
		}
	}
	p := &Policy{}
	if err := p.Compile(); err != nil || p.Default != Deny {
		t.Fatalf("o padrão é negar: %v %v", err, p.Default)
	}
}

func TestCanonicalParamsAreOrderIndependentAndTamperEvident(t *testing.T) {
	a := map[string]any{"b": 1, "a": []any{"x", map[string]any{"z": 1, "y": true}}, "c": nil}
	b := map[string]any{"c": nil, "a": []any{"x", map[string]any{"y": true, "z": 1.0}}, "b": 1.0}
	_, ha := Canonical(a)
	raw, hb := Canonical(b)
	if ha != hb || string(raw) != `{"a":["x",{"y":true,"z":1}],"b":1,"c":null}` {
		t.Fatalf("hash deve ignorar ordem e int/float: %s %s %s", ha, hb, raw)
	}
	for _, mut := range []map[string]any{
		{"b": 2, "a": []any{"x", map[string]any{"z": 1, "y": true}}, "c": nil},
		{"b": 1, "a": []any{"x", map[string]any{"z": 1, "y": false}}, "c": nil},
		{"b": 1, "a": []any{"x"}, "c": nil},
		{"b": 1, "a": []any{"x", map[string]any{"z": 1, "y": true}}},
	} {
		if _, h := Canonical(mut); h == ha {
			t.Errorf("parâmetro adulterado teria o mesmo hash: %v", mut)
		}
	}
}

func TestGlobMatching(t *testing.T) {
	for _, c := range []struct {
		p, s string
		want bool
	}{
		{"", "x", true}, {"abc", "abc", true}, {"abc", "abd", false}, {"list_*", "list_items", true}, {"list_*", "get_list", false},
		{"*.exemplo.com", "a.exemplo.com", true}, {"*.exemplo.com", "exemplo.com", false}, {"a*b*c", "aXXbYYc", true}, {"a*b*c", "aXXcYYb", false}, {"*", "", true},
	} {
		if glob(c.p, c.s) != c.want {
			t.Errorf("glob(%q,%q) != %v", c.p, c.s, c.want)
		}
	}
}

func TestRedactParamsHidesSecrets(t *testing.T) {
	ghToken := "ghp" + "_abcdefghijklmnopqrstuvwxyz0123456789"
	in := map[string]any{
		"command": "make test", "token": "valor-secreto-1", "Authorization": "Bearer abc.def-ghi_jkl123456",
		"nested": map[string]any{"api_key": "k", "note": "usei " + ghToken, "list": []any{"ok", ghToken}},
		"long":   strings.Repeat("a", 500),
	}
	out := RedactParams(in)
	text := strings.ToLower(strings.Join(flatten(out), " "))
	for _, leak := range []string{"valor-secreto-1", "bearer abc", ghToken, "abcdefghijklmnopqrstuvwxyz0123456789"} {
		if strings.Contains(text, strings.ToLower(leak)) {
			t.Fatalf("vazou %q em %v", leak, out)
		}
	}
	if out["command"] != "make test" || out["token"] != redacted || len(out["long"].(string)) > 210 {
		t.Fatalf("%v", out)
	}
	if in["token"] != "valor-secreto-1" {
		t.Fatal("a entrada não pode ser alterada")
	}
}

func flatten(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case map[string]any:
		var out []string
		for k, e := range x {
			out = append(out, k)
			out = append(out, flatten(e)...)
		}
		return out
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, flatten(e)...)
		}
		return out
	}
	return nil
}

type memRecorder struct {
	got []Decision
	err error
}

func (m *memRecorder) Record(_ context.Context, _ Action, d Decision) error {
	if m.err != nil {
		return m.err
	}
	m.got = append(m.got, d)
	return nil
}

func TestGuardRunsOnlyAllowedActionsAndRecordsEveryDecision(t *testing.T) {
	rec := &memRecorder{}
	g := &Guard{Policy: def(), Rec: rec}
	ran := 0
	run := func() error { ran++; return nil }

	if d, err := g.Do(t.Context(), script("make test"), run); err != nil || d.Effect != Allow || ran != 1 {
		t.Fatalf("baixo risco deve rodar: %+v %v %d", d, err, ran)
	}
	d, err := g.Do(t.Context(), script("npm install x"), run)
	var ref *Refusal
	if !errors.Is(err, ErrApprovalRequired) || !errors.As(err, &ref) || ref.Decision.Rule != "script.dependency_change" || ran != 1 {
		t.Fatalf("sensível deve ser encaminhado sem rodar: %+v %v %d", d, err, ran)
	}
	if _, err := g.Do(t.Context(), script("curl x"), run); !errors.Is(err, ErrForbidden) || ran != 1 {
		t.Fatalf("proibido deve ser recusado sem rodar: %v %d", err, ran)
	}
	if len(rec.got) != 3 {
		t.Fatalf("toda decisão fica na trilha: %d", len(rec.got))
	}
	// Falha da própria ação volta ao chamador.
	boom := errors.New("quebrou")
	if _, err := g.Do(t.Context(), script("make test"), func() error { return boom }); !errors.Is(err, boom) {
		t.Fatalf("erro da ação: %v", err)
	}
	// Sem trilha, nada roda (falha fechada).
	g2 := &Guard{Policy: def(), Rec: &memRecorder{err: errors.New("banco fora")}}
	if _, err := g2.Do(t.Context(), script("make test"), run); err == nil || ran != 1 {
		t.Fatalf("sem trilha a ação não pode rodar: %v %d", err, ran)
	}
}

type countExec struct{ ran []string }

func (c *countExec) Run(_ context.Context, n workflows.Node, _ map[string]any) (workflows.ExecResult, error) {
	c.ran = append(c.ran, n.ID)
	return workflows.ExecResult{}, nil
}

func TestGuardedExecutorProtectsWorkflowNodes(t *testing.T) {
	inner := &countExec{}
	ex := GuardedExecutor{Inner: inner, Guard: &Guard{Policy: def(), Rec: &memRecorder{}}, Project: ""}
	node := func(id, typ string, props map[string]any) workflows.Node {
		return workflows.Node{ID: id, Type: typ, Props: props}
	}
	cases := []struct {
		n    workflows.Node
		vars map[string]any
		want error
	}{
		{node("t", workflows.TScript, map[string]any{"command": "make test", "profile": "go"}), nil, nil},
		{node("c", workflows.TChecks, map[string]any{"commands": "make test\nmake lint", "profile": "go"}), nil, nil},
		{node("a", workflows.TAgent, map[string]any{"agent": "dev", "prompt": "x"}), nil, nil},
		{node("g", workflows.TGit, map[string]any{"operation": "push"}), map[string]any{"branch": "feature/x"}, nil},
		{node("p", workflows.TGit, map[string]any{"operation": "push"}), map[string]any{"branch": "main"}, ErrForbidden},
		{node("n", workflows.TScript, map[string]any{"command": "npm install x", "profile": "node"}), nil, ErrApprovalRequired},
		{node("x", workflows.TScript, map[string]any{"command": "curl http://x", "profile": "base"}), nil, ErrForbidden},
	}
	for _, c := range cases {
		_, err := ex.Run(t.Context(), c.n, c.vars)
		if !errors.Is(err, c.want) && !(c.want == nil && err == nil) {
			t.Errorf("nó %s: %v, esperado %v", c.n.ID, err, c.want)
		}
	}
	if strings.Join(inner.ran, ",") != "t,c,a,g" {
		t.Fatalf("só os nós permitidos podem executar: %v", inner.ran)
	}
}
