package workflows

import (
	"encoding/json"
	"testing"
)

// O tipo "end" não tem saídas e o "start" não tem entradas: no JSON isso tem que ser [] e nunca null, senão
// o editor do painel quebrava ao abrir qualquer fluxo com nó de fim (página em branco).
func TestCatalogNeverSerializesNullLists(t *testing.T) {
	raw, err := json.Marshal(Catalog())
	if err != nil {
		t.Fatal(err)
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		t.Fatal(err)
	}
	if len(items) == 0 {
		t.Fatal("catálogo vazio")
	}
	for _, it := range items {
		for _, k := range []string{"inputs", "outputs", "fields"} {
			if it[k] == nil {
				t.Errorf("tipo %v: %q saiu como null", it["type"], k)
			}
		}
	}
}
