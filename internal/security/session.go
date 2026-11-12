// Package security reúne identidade (senha, TOTP, recuperação), sessões,
// limitação de tentativas, trilha de autenticação, cofre de conexões e
// redação de segredos (E04).
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
	// CSRFToken deve voltar no cabeçalho X-CSRF-Token em toda requisição que altera estado.
	CSRFToken string `json:"csrf_token"`
	// ReauthUntil: até quando ações sensíveis são aceitas sem nova reautenticação.
	ReauthUntil *time.Time `json:"reauth_until,omitempty"`
	// TOTPResetRequired: entrou com código de recuperação; o 2FA deve ser substituído.
	TOTPResetRequired bool `json:"totp_reset_required"`
}

// Meta descreve a origem de uma requisição para sessão e trilha de auditoria.
type Meta struct {
	IP            string
	UserAgent     string
	CorrelationID string
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// HashToken devolve o SHA-256 do token; só o hash é persistido.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// NewToken gera um token de 256 bits.
func NewToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

const (
	MethodTOTP     = "totp"
	MethodRecovery = "recovery"
)

func createSession(ctx context.Context, db persistence.DBTX, adminID string, ttl time.Duration, m Meta, method string) (string, Session, error) {
	token, err := NewToken()
	if err != nil {
		return "", Session{}, err
	}
	csrf, err := NewToken()
	if err != nil {
		return "", Session{}, err
	}
	s := Session{AdminID: adminID, CSRFToken: csrf}
	// Sessão nova já conta como reautenticação (acabou de provar senha + 2º fator).
	err = db.QueryRow(ctx, `
		INSERT INTO admin_sessions (admin_id, token_hash, expires_at, csrf_token, ip, user_agent, auth_method, reauth_at)
		VALUES ($1, $2, now() + make_interval(secs => $3), $4, $5, $6, $7, now())
		RETURNING id, expires_at`,
		adminID, HashToken(token), ttl.Seconds(), csrf, clip(m.IP, 64), clip(m.UserAgent, 200), method).Scan(&s.ID, &s.ExpiresAt)
	if err != nil {
		return "", Session{}, persistence.MapError(err)
	}
	return token, s, nil
}

// CreateSession persiste uma sessão para o administrador e devolve o token em claro
// (única vez em que ele existe fora do cliente). O fluxo de login usa Auth, que
// só chama isto depois do segundo fator; esta função existe para testes e para o
// código interno que já provou a identidade.
func CreateSession(ctx context.Context, db persistence.DBTX, adminID string, ttl time.Duration) (string, Session, error) {
	token, s, err := createSession(ctx, db, adminID, ttl, Meta{}, MethodTOTP)
	if err != nil {
		return "", Session{}, err
	}
	return token, s, nil
}

// Authenticate valida o token: existente, não expirado e não revogado.
// reauthWindow define quanto tempo uma reautenticação vale.
func Authenticate(ctx context.Context, db persistence.DBTX, token string, reauthWindow time.Duration) (Session, error) {
	if token == "" || len(token) > 256 {
		return Session{}, ErrNoSession
	}
	var s Session
	var reauthAt *time.Time
	err := db.QueryRow(ctx, `
		SELECT s.id, s.admin_id, a.username, s.expires_at, s.csrf_token, s.reauth_at, a.totp_reset_required
		FROM admin_sessions s JOIN admins a ON a.id = s.admin_id
		WHERE s.token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > now()`,
		HashToken(token)).Scan(&s.ID, &s.AdminID, &s.Username, &s.ExpiresAt, &s.CSRFToken, &reauthAt, &s.TOTPResetRequired)
	if err != nil {
		if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
			return Session{}, ErrNoSession
		}
		return Session{}, persistence.MapError(err)
	}
	if reauthAt != nil {
		until := reauthAt.Add(reauthWindow)
		if until.After(time.Now()) {
			s.ReauthUntil = &until
		}
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
