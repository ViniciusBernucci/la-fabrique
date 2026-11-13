package checks

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Classificações.
const (
	ClassPass         = "pass"
	ClassRegression   = "regression"
	ClassPreexisting  = "preexisting"
	ClassUnknown      = "unknown"      // origem desconhecida: em investigação
	ClassInconclusive = "inconclusive" // o check não pôde ser concluído
)

// Erros.
var (
	ErrNotResolvable = errors.New("só falhas em investigação podem ser resolvidas")
	ErrNoProof       = errors.New("a falha só vira anterior com a prova de uma execução de baseline com a mesma assinatura")
)

// Item é o resultado classificado de um check.
type Item struct {
	Check          string `json:"check"`
	Class          string `json:"class"`
	Reason         string `json:"reason"`
	CandidateRunID string `json:"candidate_run_id"`
	BaselineRunID  string `json:"baseline_run_id,omitempty"`
}

// Evaluation é o veredito da revisão candidata.
type Evaluation struct {
	Revision string   `json:"revision"`
	Baseline string   `json:"baseline"`
	Items    []Item   `json:"items"`
	Blocked  bool     `json:"blocked"`
	Reasons  []string `json:"reasons"`
}

func blocking(class string) bool {
	return class == ClassRegression || class == ClassUnknown || class == ClassInconclusive
}

// latest devolve a execução mais recente de (feature, revisão, papel, check).
func latest(ctx context.Context, q persistence.DBTX, featureID, revision, role, check string) (*Run, error) {
	r, err := scanRun(q.QueryRow(ctx, `SELECT `+runCols+` FROM check_runs WHERE feature_id = $1 AND revision = $2 AND role = $3 AND check_name = $4 ORDER BY seq DESC LIMIT 1`,
		featureID, revision, role, check))
	if errors.Is(err, persistence.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// candidates lista, por check, a execução mais recente da revisão candidata.
func candidates(ctx context.Context, q persistence.DBTX, featureID, revision string) ([]Run, error) {
	rows, err := q.Query(ctx, `SELECT DISTINCT ON (check_name) `+runCols+` FROM check_runs
		WHERE feature_id = $1 AND revision = $2 AND role = 'candidate' ORDER BY check_name, seq DESC`, featureID, revision)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}

// Evaluate classifica cada check da revisão candidata contra o baseline e grava o
// resultado. Falha sem baseline que a comprove fica em investigação (bloqueia) e
// não é registrada como falha anterior. Execuções já resolvidas por uma pessoa
// mantêm a resolução.
func (s *Store) Evaluate(ctx context.Context, projectID, featureID, candidateRev, baselineRev string) (Evaluation, error) {
	ev := Evaluation{Revision: candidateRev, Baseline: baselineRev, Items: []Item{}, Reasons: []string{}}
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		runs, err := candidates(ctx, tx, featureID, candidateRev)
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			ev.Blocked = true
			ev.Reasons = append(ev.Reasons, "não há execução de checks para esta revisão: rode os checks de novo")
			return nil
		}
		for _, c := range runs {
			it := Item{Check: c.CheckName, CandidateRunID: c.ID}
			if c.Resolution != "" { // decisão humana prevalece
				it.Class, it.Reason, it.BaselineRunID = c.Classification, "resolvido manualmente: "+c.Resolution, c.BaselineRunID
				ev.add(it)
				continue
			}
			switch c.Status {
			case StatusPass:
				it.Class, it.Reason = ClassPass, "passou"
			case StatusError, StatusTimeout:
				it.Class, it.Reason = ClassInconclusive, "o check não terminou ("+c.Status+"): "+c.Reason
			case StatusFail:
				if err := s.classifyFailure(ctx, tx, projectID, featureID, baselineRev, c, &it); err != nil {
					return err
				}
			}
			if _, err := tx.Exec(ctx, `UPDATE check_runs SET classification = $2, baseline_run_id = NULLIF($3, '')::uuid WHERE id = $1`, c.ID, it.Class, it.BaselineRunID); err != nil {
				return persistence.MapError(err)
			}
			ev.add(it)
		}
		return nil
	})
	return ev, err
}

func (e *Evaluation) add(it Item) {
	e.Items = append(e.Items, it)
	if blocking(it.Class) {
		e.Blocked = true
		e.Reasons = append(e.Reasons, fmt.Sprintf("%s: %s (%s)", it.Check, it.Class, it.Reason))
	}
}

func (s *Store) classifyFailure(ctx context.Context, tx pgx.Tx, projectID, featureID, baselineRev string, c Run, it *Item) error {
	base, err := latest(ctx, tx, featureID, baselineRev, RoleBaseline, c.CheckName)
	if err != nil {
		return err
	}
	switch {
	case base != nil && base.Status == StatusPass:
		it.Class, it.Reason, it.BaselineRunID = ClassRegression, "o check passava no baseline e falha na revisão candidata", base.ID
	case base != nil && base.Status == StatusFail && base.Signature == c.Signature:
		it.Class, it.Reason, it.BaselineRunID = ClassPreexisting, "falha anterior comprovada: o baseline falha da mesma forma", base.ID
		if _, err := tx.Exec(ctx, `INSERT INTO known_failures (project_id, check_name, signature, run_id, revision, note) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (project_id, check_name, signature) DO NOTHING`, projectID, c.CheckName, c.Signature, base.ID, baselineRev, "registrada pela avaliação da revisão "+c.Revision); err != nil {
			return persistence.MapError(err)
		}
	case base != nil && base.Status == StatusFail:
		it.Class, it.Reason, it.BaselineRunID = ClassRegression, "o baseline também falha, mas de outra forma: é uma falha nova", base.ID
	default:
		// Sem baseline utilizável (não rodou, inviável, erro ou tempo limite). Uma falha já
		// comprovada antes, com a mesma assinatura, ainda vale (a prova é a execução registrada).
		var kRun, kRev string
		kerr := tx.QueryRow(ctx, `SELECT run_id::text, revision FROM known_failures WHERE project_id = $1 AND check_name = $2 AND signature = $3`,
			projectID, c.CheckName, c.Signature).Scan(&kRun, &kRev)
		if kerr == nil {
			it.Class, it.BaselineRunID = ClassPreexisting, kRun
			it.Reason = fmt.Sprintf("falha anterior já comprovada no baseline da revisão %s", kRev)
			return nil
		}
		if !errors.Is(persistence.MapError(kerr), persistence.ErrNotFound) {
			return persistence.MapError(kerr)
		}
		why := "não há execução de baseline para este check"
		if base != nil {
			why = fmt.Sprintf("o baseline não é utilizável (%s)", base.Status)
			if base.Reason != "" {
				why += ": " + base.Reason
			}
		}
		it.Class, it.Reason = ClassUnknown, why+"; a origem da falha é desconhecida e fica em investigação"
	}
	return nil
}

// Gate devolve o veredito PERSISTIDO da revisão (sem reexecutar). Só vale para a
// revisão exata: outra revisão não herda evidência.
func (s *Store) Gate(ctx context.Context, featureID, revision string) (Evaluation, error) {
	ev := Evaluation{Revision: revision, Items: []Item{}, Reasons: []string{}}
	runs, err := candidates(ctx, s.DB, featureID, revision)
	if err != nil {
		return ev, err
	}
	if len(runs) == 0 {
		ev.Blocked = true
		ev.Reasons = append(ev.Reasons, "sem evidência para esta revisão: uma alteração depois dos checks exige rodá-los de novo")
		return ev, nil
	}
	for _, c := range runs {
		it := Item{Check: c.CheckName, Class: c.Classification, CandidateRunID: c.ID, BaselineRunID: c.BaselineRunID, Reason: c.Resolution}
		if c.Classification == "" {
			ev.Blocked = true
			ev.Reasons = append(ev.Reasons, fmt.Sprintf("%s: ainda não foi avaliado contra o baseline", c.CheckName))
			it.Class = "unevaluated"
			ev.Items = append(ev.Items, it)
			continue
		}
		ev.add(it)
	}
	return ev, nil
}

// Resolve encerra uma investigação: "regression" confirma o defeito (continua
// bloqueando até corrigir); "preexisting" exige a PROVA de uma execução de
// baseline do mesmo check com a mesma assinatura (nada vira baseline por suposição).
func (s *Store) Resolve(ctx context.Context, projectID, runID, outcome, note, proofRunID string) (Run, error) {
	if strings.TrimSpace(note) == "" || len(note) > 1000 {
		return Run{}, &FieldError{"note", "justifique a resolução"}
	}
	if outcome != ClassRegression && outcome != ClassPreexisting {
		return Run{}, &FieldError{"outcome", "use regression ou preexisting"}
	}
	var out Run
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		r, err := scanRun(tx.QueryRow(ctx, `SELECT `+runCols+` FROM check_runs WHERE id = $1 AND project_id = $2 AND role = 'candidate' FOR UPDATE`, runID, projectID))
		if err != nil {
			return err
		}
		if r.Classification != ClassUnknown && r.Classification != ClassInconclusive {
			return fmt.Errorf("%w (classificação atual: %q)", ErrNotResolvable, r.Classification)
		}
		proof := ""
		if outcome == ClassPreexisting {
			p, err := scanRun(tx.QueryRow(ctx, `SELECT `+runCols+` FROM check_runs WHERE id = $1 AND project_id = $2`, proofRunID, projectID))
			if err != nil || p.Role != RoleBaseline || p.Status != StatusFail || p.CheckName != r.CheckName || r.Signature == "" || p.Signature != r.Signature {
				return ErrNoProof
			}
			proof = p.ID
			if _, err := tx.Exec(ctx, `INSERT INTO known_failures (project_id, check_name, signature, run_id, revision, note) VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (project_id, check_name, signature) DO NOTHING`, projectID, r.CheckName, r.Signature, p.ID, p.Revision, note); err != nil {
				return persistence.MapError(err)
			}
		}
		out, err = scanRun(tx.QueryRow(ctx, `UPDATE check_runs SET classification = $2, resolution = $3, baseline_run_id = NULLIF($4, '')::uuid WHERE id = $1 RETURNING `+runCols,
			runID, outcome, note, proof))
		return err
	})
	return out, err
}

// KnownFailure é uma falha anterior comprovada.
type KnownFailure struct {
	CheckName string `json:"check_name"`
	Signature string `json:"signature"`
	RunID     string `json:"run_id"`
	Revision  string `json:"revision"`
	Note      string `json:"note"`
}

// KnownFailures lista as falhas anteriores comprovadas do projeto.
func (s *Store) KnownFailures(ctx context.Context, projectID string) ([]KnownFailure, error) {
	rows, err := s.DB.Query(ctx, `SELECT check_name, signature, run_id, revision, note FROM known_failures WHERE project_id = $1 ORDER BY check_name, created_at`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []KnownFailure{}
	for rows.Next() {
		var k KnownFailure
		if err := rows.Scan(&k.CheckName, &k.Signature, &k.RunID, &k.Revision, &k.Note); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, k)
	}
	return out, persistence.MapError(rows.Err())
}

// Runs lista as execuções de uma revisão (evidências), da mais recente para a mais antiga.
func (s *Store) Runs(ctx context.Context, featureID, revision string) ([]Run, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+runCols+` FROM check_runs WHERE feature_id = $1 AND revision = $2 ORDER BY seq DESC`, featureID, revision)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Run{}
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, persistence.MapError(rows.Err())
}
