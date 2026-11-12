package persistence

import (
	"context"
	"encoding/json"
	"fmt"
)

// Estados da operação externa (ARQUITETURA, "Estados, fila e exclusão mútua").
const (
	OutboxIntent    = "intent"
	OutboxSending   = "sending"
	OutboxConfirmed = "confirmed"
	OutboxFailed    = "failed"
	OutboxUncertain = "uncertain"
)

var outboxTransitions = map[string][]string{
	OutboxIntent:    {OutboxSending},
	OutboxSending:   {OutboxConfirmed, OutboxFailed, OutboxUncertain},
	OutboxUncertain: {OutboxConfirmed, OutboxFailed, OutboxSending},
	OutboxFailed:    {OutboxSending},
}

// OutboxEntry é a intenção de um efeito externo, única por chave de negócio.
type OutboxEntry struct {
	ID          string
	ProjectID   string
	BusinessKey string
	Destination string
	Request     json.RawMessage
	State       string
	Proof       json.RawMessage
	Attempts    int
	Version     int
}

const outboxCols = `id, project_id, business_key, destination, request, state, proof, attempts, version`

func scanOutbox(row interface{ Scan(...any) error }) (OutboxEntry, error) {
	var e OutboxEntry
	var req, proof []byte
	if err := row.Scan(&e.ID, &e.ProjectID, &e.BusinessKey, &e.Destination, &req, &e.State, &proof, &e.Attempts, &e.Version); err != nil {
		return OutboxEntry{}, MapError(err)
	}
	e.Request, e.Proof = req, proof
	return e, nil
}

// EnqueueOutbox registra a intenção. Repetir a mesma chave de negócio devolve a
// entrada existente (created=false) sem duplicar o efeito lógico.
func EnqueueOutbox(ctx context.Context, db DBTX, projectID, businessKey, destination string, request any) (entry OutboxEntry, created bool, err error) {
	raw, err := json.Marshal(request)
	if err != nil || request == nil {
		raw = []byte("{}")
	}
	row := db.QueryRow(ctx, `
		INSERT INTO outbox (project_id, business_key, destination, request)
		VALUES ($1, $2, $3, $4::jsonb)
		ON CONFLICT (project_id, business_key) DO NOTHING
		RETURNING `+outboxCols, projectID, businessKey, destination, string(raw))
	entry, err = scanOutbox(row)
	if err == nil {
		return entry, true, nil
	}
	if err != ErrNotFound {
		return OutboxEntry{}, false, err
	}
	entry, err = scanOutbox(db.QueryRow(ctx,
		`SELECT `+outboxCols+` FROM outbox WHERE project_id = $1 AND business_key = $2`, projectID, businessKey))
	return entry, false, err
}

// TransitionOutbox move a entrada para `to` se a transição for válida e a
// versão esperada ainda for a atual. Tentativas contam entradas em `sending`.
func TransitionOutbox(ctx context.Context, db DBTX, projectID, id string, expectedVersion int, to string, proof any) (OutboxEntry, error) {
	current, err := scanOutbox(db.QueryRow(ctx,
		`SELECT `+outboxCols+` FROM outbox WHERE project_id = $1 AND id = $2`, projectID, id))
	if err != nil {
		return OutboxEntry{}, err
	}
	allowed := false
	for _, s := range outboxTransitions[current.State] {
		if s == to {
			allowed = true
		}
	}
	if !allowed {
		return OutboxEntry{}, fmt.Errorf("%w: transição %s → %s não permitida", ErrInvalidInput, current.State, to)
	}
	var proofJSON any
	if proof != nil {
		b, err := json.Marshal(proof)
		if err != nil {
			return OutboxEntry{}, fmt.Errorf("%w: prova não serializável", ErrInvalidInput)
		}
		proofJSON = string(b)
	}
	row := db.QueryRow(ctx, `
		UPDATE outbox SET state = $4, proof = COALESCE($5::jsonb, proof),
		       attempts = attempts + CASE WHEN $4 = 'sending' THEN 1 ELSE 0 END,
		       version = version + 1, updated_at = now()
		WHERE project_id = $1 AND id = $2 AND version = $3
		RETURNING `+outboxCols, projectID, id, expectedVersion, to, proofJSON)
	entry, err := scanOutbox(row)
	if err == ErrNotFound {
		return OutboxEntry{}, ErrVersionConflict
	}
	return entry, err
}
