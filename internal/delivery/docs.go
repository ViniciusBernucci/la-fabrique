package delivery

import (
	"path"
	"sort"
	"strings"
)

// DocRule liga um módulo (padrão de caminho com um segmento curinga) à documentação exigida.
type DocRule struct {
	// Module: prefixo do módulo com `*` no segmento que o nomeia, ex.: "internal/*/".
	Module string
	// Docs: caminhos aceitos (qualquer um serve); `{}` é trocado pelo nome do módulo.
	Docs []string
}

// DefaultDocRules: pacotes Go e áreas do painel.
func DefaultDocRules() []DocRule {
	return []DocRule{
		{Module: "internal/*/", Docs: []string{"internal/{}/doc.go", "docs/modulos/{}.md"}},
		{Module: "apps/web/src/*/", Docs: []string{"docs/modulos/web-{}.md", "docs/modulos/web.md"}},
	}
}

// Gap é um módulo alterado sem a documentação exigida.
type Gap struct {
	Module   string   `json:"module"`
	Expected []string `json:"expected"`
}

func isCode(p string) bool {
	switch path.Ext(p) {
	case ".go", ".ts", ".tsx", ".js", ".jsx", ".py":
		return true
	}
	return false
}

func isTestFile(p string) bool {
	b := path.Base(p)
	return strings.HasSuffix(b, "_test.go") || strings.Contains(b, ".test.") || strings.Contains(b, ".spec.") || strings.HasPrefix(b, "test_")
}

// moduleOf devolve o nome do módulo de `p` segundo a regra (vazio se não casa).
func moduleOf(rule DocRule, p string) string {
	pre := strings.TrimSuffix(rule.Module, "/")
	parts, segs := strings.Split(pre, "/"), strings.Split(p, "/")
	if len(segs) <= len(parts) {
		return ""
	}
	name := ""
	for i, s := range parts {
		if s == "*" {
			name = segs[i]
			continue
		}
		if segs[i] != s {
			return ""
		}
	}
	return name
}

// DocsGaps lista os módulos com código alterado (não só teste) sem a documentação do módulo no diff.
func DocsGaps(files []string, rules []DocRule) []Gap {
	changed := map[string]bool{}
	for _, f := range files {
		changed[f] = true
	}
	seen := map[string]bool{}
	var out []Gap
	for _, f := range files {
		if !isCode(f) || isTestFile(f) {
			continue
		}
		for _, r := range rules {
			name := moduleOf(r, f)
			if name == "" || seen[r.Module+"\x00"+name] {
				continue
			}
			seen[r.Module+"\x00"+name] = true
			var exp []string
			ok := false
			for _, d := range r.Docs {
				d = strings.ReplaceAll(d, "{}", name)
				exp = append(exp, d)
				ok = ok || changed[d]
			}
			if !ok {
				out = append(out, Gap{Module: strings.TrimSuffix(r.Module, "/*/") + "/" + name, Expected: exp})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Module < out[j].Module })
	return out
}

// FeatureDocPath é o caminho do histórico vivo da feature no repositório.
func FeatureDocPath(externalKey string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(externalKey) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return "docs/features/" + strings.Trim(b.String(), "-") + ".md"
}
