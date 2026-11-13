package prcycle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// conclude trata uma PR integrada (AJ09-04). Só o merge confirmado conclui: confere repositório,
// PR, base e cabeça, guarda a prova (SHA do merge, data, cabeça integrada) e entrega à Sentinela a
// transição da Issue principal e das correções comprovadamente incorporadas. Aprovação e saída de
// Draft nunca chegam aqui. Devolve true quando a Issue principal foi concluída nesta passagem.
func (s *Service) conclude(ctx context.Context, t Target, w Watch, st gitops.PRState) (bool, []string, error) {
	var pending []string
	if st.Number != w.Number || strings.TrimSpace(st.MergeSHA) == "" || strings.TrimSpace(st.HeadSHA) == "" {
		return false, []string{"merge sem prova suficiente (número, SHA do merge ou cabeça ausentes): nada é concluído"}, nil
	}
	if w.Base != "" && st.Base != w.Base {
		return false, []string{fmt.Sprintf("a PR foi integrada em %q, mas a base registrada é %q: nada é concluído", st.Base, w.Base)}, nil
	}
	if w.Repo != "" && w.Repo != t.Reader.Repo() {
		return false, []string{"a PR pertence a outro repositório: nada é concluído"}, nil
	}
	at := st.MergedAt
	if at.IsZero() {
		at = s.now()
	}
	if _, err := s.DB.Exec(ctx, `UPDATE pr_watch SET state = 'merged', merge_sha = $2, merged_head_sha = $3, merged_at = COALESCE(merged_at, $4) WHERE id = $1`,
		w.ID, st.MergeSHA, st.HeadSHA, at); err != nil {
		return false, nil, persistence.MapError(err)
	}
	w.MergeSHA, w.MergedHead = st.MergeSHA, st.HeadSHA

	main, err := s.Sentinel.Transition(ctx, sentinel.Request{ProjectID: w.ProjectID, IssueKey: w.IssueKey, To: flow.Concluido, Actor: sentinel.ActorSentinel,
		Observer: "github", ObservationKey: fmt.Sprintf("merge:%d:%s", w.Number, st.MergeSHA), Evidence: flow.Evidence{MergeConfirmed: true}})
	if err != nil {
		return false, pending, err
	}
	concluded := main.Decision == sentinel.DecisionApplied || main.Duplicate
	if !concluded {
		for _, r := range main.Reasons {
			pending = append(pending, "Issue principal: "+r.Message)
		}
	}

	rows, err := s.DB.Query(ctx, `SELECT id::text, seq, issue_key, published_head, incorporated_at IS NOT NULL FROM pr_rounds WHERE watch_id = $1 AND status = 'issue_created' ORDER BY seq`, w.ID)
	if err != nil {
		return concluded, pending, persistence.MapError(err)
	}
	type rd struct {
		id, key, head string
		seq           int
		done          bool
	}
	var rs []rd
	for rows.Next() {
		var r rd
		if err := rows.Scan(&r.id, &r.seq, &r.key, &r.head, &r.done); err != nil {
			rows.Close()
			return concluded, pending, err
		}
		rs = append(rs, r)
	}
	rows.Close()
	for _, r := range rs {
		if !r.done {
			if r.head == "" {
				pending = append(pending, fmt.Sprintf("correção da rodada %d (%s) não publicou commits na PR: aguarda tratamento", r.seq, r.key))
				continue
			}
			ok, err := t.Reader.Contains(ctx, r.head, st.HeadSHA)
			if err != nil {
				return concluded, pending, err
			}
			if !ok {
				pending = append(pending, fmt.Sprintf("correção da rodada %d (%s): a cabeça integrada não contém a cabeça publicada: aguarda tratamento", r.seq, r.key))
				continue
			}
			proof, _ := json.Marshal(map[string]any{"merge_sha": st.MergeSHA, "merged_head": st.HeadSHA, "published_head": r.head, "pr": w.Number})
			if _, err := s.DB.Exec(ctx, `UPDATE pr_rounds SET incorporated_at = COALESCE(incorporated_at, $2), incorporation_proof = COALESCE(incorporation_proof, $3::jsonb) WHERE id = $1`, r.id, at, string(proof)); err != nil {
				return concluded, pending, persistence.MapError(err)
			}
		}
		out, err := s.Sentinel.Transition(ctx, sentinel.Request{ProjectID: w.ProjectID, IssueKey: r.key, To: flow.Concluido, Actor: sentinel.ActorSentinel,
			Observer: "github", ObservationKey: fmt.Sprintf("merge:%d:%s:%s", w.Number, st.MergeSHA, r.key), Evidence: flow.Evidence{MergeConfirmed: true}})
		if err != nil {
			return concluded, pending, err
		}
		if out.Decision != sentinel.DecisionApplied && !out.Duplicate {
			for _, x := range out.Reasons {
				pending = append(pending, fmt.Sprintf("correção %s: %s", r.key, x.Message))
			}
		}
	}
	// Após o merge o que não foi incorporado não muda mais: fica listado como pendência para tratamento humano.
	if concluded {
		if _, err := s.DB.Exec(ctx, `UPDATE pr_watch SET concluded_at = COALESCE(concluded_at, $2) WHERE id = $1`, w.ID, s.now()); err != nil {
			return concluded, pending, persistence.MapError(err)
		}
	}
	return concluded, pending, nil
}
