package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/selection"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

type selEnv struct {
	*relEnv
	store    *selection.Store
	a, b, cc string // fluxos de feature publicados
	sub      string // subfluxo publicado
	unpub    string // fluxo não publicado
}

func newSelEnv(t *testing.T) *selEnv {
	t.Helper()
	r := newRelEnv(t)
	pub := func(name, kind string) string {
		w := r.draft(t, name, kind, workflows.Starter())
		if rs := r.publish(w); rs.StatusCode != 201 {
			t.Fatalf("%s: %d %s (versão %d)", name, rs.StatusCode, rs.Body, w.Version)
		}
		return w.ID
	}
	s := &selEnv{relEnv: r, store: &selection.Store{DB: r.e.db.Pool}}
	s.a, s.b, s.cc, s.sub = pub("fluxo-a", "feature"), pub("fluxo-b", "feature"), pub("fluxo-c", "feature"), pub("sub-x", "subflow")
	s.unpub = r.draft(t, "fluxo-rascunho", "feature", workflows.Starter()).ID
	return s
}

func rule(name, wf string, enabled bool, conds ...selection.Condition) selection.Rule {
	return selection.Rule{Name: name, Conditions: conds, WorkflowID: wf, Enabled: enabled}
}

func cond(field, op string, vals ...string) selection.Condition {
	return selection.Condition{Field: field, Op: op, Values: vals}
}

// T07-04 — duas regras, nenhum match, escolha determinística e motivo persistido.
func TestRulesPickTheFirstMatchTheDefaultOrFailExplicitlyAndPersistTheReason(t *testing.T) {
	s := newSelEnv(t)
	ctx := context.Background()
	_, err := s.store.Set(ctx, s.proj, selection.Config{
		Rules: []selection.Rule{
			rule("bugs do Jira", s.a, true, cond("platform", "eq", "jira"), cond("type", "in", "bug", "defeito")),
			rule("qualquer bug", s.b, true, cond("type", "eq", "bug")),
			rule("docs", s.cc, true, cond("labels", "has_any", "docs", "documentação")),
		},
		Default: &selection.Target{WorkflowID: s.cc, SubflowID: s.sub},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Casam as regras 1 e 2: vence a primeira; a segunda aparece no rastro.
	sel, err := s.store.Select(ctx, s.proj, "PROJ-1", selection.Attrs{"platform": " Jira ", "type": "BUG"})
	if err != nil || sel.WorkflowID != s.a || sel.Position != 1 || sel.UsedDefault || len(sel.AlsoMatched) != 1 || sel.AlsoMatched[0] != 2 {
		t.Fatalf("%+v %v", sel, err)
	}
	// Só a regra 2 casa (outra plataforma).
	if sel, err = s.store.Select(ctx, s.proj, "PROJ-2", selection.Attrs{"platform": "clickup", "type": "bug"}); err != nil || sel.WorkflowID != s.b || sel.Position != 2 {
		t.Fatalf("%+v %v", sel, err)
	}
	// Listas normalizadas: caixa, espaços e repetição não importam.
	if sel, err = s.store.Select(ctx, s.proj, "PROJ-3", selection.Attrs{"type": "task", "labels": []string{"  DOCS ", "docs", "x"}}); err != nil || sel.WorkflowID != s.cc || sel.Position != 3 {
		t.Fatalf("%+v %v", sel, err)
	}
	// Nenhum match: o padrão (fluxo e subfluxo), com o motivo.
	sel, err = s.store.Select(ctx, s.proj, "PROJ-4", selection.Attrs{"type": "story", "platform": "jira"})
	if err != nil || !sel.UsedDefault || sel.WorkflowID != s.cc || sel.SubflowID != s.sub || !strings.Contains(sel.Reason, "padrão") {
		t.Fatalf("%+v %v", sel, err)
	}
	// Determinismo: a mesma entrada, 20 vezes, a mesma escolha.
	for i := 0; i < 20; i++ {
		x, err := s.store.Select(ctx, s.proj, "PROJ-1", selection.Attrs{"type": "bug", "platform": "jira"})
		if err != nil || x.WorkflowID != s.a || x.Position != 1 {
			t.Fatalf("%+v %v", x, err)
		}
	}
	// Os rastros guardam entrada normalizada, regra e motivo.
	tr, _ := s.store.Traces(ctx, s.proj, 100)
	if len(tr) != 24 {
		t.Fatalf("%d rastros", len(tr))
	}
	first := tr[len(tr)-1]
	if first.FeatureRef != "PROJ-1" || first.RulePosition == nil || *first.RulePosition != 1 || !strings.Contains(first.Reason, "primeira a casar") || !strings.Contains(string(first.Input), `"platform": "jira"`) || !strings.Contains(first.Reason, "também casariam") {
		t.Fatalf("%+v", first)
	}
	// Reordenar muda o vencedor (a ordem é o critério).
	cfg, _ := s.store.Get(ctx, s.proj)
	cfg.Rules[0], cfg.Rules[1] = cfg.Rules[1], cfg.Rules[0]
	if _, err := s.store.Set(ctx, s.proj, cfg); err != nil {
		t.Fatal(err)
	}
	if x, _ := s.store.Select(ctx, s.proj, "PROJ-1", selection.Attrs{"type": "bug", "platform": "jira"}); x.WorkflowID != s.b || x.Position != 1 {
		t.Fatalf("%+v", x)
	}
	// Sem padrão e sem match: falha explícita, com rastro.
	cfg.Default = nil
	if _, err := s.store.Set(ctx, s.proj, cfg); err != nil {
		t.Fatal(err)
	}
	x, err := s.store.Select(ctx, s.proj, "PROJ-9", selection.Attrs{"type": "story"})
	if !errors.Is(err, selection.ErrNoSelection) || x.TraceID == 0 {
		t.Fatalf("%+v %v", x, err)
	}
	tr, _ = s.store.Traces(ctx, s.proj, 1)
	if tr[0].FeatureRef != "PROJ-9" || tr[0].WorkflowID != nil || !strings.Contains(tr[0].Reason, "padrão") {
		t.Fatalf("%+v", tr[0])
	}
}

func TestSimulationExplainsEveryRuleWithoutRecordingAndDisabledRulesAreSkipped(t *testing.T) {
	s := newSelEnv(t)
	ctx := context.Background()
	if _, err := s.store.Set(ctx, s.proj, selection.Config{Rules: []selection.Rule{
		rule("desativada", s.a, false, cond("type", "eq", "bug")),
		rule("prefixo", s.b, true, cond("title", "prefix", "hotfix"), cond("priority", "eq", "alta")),
	}, Default: &selection.Target{WorkflowID: s.cc}}); err != nil {
		t.Fatal(err)
	}
	sim, err := s.store.Simulate(ctx, s.proj, selection.Attrs{"type": "bug", "title": "Hotfix: login", "priority": "Baixa"})
	if err != nil || !sim.UsedDefault || len(sim.Evals) != 2 || sim.Evals[0].Why != "regra desativada" || !strings.Contains(sim.Evals[1].Why, "priority") {
		t.Fatalf("%+v %v", sim, err)
	}
	sim, _ = s.store.Simulate(ctx, s.proj, selection.Attrs{"type": "bug", "title": "  HOTFIX: login ", "priority": "ALTA"})
	if sim.WorkflowID != s.b || !sim.Evals[1].Matched {
		t.Fatalf("%+v", sim)
	}
	if tr, _ := s.store.Traces(ctx, s.proj, 10); len(tr) != 0 {
		t.Fatal("simular não grava rastro")
	}
}

func TestSelectionConfigValidationAndPublishedTargets(t *testing.T) {
	s := newSelEnv(t)
	ctx := context.Background()
	bad := map[string]selection.Config{
		"sem condição":               {Rules: []selection.Rule{rule("x", s.a, true)}},
		"campo":                      {Rules: []selection.Rule{rule("x", s.a, true, cond("cor", "eq", "azul"))}},
		"operador":                   {Rules: []selection.Rule{rule("x", s.a, true, cond("type", "regex", ".*"))}},
		"lista com eq":               {Rules: []selection.Rule{rule("x", s.a, true, cond("labels", "eq", "a"))}},
		"texto com has_any":          {Rules: []selection.Rule{rule("x", s.a, true, cond("type", "has_any", "a"))}},
		"sem valores":                {Rules: []selection.Rule{rule("x", s.a, true, cond("type", "eq"))}},
		"sem nome":                   {Rules: []selection.Rule{rule(" ", s.a, true, cond("type", "eq", "a"))}},
		"fluxo é subfluxo":           {Rules: []selection.Rule{rule("x", s.sub, true, cond("type", "eq", "a"))}},
		"fluxo inexistente":          {Rules: []selection.Rule{rule("x", "00000000-0000-4000-8000-000000000000", true, cond("type", "eq", "a"))}},
		"padrão com subfluxo errado": {Default: &selection.Target{WorkflowID: s.a, SubflowID: s.b}},
	}
	for name, cfg := range bad {
		if _, err := s.store.Set(ctx, s.proj, cfg); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	// Uma configuração recusada não altera a anterior (atômica).
	if _, err := s.store.Set(ctx, s.proj, selection.Config{Rules: []selection.Rule{rule("ok", s.a, true, cond("type", "eq", "bug"))}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.store.Set(ctx, s.proj, bad["campo"]); err == nil {
		t.Fatal("deveria falhar")
	}
	if cfg, _ := s.store.Get(ctx, s.proj); len(cfg.Rules) != 1 || cfg.Rules[0].Name != "ok" {
		t.Fatalf("%+v", cfg)
	}
	// Destino não publicado: o rastro explica e a seleção recusa.
	if _, err := s.store.Set(ctx, s.proj, selection.Config{Rules: []selection.Rule{rule("rascunho", s.unpub, true, cond("type", "eq", "bug"))}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.store.Select(ctx, s.proj, "PROJ-7", selection.Attrs{"type": "bug"}); !errors.Is(err, releases.ErrNotPublished) {
		t.Fatalf("%v", err)
	}
	if tr, _ := s.store.Traces(ctx, s.proj, 1); !strings.Contains(tr[0].Reason, "não está publicado") {
		t.Fatalf("%+v", tr[0])
	}
}

func TestSelectionAPI(t *testing.T) {
	s := newSelEnv(t)
	c := s.relEnv.c
	body := map[string]any{
		"rules":   []map[string]any{{"name": "bugs", "workflow_id": s.a, "enabled": true, "conditions": []map[string]any{{"field": "type", "op": "eq", "values": []string{"Bug"}}}}},
		"default": map[string]any{"workflow_id": s.b},
	}
	var cfg selection.Config
	c.expect(c.do("PUT", "/api/projects/"+s.proj+"/selection", body, nil), 200).JSON(&cfg)
	if len(cfg.Rules) != 1 || cfg.Rules[0].Position != 1 || cfg.Rules[0].Conditions[0].Values[0] != "bug" || cfg.Default == nil {
		t.Fatalf("%+v", cfg)
	}
	var sel selection.Selection
	c.expect(c.post("/api/projects/"+s.proj+"/selection/simulate", map[string]any{"attributes": map[string]any{"type": "BUG"}}), 200).JSON(&sel)
	if sel.WorkflowID != s.a || sel.Position != 1 {
		t.Fatalf("%+v", sel)
	}
	var tr struct{ Items []any }
	c.expect(c.get("/api/projects/"+s.proj+"/selection/traces"), 200).JSON(&tr)
	if len(tr.Items) != 0 {
		t.Fatal("simulação não grava")
	}
	if r := c.do("PUT", "/api/projects/"+s.proj+"/selection", map[string]any{"rules": []map[string]any{{"name": "x", "workflow_id": s.a, "enabled": true, "conditions": []map[string]any{}}}}, nil); r.StatusCode != 400 {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
	if r := s.e.newClient().get("/api/projects/" + s.proj + "/selection"); r.StatusCode != 401 {
		t.Fatalf("%d", r.StatusCode)
	}
	// Sem padrão nem match na simulação: 409 no_selection.
	c.expect(c.do("PUT", "/api/projects/"+s.proj+"/selection", map[string]any{"rules": []any{}}, nil), 200)
	if r := c.post("/api/projects/"+s.proj+"/selection/simulate", map[string]any{"attributes": map[string]any{"type": "x"}}); r.StatusCode != 409 || r.ErrCode() != "no_selection" {
		t.Fatalf("%d %s", r.StatusCode, r.Body)
	}
}
