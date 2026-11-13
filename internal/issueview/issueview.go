// Package issueview monta, para a página Issues (AJ02-03), o que o administrador precisa ver além do
// status externo: o estado principal da Sentinela, o avanço interno por trilha, a etapa obrigatória que
// segura o avanço, os motivos de bloqueio, os commits e evidências ligados à Issue e a diferença entre o
// estado interno e o que já foi projetado na origem.
package issueview

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/trackadmit"
)

// TrackStage é a etapa de uma trilha.
type TrackStage struct {
	Track string `json:"track"`
	Stage string `json:"stage"` // none | implementing | implemented | reviewed
}

// Evidence liga um resultado de trilha à revisão do código.
type Evidence struct {
	Track    string `json:"track"`
	Revision string `json:"revision"`
}

// Publication é uma publicação na PR.
type Publication struct {
	Kind  string `json:"kind"`
	Head  string `json:"head"`
	PRURL string `json:"pr_url,omitempty"`
	State string `json:"state"`
}

// Projection compara o estado interno com o que a origem mostra.
type Projection struct {
	Internal string `json:"internal"`
	External string `json:"external"` // status canônico observado na origem
	InSync   bool   `json:"in_sync"`
	Pending  string `json:"pending,omitempty"` // motivo do atraso, se houver
}

// Progress é a visão por Issue.
type Progress struct {
	Key        string        `json:"key"`
	Internal   string        `json:"internal_status"`
	Aggregate  string        `json:"aggregate_status,omitempty"`
	HeldBy     string        `json:"held_by,omitempty"` // trilha que segura o avanço
	Tracks     []TrackStage  `json:"tracks"`
	Blocking   []string      `json:"blocking"`
	Commits    []string      `json:"commits"`
	Evidence   []Evidence    `json:"evidence"`
	Publish    []Publication `json:"publications"`
	Projection Projection    `json:"projection"`
}

var stageName = map[flow.Stage]string{flow.StageNone: "none", flow.StageImplementing: "implementing", flow.StageImplemented: "implemented", flow.StageReviewed: "reviewed"}

// DB é o que a visão precisa.
type DB interface{ persistence.DBTX }

// ForProject devolve a visão de todas as Issues rastreadas do projeto, por chave.
func ForProject(ctx context.Context, db DB, projectID string) (map[string]Progress, error) {
	rows, err := db.Query(ctx, `SELECT i.id::text, i.external_key, c.code, COALESCE(i.feature_id::text,''), COALESCE(r.status_code,'')
		FROM flow_issues i JOIN status_catalog c ON c.id = i.status_id
		LEFT JOIN issue_registry r ON r.project_id = i.project_id AND r.issue_key = i.external_key
		WHERE i.project_id = $1 ORDER BY i.external_key`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	type row struct{ id, key, internal, feature, external string }
	var rs []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.key, &r.internal, &r.feature, &r.external); err != nil {
			rows.Close()
			return nil, err
		}
		rs = append(rs, r)
	}
	rows.Close()
	out := map[string]Progress{}
	pend := map[string]string{}
	pr, err := db.Query(ctx, `SELECT issue_key, reason || CASE WHEN detail <> '' THEN ': ' || detail ELSE '' END FROM projection_pending WHERE project_id = $1`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	for pr.Next() {
		var k, v string
		if err := pr.Scan(&k, &v); err != nil {
			pr.Close()
			return nil, err
		}
		pend[k] = v
	}
	pr.Close()
	for _, r := range rs {
		p := Progress{Key: r.key, Internal: r.internal, Tracks: []TrackStage{}, Blocking: []string{}, Commits: []string{}, Evidence: []Evidence{}, Publish: []Publication{}}
		p.Projection = Projection{Internal: r.internal, External: r.external, InSync: r.external == "" || r.external == r.internal, Pending: pend[r.key]}
		if r.feature != "" {
			if err := fill(ctx, db, r.feature, &p); err != nil {
				return nil, err
			}
		}
		// Motivos de bloqueio: a última recusa da Sentinela, se for mais nova que a última aplicação.
		br, err := db.Query(ctx, `SELECT decision, to_code, reasons FROM flow_transitions WHERE issue_id::text = $1 ORDER BY id DESC LIMIT 5`, r.id)
		if err != nil {
			return nil, persistence.MapError(err)
		}
		for br.Next() {
			var dec, to string
			var reasons []byte
			if err := br.Scan(&dec, &to, &reasons); err != nil {
				br.Close()
				return nil, err
			}
			if dec == "applied" {
				break
			}
			var rs []flow.Problem
			_ = json.Unmarshal(reasons, &rs)
			for _, x := range rs {
				p.Blocking = append(p.Blocking, "para "+to+": "+x.Message)
			}
			break
		}
		br.Close()
		out[r.key] = p
	}
	return out, nil
}

func fill(ctx context.Context, db DB, feature string, p *Progress) error {
	status, behind, tracks, err := trackadmit.ProgressOf(ctx, db, feature)
	if err != nil {
		return err
	}
	if len(tracks) > 0 {
		p.Aggregate, p.HeldBy = status, string(behind)
		for _, tr := range []flow.Track{flow.Frontend, flow.Backend} {
			if st, ok := tracks[tr]; ok {
				p.Tracks = append(p.Tracks, TrackStage{Track: string(tr), Stage: stageName[st]})
			}
		}
	}
	cr, err := db.Query(ctx, `SELECT integrated_sha FROM consolidated_commits WHERE feature_id::text = $1 ORDER BY created_at, integrated_sha`, feature)
	if err != nil {
		return persistence.MapError(err)
	}
	for cr.Next() {
		var s string
		if err := cr.Scan(&s); err != nil {
			cr.Close()
			return err
		}
		p.Commits = append(p.Commits, s)
	}
	cr.Close()
	er, err := db.Query(ctx, `SELECT track, revision FROM track_results WHERE feature_id::text = $1 ORDER BY track, created_at`, feature)
	if err != nil {
		return persistence.MapError(err)
	}
	for er.Next() {
		var e Evidence
		if err := er.Scan(&e.Track, &e.Revision); err != nil {
			er.Close()
			return err
		}
		p.Evidence = append(p.Evidence, e)
	}
	er.Close()
	pr, err := db.Query(ctx, `SELECT kind, head_sha, pr_url, state FROM publications WHERE feature_id::text = $1 ORDER BY created_at`, feature)
	if err != nil {
		return persistence.MapError(err)
	}
	for pr.Next() {
		var x Publication
		if err := pr.Scan(&x.Kind, &x.Head, &x.PRURL, &x.State); err != nil {
			pr.Close()
			return err
		}
		p.Publish = append(p.Publish, x)
	}
	pr.Close()
	if p.HeldBy != "" {
		p.Blocking = append(p.Blocking, "a trilha "+strings.ToLower(p.HeldBy)+" é a etapa obrigatória mais atrasada e segura o avanço")
	}
	return nil
}
