package clilogin

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	ansi    = regexp.MustCompile(`\x1b\[[0-9;?]*[ -/]*[@-~]`)
	urlRe   = regexp.MustCompile(`https://[^\s"'<>\x1b]+`)
	codeRe  = regexp.MustCompile(`\b[A-Z0-9]{4}-[A-Z0-9]{4,6}\b`)
	anyHTTP = regexp.MustCompile(`https?://\S+`)
)

// Hosts em que o link de login pode apontar. A saída do cliente é tratada como dado: um link fora
// destas listas nunca chega à tela (a pessoa não deve ser levada a abrir um endereço qualquer).
var allowedHosts = map[Provider][]string{
	Claude: {"claude.com", "claude.ai", "platform.claude.com", "console.anthropic.com"},
	Codex:  {"auth.openai.com", "chatgpt.com"},
}

// Challenge é o que a pessoa precisa para autorizar.
type Challenge struct {
	URL       string
	Code      string // só o Codex mostra um código para digitar no site
	NeedsCode bool   // o Claude pede para colar de volta o código que o site mostra
}

func stripANSI(s string) string { return ansi.ReplaceAllString(s, "") }

func hostAllowed(p Provider, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, a := range allowedHosts[p] {
		if h == a {
			return true
		}
	}
	return false
}

// Extract procura o desafio na saída acumulada do cliente. ok só quando tudo o que a pessoa
// precisa já apareceu: o link e, no Codex, também o código.
func Extract(p Provider, output string) (Challenge, bool) {
	clean := stripANSI(output)
	var link string
	for _, m := range urlRe.FindAllString(clean, -1) {
		m = strings.TrimRight(m, ".,;)")
		if hostAllowed(p, m) {
			link = m
			break
		}
	}
	if link == "" {
		return Challenge{}, false
	}
	switch p {
	case Claude:
		return Challenge{URL: link, NeedsCode: true}, true
	case Codex:
		// O código vem depois do link; procurar só o que aparece depois evita pegar a versão do cliente.
		after := clean[strings.Index(clean, link)+len(link):]
		code := codeRe.FindString(after)
		if code == "" {
			return Challenge{}, false
		}
		return Challenge{URL: link, Code: code}, true
	}
	return Challenge{}, false
}

// LastLine devolve a última linha útil da saída, sem cores nem endereços (que podem conter state
// ou códigos de uso único), para explicar uma falha sem vazar nada.
func LastLine(output string) string {
	lines := strings.Split(stripANSI(output), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(strings.ReplaceAll(lines[i], "\r", ""))
		if l == "" {
			continue
		}
		l = anyHTTP.ReplaceAllString(l, "<endereço>")
		l = codeRe.ReplaceAllString(l, "<código>")
		if len(l) > 200 {
			l = l[:200]
		}
		return l
	}
	return ""
}
