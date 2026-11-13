package techplan

import (
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
)

func task(key, track string, paths []string, deps ...string) planning.Task {
	return planning.Task{Key: key, Title: "Tarefa " + key, Description: "faz " + key, Risk: "medium", Track: track, Paths: paths, DependsOn: deps,
		Criteria: []planning.Criterion{{Text: "os testes de " + key + " passam", Check: planning.CheckTest, Target: "./" + key}}}
}

func has(ps []planning.Problem, field, sub string) bool {
	for _, p := range ps {
		if p.Field == field && strings.Contains(p.Message, sub) {
			return true
		}
	}
	return false
}

func TestMixedIssueSplitsIntoClassifiedUnitsWithDependenciesAndResources(t *testing.T) {
	p := planning.Plan{Summary: "x", Tasks: []planning.Task{
		task("api", "BACKEND", []string{"internal/x/x.go", "contracts/openapi.yaml", "db/migrations/00099_x.sql"}),
		task("tela", "FRONTEND", []string{"apps/web/src/pages/X.tsx"}, "api"),
	}}
	if ps := Validate(p); len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
	subs := Materialize(p)
	if len(subs) != 2 || subs[0].Key != "api" || subs[0].Track != flow.Backend || subs[1].Track != flow.Frontend || subs[1].DependsOn[0] != "api" {
		t.Fatalf("%+v", subs)
	}
	if got := strings.Join(subs[0].Resources, ","); got != "contract:api,schema:db" {
		t.Fatalf("recursos derivados: %s", got)
	}
	if len(subs[1].Resources) != 0 {
		t.Fatalf("%+v", subs[1])
	}
}

func TestRejectsMissingWrongOrMixedTracks(t *testing.T) {
	if ps := Validate(planning.Plan{Tasks: []planning.Task{task("a", "", []string{"internal/a.go"})}}); !has(ps, "track", "classifique") {
		t.Fatalf("%v", ps)
	}
	if ps := Validate(planning.Plan{Tasks: []planning.Task{task("a", "FRONTEND", []string{"internal/a.go"})}}); !has(ps, "track", "não confere") {
		t.Fatalf("%v", ps)
	}
	mixed := task("m", "BACKEND", []string{"internal/a.go", "apps/web/src/A.tsx"})
	if ps := Validate(planning.Plan{Tasks: []planning.Task{mixed}}); !has(ps, "track", "tarefa mista") {
		t.Fatalf("%v", ps)
	}
	// Indivisível: precisa declarar a responsabilidade predominante.
	mixed.Indivisible = true
	if ps := Validate(planning.Plan{Tasks: []planning.Task{mixed}}); len(ps) != 0 {
		t.Fatalf("%v", ps)
	}
	mixed.Track = "FRONTEND"
	if ps := Validate(planning.Plan{Tasks: []planning.Task{mixed}}); !has(ps, "track", "predominante") {
		t.Fatalf("%v", ps)
	}
	// Acesso exclusivo às duas trilhas só para indivisível.
	both := task("b", "BACKEND", []string{"internal/a.go"})
	both.ExclusiveBoth = true
	if ps := Validate(planning.Plan{Tasks: []planning.Task{both}}); !has(ps, "exclusive_both", "indivisível") {
		t.Fatalf("%v", ps)
	}
}

func TestDependencyCycleBlocksPublication(t *testing.T) {
	p := planning.Plan{Tasks: []planning.Task{
		task("a", "BACKEND", []string{"internal/a.go"}, "b"),
		task("b", "BACKEND", []string{"internal/b.go"}, "a"),
	}}
	if ps := Validate(p); !has(ps, "depends_on", "ciclo") {
		t.Fatalf("%v", ps)
	}
}
