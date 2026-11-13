package projection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// Motivos de pendência.
const (
	ReasonMapping      = "mapping"
	ReasonConflict     = "conflict"
	ReasonWrite        = "write"
	ReasonUnregistered = "unregistered"
	ReasonUncertain    = "uncertain"
)

// DB é o que o pacote precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Projector projeta as intenções pendentes.
type Projector struct {
	DB     DB
	Ledger *effects.Ledger
}

// Intent é uma intenção de projeção lida da outbox.
type Intent struct {
	BusinessKey string
	Issue       string
	To          string
	Version     int
	State       string
}

// Result é o resultado de uma intenção.
type Result struct {
	BusinessKey string `json:"business_key"`
	Issue       string `json:"issue"`
	Target      string `json:"target"`
	Outcome     string `json:"outcome"` // applied | already_confirmed | reconciled | superseded | pending
	Reason      string `json:"reason,omitempty"`
}

// Pending é uma pendência exposta ao painel.
type Pending struct {
	Issue    string    `json:"issue"`
	Version  int       `json:"version"`
	Target   string    `json:"target"`
	Reason   string    `json:"reason"`
	Detail   string    `json:"detail"`
	Attempts int       `json:"attempts"`
	Updated  time.Time `json:"updated_at"`
}

func (p *Projector) intents(ctx context.Context, projectID string) ([]Intent, error) {
	rows, err := p.DB.Query(ctx, `SELECT business_key, request, state FROM outbox WHERE project_id=$1 AND destination=$2
		AND state IN ('intent','failed','sending','uncertain') AND NOT (state='failed' AND proof ? 'superseded') ORDER BY created_at, business_key LIMIT 200`, projectID, sentinel.DestProjection)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Intent
	for rows.Next() {
		var in Intent
		var raw []byte
		if err := rows.Scan(&in.BusinessKey, &raw, &in.State); err != nil {
			return nil, err
		}
		var req struct {
			Issue   string `json:"issue"`
			To      string `json:"to"`
			Version int    `json:"version"`
		}
		_ = json.Unmarshal(raw, &req)
		in.Issue, in.To, in.Version = req.Issue, req.To, req.Version
		out = append(out, in)
	}
	return out, rows.Err()
}

func (p *Projector) setPending(ctx context.Context, projectID string, in Intent, reason, detail string) error {
	_, err := p.DB.Exec(ctx, `INSERT INTO projection_pending (project_id, business_key, issue_key, version, target_code, reason, detail, attempts)
		VALUES ($1,$2,$3,$4,$5,$6,$7,1) ON CONFLICT (project_id, business_key) DO UPDATE
		SET reason=$6, detail=$7, attempts=projection_pending.attempts+1, updated_at=now()`,
		projectID, in.BusinessKey, in.Issue, in.Version, in.To, reason, truncate(detail, 300))
	return persistence.MapError(err)
}

func (p *Projector) clearPending(ctx context.Context, projectID, key string) error {
	_, err := p.DB.Exec(ctx, `DELETE FROM projection_pending WHERE project_id=$1 AND business_key=$2`, projectID, key)
	return persistence.MapError(err)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// Run projeta as intenções pendentes do projeto, na ordem. Intenção mais antiga de uma Issue que
// já tem outra mais nova é descartada como superada (evento antigo não desfaz estado novo).
func (p *Projector) Run(ctx context.Context, projectID string, tr integrations.Tracker, m flow.ProjectMap) ([]Result, error) {
	ints, err := p.intents(ctx, projectID)
	if err != nil {
		return nil, err
	}
	newest := map[string]int{}
	for _, in := range ints {
		if in.Version > newest[in.Issue] {
			newest[in.Issue] = in.Version
		}
	}
	var out []Result
	for _, in := range ints {
		res := Result{BusinessKey: in.BusinessKey, Issue: in.Issue, Target: in.To}
		if in.Version < newest[in.Issue] {
			if err := p.supersede(ctx, projectID, in); err != nil {
				return out, err
			}
			res.Outcome = "superseded"
			out = append(out, res)
			continue
		}
		r, err := p.one(ctx, projectID, tr, m, in)
		if err != nil {
			return out, err
		}
		out = append(out, r)
	}
	return out, nil
}

// supersede encerra a intenção antiga sem escrever na origem (marca falha com prova de superação).
func (p *Projector) supersede(ctx context.Context, projectID string, in Intent) error {
	return p.supersedeEntry(ctx, projectID, in)
}

func (p *Projector) supersedeEntry(ctx context.Context, projectID string, in Intent) error {
	var id string
	if err := p.DB.QueryRow(ctx, `SELECT id::text FROM outbox WHERE project_id=$1 AND business_key=$2`, projectID, in.BusinessKey).Scan(&id); err != nil {
		return persistence.MapError(err)
	}
	e, err := persistence.GetOutbox(ctx, p.DB, projectID, id)
	if err != nil {
		return err
	}
	if e.State == persistence.OutboxIntent || e.State == persistence.OutboxFailed {
		if e, err = persistence.TransitionOutbox(ctx, p.DB, projectID, id, e.Version, persistence.OutboxSending, nil); err != nil {
			return err
		}
		if _, err = persistence.TransitionOutbox(ctx, p.DB, projectID, id, e.Version, persistence.OutboxFailed, map[string]any{"superseded": true}); err != nil {
			return err
		}
	}
	return p.clearPending(ctx, projectID, in.BusinessKey)
}

func (p *Projector) one(ctx context.Context, projectID string, tr integrations.Tracker, m flow.ProjectMap, in Intent) (Result, error) {
	res := Result{BusinessKey: in.BusinessKey, Issue: in.Issue, Target: in.To, Outcome: "pending"}
	fail := func(reason, detail string) (Result, error) {
		res.Reason = reason
		return res, p.setPending(ctx, projectID, in, reason, detail)
	}
	move := strings.TrimSpace(m.Moves[in.To])
	if move == "" {
		return fail(ReasonMapping, "sem status ou transição remota mapeada para "+in.To)
	}
	var remoteID string
	err := p.DB.QueryRow(ctx, `SELECT remote_id FROM issue_registry WHERE project_id=$1 AND issue_key=$2 AND source=$3`, projectID, in.Issue, m.Source).Scan(&remoteID)
	if err != nil {
		return fail(ReasonUnregistered, "a Issue não está no registro da origem")
	}
	ad := &adapter{t: tr, item: remoteID, move: move, target: in.To, m: m}
	r, err := p.Ledger.Run(ctx, effects.Spec{ProjectID: projectID, BusinessKey: in.BusinessKey, Operation: sentinel.DestProjection,
		Request: map[string]any{"issue": in.Issue, "to": in.To, "version": in.Version}}, ad)
	switch {
	case err == nil:
		res.Outcome = r.Outcome
		return res, p.clearPending(ctx, projectID, in.BusinessKey)
	case errors.Is(err, effects.ErrBlocked):
		return fail(ReasonUncertain, "efeito inconclusivo: decisão humana necessária")
	case errors.Is(err, effects.ErrInFlight):
		return res, nil
	case errors.Is(err, effects.ErrNotApplied):
		return fail(ReasonConflict, err.Error())
	default:
		return fail(ReasonWrite, err.Error())
	}
}

var errConflict = "a origem não aceita a transição agora"

// adapter escreve UMA transição de status; nenhum outro campo é tocado.
type adapter struct {
	t      integrations.Tracker
	item   string
	move   string
	target string
	m      flow.ProjectMap
}

func (a *adapter) is(ctx context.Context) (bool, string, error) {
	cur, err := a.t.GetItem(ctx, a.item)
	if err != nil {
		return false, "", err
	}
	r, ok := a.m.Resolve(cur.Status)
	return ok && r.Code == a.target, cur.Status, nil
}

func (a *adapter) Apply(ctx context.Context, _ effects.Op) (any, error) {
	already, cur, err := a.is(ctx)
	if err != nil {
		return nil, err // indisponível: o ledger verifica antes de repetir
	}
	if already {
		return map[string]any{"status": cur, "already": true}, nil
	}
	ts, err := a.t.Transitions(ctx, a.item)
	if err != nil {
		return nil, err
	}
	for _, tr := range ts {
		if tr.ID == a.move || tr.ToStatus == a.move {
			if err := a.t.Transition(ctx, a.item, tr.ID); err != nil {
				return nil, err
			}
			return map[string]any{"from": cur, "to": tr.ToStatus}, nil
		}
	}
	return nil, fmt.Errorf("%w: %s (de %q para %q)", effects.ErrNotApplied, errConflict, cur, a.move)
}

func (a *adapter) Verify(ctx context.Context, _ effects.Op) (effects.Verdict, error) {
	ok, cur, err := a.is(ctx)
	if err != nil {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "não foi possível ler a Issue: " + err.Error()}, nil
	}
	if ok {
		return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"status": cur}}, nil
	}
	return effects.Verdict{Status: effects.VerdictAbsent}, nil
}

// PendingFor lista as pendências do projeto (mapeamento, conflito, escrita) para o painel.
func (p *Projector) PendingFor(ctx context.Context, projectID string) ([]Pending, error) {
	rows, err := p.DB.Query(ctx, `SELECT issue_key, version, target_code, reason, detail, attempts, updated_at FROM projection_pending
		WHERE project_id=$1 ORDER BY issue_key, version`, projectID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Pending
	for rows.Next() {
		var x Pending
		if err := rows.Scan(&x.Issue, &x.Version, &x.Target, &x.Reason, &x.Detail, &x.Attempts, &x.Updated); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// Resolver entrega o adaptador e o mapa de status do projeto.
type Resolver func(ctx context.Context, projectID string) (integrations.Tracker, flow.ProjectMap, error)

// Routine é a rotina periódica de projeção. Qualquer pendência devolve erro à agenda (backoff).
func (p *Projector) Routine(resolve Resolver, every time.Duration) sentinel.Routine {
	return sentinel.Routine{Name: sentinel.RoutineProjection, Every: every, Run: func(ctx context.Context, scope, _ string) (string, error) {
		if scope == "" {
			return "", nil
		}
		var n int
		if err := p.DB.QueryRow(ctx, `SELECT count(*) FROM outbox WHERE project_id=$1 AND destination=$2 AND state IN ('intent','failed','sending','uncertain')`, scope, sentinel.DestProjection).Scan(&n); err != nil || n == 0 {
			return "", err
		}
		tr, m, err := resolve(ctx, scope)
		if err != nil {
			return "", fmt.Errorf("origem indisponível para a projeção: %w", err)
		}
		res, err := p.Run(ctx, scope, tr, m)
		if err != nil {
			return "", err
		}
		for _, r := range res {
			if r.Outcome == "pending" {
				return "", fmt.Errorf("projeção pendente de %s: %s", r.Issue, r.Reason)
			}
		}
		return "", nil
	}}
}
