package e2e

import (
	"path"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/risk"
)

// Config define o que é UI no projeto.
type Config struct {
	UIPaths []string `json:"ui_paths"`
}

// DefaultConfig: a UI do monorepo.
func DefaultConfig() Config { return Config{UIPaths: []string{"apps/web/**"}} }

// TouchesUI diz se algum arquivo do diff está na UI.
func (c Config) TouchesUI(files []risk.File) bool {
	for _, f := range files {
		for _, p := range c.UIPaths {
			if risk.Match(p, f.Path) {
				return true
			}
		}
	}
	return false
}

// Decision diz se E2E é exigido e por quê.
type Decision struct {
	Required bool   `json:"required"`
	Reason   string `json:"reason"`
}

// Decide escolhe o E2E pelo nível de risco e pelo impacto na UI.
func Decide(level string, files []risk.File, cfg Config) Decision {
	switch level {
	case risk.Low:
		return Decision{false, "risco baixo: E2E não exigido"}
	case risk.Medium:
		if cfg.TouchesUI(files) {
			return Decision{true, "risco médio e a mudança toca a UI"}
		}
		return Decision{false, "risco médio sem mudança na UI: E2E não exigido"}
	}
	return Decision{true, "risco alto: E2E exigido"}
}

// TestGap é um arquivo de código alterado sem teste correspondente no diff.
type TestGap struct {
	File     string `json:"file"`
	Expected string `json:"expected"`
}

var codeExt = map[string]bool{".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true, ".py": true}

func isTest(p string) bool {
	b := path.Base(p)
	return strings.HasSuffix(b, "_test.go") || strings.Contains(b, ".test.") || strings.Contains(b, ".spec.") ||
		(strings.HasPrefix(b, "test_") && strings.HasSuffix(b, ".py")) || strings.HasPrefix(p, "tests/") || strings.Contains(p, "/tests/") || strings.Contains(p, "/__tests__/")
}

func skipGuidance(p string) bool {
	return strings.HasPrefix(p, "docs/") || strings.Contains(p, "/migrations/") || strings.HasPrefix(p, "db/") || strings.HasPrefix(p, "scripts/") || strings.HasPrefix(p, "cmd/") ||
		strings.HasSuffix(p, "/main.tsx") || strings.HasSuffix(p, "/doc.go") || strings.Contains(p, ".d.ts") || strings.HasSuffix(p, ".config.ts")
}

// TestGaps lista o código alterado que ficou sem teste criado ou atualizado no diff.
func TestGaps(files []risk.File) []TestGap {
	changed := map[string]bool{}
	dirHasGoTest := map[string]bool{}
	for _, f := range files {
		changed[f.Path] = true
		if strings.HasSuffix(f.Path, "_test.go") {
			dirHasGoTest[path.Dir(f.Path)] = true
		}
	}
	var out []TestGap
	for _, f := range files {
		p := f.Path
		ext := path.Ext(p)
		if !codeExt[ext] || isTest(p) || skipGuidance(p) {
			continue
		}
		dir, base := path.Dir(p), strings.TrimSuffix(path.Base(p), ext)
		switch ext {
		case ".go":
			if !dirHasGoTest[dir] {
				out = append(out, TestGap{p, path.Join(dir, base+"_test.go")})
			}
		case ".py":
			if !changed[path.Join(dir, "test_"+base+".py")] && !changed[path.Join("tests", "test_"+base+".py")] {
				out = append(out, TestGap{p, path.Join(dir, "test_"+base+".py")})
			}
		default:
			ok := false
			for _, t := range []string{".test", ".spec"} {
				for _, e := range []string{".ts", ".tsx", ".js", ".jsx"} {
					ok = ok || changed[path.Join(dir, base+t+e)]
				}
			}
			if !ok {
				out = append(out, TestGap{p, path.Join(dir, base+".test"+ext)})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].File < out[j].File })
	return out
}
