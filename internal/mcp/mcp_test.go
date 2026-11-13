package mcp

import (
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"strings"
	"testing"
)

func TestSchemaHashIgnoresKeyOrderAndDescriptionButNotContract(t *testing.T) {
	a := ToolDef{Name: "t", Description: "x", InputSchema: map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "string"}, "b": map[string]any{"type": "integer"}}}}
	b := ToolDef{Name: "t", Description: "outra", InputSchema: map[string]any{"properties": map[string]any{"b": map[string]any{"type": "integer"}, "a": map[string]any{"type": "string"}}, "type": "object"}}
	if a.SchemaHash() != b.SchemaHash() {
		t.Fatal("ordem de chaves e descrição não mudam o contrato")
	}
	c := b
	c.InputSchema = map[string]any{"type": "object", "properties": map[string]any{"a": map[string]any{"type": "number"}}}
	if a.SchemaHash() == c.SchemaHash() {
		t.Fatal("mudar o tipo muda o contrato")
	}
	d := a
	d.Name = "u"
	if a.SchemaHash() == d.SchemaHash() {
		t.Fatal("o nome faz parte do contrato")
	}
}

func TestCleanTextDropsControlCharactersAndLimitsSize(t *testing.T) {
	if got := CleanText("a\x00b\x1b[31mc\nd\te", 100); got != "ab[31mc\nd\te" {
		t.Fatalf("%q", got)
	}
	if got := CleanText(strings.Repeat("x", 50), 10); len(got) != 10 {
		t.Fatalf("%d", len(got))
	}
}

func TestClientRefusesUnsafeURLs(t *testing.T) {
	for _, u := range []string{"http://exemplo.com", "https://u:p@exemplo.com", "https://127.0.0.1", "https://localhost", "https://169.254.169.254", "ftp://exemplo.com", ""} {
		if _, err := NewClient(u, security.Secret{}, false); err == nil {
			t.Errorf("%q deveria ser recusada", u)
		}
	}
	if _, err := NewClient("https://exemplo.com/mcp", security.Secret{}, false); err != nil {
		t.Fatal(err)
	}
}

func TestValidateArgs(t *testing.T) {
	schema := map[string]any{"type": "object", "required": []any{"q"}, "properties": map[string]any{
		"q":    map[string]any{"type": "string", "maxLength": float64(3)},
		"n":    map[string]any{"type": "integer", "minimum": float64(1), "maximum": float64(5)},
		"modo": map[string]any{"type": "string", "enum": []any{"a", "b"}},
		"ok":   map[string]any{"type": "boolean"},
	}}
	good := []map[string]any{{"q": "abc"}, {"q": "a", "n": float64(5), "modo": "b", "ok": true}}
	for _, g := range good {
		if err := ValidateArgs(schema, g); err != nil {
			t.Errorf("%v: %v", g, err)
		}
	}
	bad := []map[string]any{
		{}, {"q": "abcd"}, {"q": 1}, {"q": "a", "n": 0.5}, {"q": "a", "n": float64(9)},
		{"q": "a", "modo": "c"}, {"q": "a", "ok": "sim"}, {"q": "a", "extra": 1},
	}
	for _, b := range bad {
		if err := ValidateArgs(schema, b); err == nil {
			t.Errorf("%v deveria falhar", b)
		}
	}
	open := map[string]any{"type": "object", "additionalProperties": true}
	if err := ValidateArgs(open, map[string]any{"x": 1}); err != nil {
		t.Errorf("additionalProperties:true aceita extras: %v", err)
	}
}
