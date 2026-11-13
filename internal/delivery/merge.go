package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// Estados da projeção do status na origem.
const (
	ProjectionNone    = "none"
	ProjectionPending = "pending"
	ProjectionDone    = "done"
	ProjectionFailed  = "failed"
)

// OriginCompleter conclui o item na origem SOMENTE com merge confirmado (tasksync.Publisher implementa).
type OriginCompleter interface {
	Complete(ctx context.Context, project, keyBase, itemID, prURL string) (effects.Result, error)
}

// Completer observa o merge e propaga a conclusão.
type Completer struct {
	DB     DB
	Merges tasksync.MergeVerifier
	Origin OriginCompleter
	Queue  *queue.Queue // opcional: devolve à fila o que espera esse merge
	Now    func() time.Time
}

func (c *Completer) now() time.Time {
	if c.Now != nil {
		return c.Now().UTC()
	}
	return time.Now().UTC()
}

// Observation é o resultado de uma consulta.
type Observation struct {
	DeliveryID      string       `json:"delivery_id"`
	Outcome         string       `json:"outcome"` // open | closed | merged | unknown
	MergeSHA        string       `json:"merge_sha,omitempty"`
	Projection      string       `json:"projection"`
	ProjectionError string       `json:"projection_error,omitempty"`
	Released        []queue.Item `json:"released"`
	Detail          string       `json:"detail,omitempty"`
}

type deliveryRow struct {
	project, feature, branch, prURL, state, mergeSHA, projection string
	attempt                                                      int
}

func (c *Completer) load(ctx context.Context, id string) (deliveryRow, error) {
	var r deliveryRow
	err := c.DB.QueryRow(ctx, `SELECT project_id::text, feature_id::text, branch, pr_url, state, merge_sha, projection, attempt FROM deliveries WHERE id::text = $1`, id).
		Scan(&r.project, &r.feature, &r.branch, &r.prURL, &r.state, &r.mergeSHA, &r.projection, &r.attempt)
	return r, persistence.MapError(err)
}

func (c *Completer) note(ctx context.Context, id, outcome, detail string) {
	_, _ = c.DB.Exec(ctx, `INSERT INTO delivery_observations (delivery_id, outcome, detail) VALUES ($1, $2, $3)`, id, outcome, detail)
	_, _ = c.DB.Exec(ctx, `UPDATE deliveries SET observed_at = $2 WHERE id = $1`, id, c.now())
}

// Observe consulta o GitHub (só quando chamado: ponto de descoberta D89) e age conforme o estado:
//   - aberta (rascunho ou não): nada é liberado;
//   - fechada SEM merge: a entrega fica `closed` e nada é liberado (decisão humana: T15-04);
//   - integrada com SHA: registra o SHA, conclui o item na origem (ledger) e devolve à fila o que
//     esperava esse merge. Falha ao projetar o status NÃO desfaz a liberação: fica visível
//     (`projection = failed`) e `Reconcile`/nova observação tenta de novo, sem duplicar.
//
// Uma consulta que falha (rede, limite) deixa tudo como está e é registrada como `unknown`.
func (c *Completer) Observe(ctx context.Context, deliveryID, keyBase, itemID string) (Observation, error) {
	obs := Observation{DeliveryID: deliveryID, Released: []queue.Item{}}
	d, err := c.load(ctx, deliveryID)
	if err != nil {
		return obs, err
	}
	obs.Projection = d.projection
	if d.prURL == "" || (d.state != "draft_open" && d.state != "merged" && d.state != "closed") {
		return obs, &FieldError{"delivery", "a entrega não tem uma PR aberta para observar"}
	}
	if d.state == "merged" {
		// Já integrada: só garante a projeção pendente ou falha.
		return c.finishMerged(ctx, d, deliveryID, keyBase, itemID, d.mergeSHA, obs)
	}
	st, err := c.Merges.Merged(ctx, d.prURL)
	if err != nil {
		obs.Outcome, obs.Detail = "unknown", "não foi possível consultar a PR: "+err.Error()
		c.note(ctx, deliveryID, "unknown", obs.Detail)
		return obs, nil
	}
	switch {
	case st.Merged && strings.TrimSpace(st.SHA) != "":
		obs.Outcome, obs.MergeSHA = "merged", st.SHA
		c.note(ctx, deliveryID, "merged", "merge confirmado: "+st.SHA)
		if _, err := c.DB.Exec(ctx, `UPDATE deliveries SET state = 'merged', merge_sha = $2, merged_at = COALESCE(merged_at, $3), projection = CASE WHEN projection = 'done' THEN 'done' ELSE 'pending' END, updated_at = $3 WHERE id = $1`,
			deliveryID, st.SHA, c.now()); err != nil {
			return obs, persistence.MapError(err)
		}
		d.mergeSHA = st.SHA
		return c.finishMerged(ctx, d, deliveryID, keyBase, itemID, st.SHA, obs)
	case st.Merged:
		obs.Outcome, obs.Detail = "unknown", "a PR consta como integrada, mas sem o SHA do merge: sem prova, nada é liberado"
		c.note(ctx, deliveryID, "unknown", obs.Detail)
	case st.Closed:
		obs.Outcome, obs.Detail = "closed", "a PR foi fechada sem merge: nada é liberado"
		c.note(ctx, deliveryID, "closed", obs.Detail)
		if _, err := c.DB.Exec(ctx, `UPDATE deliveries SET state = 'closed', closed_at = COALESCE(closed_at, $2), updated_at = $2 WHERE id = $1`, deliveryID, c.now()); err != nil {
			return obs, persistence.MapError(err)
		}
	default:
		obs.Outcome, obs.Detail = "open", "a PR segue aberta: nada é liberado"
		c.note(ctx, deliveryID, "open", obs.Detail)
	}
	return obs, nil
}

// finishMerged projeta o status na origem (se faltar) e libera quem esperava o merge.
func (c *Completer) finishMerged(ctx context.Context, d deliveryRow, deliveryID, keyBase, itemID, sha string, obs Observation) (Observation, error) {
	obs.Outcome, obs.MergeSHA = "merged", sha
	// 1. Liberação dos dependentes: o merge está provado e registrado.
	if c.Queue != nil {
		rel, err := c.releaseWaiting(ctx, d)
		if err != nil {
			return obs, err
		}
		obs.Released = rel
	}
	// 2. Projeção do status (ledger: repetir não duplica).
	if d.projection != ProjectionDone && c.Origin != nil && itemID != "" {
		_, err := c.Origin.Complete(ctx, d.project, keyBase, itemID, d.prURL)
		switch {
		case err == nil:
			obs.Projection, obs.ProjectionError = ProjectionDone, ""
		default:
			obs.Projection, obs.ProjectionError = ProjectionFailed, truncate(err.Error(), 480)
			if errors.Is(err, effects.ErrBlocked) {
				obs.ProjectionError = "a projeção do status ficou inconclusiva e a repetição automática está bloqueada: " + truncate(err.Error(), 400)
			}
		}
		if _, e := c.DB.Exec(ctx, `UPDATE deliveries SET projection = $2, projection_error = $3, updated_at = $4 WHERE id = $1`, deliveryID, obs.Projection, obs.ProjectionError, c.now()); e != nil {
			return obs, persistence.MapError(e)
		}
	}
	return obs, nil
}

// releaseWaiting devolve à fila os itens do projeto que esperam o merge desta PR.
func (c *Completer) releaseWaiting(ctx context.Context, d deliveryRow) ([]queue.Item, error) {
	rows, err := c.DB.Query(ctx, `SELECT id::text FROM queue_items WHERE project_id = $1 AND state = 'waiting' AND wait_reason = 'merge' AND wait_ref = $2 ORDER BY seq`, d.project, d.prURL)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, persistence.MapError(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	out := []queue.Item{}
	for _, id := range ids {
		it, err := c.Queue.Resume(ctx, id)
		if err != nil {
			return out, fmt.Errorf("liberar o item %s: %w", id, err)
		}
		out = append(out, it)
	}
	return out, nil
}

// Reconcile tenta de novo a projeção do status das entregas já integradas cuja projeção falhou
// ou ficou pendente. Idempotente: o ledger não duplica a conclusão.
func (c *Completer) Reconcile(ctx context.Context, deliveryID, keyBase, itemID string) (Observation, error) {
	d, err := c.load(ctx, deliveryID)
	if err != nil {
		return Observation{}, err
	}
	if d.state != "merged" {
		return Observation{DeliveryID: deliveryID, Outcome: d.state, Projection: d.projection, Released: []queue.Item{}}, &FieldError{"delivery", "só uma entrega integrada tem status a reconciliar"}
	}
	return c.Observe(ctx, deliveryID, keyBase, itemID)
}

// Status devolve o que o painel mostra: estado da PR, merge e projeção na origem.
func (c *Completer) Status(ctx context.Context, deliveryID string) (Observation, error) {
	var o Observation
	o.DeliveryID, o.Released = deliveryID, []queue.Item{}
	var state string
	if err := c.DB.QueryRow(ctx, `SELECT state, merge_sha, projection, projection_error FROM deliveries WHERE id::text = $1`, deliveryID).Scan(&state, &o.MergeSHA, &o.Projection, &o.ProjectionError); err != nil {
		return o, persistence.MapError(err)
	}
	o.Outcome = map[string]string{"merged": "merged", "closed": "closed", "draft_open": "open"}[state]
	if o.Outcome == "" {
		o.Outcome = "unknown"
	}
	return o, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
