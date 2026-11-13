// Package qualitypolicy aplica a política de testes, risco e review por trilha (AJ08-01).
//
//   - Todas as tarefas: testes automatizados reais e checks aplicáveis, com comandos, resultados e
//     versão do código (os checks aplicáveis seguem obrigatórios no Backend de risco baixo/médio).
//   - Frontend: review visual em navegador isolado (MCP) sempre; sem a evidência, o gate fica pendente.
//   - Backend: review por agente de IA só em alto risco, com o risco reavaliado sobre o diff real.
//   - Em ambas as trilhas a avaliação inicial é independente.
//   - Toda evidência é ligada à revisão do código: o que mudou depois dela deixa de valer.
package qualitypolicy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
)

// Input é o que a decisão precisa.
type Input struct {
	Track     flow.Track
	Suggested string // risco sugerido (plano/IA); a reavaliação sobre o diff só eleva
	Policy    risk.Policy
	Files     []risk.File // diff real
	Available []string    // checks que o projeto sabe executar
}

// Requirement é o que a tarefa precisa comprovar.
type Requirement struct {
	Track              flow.Track      `json:"track"`
	Assessment         risk.Assessment `json:"assessment"`
	Tests              bool            `json:"tests"` // sempre verdadeiro
	Checks             []string        `json:"checks"`
	NotApplicable      []string        `json:"not_applicable"` // exigem justificativa registrada
	BrowserReview      bool            `json:"browser_review"`
	AIReview           bool            `json:"ai_review"`
	IndependentInitial bool            `json:"independent_initial"`
}

// Decide aplica a política. O risco é sempre reavaliado sobre o diff.
func Decide(in Input) Requirement {
	a := risk.Assess(in.Policy, in.Suggested, in.Files)
	plan := risk.Plan(a.Final, in.Available)
	r := Requirement{Track: in.Track, Assessment: a, Tests: true, Checks: plan.Required, NotApplicable: plan.NotApplicable, IndependentInitial: true}
	switch in.Track {
	case flow.Frontend:
		r.BrowserReview = true // sempre, qualquer risco
		r.AIReview = a.Final == risk.High
	default:
		r.AIReview = a.Final == risk.High
	}
	return r
}

// TestRun é a execução real dos testes para uma revisão do código.
type TestRun struct {
	Revision string   `json:"revision"`
	Commands []string `json:"commands"`
	Passed   bool     `json:"passed"`
	Errors   []string `json:"errors,omitempty"`
}

// Review é o resultado de um review (visual ou de IA) para uma revisão.
type Review struct {
	Revision    string `json:"revision"`
	Approved    bool   `json:"approved"`
	Independent bool   `json:"independent"` // sessão inicial separada do implementador
	Pending     bool   `json:"pending"`     // ex.: navegador/MCP indisponível
}

// Evidence é o que foi comprovado.
type Evidence struct {
	Tests          *TestRun          `json:"tests,omitempty"`
	ChecksPassed   map[string]string `json:"checks_passed,omitempty"` // check → revisão em que passou
	Justifications map[string]string `json:"justifications,omitempty"`
	Browser        *Review           `json:"browser,omitempty"`
	AI             *Review           `json:"ai,omitempty"`
}

// Problem é um motivo de bloqueio.
type Problem struct{ Code, Message string }

func (p Problem) String() string { return p.Code + ": " + p.Message }

// Verify devolve o que impede aprovar a revisão `revision`. Vazio = todos os gates do nível passaram.
func Verify(r Requirement, e Evidence, revision string) []Problem {
	var out []Problem
	add := func(code, msg string) { out = append(out, Problem{code, msg}) }
	switch {
	case e.Tests == nil:
		add("tests_missing", "os testes automatizados reais ainda não foram executados")
	case e.Tests.Revision != revision:
		add("tests_stale", fmt.Sprintf("os testes foram executados em %s, não na revisão atual %s", e.Tests.Revision, revision))
	case len(e.Tests.Commands) == 0:
		add("tests_missing", "a evidência de testes não traz os comandos executados")
	case !e.Tests.Passed:
		add("tests_failed", "os testes falharam: "+strings.Join(e.Tests.Errors, "; "))
	}
	for _, c := range r.Checks {
		switch rev, ok := e.ChecksPassed[c]; {
		case !ok:
			add("check_missing", "check obrigatório sem resultado: "+c)
		case rev != revision:
			add("check_stale", fmt.Sprintf("o check %s passou em %s, não na revisão atual", c, rev))
		}
	}
	for _, c := range r.NotApplicable {
		if strings.TrimSpace(e.Justifications[c]) == "" {
			add("check_unjustified", "check previsto sem comando no projeto exige justificativa: "+c)
		}
	}
	if r.BrowserReview {
		out = append(out, reviewProblems("browser", "review visual no navegador", e.Browser, revision, r.IndependentInitial)...)
	}
	if r.AIReview {
		out = append(out, reviewProblems("ai", "review independente por IA", e.AI, revision, r.IndependentInitial)...)
	}
	return out
}

func reviewProblems(code, label string, rv *Review, revision string, independent bool) []Problem {
	switch {
	case rv == nil:
		return []Problem{{code + "_missing", label + " obrigatório ainda não feito"}}
	case rv.Pending:
		return []Problem{{code + "_pending", label + " pendente (falha de prontidão ou do MCP não aprova)"}}
	case rv.Revision != revision:
		return []Problem{{code + "_stale", fmt.Sprintf("%s foi feito em %s, não na revisão atual %s", label, rv.Revision, revision)}}
	case independent && !rv.Independent:
		return []Problem{{code + "_not_independent", label + " exige sessão independente na avaliação inicial"}}
	case !rv.Approved:
		return []Problem{{code + "_rejected", label + " reprovou a revisão"}}
	}
	return nil
}

// Affected lista os gates que uma correção obriga a repetir, dado o diff da correção. Testes e checks
// sempre; o review visual quando a trilha é Frontend; o de IA quando exigido pelo risco.
func Affected(r Requirement, changed []string) []string {
	if len(changed) == 0 {
		return nil
	}
	out := []string{"tests"}
	out = append(out, r.Checks...)
	if r.BrowserReview {
		out = append(out, "browser")
	}
	if r.AIReview {
		out = append(out, "ai")
	}
	sort.Strings(out[1:])
	return out
}
