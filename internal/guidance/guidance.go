// Package guidance entrega orientações do administrador a uma execução e acompanha a confirmação técnica
// (AJ07-03). A entrega segue a capacidade do executor: imediata quando há interação segura, enfileirada
// para o próximo ponto seguro, ou só registrada para a retomada. O estado reflete o que foi COMPROVADO:
// `acknowledged` só com confirmação do cliente; envio sem confirmação fica `sent`; falha depois do envio fica
// `uncertain` (entrega incerta, a reconciliar); execução encerrada antes da entrega fica `undeliverable`.
// A orientação é só texto para o agente: não amplia permissões, não substitui aprovação e não altera o
// contrato publicado, e uma orientação já confirmada não é reenviada na retomada.
package guidance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Estados.
const (
	Queued        = "queued"
	Sent          = "sent"
	Acknowledged  = "acknowledged"
	Uncertain     = "uncertain"
	Undeliverable = "undeliverable"
)

// Erros.
var (
	ErrRejected = errors.New("orientação recusada")
	ErrEnded    = errors.New("a execução já terminou")
)

// Confirmation é o que o executor comprova.
type Confirmation int

// Confirmações.
const (
	// Unconfirmed: foi enviada, mas o cliente não confirma recebimento.
	Unconfirmed Confirmation = iota
	// Confirmed: o cliente reconheceu a orientação.
	Confirmed
)

// Deliverer entrega a mensagem ao executor da execução. Erro significa que o resultado é desconhecido
// (o envio pode ou não ter chegado).
type Deliverer interface {
	Deliver(ctx context.Context, executionID, text string) (Confirmation, error)
}

// DB é o que o serviço precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Service gerencia as orientações.
type Service struct {
	DB  DB
	Now func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// Item é uma orientação.
type Item struct {
	ID     string    `json:"id"`
	Seq    int       `json:"seq"`
	Author string    `json:"author"`
	Body   string    `json:"body"`
	Mode   string    `json:"mode"`
	State  string    `json:"state"`
	Detail string    `json:"detail,omitempty"`
	At     time.Time `json:"created_at"`
}

const maxBody = 4000

// Validate recusa o que não é orientação: vazio, longo, com controle, ou que imita o cabeçalho do contrato
// ou um comando da fábrica.
func Validate(body string) error {
	t := strings.TrimSpace(body)
	switch {
	case t == "":
		return fmt.Errorf("%w: texto vazio", ErrRejected)
	case len(t) > maxBody:
		return fmt.Errorf("%w: texto longo demais", ErrRejected)
	}
	for _, r := range t {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return fmt.Errorf("%w: caracteres de controle", ErrRejected)
		}
	}
	for _, line := range strings.Split(t, "\n") {
		l := strings.TrimSpace(strings.ToLower(line))
		if strings.HasPrefix(l, "[contrato ") || strings.HasPrefix(l, "[skill ") || strings.HasPrefix(l, "/approve") || strings.HasPrefix(l, "/fabrique") {
			return fmt.Errorf("%w: o texto imita o cabeçalho do contrato ou um comando da fábrica", ErrRejected)
		}
	}
	return nil
}

// Envelope é o que o agente recebe: marca a origem e o limite da orientação.
func Envelope(author, body string) string {
	return "[orientação de " + author + " — não altera o contrato, as permissões nem as aprovações]\n" + strings.TrimSpace(body)
}

// Send registra a orientação (autoria gravada) e a entrega conforme a capacidade. Devolve o item no
// estado resultante.
func (s *Service) Send(ctx context.Context, d Deliverer, caps executor.Capabilities, projectID, executionID, author, body string, ended bool) (Item, error) {
	if strings.TrimSpace(author) == "" {
		return Item{}, fmt.Errorf("%w: autoria obrigatória", ErrRejected)
	}
	if err := Validate(body); err != nil {
		return Item{}, err
	}
	if ended {
		return Item{}, ErrEnded
	}
	mode := string(caps.Interaction)
	var it Item
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT 1 FROM executions WHERE id = $1 AND project_id = $2 FOR UPDATE`, executionID, projectID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `INSERT INTO agent_guidance (project_id, execution_id, seq, author, body, mode)
			SELECT $1,$2, COALESCE(MAX(seq),0)+1, $3, $4, $5 FROM agent_guidance WHERE execution_id = $2
			RETURNING id::text, seq, author, body, mode, state, detail, created_at`, projectID, executionID, author, strings.TrimSpace(body), mode).
			Scan(&it.ID, &it.Seq, &it.Author, &it.Body, &it.Mode, &it.State, &it.Detail, &it.At)
	})
	if err != nil {
		return Item{}, persistence.MapError(err)
	}
	if caps.Interaction == executor.InteractionImmediate {
		return s.deliver(ctx, d, executionID, it)
	}
	return it, nil
}

func (s *Service) deliver(ctx context.Context, d Deliverer, executionID string, it Item) (Item, error) {
	conf, err := d.Deliver(ctx, executionID, Envelope(it.Author, it.Body))
	state, detail := Sent, "enviada; o cliente não confirma o recebimento"
	switch {
	case err != nil:
		state, detail = Uncertain, "falha depois do envio: a entrega é incerta e precisa ser reconciliada"
	case conf == Confirmed:
		state, detail = Acknowledged, "confirmada pelo cliente"
	}
	ack := any(nil)
	if state == Acknowledged {
		ack = s.now()
	}
	if _, e := s.DB.Exec(ctx, `UPDATE agent_guidance SET state=$2, detail=$3, sent_at=$4, acked_at=$5 WHERE id::text=$1 AND state IN ('queued','uncertain')`, it.ID, state, detail, s.now(), ack); e != nil {
		return it, persistence.MapError(e)
	}
	it.State, it.Detail = state, detail
	return it, nil
}

// SafePoint entrega, em ordem, as orientações enfileiradas (o executor chama no próximo ponto seguro).
func (s *Service) SafePoint(ctx context.Context, d Deliverer, projectID, executionID string) ([]Item, error) {
	q, err := s.List(ctx, projectID, executionID)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, it := range q {
		if it.State != Queued || it.Mode != string(executor.InteractionNextPoint) {
			continue
		}
		got, err := s.deliver(ctx, d, executionID, it)
		if err != nil {
			return out, err
		}
		out = append(out, got)
	}
	return out, nil
}

// List devolve o histórico da execução em ordem.
func (s *Service) List(ctx context.Context, projectID, executionID string) ([]Item, error) {
	rows, err := s.DB.Query(ctx, `SELECT id::text, seq, author, body, mode, state, detail, created_at FROM agent_guidance WHERE project_id = $1 AND execution_id = $2 ORDER BY seq`, projectID, executionID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Seq, &it.Author, &it.Body, &it.Mode, &it.State, &it.Detail, &it.At); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// Ended marca como não entregues as orientações que ainda esperavam quando a execução terminou, expondo o
// resultado e preservando o histórico para a retomada ou a decisão do administrador.
func (s *Service) Ended(ctx context.Context, projectID, executionID string) (int64, error) {
	tag, err := s.DB.Exec(ctx, `UPDATE agent_guidance SET state='undeliverable', detail='a execução terminou antes da entrega: preservada para a retomada ou decisão do administrador'
		WHERE project_id=$1 AND execution_id=$2 AND state='queued'`, projectID, executionID)
	return tag.RowsAffected(), persistence.MapError(err)
}

// Reconcile resolve uma entrega incerta com a prova obtida (leitura do estado do agente ou decisão humana).
func (s *Service) Reconcile(ctx context.Context, projectID, executionID, id string, delivered bool) error {
	state, detail := Queued, "reconciliada: não foi entregue; volta à fila"
	if delivered {
		state, detail = Acknowledged, "reconciliada: entrega comprovada"
	}
	tag, err := s.DB.Exec(ctx, `UPDATE agent_guidance SET state=$4, detail=$5, acked_at = CASE WHEN $4='acknowledged' THEN $6::timestamptz END
		WHERE id::text=$3 AND project_id=$1 AND execution_id=$2 AND state IN ('uncertain','sent')`, projectID, executionID, id, state, detail, s.now())
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return persistence.ErrNotFound
	}
	return nil
}

// ForResume devolve o que a retomada precisa saber: o que ainda precisa ser entregue (enfileirada,
// não entregue ou incerta, esta marcada) e o histórico já confirmado, que NÃO é reenviado.
func (s *Service) ForResume(ctx context.Context, projectID, executionID string) (pending []Item, confirmed []Item, err error) {
	all, err := s.List(ctx, projectID, executionID)
	if err != nil {
		return nil, nil, err
	}
	for _, it := range all {
		switch it.State {
		case Acknowledged:
			confirmed = append(confirmed, it)
		case Sent:
			// enviada sem confirmação possível: não se reenvia sem prova; fica no histórico
			confirmed = append(confirmed, it)
		default:
			pending = append(pending, it)
		}
	}
	return pending, confirmed, nil
}
