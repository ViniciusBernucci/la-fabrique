package selection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/releases"
	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// ErrNoSelection: nenhuma regra casou e o projeto não tem padrão.
var ErrNoSelection = errors.New("nenhuma regra casou e o projeto não tem fluxo padrão")

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Atributos aceitos e operadores.
var fields = map[string]bool{"platform": true, "type": true, "priority": true, "title": true, "labels": true, "components": true}
var listFields = map[string]bool{"labels": true, "components": true}
var ops = map[string]bool{"eq": true, "in": true, "prefix": true, "contains": true, "has_any": true, "has_all": true}

// Condition compara um atributo normalizado.
type Condition struct {
	Field  string   `json:"field"`
	Op     string   `json:"op"`
	Values []string `json:"values"`
}

// Rule é uma regra ordenada.
type Rule struct {
	ID         string      `json:"id,omitempty"`
	Position   int         `json:"position"`
	Name       string      `json:"name"`
	Conditions []Condition `json:"conditions"`
	WorkflowID string      `json:"workflow_id"`
	SubflowID  string      `json:"subflow_id,omitempty"`
	Enabled    bool        `json:"enabled"`
}

// Target é um destino (fluxo e subfluxo opcional).
type Target struct {
	WorkflowID string `json:"workflow_id"`
	SubflowID  string `json:"subflow_id,omitempty"`
}

// Attrs são os atributos brutos da feature.
type Attrs map[string]any

func norm(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }

// Normalize devolve atributos normalizados: textos em minúsculas e sem espaços
// sobrando; listas sem repetição e ordenadas. Campos desconhecidos são ignorados.
func Normalize(a Attrs) map[string]any {
	out := map[string]any{}
	for k, v := range a {
		k = norm(k)
		if !fields[k] {
			continue
		}
		if listFields[k] {
			set := map[string]bool{}
			var items []any
			switch x := v.(type) {
			case []string:
				for _, s := range x {
					items = append(items, s)
				}
			case []any:
				items = x
			case string:
				for _, s := range strings.Split(x, ",") {
					items = append(items, s)
				}
			}
			var l []string
			for _, it := range items {
				if s, ok := it.(string); ok {
					if n := norm(s); n != "" && !set[n] {
						set[n] = true
						l = append(l, n)
					}
				}
			}
			sort.Strings(l)
			out[k] = l
			continue
		}
		if s, ok := v.(string); ok {
			out[k] = norm(s)
		}
	}
	return out
}

func (c Condition) eval(attrs map[string]any) (bool, string) {
	vals := make([]string, len(c.Values))
	for i, v := range c.Values {
		vals[i] = norm(v)
	}
	if listFields[c.Field] {
		have, _ := attrs[c.Field].([]string)
		set := map[string]bool{}
		for _, h := range have {
			set[h] = true
		}
		switch c.Op {
		case "has_any":
			for _, v := range vals {
				if set[v] {
					return true, ""
				}
			}
			return false, fmt.Sprintf("%s não tem nenhum de %v", c.Field, vals)
		case "has_all":
			for _, v := range vals {
				if !set[v] {
					return false, fmt.Sprintf("%s não tem %q", c.Field, v)
				}
			}
			return true, ""
		}
		return false, "operador inválido para lista"
	}
	got, _ := attrs[c.Field].(string)
	switch c.Op {
	case "eq", "in":
		for _, v := range vals {
			if got == v {
				return true, ""
			}
		}
		return false, fmt.Sprintf("%s é %q (esperado %v)", c.Field, got, vals)
	case "prefix":
		for _, v := range vals {
			if strings.HasPrefix(got, v) {
				return true, ""
			}
		}
		return false, fmt.Sprintf("%s (%q) não começa com %v", c.Field, got, vals)
	case "contains":
		for _, v := range vals {
			if strings.Contains(got, v) {
				return true, ""
			}
		}
		return false, fmt.Sprintf("%s (%q) não contém %v", c.Field, got, vals)
	}
	return false, "operador inválido para texto"
}

func (r Rule) match(attrs map[string]any) (bool, string) {
	for _, c := range r.Conditions {
		if ok, why := c.eval(attrs); !ok {
			return false, why
		}
	}
	return true, ""
}

func validateConditions(cs []Condition) error {
	if len(cs) == 0 || len(cs) > 10 {
		return &FieldError{"conditions", "Cada regra precisa de 1 a 10 condições."}
	}
	for i := range cs {
		c := &cs[i]
		c.Field, c.Op = norm(c.Field), norm(c.Op)
		if !fields[c.Field] || !ops[c.Op] {
			return &FieldError{"conditions", "Campo ou operador inválido."}
		}
		if listFields[c.Field] != (c.Op == "has_any" || c.Op == "has_all") {
			return &FieldError{"conditions", "Operador incompatível com o tipo do campo (listas usam has_any/has_all)."}
		}
		if len(c.Values) == 0 || len(c.Values) > 20 {
			return &FieldError{"conditions", "Informe de 1 a 20 valores."}
		}
		for j, v := range c.Values {
			if c.Values[j] = norm(v); c.Values[j] == "" || len(c.Values[j]) > 100 {
				return &FieldError{"conditions", "Valor vazio ou longo demais."}
			}
		}
	}
	return nil
}

// DB é o que o Store precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store guarda regras, padrão e rastros.
type Store struct{ DB DB }

func checkTarget(ctx context.Context, q persistence.DBTX, t Target) error {
	var kind string
	if err := q.QueryRow(ctx, `SELECT kind FROM workflows WHERE id = $1`, t.WorkflowID).Scan(&kind); err != nil || kind != workflows.KindFeature {
		return &FieldError{"workflow_id", "O fluxo precisa existir e ser do tipo feature."}
	}
	if t.SubflowID != "" {
		if err := q.QueryRow(ctx, `SELECT kind FROM workflows WHERE id = $1`, t.SubflowID).Scan(&kind); err != nil || kind != workflows.KindSubflow {
			return &FieldError{"subflow_id", "O subfluxo precisa existir e ser do tipo subflow."}
		}
	}
	return nil
}

// Config é a configuração de seleção de um projeto.
type Config struct {
	Rules   []Rule  `json:"rules"`
	Default *Target `json:"default"`
}

// Set substitui, de forma atômica, as regras (na ordem recebida) e o padrão.
func (s *Store) Set(ctx context.Context, projectID string, cfg Config) (Config, error) {
	if len(cfg.Rules) > 100 {
		return Config{}, &FieldError{"rules", "No máximo 100 regras."}
	}
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM selection_rules WHERE project_id = $1`, projectID); err != nil {
			return persistence.MapError(err)
		}
		for i, r := range cfg.Rules {
			r.Name = strings.TrimSpace(r.Name)
			if r.Name == "" || len(r.Name) > 80 {
				return &FieldError{"name", "O nome da regra deve ter de 1 a 80 caracteres."}
			}
			if err := validateConditions(r.Conditions); err != nil {
				return err
			}
			if err := checkTarget(ctx, tx, Target{r.WorkflowID, r.SubflowID}); err != nil {
				return err
			}
			cond, _ := json.Marshal(r.Conditions)
			var sub any
			if r.SubflowID != "" {
				sub = r.SubflowID
			}
			if _, err := tx.Exec(ctx, `INSERT INTO selection_rules (project_id, position, name, conditions, workflow_id, subflow_id, enabled) VALUES ($1, $2, $3, $4::jsonb, $5, $6::uuid, $7)`,
				projectID, i+1, r.Name, string(cond), r.WorkflowID, sub, r.Enabled); err != nil {
				return persistence.MapError(err)
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM selection_defaults WHERE project_id = $1`, projectID); err != nil {
			return persistence.MapError(err)
		}
		if cfg.Default != nil {
			if err := checkTarget(ctx, tx, *cfg.Default); err != nil {
				return err
			}
			var sub any
			if cfg.Default.SubflowID != "" {
				sub = cfg.Default.SubflowID
			}
			if _, err := tx.Exec(ctx, `INSERT INTO selection_defaults (project_id, workflow_id, subflow_id) VALUES ($1, $2, $3::uuid)`, projectID, cfg.Default.WorkflowID, sub); err != nil {
				return persistence.MapError(err)
			}
		}
		return nil
	})
	if err != nil {
		return Config{}, err
	}
	return s.Get(ctx, projectID)
}

// Get lê as regras (em ordem) e o padrão.
func (s *Store) Get(ctx context.Context, projectID string) (Config, error) {
	cfg := Config{Rules: []Rule{}}
	rows, err := s.DB.Query(ctx, `SELECT id, position, name, conditions, workflow_id, COALESCE(subflow_id::text, ''), enabled FROM selection_rules WHERE project_id = $1 ORDER BY position`, projectID)
	if err != nil {
		return cfg, persistence.MapError(err)
	}
	for rows.Next() {
		var r Rule
		var cond []byte
		if err := rows.Scan(&r.ID, &r.Position, &r.Name, &cond, &r.WorkflowID, &r.SubflowID, &r.Enabled); err != nil {
			rows.Close()
			return cfg, persistence.MapError(err)
		}
		_ = json.Unmarshal(cond, &r.Conditions)
		cfg.Rules = append(cfg.Rules, r)
	}
	rows.Close()
	var t Target
	err = s.DB.QueryRow(ctx, `SELECT workflow_id, COALESCE(subflow_id::text, '') FROM selection_defaults WHERE project_id = $1`, projectID).Scan(&t.WorkflowID, &t.SubflowID)
	if err == nil {
		cfg.Default = &t
	} else if !errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return cfg, persistence.MapError(err)
	}
	return cfg, nil
}

// RuleEval explica o que aconteceu com uma regra na simulação.
type RuleEval struct {
	Position int    `json:"position"`
	Name     string `json:"name"`
	Matched  bool   `json:"matched"`
	Enabled  bool   `json:"enabled"`
	Why      string `json:"why,omitempty"` // por que não casou
}

// Selection é o resultado de uma seleção.
type Selection struct {
	WorkflowID  string         `json:"workflow_id"`
	SubflowID   string         `json:"subflow_id,omitempty"`
	RuleID      string         `json:"rule_id,omitempty"`
	Position    int            `json:"rule_position,omitempty"`
	UsedDefault bool           `json:"used_default"`
	AlsoMatched []int          `json:"also_matched"` // posições de outras regras que também casariam
	Reason      string         `json:"reason"`
	Input       map[string]any `json:"input"`
	Evals       []RuleEval     `json:"evals,omitempty"`
	TraceID     int64          `json:"trace_id,omitempty"`
}

func (s *Store) decide(cfg Config, attrs map[string]any) (Selection, error) {
	sel := Selection{Input: attrs, AlsoMatched: []int{}}
	for _, r := range cfg.Rules {
		ev := RuleEval{Position: r.Position, Name: r.Name, Enabled: r.Enabled}
		if r.Enabled {
			ev.Matched, ev.Why = r.match(attrs)
		} else {
			ev.Why = "regra desativada"
		}
		sel.Evals = append(sel.Evals, ev)
		if !ev.Matched {
			continue
		}
		if sel.WorkflowID == "" {
			sel.WorkflowID, sel.SubflowID, sel.RuleID, sel.Position = r.WorkflowID, r.SubflowID, r.ID, r.Position
			sel.Reason = fmt.Sprintf("a regra %d (%q) foi a primeira a casar", r.Position, r.Name)
		} else {
			sel.AlsoMatched = append(sel.AlsoMatched, r.Position)
		}
	}
	if sel.WorkflowID != "" {
		if len(sel.AlsoMatched) > 0 {
			sel.Reason += fmt.Sprintf("; também casariam as regras %v, que ficam atrás na ordem", sel.AlsoMatched)
		}
		return sel, nil
	}
	if cfg.Default == nil {
		return sel, ErrNoSelection
	}
	sel.WorkflowID, sel.SubflowID, sel.UsedDefault = cfg.Default.WorkflowID, cfg.Default.SubflowID, true
	sel.Reason = "nenhuma regra casou; vale o fluxo padrão do projeto"
	return sel, nil
}

// Simulate explica a seleção, regra por regra, SEM gravar rastro.
func (s *Store) Simulate(ctx context.Context, projectID string, attrs Attrs) (Selection, error) {
	cfg, err := s.Get(ctx, projectID)
	if err != nil {
		return Selection{}, err
	}
	return s.decide(cfg, Normalize(attrs))
}

// Select escolhe o fluxo e grava o rastro (com o motivo). O destino precisa estar
// publicado; senão o rastro registra o motivo e a seleção devolve releases.ErrNotPublished.
func (s *Store) Select(ctx context.Context, projectID, featureRef string, attrs Attrs) (Selection, error) {
	cfg, err := s.Get(ctx, projectID)
	if err != nil {
		return Selection{}, err
	}
	norm := Normalize(attrs)
	sel, derr := s.decide(cfg, norm)
	if derr != nil {
		sel.Reason = derr.Error()
	}
	var pubErr error
	if derr == nil {
		for _, id := range []string{sel.WorkflowID, sel.SubflowID} {
			if id == "" {
				continue
			}
			if _, err := releases.Published(ctx, s.DB, id); err != nil {
				pubErr = fmt.Errorf("%w (workflow %s)", releases.ErrNotPublished, id)
				sel.Reason += "; o destino escolhido não está publicado"
				break
			}
		}
	}
	input, _ := json.Marshal(norm)
	matched, _ := json.Marshal(sel.AlsoMatched)
	var rule, wf, sub any
	var pos any
	if sel.RuleID != "" {
		rule, pos = sel.RuleID, sel.Position
	}
	if sel.WorkflowID != "" {
		wf = sel.WorkflowID
	}
	if sel.SubflowID != "" {
		sub = sel.SubflowID
	}
	if err := s.DB.QueryRow(ctx, `INSERT INTO selection_traces (project_id, feature_ref, input, rule_id, rule_position, used_default, workflow_id, subflow_id, matched, reason)
		VALUES ($1, $2, $3::jsonb, $4::uuid, $5, $6, $7::uuid, $8::uuid, $9::jsonb, $10) RETURNING id`,
		projectID, featureRef, string(input), rule, pos, sel.UsedDefault, wf, sub, string(matched), sel.Reason).Scan(&sel.TraceID); err != nil {
		return sel, persistence.MapError(err)
	}
	if derr != nil {
		return sel, derr
	}
	return sel, pubErr
}

// Trace é uma seleção registrada.
type Trace struct {
	ID           int64           `json:"id"`
	FeatureRef   string          `json:"feature_ref"`
	Input        json.RawMessage `json:"input"`
	RuleID       *string         `json:"rule_id"`
	RulePosition *int            `json:"rule_position"`
	UsedDefault  bool            `json:"used_default"`
	WorkflowID   *string         `json:"workflow_id"`
	SubflowID    *string         `json:"subflow_id"`
	Reason       string          `json:"reason"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Traces devolve os rastros mais recentes do projeto.
func (s *Store) Traces(ctx context.Context, projectID string, limit int) ([]Trace, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := s.DB.Query(ctx, `SELECT id, feature_ref, input, rule_id::text, rule_position, used_default, workflow_id::text, subflow_id::text, reason, created_at
		FROM selection_traces WHERE project_id = $1 ORDER BY id DESC LIMIT $2`, projectID, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Trace{}
	for rows.Next() {
		var t Trace
		var in []byte
		if err := rows.Scan(&t.ID, &t.FeatureRef, &in, &t.RuleID, &t.RulePosition, &t.UsedDefault, &t.WorkflowID, &t.SubflowID, &t.Reason, &t.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		t.Input = in
		out = append(out, t)
	}
	return out, persistence.MapError(rows.Err())
}
