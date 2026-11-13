// Package activation prepara, ativa e reverte com segurança a troca do despachante do projeto (AJ03-05).
// A prévia é só leitura e lista o que bloqueia; a ativação exige workflow do projeto publicado e válido,
// mapa de status e um ponto seguro (nenhuma execução do despachante anterior em andamento), normaliza o
// alias legado REPROVED_PR sem perder o valor bruto, rastreia as Issues conhecidas e é repetível; só UM
// despachante admite por projeto (nenhum trabalho adquire duas reservas). A reversão drena a Sentinela,
// desabilita novas admissões, lista efeitos externos a reconciliar e só devolve ao legado depois disso,
// preservando todos os dados e efeitos já aplicados.
package activation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projectres"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Despachantes.
const (
	Legacy   = "legacy"
	Sentinel = "sentinel"
	None     = "none"
)

// Erros.
var (
	ErrBlocked    = errors.New("a ativação está bloqueada: resolva as pendências da prévia")
	ErrNotSafe    = errors.New("não é um ponto seguro: há execuções do despachante anterior em andamento")
	ErrNotDrained = errors.New("ainda há trabalho em execução ou efeitos externos a reconciliar")
)

// DB é o que o serviço precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Service gerencia a ativação.
type Service struct {
	DB DB
}

// Preview é o relatório da prévia (somente leitura).
type Preview struct {
	Blocking         []string `json:"blocking"`
	LegacyRunning    []string `json:"legacy_running"` // itens do despachante anterior em execução
	AliasToNormalize int      `json:"alias_to_normalize"`
	KnownIssues      int      `json:"known_issues"`
	Untracked        int      `json:"untracked"`         // Issues conhecidas ainda não rastreadas pela Sentinela
	ExternalSubtasks int      `json:"external_subtasks"` // subtarefas com ID remoto (legado), preservadas
	WorkflowReady    bool     `json:"workflow_ready"`
	Safe             bool     `json:"safe_point"`
}

// Preview inspeciona sem alterar nada.
func (s *Service) Preview(ctx context.Context, projectID string) (Preview, error) {
	pv := Preview{Blocking: []string{}, LegacyRunning: []string{}}
	rd, err := projectres.Ready(ctx, s.DB, projectID)
	if err != nil {
		return pv, err
	}
	pv.WorkflowReady = rd.Ready
	if !rd.Ready {
		for _, p := range rd.Problems {
			pv.Blocking = append(pv.Blocking, "workflow/projeto: "+p)
		}
	}
	var m int
	if err := s.DB.QueryRow(ctx, `SELECT count(*) FROM project_status_maps WHERE project_id = $1`, projectID).Scan(&m); err != nil {
		return pv, persistence.MapError(err)
	}
	if m == 0 {
		pv.Blocking = append(pv.Blocking, "mapa de status do projeto ainda não configurado")
	}
	rows, err := s.DB.Query(ctx, `SELECT task_key FROM queue_items WHERE project_id = $1 AND state = 'running' ORDER BY seq`, projectID)
	if err != nil {
		return pv, persistence.MapError(err)
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return pv, err
		}
		pv.LegacyRunning = append(pv.LegacyRunning, k)
	}
	rows.Close()
	pv.Safe = len(pv.LegacyRunning) == 0
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM issue_registry WHERE project_id = $1 AND upper(status_code) = $2`, projectID, flow.LegacyReprovedPR).Scan(&pv.AliasToNormalize)
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM issue_registry WHERE project_id = $1`, projectID).Scan(&pv.KnownIssues)
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM issue_registry r WHERE r.project_id = $1 AND NOT EXISTS (SELECT 1 FROM flow_issues i WHERE i.project_id = r.project_id AND i.external_key = r.issue_key)`, projectID).Scan(&pv.Untracked)
	_ = s.DB.QueryRow(ctx, `SELECT count(*) FROM plan_tasks t JOIN features f ON f.id = t.feature_id WHERE f.project_id = $1 AND t.remote_id <> ''`, projectID).Scan(&pv.ExternalSubtasks)
	return pv, nil
}

// State é o estado atual.
type State struct {
	State      string `json:"state"`
	Dispatcher string `json:"dispatcher"`
}

// Get devolve o estado (legado quando nunca ativado).
func (s *Service) Get(ctx context.Context, projectID string) (State, error) {
	st := State{State: "legacy", Dispatcher: Legacy}
	err := s.DB.QueryRow(ctx, `SELECT state, dispatcher FROM flow_activation WHERE project_id = $1`, projectID).Scan(&st.State, &st.Dispatcher)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return st, persistence.MapError(err)
	}
	return st, nil
}

// CanAdmit informa se o despachante pode admitir trabalho no projeto: só o ativo admite.
func (s *Service) CanAdmit(ctx context.Context, projectID, who string) (bool, error) {
	st, err := s.Get(ctx, projectID)
	return st.Dispatcher == who, err
}

// Activate ativa a Sentinela como despachante. Repetível; recusa fora de ponto seguro ou com pendências.
func (s *Service) Activate(ctx context.Context, projectID string, c *sentinel.Coordinator) (Preview, error) {
	pv, err := s.Preview(ctx, projectID)
	if err != nil {
		return pv, err
	}
	cur, err := s.Get(ctx, projectID)
	if err != nil {
		return pv, err
	}
	if cur.State == "active" {
		return pv, nil // repetição: nada a fazer
	}
	if len(pv.Blocking) > 0 {
		return pv, fmt.Errorf("%w: %v", ErrBlocked, pv.Blocking)
	}
	if !pv.Safe {
		return pv, fmt.Errorf("%w: %v", ErrNotSafe, pv.LegacyRunning)
	}
	// Rastreia as Issues conhecidas (idempotente) com o status canônico (alias normalizado na leitura).
	rows, err := s.DB.Query(ctx, `SELECT issue_key, kind, status_code FROM issue_registry WHERE project_id = $1 ORDER BY issue_key`, projectID)
	if err != nil {
		return pv, persistence.MapError(err)
	}
	type k struct{ key, kind, code string }
	var ks []k
	for rows.Next() {
		var x k
		if err := rows.Scan(&x.key, &x.kind, &x.code); err != nil {
			rows.Close()
			return pv, err
		}
		ks = append(ks, x)
	}
	rows.Close()
	for _, x := range ks {
		if _, ok := flow.Normalize(x.code); !ok {
			continue // status sem mapa: o sync trata; não bloqueia a ativação
		}
		if _, err := c.Track(ctx, sentinel.NewIssue{ProjectID: projectID, Key: x.key, Kind: x.kind, Status: x.code}); err != nil {
			return pv, err
		}
	}
	pj, _ := json.Marshal(pv)
	err = persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		// O alias legado converge; o valor bruto observado fica em raw_status (preservado).
		if _, err := tx.Exec(ctx, `UPDATE issue_registry SET status_code = $2 WHERE project_id = $1 AND upper(status_code) = $3`, projectID, flow.ReprovadoPR, flow.LegacyReprovedPR); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO flow_activation (project_id, state, dispatcher, preview, activated_at, updated_at) VALUES ($1,'active','sentinel',$2,now(),now())
			ON CONFLICT (project_id) DO UPDATE SET state='active', dispatcher='sentinel', preview=EXCLUDED.preview, activated_at=now(), rollback_at=NULL, updated_at=now()`, projectID, string(pj))
		return err
	})
	return pv, persistence.MapError(err)
}

// Drain é o estado da drenagem na reversão.
type Drain struct {
	Running     []string `json:"running"`      // trabalho da Sentinela ainda em execução
	ToReconcile []string `json:"to_reconcile"` // efeitos externos ainda em envio ou incertos
	Done        bool     `json:"done"`
}

// Rollback desabilita novas admissões (despachante `none`) e começa a drenar. Nada é apagado.
func (s *Service) Rollback(ctx context.Context, projectID string) (Drain, error) {
	_, err := s.DB.Exec(ctx, `INSERT INTO flow_activation (project_id, state, dispatcher, rollback_at, updated_at) VALUES ($1,'draining','none',now(),now())
		ON CONFLICT (project_id) DO UPDATE SET state = CASE WHEN flow_activation.state = 'rolled_back' THEN 'rolled_back' ELSE 'draining' END,
		dispatcher = CASE WHEN flow_activation.state = 'rolled_back' THEN flow_activation.dispatcher ELSE 'none' END, rollback_at = COALESCE(flow_activation.rollback_at, now()), updated_at = now()`, projectID)
	if err != nil {
		return Drain{}, persistence.MapError(err)
	}
	return s.Draining(ctx, projectID)
}

// Draining informa o que ainda impede devolver o projeto ao despachante anterior.
func (s *Service) Draining(ctx context.Context, projectID string) (Drain, error) {
	d := Drain{Running: []string{}, ToReconcile: []string{}}
	rows, err := s.DB.Query(ctx, `SELECT feature_id::text || '/' || task_key FROM subtasks WHERE project_id = $1 AND (impl_state = 'running' OR review_state = 'running') ORDER BY feature_id, task_key`, projectID)
	if err != nil {
		return d, persistence.MapError(err)
	}
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			rows.Close()
			return d, err
		}
		d.Running = append(d.Running, k)
	}
	rows.Close()
	er, err := s.DB.Query(ctx, `SELECT business_key || ' (' || state || ')' FROM outbox WHERE project_id = $1 AND state IN ('sending', 'uncertain') ORDER BY created_at, business_key`, projectID)
	if err != nil {
		return d, persistence.MapError(err)
	}
	for er.Next() {
		var k string
		if err := er.Scan(&k); err != nil {
			er.Close()
			return d, err
		}
		d.ToReconcile = append(d.ToReconcile, k)
	}
	er.Close()
	d.Done = len(d.Running) == 0 && len(d.ToReconcile) == 0
	return d, nil
}

// FinishRollback devolve o projeto ao despachante anterior, só com a drenagem concluída e os efeitos
// reconciliados. Os dados e o histórico permanecem.
func (s *Service) FinishRollback(ctx context.Context, projectID string) error {
	st, err := s.Get(ctx, projectID)
	if err != nil {
		return err
	}
	if st.State == "rolled_back" {
		return nil
	}
	if st.State != "draining" {
		return fmt.Errorf("%w: a reversão não foi iniciada", persistence.ErrInvalidInput)
	}
	d, err := s.Draining(ctx, projectID)
	if err != nil {
		return err
	}
	if !d.Done {
		return fmt.Errorf("%w: execuções %v; efeitos %v", ErrNotDrained, d.Running, d.ToReconcile)
	}
	_, err = s.DB.Exec(ctx, `UPDATE flow_activation SET state='rolled_back', dispatcher='legacy', updated_at=now() WHERE project_id = $1`, projectID)
	return persistence.MapError(err)
}
