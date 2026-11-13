package prcycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrNoTarget: o projeto não tem GitHub/origem configurados; a rotina não tem o que fazer.
var ErrNoTarget = errors.New("projeto sem repositório GitHub ou origem configurados")

// pollOne lê uma PR: eventos novos (com cursor), rodadas, Issues das rodadas e estado/merge.
func (s *Service) pollOne(ctx context.Context, t Target, w Watch) (Report, error) {
	var rep Report
	st, err := t.Reader.ReadPR(ctx, w.Number)
	if err != nil {
		return rep, err
	}
	if w.Repo != "" && w.Repo != t.Reader.Repo() {
		return rep, fmt.Errorf("a PR #%d foi vinculada a outro repositório (%s)", w.Number, w.Repo)
	}
	// Estado e cabeça observada primeiro: a rodada registra a cabeça em que foi fechada.
	if _, err := s.DB.Exec(ctx, `UPDATE pr_watch SET repo = $2, base = CASE WHEN base = '' THEN $3 ELSE base END, draft = $4, head_sha = $5,
		branch = CASE WHEN branch = '' THEN $6 ELSE branch END WHERE id = $1`, w.ID, t.Reader.Repo(), st.Base, st.Draft, st.HeadSHA, st.HeadRef); err != nil {
		return rep, persistence.MapError(err)
	}
	w.Repo, w.HeadSHA, w.Draft = t.Reader.Repo(), st.HeadSHA, st.Draft
	if w.Base == "" {
		w.Base = st.Base
	}

	if w.State != "merged" {
		if err := s.readEvents(ctx, t, &w); err != nil {
			return rep, err
		}
		n, err := s.closeRounds(ctx, w)
		if err != nil {
			return rep, err
		}
		rep.Rounds = n
	}

	switch {
	case st.Merged:
		if w.State != "merged" || w.ConcludedAt == nil {
			ok, pend, err := s.conclude(ctx, t, w, st)
			rep.Pending = append(rep.Pending, pend...)
			if ok {
				rep.Merged++
			}
			if err != nil {
				return rep, err
			}
		}
	case st.State == "closed":
		if _, err := s.DB.Exec(ctx, `UPDATE pr_watch SET state = 'closed', closed_at = COALESCE(closed_at, $2) WHERE id = $1 AND state <> 'merged'`, w.ID, s.now()); err != nil {
			return rep, persistence.MapError(err)
		}
	default:
		// Reaberta: volta ao estado aberto (a decisão humana da D93 foi reabrir).
		if _, err := s.DB.Exec(ctx, `UPDATE pr_watch SET state = 'open', closed_at = NULL WHERE id = $1 AND state = 'closed'`, w.ID); err != nil {
			return rep, persistence.MapError(err)
		}
	}

	if w.State != "merged" && !st.Merged {
		n, err := s.createIssues(ctx, t, w)
		rep.Issues = n
		if err != nil {
			return rep, err
		}
	}
	return rep, nil
}

// readEvents lê revisões e comentários novos (cursores por PR) e os registra. O primeiro snapshot
// do corpo é o auditável: reentrega ou edição posterior só atualiza current_body.
func (s *Service) readEvents(ctx context.Context, t Target, w *Watch) error {
	perms := map[string]bool{}
	authorized := func(login string) (bool, error) {
		if v, ok := perms[login]; ok {
			return v, nil
		}
		p, err := t.Reader.Permission(ctx, login)
		if err != nil {
			return false, err
		}
		perms[login] = canWrite(p)
		return perms[login], nil
	}
	record := func(kind string, id, reviewID int64, author, state, body, url string, at any, auth bool) error {
		var rid any
		if reviewID != 0 {
			rid = reviewID
		}
		_, err := s.DB.Exec(ctx, `INSERT INTO pr_events (watch_id, kind, remote_id, review_id, author, state, body, current_body, url, authorized, remote_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$7,$8,$9,$10)
			ON CONFLICT (watch_id, kind, remote_id) DO UPDATE SET current_body = EXCLUDED.current_body, state = EXCLUDED.state`,
			w.ID, kind, id, rid, author, state, body, url, auth, at)
		return persistence.MapError(err)
	}

	reviews, _, err := t.Reader.ListReviews(ctx, w.Number, w.ReviewCursor)
	if err != nil {
		return err
	}
	maxReview := w.ReviewCursor
	for _, r := range reviews {
		a, err := authorized(r.Author)
		if err != nil {
			return err
		}
		if err := record("review", r.ID, 0, r.Author, r.State, r.Body, r.URL, r.At, a); err != nil {
			return err
		}
		if r.ID > maxReview {
			maxReview = r.ID
		}
	}

	conv, _, err := t.Reader.ListConversation(ctx, w.Number, w.CommentCursor)
	if err != nil {
		return err
	}
	lines, _, err := t.Reader.ListLineComments(ctx, w.Number, w.LineCursor)
	if err != nil {
		return err
	}
	maxConv, maxLine := w.CommentCursor, w.LineCursor
	for _, c := range conv {
		kind, auth := "comment", false
		if IsCommand(c.Body) {
			kind = "command"
			if auth, err = authorized(c.Author); err != nil {
				return err
			}
		}
		if err := record(kind, c.ID, 0, c.Author, "", c.Body, c.URL, c.At, auth); err != nil {
			return err
		}
		if c.ID > maxConv {
			maxConv = c.ID
		}
	}
	for _, c := range lines {
		if err := record("line_comment", c.ID, c.ReviewID, c.Author, "", c.Body, c.URL, c.At, false); err != nil {
			return err
		}
		if c.ID > maxLine {
			maxLine = c.ID
		}
	}
	// Conversa e linha têm sequências de ID próprias: cada uma tem o seu cursor.
	if _, err := s.DB.Exec(ctx, `UPDATE pr_watch SET review_cursor = GREATEST(review_cursor, $2), comment_cursor = GREATEST(comment_cursor, $3), line_cursor = GREATEST(line_cursor, $4) WHERE id = $1`, w.ID, maxReview, maxConv, maxLine); err != nil {
		return persistence.MapError(err)
	}
	w.ReviewCursor, w.CommentCursor, w.LineCursor = maxReview, maxConv, maxLine
	return nil
}
