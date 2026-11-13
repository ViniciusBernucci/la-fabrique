package e2e

import (
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
)

func f(paths ...string) []risk.File {
	var out []risk.File
	for _, p := range paths {
		out = append(out, risk.File{Path: p, Added: 5})
	}
	return out
}

func TestE2EIsChosenByRiskAndByImpactOnTheUI(t *testing.T) {
	cfg := DefaultConfig()
	if d := Decide(risk.Low, f("apps/web/src/App.tsx"), cfg); d.Required {
		t.Fatalf("%+v", d)
	}
	if d := Decide(risk.Medium, f("internal/x/x.go"), cfg); d.Required {
		t.Fatalf("médio sem UI: %+v", d)
	}
	if d := Decide(risk.Medium, f("apps/web/src/pages/A.tsx"), cfg); !d.Required || !strings.Contains(d.Reason, "UI") {
		t.Fatalf("médio com UI: %+v", d)
	}
	if d := Decide(risk.High, f("internal/x/x.go"), cfg); !d.Required {
		t.Fatalf("alto sempre exige: %+v", d)
	}
	if d := Decide("?", nil, cfg); !d.Required {
		t.Fatal("nível desconhecido: falha fechada")
	}
	custom := Config{UIPaths: []string{"frontend/**"}}
	if !custom.TouchesUI(f("frontend/a.vue")) || custom.TouchesUI(f("apps/web/a.tsx")) {
		t.Fatal("UI configurável")
	}
}

func TestTestGuidanceAsksForATestPerChangedCodeFile(t *testing.T) {
	gaps := func(paths ...string) []string {
		var out []string
		for _, g := range TestGaps(f(paths...)) {
			out = append(out, g.File+"→"+g.Expected)
		}
		return out
	}
	// Go: um *_test.go no mesmo diretório basta.
	if g := gaps("internal/pedidos/pedidos.go", "internal/pedidos/pedidos_test.go"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
	if g := gaps("internal/pedidos/pedidos.go"); len(g) != 1 || g[0] != "internal/pedidos/pedidos.go→internal/pedidos/pedidos_test.go" {
		t.Fatalf("%v", g)
	}
	// TS: o .test do arquivo.
	if g := gaps("apps/web/src/A.tsx", "apps/web/src/A.test.tsx"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
	if g := gaps("apps/web/src/B.tsx"); len(g) != 1 || !strings.HasSuffix(g[0], "B.test.tsx") {
		t.Fatalf("%v", g)
	}
	// Python.
	if g := gaps("app/svc.py"); len(g) != 1 || !strings.HasSuffix(g[0], "test_svc.py") {
		t.Fatalf("%v", g)
	}
	if g := gaps("app/svc.py", "tests/test_svc.py"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
	// Documentação, migration, scripts, comandos e os próprios testes não pedem teste.
	if g := gaps("docs/a.md", "db/migrations/1.sql", "scripts/x.sh", "cmd/api/main.go", "internal/x/doc.go", "internal/x/x_test.go", "tests/integration/a_test.go"); len(g) != 0 {
		t.Fatalf("%v", g)
	}
}
