// Comando ensaios: roda os cenários de falha, segurança e integração (T17-03) e gera
// docs/aceite/03-ensaios-de-falha-e-seguranca.md com o esperado e o observado.
// Uso: TEST_DATABASE_URL=... go run ./cmd/ensaios   (na raiz do repositório; `make ensaios`)
// Sai com código 1 se algum cenário falhar. Sem TEST_DATABASE_URL, os cenários que exigem o
// PostgreSQL ficam NOT_RUN (nunca PASS).
package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/acceptance"
)

func revision() string {
	out, err := exec.Command("git", "rev-parse", "--short=12", "HEAD").Output()
	if err != nil {
		return "desconhecida"
	}
	rev := strings.TrimSpace(string(out))
	if d, err := exec.Command("git", "status", "--porcelain").Output(); err == nil && len(bytes.TrimSpace(d)) > 0 {
		rev += "+alterações locais"
	}
	return rev
}

func main() {
	if problems := acceptance.CheckCatalog("."); len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "ensaios: "+p)
		}
		os.Exit(1)
	}
	hasDB := os.Getenv("TEST_DATABASE_URL") != ""
	var results []acceptance.Result
	failed := false
	for _, sc := range acceptance.Catalog() {
		r := acceptance.Result{Scenario: sc}
		needsDB := false
		byPkg := map[string][]string{}
		var order []string
		for _, t := range sc.Tests {
			if _, ok := byPkg[t.Pkg]; !ok {
				order = append(order, t.Pkg)
			}
			byPkg[t.Pkg] = append(byPkg[t.Pkg], t.Name)
			needsDB = needsDB || strings.Contains(t.Pkg, "tests/integration")
		}
		if needsDB && !hasDB {
			r.Status, r.Detail = "NOT_RUN", "sem TEST_DATABASE_URL (PostgreSQL real indisponível)"
			results = append(results, r)
			continue
		}
		start := time.Now()
		r.Status = "PASS"
		for _, pkg := range order {
			cmd := exec.Command("go", "test", "-count=1", "-run", "^("+strings.Join(byPkg[pkg], "|")+")$", pkg)
			var out bytes.Buffer
			cmd.Stdout, cmd.Stderr = &out, &out
			if err := cmd.Run(); err != nil || strings.Contains(out.String(), "no tests to run") {
				r.Status = "FAIL"
				failed = true
				lines := strings.Split(strings.TrimSpace(out.String()), "\n")
				if len(lines) > 4 {
					lines = lines[len(lines)-4:]
				}
				r.Detail = strings.Join(lines, " ")
				if len(r.Detail) > 300 {
					r.Detail = r.Detail[:300]
				}
			}
		}
		r.Duration = time.Since(start)
		fmt.Printf("%s %-4s %s\n", r.Status, sc.ID, sc.Name)
		results = append(results, r)
	}
	if err := os.MkdirAll(filepath.Join("docs", "aceite"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "ensaios: "+err.Error())
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join("docs", "aceite", "03-ensaios-de-falha-e-seguranca.md"), []byte(acceptance.RenderRehearsal(results, time.Now(), revision())), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "ensaios: "+err.Error())
		os.Exit(1)
	}
	if failed {
		os.Exit(1)
	}
}
