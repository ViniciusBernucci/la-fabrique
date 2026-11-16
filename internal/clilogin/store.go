// Package clilogin conecta as contas de assinatura de IA (Claude e Codex) pelo painel (ADR-0065).
// A API só registra a intenção (Store); o worker roda o login OFICIAL do cliente num container, num
// volume por conta, e publica o link/código para a pessoa autorizar. A fábrica nunca lê o token: ele
// vive no volume (ADR-0011), e o cofre guarda só um marcador.
package clilogin

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Provider é o cliente de IA da conta.
type Provider string

const (
	Claude Provider = "claude"
	Codex  Provider = "codex"
)

// Valid diz se é um provedor com login suportado.
func (p Provider) Valid() bool { return p == Claude || p == Codex }

// Estados do login.
const (
	StatePending      = "pending"       // registrado, o worker ainda não pegou
	StatePreparing    = "preparing"     // worker preparando imagem e volume
	StateAwaitingUser = "awaiting_user" // link/código prontos: a pessoa autoriza
	StateVerifying    = "verifying"     // o cliente terminou; confirmando o login
	StateDone         = "done"
	StateFailed       = "failed"
	StateCancelled    = "cancelled"
)

// TTL é quanto tempo a pessoa tem para autorizar (o código do Codex vale 15 minutos).
const TTL = 15 * time.Minute

// Erros do fluxo.
var (
	ErrInProgress = errors.New("já existe uma conexão deste provedor em andamento")
	ErrNameTaken  = errors.New("já existe uma conexão ativa com esse nome")
	ErrNotWaiting = errors.New("o login não está esperando um código")
	ErrBadCode    = errors.New("código inválido")
)

// Login é uma tentativa de conectar uma conta.
type Login struct {
	ID           string
	Provider     Provider
	Name         string
	ConnectionID string
	State        string
	URL          string
	Code         string
	NeedsCode    bool
	Error        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	ExpiresAt    time.Time
}

// Open diz se o login ainda pode andar.
func (l Login) Open() bool {
	switch l.State {
	case StatePending, StatePreparing, StateAwaitingUser, StateVerifying:
		return true
	}
	return false
}

// Store é a persistência dos logins.
type Store struct {
	DB  persistence.DBTX
	Now func() time.Time
}

func (s *Store) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

const cols = `id::text, provider, name, connection_id::text, state, COALESCE(challenge_url, ''), COALESCE(challenge_code, ''),
	needs_code, error, created_at, updated_at, expires_at`

func scan(row interface{ Scan(...any) error }) (Login, error) {
	var l Login
	var p string
	err := row.Scan(&l.ID, &p, &l.Name, &l.ConnectionID, &l.State, &l.URL, &l.Code, &l.NeedsCode, &l.Error, &l.CreatedAt, &l.UpdatedAt, &l.ExpiresAt)
	l.Provider = Provider(p)
	return l, persistence.MapError(err)
}

// Start registra um login novo. Só um por provedor pode estar aberto: a pessoa conclui ou cancela.
func (s *Store) Start(ctx context.Context, p Provider, name string) (Login, error) {
	name = strings.TrimSpace(name)
	if !p.Valid() {
		return Login{}, fmt.Errorf("%w: provedor sem login por assinatura", persistence.ErrInvalidInput)
	}
	if name == "" || len(name) > 80 {
		return Login{}, fmt.Errorf("%w: informe um nome de até 80 caracteres", persistence.ErrInvalidInput)
	}
	var taken bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM connections WHERE name = $1 AND state = 'active')`, name).Scan(&taken); err != nil {
		return Login{}, persistence.MapError(err)
	}
	if taken {
		return Login{}, ErrNameTaken
	}
	// Abertos vencidos não bloqueiam: o worker os encerra no próximo ciclo, e aqui já valem como mortos.
	var open bool
	if err := s.DB.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM cli_logins WHERE provider = $1 AND expires_at > $2
		AND state IN ('pending','preparing','awaiting_user','verifying'))`, string(p), s.now()).Scan(&open); err != nil {
		return Login{}, persistence.MapError(err)
	}
	if open {
		return Login{}, ErrInProgress
	}
	return scan(s.DB.QueryRow(ctx, `INSERT INTO cli_logins (provider, name, expires_at) VALUES ($1, $2, $3) RETURNING `+cols,
		string(p), name, s.now().Add(TTL)))
}

// Get lê um login.
func (s *Store) Get(ctx context.Context, id string) (Login, error) {
	return scan(s.DB.QueryRow(ctx, `SELECT `+cols+` FROM cli_logins WHERE id = $1`, id))
}

// Claim entrega ao worker o próximo login pendente (e já o marca como em preparo).
func (s *Store) Claim(ctx context.Context) (Login, bool, error) {
	l, err := scan(s.DB.QueryRow(ctx, `UPDATE cli_logins SET state = 'preparing', updated_at = now()
		WHERE id = (SELECT id FROM cli_logins WHERE state = 'pending' AND expires_at > $1 ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1)
		RETURNING `+cols, s.now()))
	if errors.Is(err, persistence.ErrNotFound) {
		return Login{}, false, nil
	}
	return l, err == nil, err
}

// SetState muda o estado de um login aberto.
func (s *Store) SetState(ctx context.Context, id, state string) error {
	_, err := s.DB.Exec(ctx, `UPDATE cli_logins SET state = $2, updated_at = now()
		WHERE id = $1 AND state IN ('pending','preparing','awaiting_user','verifying')`, id, state)
	return persistence.MapError(err)
}

// SetChallenge publica o link/código para a pessoa autorizar.
func (s *Store) SetChallenge(ctx context.Context, id string, c Challenge) error {
	_, err := s.DB.Exec(ctx, `UPDATE cli_logins SET state = 'awaiting_user', challenge_url = $2, challenge_code = NULLIF($3, ''),
		needs_code = $4, updated_at = now() WHERE id = $1 AND state IN ('preparing','awaiting_user')`, id, c.URL, c.Code, c.NeedsCode)
	return persistence.MapError(err)
}

var codeShape = regexp.MustCompile(`^[A-Za-z0-9_#\-.~+/=]{4,512}$`)

// SubmitCode guarda o código que a pessoa colou (Claude), para o worker entregar ao cliente.
func (s *Store) SubmitCode(ctx context.Context, id, code string) error {
	code = strings.TrimSpace(code)
	if !codeShape.MatchString(code) {
		return ErrBadCode
	}
	tag, err := s.DB.Exec(ctx, `UPDATE cli_logins SET submitted_code = $2, updated_at = now()
		WHERE id = $1 AND state = 'awaiting_user' AND needs_code`, id, code)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		if _, gerr := s.Get(ctx, id); gerr != nil {
			return gerr
		}
		return ErrNotWaiting
	}
	return nil
}

// TakeCode entrega o código colado ao worker UMA vez e o apaga em seguida.
func (s *Store) TakeCode(ctx context.Context, id string) (string, bool, error) {
	var code string
	err := s.DB.QueryRow(ctx, `WITH old AS (SELECT submitted_code FROM cli_logins WHERE id = $1 AND submitted_code IS NOT NULL FOR UPDATE),
		cleared AS (UPDATE cli_logins SET submitted_code = NULL WHERE id = $1 AND submitted_code IS NOT NULL)
		SELECT submitted_code FROM old`, id).Scan(&code)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return "", false, nil
	}
	return code, err == nil, persistence.MapError(err)
}

// Cancel encerra um login aberto; o worker vê o estado e para o container.
func (s *Store) Cancel(ctx context.Context, id string) (Login, error) {
	l, err := s.Get(ctx, id)
	if err != nil {
		return Login{}, err
	}
	if !l.Open() {
		return l, nil // já terminou: idempotente
	}
	return scan(s.DB.QueryRow(ctx, `UPDATE cli_logins SET state = 'cancelled', error = 'Cancelado.', challenge_url = NULL,
		challenge_code = NULL, submitted_code = NULL, updated_at = now() WHERE id = $1 RETURNING `+cols, id))
}

// Finish encerra o login e apaga tudo que era de curta duração.
func (s *Store) Finish(ctx context.Context, id, state, msg string) error {
	if len(msg) > 400 {
		msg = msg[:400]
	}
	_, err := s.DB.Exec(ctx, `UPDATE cli_logins SET state = $2, error = $3, challenge_url = NULL, challenge_code = NULL,
		submitted_code = NULL, updated_at = now() WHERE id = $1 AND state IN ('pending','preparing','awaiting_user','verifying')`, id, state, msg)
	return persistence.MapError(err)
}

// Leftovers lista os logins ainda abertos (para o worker varrer ao subir ou ao vencer).
func (s *Store) Leftovers(ctx context.Context) ([]Login, error) {
	rows, err := s.DB.Query(ctx, `SELECT `+cols+` FROM cli_logins WHERE state IN ('pending','preparing','awaiting_user','verifying')`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	var out []Login
	for rows.Next() {
		l, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, persistence.MapError(rows.Err())
}
