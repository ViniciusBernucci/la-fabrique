package factory

import (
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/agents"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/techplan"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

func TestPlannerIsRecognizedBySkillAndTrackByNodeText(t *testing.T) {
	if !isPlanner(agents.Effective{Skills: []string{"x", "lf-tech-lead"}}) || isPlanner(agents.Effective{Skills: []string{"lf-implementar-tarefa"}}) {
		t.Fatal("o planejador é o agente com a skill lf-tech-lead")
	}
	n := func(id, prompt string) workflows.Node {
		return workflows.Node{ID: id, Type: "agent", Props: map[string]any{"prompt": prompt}}
	}
	cases := []struct {
		node  workflows.Node
		agent string
		want  string
	}{
		{n("guard_impl_front", "Implementar as subtarefas FRONTEND."), "implementador-frontend", "FRONTEND"},
		{n("guard_impl_back", "Implementar as subtarefas BACKEND."), "implementador-backend", "BACKEND"},
		{n("x", "revisar"), "revisor-frontend", "FRONTEND"},
		{n("guard_planning", "Planejar a feature"), "tech-lead", ""},
		{n("misto", "frontend e backend"), "dev", ""},
	}
	for _, c := range cases {
		if got := trackOfNode(c.node, agents.Effective{AgentName: c.agent}); got != c.want {
			t.Errorf("%s/%s: trilha %q, esperado %q", c.node.ID, c.agent, got, c.want)
		}
	}
}

// O formato que a fábrica ensina ao Tech Lead precisa ser aceito pelo validador real: se o schema do prompt
// divergir do planning/techplan, o plano nunca passaria.
func TestPlannerFormatExampleMatchesTheRealValidator(t *testing.T) {
	plan := planning.Plan{Summary: "x", Tasks: []planning.Task{
		{Key: "backend-saude", Title: "Endpoint", Track: "BACKEND", Risk: "low", Paths: []string{"fabrica-teste/backend/saude.md"},
			Criteria: []planning.Criterion{{Text: "o arquivo existe", Check: planning.CheckFile, Path: "fabrica-teste/backend/saude.md"}}},
		{Key: "frontend-selo", Title: "Selo", Track: "FRONTEND", Risk: "low", DependsOn: []string{"backend-saude"}, Paths: []string{"fabrica-teste/frontend/selo-saude.html"},
			Criteria: []planning.Criterion{{Text: "a página existe", Check: planning.CheckFile, Path: "fabrica-teste/frontend/selo-saude.html"}}},
	}}
	if ps := techplan.Validate(plan); len(ps) != 0 {
		t.Fatalf("o plano de exemplo deveria ser válido: %v", ps)
	}
	for _, want := range []string{planFile, `"track"`, `"criteria"`, "FRONTEND", "BACKEND", `"check"`, "depends_on"} {
		if !strings.Contains(plannerFormat, want) {
			t.Errorf("o formato do plano deveria mencionar %q", want)
		}
	}
	// plano misto sem indivisível é recusado com o motivo
	bad := planning.Plan{Tasks: []planning.Task{{Key: "misto", Title: "Misto", Track: "BACKEND", Risk: "low",
		Paths:    []string{"api/x.go", "apps/web/y.tsx"},
		Criteria: []planning.Criterion{{Text: "algo verificável", Check: planning.CheckCommand, Command: "make test"}}}}}
	if ps := techplan.Validate(bad); len(ps) == 0 {
		t.Fatal("tarefa mista deveria ser recusada")
	}
}
