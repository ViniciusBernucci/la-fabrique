package integration

import (
	"context"
	"errors"
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
)

// AJ02-01 — catálogo estável, alias convergente e mapa por projeto.
func TestStatusCatalogIsIdempotentAndMapsPerProject(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	s := &statusflow.Store{DB: db.Pool}
	for i := 0; i < 2; i++ { // reaplicar mantém IDs e linhas
		if err := s.EnsureCatalog(ctx); err != nil {
			t.Fatal(err)
		}
	}
	var n, tr, al int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM status_catalog`).Scan(&n)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM status_transitions`).Scan(&tr)
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM status_aliases`).Scan(&al)
	if n != len(statusflow.Catalog) || tr != len(statusflow.Transitions) || al != 1 {
		t.Fatalf("catálogo %d transições %d aliases %d", n, tr, al)
	}
	var viaAlias, canon string
	_ = db.Pool.QueryRow(ctx, `SELECT status_id FROM status_aliases WHERE alias='REPROVED_PR'`).Scan(&viaAlias)
	_ = db.Pool.QueryRow(ctx, `SELECT id FROM status_catalog WHERE code='REPROVADO_PR'`).Scan(&canon)
	if viaAlias == "" || viaAlias != canon {
		t.Fatalf("alias %q canônico %q", viaAlias, canon)
	}

	p := mustProject(t, db.Pool, "flow-map")
	moves := map[string]string{}
	for _, c := range statusflow.ProjectedCodes() {
		moves[c] = "id-" + c
	}
	bad := statusflow.ProjectMap{Source: "jira", Remote: map[string]string{"Feito": "NADA"}, Moves: moves}
	var me *statusflow.MapError
	if _, err := s.SaveMap(ctx, p.ID, 0, bad); !errors.As(err, &me) || me.Problems[0].Field != "remote[Feito]" {
		t.Fatalf("%v", err)
	}
	good := statusflow.ProjectMap{Source: "jira", Remote: map[string]string{"Reprovado": statusflow.LegacyReprovedPR}, Moves: moves}
	saved, err := s.SaveMap(ctx, p.ID, 0, good)
	if err != nil || saved.Version != 1 || saved.Remote["Reprovado"] != statusflow.ReprovadoPR {
		t.Fatalf("%+v %v", saved, err)
	}
	if _, err := s.SaveMap(ctx, p.ID, 0, good); !errors.Is(err, statusflow.ErrStaleMap) {
		t.Fatalf("%v", err)
	}
	if v, err := s.SaveMap(ctx, p.ID, 1, good); err != nil || v.Version != 2 {
		t.Fatalf("%+v %v", v, err)
	}
	got, ok, err := s.GetMap(ctx, p.ID)
	if err != nil || !ok || got.Version != 2 || got.Moves[statusflow.Concluido] != "id-CONCLUIDO" {
		t.Fatalf("%+v %v", got, err)
	}
}
