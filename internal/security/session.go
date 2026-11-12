// Package security reúne identidade, sessões e (nos épicos seguintes) cofre,
// política de autorização e auditoria.
//
// Este arquivo cobre apenas o necessário para o servidor recusar acesso sem
// sessão (T03-03). Login, TOTP, recuperação, CSRF e reautenticação são do T04-01/T04-02.
package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// ErrNoSession indica sessão ausente, inválida, expirada ou revogada.
var ErrNoSession = errors.New("sessão inválida")

// Session é uma sessão administrativa válida.
type Session struct {
	ID        string    `json:"id"`
	AdminID   string    `json:"admin_id"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

// HashToken devolve o SHA-256 do token; só o hash é persistido.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// NewToken gera um token de sessão de 256 bits.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateSession persiste uma sessão para o administrador e devolve o token em claro
// (única vez em que ele existe fora do cliente).
func CreateSession(ctx context.Context, db persistence.DBTX, adminID string, ttl time.Duration) (string, Session, error) {
	token, err := NewToken()
	if err != nil {
		return "", Session{}, err
	}
	s := Session{AdminID: adminID}
	err = db.QueryRow(ctx, `
		INSERT INTO admin_sessions (admin_id, token_hash, expires_at)
		VALUES ($1, $2, now() + make_interval(secs => $3))
		RETURNING id, expires_at`, adminID, HashToken(token), ttl.Seconds()).Scan(&s.ID, &s.ExpiresAt)
	if err != nil {
		return "", Session{}, persistence.MapError(err)
	}
	return token, s, nil
}

// Authenticate valida o token: existente, não expirado e não revogado.
func Authenticate(ctx context.Context, db persistence.DBTX, token string) (Session, error) {
	if token == "" || len(token) > 256 {
		return Session{}, ErrNoSession
	}
	var s Session
	err := db.QueryRow(ctx, `
		SELECT s.id, s.admin_id, a.username, s.expires_at
		FROM admin_sessions s JOIN admins a ON a.id = s.admin_id
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > now()`,
		HashToken(token)).Scan(&s.ID, &s.AdminID, &s.Username, &s.ExpiresAt)
	if err != nil {
		if persistence.MapError(err) == persistence.ErrNotFound {
			return Session{}, ErrNoSession
		}
		return Session{}, persistence.MapError(err)
	}
	return s, nil
}

// Revoke revoga a sessão; a partir daqui Authenticate falha.
func Revoke(ctx context.Context, db persistence.DBTX, sessionID string) error {
	tag, err := db.Exec(ctx, `UPDATE admin_sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, sessionID)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return persistence.ErrNotFound
	}
	return nil
}
