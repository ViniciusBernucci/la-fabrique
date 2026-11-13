package prdgate

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
)

// maxContent é o tamanho a partir do qual a descrição é tratada como possivelmente truncada pela origem.
const maxContent = 30000

var (
	headerRe = regexp.MustCompile(`^\s*(?:#{1,6}\s*)?(?:\*\*)?([^:*#]{2,60}?)(?:\*\*)?\s*:?\s*$`)
	bulletRe = regexp.MustCompile(`^\s*(?:[-*•]|\d+[.)])\s+(.+)$`)
	urlRe    = regexp.MustCompile(`https?://[^\s)>\]]+`)
)

func fold(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, _ := transform.String(t, s)
	return strings.ToLower(strings.TrimSpace(out))
}

type section struct {
	title string
	lines []string
}

func sections(desc string) (intro string, secs []section) {
	var cur *section
	var introLines []string
	for _, line := range strings.Split(strings.ReplaceAll(desc, "\r\n", "\n"), "\n") {
		trim := strings.TrimSpace(line)
		isHeader := trim != "" && !bulletRe.MatchString(line) && (strings.HasPrefix(trim, "#") || strings.HasSuffix(trim, ":") || (strings.HasPrefix(trim, "**") && strings.HasSuffix(trim, "**")))
		if isHeader {
			if m := headerRe.FindStringSubmatch(line); m != nil {
				secs = append(secs, section{title: strings.TrimSpace(m[1])})
				cur = &secs[len(secs)-1]
				continue
			}
		}
		if cur == nil {
			introLines = append(introLines, line)
		} else {
			cur.lines = append(cur.lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(introLines, "\n")), secs
}

// FromItem normaliza uma Issue lida da origem (Jira/ClickUp) no PRD que o gate avalia. A descrição é lida
// por seções (`## Critérios de aceite`, `Referências:`…); critérios são os itens de lista da seção de
// critérios; campos obrigatórios do projeto são as seções de mesmo nome; referências são os links e chaves
// citados. Conteúdo longo demais é tratado como possivelmente truncado (inacessível) em vez de aprovado.
func FromItem(it integrations.Item, requiredFields []string) PRD {
	intro, secs := sections(it.Description)
	p := PRD{Key: it.Key, Revision: it.Version, Title: it.Title, Fields: map[string]string{}, Criteria: []string{}, References: []string{},
		Accessible: len(it.Description) < maxContent && !strings.HasSuffix(strings.TrimSpace(it.Description), "…")}
	desc := intro
	for _, s := range secs {
		body := strings.TrimSpace(strings.Join(s.lines, "\n"))
		switch f := fold(s.title); {
		case strings.Contains(f, "criterio") && strings.Contains(f, "aceite"):
			for _, l := range s.lines {
				if m := bulletRe.FindStringSubmatch(l); m != nil {
					p.Criteria = append(p.Criteria, strings.TrimSpace(m[1]))
				}
			}
		case strings.HasPrefix(f, "referencia") || strings.HasPrefix(f, "links") || strings.HasPrefix(f, "documentos"):
			p.References = append(p.References, urlRe.FindAllString(body, -1)...)
		}
		p.Fields[s.title] = body
		if desc == "" && body != "" && !strings.Contains(fold(s.title), "criterio") {
			desc = body // sem introdução: a primeira seção de texto serve de descrição
		}
	}
	p.Description = desc
	// Campos obrigatórios por nome normalizado.
	for _, rf := range requiredFields {
		if _, ok := p.Fields[rf]; ok {
			continue
		}
		for _, s := range secs {
			if fold(s.title) == fold(rf) {
				p.Fields[rf] = strings.TrimSpace(strings.Join(s.lines, "\n"))
			}
		}
	}
	return p
}

// RunItem lê o PRD da Issue da origem e roda o gate.
func (g *Gate) RunItem(ctx context.Context, cfg Config, it integrations.Item) (Result, error) {
	return g.Run(ctx, cfg, FromItem(it, cfg.RequiredFields))
}
