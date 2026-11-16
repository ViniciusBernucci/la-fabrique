package integration

import (
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/models"
)

// ADR-0067 — o catálogo inicial de Claude e Codex existe, é idempotente e deixa os modelos utilizáveis.
func TestDefaultModelsAreInstalledOnceAndUsable(t *testing.T) {
	db := newMigratedDB(t)
	added, err := models.InstallDefaults(ctx, db.Pool)
	if err != nil || len(added) != models.DefaultCount() || len(added) != 12 {
		t.Fatalf("instalação: %d %v", len(added), err)
	}
	all, err := models.List(ctx, db.Pool)
	if err != nil || len(all) != 12 {
		t.Fatalf("catálogo: %d %v", len(all), err)
	}
	counts := map[string]int{}
	for _, m := range all {
		counts[m.Provider]++
		if m.State != "active" || !m.SupportsTools || m.Gateway != "" {
			t.Errorf("modelo padrão mal cadastrado: %+v", m)
		}
		// um agente com ferramentas pode usar o modelo (é isso que a verificação exige)
		v, err := models.Check(ctx, db.Pool, models.Requirement{Provider: m.Provider, Model: m.Name, NeedsTools: true})
		if err != nil || !v.OK {
			t.Errorf("%s/%s deveria ser compatível: %+v %v", m.Provider, m.Name, v, err)
		}
	}
	if counts["claude"] != 4 || counts["codex"] != 8 {
		t.Fatalf("por provedor: %v", counts)
	}
	// um modelo que não está no catálogo continua sendo recusado, com o motivo
	if v, _ := models.Check(ctx, db.Pool, models.Requirement{Provider: "claude", Model: "modelo-inventado"}); v.OK || len(v.Problems) == 0 {
		t.Fatalf("modelo fora do catálogo deveria reprovar: %+v", v)
	}

	// segunda subida não recria nada, e quem desativou um modelo não o vê reativado
	var first models.Model
	for _, m := range all {
		if m.Name == "gpt-6-luna" {
			first = m
		}
	}
	if _, err := models.SetState(ctx, db.Pool, first.ID, "disabled", first.Version); err != nil {
		t.Fatal(err)
	}
	added, err = models.InstallDefaults(ctx, db.Pool)
	if err != nil || len(added) != 0 {
		t.Fatalf("idempotência: %v %v", added, err)
	}
	cur, _ := models.Get(ctx, db.Pool, first.ID)
	if cur.State != "disabled" {
		t.Fatalf("a escolha da pessoa deve ser mantida: %+v", cur)
	}
}

func TestDefaultModelsAreListedByTheAPI(t *testing.T) {
	e := newAuthEnv(t, nil)
	if _, err := models.InstallDefaults(ctx, e.db.Pool); err != nil {
		t.Fatal(err)
	}
	c := e.newClient()
	c.login(nil)
	var list struct{ Items []models.Model }
	c.expect(c.get("/api/models"), 200).JSON(&list)
	if len(list.Items) != 12 {
		t.Fatalf("modelos: %d", len(list.Items))
	}
}
