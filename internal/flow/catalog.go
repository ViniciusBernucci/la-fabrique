package flow

import (
	"sort"
	"strings"
)

// Códigos canônicos.
const (
	PRDDoc          = "PRD-DOC"
	PRDDocConcluido = "PRD-DOC-CONCLUIDO"
	Planejamento    = "PLANEJAMENTO"
	Implementacao   = "IMPLEMENTACAO"
	Review          = "REVIEW"
	DraftPR         = "DRAFT_PR"
	Concluido       = "CONCLUIDO"
	ReprovadoPR     = "REPROVADO_PR"
	CorrecaoPR      = "CORRECAO_PR"
)

// LegacyReprovedPR é o alias legado de REPROVADO_PR.
const LegacyReprovedPR = "REPROVED_PR"

// Status é uma linha do catálogo.
type Status struct {
	ID      string `json:"id"`
	Code    string `json:"code"`
	Label   string `json:"label"`
	Ordinal int    `json:"ordinal"`
}

// Catalog é o catálogo estável. Os IDs nunca mudam nem são reaproveitados.
var Catalog = []Status{
	{"st_prd_doc", PRDDoc, "Documentação do PRD", 10},
	{"st_prd_doc_concluido", PRDDocConcluido, "PRD concluído", 20},
	{"st_planejamento", Planejamento, "Planejamento", 30},
	{"st_implementacao", Implementacao, "Implementação", 40},
	{"st_review", Review, "Review", 50},
	{"st_draft_pr", DraftPR, "Draft PR", 60},
	{"st_concluido", Concluido, "Concluído", 70},
	{"st_reprovado_pr", ReprovadoPR, "Reprovado na PR", 5},
	{"st_correcao_pr", CorrecaoPR, "Correção da PR", 6},
}

// Aliases mapeia valores legados para o código canônico.
var Aliases = map[string]string{LegacyReprovedPR: ReprovadoPR}

// Transition é uma aresta permitida da tabela de transições.
type Transition struct{ From, To string }

// Transitions é a tabela completa. Fluxo principal e fluxo de correção.
var Transitions = []Transition{
	{PRDDoc, PRDDocConcluido},
	{PRDDocConcluido, Planejamento},
	{Planejamento, Implementacao},
	{Implementacao, Review},
	{Review, DraftPR},
	{DraftPR, Concluido},
	{ReprovadoPR, CorrecaoPR},
	{CorrecaoPR, Planejamento},
}

// Resolved é o resultado de normalizar um valor lido.
type Resolved struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Raw   string `json:"raw"`   // valor bruto observado, preservado para auditoria
	Alias bool   `json:"alias"` // veio de um alias legado
}

var byCode = func() map[string]Status {
	m := map[string]Status{}
	for _, s := range Catalog {
		m[s.Code] = s
	}
	return m
}()

// Normalize resolve um código (ou alias) ao status canônico. ok=false se desconhecido.
func Normalize(raw string) (Resolved, bool) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	alias := false
	if c, ok := Aliases[code]; ok {
		code, alias = c, true
	}
	s, ok := byCode[code]
	if !ok {
		return Resolved{Raw: raw}, false
	}
	return Resolved{ID: s.ID, Code: s.Code, Raw: raw, Alias: alias}, true
}

// Known informa se o código canônico (ou alias) existe.
func Known(code string) bool { _, ok := Normalize(code); return ok }

// Allowed informa se a aresta from→to está na tabela (após normalizar).
func Allowed(from, to string) bool {
	f, ok1 := Normalize(from)
	t, ok2 := Normalize(to)
	if !ok1 || !ok2 {
		return false
	}
	for _, tr := range Transitions {
		if tr.From == f.Code && tr.To == t.Code {
			return true
		}
	}
	return false
}

// Codes lista os códigos canônicos em ordem estável.
func Codes() []string {
	out := make([]string, 0, len(Catalog))
	for _, s := range Catalog {
		out = append(out, s.Code)
	}
	sort.Strings(out)
	return out
}

// ProjectMap é o mapa de status entre a origem (Jira/ClickUp) e o catálogo, por projeto.
type ProjectMap struct {
	Source string            `json:"source"` // jira | clickup
	Remote map[string]string `json:"remote"` // status remoto → código canônico
	Moves  map[string]string `json:"moves"`  // código canônico → ID da transição/status remoto a aplicar
}

// MapProblem aponta o campo ou a transição que impede a projeção.
type MapProblem struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (p MapProblem) String() string { return p.Field + ": " + p.Message }

// Validate confere o mapa: códigos conhecidos, e cada destino projetável com id remoto.
// Todos os status que a fábrica projeta precisam ter entrada em Moves.
func (m ProjectMap) Validate(project []string) []MapProblem {
	var out []MapProblem
	if m.Source != "jira" && m.Source != "clickup" {
		out = append(out, MapProblem{"source", "use jira ou clickup"})
	}
	remotes := make([]string, 0, len(m.Remote))
	for r := range m.Remote {
		remotes = append(remotes, r)
	}
	sort.Strings(remotes)
	for _, r := range remotes {
		if strings.TrimSpace(r) == "" {
			out = append(out, MapProblem{"remote", "status remoto vazio"})
			continue
		}
		if _, ok := Normalize(m.Remote[r]); !ok {
			out = append(out, MapProblem{"remote[" + r + "]", "código desconhecido: " + m.Remote[r]})
		}
	}
	for _, c := range project {
		n, ok := Normalize(c)
		if !ok {
			out = append(out, MapProblem{"moves[" + c + "]", "código desconhecido"})
			continue
		}
		if strings.TrimSpace(m.Moves[n.Code]) == "" {
			out = append(out, MapProblem{"moves[" + n.Code + "]", "sem status ou transição remota para projetar"})
		}
	}
	return out
}

// Resolve traduz um status remoto observado para o canônico, preservando o bruto.
func (m ProjectMap) Resolve(remote string) (Resolved, bool) {
	code, ok := m.Remote[remote]
	if !ok {
		for k, v := range m.Remote {
			if strings.EqualFold(k, remote) {
				code, ok = v, true
				break
			}
		}
	}
	if !ok {
		return Resolved{Raw: remote}, false
	}
	r, ok := Normalize(code)
	r.Raw = remote
	return r, ok
}
