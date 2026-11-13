package risk

import (
	"fmt"
	"sort"
	"strings"
)

// Níveis.
const (
	Low    = "low"
	Medium = "medium"
	High   = "high"
)

var rank = map[string]int{Low: 0, Medium: 1, High: 2}

func levelOf(r int) string {
	switch {
	case r >= 2:
		return High
	case r == 1:
		return Medium
	}
	return Low
}

// Max devolve o maior nível.
func Max(a, b string) string {
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// ValidLevel informa se é um nível conhecido.
func ValidLevel(l string) bool { _, ok := rank[l]; return ok }

// Rule é um piso por caminho: tocar um arquivo que casa com o padrão exige ao menos o nível.
type Rule struct {
	Pattern string `json:"pattern"` // glob: * dentro de um segmento, ** atravessa diretórios
	Floor   string `json:"floor"`
	Reason  string `json:"reason"`
}

// Policy é a política de risco de um projeto.
type Policy struct {
	Rules []Rule `json:"rules"`
	// Limiares de tamanho do diff (linhas adicionadas + removidas): a partir de MediumLines o
	// piso é médio; a partir de HighLines, alto. Zero desliga o limiar.
	MediumLines int `json:"medium_lines"`
	HighLines   int `json:"high_lines"`
	// MediumFiles/HighFiles: o mesmo por número de arquivos alterados.
	MediumFiles int `json:"medium_files"`
	HighFiles   int `json:"high_files"`
}

// DefaultPolicy é o piso padrão: áreas que costumam exigir cuidado.
func DefaultPolicy() Policy {
	return Policy{
		Rules: []Rule{
			{"**/migrations/**", High, "migração de banco de dados"},
			{"**/*.sql", High, "SQL"},
			{"**/auth/**", High, "autenticação e sessões"},
			{"**/security/**", High, "segurança e criptografia"},
			{"**/*secret*", High, "segredos"},
			{"**/*crypto*", High, "criptografia"},
			{".github/workflows/**", High, "pipeline de CI/CD"},
			{"**/Dockerfile*", High, "imagem de container"},
			{"**/docker-compose*", High, "composição de containers"},
			{"**/compose*.y*ml", High, "composição de containers"},
			{"go.mod", Medium, "dependências"},
			{"go.sum", Medium, "dependências"},
			{"**/package.json", Medium, "dependências"},
			{"**/*lock*", Medium, "dependências"},
			{"**/requirements*.txt", Medium, "dependências"},
			{"**/config/**", Medium, "configuração"},
			{"**/.env*", High, "variáveis de ambiente"},
		},
		MediumLines: 200, HighLines: 800, MediumFiles: 10, HighFiles: 30,
	}
}

// Validate confere a política.
func (p Policy) Validate() error {
	if len(p.Rules) > 200 {
		return fmt.Errorf("regras demais (máximo 200)")
	}
	for i, r := range p.Rules {
		if strings.TrimSpace(r.Pattern) == "" || len(r.Pattern) > 200 || strings.HasPrefix(r.Pattern, "/") || strings.Contains(r.Pattern, "..") {
			return fmt.Errorf("regra %d: padrão inválido %q", i+1, r.Pattern)
		}
		if r.Floor != Medium && r.Floor != High {
			return fmt.Errorf("regra %d: o piso deve ser medium ou high", i+1)
		}
		if len(r.Reason) > 200 {
			return fmt.Errorf("regra %d: justificativa longa demais", i+1)
		}
	}
	for _, n := range []int{p.MediumLines, p.HighLines, p.MediumFiles, p.HighFiles} {
		if n < 0 || n > 1_000_000 {
			return fmt.Errorf("limiar fora da faixa")
		}
	}
	if p.MediumLines > 0 && p.HighLines > 0 && p.HighLines < p.MediumLines {
		return fmt.Errorf("o limiar alto de linhas não pode ser menor que o médio")
	}
	if p.MediumFiles > 0 && p.HighFiles > 0 && p.HighFiles < p.MediumFiles {
		return fmt.Errorf("o limiar alto de arquivos não pode ser menor que o médio")
	}
	return nil
}

// Match diz se o caminho casa com o glob (sensível a maiúsculas, com `**` entre diretórios).
func Match(pattern, path string) bool {
	return matchSeg(strings.Split(pattern, "/"), strings.Split(path, "/"))
}

func matchSeg(p, s []string) bool {
	if len(p) == 0 {
		return len(s) == 0
	}
	if p[0] == "**" {
		for i := 0; i <= len(s); i++ {
			if matchSeg(p[1:], s[i:]) {
				return true
			}
		}
		return false
	}
	if len(s) == 0 || !matchWord(p[0], s[0]) {
		return false
	}
	return matchSeg(p[1:], s[1:])
}

// matchWord casa um segmento com `*` (qualquer sequência, sem "/") e `?`.
func matchWord(p, s string) bool {
	if p == "" {
		return s == ""
	}
	switch p[0] {
	case '*':
		for i := 0; i <= len(s); i++ {
			if matchWord(p[1:], s[i:]) {
				return true
			}
		}
		return false
	case '?':
		return s != "" && matchWord(p[1:], s[1:])
	}
	return s != "" && s[0] == p[0] && matchWord(p[1:], s[1:])
}

// File é um arquivo do diff (ou previsto pela tarefa, com Added/Removed zerados).
type File struct {
	Path    string `json:"path"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
}

// Reason explica quem exigiu o nível.
type Reason struct {
	Source string `json:"source"` // ai | rule | size
	Level  string `json:"level"`
	Detail string `json:"detail"`
}

// Assessment é o resultado justificado.
type Assessment struct {
	Suggested string   `json:"suggested"`
	Final     string   `json:"final"`
	Raised    bool     `json:"raised"` // os pisos elevaram o nível acima da sugestão
	Reasons   []Reason `json:"reasons"`
}

// Assess combina a sugestão da IA com os pisos da política sobre os arquivos informados.
// Sugestão inválida é tratada como alta (falha fechada). Só eleva, nunca rebaixa.
func Assess(p Policy, suggested string, files []File) Assessment {
	a := Assessment{Suggested: suggested, Reasons: []Reason{}}
	if !ValidLevel(suggested) {
		a.Suggested = High
		a.Reasons = append(a.Reasons, Reason{"ai", High, fmt.Sprintf("sugestão ausente ou inválida (%q): tratada como alta", suggested)})
	} else {
		a.Reasons = append(a.Reasons, Reason{"ai", suggested, "estimativa da IA"})
	}
	final := rank[a.Suggested]
	fs := append([]File{}, files...)
	sort.Slice(fs, func(i, j int) bool { return fs[i].Path < fs[j].Path })
	seen := map[string]bool{}
	var lines int
	for _, f := range fs {
		lines += f.Added + f.Removed
		for _, r := range p.Rules {
			if Match(r.Pattern, f.Path) {
				key := r.Pattern + "\x00" + f.Path
				if seen[key] {
					continue
				}
				seen[key] = true
				a.Reasons = append(a.Reasons, Reason{"rule", r.Floor, fmt.Sprintf("%s (%s): %s", f.Path, r.Pattern, r.Reason)})
				if rank[r.Floor] > final {
					final = rank[r.Floor]
				}
			}
		}
	}
	size := func(n, med, high int, what string) {
		switch {
		case high > 0 && n >= high:
			a.Reasons = append(a.Reasons, Reason{"size", High, fmt.Sprintf("%d %s (limiar alto: %d)", n, what, high)})
			final = max(final, 2)
		case med > 0 && n >= med:
			a.Reasons = append(a.Reasons, Reason{"size", Medium, fmt.Sprintf("%d %s (limiar médio: %d)", n, what, med)})
			final = max(final, 1)
		}
	}
	size(lines, p.MediumLines, p.HighLines, "linhas alteradas")
	size(len(fs), p.MediumFiles, p.HighFiles, "arquivos alterados")
	a.Final = levelOf(final)
	a.Raised = rank[a.Final] > rank[a.Suggested] && ValidLevel(suggested)
	return a
}

// Checks é o conjunto de verificações previstas para um nível.
type Checks struct {
	Level         string   `json:"level"`
	Required      []string `json:"required"`       // checks a rodar (disponíveis no projeto)
	NotApplicable []string `json:"not_applicable"` // previstos para o nível, mas sem comando no projeto: exigem justificativa
	Review        string   `json:"review"`         // none | standard | independent_security
	E2E           string   `json:"e2e"`            // no | if_ui | required
}

var base = []string{"lint", "build", "test"}
var security = []string{"secret_scan", "dependency_audit", "static_analysis"}

// Plan escolhe as verificações do nível. `available` são os checks que o projeto sabe
// executar (nomes do perfil técnico). Os básicos ausentes também ficam em NotApplicable.
func Plan(level string, available []string) Checks {
	if !ValidLevel(level) {
		level = High
	}
	have := map[string]bool{}
	for _, a := range available {
		have[a] = true
	}
	want := append([]string{}, base...)
	c := Checks{Level: level, Required: []string{}, NotApplicable: []string{}, Review: "none", E2E: "no"}
	switch level {
	case Medium:
		c.Review, c.E2E = "standard", "if_ui"
	case High:
		want = append(want, security...)
		c.Review, c.E2E = "independent_security", "required"
	}
	for _, w := range want {
		if have[w] {
			c.Required = append(c.Required, w)
		} else {
			c.NotApplicable = append(c.NotApplicable, w)
		}
	}
	return c
}
