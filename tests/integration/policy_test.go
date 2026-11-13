package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// T13-01 — "Redigir trilha": a trilha guarda a decisão, nunca valores de segredo.
func TestPolicyTrailRecordsDecisionsWithoutSecrets(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "politica")
	sentinel := "SENTINELA-POLITICA-3f7a-nao-pode-vazar"
	ghToken := "ghp" + "_abcdefghijklmnopqrstuvwxyz0123456789"

	// O redator do cofre conhece o valor exato do segredo (aqui, o sentinela).
	rd := security.NewRedactor()
	rd.Register(sentinel)
	g := &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{AllowedAPIHosts: []string{"api.github.com"}}), Rec: policy.Store{DB: db.Pool, Redactor: rd}}
	ran := 0
	run := func() error { ran++; return nil }
	actions := []policy.Action{
		{Project: p.ID, Tool: "script", Operation: "run", Resource: "no-1", Params: map[string]any{"command": "make test", "token": sentinel}},
		{Project: p.ID, Tool: "api", Operation: "POST", Target: "https://api.github.com/repos/o/r/issues?access_token=" + ghToken,
			Params: map[string]any{"body": "titulo", "Authorization": "Bearer " + sentinel, "nested": map[string]any{"api_key": sentinel, "nota": "usei " + ghToken}}},
		{Project: p.ID, Tool: "script", Operation: "run", Params: map[string]any{"command": "curl -H 'X-Token: " + sentinel + "' http://x"}},
		{Tool: "api", Operation: "GET", Target: "http://169.254.169.254/latest", Params: map[string]any{}},
	}
	wantEffects := []policy.Effect{policy.Allow, policy.Approve, policy.Deny, policy.Deny}
	for i, a := range actions {
		d, _ := g.Do(ctx, a, run)
		if d.Effect != wantEffects[i] {
			t.Fatalf("ação %d: %+v", i, d)
		}
	}
	if ran != 1 {
		t.Fatalf("só a ação de baixo risco pode ter rodado: %d", ran)
	}

	all, err := policy.Recent(ctx, db.Pool, "", 50)
	if err != nil || len(all) != 4 {
		t.Fatalf("trilha: %d %v", len(all), err)
	}
	mine, _ := policy.Recent(ctx, db.Pool, p.ID, 50)
	if len(mine) != 3 {
		t.Fatalf("filtro por projeto: %d", len(mine))
	}
	// Mais recente primeiro; cada linha com regra, motivo e hash dos parâmetros.
	if all[0].Effect != "deny" || all[0].Rule != "builtin.internal_target" || all[0].Target != "169.254.169.254" || all[3].Effect != "allow" {
		t.Fatalf("ordem/campos: %+v", all)
	}
	for _, e := range all {
		if len(e.Hash) != 64 || e.Rule == "" || e.Reason == "" {
			t.Fatalf("linha incompleta: %+v", e)
		}
	}

	// Busca pelo segredo sentinela em TODAS as colunas da tabela.
	var found int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM policy_decisions d WHERE d::text LIKE '%'||$1||'%' OR d::text LIKE '%'||$2||'%'`, sentinel, "abcdefghijklmnopqrstuvwxyz0123456789").Scan(&found); err != nil {
		t.Fatal(err)
	}
	if found != 0 {
		t.Fatal("a trilha da política contém valor de segredo")
	}
	var params string
	if err := db.Pool.QueryRow(ctx, `SELECT params_redacted::text FROM policy_decisions WHERE operation = 'POST'`).Scan(&params); err != nil || !strings.Contains(params, "[REDACTED]") || !strings.Contains(params, "titulo") {
		t.Fatalf("parâmetros redigidos: %s %v", params, err)
	}
}

func TestGuardFailsClosedWhenTheTrailIsUnavailable(t *testing.T) {
	db := newMigratedDB(t)
	g := &policy.Guard{Policy: policy.DefaultPolicy(policy.Options{}), Rec: policy.Store{DB: db.Pool}}
	if _, err := db.Pool.Exec(context.Background(), `DROP TABLE policy_decisions`); err != nil {
		t.Fatal(err)
	}
	ran := 0
	_, err := g.Do(context.Background(), policy.Action{Tool: "script", Operation: "run", Params: map[string]any{"command": "make test"}}, func() error { ran++; return nil })
	if err == nil || ran != 0 || errors.Is(err, policy.ErrForbidden) {
		t.Fatalf("sem trilha a ação não roda: %v %d", err, ran)
	}
}
