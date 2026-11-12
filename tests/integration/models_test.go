package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/gateways"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func registerModel(t *testing.T, c *client, in map[string]any) models.Model {
	t.Helper()
	var m models.Model
	c.expect(c.post("/api/models", in), 201).JSON(&m)
	return m
}

func apiModel(gateway, name string, tools bool) map[string]any {
	return map[string]any{"provider": "api", "gateway": gateway, "name": name, "supports_tools": tools, "supports_streaming": true,
		"context_window": 200000, "max_output_tokens": 8000, "params": map[string]any{"temperature": 0.2, "max_tokens": 4000}}
}

func TestModelCatalogValidatesRegistrationAndStates(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	m := registerModel(t, c, apiModel("openrouter", "vendor/model-x", true))
	if m.State != "active" || m.Version != 1 || m.Params.Temperature == nil || *m.Params.Temperature != 0.2 || m.ContextWindow != 200000 {
		t.Fatalf("%+v", m)
	}
	cli := registerModel(t, c, map[string]any{"provider": "claude", "name": "claude-sonnet-5-5", "supports_tools": true})
	if cli.Gateway != "" {
		t.Fatalf("%+v", cli)
	}
	var list struct{ Items []models.Model }
	c.expect(c.get("/api/models"), 200).JSON(&list)
	if len(list.Items) != 2 {
		t.Fatalf("%d", len(list.Items))
	}
	// Mesmo nome em outro gateway é outro modelo; repetido no mesmo é duplicado.
	registerModel(t, c, apiModel("vercel", "vendor/model-x", false))
	if r := c.post("/api/models", apiModel("openrouter", "vendor/model-x", true)); r.StatusCode != 409 {
		t.Fatalf("duplicado: %d %s", r.StatusCode, r.Body)
	}
	bad := map[string]map[string]any{
		"temperatura":        {"provider": "api", "gateway": "openrouter", "name": "a/b", "params": map[string]any{"temperature": 3}},
		"top_p":              {"provider": "api", "gateway": "openrouter", "name": "a/b", "params": map[string]any{"top_p": 1.5}},
		"max_tokens > saída": {"provider": "api", "gateway": "openrouter", "name": "a/b", "max_output_tokens": 100, "params": map[string]any{"max_tokens": 500}},
		"saída > contexto":   {"provider": "api", "gateway": "openrouter", "name": "a/b", "context_window": 100, "max_output_tokens": 500},
		"api sem gateway":    {"provider": "api", "name": "a/b"},
		"gateway inválido":   {"provider": "api", "gateway": "outro", "name": "a/b"},
		"cli com gateway":    {"provider": "claude", "gateway": "openrouter", "name": "a/b"},
		"provedor inválido":  {"provider": "x", "name": "a/b"},
		"nome inválido":      {"provider": "claude", "name": "nome com espaço"},
		"campo desconhecido": {"provider": "claude", "name": "a", "admin": true},
	}
	for name, body := range bad {
		if r := c.post("/api/models", body); r.StatusCode != 400 {
			t.Errorf("%s: %d %s", name, r.StatusCode, r.Body)
		}
	}
	// Estados com controle otimista.
	var off models.Model
	c.expect(c.post("/api/models/"+m.ID+"/state", map[string]any{"state": "disabled", "expected_version": 1}), 200).JSON(&off)
	if off.State != "disabled" || off.Version != 2 {
		t.Fatalf("%+v", off)
	}
	if r := c.post("/api/models/"+m.ID+"/state", map[string]any{"state": "active", "expected_version": 1}); r.StatusCode != 409 {
		t.Fatalf("versão obsoleta: %d", r.StatusCode)
	}
	if r := c.post("/api/models/"+m.ID+"/state", map[string]any{"state": "quebrado", "expected_version": 2}); r.StatusCode != 400 {
		t.Fatalf("estado inválido: %d", r.StatusCode)
	}
	if r := c.post("/api/models/00000000-0000-4000-8000-000000000000/state", map[string]any{"state": "active", "expected_version": 1}); r.StatusCode != 404 {
		t.Fatalf("inexistente: %d", r.StatusCode)
	}
	// Sessão e CSRF.
	anon := newAuthEnvClient(t)
	if r := anon.get("/api/models"); r.StatusCode != 401 {
		t.Fatalf("sem sessão: %d", r.StatusCode)
	}
	if r := c.do("POST", "/api/models", apiModel("vercel", "x/y", true), map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Fatalf("sem CSRF: %d", r.StatusCode)
	}
}

func newAuthEnvClient(t *testing.T) *client {
	t.Helper()
	e := newAuthEnv(t, nil)
	return e.newClient()
}

// T09-02 — "Modelo só é usado quando compatível."
func TestModelIsOnlyUsableWhenCompatible(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	orConn := connWith(t, c, "api", "or", "openrouter")
	vcConn := connWith(t, c, "api", "vc", "vercel")
	oddConn := connWith(t, c, "api", "odd", "meu-gateway")
	cliConn := connWith(t, c, "cli", "cli-claude", "claude")
	registerModel(t, c, apiModel("openrouter", "vendor/tools", true))
	plain := registerModel(t, c, apiModel("openrouter", "vendor/plain", false))
	registerModel(t, c, map[string]any{"provider": "claude", "name": "claude-sonnet-5-5", "supports_tools": true})

	check := func(body map[string]any) models.Verdict {
		var v models.Verdict
		c.expect(c.post("/api/models/check", body), 200).JSON(&v)
		return v
	}
	if v := check(map[string]any{"provider": "api", "connection_id": orConn.ID, "model": "vendor/tools", "needs_tools": true}); !v.OK || v.Model == nil || len(v.Problems) != 0 {
		t.Fatalf("compatível: %+v", v)
	}
	cases := map[string]struct {
		body map[string]any
		want string
	}{
		"sem suporte a ferramentas":    {map[string]any{"provider": "api", "connection_id": orConn.ID, "model": "vendor/plain", "needs_tools": true}, "não suporta ferramentas"},
		"não cadastrado":               {map[string]any{"provider": "api", "connection_id": orConn.ID, "model": "vendor/outro"}, "não está cadastrado"},
		"cadastrado em outro gateway":  {map[string]any{"provider": "api", "connection_id": vcConn.ID, "model": "vendor/tools"}, "não está cadastrado"},
		"destino não suportado":        {map[string]any{"provider": "api", "connection_id": oddConn.ID, "model": "vendor/tools"}, "não é um gateway suportado"},
		"conta de CLI no provedor api": {map[string]any{"provider": "api", "connection_id": cliConn.ID, "model": "vendor/tools"}, "tipo api"},
		"api sem conta":                {map[string]any{"provider": "api", "model": "vendor/tools"}, "exige uma conta"},
		"api sem modelo":               {map[string]any{"provider": "api", "connection_id": orConn.ID}, "exige um modelo"},
		"conta inexistente":            {map[string]any{"provider": "api", "connection_id": "00000000-0000-4000-8000-000000000000", "model": "vendor/tools"}, "conta não existe"},
		"cli não cadastrado":           {map[string]any{"provider": "claude", "model": "modelo-que-nao-existe"}, "não está cadastrado"},
	}
	for name, tc := range cases {
		v := check(tc.body)
		if v.OK || !strings.Contains(strings.Join(v.Problems, " "), tc.want) {
			t.Errorf("%s: %+v", name, v)
		}
	}
	// CLI sem modelo explícito usa o padrão da conta: compatível.
	if v := check(map[string]any{"provider": "claude"}); !v.OK {
		t.Fatalf("%+v", v)
	}
	// Desativado deixa de servir; reativado volta.
	tools := check(map[string]any{"provider": "api", "connection_id": orConn.ID, "model": "vendor/tools"}).Model
	c.expect(c.post("/api/models/"+tools.ID+"/state", map[string]any{"state": "disabled", "expected_version": tools.Version}), 200)
	if v := check(map[string]any{"provider": "api", "connection_id": orConn.ID, "model": "vendor/tools"}); v.OK || !strings.Contains(strings.Join(v.Problems, " "), "desativado") {
		t.Fatalf("%+v", v)
	}
	// Conta revogada.
	c.expect(c.post("/api/connections/"+orConn.ID+"/revoke", nil), 200)
	if v := check(map[string]any{"provider": "api", "connection_id": orConn.ID, "model": "vendor/plain"}); v.OK || !strings.Contains(strings.Join(v.Problems, " "), "revogada") {
		t.Fatalf("%+v", v)
	}
	_ = plain
}

// T09-02 — "Sobreposição modelo/agente/nó é verificável."
func TestModelOverridesAreVerifiablePerNode(t *testing.T) {
	_, c, _ := newProjectEnv(t)
	orConn := connWith(t, c, "api", "or", "openrouter")
	vcConn := connWith(t, c, "api", "vc", "vercel")
	registerModel(t, c, apiModel("openrouter", "vendor/tools", true))
	registerModel(t, c, apiModel("openrouter", "vendor/plain", false))
	registerModel(t, c, apiModel("vercel", "vendor/vc-tools", true))

	body := agentBody("dev", &orConn.ID)
	body["provider"], body["model"], body["tools"] = "api", "vendor/tools", []string{"read_file"}
	var base agents.Agent
	c.expect(c.post("/api/agents", body), 201).JSON(&base)

	type eff struct {
		agents.Effective
		ModelCheck models.Verdict `json:"model_check"`
	}
	seq := 0
	get := func(props map[string]any) eff {
		seq++
		w, n := workflowWithAgent(t, c, fmt.Sprintf("Fluxo %d", seq), props)
		var e eff
		c.expect(c.get("/api/workflows/"+w.ID+"/nodes/"+n+"/effective"), 200).JSON(&e)
		return e
	}
	// Sem sobrescrita: vale o modelo do agente.
	if e := get(map[string]any{"timeout_minutes": 5}); e.Model != "vendor/tools" || e.Sources["model"] != "agent" || !e.ModelCheck.OK {
		t.Fatalf("%+v", e)
	}
	// O nó troca para um modelo que não suporta ferramentas: a origem é o nó e o veredito explica.
	e := get(map[string]any{"model": "vendor/plain"})
	if e.Model != "vendor/plain" || e.Sources["model"] != "node" || e.ModelCheck.OK || !strings.Contains(strings.Join(e.ModelCheck.Problems, " "), "não suporta ferramentas") {
		t.Fatalf("%+v", e)
	}
	// Reduzir as ferramentas do nó a zero... torna o mesmo modelo compatível (o requisito é do efetivo).
	if e := get(map[string]any{"model": "vendor/plain", "tools": ""}); e.Sources["tools"] == "node" && len(e.Tools) == 0 && !e.ModelCheck.OK {
		// "tools" vazio não é sobrescrita (string vazia); mantém a base: continua incompatível.
		t.Fatalf("%+v", e)
	}
	// Modelo não cadastrado no nó.
	if e := get(map[string]any{"model": "vendor/fantasma"}); e.ModelCheck.OK || !strings.Contains(strings.Join(e.ModelCheck.Problems, " "), "não está cadastrado") {
		t.Fatalf("%+v", e)
	}
	// Troca de conta no nó para o outro gateway: o modelo da base não existe lá.
	e = get(map[string]any{"connection": vcConn.ID})
	if e.Sources["connection"] != "node" || e.ModelCheck.OK || !strings.Contains(strings.Join(e.ModelCheck.Problems, " "), "vercel") {
		t.Fatalf("%+v", e)
	}
	// Conta e modelo trocados juntos: compatível.
	if e := get(map[string]any{"connection": vcConn.ID, "model": "vendor/vc-tools"}); !e.ModelCheck.OK || e.Sources["model"] != "node" || e.Sources["connection"] != "node" {
		t.Fatalf("%+v", e)
	}
	// A base do catálogo não mudou com nenhuma sobrescrita.
	var after agents.Agent
	c.expect(c.get("/api/agents/"+base.ID), 200).JSON(&after)
	if after.Model != "vendor/tools" || after.Version != base.Version {
		t.Fatalf("base alterada: %+v", after)
	}
}

// A conta do gateway vem do cofre; nada vaza, e conta revogada ou de outro tipo não abre cliente.
func TestGatewayClientOpensFromTheVaultWithoutLeaking(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	token := "TOKEN-GWAPI-77aa-nao-pode-vazar"
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"content": "olá"}, "finish_reason": "stop"}}})
	}))
	defer srv.Close()
	var cn security.Connection
	c.expect(c.post("/api/connections", map[string]string{"kind": "api", "name": "or", "target": "openrouter", "secret": token}), 201).JSON(&cn)

	sealer, err := security.NewSealer(e.cfg.VaultKey, e.cfg.VaultKeyID)
	if err != nil {
		t.Fatal(err)
	}
	vault := security.NewVault(e.db.Pool, sealer, security.NewRedactor())
	ctx := context.Background()
	gw, err := gateways.Open(ctx, vault, cn.ID, srv.URL+"/v1", true)
	if err != nil {
		t.Fatal(err)
	}
	res, err := gw.Chat(ctx, gateways.ChatRequest{Model: "m", Messages: []gateways.Message{{Role: "user", Content: "oi"}}})
	if err != nil || res.Text != "olá" || seen != "Bearer "+token {
		t.Fatalf("%+v %v %q", res, err, seen)
	}
	// Tipo errado e revogada.
	gh := conn(t, c, "github", "gh")
	if _, err := gateways.Open(ctx, vault, gh.ID, srv.URL+"/v1", true); err == nil {
		t.Fatal("conexão github não abre gateway")
	}
	c.expect(c.post("/api/connections/"+cn.ID+"/revoke", nil), 200)
	if _, err := gateways.Open(ctx, vault, cn.ID, srv.URL+"/v1", true); err == nil {
		t.Fatal("conta revogada não abre gateway")
	}
	for _, p := range []string{"/api/models", "/api/connections", "/api/agents"} {
		if strings.Contains(string(c.get(p).Body), token) {
			t.Fatalf("%s vazou o token", p)
		}
	}
	if strings.Contains(e.logs.String(), token) {
		t.Fatal("logs vazaram o token")
	}
}
