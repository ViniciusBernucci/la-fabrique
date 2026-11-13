// Comando matriz: gera docs/aceite/02-matriz-de-evidencias.md a partir dos
// documentos do backlog. Uso: go run ./cmd/matriz   (na raiz do repositório)
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ViniciusBernucci/le-fabrique/internal/acceptance"
)

func read(p string) string {
	b, err := os.ReadFile(p)
	if err != nil {
		fmt.Fprintln(os.Stderr, "matriz: "+err.Error())
		os.Exit(1)
	}
	return string(b)
}

func main() {
	d := filepath.Join("docs", "NEW MVP")
	src := acceptance.Sources{
		Epics: read(filepath.Join(d, "02-EPICOS.md")), Backlog: read(filepath.Join(d, "03-BACKLOG.md")),
		Checklist: read(filepath.Join(d, "07-CHECKLIST.md")), Pilot: read(filepath.Join(d, "05-ACEITE-E-PILOTO.md")),
		Trace: read(filepath.Join(d, "04-RASTREABILIDADE.md")),
	}
	if problems := acceptance.Check(src); len(problems) > 0 {
		for _, p := range problems {
			fmt.Fprintln(os.Stderr, "matriz: "+p)
		}
		os.Exit(1)
	}
	out, err := acceptance.Render(src)
	if err != nil {
		fmt.Fprintln(os.Stderr, "matriz: "+err.Error())
		os.Exit(1)
	}
	if err := os.MkdirAll(filepath.Join("docs", "aceite"), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "matriz: "+err.Error())
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join("docs", "aceite", "02-matriz-de-evidencias.md"), []byte(out), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "matriz: "+err.Error())
		os.Exit(1)
	}
	fmt.Println("docs/aceite/02-matriz-de-evidencias.md gerado")
}
