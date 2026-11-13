package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Estados e tipos.
const (
	StatePending     = "pending"
	StateOfficial    = "official"
	StateInvalidated = "invalidated"
)

var kinds = map[string]bool{"fact": true, "decision": true, "constraint": true, "lesson": true}
var confidences = map[string]bool{"low": true, "medium": true, "high": true}

// ErrNotMerged: só uma entrega integrada com SHA promove memória.
var ErrNotMerged = errors.New("a entrega não foi integrada: nada é promovido")

// FieldError é um campo inválido.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o pacote precisa do banco.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Evidence vincula o registro a uma prova (revisão, check, commit, decisão…).
type Evidence struct {
	Label string `json:"label"`
	Ref   string `json:"ref"`
}

// Record é um registro de memória.
type Record struct {
	ID               string     `json:"id"`
	ProjectID        string     `json:"project_id"`
	FeatureID        string     `json:"feature_id"`
	Branch           string     `json:"branch"`
	PRURL            string     `json:"pr_url,omitempty"`
	Key              string     `json:"key"`
	Kind             string     `json:"kind"`
	Content          string     `json:"content"`
	Source           string     `json:"source"`
	Evidence         []Evidence `json:"evidence"`
	Revision         string     `json:"revision"`
	Confidence       string     `json:"confidence"`
	State            string     `json:"state"`
	PromotedRevision string     `json:"promoted_revision,omitempty"`
	PromotedAt       *time.Time `json:"promoted_at,omitempty"`
	InvalidReason    string     `json:"invalid_reason,omitempty"`
	Version          int        `json:"version"`
}

// Input é um registro a gravar (sempre nasce pendente, preso à branch).
type Input struct {
	ProjectID, FeatureID, Branch, PRURL string
	Key, Kind, Content, Source          string
	Evidence                            []Evidence
	Revision                            string
	Confidence                          string
}

// Store é o repositório da memória.
type Store struct {
	DB       DB
	Redactor *security.Redactor
	// Validate, se informado, pode recusar um registro na promoção (devolve o motivo; vazio = vale).
	Validate func(Record) string
	Now      func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// NormalizeKey gera a chave semântica: minúsculas e espaços colapsados.
func NormalizeKey(k string) string {
	return strings.Join(strings.Fields(strings.ToLower(k)), " ")
}

func hash(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }

const cols = `id::text, project_id::text, feature_id::text, branch, pr_url, key, kind, content, source, evidence, revision, confidence, state, promoted_revision, promoted_at, invalid_reason, version`

func scan(row interface{ Scan(...any) error }) (Record, error) {
	var r Record
	var ev []byte
	if err := row.Scan(&r.ID, &r.ProjectID, &r.FeatureID, &r.Branch, &r.PRURL, &r.Key, &r.Kind, &r.Content, &r.Source, &ev, &r.Revision, &r.Confidence, &r.State, &r.PromotedRevision, &r.PromotedAt, &r.InvalidReason, &r.Version); err != nil {
		return r, persistence.MapError(err)
	}
	r.Evidence = []Evidence{}
	_ = json.Unmarshal(ev, &r.Evidence)
	return r, nil
}

func mergeEvidence(a, b []Evidence) []Evidence {
	seen := map[Evidence]bool{}
	out := []Evidence{}
	for _, e := range append(append([]Evidence{}, a...), b...) {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}

func (s *Store) clean(in Input) (Input, error) {
	in.Key, in.Content, in.Source = NormalizeKey(in.Key), strings.TrimSpace(in.Content), strings.TrimSpace(in.Source)
	in.Branch, in.Revision = strings.TrimSpace(in.Branch), strings.TrimSpace(in.Revision)
	if in.Confidence == "" {
		in.Confidence = "medium"
	}
	switch {
	case in.ProjectID == "" || in.FeatureID == "":
		return in, &FieldError{"project", "projeto e feature são obrigatórios"}
	case in.Key == "" || len(in.Key) > 120:
		return in, &FieldError{"key", "a chave deve ter de 1 a 120 caracteres"}
	case !kinds[in.Kind]:
		return in, &FieldError{"kind", "tipo desconhecido (fact, decision, constraint, lesson)"}
	case in.Content == "" || len(in.Content) > 2000:
		return in, &FieldError{"content", "o conteúdo deve ter de 1 a 2000 caracteres"}
	case in.Source == "" || len(in.Source) > 200:
		return in, &FieldError{"source", "informe a fonte"}
	case in.Branch == "" || len(in.Branch) > 200:
		return in, &FieldError{"branch", "informe a branch da feature"}
	case in.Revision == "" || len(in.Revision) > 100:
		return in, &FieldError{"revision", "informe a revisão a que a evidência se refere"}
	case !confidences[in.Confidence]:
		return in, &FieldError{"confidence", "confiança desconhecida (low, medium, high)"}
	case len(in.Evidence) == 0 || len(in.Evidence) > 20:
		return in, &FieldError{"evidence", "vincule de 1 a 20 evidências"}
	}
	for _, e := range in.Evidence {
		if strings.TrimSpace(e.Label) == "" || strings.TrimSpace(e.Ref) == "" || len(e.Label) > 100 || len(e.Ref) > 300 {
			return in, &FieldError{"evidence", "cada evidência precisa de rótulo e referência"}
		}
	}
	if s.Redactor != nil {
		in.Content = s.Redactor.Redact(in.Content)
	}
	return in, nil
}

// Record grava um registro PENDENTE da branch. A mesma chave na mesma branch não cria um segundo
// registro: as evidências se somam e, se o conteúdo mudou, a versão mais recente vale.
func (s *Store) Record(ctx context.Context, in Input) (rec Record, created bool, err error) {
	in, err = s.clean(in)
	if err != nil {
		return Record{}, false, err
	}
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		cur, e := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM memory_records WHERE project_id = $1 AND branch = $2 AND key = $3 AND state = 'pending' FOR UPDATE`, in.ProjectID, in.Branch, in.Key))
		if e == nil {
			ev, _ := json.Marshal(mergeEvidence(cur.Evidence, in.Evidence))
			rec, e = scan(tx.QueryRow(ctx, `UPDATE memory_records SET content = $2, content_hash = $3, evidence = $4::jsonb, revision = $5, confidence = $6, kind = $7, source = $8,
				pr_url = CASE WHEN $9 <> '' THEN $9 ELSE pr_url END, version = version + 1, updated_at = $10 WHERE id::text = $1 RETURNING `+cols,
				cur.ID, in.Content, hash(in.Content), string(ev), in.Revision, in.Confidence, in.Kind, in.Source, in.PRURL, s.now()))
			return e
		}
		if !errors.Is(e, persistence.ErrNotFound) {
			return e
		}
		ev, _ := json.Marshal(mergeEvidence(nil, in.Evidence))
		rec, e = scan(tx.QueryRow(ctx, `INSERT INTO memory_records (project_id, feature_id, branch, pr_url, key, kind, content, content_hash, source, evidence, revision, confidence)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12) RETURNING `+cols,
			in.ProjectID, in.FeatureID, in.Branch, in.PRURL, in.Key, in.Kind, in.Content, hash(in.Content), in.Source, string(ev), in.Revision, in.Confidence))
		created = e == nil
		return e
	})
	return rec, created, err
}

// Get lê um registro do projeto (de outro projeto é "não encontrado").
func (s *Store) Get(ctx context.Context, projectID, id string) (Record, error) {
	return scan(s.DB.QueryRow(ctx, `SELECT `+cols+` FROM memory_records WHERE project_id::text = $1 AND id::text = $2`, projectID, id))
}

// Relevant devolve a memória que a execução pode consumir: os registros OFICIAIS do projeto e os
// PENDENTES da própria branch (nunca de outra branch nem de outro projeto), filtrados por termos.
func (s *Store) Relevant(ctx context.Context, projectID, branch, query string, limit int) ([]Record, error) {
	if limit < 1 || limit > 50 {
		limit = 20
	}
	where := []string{`project_id::text = $1`, `(state = 'official' OR (state = 'pending' AND branch = $2))`}
	args := []any{projectID, branch}
	for _, t := range strings.Fields(strings.ToLower(query)) {
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(t)+"%")
		where = append(where, fmt.Sprintf(`(lower(key) LIKE $%d OR lower(content) LIKE $%d)`, len(args), len(args)))
	}
	args = append(args, limit)
	rows, err := s.DB.Query(ctx, `SELECT `+cols+` FROM memory_records WHERE `+strings.Join(where, " AND ")+
		fmt.Sprintf(` ORDER BY (state = 'official') DESC, updated_at DESC, key LIMIT $%d`, len(args)), args...)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}

// Invalidate invalida um registro pendente ou oficial, com o motivo.
func (s *Store) Invalidate(ctx context.Context, projectID, id, reason string) (Record, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 500 {
		return Record{}, &FieldError{"reason", "informe o motivo (até 500 caracteres)"}
	}
	r, err := scan(s.DB.QueryRow(ctx, `UPDATE memory_records SET state = 'invalidated', invalid_reason = $3, version = version + 1, updated_at = $4
		WHERE project_id::text = $1 AND id::text = $2 AND state <> 'invalidated' RETURNING `+cols, projectID, id, reason, s.now()))
	if errors.Is(err, persistence.ErrNotFound) {
		return s.Get(ctx, projectID, id) // inexistente → não encontrado; já invalidado → devolve como está
	}
	return r, err
}

// Promotion é o resultado de uma promoção.
type Promotion struct {
	DeliveryID string   `json:"delivery_id"`
	MergeSHA   string   `json:"merge_sha"`
	Promoted   []Record `json:"promoted"`
	Rejected   []Record `json:"rejected"`
	Duplicates []Record `json:"duplicates"`
}

// Promote promove a memória pendente da branch da entrega INTEGRADA. Uma PR fechada, cancelada,
// aberta ou "integrada" sem SHA não promove nada. Só vale o registro cuja evidência se refere a
// uma revisão que entrou no merge (a revisão entregue ou um commit da feature) e que o validador
// opcional aceita; os demais são invalidados com o motivo. Registro igual a um oficial vira
// duplicata (evidências somadas); diferente, substitui o oficial anterior, que fica invalidado.
// Repetir não muda nada.
func (s *Store) Promote(ctx context.Context, projectID, deliveryID string) (Promotion, error) {
	out := Promotion{DeliveryID: deliveryID, Promoted: []Record{}, Rejected: []Record{}, Duplicates: []Record{}}
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		var feature, branch, revision, state, sha string
		if err := tx.QueryRow(ctx, `SELECT feature_id::text, branch, revision, state, merge_sha FROM deliveries WHERE project_id::text = $1 AND id::text = $2 FOR SHARE`, projectID, deliveryID).
			Scan(&feature, &branch, &revision, &state, &sha); err != nil {
			return persistence.MapError(err)
		}
		if state != "merged" || sha == "" {
			return ErrNotMerged
		}
		out.MergeSHA = sha
		valid := map[string]bool{revision: true, sha: true}
		rows, err := tx.Query(ctx, `SELECT sha FROM feature_commits WHERE project_id::text = $1 AND feature_id::text = $2`, projectID, feature)
		if err != nil {
			return persistence.MapError(err)
		}
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				rows.Close()
				return persistence.MapError(err)
			}
			valid[c] = true
		}
		rows.Close()
		prs, err := tx.Query(ctx, `SELECT `+cols+` FROM memory_records WHERE project_id::text = $1 AND feature_id::text = $2 AND branch = $3 AND state = 'pending' ORDER BY created_at, id FOR UPDATE`, projectID, feature, branch)
		if err != nil {
			return persistence.MapError(err)
		}
		var pending []Record
		for prs.Next() {
			r, err := scan(prs)
			if err != nil {
				prs.Close()
				return err
			}
			pending = append(pending, r)
		}
		prs.Close()
		for _, r := range pending {
			reason := ""
			if !valid[r.Revision] {
				reason = fmt.Sprintf("a evidência se refere à revisão %s, que não integrou o merge %s", short(r.Revision), short(sha))
			} else if s.Validate != nil {
				reason = strings.TrimSpace(s.Validate(r))
			}
			if reason != "" {
				u, err := scan(tx.QueryRow(ctx, `UPDATE memory_records SET state = 'invalidated', invalid_reason = $2, version = version + 1, updated_at = $3 WHERE id::text = $1 RETURNING `+cols, r.ID, truncate(reason, 500), s.now()))
				if err != nil {
					return err
				}
				out.Rejected = append(out.Rejected, u)
				continue
			}
			off, err := scan(tx.QueryRow(ctx, `SELECT `+cols+` FROM memory_records WHERE project_id::text = $1 AND key = $2 AND state = 'official' FOR UPDATE`, projectID, r.Key))
			if err != nil && !errors.Is(err, persistence.ErrNotFound) {
				return err
			}
			if err == nil {
				if hash(off.Content) == hash(r.Content) {
					ev, _ := json.Marshal(mergeEvidence(off.Evidence, r.Evidence))
					if _, err := tx.Exec(ctx, `UPDATE memory_records SET evidence = $2::jsonb, version = version + 1, updated_at = $3 WHERE id::text = $1`, off.ID, string(ev), s.now()); err != nil {
						return persistence.MapError(err)
					}
					u, err := scan(tx.QueryRow(ctx, `UPDATE memory_records SET state = 'invalidated', invalid_reason = $2, version = version + 1, updated_at = $3 WHERE id::text = $1 RETURNING `+cols,
						r.ID, "duplicada do registro oficial "+off.ID, s.now()))
					if err != nil {
						return err
					}
					out.Duplicates = append(out.Duplicates, u)
					continue
				}
				if _, err := tx.Exec(ctx, `UPDATE memory_records SET state = 'invalidated', invalid_reason = $2, version = version + 1, updated_at = $3 WHERE id::text = $1`,
					off.ID, "substituída pelo merge "+short(sha), s.now()); err != nil {
					return persistence.MapError(err)
				}
			}
			u, err := scan(tx.QueryRow(ctx, `UPDATE memory_records SET state = 'official', promoted_revision = $2, promoted_at = $3, version = version + 1, updated_at = $3 WHERE id::text = $1 RETURNING `+cols, r.ID, sha, s.now()))
			if err != nil {
				return err
			}
			out.Promoted = append(out.Promoted, u)
		}
		return nil
	})
	return out, err
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ExtractFromDelivery extrai lições da entrega: cada achado de revisão crítico ou relevante já
// corrigido vira um registro pendente da branch, com a revisão e o achado como evidência. A chave
// vem do título do achado, então repetir (ou achados iguais) não duplica.
func (s *Store) ExtractFromDelivery(ctx context.Context, projectID, deliveryID string) ([]Record, error) {
	var feature, branch string
	if err := s.DB.QueryRow(ctx, `SELECT feature_id::text, branch FROM deliveries WHERE project_id::text = $1 AND id::text = $2`, projectID, deliveryID).Scan(&feature, &branch); err != nil {
		return nil, persistence.MapError(err)
	}
	rows, err := s.DB.Query(ctx, `SELECT f.id::text, f.title, f.body, f.file, f.severity, r.id::text, r.task_key, f.fix_revision
		FROM review_findings f JOIN reviews r ON r.id = f.review_id
		WHERE r.project_id::text = $1 AND r.feature_id::text = $2 AND f.state = 'fixed' AND f.severity IN ('critical', 'relevant') AND f.fix_revision <> '' ORDER BY f.created_at, f.id`, projectID, feature)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	type cand struct{ fid, title, body, file, sev, rid, task, rev string }
	var cs []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.fid, &c.title, &c.body, &c.file, &c.sev, &c.rid, &c.task, &c.rev); err != nil {
			rows.Close()
			return nil, persistence.MapError(err)
		}
		cs = append(cs, c)
	}
	rows.Close()
	out := []Record{}
	for _, c := range cs {
		content := c.title
		if strings.TrimSpace(c.body) != "" {
			content += ": " + strings.TrimSpace(c.body)
		}
		if c.file != "" {
			content += " (" + c.file + ")"
		}
		conf := "medium"
		if c.sev == "critical" {
			conf = "high"
		}
		r, _, err := s.Record(ctx, Input{ProjectID: projectID, FeatureID: feature, Branch: branch, Key: "lição: " + c.title, Kind: "lesson", Content: truncate(content, 2000),
			Source: "revisão independente da tarefa " + c.task, Revision: c.rev, Confidence: conf,
			Evidence: []Evidence{{"revisão", c.rid}, {"achado", c.fid}}})
		if err != nil {
			return out, err
		}
		out = append(out, r)
	}
	return out, nil
}

// List lista os registros do projeto para o painel (de qualquer branch e estado, do mais recente
// ao mais antigo), com filtros opcionais de estado e de termos. Só do projeto informado.
func (s *Store) List(ctx context.Context, projectID, state, query string, limit int) ([]Record, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	if state != "" && state != StatePending && state != StateOfficial && state != StateInvalidated {
		return nil, &FieldError{"state", "estado desconhecido (pending, official, invalidated)"}
	}
	where := []string{`project_id::text = $1`, `($2 = '' OR state = $2)`}
	args := []any{projectID, state}
	for _, t := range strings.Fields(strings.ToLower(query)) {
		args = append(args, "%"+strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(t)+"%")
		where = append(where, fmt.Sprintf(`(lower(key) LIKE $%d OR lower(content) LIKE $%d)`, len(args), len(args)))
	}
	args = append(args, limit)
	rows, err := s.DB.Query(ctx, `SELECT `+cols+` FROM memory_records WHERE `+strings.Join(where, " AND ")+
		fmt.Sprintf(` ORDER BY updated_at DESC, key LIMIT $%d`, len(args)), args...)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Record{}
	for rows.Next() {
		r, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}
