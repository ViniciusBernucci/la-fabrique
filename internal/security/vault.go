package security

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Tipos de conexão suportados (E04).
var connectionKinds = map[string]bool{"github": true, "jira": true, "clickup": true, "cli": true, "api": true, "mcp": true}

// Erros do cofre.
var (
	ErrConnectionRevoked = errors.New("conexão revogada")
	ErrInvalidSecret     = errors.New("segredo inválido")
)

// Connection é a visão pública de uma conexão. NUNCA contém o segredo.
type Connection struct {
	ID             string     `json:"id"`
	Kind           string     `json:"kind"`
	Name           string     `json:"name"`
	Target         string     `json:"target"`
	Scopes         string     `json:"scopes"`
	State          string     `json:"state"`
	SecretVersion  int        `json:"secret_version"`
	RotatedAt      *time.Time `json:"rotated_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	LastTestAt     *time.Time `json:"last_test_at,omitempty"`
	LastTestStatus string     `json:"last_test_status"`
	LastTestDetail string     `json:"last_test_detail"`
	Version        int        `json:"version"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// NewConnection reúne os dados de criação.
type NewConnection struct {
	Kind, Name, Target, Scopes string
	Secret                     Secret
}

// Vault guarda segredos cifrados com chave externa ao banco. Serviços acessam o
// valor só por Resolve, que consulta o estado a cada uso: uma revogação vale
// para o próximo uso, sem cache.
type Vault struct {
	db       persistence.DBTX
	sealer   *Sealer
	redactor *Redactor
}

// NewVault cria o cofre; redactor recebe todo segredo que passar por aqui.
func NewVault(db persistence.DBTX, sealer *Sealer, redactor *Redactor) *Vault {
	return &Vault{db: db, sealer: sealer, redactor: redactor}
}

const connectionColumns = `id, kind, name, target, scopes, state, secret_version, rotated_at, revoked_at,
	last_test_at, last_test_status, last_test_detail, version, created_at, updated_at`

type rowScanner interface{ Scan(dest ...any) error }

func scanConnection(r rowScanner) (Connection, error) {
	var c Connection
	err := r.Scan(&c.ID, &c.Kind, &c.Name, &c.Target, &c.Scopes, &c.State, &c.SecretVersion, &c.RotatedAt, &c.RevokedAt,
		&c.LastTestAt, &c.LastTestStatus, &c.LastTestDetail, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	return c, persistence.MapError(err)
}

func connAAD(id string) []byte { return []byte("connection:" + id) }

func validateSecret(s Secret) error {
	v := s.Reveal()
	if strings.TrimSpace(v) == "" || len(v) > 8192 {
		return fmt.Errorf("%w: informe um valor de até 8192 bytes", ErrInvalidSecret)
	}
	return nil
}

// Create cifra e grava a conexão. O id vira o AAD, então o id é gerado antes.
func (v *Vault) Create(ctx context.Context, n NewConnection) (Connection, error) {
	n.Name, n.Target, n.Scopes = strings.TrimSpace(n.Name), strings.TrimSpace(n.Target), strings.TrimSpace(n.Scopes)
	if !connectionKinds[n.Kind] {
		return Connection{}, fmt.Errorf("%w: tipo de conexão desconhecido", persistence.ErrInvalidInput)
	}
	if err := validateSecret(n.Secret); err != nil {
		return Connection{}, err
	}
	var id string
	if err := v.db.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&id); err != nil {
		return Connection{}, persistence.MapError(err)
	}
	sealed, err := v.sealer.Seal([]byte(n.Secret.Reveal()), connAAD(id))
	if err != nil {
		return Connection{}, err
	}
	v.redactor.Register(n.Secret.Reveal())
	c, err := scanConnection(v.db.QueryRow(ctx, `
		INSERT INTO connections (id, kind, name, target, scopes, secret_ciphertext, key_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING `+connectionColumns,
		id, n.Kind, n.Name, n.Target, n.Scopes, sealed, v.sealer.KeyID()))
	return c, err
}

// List devolve todas as conexões, mais novas primeiro.
func (v *Vault) List(ctx context.Context) ([]Connection, error) {
	rows, err := v.db.Query(ctx, `SELECT `+connectionColumns+` FROM connections ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Connection{}
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, persistence.MapError(rows.Err())
}

// Get devolve uma conexão.
func (v *Vault) Get(ctx context.Context, id string) (Connection, error) {
	return scanConnection(v.db.QueryRow(ctx, `SELECT `+connectionColumns+` FROM connections WHERE id = $1`, id))
}

// Rotate troca o segredo mantendo id e vínculos. O valor anterior é descartado
// (continua redigido em logs enquanto o processo viver). expectedVersion evita
// sobrescrever uma alteração concorrente.
func (v *Vault) Rotate(ctx context.Context, id string, secret Secret, expectedVersion int) (Connection, error) {
	if err := validateSecret(secret); err != nil {
		return Connection{}, err
	}
	sealed, err := v.sealer.Seal([]byte(secret.Reveal()), connAAD(id))
	if err != nil {
		return Connection{}, err
	}
	v.redactor.Register(secret.Reveal())
	c, err := scanConnection(v.db.QueryRow(ctx, `
		UPDATE connections SET secret_ciphertext = $2, key_id = $3, secret_version = secret_version + 1,
		       rotated_at = now(), version = version + 1, updated_at = now(),
		       last_test_status = 'not_run', last_test_detail = '', last_test_at = NULL
		WHERE id = $1 AND version = $4 AND state = 'active' RETURNING `+connectionColumns,
		id, sealed, v.sealer.KeyID(), expectedVersion))
	if errors.Is(err, persistence.ErrNotFound) {
		return Connection{}, v.diagnoseNoUpdate(ctx, id)
	}
	return c, err
}

// diagnoseNoUpdate explica por que um UPDATE condicional não afetou linhas.
func (v *Vault) diagnoseNoUpdate(ctx context.Context, id string) error {
	cur, err := v.Get(ctx, id)
	if err != nil {
		return err
	}
	if cur.State == "revoked" {
		return ErrConnectionRevoked
	}
	return persistence.ErrVersionConflict
}

// Revoke apaga o segredo e marca a conexão como revogada. Irreversível: para
// voltar a usar o destino, crie ou rotacione outra conexão.
func (v *Vault) Revoke(ctx context.Context, id string) (Connection, error) {
	c, err := scanConnection(v.db.QueryRow(ctx, `
		UPDATE connections SET state = 'revoked', secret_ciphertext = NULL, key_id = NULL, revoked_at = now(),
		       version = version + 1, updated_at = now()
		WHERE id = $1 AND state = 'active' RETURNING `+connectionColumns, id))
	if errors.Is(err, persistence.ErrNotFound) {
		cur, gerr := v.Get(ctx, id)
		if gerr != nil {
			return Connection{}, gerr
		}
		return cur, nil // já revogada: idempotente
	}
	return c, err
}

// Resolve devolve o segredo para uso de um serviço autorizado. Consulta o banco
// a cada chamada, então revogação e rotação valem no próximo uso.
func (v *Vault) Resolve(ctx context.Context, id string) (Secret, Connection, error) {
	var sealed []byte
	c := Connection{}
	row := v.db.QueryRow(ctx, `SELECT secret_ciphertext, `+connectionColumns+` FROM connections WHERE id = $1`, id)
	err := row.Scan(&sealed, &c.ID, &c.Kind, &c.Name, &c.Target, &c.Scopes, &c.State, &c.SecretVersion, &c.RotatedAt, &c.RevokedAt,
		&c.LastTestAt, &c.LastTestStatus, &c.LastTestDetail, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return Secret{}, Connection{}, persistence.MapError(err)
	}
	if c.State != "active" || sealed == nil {
		return Secret{}, c, ErrConnectionRevoked
	}
	plain, err := v.sealer.Open(sealed, connAAD(id))
	if err != nil {
		return Secret{}, c, fmt.Errorf("decifrar conexão %s: %w", id, err)
	}
	v.redactor.Register(string(plain))
	return NewSecret(string(plain)), c, nil
}

var refPattern = regexp.MustCompile(`^connection:([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

// ConnectionRef é a referência que workflows, agentes e prompts usam no lugar
// do valor: `connection:<uuid>`.
func ConnectionRef(id string) string { return "connection:" + id }

// ResolveRef resolve uma referência `connection:<uuid>` (injeção por referência).
func (v *Vault) ResolveRef(ctx context.Context, ref string) (Secret, Connection, error) {
	m := refPattern.FindStringSubmatch(ref)
	if m == nil {
		return Secret{}, Connection{}, fmt.Errorf("%w: referência inválida", persistence.ErrInvalidInput)
	}
	return v.Resolve(ctx, m[1])
}

// RecordTest grava o resultado de um teste (detalhe já sem segredo).
func (v *Vault) RecordTest(ctx context.Context, id, status, detail string) (Connection, error) {
	detail = v.redactor.Redact(detail)
	if len(detail) > 300 {
		detail = detail[:300]
	}
	return scanConnection(v.db.QueryRow(ctx, `
		UPDATE connections SET last_test_at = now(), last_test_status = $2, last_test_detail = $3, updated_at = now()
		WHERE id = $1 RETURNING `+connectionColumns, id, status, detail))
}

// LoadRedactor registra no redator todos os segredos ativos (chamar na subida
// da API, para que valores gravados antes do restart também sejam redigidos).
// Devolve quantas conexões ativas NÃO puderam ser decifradas — sinal típico de
// VAULT_KEY errada. Isso não é erro fatal: o login por código de recuperação
// não depende da chave e deve continuar possível.
func (v *Vault) LoadRedactor(ctx context.Context) (undecryptable int, err error) {
	rows, err := v.db.Query(ctx, `SELECT id, secret_ciphertext FROM connections WHERE state = 'active'`)
	if err != nil {
		return 0, persistence.MapError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var sealed []byte
		if err := rows.Scan(&id, &sealed); err != nil {
			return 0, persistence.MapError(err)
		}
		plain, err := v.sealer.Open(sealed, connAAD(id))
		if err != nil {
			undecryptable++
			continue
		}
		v.redactor.Register(string(plain))
	}
	return undecryptable, persistence.MapError(rows.Err())
}
