package acceptance

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Sources são os textos de entrada.
type Sources struct {
	Epics     string // 02-EPICOS.md
	Backlog   string // 03-BACKLOG.md
	Checklist string // 07-CHECKLIST.md
	Pilot     string // 05-ACEITE-E-PILOTO.md
	Trace     string // 04-RASTREABILIDADE.md
}

// Ticket reúne o que se sabe de um ticket.
type Ticket struct {
	ID         string
	Epic       string
	Title      string
	State      string // x | ~ | space
	Acceptance string
	Evidence   []string // links de entrega (texto markdown)
	Note       string   // nota do checklist (linhas recuadas), em uma linha
}

// Case é um caso de verificação P01–P18.
type Case struct {
	ID, Name, Procedure, Evidence string
	Epics                         []string
}

// Epic é um épico do checklist.
type Epic struct{ ID, Name string }

var (
	ticketLine = regexp.MustCompile(`^- \[( |x|~)\] \*\*\[(T\d\d-\d\d)\]\([^)]*\)\*\* — (.+?) · dep\.:`)
	epicHead   = regexp.MustCompile(`^## (E\d\d) — (.+)$`)
	backlogT   = regexp.MustCompile(`^### (T\d\d-\d\d)$`)
	acceptLine = regexp.MustCompile(`^- \*\*Aceite do ticket\*\* · (.+)$`)
	deliveryRe = regexp.MustCompile(`Entrega: \[([^\]]+)\]\(([^)]+)\)`)
	caseRow    = regexp.MustCompile(`^\| (P\d\d) · ([^|]+?) \| (.+?) \| (.+?) \| ((?:E\d\d)(?:, E\d\d)*) \|$`)
	ticketRef  = regexp.MustCompile(`T\d\d-\d\d`)
)

// Parse lê as fontes.
func Parse(s Sources) (epics []Epic, tickets []Ticket, cases []Case, err error) {
	accept := map[string]string{}
	cur := ""
	for _, l := range strings.Split(s.Backlog, "\n") {
		if m := backlogT.FindStringSubmatch(l); m != nil {
			cur = m[1]
			continue
		}
		if m := acceptLine.FindStringSubmatch(l); m != nil && cur != "" {
			accept[cur] = strings.TrimSpace(m[1])
		}
	}
	var last *Ticket
	curEpic := ""
	for _, l := range strings.Split(s.Checklist, "\n") {
		if m := epicHead.FindStringSubmatch(l); m != nil {
			curEpic = m[1]
			epics = append(epics, Epic{ID: m[1], Name: strings.TrimSpace(m[2])})
			last = nil
			continue
		}
		if m := ticketLine.FindStringSubmatch(l); m != nil {
			tickets = append(tickets, Ticket{ID: m[2], Epic: curEpic, Title: strings.TrimSpace(m[3]), State: m[1], Acceptance: accept[m[2]]})
			last = &tickets[len(tickets)-1]
			continue
		}
		if last != nil && strings.HasPrefix(l, "  ") {
			last.Note = strings.TrimSpace(last.Note + " " + strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(l), "- ")))
			for _, d := range deliveryRe.FindAllStringSubmatch(l, -1) {
				last.Evidence = append(last.Evidence, fmt.Sprintf("[%s](%s)", d[1], strings.TrimPrefix(d[2], "../")))
			}
		}
	}
	for _, l := range strings.Split(s.Pilot, "\n") {
		if m := caseRow.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			cases = append(cases, Case{ID: m[1], Name: strings.TrimSpace(m[2]), Procedure: strings.TrimSpace(m[3]), Evidence: strings.TrimSpace(m[4]), Epics: strings.Split(m[5], ", ")})
		}
	}
	if len(epics) == 0 || len(tickets) == 0 || len(cases) == 0 {
		return nil, nil, nil, fmt.Errorf("fontes ilegíveis: %d épicos, %d tickets, %d casos", len(epics), len(tickets), len(cases))
	}
	return epics, tickets, cases, nil
}

func esc(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "|", "\\|"), "\n", " ") }

func stateLabel(s string) string {
	switch s {
	case "x":
		return "feito"
	case "~":
		return "parcial"
	}
	return "pendente"
}

type counts struct{ done, partial, pending int }

func (c counts) String() string {
	return fmt.Sprintf("%d feito(s), %d parcial(is), %d pendente(s)", c.done, c.partial, c.pending)
}

// Render gera a matriz em Markdown (determinística).
func Render(s Sources) (string, error) {
	epics, tickets, cases, err := Parse(s)
	if err != nil {
		return "", err
	}
	byEpic := map[string][]Ticket{}
	for _, t := range tickets {
		byEpic[t.Epic] = append(byEpic[t.Epic], t)
	}
	epicCounts := func(e string) counts {
		var c counts
		for _, t := range byEpic[e] {
			switch t.State {
			case "x":
				c.done++
			case "~":
				c.partial++
			default:
				c.pending++
			}
		}
		return c
	}
	var total counts
	for _, e := range epics {
		c := epicCounts(e.ID)
		total.done += c.done
		total.partial += c.partial
		total.pending += c.pending
	}
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# Matriz de evidências do aceite\n\n")
	w("> **Gerado** por `make matriz` a partir de `NEW MVP/05-ACEITE-E-PILOTO.md`, `03-BACKLOG.md` e `07-CHECKLIST.md`. Não edite à mão: um teste confere que este arquivo é igual ao gerado.\n\n")
	w("A coluna **Evidência no projeto real** fica *pendente* até alguém registrá-la ([ficha de preparação](01-ficha-de-preparacao.md)). Evidência de software (testes e entregas) **não** substitui a do piloto: o aceite exige o projeto real (D95).\n\n")
	w("**Situação dos tickets:** %s, de %d.\n\n", total, len(tickets))
	w("## A. Casos de verificação (P01–P18)\n\n")
	w("| Caso | Épicos | Evidência mínima exigida | Evidência de software hoje | Evidência no projeto real |\n|---|---|---|---|---|\n")
	for _, c := range cases {
		var parts []string
		for _, e := range c.Epics {
			cc := epicCounts(e)
			var open []string
			for _, t := range byEpic[e] {
				if t.State != "x" {
					open = append(open, t.ID)
				}
			}
			p := fmt.Sprintf("%s: %s", e, cc)
			if len(open) > 0 {
				p += " (abertos: " + strings.Join(open, ", ") + ")"
			}
			parts = append(parts, p)
		}
		w("| **%s** · %s | %s | %s | %s | pendente |\n", c.ID, esc(c.Name), strings.Join(c.Epics, ", "), esc(c.Evidence), esc(strings.Join(parts, "<br>")))
	}
	w("\n## B. Critérios por épico e ticket\n\n")
	for _, e := range epics {
		w("### %s — %s\n\n%s.\n\n", e.ID, e.Name, epicCounts(e.ID))
		w("| Ticket | Estado | Critério de aceite do ticket | Evidência registrada | Evidência no projeto real |\n|---|---|---|---|---|\n")
		for _, t := range byEpic[e.ID] {
			ev := "—"
			if len(t.Evidence) > 0 {
				seen := map[string]bool{}
				var u []string
				for _, x := range t.Evidence {
					if !seen[x] {
						seen[x] = true
						u = append(u, x)
					}
				}
				sort.Strings(u)
				ev = strings.Join(u, "<br>")
			}
			w("| **%s** · %s | %s | %s | %s | pendente |\n", t.ID, esc(t.Title), stateLabel(t.State), esc(t.Acceptance), ev)
		}
		w("\n")
	}
	return b.String(), nil
}

// Check confere a rastreabilidade e devolve os problemas (vazio = fecha).
func Check(s Sources) []string {
	epics, tickets, cases, err := Parse(s)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	known := map[string]bool{}
	epicSet := map[string]bool{}
	for _, e := range epics {
		epicSet[e.ID] = true
	}
	for _, t := range tickets {
		known[t.ID] = true
		if t.Acceptance == "" {
			out = append(out, fmt.Sprintf("o ticket %s não tem critério de aceite no backlog", t.ID))
		}
		if !epicSet[t.Epic] {
			out = append(out, fmt.Sprintf("o ticket %s está fora de um épico do checklist", t.ID))
		}
	}
	inBacklog := map[string]bool{}
	for _, l := range strings.Split(s.Backlog, "\n") {
		if m := backlogT.FindStringSubmatch(l); m != nil {
			inBacklog[m[1]] = true
		}
	}
	for id := range inBacklog {
		if !known[id] {
			out = append(out, fmt.Sprintf("o ticket %s está no backlog mas não no checklist", id))
		}
	}
	for id := range known {
		if !inBacklog[id] {
			out = append(out, fmt.Sprintf("o ticket %s está no checklist mas não no backlog", id))
		}
	}
	covered := map[string]bool{}
	for _, c := range cases {
		for _, e := range c.Epics {
			covered[e] = true
			if !epicSet[e] {
				out = append(out, fmt.Sprintf("o caso %s aponta para o épico %s, que não existe", c.ID, e))
			}
		}
	}
	for _, e := range epics {
		if !covered[e.ID] {
			out = append(out, fmt.Sprintf("o épico %s não é coberto por nenhum caso P01–P18", e.ID))
		}
	}
	for _, id := range ticketRef.FindAllString(s.Trace, -1) {
		if !known[id] {
			out = append(out, fmt.Sprintf("a matriz de decisões cita %s, que não existe", id))
		}
	}
	sort.Strings(out)
	return dedupe(out)
}

func dedupe(in []string) []string {
	var out []string
	for i, s := range in {
		if i == 0 || s != in[i-1] {
			out = append(out, s)
		}
	}
	return out
}

// FichaFields lê os campos da "Ficha de preparação" de 05-ACEITE-E-PILOTO.md.
func FichaFields(pilot string) []string {
	var out []string
	in := false
	for _, l := range strings.Split(pilot, "\n") {
		t := strings.TrimSpace(l)
		switch {
		case strings.HasPrefix(t, "## Ficha de preparação"):
			in = true
		case strings.HasPrefix(t, "## ") && in:
			return out
		case in && strings.HasPrefix(t, "| ") && !strings.HasPrefix(t, "| Campo") && !strings.HasPrefix(t, "|---"):
			if i := strings.Index(t[2:], " |"); i > 0 {
				out = append(out, strings.TrimSpace(t[2:2+i]))
			}
		}
	}
	return out
}

// CheckFicha confere que a ficha pré-preenchida cobre todos os campos do contrato
// e que cada pendência (F01…) tem uma linha na tabela de pendências.
func CheckFicha(pilot, ficha string) []string {
	var out []string
	fields := FichaFields(pilot)
	if len(fields) == 0 {
		return []string{"a ficha de preparação de 05-ACEITE-E-PILOTO.md não foi encontrada"}
	}
	for i, f := range fields {
		if !strings.Contains(ficha, "**"+f+"**") {
			out = append(out, fmt.Sprintf("a ficha não cobre o campo %q", f))
		}
		id := fmt.Sprintf("F%02d", i+1)
		if !strings.Contains(ficha, "| "+id+" |") {
			out = append(out, fmt.Sprintf("falta a pendência %s (%s)", id, f))
		}
	}
	return out
}
