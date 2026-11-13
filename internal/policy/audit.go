package policy

import (
	"context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

var sensitiveKey = regexp.MustCompile(`(?i)(secret|token|password|passwd|authorization|api[_-]?key|cookie|credential|private[_-]?key)`)

const redacted = "[REDACTED]"

// RedactParams devolve uma cópia dos parâmetros segura para a trilha: chaves
// sensíveis e valores com formato de segredo viram [REDACTED]; textos longos são
// truncados.
func RedactParams(params map[string]any) map[string]any {
	return redactMap(params, security.NewRedactor())
}

// RedactParamsWith usa o redator informado, que deve conhecer os valores exatos
// dos segredos do cofre (os padrões sozinhos não pegam segredo arbitrário).
func RedactParamsWith(params map[string]any, rd *security.Redactor) map[string]any {
	if rd == nil {
		rd = security.NewRedactor()
	}
	return redactMap(params, rd)
}

func redactMap(m map[string]any, rd *security.Redactor) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if sensitiveKey.MatchString(k) {
			out[k] = redacted
			continue
		}
		out[k] = redactValue(v, rd)
	}
	return out
}

func redactValue(v any, rd *security.Redactor) any {
	switch x := v.(type) {
	case string:
		s := rd.Redact(x)
		if len(s) > 200 {
			s = s[:200] + "…"
		}
		return s
	case map[string]any:
		return redactMap(x, rd)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = redactValue(e, rd)
		}
		return out
	case []string:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = redactValue(e, rd)
		}
		return out
	}
	return v
}

// Store grava as decisões em `policy_decisions`. Redactor deve ser o mesmo do
// cofre (valores exatos dos segredos); nulo usa só os padrões de token.
type Store struct {
	DB       persistence.DBTX
	Redactor *security.Redactor
}

// Record implementa Recorder.
func (s Store) Record(ctx context.Context, a Action, d Decision) error {
	raw, _ := json.Marshal(RedactParamsWith(a.Params, s.Redactor))
	var project any
	if a.Project != "" {
		project = a.Project
	}
	_, err := s.DB.Exec(ctx, `
		INSERT INTO policy_decisions (project_id, tool, operation, resource, target, effect, risk, rule, reason, params_hash, params_redacted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb)`,
		project, clipTo(a.Tool, 40), clipTo(a.Operation, 200), clipTo(s.text(a.Resource), 300), clipTo(HostOf(a.Target), 200),
		string(d.Effect), string(d.Risk), d.Rule, clipTo(d.Reason, 500), d.ParamsHash, string(raw))
	return persistence.MapError(err)
}

func (s Store) text(v string) string {
	if s.Redactor == nil {
		return security.NewRedactor().Redact(v)
	}
	return s.Redactor.Redact(v)
}

func clipTo(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Entry é uma linha da trilha.
type Entry struct {
	ID        int64          `json:"id"`
	Tool      string         `json:"tool"`
	Operation string         `json:"operation"`
	Resource  string         `json:"resource"`
	Target    string         `json:"target"`
	Effect    string         `json:"effect"`
	Risk      string         `json:"risk"`
	Rule      string         `json:"rule"`
	Reason    string         `json:"reason"`
	Hash      string         `json:"params_hash"`
	Params    map[string]any `json:"params"`
	CreatedAt time.Time      `json:"created_at"`
}

// Recent devolve as decisões mais recentes (opcionalmente de um projeto).
func Recent(ctx context.Context, db persistence.DBTX, project string, limit int) ([]Entry, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	q := `SELECT id, tool, operation, resource, target, effect, risk, rule, reason, params_hash, params_redacted, created_at FROM policy_decisions`
	args := []any{limit}
	if project != "" {
		q += ` WHERE project_id = $2::uuid`
		args = append(args, project)
	}
	rows, err := db.Query(ctx, q+` ORDER BY id DESC LIMIT $1`, args...)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		var raw []byte
		if err := rows.Scan(&e.ID, &e.Tool, &e.Operation, &e.Resource, &e.Target, &e.Effect, &e.Risk, &e.Rule, &e.Reason, &e.Hash, &raw, &e.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		_ = json.Unmarshal(raw, &e.Params)
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}
