package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

func pol() *policy.Policy { return policy.DefaultPolicy(policy.Options{}) }

type countRec struct{ n int }

func (c *countRec) Record(context.Context, policy.Action, policy.Decision) error { c.n++; return nil }

func grantAll(names ...string) Grant {
	return Grant{Tools: names, Permissions: []string{PermRead, PermWrite, PermExec, PermGit, PermAPI}}
}

func TestNativeCatalogIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, tl := range Natives() {
		if seen[tl.Name] {
			t.Fatalf("repetida: %s", tl.Name)
		}
		seen[tl.Name] = true
		if err := tl.Compile(); err != nil {
			t.Errorf("%s: %v", tl.Name, err)
		}
		if tl.Kind != KindNative || len(tl.Permissions) == 0 || len(tl.Output) == 0 {
			t.Errorf("%s incompleta: %+v", tl.Name, tl)
		}
	}
	for _, want := range []string{"read_file", "write_file", "edit_file", "list_dir", "search", "run_command", "git_status", "git_diff"} {
		if !seen[want] {
			t.Errorf("faltou %s", want)
		}
	}
}

func TestSafePathConfinesToTheWorkspace(t *testing.T) {
	ok := map[string]string{
		"a.txt": "/workspace/a.txt", "sub/b.txt": "/workspace/sub/b.txt", "/workspace/c": "/workspace/c", "/workspace": "/workspace", "./d": "/workspace/d",
	}
	for in, want := range ok {
		if got, err := SafePath(in); err != nil || got != want {
			t.Errorf("%q → %q %v", in, got, err)
		}
	}
	for _, in := range []string{"", "../etc/passwd", "/etc/passwd", "/workspace/../etc", "a/../../b", "/workspacex/a", "..", "/workspace/x/../y", "a\x00b", `a\b`, strings.Repeat("a", 400), "/proc/self/environ", "/var/run/docker.sock"} {
		if got, err := SafePath(in); err == nil {
			t.Errorf("%q deveria ser recusado (veio %q)", in, got)
		}
	}
}

func byName(t *testing.T, name string) Tool {
	t.Helper()
	for _, tl := range Natives() {
		if tl.Name == name {
			_ = (&tl).Compile()
			return tl
		}
	}
	t.Fatalf("sem %s", name)
	return Tool{}
}

func TestValidateRejectsBeforeAnyEffect(t *testing.T) {
	edit := byName(t, "edit_file")
	okArgs := map[string]any{"path": "a.txt", "old": "x", "new": "y"}
	if got, err := edit.Validate(okArgs); err != nil || got["path"] != "/workspace/a.txt" {
		t.Fatalf("%v %v", got, err)
	}
	for name, args := range map[string]map[string]any{
		"sem path":            {"old": "x", "new": "y"},
		"parâmetro extra":     {"path": "a", "old": "x", "new": "y", "force": true},
		"path que escapa":     {"path": "../etc/passwd", "old": "x", "new": "y"},
		"tipo errado":         {"path": "a", "old": 1, "new": "y"},
		"texto grande demais": {"path": "a", "old": strings.Repeat("a", 100001), "new": "y"},
		"nil obrigatório":     {"path": "a", "old": nil, "new": "y"},
	} {
		if _, err := edit.Validate(args); err == nil {
			t.Errorf("%s: deveria falhar", name)
		}
	}
	read := byName(t, "read_file")
	for name, args := range map[string]map[string]any{
		"inteiro fracionário": {"path": "a", "max_bytes": 1.5},
		"abaixo do mínimo":    {"path": "a", "max_bytes": 0},
		"acima do máximo":     {"path": "a", "max_bytes": 2 << 20},
		"inteiro como texto":  {"path": "a", "max_bytes": "10"},
	} {
		if _, err := read.Validate(args); err == nil {
			t.Errorf("%s: deveria falhar", name)
		}
	}
	if got, err := read.Validate(map[string]any{"path": "a", "max_bytes": float64(10)}); err != nil || got["max_bytes"] != 10 {
		t.Fatalf("%v %v", got, err)
	}
	var ae *ArgError
	if _, err := edit.Validate(map[string]any{"path": "/etc/passwd", "old": "x", "new": "y"}); !errors.As(err, &ae) || ae.Field != "path" {
		t.Fatalf("o erro deve apontar o campo: %v", err)
	}
	enum := Tool{Name: "ft", Kind: KindScript, Description: "d", Permissions: []string{PermExec}, Limits: Limits{1, 1}, Command: "echo",
		Input: []Field{{Name: "modo", Type: TypeEnum, Options: []string{"a", "b"}, Required: true}, {Name: "ok", Type: TypeBool}, {Name: "id", Type: TypeString, Pattern: `^[0-9]+$`}}}
	if err := enum.Compile(); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []map[string]any{{"modo": "c"}, {"modo": "a", "ok": "sim"}, {"modo": "a", "id": "x1"}} {
		if _, err := enum.Validate(bad); err == nil {
			t.Errorf("%v deveria falhar", bad)
		}
	}
}

func TestCheckOrderUnknownThenGrantThenScopeThenArgs(t *testing.T) {
	r := NewRegistry(nil)
	ctx := context.Background()
	good := map[string]any{"path": "a.txt"}
	if _, _, err := r.Check(ctx, "nao_existe", good, grantAll("nao_existe")); !errors.Is(err, ErrUnknownTool) {
		t.Fatalf("%v", err)
	}
	// Ferramenta existente, mas não concedida: nasce sem autorização, mesmo com argumentos válidos.
	if _, _, err := r.Check(ctx, "read_file", good, Grant{Permissions: []string{PermRead}}); !errors.Is(err, ErrNotGranted) {
		t.Fatalf("%v", err)
	}
	if _, _, err := r.Check(ctx, "read_file", good, Grant{Tools: []string{"write_file"}, Permissions: []string{PermRead}}); !errors.Is(err, ErrNotGranted) {
		t.Fatalf("conceder outra ferramenta não vale: %v", err)
	}
	// Concedida, mas sem o escopo (permissão): edit exige read e write.
	if _, _, err := r.Check(ctx, "edit_file", map[string]any{"path": "a", "old": "x", "new": "y"}, Grant{Tools: []string{"edit_file"}, Permissions: []string{PermRead}}); !errors.Is(err, ErrScope) {
		t.Fatalf("%v", err)
	}
	if _, _, err := r.Check(ctx, "read_file", map[string]any{"path": "../x"}, grantAll("read_file")); err == nil || errors.Is(err, ErrScope) {
		t.Fatalf("argumento inválido: %v", err)
	}
	if tl, args, err := r.Check(ctx, "read_file", good, grantAll("read_file")); err != nil || tl.Name != "read_file" || args["path"] != "/workspace/a.txt" {
		t.Fatalf("%v", err)
	}
	// Sem ambiente nada executa, e nada chegou à política.
	if _, err := r.Call(ctx, Env{}, "read_file", good, grantAll("read_file")); !errors.Is(err, ErrNotReady) {
		t.Fatalf("sem política/sandbox: %v", err)
	}
}

func TestCustomToolRegistrationRules(t *testing.T) {
	base := func() Tool {
		return Tool{Name: "contar", Kind: KindScript, Description: "Conta linhas.", Permissions: []string{PermRead}, Limits: Limits{30, 1 << 16},
			Command: `grep -c "" "$PARAM_path"`, Input: []Field{{Name: "path", Type: TypePath, Required: true}}}
	}
	if err := func() error { x := base(); return x.Compile() }(); err != nil {
		t.Fatal(err)
	}
	bad := map[string]func(*Tool){
		"nome":               func(x *Tool) { x.Name = "Contar Linhas" },
		"tipo":               func(x *Tool) { x.Kind = "magica" },
		"descrição":          func(x *Tool) { x.Description = " " },
		"permissão":          func(x *Tool) { x.Permissions = []string{"root"} },
		"sem permissão":      func(x *Tool) { x.Permissions = nil },
		"limites":            func(x *Tool) { x.Limits.TimeoutSeconds = 0 },
		"script sem comando": func(x *Tool) { x.Command = "" },
		"integração sem op":  func(x *Tool) { x.Kind = KindIntegration; x.Operation = "" },
		"campo repetido":     func(x *Tool) { x.Input = append(x.Input, Field{Name: "path", Type: TypeString}) },
		"tipo de campo":      func(x *Tool) { x.Input = []Field{{Name: "a", Type: "xyz"}} },
		"enum sem opções":    func(x *Tool) { x.Input = []Field{{Name: "a", Type: TypeEnum}} },
		"lista na entrada":   func(x *Tool) { x.Input = []Field{{Name: "a", Type: TypeList}} },
		"padrão inválido":    func(x *Tool) { x.Input = []Field{{Name: "a", Type: TypeString, Pattern: "("}} },
	}
	for name, mut := range bad {
		x := base()
		mut(&x)
		if err := x.Compile(); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	out := Tool{Output: []Field{{Name: "n", Type: TypeInt, Required: true}, {Name: "s", Type: TypeString}, {Name: "l", Type: TypeList}}}
	got, err := out.TypedOutput(map[string]any{"n": float64(3), "s": "x", "l": []string{"a"}, "extra": "descartado"})
	if err != nil || !reflect.DeepEqual(got, map[string]any{"n": 3, "s": "x", "l": []string{"a"}}) {
		t.Fatalf("%v %v", got, err)
	}
	for _, raw := range []map[string]any{{"s": "x"}, {"n": "3"}, {"n": 1, "s": 2}, {"n": 1, "l": []any{"a"}}} {
		if _, err := out.TypedOutput(raw); err == nil {
			t.Errorf("%v deveria falhar", raw)
		}
	}
}

func dockerSandbox(t *testing.T) *execution.Sandbox {
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

func call(t *testing.T, r *Registry, env Env, name string, args map[string]any) (Result, error) {
	t.Helper()
	return r.Call(t.Context(), env, name, args, grantAll(name))
}

func TestNativeToolsReturnTypedOutputInTheSandbox(t *testing.T) {
	sb := dockerSandbox(t)
	rec := &countRec{}
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: rec}}
	r := NewRegistry(nil)

	w, err := call(t, r, env, "write_file", map[string]any{"path": "src/a.txt", "content": "linha 1\nlinha 2\nlinha 3\n"})
	if err != nil || w.Output["path"] != "/workspace/src/a.txt" || w.Output["bytes"] != 24 {
		t.Fatalf("write: %+v %v", w, err)
	}
	rd, err := call(t, r, env, "read_file", map[string]any{"path": "src/a.txt"})
	if err != nil || rd.Output["content"] != "linha 1\nlinha 2\nlinha 3\n" || rd.Output["truncated"] != false || rd.Output["bytes"] != 24 {
		t.Fatalf("read: %+v %v", rd, err)
	}
	if short, err := call(t, r, env, "read_file", map[string]any{"path": "src/a.txt", "max_bytes": 5}); err != nil || short.Output["content"] != "linha" || short.Output["truncated"] != true || !short.Truncated {
		t.Fatalf("read truncado: %+v %v", short, err)
	}
	if _, err := call(t, r, env, "read_file", map[string]any{"path": "nao/existe"}); err == nil {
		t.Fatal("arquivo inexistente deve falhar")
	}

	ed, err := call(t, r, env, "edit_file", map[string]any{"path": "src/a.txt", "old": "linha 2", "new": "LINHA DOIS"})
	if err != nil || ed.Output["replacements"] != 1 {
		t.Fatalf("edit: %+v %v", ed, err)
	}
	if rd, _ := call(t, r, env, "read_file", map[string]any{"path": "src/a.txt"}); rd.Output["content"] != "linha 1\nLINHA DOIS\nlinha 3\n" {
		t.Fatalf("depois do edit: %v", rd.Output["content"])
	}
	for name, args := range map[string]map[string]any{
		"ambígua":     {"path": "src/a.txt", "old": "linha", "new": "x"},
		"inexistente": {"path": "src/a.txt", "old": "nada disso", "new": "x"},
		"vazia":       {"path": "src/a.txt", "old": "", "new": "x"},
	} {
		if _, err := call(t, r, env, "edit_file", args); err == nil {
			t.Errorf("edição %s deveria falhar", name)
		}
	}

	ls, err := call(t, r, env, "list_dir", map[string]any{"path": "."})
	if err != nil || !reflect.DeepEqual(ls.Output["entries"], []string{"src/"}) {
		t.Fatalf("list: %+v %v", ls, err)
	}
	se, err := call(t, r, env, "search", map[string]any{"pattern": "LINHA"})
	if err != nil || !reflect.DeepEqual(se.Output["matches"], []string{"/workspace/src/a.txt:2:LINHA DOIS"}) {
		t.Fatalf("search: %+v %v", se, err)
	}
	if none, err := call(t, r, env, "search", map[string]any{"pattern": "inexistente-xyz"}); err != nil || len(none.Output["matches"].([]string)) != 0 {
		t.Fatalf("sem ocorrências não é erro: %+v %v", none, err)
	}

	rc, err := call(t, r, env, "run_command", map[string]any{"command": "echo ola; echo erro >&2"})
	if err != nil || rc.Output["exit_code"] != 0 || strings.TrimSpace(rc.Output["stdout"].(string)) != "ola" || strings.TrimSpace(rc.Output["stderr"].(string)) != "erro" {
		t.Fatalf("run: %+v %v", rc, err)
	}
	if rec.n == 0 {
		t.Fatal("as chamadas válidas passam pela política e ficam na trilha")
	}
	if _, err := call(t, r, env, "git_status", nil); err == nil || !strings.Contains(err.Error(), "git") {
		t.Fatalf("sem git na imagem a falha deve ser clara: %v", err)
	}
}

func TestInvalidScopeOrArgsFailBeforeAnyEffect(t *testing.T) {
	sb := dockerSandbox(t)
	rec := &countRec{}
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: rec}}
	r := NewRegistry(nil)
	ctx := t.Context()

	// Argumento indevido: caminho que escapa. Nada é gravado, a política nem é consultada.
	if _, err := call(t, r, env, "write_file", map[string]any{"path": "../../tmp/pwned", "content": "x"}); err == nil {
		t.Fatal("deveria falhar")
	}
	// Escopo indevido: concedida a ferramenta de escrita, mas só a permissão de leitura.
	if _, err := r.Call(ctx, env, "write_file", map[string]any{"path": "a.txt", "content": "x"}, Grant{Tools: []string{"write_file"}, Permissions: []string{PermRead}}); !errors.Is(err, ErrScope) {
		t.Fatalf("%v", err)
	}
	// Ferramenta não concedida.
	if _, err := r.Call(ctx, env, "run_command", map[string]any{"command": "touch /workspace/pwned"}, Grant{Tools: []string{"read_file"}, Permissions: []string{PermRead, PermExec}}); !errors.Is(err, ErrNotGranted) {
		t.Fatalf("%v", err)
	}
	if rec.n != 0 {
		t.Fatalf("falhas de validação/escopo não podem chegar à política: %d", rec.n)
	}
	out, _ := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "ls /workspace /tmp | grep -c -E 'pwned|a.txt' || true"}})
	if strings.TrimSpace(out.Stdout) != "0" {
		t.Fatalf("algum efeito aconteceu: %q", out.Stdout)
	}
}

func TestRunCommandIsSubjectToPolicy(t *testing.T) {
	sb := dockerSandbox(t)
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: &countRec{}}}
	r := NewRegistry(nil)
	if _, err := call(t, r, env, "run_command", map[string]any{"command": "curl http://exemplo.com > /workspace/baixado"}); !errors.Is(err, policy.ErrForbidden) {
		t.Fatalf("proibido: %v", err)
	}
	if _, err := call(t, r, env, "run_command", map[string]any{"command": "npm install left-pad"}); !errors.Is(err, policy.ErrApprovalRequired) {
		t.Fatalf("sensível: %v", err)
	}
	// Misturar um comando sensível com um fora da lista continua proibido: o mais restritivo vence.
	if _, err := call(t, r, env, "run_command", map[string]any{"command": "npm install left-pad && touch /workspace/instalou"}); !errors.Is(err, policy.ErrForbidden) {
		t.Fatalf("misto: %v", err)
	}
	out, _ := sb.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", "ls /workspace | wc -l"}})
	if strings.TrimSpace(out.Stdout) != "0" {
		t.Fatalf("nenhum efeito era esperado: %q", out.Stdout)
	}
}

func TestCustomScriptToolsPassParametersByEnvironmentAndTypeTheirOutput(t *testing.T) {
	sb := dockerSandbox(t)
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: &countRec{}}}
	tools := map[string]Tool{
		"ler_meta": {Name: "ler_meta", Kind: KindScript, Description: "Lê um JSON de metadados.", Permissions: []string{PermRead}, Limits: Limits{20, 1 << 16},
			Command: `cat "$PARAM_path"`, Input: []Field{{Name: "path", Type: TypePath, Required: true}},
			Output: []Field{{Name: "nome", Type: TypeString, Required: true}, {Name: "pronto", Type: TypeBool, Required: true}, {Name: "tags", Type: TypeList}}},
		"mostrar": {Name: "mostrar", Kind: KindScript, Description: "Mostra um valor.", Permissions: []string{PermExec}, Limits: Limits{20, 12},
			Command: `echo "$PARAM_valor"`, Input: []Field{{Name: "valor", Type: TypeString, Required: true, Max: 200}}},
		"mostrar_longo": {Name: "mostrar_longo", Kind: KindScript, Description: "Mostra um valor.", Permissions: []string{PermExec}, Limits: Limits{20, 1 << 16},
			Command: `echo "$PARAM_valor"`, Input: []Field{{Name: "valor", Type: TypeString, Required: true, Max: 200}}},
		"lento": {Name: "lento", Kind: KindScript, Description: "Demora.", Permissions: []string{PermExec}, Limits: Limits{1, 1024}, Command: `tail -f /dev/null`},
	}
	r := NewRegistry(func(_ context.Context, name string) (Tool, error) {
		if t, ok := tools[name]; ok {
			return t, nil
		}
		return Tool{}, errors.New("não")
	})
	call(t, r, env, "write_file", map[string]any{"path": "meta.json", "content": `{"nome":"exemplo","pronto":true,"tags":["a","b"],"ignorado":1}`})

	res, err := call(t, r, env, "ler_meta", map[string]any{"path": "meta.json"})
	if err != nil || !reflect.DeepEqual(res.Output, map[string]any{"nome": "exemplo", "pronto": true, "tags": []string{"a", "b"}}) {
		t.Fatalf("saída tipada: %+v %v", res, err)
	}
	call(t, r, env, "write_file", map[string]any{"path": "ruim.json", "content": `{"nome": 1, "pronto": true}`})
	if _, err := call(t, r, env, "ler_meta", map[string]any{"path": "ruim.json"}); err == nil || !strings.Contains(err.Error(), "nome") {
		t.Fatalf("saída de tipo errado deve falhar: %v", err)
	}
	call(t, r, env, "write_file", map[string]any{"path": "nao-json.json", "content": `isto não é json`})
	if _, err := call(t, r, env, "ler_meta", map[string]any{"path": "nao-json.json"}); err == nil {
		t.Fatal("saída que não é JSON deve falhar")
	}

	// O valor do parâmetro nunca é interpretado como comando.
	inj, err := call(t, r, env, "mostrar_longo", map[string]any{"valor": `"; touch /workspace/pwned; echo "`})
	if err != nil || !strings.Contains(inj.Output["stdout"].(string), "touch /workspace/pwned") {
		t.Fatalf("o valor deve sair como texto: %+v %v", inj, err)
	}
	if lst, _ := call(t, r, env, "list_dir", map[string]any{"path": "."}); containsStr(lst.Output["entries"].([]string), "pwned") {
		t.Fatal("injeção de comando pelo parâmetro")
	}
	// Limite de saída e de tempo.
	long, err := call(t, r, env, "mostrar", map[string]any{"valor": strings.Repeat("x", 100)})
	if err != nil || !long.Truncated || len(long.Output["stdout"].(string)) != 12 {
		t.Fatalf("limite de saída: %+v %v", long, err)
	}
	if _, err := call(t, r, env, "lento", nil); err == nil || !strings.Contains(err.Error(), "tempo") {
		t.Fatalf("limite de tempo: %v", err)
	}
	_ = os.Getpid
	_ = fmt.Sprint
}

func containsStr(l []string, s string) bool {
	for _, x := range l {
		if x == s || strings.Contains(x, s) {
			return true
		}
	}
	return false
}

func TestIntegrationToolsUseTheConnectorRunner(t *testing.T) {
	var gotOp string
	var gotArgs map[string]any
	env := Env{Guard: &policy.Guard{Policy: pol(), Rec: &countRec{}}, Connector: func(_ context.Context, op string, args map[string]any) (map[string]any, error) {
		gotOp, gotArgs = op, args
		return map[string]any{"key": "ABC-1"}, nil
	}}
	tool := Tool{Name: "criar_sub", Kind: KindIntegration, Operation: "create_subtask", Description: "Cria subtarefa.", Permissions: []string{PermAPI}, Limits: Limits{20, 1024},
		Input:  []Field{{Name: "titulo", Type: TypeString, Required: true, Max: 200}},
		Output: []Field{{Name: "key", Type: TypeString, Required: true}}}
	r := NewRegistry(func(_ context.Context, name string) (Tool, error) { return tool, nil })
	res, err := r.Call(t.Context(), env, "criar_sub", map[string]any{"titulo": "Nova"}, grantAll("criar_sub"))
	if err != nil || res.Output["key"] != "ABC-1" || gotOp != "create_subtask" || gotArgs["titulo"] != "Nova" {
		t.Fatalf("%+v %v %q %v", res, err, gotOp, gotArgs)
	}
	// Uma operação fora da política (renomear) é recusada antes de chamar o conector.
	tool.Operation = "rename_project"
	gotOp = ""
	if _, err := r.Call(t.Context(), env, "criar_sub", map[string]any{"titulo": "Nova"}, grantAll("criar_sub")); !errors.Is(err, policy.ErrForbidden) || gotOp != "" {
		t.Fatalf("%v %q", err, gotOp)
	}
	env.Connector = nil
	tool.Operation = "create_subtask"
	if _, err := r.Call(t.Context(), env, "criar_sub", map[string]any{"titulo": "Nova"}, grantAll("criar_sub")); !errors.Is(err, ErrNotReady) {
		t.Fatalf("sem conector: %v", err)
	}
}
