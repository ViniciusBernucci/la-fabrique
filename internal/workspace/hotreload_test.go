package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// O reload automático do desenvolvimento (air) não pode observar a área de workspaces: o clone
// escreve milhares de arquivos ali dentro e o watcher reiniciava a API no meio da requisição
// (o front via 502 "socket hang up"). Vale para a API e o worker.
func TestHotReloadIgnoresWorkspacesDir(t *testing.T) {
	for _, name := range []string{"air.api.toml", "air.worker.toml"} {
		raw, err := os.ReadFile(filepath.Join("..", "..", "infra", "images", name))
		if err != nil {
			t.Fatal(err)
		}
		var line string
		for _, l := range strings.Split(string(raw), "\n") {
			if strings.Contains(l, "exclude_dir") {
				line = l
			}
		}
		for _, dir := range []string{"workspaces", "node_modules", "backups"} {
			if !strings.Contains(line, `"`+dir+`"`) {
				t.Errorf("%s: exclude_dir deve conter %q: %s", name, dir, line)
			}
		}
	}
}
