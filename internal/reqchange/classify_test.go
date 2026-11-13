package reqchange

import (
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/resume"
)

func req() resume.Requirement {
	return resume.Requirement{Title: "Exportar CSV", Description: "Relatórios", Status: "To Do", Priority: "High", Labels: []string{"a", "b"}, Dependencies: []string{"X-1"}}
}

func TestClassifySeparatesMetadataFromContent(t *testing.T) {
	cases := map[string]struct {
		mut    func(*resume.Requirement)
		impact string
	}{
		"igual":             {func(*resume.Requirement) {}, ""},
		"ordem dos rótulos": {func(r *resume.Requirement) { r.Labels = []string{"b", "a"} }, ""},
		"espaços":           {func(r *resume.Requirement) { r.Title = "  Exportar CSV " }, ""},
		"prioridade":        {func(r *resume.Requirement) { r.Priority = "Low" }, ImpactMetadata},
		"rótulo":            {func(r *resume.Requirement) { r.Labels = []string{"a"} }, ImpactMetadata},
		"estado":            {func(r *resume.Requirement) { r.Status = "In Progress" }, ImpactMetadata},
		"título":            {func(r *resume.Requirement) { r.Title = "Exportar XML" }, ImpactContent},
		"descrição":         {func(r *resume.Requirement) { r.Description = "Outra" }, ImpactContent},
		"dependência":       {func(r *resume.Requirement) { r.Dependencies = []string{"X-2"} }, ImpactContent},
		"misto":             {func(r *resume.Requirement) { r.Priority = "Low"; r.Title = "Outro" }, ImpactContent},
	}
	for name, c := range cases {
		cur := req()
		c.mut(&cur)
		got, _ := Classify(req(), cur)
		if got != c.impact {
			t.Errorf("%s: %q, esperado %q", name, got, c.impact)
		}
	}
	a, b := req(), req()
	b.Labels = []string{"b", "a"}
	if HashOf(a) != HashOf(b) {
		t.Fatal("o hash ignora a ordem")
	}
}
