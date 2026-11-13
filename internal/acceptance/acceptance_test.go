package acceptance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repo(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func sources(t *testing.T) Sources {
	return Sources{
		Epics: repo(t, "docs/NEW MVP/02-EPICOS.md"), Backlog: repo(t, "docs/NEW MVP/03-BACKLOG.md"),
		Checklist: repo(t, "docs/NEW MVP/07-CHECKLIST.md"), Pilot: repo(t, "docs/NEW MVP/05-ACEITE-E-PILOTO.md"),
		Trace: repo(t, "docs/NEW MVP/04-RASTREABILIDADE.md"),
	}
}

// T17-01 — critérios rastreáveis: o backlog, o checklist, os casos P e a matriz de decisões fecham.
func TestTraceabilityCloses(t *testing.T) {
	if problems := Check(sources(t)); len(problems) > 0 {
		t.Fatalf("rastreabilidade quebrada:\n- %s", strings.Join(problems, "\n- "))
	}
	_, tickets, cases, err := Parse(sources(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(tickets) != 67 || len(cases) != 18 {
		t.Fatalf("%d tickets, %d casos", len(tickets), len(cases))
	}
}

// A matriz versionada é a derivada: depois de mexer no checklist rode `make matriz`.
func TestTheCommittedMatrixIsTheGeneratedOne(t *testing.T) {
	want, err := Render(sources(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := repo(t, "docs/aceite/02-matriz-de-evidencias.md"); got != want {
		t.Fatal("docs/aceite/02-matriz-de-evidencias.md está desatualizada: rode `make matriz` e inclua o arquivo no commit")
	}
}

func TestTheFichaCoversEveryContractField(t *testing.T) {
	s := sources(t)
	if fields := FichaFields(s.Pilot); len(fields) != 12 {
		t.Fatalf("campos da ficha: %v", fields)
	}
	if problems := CheckFicha(s.Pilot, repo(t, "docs/aceite/01-ficha-de-preparacao.md")); len(problems) > 0 {
		t.Fatalf("ficha incompleta:\n- %s", strings.Join(problems, "\n- "))
	}
}

func TestCheckFindsBrokenReferences(t *testing.T) {
	s := sources(t)
	s.Trace += "\n| D99 | x | y | Vigente | T99-99 |\n"
	if p := Check(s); len(p) != 1 || !strings.Contains(p[0], "T99-99") {
		t.Fatalf("%v", p)
	}
	s = sources(t)
	s.Pilot = strings.Replace(s.Pilot, "E17 |", "E99 |", 1)
	if p := Check(s); len(p) == 0 {
		t.Fatal("caso que aponta para épico inexistente")
	}
	if p := CheckFicha(sources(t).Pilot, "nada"); len(p) < 12 {
		t.Fatalf("%v", p)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	a, _ := Render(sources(t))
	b, _ := Render(sources(t))
	if a != b || !strings.Contains(a, "**P18** · Aceite") || !strings.Contains(a, "### E17 — ") {
		t.Fatal("matriz não determinística ou incompleta")
	}
}
