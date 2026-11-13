package integration

import (
	"context"
	"strings"
	"testing"

	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/issueview"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// AJ02-03 / CT06, CT15 — estado principal, avanço por trilha, o que segura o avanço, vínculos e a diferença
// entre o estado interno e a projeção externa.
func TestIssueViewShowsTrackProgressBlockingAndProjectionGap(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, "iv")
	co := &sentinel.Coordinator{DB: db.Pool}
	var feature string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'IV-1','f') RETURNING id::text`, p.ID).Scan(&feature)
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "IV-1", Kind: "feature", Status: statusflow.Implementacao, FeatureID: feature}); err != nil {
		t.Fatal(err)
	}
	// O registro externo ainda mostra PLANEJAMENTO; há uma pendência de projeção.
	_, _ = db.Pool.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, title, raw_status, status_code, revision, kind)
		VALUES ($1,'jira','1','IV-1','t','Planning','PLANEJAMENTO','v1','feature')`, p.ID)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO projection_pending (project_id, business_key, issue_key, version, target_code, reason, detail) VALUES ($1,'k','IV-1',2,'IMPLEMENTACAO','write','falha de escrita')`, p.ID)
	seedSubtask(t, db, p.ID, feature, "front-a", "FRONTEND", `[]`, `[]`)
	seedSubtask(t, db, p.ID, feature, "back-a", "BACKEND", `[]`, `[]`)
	_, _ = db.Pool.Exec(ctx, `UPDATE subtasks SET impl_state='done', review_state='done' WHERE feature_id=$1 AND task_key='front-a'`, feature)
	_, _ = db.Pool.Exec(ctx, `UPDATE subtasks SET impl_state='running' WHERE feature_id=$1 AND task_key='back-a'`, feature)
	var run string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO consolidation_runs (project_id, feature_id, delivery_branch, lease_until, state) VALUES ($1,$2,'b', now(),'done') RETURNING id::text`, p.ID, feature).Scan(&run)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO consolidated_commits (feature_id, original_sha, integrated_sha, changed, run_id) VALUES ($1,'o1','i1',true,$2)`, feature, run)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO track_results (feature_id, track, revision, requirement, evidence) VALUES ($1,'FRONTEND','r1','{}','{}')`, feature)
	_, _ = db.Pool.Exec(ctx, `INSERT INTO publications (project_id, feature_id, kind, branch, head_sha, pr_url) VALUES ($1,$2,'draft','b','h1','https://gh/pull/1')`, p.ID, feature)
	// Uma recusa da Sentinela vira motivo de bloqueio.
	if _, err := co.Transition(ctx, sentinel.Request{ProjectID: p.ID, IssueKey: "IV-1", To: statusflow.Review, Actor: sentinel.ActorSentinel, ObservationKey: "x1",
		Evidence: statusflow.Evidence{Tracks: map[statusflow.Track]statusflow.Stage{statusflow.Frontend: statusflow.StageReviewed, statusflow.Backend: statusflow.StageImplementing}, GatesVerified: true}}); err != nil {
		t.Fatal(err)
	}
	views, err := issueview.ForProject(ctx, db.Pool, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	v := views["IV-1"]
	if v.Internal != statusflow.Implementacao || v.Aggregate != statusflow.Implementacao || v.HeldBy != "BACKEND" {
		t.Fatalf("%+v", v)
	}
	if len(v.Tracks) != 2 || v.Tracks[0].Track != "FRONTEND" || v.Tracks[0].Stage != "reviewed" || v.Tracks[1].Stage != "implementing" {
		t.Fatalf("%+v", v.Tracks)
	}
	if v.Projection.InSync || v.Projection.External != "PLANEJAMENTO" || !strings.Contains(v.Projection.Pending, "falha de escrita") {
		t.Fatalf("%+v", v.Projection)
	}
	all := strings.Join(v.Blocking, "|")
	if !strings.Contains(all, "para REVIEW") || !strings.Contains(all, "segura o avanço") {
		t.Fatalf("%v", v.Blocking)
	}
	if len(v.Commits) != 1 || v.Commits[0] != "i1" || len(v.Evidence) != 1 || len(v.Publish) != 1 || v.Publish[0].PRURL == "" {
		t.Fatalf("%+v", v)
	}
}
