package delivery

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
)

// TaskReport reúne o que se sabe de uma tarefa para o histórico.
type TaskReport struct {
	Key      string
	Title    string
	State    string
	Criteria []string
	Commits  []CommitRef
	Risk     string
	RiskWhy  string
	Review   string // resumo
	RemoteID string
}

// CommitRef é um commit de tarefa.
type CommitRef struct {
	SHA     string
	Subject string
}

// CheckRef é um check da revisão entregue.
type CheckRef struct {
	Name           string
	Status         string
	Classification string
}

// Report é o histórico da feature para o documento vivo.
type Report struct {
	ExternalKey string
	Title       string
	Revision    string
	Branch      string
	Requirement string // resumo do requisito vigente
	Tasks       []TaskReport
	Checks      []CheckRef
	E2E         []string
	Waivers     []string
}

// BuildReport monta o histórico a partir do que está registrado, em ordem estável.
func BuildReport(ctx context.Context, db persistence.DBTX, projectID, featureID, revision, branch string) (Report, error) {
	r := Report{Revision: revision, Branch: branch}
	if err := db.QueryRow(ctx, `SELECT external_id, title FROM features WHERE project_id = $1 AND id = $2`, projectID, featureID).Scan(&r.ExternalKey, &r.Title); err != nil {
		return r, persistence.MapError(err)
	}
	var reqTitle, reqHash string
	if err := db.QueryRow(ctx, `SELECT content->>'title', content_hash FROM requirement_revisions WHERE feature_id = $1 AND state = 'applied'`, featureID).Scan(&reqTitle, &reqHash); err == nil {
		r.Requirement = fmt.Sprintf("%s (revisão do requisito %s)", reqTitle, reqHash[:12])
	}
	plan, err := (&planning.Store{}).CurrentOf(ctx, db, projectID, featureID)
	if err != nil {
		return r, err
	}
	states := map[string]string{}
	remote := map[string]string{}
	rows, err := db.Query(ctx, `SELECT task_key, state, remote_id FROM plan_tasks WHERE feature_id = $1 AND NOT removed`, featureID)
	if err != nil {
		return r, persistence.MapError(err)
	}
	for rows.Next() {
		var k, st, rid string
		if err := rows.Scan(&k, &st, &rid); err != nil {
			rows.Close()
			return r, persistence.MapError(err)
		}
		states[k], remote[k] = st, rid
	}
	rows.Close()
	commits := map[string][]CommitRef{}
	crows, err := db.Query(ctx, `SELECT task_key, sha, subject FROM feature_commits WHERE feature_id = $1 ORDER BY id`, featureID)
	if err != nil {
		return r, persistence.MapError(err)
	}
	for crows.Next() {
		var k string
		var c CommitRef
		if err := crows.Scan(&k, &c.SHA, &c.Subject); err != nil {
			crows.Close()
			return r, persistence.MapError(err)
		}
		commits[k] = append(commits[k], c)
	}
	crows.Close()
	for _, t := range plan.Plan.Tasks {
		tr := TaskReport{Key: t.Key, Title: t.Title, State: states[t.Key], Commits: commits[t.Key], RemoteID: remote[t.Key]}
		for _, c := range t.Criteria {
			tr.Criteria = append(tr.Criteria, strings.TrimSpace(c.Text))
		}
		var final string
		var reasons int
		if err := db.QueryRow(ctx, `SELECT final, jsonb_array_length(reasons) FROM risk_assessments WHERE feature_id = $1 AND task_key = $2 ORDER BY seq DESC LIMIT 1`, featureID, t.Key).Scan(&final, &reasons); err == nil {
			tr.Risk, tr.RiskWhy = final, fmt.Sprintf("%d razão(ões) registrada(s)", reasons)
		}
		var status string
		var round int
		if err := db.QueryRow(ctx, `SELECT status, round FROM reviews WHERE feature_id = $1 AND task_key = $2 ORDER BY seq DESC LIMIT 1`, featureID, t.Key).Scan(&status, &round); err == nil {
			var crit, rel, low int
			_ = db.QueryRow(ctx, `SELECT count(*) FILTER (WHERE severity = 'critical'), count(*) FILTER (WHERE severity = 'relevant'), count(*) FILTER (WHERE severity = 'low')
				FROM review_findings f JOIN reviews rv ON rv.id = f.review_id WHERE rv.feature_id = $1 AND rv.task_key = $2`, featureID, t.Key).Scan(&crit, &rel, &low)
			tr.Review = fmt.Sprintf("%s na rodada %d (achados: %d crítico(s), %d relevante(s), %d baixo(s))", status, round, crit, rel, low)
		}
		r.Tasks = append(r.Tasks, tr)
	}
	chk, err := db.Query(ctx, `SELECT DISTINCT ON (check_name) check_name, status, COALESCE(classification, '') FROM check_runs
		WHERE feature_id = $1 AND revision = $2 AND role = 'candidate' ORDER BY check_name, seq DESC`, featureID, revision)
	if err != nil {
		return r, persistence.MapError(err)
	}
	for chk.Next() {
		var c CheckRef
		if err := chk.Scan(&c.Name, &c.Status, &c.Classification); err != nil {
			chk.Close()
			return r, persistence.MapError(err)
		}
		r.Checks = append(r.Checks, c)
	}
	chk.Close()
	ev, err := db.Query(ctx, `SELECT DISTINCT ON (journey) journey, status, browser, (SELECT count(*) FROM e2e_captures c WHERE c.evidence_id = e.id)
		FROM e2e_evidence e WHERE feature_id = $1 AND revision = $2 ORDER BY journey, created_at DESC, id DESC`, featureID, revision)
	if err != nil {
		return r, persistence.MapError(err)
	}
	for ev.Next() {
		var j, st, br string
		var n int
		if err := ev.Scan(&j, &st, &br, &n); err != nil {
			ev.Close()
			return r, persistence.MapError(err)
		}
		r.E2E = append(r.E2E, fmt.Sprintf("jornada %s: %s (%s, %d captura(s))", j, st, br, n))
	}
	ev.Close()
	wv, err := db.Query(ctx, `SELECT subject, reason, alt_kind FROM e2e_waivers WHERE feature_id = $1 AND revision = $2 ORDER BY created_at, id`, featureID, revision)
	if err != nil {
		return r, persistence.MapError(err)
	}
	for wv.Next() {
		var s, why, alt string
		if err := wv.Scan(&s, &why, &alt); err != nil {
			wv.Close()
			return r, persistence.MapError(err)
		}
		r.Waivers = append(r.Waivers, fmt.Sprintf("%s: %s (evidência alternativa: %s)", s, why, alt))
	}
	wv.Close()
	return r, nil
}

// Render gera o Markdown do histórico da feature (determinístico).
func (r Report) Render() string {
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f, a...) }
	w("# %s — %s\n\n", r.ExternalKey, r.Title)
	w("> Gerado pela La fabrique. Branch `%s`, revisão `%s`. Não edite à mão: o histórico é regenerado a cada entrega.\n\n", r.Branch, r.Revision)
	if r.Requirement != "" {
		w("## Requisito\n\n%s\n\n", r.Requirement)
	}
	w("## Tarefas\n\n")
	for _, t := range r.Tasks {
		w("### %s — %s\n\n- Estado: %s\n", t.Key, t.Title, t.State)
		if t.RemoteID != "" {
			w("- Item na origem: %s\n", t.RemoteID)
		}
		if t.Risk != "" {
			w("- Risco: %s (%s)\n", t.Risk, t.RiskWhy)
		}
		if t.Review != "" {
			w("- Revisão: %s\n", t.Review)
		}
		for _, c := range t.Criteria {
			w("- Critério: %s\n", c)
		}
		for _, c := range t.Commits {
			sha := c.SHA
			if len(sha) > 12 {
				sha = sha[:12]
			}
			w("- Commit `%s` %s\n", sha, c.Subject)
		}
		w("\n")
	}
	w("## Checks da revisão\n\n")
	if len(r.Checks) == 0 {
		w("Nenhum check registrado para esta revisão.\n\n")
	}
	for _, c := range r.Checks {
		cl := c.Classification
		if cl == "" {
			cl = "não avaliado"
		}
		w("- %s: %s (%s)\n", c.Name, c.Status, cl)
	}
	if len(r.E2E) > 0 || len(r.Waivers) > 0 {
		w("\n## E2E\n\n")
		for _, e := range r.E2E {
			w("- %s\n", e)
		}
		for _, x := range r.Waivers {
			w("- Justificativa: %s\n", x)
		}
	}
	return b.String()
}

// Hash é o SHA-256 do documento.
func (r Report) Hash() string {
	h := sha256.Sum256([]byte(r.Render()))
	return hex.EncodeToString(h[:])
}
