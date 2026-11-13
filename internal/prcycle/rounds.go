package prcycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

type event struct {
	ID                int64
	Kind, Author, URL string
	State, Body       string
	RemoteID          int64
	ReviewID          *int64
	Authorized        bool
	At                time.Time
}

func loadOpen(ctx context.Context, tx pgx.Tx, watchID string) ([]event, error) {
	rows, err := tx.Query(ctx, `SELECT id, kind, remote_id, review_id, author, state, body, url, authorized, remote_at
		FROM pr_events WHERE watch_id = $1 AND round_id IS NULL ORDER BY remote_at, id`, watchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []event
	for rows.Next() {
		var e event
		if err := rows.Scan(&e.ID, &e.Kind, &e.RemoteID, &e.ReviewID, &e.Author, &e.State, &e.Body, &e.URL, &e.Authorized, &e.At); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func isCloser(e event) bool {
	switch e.Kind {
	case "review":
		return e.Authorized && e.State == "CHANGES_REQUESTED"
	case "command":
		return e.Authorized
	}
	return false
}

func actionable(e event) bool {
	return (e.Kind == "comment" || e.Kind == "line_comment") && strings.TrimSpace(e.Body) != ""
}

func roundKey(pr int, set []event, closer event) string {
	var ids []string
	for _, e := range set {
		ids = append(ids, fmt.Sprintf("%s:%d", e.Kind, e.RemoteID))
	}
	if len(ids) == 0 {
		ids = append(ids, fmt.Sprintf("%s:%d", closer.Kind, closer.RemoteID))
	}
	sort.Strings(ids)
	h := sha256.Sum256([]byte(fmt.Sprintf("pr:%d|%s", pr, strings.Join(ids, ","))))
	return hex.EncodeToString(h[:16])
}

// closeRounds fecha as rodadas pendentes da PR, em ordem estável de eventos. Cada fechamento
// (revisão formal pedindo mudanças ou comando de revisor autorizado) leva os comentários acionáveis
// ainda sem rodada que ele abrange. O mesmo conjunto gera a mesma rodada (chave estável), então
// repetição e reentrega não criam outra. Comentários posteriores ficam para a próxima rodada.
func (s *Service) closeRounds(ctx context.Context, w Watch) (int, error) {
	created := 0
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		// Trava a PR: duas passagens concorrentes não fecham a mesma rodada duas vezes.
		var lock string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM pr_watch WHERE id = $1 FOR UPDATE`, w.ID).Scan(&lock); err != nil {
			return err
		}
		for {
			open, err := loadOpen(ctx, tx, w.ID)
			if err != nil {
				return err
			}
			var closer *event
			for i := range open {
				if isCloser(open[i]) {
					closer = &open[i]
					break
				}
			}
			if closer == nil {
				return nil
			}
			// Linhas de uma revisão que ainda vai fechar a sua própria rodada pertencem a ela.
			pendingReviews := map[int64]bool{}
			for _, e := range open {
				if e.Kind == "review" && isCloser(e) && e.RemoteID != closer.RemoteID {
					pendingReviews[e.RemoteID] = true
				}
			}
			var set []event
			for _, e := range open {
				if !actionable(e) || e.At.After(closer.At) && !(closer.Kind == "review" && e.ReviewID != nil && *e.ReviewID == closer.RemoteID) {
					continue
				}
				if e.ReviewID != nil && pendingReviews[*e.ReviewID] {
					continue
				}
				set = append(set, e)
			}
			consumed := append([]event{}, set...)
			consumed = append(consumed, *closer)
			key := roundKey(w.Number, set, *closer)

			var id string
			var existing bool
			err = tx.QueryRow(ctx, `SELECT id::text FROM pr_rounds WHERE watch_id = $1 AND round_key = $2`, w.ID, key).Scan(&id)
			switch {
			case err == nil:
				existing = true
			case errors.Is(err, pgx.ErrNoRows):
			default:
				return err
			}
			if !existing && len(set) == 0 && closer.Kind == "command" {
				// Comando sem comentários novos: se a rodada anterior acabou de ser fechada por revisão
				// formal, é o mesmo conjunto; senão não há achados e a pendência fica visível.
				var prev string
				e := tx.QueryRow(ctx, `SELECT id::text FROM pr_rounds WHERE watch_id = $1 AND status <> 'ambiguous' ORDER BY seq DESC LIMIT 1`, w.ID).Scan(&prev)
				if e == nil {
					id, existing = prev, true
				}
			}
			if !existing {
				status, detail := "pending_issue", ""
				if len(set) == 0 && !(closer.Kind == "review" && strings.TrimSpace(closer.Body) != "") {
					status, detail = "ambiguous", "fechamento sem comentários acionáveis: confira a PR antes de gerar a correção"
				}
				head := w.HeadSHA
				var seq int
				if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(seq),0)+1 FROM pr_rounds WHERE watch_id = $1`, w.ID).Scan(&seq); err != nil {
					return err
				}
				if err := tx.QueryRow(ctx, `INSERT INTO pr_rounds (watch_id, round_key, seq, closing_kind, closing_id, closed_by, head_sha, status, detail)
					VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text`, w.ID, key, seq, closer.Kind, closer.RemoteID, closer.Author, head, status, detail).Scan(&id); err != nil {
					return err
				}
				created++
			}
			for _, e := range consumed {
				if _, err := tx.Exec(ctx, `UPDATE pr_events SET round_id = $2 WHERE id = $1 AND round_id IS NULL`, e.ID, id); err != nil {
					return err
				}
			}
		}
	})
	return created, persistence.MapError(err)
}

type roundRow struct {
	ID, Key  string
	Seq      int
	Head     string
	Closer   string
	CloserBy string
}

// createIssues cria na origem a Issue REPROVADO_PR de cada rodada pendente, pelo ledger de efeitos:
// a chave é a rodada, o marcador no corpo prova a existência e um timeout depois da criação é
// reconciliado por leitura antes de qualquer repetição.
func (s *Service) createIssues(ctx context.Context, t Target, w Watch) (int, error) {
	rows, err := s.DB.Query(ctx, `SELECT id::text, round_key, seq, head_sha, closing_kind, closed_by FROM pr_rounds WHERE watch_id = $1 AND status = 'pending_issue' ORDER BY seq`, w.ID)
	if err != nil {
		return 0, persistence.MapError(err)
	}
	var rs []roundRow
	for rows.Next() {
		var r roundRow
		if err := rows.Scan(&r.ID, &r.Key, &r.Seq, &r.Head, &r.Closer, &r.CloserBy); err != nil {
			rows.Close()
			return 0, err
		}
		rs = append(rs, r)
	}
	rows.Close()
	made := 0
	for _, r := range rs {
		body, err := s.roundBody(ctx, w, r)
		if err != nil {
			return made, err
		}
		title := fmt.Sprintf("Correção da PR #%d (%s) — rodada %d", w.Number, w.IssueKey, r.Seq)
		marker := Marker(r.Key)
		ad := &issueAdapter{t: t, title: title, body: body + "\n\n" + marker, marker: marker, status: rawStatus(t, flow.ReprovadoPR)}
		res, err := s.Ledger.Run(ctx, effects.Spec{ProjectID: w.ProjectID, BusinessKey: "round:" + w.ID + ":" + r.Key, Operation: "create_correction_issue",
			Request: map[string]any{"pr": w.Number, "round": r.Key, "origin_issue": w.IssueKey}}, ad)
		if err != nil {
			_, _ = s.DB.Exec(ctx, `UPDATE pr_rounds SET detail = $2 WHERE id = $1`, r.ID, clip("a criação da Issue na origem aguarda reconciliação: "+err.Error()))
			return made, err
		}
		var proof struct {
			ID, Key, URL, Title, Version, Priority string
		}
		if res.Entry.Proof != nil {
			_ = json.Unmarshal(res.Entry.Proof, &proof)
		}
		if proof.ID == "" {
			return made, fmt.Errorf("a origem não devolveu a identidade da Issue da rodada %d", r.Seq)
		}
		it := integrations.Item{ID: proof.ID, Key: proof.Key, Title: proof.Title, URL: proof.URL, Version: proof.Version, Priority: proof.Priority, Description: body}
		if t.Register != nil {
			if err := t.Register(ctx, it, w.IssueKey); err != nil {
				return made, err
			}
		}
		if _, err := s.DB.Exec(ctx, `UPDATE pr_rounds SET status = 'issue_created', issue_remote_id = $2, issue_key = $3, detail = '' WHERE id = $1`, r.ID, proof.ID, proof.Key); err != nil {
			return made, persistence.MapError(err)
		}
		made++
	}
	return made, nil
}

// Marker é o marcador da rodada no corpo da Issue (prova de existência por leitura).
func Marker(roundKey string) string { return "[lf-round:" + roundKey + "]" }

func clip(s string) string {
	if len(s) > 300 {
		return s[:300]
	}
	return s
}

func rawStatus(t Target, code string) string {
	var best string
	for raw, c := range t.Map.Remote {
		if n, ok := flow.Normalize(c); ok && n.Code == code {
			if best == "" || raw < best {
				best = raw
			}
		}
	}
	return best
}

// roundBody monta o contexto da Issue: PR, Issue original, revisão, comentários com IDs e links.
func (s *Service) roundBody(ctx context.Context, w Watch, r roundRow) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Correção solicitada na revisão da PR #%d (%s).\n\n", w.Number, w.Repo)
	fmt.Fprintf(&b, "- Issue original: %s\n- Rodada: %d (fechada por %s de %s)\n- Cabeça observada: %s\n\n", w.IssueKey, r.Seq, closerName(r.Closer), r.CloserBy, r.Head)
	rows, err := s.DB.Query(ctx, `SELECT kind, remote_id, COALESCE(review_id,0), author, state, body, url FROM pr_events WHERE round_id = $1 ORDER BY remote_at, id`, r.ID)
	if err != nil {
		return "", persistence.MapError(err)
	}
	defer rows.Close()
	b.WriteString("Comentários da rodada:\n")
	for rows.Next() {
		var kind, author, state, body, url string
		var id, review int64
		if err := rows.Scan(&kind, &id, &review, &author, &state, &body, &url); err != nil {
			return "", err
		}
		switch kind {
		case "review":
			fmt.Fprintf(&b, "\n• Revisão #%d de %s (%s) %s\n%s\n", id, author, state, url, quote(body))
		case "command":
			fmt.Fprintf(&b, "\n• Comando #%d de %s %s\n", id, author, url)
		default:
			ref := ""
			if review != 0 {
				ref = fmt.Sprintf(" (revisão #%d)", review)
			}
			fmt.Fprintf(&b, "\n• Comentário #%d de %s%s %s\n%s\n", id, author, ref, url, quote(body))
		}
	}
	return b.String(), rows.Err()
}

func closerName(k string) string {
	if k == "command" {
		return "comando"
	}
	return "revisão formal"
}

func quote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		s = s[:2000] + "…"
	}
	if s == "" {
		return ""
	}
	return "> " + strings.ReplaceAll(s, "\n", "\n> ")
}

type issueAdapter struct {
	t      Target
	title  string
	body   string
	marker string
	status string
}

func (a *issueAdapter) proof(it integrations.Item) map[string]any {
	return map[string]any{"ID": it.ID, "Key": it.Key, "URL": it.URL, "Title": it.Title, "Version": it.Version, "Priority": it.Priority}
}

func (a *issueAdapter) Apply(ctx context.Context, _ effects.Op) (any, error) {
	it, err := a.t.Origin.CreateItem(ctx, a.t.Container, integrations.ItemInput{Title: a.title, Description: a.body, Status: a.status})
	if err != nil {
		var ie *integrations.Error
		if errors.As(err, &ie) && ie.Kind == integrations.ErrInvalid {
			return nil, fmt.Errorf("%w: %v", effects.ErrNotApplied, err)
		}
		return nil, err // timeout: resultado desconhecido, o ledger verifica
	}
	return a.proof(it), nil
}

func (a *issueAdapter) Verify(ctx context.Context, _ effects.Op) (effects.Verdict, error) {
	cursor := ""
	for i := 0; i < 20; i++ {
		page, err := a.t.Origin.ListItems(ctx, a.t.Container, integrations.ListOptions{IncludeDone: true, Cursor: cursor, Limit: 100})
		if err != nil {
			return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "não foi possível listar as Issues da origem: " + err.Error()}, nil
		}
		for _, it := range page.Items {
			if strings.Contains(it.Description, a.marker) {
				return effects.Verdict{Status: effects.VerdictConfirmed, Proof: a.proof(it)}, nil
			}
		}
		if page.NextCursor == "" {
			return effects.Verdict{Status: effects.VerdictAbsent}, nil
		}
		cursor = page.NextCursor
	}
	return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "a busca pelo marcador da rodada excedeu o limite de páginas"}, nil
}
