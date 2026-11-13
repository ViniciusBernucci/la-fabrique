package planning

import (
	"strings"
	"testing"
)

func crit() []Criterion {
	return []Criterion{{Text: "Os testes do pacote passam", Check: CheckTest, Target: "./internal/x"}}
}

func task(key string, deps ...string) Task {
	return Task{Key: key, Title: "Tarefa " + key, Risk: "low", Criteria: crit(), DependsOn: deps}
}

func problemsText(ps []Problem) string {
	var s []string
	for _, p := range ps {
		s = append(s, p.String())
	}
	return strings.Join(s, " | ")
}

func TestValidPlanPassesAndOrdersTopologically(t *testing.T) {
	p := Plan{Tasks: []Task{task("c", "a", "b"), task("a"), {Key: "b", Title: "B", Risk: "high", Criteria: crit(), DependsOn: []string{"a"}, Order: 1}}}
	if ps := Validate(p); len(ps) != 0 {
		t.Fatal(problemsText(ps))
	}
	if got := strings.Join(Order(p), ","); got != "a,b,c" {
		t.Fatalf("ordem: %s", got)
	}
	// Desempate pela estimativa de ordem e depois pela chave.
	q := Plan{Tasks: []Task{{Key: "z", Title: "Z", Risk: "low", Criteria: crit(), Order: 1}, {Key: "m", Title: "M", Risk: "low", Criteria: crit(), Order: 2}, {Key: "a", Title: "A", Risk: "low", Criteria: crit(), Order: 2}}}
	if got := strings.Join(Order(q), ","); got != "z,a,m" {
		t.Fatalf("desempate: %s", got)
	}
}

func TestValidateRefusesCyclesMissingReferencesAndSelfDependency(t *testing.T) {
	cycle := Plan{Tasks: []Task{task("a", "c"), task("b", "a"), task("c", "b")}}
	ps := Validate(cycle)
	if len(ps) != 1 || !strings.Contains(ps[0].Message, "ciclo") || !strings.Contains(ps[0].Message, "→") {
		t.Fatalf("%s", problemsText(ps))
	}
	if ps := Validate(Plan{Tasks: []Task{task("a", "fantasma")}}); !strings.Contains(problemsText(ps), "referência ausente") {
		t.Fatal(problemsText(ps))
	}
	if ps := Validate(Plan{Tasks: []Task{task("a", "a")}}); !strings.Contains(problemsText(ps), "si mesma") {
		t.Fatal(problemsText(ps))
	}
	if ps := Validate(Plan{Tasks: []Task{task("a"), task("a")}}); !strings.Contains(problemsText(ps), "repetida") {
		t.Fatal(problemsText(ps))
	}
	if ps := Validate(Plan{}); len(ps) != 1 {
		t.Fatal(problemsText(ps))
	}
}

func TestValidateRefusesCriteriaThatCannotBeVerified(t *testing.T) {
	bad := func(c Criterion) string {
		return problemsText(Validate(Plan{Tasks: []Task{{Key: "a", Title: "A", Risk: "low", Criteria: []Criterion{c}}}}))
	}
	cases := map[string]Criterion{
		"sem tipo":          {Text: "Funciona direitinho para o usuário"},
		"tipo desconhecido": {Text: "Funciona direitinho para o usuário", Check: "vibes"},
		"texto curto":       {Text: "ok", Check: CheckTest, Target: "x"},
		"comando vazio":     {Text: "Compila sem erros no CI", Check: CheckCommand},
		"teste sem alvo":    {Text: "Os testes passam sempre", Check: CheckTest},
		"arquivo absoluto":  {Text: "O arquivo de saída existe", Check: CheckFile, Path: "/etc/passwd"},
		"arquivo com ..":    {Text: "O arquivo de saída existe", Check: CheckFile, Path: "a/../../b"},
		"revisão sem prova": {Text: "Interface fica agradável", Check: CheckReview, Evidence: "ok"},
	}
	for name, c := range cases {
		if bad(c) == "" {
			t.Errorf("%s: deveria ser recusado", name)
		}
	}
	if bad(Criterion{Text: "Interface mostra o erro em vermelho", Check: CheckReview, Evidence: "captura de tela do formulário com erro"}) != "" {
		t.Fatal("revisão com evidência observável é verificável")
	}
	if ps := Validate(Plan{Tasks: []Task{{Key: "a", Title: "A", Risk: "low"}}}); !strings.Contains(problemsText(ps), "critério") {
		t.Fatal("tarefa sem critério")
	}
	if ps := Validate(Plan{Tasks: []Task{{Key: "a", Title: "A", Criteria: crit()}}}); !strings.Contains(problemsText(ps), "risco") {
		t.Fatal("sem risco")
	}
	if ps := Validate(Plan{Tasks: []Task{{Key: "A!", Title: "A", Risk: "low", Criteria: crit()}}}); ps == nil {
		t.Fatal("chave inválida")
	}
}

func TestHashIgnoresOrderOfTasksAndDependencies(t *testing.T) {
	a := Plan{Summary: " resumo ", Tasks: []Task{task("b", "a", "c"), task("a"), task("c")}}
	b := Plan{Summary: "resumo", Tasks: []Task{task("c"), task("a"), task("b", "c", "a")}}
	if Hash(a) != Hash(b) {
		t.Fatal("o hash não pode depender da ordem de escrita")
	}
	c := b
	c.Tasks = append([]Task{}, b.Tasks...)
	c.Tasks[0].Title = "outro"
	if Hash(a) == Hash(c) {
		t.Fatal("conteúdo diferente, hash diferente")
	}
	x, y := task("a"), task("a")
	y.Order = 9
	if TaskHash(x) != TaskHash(y) {
		t.Fatal("a ordem estimada não faz parte da identidade de conteúdo da tarefa")
	}
}
