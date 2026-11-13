package agents

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

const conn1 = "11111111-1111-4111-8111-111111111111"
const conn2 = "22222222-2222-4222-8222-222222222222"

func base() Agent {
	c := conn1
	return Agent{ID: "a1", Name: "dev", Version: 4, Instructions: "Seja objetivo.", Provider: ProviderClaude, ConnectionID: &c,
		Model: "claude-sonnet-5-5", Skills: []string{"revisao"}, Tools: []string{"Read", "Edit", "Bash"}, TimeoutMinutes: 30,
		Fallback: []FallbackStep{{ConnectionID: conn2, Model: "m"}}}
}

func node(props map[string]any) workflows.Node {
	props["agent"] = "dev"
	return workflows.Node{ID: "n1", Type: workflows.TAgent, Props: props}
}

func TestResolveKeepsBaseAndAppliesNodeOverrides(t *testing.T) {
	b := base()
	snapshot := base()

	// Sem sobrescritas: tudo vem do catálogo.
	e, err := Resolve(b, node(map[string]any{"prompt": "Implemente X"}))
	if err != nil {
		t.Fatal(err)
	}
	if e.Model != b.Model || *e.ConnectionID != conn1 || e.TimeoutMinutes != 30 || e.Prompt != "Implemente X" || e.AgentVersion != 4 {
		t.Fatalf("%+v", e)
	}
	for k, v := range e.Sources {
		if v != FromAgent {
			t.Errorf("%s deveria vir da base: %s", k, v)
		}
	}

	// Com sobrescritas: o efetivo muda, a origem diz de onde veio.
	e, err = Resolve(b, node(map[string]any{
		"model": "claude-opus-5-5", "connection": conn2, "tools": "Read, Edit", "timeout_minutes": float64(10), "instructions_extra": "Foque em testes.",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if e.Model != "claude-opus-5-5" || *e.ConnectionID != conn2 || !reflect.DeepEqual(e.Tools, []string{"Read", "Edit"}) || e.TimeoutMinutes != 10 ||
		e.Instructions != "Seja objetivo.\n\nFoque em testes." {
		t.Fatalf("%+v", e)
	}
	for _, k := range []string{"model", "connection", "tools", "timeout_minutes", "instructions"} {
		if e.Sources[k] != FromNode {
			t.Errorf("%s deveria vir do nó", k)
		}
	}
	if e.Sources["skills"] != FromAgent || e.Sources["provider"] != FromAgent {
		t.Error("o que não foi sobrescrito segue vindo da base")
	}

	// A base continua exatamente como estava (nada compartilhado por referência).
	e.Skills[0], e.Tools[0], e.Fallback[0].Model = "mudou", "mudou", "mudou"
	if !reflect.DeepEqual(b, snapshot) {
		t.Fatalf("a base do catálogo foi alterada: %+v", b)
	}
}

func TestResolveRefusesToWidenToolsOrUseBadValues(t *testing.T) {
	b := base()
	for name, props := range map[string]map[string]any{
		"amplia ferramentas": {"tools": "Read, Write"},
		"ferramenta nova":    {"tools": "Deploy"},
		"modelo inválido":    {"model": "x; rm -rf /"},
		"conexão inválida":   {"connection": "../etc"},
	} {
		if _, err := Resolve(b, node(props)); !errors.Is(err, ErrOverride) {
			t.Errorf("%s: deveria ser recusado (%v)", name, err)
		}
	}
	// Reduzir para nenhuma ferramenta é permitido (lista vazia explícita não existe: campo vazio = sem sobrescrita).
	if e, err := Resolve(b, node(map[string]any{"tools": "Read"})); err != nil || len(e.Tools) != 1 {
		t.Fatalf("reduzir: %+v %v", e, err)
	}
}

func TestInputValidation(t *testing.T) {
	ok := Input{Name: "dev", Instructions: "x", Provider: ProviderClaude}
	if err := (&ok).normalize(); err != nil || ok.TimeoutMinutes != 30 || ok.Skills == nil || ok.Fallback == nil {
		t.Fatalf("%+v %v", ok, err)
	}
	for name, mut := range map[string]func(*Input){
		"nome com maiúscula": func(i *Input) { i.Name = "Dev" },
		"nome vazio":         func(i *Input) { i.Name = "" },
		"sem instruções":     func(i *Input) { i.Instructions = "  " },
		"provedor":           func(i *Input) { i.Provider = "gemini" },
		"modelo":             func(i *Input) { i.Model = "a b" },
		"conexão":            func(i *Input) { c := "x"; i.ConnectionID = &c },
		"skill":              func(i *Input) { i.Skills = []string{"a b"} },
		"tool":               func(i *Input) { i.Tools = []string{"x;y"} },
		"timeout":            func(i *Input) { i.TimeoutMinutes = 9999 },
		"fallback":           func(i *Input) { i.Fallback = []FallbackStep{{ConnectionID: "x"}} },
		"fallback demais":    func(i *Input) { i.Fallback = make([]FallbackStep, 6) },
	} {
		in := Input{Name: "dev", Instructions: "x", Provider: ProviderClaude}
		mut(&in)
		if err := in.normalize(); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
}
