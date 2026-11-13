package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// CommentSource lê os comentários da PR (gitops.PRClient implementa).
type CommentSource interface {
	Comments(ctx context.Context, prURL string) ([]gitops.Comment, error)
}

// PRReopener reabre a PR (gitops.PRClient implementa).
type PRReopener interface {
	Reopen(ctx context.Context, prURL string) error
}

// Ações oferecidas quando a PR é fechada sem merge.
const (
	ActionCancel = "cancel"
	ActionReopen = "reopen"
	ActionRetry  = "retry"
)

// DefaultMaxAttempts limita as tentativas de entrega de uma feature.
const DefaultMaxAttempts = 3

// External trata a revisão externa e a PR fechada. Nada roda sozinho: tudo é pedido.
type External struct {
	DB          persistenceDB
	Comments    CommentSource
	Reopener    PRReopener
	Merges      tasksync.MergeVerifier
	Queue       *queue.Queue
	Redactor    *security.Redactor
	Authorized  []string // logins autorizados a pedir correção; vazio = ninguém
	MaxAttempts int
	Now         func() time.Time
}

type persistenceDB interface {
	persistence.DBTX
	persistence.TxStarter
}

func (e *External) now() time.Time {
	if e.Now != nil {
		return e.Now().UTC()
	}
	return time.Now().UTC()
}

func (e *External) authorized(login string) bool {
	for _, a := range e.Authorized {
		if strings.EqualFold(strings.TrimSpace(a), login) && login != "" {
			return true
		}
	}
	return false
}

// Discovery é o resultado de uma descoberta de comentários.
type Discovery struct {
	DeliveryID string      `json:"delivery_id"`
	New        int         `json:"new"`     // comentários registrados agora
	Ignored    int         `json:"ignored"` // de autores não autorizados ou vazios
	Held       int         `json:"held"`    // registrados, mas sem trabalho (PR não aberta)
	Item       *queue.Item `json:"item,omitempty"`
	Pending    []PRComment `json:"pending"`
}

// PRComment é um comentário registrado.
type PRComment struct {
	ID          int64      `json:"id"`
	CommentID   int64      `json:"comment_id"`
	Author      string     `json:"author"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	QueueItemID string     `json:"queue_item_id,omitempty"`
	HandledAt   *time.Time `json:"handled_at,omitempty"`
}

func (e *External) comments(ctx context.Context, deliveryID, where string) ([]PRComment, error) {
	rows, err := e.DB.Query(ctx, `SELECT id, comment_id, author, body, state, COALESCE(queue_item_id::text, ''), handled_at FROM pr_comments WHERE delivery_id::text = $1 `+where+` ORDER BY id`, deliveryID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []PRComment{}
	for rows.Next() {
		var c PRComment
		if err := rows.Scan(&c.ID, &c.CommentID, &c.Author, &c.Body, &c.State, &c.QueueItemID, &c.HandledAt); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, c)
	}
	return out, persistence.MapError(rows.Err())
}

// List lista os comentários registrados da entrega.
func (e *External) List(ctx context.Context, deliveryID string) ([]PRComment, error) {
	return e.comments(ctx, deliveryID, "")
}

func (e *External) load(ctx context.Context, id string) (deliveryRow, error) {
	return (&Completer{DB: e.DB}).load(ctx, id)
}

// Discover consulta a PR (só quando chamado) e registra os comentários de autores autorizados.
// O mesmo comentário (mesmo identificador e conteúdo) nunca vira um segundo trabalho; um
// comentário editado substitui o anterior ainda não tratado. Com a PR aberta, os comentários
// pendentes viram UM item de correção na fila (vai à frente na próxima vaga elegível). Com a PR
// fechada, ficam retidos até a decisão humana.
func (e *External) Discover(ctx context.Context, deliveryID, taskKey string, priority int) (Discovery, error) {
	out := Discovery{DeliveryID: deliveryID, Pending: []PRComment{}}
	d, err := e.load(ctx, deliveryID)
	if err != nil {
		return out, err
	}
	switch d.state {
	case "draft_open", "closed", "cancelled":
	default:
		return out, &FieldError{"delivery", "só uma PR aberta, fechada ou cancelada recebe comentários a tratar"}
	}
	if d.prURL == "" {
		return out, &FieldError{"delivery", "a entrega não tem PR"}
	}
	list, err := e.Comments.Comments(ctx, d.prURL)
	if err != nil {
		return out, fmt.Errorf("não foi possível ler os comentários da PR: %w", err) // nada muda
	}
	state := "queued"
	if d.state != "draft_open" {
		state = "held"
	}
	for _, c := range list {
		body := strings.TrimSpace(c.Body)
		if !e.authorized(c.Author) || body == "" || c.ID < 1 {
			out.Ignored++
			continue
		}
		if e.Redactor != nil {
			body = e.Redactor.Redact(body)
		}
		if len(body) > 4000 {
			body = body[:4000]
		}
		hash := sha256Hex(body)
		var added bool
		err := persistence.WithTx(ctx, e.DB, func(tx pgx.Tx) error {
			var one int64
			err := tx.QueryRow(ctx, `INSERT INTO pr_comments (delivery_id, comment_id, author, body, body_hash, state) VALUES ($1, $2, $3, $4, $5, $6)
				ON CONFLICT (delivery_id, comment_id, body_hash) DO NOTHING RETURNING id`, deliveryID, c.ID, c.Author, body, hash, state).Scan(&one)
			if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
				return nil // já registrado
			}
			if err != nil {
				return persistence.MapError(err)
			}
			added = true
			// Edição: a versão anterior ainda não tratada deixa de valer.
			_, err = tx.Exec(ctx, `UPDATE pr_comments SET state = 'dismissed', handled_at = $4 WHERE delivery_id = $1 AND comment_id = $2 AND id <> $3 AND state IN ('queued', 'held')`, deliveryID, c.ID, one, e.now())
			return persistence.MapError(err)
		})
		if err != nil {
			return out, err
		}
		if added {
			out.New++
			if state == "held" {
				out.Held++
			}
		}
	}
	if d.state == "draft_open" {
		it, err := e.enqueueFix(ctx, d, deliveryID, taskKey, priority)
		if err != nil {
			return out, err
		}
		out.Item = it
	}
	out.Pending, err = e.comments(ctx, deliveryID, `AND state IN ('queued', 'held')`)
	return out, err
}

// enqueueFix cria (ou reencontra) o item de correção da PR e liga a ele os comentários sem item.
func (e *External) enqueueFix(ctx context.Context, d deliveryRow, deliveryID, taskKey string, priority int) (*queue.Item, error) {
	var waiting int
	if err := e.DB.QueryRow(ctx, `SELECT count(*) FROM pr_comments WHERE delivery_id::text = $1 AND state = 'queued' AND queue_item_id IS NULL`, deliveryID).Scan(&waiting); err != nil {
		return nil, persistence.MapError(err)
	}
	if waiting == 0 || e.Queue == nil {
		return nil, nil
	}
	it, err := e.Queue.EnqueuePRFix(ctx, d.project, d.feature, taskKey, d.prURL, priority)
	if err != nil {
		return nil, err
	}
	if _, err := e.DB.Exec(ctx, `UPDATE pr_comments SET queue_item_id = $2 WHERE delivery_id::text = $1 AND state = 'queued' AND queue_item_id IS NULL`, deliveryID, it.ID); err != nil {
		return nil, persistence.MapError(err)
	}
	return &it, nil
}

// FixProof é a prova de que a correção atualizou documentação e checks.
type FixProof struct {
	DocsUpdated  bool
	ChecksPassed bool
}

// MarkFixed dá os comentários do item como corrigidos. Exige documentação atualizada e checks
// aprovados na nova revisão; sem isso nada é marcado. Repetir não muda nada.
func (e *External) MarkFixed(ctx context.Context, deliveryID, itemID string, proof FixProof) (int, error) {
	if !proof.DocsUpdated || !proof.ChecksPassed {
		return 0, &FieldError{"proof", "a correção só conta com a documentação atualizada e os checks aprovados"}
	}
	tag, err := e.DB.Exec(ctx, `UPDATE pr_comments SET state = 'fixed', handled_at = $3 WHERE delivery_id::text = $1 AND queue_item_id::text = $2 AND state = 'queued'`, deliveryID, itemID, e.now())
	if err != nil {
		return 0, persistence.MapError(err)
	}
	return int(tag.RowsAffected()), nil
}

// ActionOption é uma ação oferecida.
type ActionOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// Offer é o que o painel mostra para uma PR fechada sem merge.
type Offer struct {
	DeliveryID  string         `json:"delivery_id"`
	State       string         `json:"state"`
	Attempt     int            `json:"attempt"`
	MaxAttempts int            `json:"max_attempts"`
	Options     []ActionOption `json:"options"`
}

func (e *External) max() int {
	if e.MaxAttempts > 0 {
		return e.MaxAttempts
	}
	return DefaultMaxAttempts
}

// Offer oferece as três ações para a PR fechada sem merge (registra a oferta uma vez).
func (e *External) Offer(ctx context.Context, deliveryID string) (Offer, error) {
	d, err := e.load(ctx, deliveryID)
	if err != nil {
		return Offer{}, err
	}
	if d.state != "closed" && d.state != "cancelled" {
		return Offer{}, &FieldError{"delivery", "só uma PR fechada sem merge tem ações a decidir"}
	}
	if _, err := e.DB.Exec(ctx, `INSERT INTO delivery_actions (delivery_id, action, actor, note)
		SELECT $1::uuid, 'offered', '', '' WHERE NOT EXISTS (SELECT 1 FROM delivery_actions WHERE delivery_id::text = $2 AND action = 'offered')`, deliveryID, deliveryID); err != nil {
		return Offer{}, persistence.MapError(err)
	}
	return Offer{DeliveryID: deliveryID, State: d.state, Attempt: d.attempt, MaxAttempts: e.max(), Options: []ActionOption{
		{ActionCancel, "Cancelar", "encerra a entrega; PR, commits, documentação e comentários ficam guardados"},
		{ActionReopen, "Reabrir", "reabre a mesma PR e retoma os comentários retidos"},
		{ActionRetry, "Nova tentativa", "libera uma nova entrega da feature em outra branch, sem apagar a anterior"},
	}}, nil
}

// Decision é o resultado de uma ação sobre a PR fechada.
type Decision struct {
	DeliveryID  string `json:"delivery_id"`
	Action      string `json:"action"`
	State       string `json:"state"`
	NextAttempt int    `json:"next_attempt,omitempty"`
	Replayed    bool   `json:"replayed"`
}

// Decide aplica a ação escolhida pelo responsável sobre a PR fechada ou cancelada:
//   - cancel: a entrega fica `cancelled`, os itens de correção ativos são cancelados e os
//     comentários ficam retidos; nada é apagado;
//   - reopen: a PR é reaberta no GitHub, a abertura é conferida e só então a entrega volta a
//     `draft_open` e os comentários retidos voltam a ser tratáveis;
//   - retry: autoriza a próxima tentativa de entrega (limite de tentativas), sem tocar na anterior.
//
// Repetir a mesma ação não repete efeito. Todas ficam no histórico com quem decidiu.
func (e *External) Decide(ctx context.Context, deliveryID, action, actor, note string) (Decision, error) {
	if action != ActionCancel && action != ActionReopen && action != ActionRetry {
		return Decision{}, &FieldError{"action", "ação desconhecida: use cancel, reopen ou retry"}
	}
	actor, note = strings.TrimSpace(actor), strings.TrimSpace(note)
	if len(actor) > 100 || len(note) > 500 {
		return Decision{}, &FieldError{"note", "texto longo demais"}
	}
	d, err := e.load(ctx, deliveryID)
	if err != nil {
		return Decision{}, err
	}
	dec := Decision{DeliveryID: deliveryID, Action: action, State: d.state}
	var last string
	err = e.DB.QueryRow(ctx, `SELECT action FROM delivery_actions WHERE delivery_id::text = $1 AND action <> 'offered' ORDER BY id DESC LIMIT 1`, deliveryID).Scan(&last)
	if err != nil && !errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return dec, persistence.MapError(err)
	}
	consistent := map[string]string{ActionCancel: "cancelled", ActionReopen: "draft_open", ActionRetry: d.state}
	if last == action && d.state == consistent[action] {
		dec.Replayed = true
		if action == ActionRetry {
			dec.NextAttempt = d.attempt + 1
		}
		return dec, nil
	}
	if d.state != "closed" && d.state != "cancelled" {
		return dec, &FieldError{"delivery", "só uma PR fechada sem merge (ou entrega cancelada) aceita essa ação"}
	}
	switch action {
	case ActionCancel:
		if d.state == "cancelled" {
			dec.Replayed = true
			return dec, nil
		}
		if _, err := e.DB.Exec(ctx, `UPDATE deliveries SET state = 'cancelled', updated_at = $2 WHERE id::text = $1`, deliveryID, e.now()); err != nil {
			return dec, persistence.MapError(err)
		}
		dec.State = "cancelled"
		if err := e.cancelFixes(ctx, d, deliveryID); err != nil {
			return dec, err
		}
	case ActionReopen:
		if err := e.reopen(ctx, d, deliveryID); err != nil {
			return dec, err
		}
		dec.State = "draft_open"
	case ActionRetry:
		var maxAttempt int
		if err := e.DB.QueryRow(ctx, `SELECT COALESCE(max(attempt), 0) FROM deliveries WHERE feature_id::text = $1`, d.feature).Scan(&maxAttempt); err != nil {
			return dec, persistence.MapError(err)
		}
		if maxAttempt >= e.max() {
			return dec, &FieldError{"action", fmt.Sprintf("o limite de %d tentativas de entrega foi atingido", e.max())}
		}
		dec.NextAttempt = maxAttempt + 1
	}
	if _, err := e.DB.Exec(ctx, `INSERT INTO delivery_actions (delivery_id, action, actor, note) VALUES ($1, $2, $3, $4)`, deliveryID, action, actor, note); err != nil {
		return dec, persistence.MapError(err)
	}
	return dec, nil
}

func (e *External) cancelFixes(ctx context.Context, d deliveryRow, deliveryID string) error {
	if e.Queue == nil {
		return nil
	}
	rows, err := e.DB.Query(ctx, `SELECT id::text FROM queue_items WHERE project_id::text = $1 AND kind = 'pr_fix' AND ref = $2 AND state IN ('queued', 'running', 'waiting')`, d.project, d.prURL)
	if err != nil {
		return persistence.MapError(err)
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return persistence.MapError(err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		if _, err := e.Queue.Cancel(ctx, id); err != nil && !errors.Is(err, queue.ErrBadState) {
			return err
		}
	}
	_, err = e.DB.Exec(ctx, `UPDATE pr_comments SET state = 'held', queue_item_id = NULL WHERE delivery_id::text = $1 AND state = 'queued'`, deliveryID)
	return persistence.MapError(err)
}

func (e *External) reopen(ctx context.Context, d deliveryRow, deliveryID string) error {
	if e.Reopener == nil || e.Merges == nil {
		return &FieldError{"action", "reabrir exige o cliente do repositório configurado"}
	}
	if err := e.Reopener.Reopen(ctx, d.prURL); err != nil {
		return fmt.Errorf("não foi possível reabrir a PR (a entrega continua como estava): %w", err)
	}
	st, err := e.Merges.Merged(ctx, d.prURL)
	if err != nil {
		return fmt.Errorf("a PR foi reaberta, mas não foi possível conferir (repita a ação): %w", err)
	}
	if st.Merged || st.Closed {
		return &FieldError{"action", "o GitHub não confirmou a PR aberta: a entrega continua como estava"}
	}
	if _, err := e.DB.Exec(ctx, `UPDATE deliveries SET state = 'draft_open', closed_at = NULL, updated_at = $2 WHERE id::text = $1`, deliveryID, e.now()); err != nil {
		return persistence.MapError(err)
	}
	_, err = e.DB.Exec(ctx, `UPDATE pr_comments SET state = 'queued' WHERE delivery_id::text = $1 AND state = 'held'`, deliveryID)
	return persistence.MapError(err)
}

// History devolve o histórico de ações da entrega, da mais antiga à mais nova.
func (e *External) History(ctx context.Context, deliveryID string) ([]string, error) {
	rows, err := e.DB.Query(ctx, `SELECT action || CASE WHEN actor <> '' THEN ' por ' || actor ELSE '' END FROM delivery_actions WHERE delivery_id::text = $1 ORDER BY id`, deliveryID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, persistence.MapError(err)
		}
		out = append(out, s)
	}
	return out, persistence.MapError(rows.Err())
}
