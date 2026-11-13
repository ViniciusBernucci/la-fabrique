package acceptance

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var limitsRe = regexp.MustCompile(`\*\*Parcial:\*\*\s*(.+?)(?:\s+Revisão independente|\s+Entrega:|$)`)

// limitsOf extrai da nota do checklist o que o ticket declara como parcial.
func limitsOf(t Ticket) string {
	if m := limitsRe.FindStringSubmatch(t.Note); m != nil {
		return strings.TrimSpace(m[1])
	}
	if t.State == " " {
		return "sem entrega registrada"
	}
	return "sem limite declarado na nota"
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// RenderReport monta o relatório de aceite integral (T17-04) a partir do checklist. Ele não
// decide nada: mostra o que está provado, o que falta e deixa a decisão do responsável em branco.
// `root` é a raiz do repositório (para conferir a existência dos demais artefatos do aceite).
func RenderReport(s Sources, root string) (string, error) {
	epics, tickets, cases, err := Parse(s)
	if err != nil {
		return "", err
	}
	var done, partial, pending int
	byEpic := map[string]*counts{}
	for _, e := range epics {
		byEpic[e.ID] = &counts{}
	}
	for _, t := range tickets {
		c := byEpic[t.Epic]
		switch t.State {
		case "x":
			done++
			c.done++
		case "~":
			partial++
			c.partial++
		default:
			pending++
			c.pending++
		}
	}
	has := func(rel string) bool { _, err := os.Stat(filepath.Join(root, rel)); return err == nil }
	var b strings.Builder
	b.WriteString("# Relatório de aceite integral do MVP\n\n")
	b.WriteString("> **Gerado** por `make matriz` a partir de `07-CHECKLIST.md`, `03-BACKLOG.md` e `05-ACEITE-E-PILOTO.md`. Não edite à mão: um teste confere que este arquivo é igual ao gerado. O relatório **não aceita nada**: mostra o que está provado e o que falta; a decisão é do responsável.\n\n")
	closed := partial == 0 && pending == 0
	verdict := "NÃO FECHADO"
	if closed {
		verdict = "pronto para a decisão do responsável (falta a evidência do projeto real, se ainda pendente)"
	}
	fmt.Fprintf(&b, "**Situação do aceite integral:** %s.\n\n", verdict)
	fmt.Fprintf(&b, "**Tickets:** %d feito(s), %d parcial(is), %d pendente(s), de %d. Todo ticket `[x]` ou `[~]` cita a verificação real feita e o que ficou NOT_RUN na própria nota.\n\n", done, partial, pending, len(tickets))
	b.WriteString("O aceite exige o **projeto real** (D95): evidência de software (testes, ensaios e entregas) **não** a substitui. Nenhum dos " + fmt.Sprint(len(cases)) + " casos P01–P18 tem evidência no projeto real registrada ([matriz](02-matriz-de-evidencias.md)).\n\n")

	b.WriteString("## Artefatos do aceite\n\n| Artefato | Situação |\n|---|---|\n")
	art := []struct{ rel, label string }{
		{"docs/aceite/01-ficha-de-preparacao.md", "[Ficha de preparação](01-ficha-de-preparacao.md) (F01–F12 a preencher pelo responsável)"},
		{"docs/aceite/02-matriz-de-evidencias.md", "[Matriz de evidências](02-matriz-de-evidencias.md) (gerada)"},
		{"docs/aceite/03-ensaios-de-falha-e-seguranca.md", "[Ensaios de falha, segurança e integração](03-ensaios-de-falha-e-seguranca.md) (gerado por `make ensaios`)"},
	}
	for _, a := range art {
		st := "ausente (rode o comando correspondente)"
		if has(a.rel) {
			st = "presente"
		}
		fmt.Fprintf(&b, "| %s | %s |\n", a.label, st)
	}

	b.WriteString("\n## Épicos\n\n| Épico | Feito | Parcial | Pendente | Situação |\n|---|---|---|---|---|\n")
	for _, e := range epics {
		c := byEpic[e.ID]
		sit := "software verificado"
		if c.partial+c.pending > 0 {
			sit = "com pendências"
		}
		fmt.Fprintf(&b, "| **%s** · %s | %d | %d | %d | %s |\n", e.ID, esc(e.Name), c.done, c.partial, c.pending, sit)
	}

	b.WriteString("\n## Pendências por ticket (o que impede o aceite integral)\n\n| Ticket | Estado | Aceite do ticket | Limites declarados |\n|---|---|---|---|\n")
	for _, t := range tickets {
		if t.State == "x" {
			continue
		}
		fmt.Fprintf(&b, "| **%s** · %s | %s | %s | %s |\n", t.ID, esc(t.Title), stateLabel(t.State), esc(truncateRunes(t.Acceptance, 160)), esc(truncateRunes(limitsOf(t), 380)))
	}

	b.WriteString("\n## Decisão do responsável\n\n")
	b.WriteString("| Campo | Valor |\n|---|---|\n| Responsável | **A PREENCHER** (F01) |\n| Decisão | **A PREENCHER** (aceitar / aceitar com ressalvas / recusar) |\n| Data | **A PREENCHER** |\n| Revisão do código aceita | **A PREENCHER** (commit e digests, F02) |\n| Ressalvas | **A PREENCHER** |\n\n")
	b.WriteString("Enquanto a decisão e a evidência do projeto real estiverem em branco, o MVP **não** está aceito. O agente não decide o aceite, não faz merge, push nem deploy.\n")
	return b.String(), nil
}
