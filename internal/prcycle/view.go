package prcycle

import (
	"context"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// RoundView é uma rodada para o painel.
type RoundView struct {
	Seq           int    `json:"seq"`
	Key           string `json:"key"`
	Status        string `json:"status"`
	Detail        string `json:"detail,omitempty"`
	ClosingKind   string `json:"closing_kind"`
	ClosedBy      string `json:"closed_by"`
	IssueKey      string `json:"issue_key,omitempty"`
	Comments      int    `json:"comments"`
	PublishedHead string `json:"published_head,omitempty"`
	Incorporated  bool   `json:"incorporated"`
}

// WatchView é uma PR gerenciada para o painel.
type WatchView struct {
	PRNumber      int         `json:"pr_number"`
	IssueKey      string      `json:"issue_key"`
	State         string      `json:"state"`
	Draft         bool        `json:"draft"`
	HeadSHA       string      `json:"head_sha"`
	MergeSHA      string      `json:"merge_sha,omitempty"`
	NeedsDecision bool        `json:"needs_decision"` // fechada sem merge (D93)
	LastError     string      `json:"last_error,omitempty"`
	LastPolledAt  *time.Time  `json:"last_polled_at"`
	Rounds        []RoundView `json:"rounds"`
	// Pendentes: rodadas ambíguas, correções não incorporadas após o merge ou sem Issue criada.
	Pending []string `json:"pending"`
}

// List devolve as PRs gerenciadas do projeto com rodadas e pendências.
func (s *Service) List(ctx context.Context, projectID string) ([]WatchView, error) {
	rows, err := s.DB.Query(ctx, `SELECT id::text, pr_number, issue_key, state, draft, head_sha, merge_sha, last_error, last_polled_at FROM pr_watch WHERE project_id = $1 ORDER BY pr_number`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var out []WatchView
	var ids []string
	for rows.Next() {
		var v WatchView
		var id string
		if err := rows.Scan(&id, &v.PRNumber, &v.IssueKey, &v.State, &v.Draft, &v.HeadSHA, &v.MergeSHA, &v.LastError, &v.LastPolledAt); err != nil {
			rows.Close()
			return nil, err
		}
		v.NeedsDecision = v.State == "closed"
		v.Rounds, v.Pending = []RoundView{}, []string{}
		out, ids = append(out, v), append(ids, id)
	}
	rows.Close()
	for i, id := range ids {
		rr, err := s.DB.Query(ctx, `SELECT r.seq, r.round_key, r.status, r.detail, r.closing_kind, r.closed_by, r.issue_key, r.published_head, r.incorporated_at IS NOT NULL,
			(SELECT count(*) FROM pr_events e WHERE e.round_id = r.id AND e.kind IN ('comment','line_comment'))
			FROM pr_rounds r WHERE r.watch_id = $1 ORDER BY r.seq`, id)
		if err != nil {
			return nil, persistence.MapError(err)
		}
		for rr.Next() {
			var r RoundView
			if err := rr.Scan(&r.Seq, &r.Key, &r.Status, &r.Detail, &r.ClosingKind, &r.ClosedBy, &r.IssueKey, &r.PublishedHead, &r.Incorporated, &r.Comments); err != nil {
				rr.Close()
				return nil, err
			}
			out[i].Rounds = append(out[i].Rounds, r)
			switch {
			case r.Status == "ambiguous":
				out[i].Pending = append(out[i].Pending, "rodada "+itoa(r.Seq)+": "+r.Detail)
			case r.Status == "pending_issue":
				out[i].Pending = append(out[i].Pending, "rodada "+itoa(r.Seq)+": Issue ainda não criada na origem")
			case out[i].State == "merged" && !r.Incorporated:
				out[i].Pending = append(out[i].Pending, "correção "+r.IssueKey+" (rodada "+itoa(r.Seq)+") não incorporada ao merge: aguarda tratamento")
			}
		}
		rr.Close()
	}
	return out, nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}
