package delivery

import (
	"strings"
	"testing"
)

func TestDocsGapsPerChangedModule(t *testing.T) {
	rules := DefaultDocRules()
	gaps := func(files ...string) []string {
		var out []string
		for _, g := range DocsGaps(files, rules) {
			out = append(out, g.Module+" → "+strings.Join(g.Expected, " ou "))
		}
		return out
	}
	// Só teste, documentação ou arquivos fora dos módulos não pedem documentação do módulo.
	if g := gaps("internal/x/x_test.go", "docs/a.md", "db/migrations/1.sql", "cmd/api/main.go", "README.md"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
	g := gaps("internal/pedidos/pedidos.go", "internal/pedidos/helper.go", "internal/frete/frete.go")
	if len(g) != 2 || !strings.HasPrefix(g[0], "internal/frete → internal/frete/doc.go ou docs/modulos/frete.md") || !strings.HasPrefix(g[1], "internal/pedidos") {
		t.Fatalf("um por módulo, em ordem estável: %v", g)
	}
	// Qualquer documentação aceita resolve só o seu módulo.
	if g := gaps("internal/pedidos/pedidos.go", "internal/pedidos/doc.go", "internal/frete/frete.go", "docs/modulos/frete.md"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
	if g := gaps("internal/pedidos/pedidos.go", "internal/frete/doc.go"); len(g) != 1 || !strings.HasPrefix(g[0], "internal/pedidos") {
		t.Fatalf("%v", g)
	}
	// Painel: área da UI.
	if g := gaps("apps/web/src/pages/A.tsx"); len(g) != 1 || !strings.Contains(g[0], "docs/modulos/web-pages.md") {
		t.Fatalf("%v", g)
	}
	if g := gaps("apps/web/src/pages/A.tsx", "docs/modulos/web.md"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
}

func TestFeatureDocPathIsSafeAndStable(t *testing.T) {
	for in, want := range map[string]string{
		"ABC-123":       "docs/features/abc-123.md",
		"../../etc/x":   "docs/features/etc-x.md",
		"Feature Nova!": "docs/features/feature-nova.md",
	} {
		if got := FeatureDocPath(in); got != want {
			t.Errorf("%q → %q (esperado %q)", in, got, want)
		}
	}
}
