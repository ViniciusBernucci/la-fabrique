package budget

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Fontes do valor registrado.
const (
	SourceProvider       = "provider_reported"
	SourceEstimated      = "estimated"
	SourceReconciliation = "reconciliation"
)

// Erros.
var (
	ErrNoBudget          = errors.New("não há orçamento global ativo: o gasto em API é recusado")
	ErrNoPrice           = errors.New("o modelo não tem preço cadastrado: não pode ser orçado")
	ErrReservationClosed = errors.New("a reserva já foi liberada")
	ErrSettled           = errors.New("a reserva já foi liquidada")
)

// ExceededError diz qual orçamento barrou a reserva.
type ExceededError struct {
	Scope     string // global | agent
	Period    string
	Limit     int64
	Used      int64 // gasto + reservas ativas
	Requested int64
}

func (e *ExceededError) Error() string {
	return fmt.Sprintf("orçamento %s (%s) esgotado: limite %d, em uso %d, pedido %d micro-USD", e.Scope, e.Period, e.Limit, e.Used, e.Requested)
}

// FieldError é um campo inválido com mensagem segura.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// DB é o que o Store precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Store guarda orçamentos, reservas e uso.
type Store struct {
	DB  DB
	Now func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

// window devolve [início, fim) do período que contém t (UTC).
func window(period string, t time.Time) (time.Time, time.Time) {
	t = t.UTC()
	switch period {
	case "day":
		st := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return st, st.AddDate(0, 0, 1)
	default:
		st := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		return st, st.AddDate(0, 1, 0)
	}
}

// Budget é um teto de gasto em API.
type Budget struct {
	ID          string    `json:"id"`
	Scope       string    `json:"scope"`
	AgentID     *string   `json:"agent_id"`
	Period      string    `json:"period"`
	LimitMicros int64     `json:"limit_micros"`
	State       string    `json:"state"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Status é um orçamento com o uso do período corrente.
type Status struct {
	Budget
	SpentMicros     int64 `json:"spent_micros"`
	ReservedMicros  int64 `json:"reserved_micros"`
	RemainingMicros int64 `json:"remaining_micros"`
}

// SetInput cria ou altera o orçamento de (escopo, agente, período).
type SetInput struct {
	Scope       string
	AgentID     string
	Period      string
	LimitMicros int64
	State       string // vazio = active
}

const budgetCols = `id, scope, agent_id, period, limit_micros, state, version, created_at, updated_at`

func scanBudget(row interface{ Scan(...any) error }) (Budget, error) {
	var b Budget
	err := row.Scan(&b.ID, &b.Scope, &b.AgentID, &b.Period, &b.LimitMicros, &b.State, &b.Version, &b.CreatedAt, &b.UpdatedAt)
	return b, persistence.MapError(err)
}

// Set cria ou atualiza (upsert) um orçamento.
func (s *Store) Set(ctx context.Context, in SetInput) (Budget, error) {
	if in.Scope != "global" && in.Scope != "agent" {
		return Budget{}, &FieldError{"scope", "Escopo inválido (global ou agent)."}
	}
	if (in.Scope == "global") != (in.AgentID == "") {
		return Budget{}, &FieldError{"agent_id", "O agente só vale no escopo agent (e é obrigatório nele)."}
	}
	if in.Period != "day" && in.Period != "month" {
		return Budget{}, &FieldError{"period", "Período inválido (day ou month)."}
	}
	if in.LimitMicros < 0 || in.LimitMicros > 1_000_000_000_000 {
		return Budget{}, &FieldError{"limit_micros", "Limite fora da faixa."}
	}
	state := in.State
	if state == "" {
		state = "active"
	}
	if state != "active" && state != "disabled" {
		return Budget{}, &FieldError{"state", "Estado inválido (active ou disabled)."}
	}
	var agent any
	if in.AgentID != "" {
		agent = in.AgentID
	}
	q := `INSERT INTO budgets (scope, agent_id, period, limit_micros, state) VALUES ($1, $2::uuid, $3, $4, $5)
		ON CONFLICT (period) WHERE scope = 'global' DO UPDATE SET limit_micros = EXCLUDED.limit_micros, state = EXCLUDED.state, version = budgets.version + 1, updated_at = now()
		RETURNING ` + budgetCols
	if in.Scope == "agent" {
		q = `INSERT INTO budgets (scope, agent_id, period, limit_micros, state) VALUES ($1, $2::uuid, $3, $4, $5)
		ON CONFLICT (agent_id, period) WHERE scope = 'agent' DO UPDATE SET limit_micros = EXCLUDED.limit_micros, state = EXCLUDED.state, version = budgets.version + 1, updated_at = now()
		RETURNING ` + budgetCols
	}
	return scanBudget(s.DB.QueryRow(ctx, q, in.Scope, agent, in.Period, in.LimitMicros, state))
}

// spentAndReserved calcula o uso de um orçamento no período corrente.
func (s *Store) spentAndReserved(ctx context.Context, q persistence.DBTX, b Budget, now time.Time) (spent, reserved int64, err error) {
	ws, we := window(b.Period, now)
	var agent any
	if b.AgentID != nil {
		agent = *b.AgentID
	}
	err = q.QueryRow(ctx, `SELECT COALESCE(SUM(cost_micros), 0) FROM usage_records
		WHERE kind = 'api_cost' AND observed_at >= $1 AND observed_at < $2 AND ($3::uuid IS NULL OR agent_id = $3::uuid)`, ws, we, agent).Scan(&spent)
	if err != nil {
		return 0, 0, persistence.MapError(err)
	}
	err = q.QueryRow(ctx, `SELECT COALESCE(SUM(amount_micros), 0) FROM usage_reservations
		WHERE state = 'reserved' AND expires_at > $1 AND ($2::uuid IS NULL OR agent_id = $2::uuid)`, now, agent).Scan(&reserved)
	return spent, reserved, persistence.MapError(err)
}

// List devolve os orçamentos com o uso do período corrente.
func (s *Store) List(ctx context.Context) ([]Status, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+budgetCols+` FROM budgets ORDER BY scope, period, id`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	var bs []Budget
	for rows.Next() {
		b, err := scanBudget(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		bs = append(bs, b)
	}
	rows.Close()
	now := s.now()
	out := make([]Status, 0, len(bs))
	for _, b := range bs {
		sp, rs, err := s.spentAndReserved(ctx, s.DB, b, now)
		if err != nil {
			return nil, err
		}
		out = append(out, Status{Budget: b, SpentMicros: sp, ReservedMicros: rs, RemainingMicros: max(0, b.LimitMicros-sp-rs)})
	}
	return out, nil
}

// ReserveRequest pede a reserva de custo antes de uma chamada paga.
type ReserveRequest struct {
	AgentID      string
	ConnectionID string
	AmountMicros int64
	TTL          time.Duration // padrão 5 min, no máximo 1 h
}

// Reservation é uma reserva ativa.
type Reservation struct {
	ID           string    `json:"id"`
	AgentID      string    `json:"agent_id"`
	ConnectionID string    `json:"connection_id"`
	AmountMicros int64     `json:"amount_micros"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// Reserve reserva custo de forma atômica contra o orçamento global e o do agente
// (quando houver). Bloqueia as linhas dos orçamentos aplicáveis, em ordem fixa
// (sem deadlock), confere gasto + reservas ativas + pedido ≤ limite e só então grava.
func (s *Store) Reserve(ctx context.Context, req ReserveRequest) (Reservation, error) {
	if req.AmountMicros < 0 || req.AmountMicros > 1_000_000_000_000 {
		return Reservation{}, &FieldError{"amount_micros", "Valor fora da faixa."}
	}
	ttl := req.TTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	if ttl > time.Hour {
		ttl = time.Hour
	}
	var agent, conn any
	if req.AgentID != "" {
		agent = req.AgentID
	}
	if req.ConnectionID != "" {
		conn = req.ConnectionID
	}
	now := s.now()
	var res Reservation
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+budgetCols+` FROM budgets
			WHERE state = 'active' AND (scope = 'global' OR agent_id = $1::uuid) ORDER BY id FOR UPDATE`, agent)
		if err != nil {
			return persistence.MapError(err)
		}
		var bs []Budget
		for rows.Next() {
			b, err := scanBudget(rows)
			if err != nil {
				rows.Close()
				return err
			}
			bs = append(bs, b)
		}
		rows.Close()
		hasGlobal := false
		for _, b := range bs {
			if b.Scope == "global" {
				hasGlobal = true
			}
		}
		if !hasGlobal {
			return ErrNoBudget
		}
		for _, b := range bs {
			sp, rs, err := s.spentAndReserved(ctx, tx, b, now)
			if err != nil {
				return err
			}
			if sp+rs+req.AmountMicros > b.LimitMicros {
				return &ExceededError{Scope: b.Scope, Period: b.Period, Limit: b.LimitMicros, Used: sp + rs, Requested: req.AmountMicros}
			}
		}
		res = Reservation{AmountMicros: req.AmountMicros, AgentID: req.AgentID, ConnectionID: req.ConnectionID, ExpiresAt: now.Add(ttl)}
		return persistence.MapError(tx.QueryRow(ctx, `INSERT INTO usage_reservations (agent_id, connection_id, amount_micros, expires_at, created_at)
			VALUES ($1::uuid, $2::uuid, $3, $4, $5) RETURNING id`, agent, conn, req.AmountMicros, res.ExpiresAt, now).Scan(&res.ID))
	})
	return res, err
}

// Actual é o que foi gasto de verdade numa chamada.
type Actual struct {
	Model        string
	InputTokens  int64
	OutputTokens int64
	CostMicros   int64
	Source       string // provider_reported | estimated
}

// Settlement é o resultado da liquidação.
type Settlement struct {
	UsageID        int64 `json:"usage_id"`
	CostMicros     int64 `json:"cost_micros"`
	OverrunMicros  int64 `json:"overrun_micros"` // gasto acima do que foi reservado
	AlreadySettled bool  `json:"already_settled"`
}

// Settle troca a reserva pelo gasto real. É idempotente (liquidar de novo devolve
// a mesma liquidação). Gasto acima da reserva é registrado (não dá para "desgastar")
// e aparece como overrun; o excesso passa a contar nas próximas reservas.
func (s *Store) Settle(ctx context.Context, reservationID string, a Actual) (Settlement, error) {
	if a.CostMicros < 0 || a.InputTokens < 0 || a.OutputTokens < 0 {
		return Settlement{}, &FieldError{"actual", "Valores negativos."}
	}
	if a.Source != SourceProvider && a.Source != SourceEstimated {
		return Settlement{}, &FieldError{"source", "Fonte inválida (provider_reported ou estimated)."}
	}
	var st Settlement
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		var state string
		var amount int64
		var agent, conn *string
		var usage *int64
		err := tx.QueryRow(ctx, `SELECT state, amount_micros, agent_id, connection_id, usage_id FROM usage_reservations WHERE id = $1 FOR UPDATE`, reservationID).Scan(&state, &amount, &agent, &conn, &usage)
		if err != nil {
			return persistence.MapError(err)
		}
		switch state {
		case "released":
			return ErrReservationClosed
		case "settled":
			var cost, over int64
			if err := tx.QueryRow(ctx, `SELECT cost_micros, overrun_micros FROM usage_records WHERE id = $1`, *usage).Scan(&cost, &over); err != nil {
				return persistence.MapError(err)
			}
			st = Settlement{UsageID: *usage, CostMicros: cost, OverrunMicros: over, AlreadySettled: true}
			return nil
		}
		over := max(0, a.CostMicros-amount)
		var id int64
		if err := tx.QueryRow(ctx, `INSERT INTO usage_records (kind, agent_id, connection_id, model, input_tokens, output_tokens, cost_micros, source, reservation_id, overrun_micros, observed_at)
			VALUES ('api_cost', $1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8::uuid, $9, $10) RETURNING id`,
			agent, conn, a.Model, a.InputTokens, a.OutputTokens, a.CostMicros, a.Source, reservationID, over, s.now()).Scan(&id); err != nil {
			return persistence.MapError(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE usage_reservations SET state = 'settled', usage_id = $2, settled_at = $3 WHERE id = $1`, reservationID, id, s.now()); err != nil {
			return persistence.MapError(err)
		}
		st = Settlement{UsageID: id, CostMicros: a.CostMicros, OverrunMicros: over}
		return nil
	})
	return st, err
}

// Release devolve a reserva sem gasto (a chamada falhou antes de custar). Idempotente.
func (s *Store) Release(ctx context.Context, reservationID string) error {
	tag, err := s.DB.Exec(ctx, `UPDATE usage_reservations SET state = 'released', settled_at = $2 WHERE id = $1 AND state = 'reserved'`, reservationID, s.now())
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 1 {
		return nil
	}
	var state string
	if err := s.DB.QueryRow(ctx, `SELECT state FROM usage_reservations WHERE id = $1`, reservationID).Scan(&state); err != nil {
		return persistence.MapError(err)
	}
	if state == "settled" {
		return ErrSettled
	}
	return nil
}

// SubscriptionUse registra consumo de uma conta de ASSINATURA (sem custo por chamada).
type SubscriptionUse struct {
	AgentID      string
	ConnectionID string
	Model        string
	Units        int64
	Unit         string // ex.: messages, tokens
	InputTokens  int64
	OutputTokens int64
	Source       string
	ObservedAt   time.Time
}

// RecordSubscription grava o consumo; nunca entra na soma de gasto em API.
func (s *Store) RecordSubscription(ctx context.Context, u SubscriptionUse) (int64, error) {
	if u.Units < 0 || u.InputTokens < 0 || u.OutputTokens < 0 || strings.TrimSpace(u.Unit) == "" || len(u.Unit) > 32 || u.ConnectionID == "" {
		return 0, &FieldError{"subscription", "Conta, unidade e valores não negativos são obrigatórios."}
	}
	if u.Source != SourceProvider && u.Source != SourceEstimated {
		return 0, &FieldError{"source", "Fonte inválida (provider_reported ou estimated)."}
	}
	if u.ObservedAt.IsZero() {
		u.ObservedAt = s.now()
	}
	var agent any
	if u.AgentID != "" {
		agent = u.AgentID
	}
	var id int64
	err := s.DB.QueryRow(ctx, `INSERT INTO usage_records (kind, agent_id, connection_id, model, input_tokens, output_tokens, quota_units, quota_unit, source, observed_at)
		VALUES ('subscription', $1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
		agent, u.ConnectionID, u.Model, u.InputTokens, u.OutputTokens, u.Units, u.Unit, u.Source, u.ObservedAt).Scan(&id)
	return id, persistence.MapError(err)
}

// QuotaObservation é uma cota que o adaptador OBSERVOU (nunca estimada).
type QuotaObservation struct {
	ConnectionID string     `json:"connection_id"`
	Used         int64      `json:"used"`
	Limit        int64      `json:"limit"`
	Unit         string     `json:"unit"`
	ResetsAt     *time.Time `json:"resets_at"`
	Source       string     `json:"source"`
	ObservedAt   time.Time  `json:"observed_at"`
}

// ObserveQuota registra uma observação de cota com fonte e horário.
func (s *Store) ObserveQuota(ctx context.Context, q QuotaObservation) error {
	if q.ConnectionID == "" || q.Used < 0 || q.Limit < 0 || strings.TrimSpace(q.Unit) == "" || strings.TrimSpace(q.Source) == "" || q.ObservedAt.IsZero() {
		return &FieldError{"quota", "Conta, unidade, fonte e horário observados são obrigatórios."}
	}
	_, err := s.DB.Exec(ctx, `INSERT INTO quota_observations (connection_id, used, quota_limit, unit, resets_at, source, observed_at) VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)`,
		q.ConnectionID, q.Used, q.Limit, q.Unit, q.ResetsAt, q.Source, q.ObservedAt)
	return persistence.MapError(err)
}

// LatestQuota devolve a observação mais recente da conta.
func (s *Store) LatestQuota(ctx context.Context, connectionID string) (QuotaObservation, error) {
	var q QuotaObservation
	err := s.DB.QueryRow(ctx, `SELECT connection_id, used, quota_limit, unit, resets_at, source, observed_at FROM quota_observations
		WHERE connection_id = $1 ORDER BY observed_at DESC, id DESC LIMIT 1`, connectionID).Scan(&q.ConnectionID, &q.Used, &q.Limit, &q.Unit, &q.ResetsAt, &q.Source, &q.ObservedAt)
	return q, persistence.MapError(err)
}

// Reconciliation é o resultado da conferência com o total informado pelo provedor.
type Reconciliation struct {
	RecordedMicros int64 `json:"recorded_micros"`
	ReportedMicros int64 `json:"reported_micros"`
	DriftMicros    int64 `json:"drift_micros"` // informado − registrado
	AdjustmentID   int64 `json:"adjustment_id,omitempty"`
}

// Reconcile confere o gasto registrado da conta num período com o total do provedor.
// Faltou registrar → lança um ajuste (fonte reconciliation) para o total bater;
// registramos a mais → guarda a diferença negativa e MANTÉM o valor maior (conservador).
func (s *Store) Reconcile(ctx context.Context, connectionID string, from, to time.Time, reportedMicros int64, source string) (Reconciliation, error) {
	if connectionID == "" || !to.After(from) || reportedMicros < 0 || strings.TrimSpace(source) == "" {
		return Reconciliation{}, &FieldError{"reconcile", "Conta, período, total informado e fonte são obrigatórios."}
	}
	var out Reconciliation
	err := persistence.WithTx(ctx, s.DB, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 16))`, connectionID); err != nil {
			return persistence.MapError(err)
		}
		if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(cost_micros), 0) FROM usage_records WHERE kind = 'api_cost' AND connection_id = $1 AND observed_at >= $2 AND observed_at < $3`,
			connectionID, from, to).Scan(&out.RecordedMicros); err != nil {
			return persistence.MapError(err)
		}
		out.ReportedMicros = reportedMicros
		out.DriftMicros = reportedMicros - out.RecordedMicros
		var adj *int64
		if out.DriftMicros > 0 {
			var id int64
			if err := tx.QueryRow(ctx, `INSERT INTO usage_records (kind, connection_id, cost_micros, source, observed_at) VALUES ('api_cost', $1::uuid, $2, 'reconciliation', $3) RETURNING id`,
				connectionID, out.DriftMicros, to.Add(-time.Second)).Scan(&id); err != nil {
				return persistence.MapError(err)
			}
			adj, out.AdjustmentID = &id, id
		}
		_, err := tx.Exec(ctx, `INSERT INTO usage_reconciliations (connection_id, period_start, period_end, recorded_micros, reported_micros, drift_micros, adjustment_id, source)
			VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`, connectionID, from, to, out.RecordedMicros, reportedMicros, out.DriftMicros, adj, source)
		return persistence.MapError(err)
	})
	return out, err
}

// Report é o uso do período, com gasto em API e assinatura SEPARADOS.
type Report struct {
	From, To     time.Time
	API          APIReport          `json:"api"`
	Subscription []SubscriptionLine `json:"subscription"`
}

// APIReport é o gasto em API.
type APIReport struct {
	SpentMicros    int64            `json:"spent_micros"`
	ReservedMicros int64            `json:"reserved_micros"`
	BySource       map[string]int64 `json:"by_source"`
	ByAgent        map[string]int64 `json:"by_agent"`
	OverrunMicros  int64            `json:"overrun_micros"`
}

// SubscriptionLine é o consumo de uma conta de assinatura numa unidade.
type SubscriptionLine struct {
	ConnectionID string `json:"connection_id"`
	Unit         string `json:"unit"`
	Units        int64  `json:"units"`
	Calls        int64  `json:"calls"`
}

// Report resume o uso entre from (inclusivo) e to (exclusivo).
func (s *Store) Report(ctx context.Context, from, to time.Time) (Report, error) {
	r := Report{From: from, To: to, API: APIReport{BySource: map[string]int64{}, ByAgent: map[string]int64{}}, Subscription: []SubscriptionLine{}}
	rows, err := s.DB.Query(ctx, `SELECT source, COALESCE(agent_id::text, ''), SUM(cost_micros), SUM(overrun_micros) FROM usage_records
		WHERE kind = 'api_cost' AND observed_at >= $1 AND observed_at < $2 GROUP BY source, agent_id`, from, to)
	if err != nil {
		return r, persistence.MapError(err)
	}
	for rows.Next() {
		var src, agent string
		var cost, over int64
		if err := rows.Scan(&src, &agent, &cost, &over); err != nil {
			rows.Close()
			return r, persistence.MapError(err)
		}
		r.API.SpentMicros += cost
		r.API.OverrunMicros += over
		r.API.BySource[src] += cost
		r.API.ByAgent[agent] += cost
	}
	rows.Close()
	if err := s.DB.QueryRow(ctx, `SELECT COALESCE(SUM(amount_micros), 0) FROM usage_reservations WHERE state = 'reserved' AND expires_at > $1`, s.now()).Scan(&r.API.ReservedMicros); err != nil {
		return r, persistence.MapError(err)
	}
	srows, err := s.DB.Query(ctx, `SELECT COALESCE(connection_id::text, ''), quota_unit, SUM(quota_units), count(*) FROM usage_records
		WHERE kind = 'subscription' AND observed_at >= $1 AND observed_at < $2 GROUP BY connection_id, quota_unit ORDER BY 1, 2`, from, to)
	if err != nil {
		return r, persistence.MapError(err)
	}
	defer srows.Close()
	for srows.Next() {
		var l SubscriptionLine
		if err := srows.Scan(&l.ConnectionID, &l.Unit, &l.Units, &l.Calls); err != nil {
			return r, persistence.MapError(err)
		}
		r.Subscription = append(r.Subscription, l)
	}
	return r, persistence.MapError(srows.Err())
}

// ---- preço e estimativa ----------------------------------------------------

// Price é o preço em micro-USD por milhão de tokens.
type Price struct{ InputPerMTok, OutputPerMTok int64 }

// PriceOf lê o preço do modelo; sem preço, ErrNoPrice (falha fechada).
func PriceOf(m models.Model) (Price, error) {
	if m.InputMicrosPerMTok == nil || m.OutputMicrosPerMTok == nil {
		return Price{}, ErrNoPrice
	}
	return Price{InputPerMTok: *m.InputMicrosPerMTok, OutputPerMTok: *m.OutputMicrosPerMTok}, nil
}

func ceilDiv(a, b int64) int64 { return (a + b - 1) / b }

// CostOf calcula o custo (arredondado para cima) de uma quantidade de tokens.
func CostOf(p Price, inputTokens, outputTokens int64) int64 {
	return ceilDiv(inputTokens*p.InputPerMTok, 1_000_000) + ceilDiv(outputTokens*p.OutputPerMTok, 1_000_000)
}

// EstimateTokens estima tokens SEM métrica oficial: ceil(bytes/3) por texto, que
// costuma superestimar (prefere reservar a mais a estourar o teto).
func EstimateTokens(texts ...string) int64 {
	var n int64
	for _, t := range texts {
		n += ceilDiv(int64(len(t)), 3)
	}
	return n
}
