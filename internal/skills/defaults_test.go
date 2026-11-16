package skills

import (
	"io/fs"
	"path"
	"sort"
	"strings"
	"testing"
)

// O conjunto embutido é o que vai para todos os projetos: cada pasta tem que ser um pacote válido,
// com o nome da pasta igual ao do SKILL.md, os recursos que ele cita e nenhum lixo de sistema.
func TestEmbeddedDefaultsAreValidPackages(t *testing.T) {
	want := []string{"escrever-prd-e-specs", "lf-descobrir-ambiente-visual", "lf-implementar-tarefa", "lf-revisar-backend", "lf-revisar-frontend", "lf-tech-lead"}
	got := DefaultNames()
	sort.Strings(got)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("skills padrão embutidas: %v", got)
	}
	sub, err := fs.Sub(defaultsFS, "defaults")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range got {
		files, err := readTree(sub, name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		pkg, err := Ingest(files, pol())
		if err != nil {
			t.Fatalf("%s não é um pacote válido: %v", name, err)
		}
		if pkg.Manifest.Name != name || pkg.Manifest.Executor != ExecAI || pkg.NeedsApproval || pkg.Manifest.Description == "" {
			t.Errorf("%s: %+v", name, pkg.Manifest)
		}
		hasRef := false
		for _, f := range files {
			if isOSJunk(f.Path) {
				t.Errorf("%s: lixo de sistema no pacote embutido: %s", name, f.Path)
			}
			if strings.HasPrefix(f.Path, "references/") && path.Ext(f.Path) == ".md" {
				hasRef = true
			}
		}
		if !hasRef {
			t.Errorf("%s: esperava arquivos em references/", name)
		}
	}
}
