package risk

import (
	"strings"
	"testing"
)

func TestGlobMatching(t *testing.T) {
	cases := []struct {
		pat, path string
		want      bool
	}{
		{"**/migrations/**", "db/migrations/00031_risk.sql", true},
		{"**/migrations/**", "migrations/x.sql", true},
		{"**/migrations/**", "db/migration/x.sql", false},
		{"**/*.sql", "a/b/c.sql", true},
		{"**/*.sql", "c.sql", true},
		{"**/*.sql", "c.sqlx", false},
		{"go.mod", "go.mod", true},
		{"go.mod", "sub/go.mod", false},
		{"**/package.json", "apps/web/package.json", true},
		{".github/workflows/**", ".github/workflows/ci.yml", true},
		{"**/Dockerfile*", "deploy/Dockerfile.prod", true},
		{"**/*secret*", "internal/security/secrets_test.go", true},
		{"**/auth/**", "internal/auth/login.go", true},
		{"**/auth/**", "internal/authz/login.go", false},
		{"a/?/c", "a/b/c", true},
		{"a/?/c", "a/bb/c", false},
		{"**/.env*", "apps/api/.env.local", true},
	}
	for _, c := range cases {
		if got := Match(c.pat, c.path); got != c.want {
			t.Errorf("Match(%q, %q) = %v", c.pat, c.path, got)
		}
	}
}

// T14-02 — "Fixture de arquivo crítico eleva nível apesar de sugestão baixa."
func TestACriticalFileRaisesTheLevelDespiteALowSuggestion(t *testing.T) {
	a := Assess(DefaultPolicy(), Low, []File{{Path: "docs/guia.md", Added: 5}, {Path: "db/migrations/00099_x.sql", Added: 10}})
	if a.Final != High || !a.Raised || a.Suggested != Low {
		t.Fatalf("%+v", a)
	}
	var ai, rule bool
	for _, r := range a.Reasons {
		ai = ai || r.Source == "ai"
		rule = rule || (r.Source == "rule" && r.Level == High && strings.Contains(r.Detail, "00099_x.sql"))
	}
	if !ai || !rule {
		t.Fatalf("a justificativa lista a IA e a regra: %+v", a.Reasons)
	}
	// Só documentação: a sugestão baixa vale.
	if d := Assess(DefaultPolicy(), Low, []File{{Path: "docs/guia.md", Added: 5}}); d.Final != Low || d.Raised {
		t.Fatalf("%+v", d)
	}
	// Os pisos só elevam: sugestão alta continua alta sem arquivo crítico.
	if h := Assess(DefaultPolicy(), High, []File{{Path: "docs/guia.md"}}); h.Final != High || h.Raised {
		t.Fatalf("%+v", h)
	}
	// Dependência: piso médio.
	if m := Assess(DefaultPolicy(), Low, []File{{Path: "apps/web/package.json"}}); m.Final != Medium || !m.Raised {
		t.Fatalf("%+v", m)
	}
	// Sugestão ausente ou inválida: falha fechada (alta).
	if x := Assess(DefaultPolicy(), "", nil); x.Final != High || x.Raised {
		t.Fatalf("%+v", x)
	}
}

func TestSizeThresholdsAndStrongestRuleWins(t *testing.T) {
	p := DefaultPolicy()
	if a := Assess(p, Low, []File{{Path: "a.go", Added: 150, Removed: 60}}); a.Final != Medium {
		t.Fatalf("%+v", a)
	}
	if a := Assess(p, Low, []File{{Path: "a.go", Added: 700, Removed: 150}}); a.Final != High {
		t.Fatalf("%+v", a)
	}
	many := make([]File, 12)
	for i := range many {
		many[i] = File{Path: "pkg/f" + string(rune('a'+i)) + ".go"}
	}
	if a := Assess(p, Low, many); a.Final != Medium {
		t.Fatalf("12 arquivos: %+v", a)
	}
	// Dependência (médio) e migration (alto) no mesmo diff: vale a mais forte, com as duas razões.
	a := Assess(p, Low, []File{{Path: "go.mod"}, {Path: "db/migrations/1.sql"}})
	if a.Final != High {
		t.Fatalf("%+v", a)
	}
	n := 0
	for _, r := range a.Reasons {
		if r.Source == "rule" {
			n++
		}
	}
	if n < 3 { // go.mod, migrations/**, **/*.sql
		t.Fatalf("as razões devem listar todas as regras: %+v", a.Reasons)
	}
	// Política do projeto: caminho próprio crítico e limiares desligados.
	custom := Policy{Rules: []Rule{{"billing/**", High, "cobrança"}}}
	if a := Assess(custom, Low, []File{{Path: "billing/invoice.go", Added: 5000}}); a.Final != High {
		t.Fatalf("%+v", a)
	}
	if a := Assess(custom, Low, []File{{Path: "outro/x.go", Added: 5000}}); a.Final != Low {
		t.Fatalf("limiar desligado: %+v", a)
	}
}

// T14-02 — "Três níveis escolhem verificações previstas."
func TestTheThreeLevelsChooseDifferentVerifications(t *testing.T) {
	all := []string{"lint", "build", "test", "secret_scan", "dependency_audit", "static_analysis"}
	low, med, high := Plan(Low, all), Plan(Medium, all), Plan(High, all)
	if strings.Join(low.Required, ",") != "lint,build,test" || low.Review != "none" || low.E2E != "no" {
		t.Fatalf("%+v", low)
	}
	if strings.Join(med.Required, ",") != "lint,build,test" || med.Review != "standard" || med.E2E != "if_ui" {
		t.Fatalf("%+v", med)
	}
	if len(high.Required) != 6 || high.Review != "independent_security" || high.E2E != "required" || len(high.NotApplicable) != 0 {
		t.Fatalf("%+v", high)
	}
	// Alto sem os checks de segurança no projeto: ficam como não aplicáveis (exigem justificativa).
	h2 := Plan(High, []string{"lint", "build", "test"})
	if strings.Join(h2.NotApplicable, ",") != "secret_scan,dependency_audit,static_analysis" || len(h2.Required) != 3 {
		t.Fatalf("%+v", h2)
	}
	// Nível inválido: falha fechada.
	if Plan("?", all).Level != High {
		t.Fatal("falha fechada")
	}
	// Básico ausente também é apontado.
	if p := Plan(Low, []string{"lint"}); strings.Join(p.NotApplicable, ",") != "build,test" {
		t.Fatalf("%+v", p)
	}
}

func TestPolicyValidation(t *testing.T) {
	bad := []Policy{
		{Rules: []Rule{{"", High, ""}}},
		{Rules: []Rule{{"/abs/**", High, ""}}},
		{Rules: []Rule{{"../x", High, ""}}},
		{Rules: []Rule{{"a/**", Low, ""}}},
		{MediumLines: 500, HighLines: 100},
		{MediumFiles: 30, HighFiles: 10},
		{MediumLines: -1},
	}
	for i, p := range bad {
		if p.Validate() == nil {
			t.Errorf("política %d deveria ser recusada", i)
		}
	}
	if err := DefaultPolicy().Validate(); err != nil {
		t.Fatal(err)
	}
}
