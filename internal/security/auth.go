package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// DB é o que a autenticação precisa do banco (*pgxpool.Pool satisfaz).
type DB interface {
	persistence.DBTX
	Begin(ctx context.Context) (pgx.Tx, error)
}

// Erros de autenticação. Mensagens ao cliente são fixas e genéricas (ver httpapi).
var (
	ErrInvalidCredentials = errors.New("credenciais inválidas")
	ErrInvalidCode        = errors.New("código inválido")
	ErrChallenge          = errors.New("desafio de login inválido ou expirado")
	ErrTOTPState          = errors.New("estado do segundo fator incompatível com a operação")
	ErrAdminExists        = errors.New("já existe um administrador")
	ErrInvalidUsername    = errors.New("usuário inválido")
)

// LockedError indica bloqueio temporário por excesso de tentativas.
type LockedError struct{ RetryAfter time.Duration }

func (e *LockedError) Error() string { return "muitas tentativas; tente mais tarde" }

// AuthConfig reúne prazos e limites.
type AuthConfig struct {
	SessionTTL         time.Duration
	ChallengeTTL       time.Duration
	ReauthWindow       time.Duration
	Issuer             string
	AccountMaxFailures int
	IPMaxFailures      int
	FailureWindow      time.Duration
	LockDuration       time.Duration
}

func (c AuthConfig) withDefaults() AuthConfig {
	def := func(d *time.Duration, v time.Duration) {
		if *d <= 0 {
			*d = v
		}
	}
	def(&c.SessionTTL, 12*time.Hour)
	def(&c.ChallengeTTL, 10*time.Minute)
	def(&c.ReauthWindow, 5*time.Minute)
	def(&c.FailureWindow, 15*time.Minute)
	def(&c.LockDuration, 15*time.Minute)
	if c.Issuer == "" {
		c.Issuer = "La fabrique"
	}
	if c.AccountMaxFailures <= 0 {
		c.AccountMaxFailures = 5
	}
	if c.IPMaxFailures <= 0 {
		c.IPMaxFailures = 20
	}
	return c
}

const (
	challengeMaxAttempts = 5
	recoveryCodeCount    = 10
	maxPasswordBytes     = 1024
)

// Auth implementa login com senha + TOTP, recuperação, reautenticação,
// revogação de sessões e a trilha de autenticação.
type Auth struct {
	db     DB
	sealer *Sealer
	cfg    AuthConfig
	log    *slog.Logger
	now    func() time.Time
}

// NewAuth cria o serviço. log pode ser nil.
func NewAuth(db DB, sealer *Sealer, cfg AuthConfig, log *slog.Logger) *Auth {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	return &Auth{db: db, sealer: sealer, cfg: cfg.withDefaults(), log: log, now: time.Now}
}

// SetClock troca o relógio usado para validar códigos TOTP (testes).
func (a *Auth) SetClock(fn func() time.Time) { a.now = fn }

// ReauthWindow é quanto tempo uma reautenticação vale.
func (a *Auth) ReauthWindow() time.Duration { return a.cfg.ReauthWindow }

// SessionTTL é a validade absoluta de uma sessão.
func (a *Auth) SessionTTL() time.Duration { return a.cfg.SessionTTL }

var usernamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{2,63}$`)

func normalizeUsername(u string) string { return strings.ToLower(strings.TrimSpace(u)) }

// --- Trilha de autenticação -------------------------------------------------

// Event é uma linha da trilha de autenticação.
type Event struct {
	ID        int64           `json:"id"`
	Type      string          `json:"type"`
	IP        string          `json:"ip"`
	UserAgent string          `json:"user_agent"`
	Detail    json.RawMessage `json:"detail"`
	CreatedAt time.Time       `json:"created_at"`
}

// Audit grava na trilha. Nunca receba aqui senha, código, token ou nome tentado.
// Falha de auditoria é logada e não derruba a operação.
func (a *Auth) Audit(ctx context.Context, adminID, typ string, m Meta, detail map[string]any) {
	if detail == nil {
		detail = map[string]any{}
	}
	raw, _ := json.Marshal(detail)
	var admin any
	if adminID != "" {
		admin = adminID
	}
	_, err := a.db.Exec(context.WithoutCancel(ctx), `
		INSERT INTO auth_events (admin_id, type, ip, user_agent, correlation_id, detail)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)`,
		admin, typ, clip(m.IP, 64), clip(m.UserAgent, 200), clip(m.CorrelationID, 64), string(raw))
	if err != nil {
		a.log.Error("falha ao gravar trilha de autenticação", "type", typ, "error", err.Error())
	}
}

// ListEvents devolve os eventos mais recentes primeiro.
func (a *Auth) ListEvents(ctx context.Context, limit int) ([]Event, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	rows, err := a.db.Query(ctx, `
		SELECT id, type, ip, user_agent, detail, created_at FROM auth_events
		ORDER BY created_at DESC, id DESC LIMIT $1`, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []Event{}
	for rows.Next() {
		var e Event
		var detail []byte
		if err := rows.Scan(&e.ID, &e.Type, &e.IP, &e.UserAgent, &detail, &e.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		e.Detail = detail
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}

// --- Limitação de tentativas -------------------------------------------------

func acctKey(username string) string { return "acct:" + clip(normalizeUsername(username), 64) }
func ipKey(ip string) string         { return "ip:" + clip(ip, 64) }

// checkLock devolve *LockedError se alguma chave estiver bloqueada.
func (a *Auth) checkLock(ctx context.Context, keys ...string) error {
	var longest time.Duration
	for _, k := range keys {
		var secs *float64
		err := a.db.QueryRow(ctx, `
			SELECT EXTRACT(EPOCH FROM (locked_until - now()))::float8 FROM auth_throttle
			WHERE key = $1 AND locked_until > now()`, k).Scan(&secs)
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return persistence.MapError(err)
		}
		if secs != nil {
			if d := time.Duration(*secs * float64(time.Second)); d > longest {
				longest = d
			}
		}
	}
	if longest > 0 {
		return &LockedError{RetryAfter: longest}
	}
	return nil
}

func (a *Auth) registerFailure(ctx context.Context, key string, max int) {
	ctx = context.WithoutCancel(ctx)
	_, err := a.db.Exec(ctx, `
		INSERT INTO auth_throttle (key, failures, window_started_at) VALUES ($1, 1, now())
		ON CONFLICT (key) DO UPDATE SET
		  failures = CASE WHEN auth_throttle.window_started_at < now() - make_interval(secs => $2)
		                   AND (auth_throttle.locked_until IS NULL OR auth_throttle.locked_until < now())
		                  THEN 1 ELSE auth_throttle.failures + 1 END,
		  window_started_at = CASE WHEN auth_throttle.window_started_at < now() - make_interval(secs => $2)
		                   AND (auth_throttle.locked_until IS NULL OR auth_throttle.locked_until < now())
		                  THEN now() ELSE auth_throttle.window_started_at END`,
		key, a.cfg.FailureWindow.Seconds())
	if err == nil {
		_, err = a.db.Exec(ctx, `
			UPDATE auth_throttle SET locked_until = now() + make_interval(secs => $3)
			WHERE key = $1 AND failures >= $2 AND (locked_until IS NULL OR locked_until < now())`,
			key, max, a.cfg.LockDuration.Seconds())
	}
	if err != nil {
		a.log.Error("falha ao registrar tentativa", "error", err.Error())
	}
}

func (a *Auth) failBoth(ctx context.Context, username string, m Meta) {
	a.registerFailure(ctx, acctKey(username), a.cfg.AccountMaxFailures)
	a.registerFailure(ctx, ipKey(m.IP), a.cfg.IPMaxFailures)
}

func (a *Auth) resetAccount(ctx context.Context, username string) {
	_, _ = a.db.Exec(context.WithoutCancel(ctx), `DELETE FROM auth_throttle WHERE key = $1`, acctKey(username))
}

// --- Administrador único (bootstrap) -----------------------------------------

// CreateAdmin cria a conta única. Falha com ErrAdminExists se já houver uma.
func (a *Auth) CreateAdmin(ctx context.Context, username, password string) (string, error) {
	username = normalizeUsername(username)
	if !usernamePattern.MatchString(username) {
		return "", fmt.Errorf("%w: use 3 a 64 caracteres [a-z0-9._-], começando por letra ou número", ErrInvalidUsername)
	}
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", err
	}
	var id string
	err = a.db.QueryRow(ctx, `INSERT INTO admins (username, password_hash) VALUES ($1, $2) RETURNING id`, username, hash).Scan(&id)
	if err != nil {
		if errors.Is(persistence.MapError(err), persistence.ErrDuplicate) {
			return "", ErrAdminExists
		}
		return "", persistence.MapError(err)
	}
	a.Audit(ctx, id, "admin_created", Meta{}, nil)
	return id, nil
}

// markFreshReauth indica na sessão recém-criada que ela já conta como
// reautenticação (a coluna reauth_at é gravada na criação).
func (a *Auth) markFreshReauth(s *Session) {
	until := time.Now().Add(a.cfg.ReauthWindow)
	s.ReauthUntil = &until
}

// --- Login em duas etapas ----------------------------------------------------

// Estágios devolvidos após a senha.
const (
	StageTOTP   = "totp_required"
	StageEnroll = "totp_enrollment_required"
)

// StartLogin valida a senha e abre um desafio de curta duração. NÃO abre sessão:
// a sessão só existe depois do segundo fator.
func (a *Auth) StartLogin(ctx context.Context, username, password string, m Meta) (challengeToken, stage string, err error) {
	username = normalizeUsername(username)
	if err := a.checkLock(ctx, acctKey(username), ipKey(m.IP)); err != nil {
		a.Audit(ctx, "", "login_blocked", m, nil)
		return "", "", err
	}
	var id, hash string
	var confirmed *time.Time
	qerr := a.db.QueryRow(ctx, `SELECT id, password_hash, totp_confirmed_at FROM admins WHERE username = $1`, username).Scan(&id, &hash, &confirmed)
	switch {
	case errors.Is(qerr, pgx.ErrNoRows):
		burnPasswordTime(password)
		a.failBoth(ctx, username, m)
		a.Audit(ctx, "", "login_failed", m, map[string]any{"reason": "credenciais"})
		return "", "", ErrInvalidCredentials
	case qerr != nil:
		return "", "", persistence.MapError(qerr)
	}
	if len(password) > maxPasswordBytes || !VerifyPassword(hash, password) {
		a.failBoth(ctx, username, m)
		a.Audit(ctx, id, "login_failed", m, map[string]any{"reason": "credenciais"})
		return "", "", ErrInvalidCredentials
	}
	token, err := NewToken()
	if err != nil {
		return "", "", err
	}
	_, err = a.db.Exec(ctx, `
		INSERT INTO auth_challenges (admin_id, token_hash, expires_at)
		VALUES ($1, $2, now() + make_interval(secs => $3))`, id, HashToken(token), a.cfg.ChallengeTTL.Seconds())
	if err != nil {
		return "", "", persistence.MapError(err)
	}
	a.Audit(ctx, id, "password_ok", m, nil)
	if confirmed == nil {
		return token, StageEnroll, nil
	}
	return token, StageTOTP, nil
}

type challenge struct {
	ID        string
	AdminID   string
	Username  string
	Confirmed bool
}

func (a *Auth) loadChallenge(ctx context.Context, token string) (challenge, error) {
	var c challenge
	if token == "" || len(token) > 256 {
		return c, ErrChallenge
	}
	err := a.db.QueryRow(ctx, `
		SELECT c.id, c.admin_id, ad.username, ad.totp_confirmed_at IS NOT NULL
		FROM auth_challenges c JOIN admins ad ON ad.id = c.admin_id
		WHERE c.token_hash = $1 AND c.consumed_at IS NULL AND c.expires_at > now() AND c.attempts < $2`,
		HashToken(token), challengeMaxAttempts).Scan(&c.ID, &c.AdminID, &c.Username, &c.Confirmed)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrChallenge
	}
	return c, persistence.MapError(err)
}

func (a *Auth) bumpAttempts(ctx context.Context, challengeID string) {
	_, _ = a.db.Exec(context.WithoutCancel(ctx), `UPDATE auth_challenges SET attempts = attempts + 1 WHERE id = $1`, challengeID)
}

// consumeChallenge marca o desafio como usado; só uma requisição consegue.
func consumeChallenge(ctx context.Context, tx pgx.Tx, id string) error {
	tag, err := tx.Exec(ctx, `UPDATE auth_challenges SET consumed_at = now() WHERE id = $1 AND consumed_at IS NULL AND expires_at > now()`, id)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrChallenge
	}
	return nil
}

func totpAAD(adminID string) []byte        { return []byte("admin-totp:" + adminID) }
func totpPendingAAD(adminID string) []byte { return []byte("admin-totp-pending:" + adminID) }

func (a *Auth) secondFactorFailed(ctx context.Context, c challenge, m Meta, typ string) {
	a.bumpAttempts(ctx, c.ID)
	a.failBoth(ctx, c.Username, m)
	a.Audit(ctx, c.AdminID, typ, m, nil)
}

// VerifyTOTP conclui o login com o código do app autenticador.
func (a *Auth) VerifyTOTP(ctx context.Context, challengeToken, code string, m Meta) (string, Session, error) {
	c, err := a.loadChallenge(ctx, challengeToken)
	if err != nil {
		return "", Session{}, err
	}
	if !c.Confirmed {
		return "", Session{}, ErrTOTPState
	}
	if err := a.checkLock(ctx, acctKey(c.Username), ipKey(m.IP)); err != nil {
		a.Audit(ctx, c.AdminID, "login_blocked", m, nil)
		return "", Session{}, err
	}
	var sealed []byte
	if err := a.db.QueryRow(ctx, `SELECT totp_secret FROM admins WHERE id = $1`, c.AdminID).Scan(&sealed); err != nil {
		return "", Session{}, persistence.MapError(err)
	}
	secret, err := a.sealer.Open(sealed, totpAAD(c.AdminID))
	if err != nil {
		return "", Session{}, fmt.Errorf("abrir segredo TOTP: %w", err)
	}
	step, ok := MatchTOTP(secret, code, a.now())
	if !ok {
		a.secondFactorFailed(ctx, c, m, "totp_failed")
		return "", Session{}, ErrInvalidCode
	}
	var token string
	var sess Session
	err = persistence.WithTx(ctx, a.db, func(tx pgx.Tx) error {
		// Replay: o passo só avança. Dois pedidos com o mesmo código não passam ambos.
		tag, err := tx.Exec(ctx, `UPDATE admins SET totp_last_step = $2 WHERE id = $1 AND totp_last_step < $2`, c.AdminID, step)
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return ErrInvalidCode
		}
		if err := consumeChallenge(ctx, tx, c.ID); err != nil {
			return err
		}
		token, sess, err = createSession(ctx, tx, c.AdminID, a.cfg.SessionTTL, m, MethodTOTP)
		return err
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCode) {
			a.secondFactorFailed(ctx, c, m, "totp_replay")
		}
		return "", Session{}, err
	}
	sess.Username = c.Username
	a.markFreshReauth(&sess)
	a.resetAccount(ctx, c.Username)
	a.Audit(ctx, c.AdminID, "login_success", m, map[string]any{"method": MethodTOTP})
	return token, sess, nil
}

// --- Códigos de recuperação ---------------------------------------------------

func hashRecoveryCode(code string) []byte {
	norm := strings.ToUpper(NormalizeCode(code))
	sum := sha256.Sum256([]byte(norm))
	return sum[:]
}

// newRecoveryCodes devolve códigos de 80 bits no formato XXXX-XXXX-XXXX-XXXX.
func newRecoveryCodes(n int) ([]string, error) {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		b := make([]byte, 10)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		s := b32.EncodeToString(b)
		out = append(out, s[0:4]+"-"+s[4:8]+"-"+s[8:12]+"-"+s[12:16])
	}
	return out, nil
}

// replaceRecoveryCodes invalida os códigos anteriores e grava um novo conjunto.
func replaceRecoveryCodes(ctx context.Context, tx pgx.Tx, adminID string) ([]string, error) {
	codes, err := newRecoveryCodes(recoveryCodeCount)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM recovery_codes WHERE admin_id = $1`, adminID); err != nil {
		return nil, persistence.MapError(err)
	}
	for _, c := range codes {
		if _, err := tx.Exec(ctx, `INSERT INTO recovery_codes (admin_id, code_hash) VALUES ($1, $2)`, adminID, hashRecoveryCode(c)); err != nil {
			return nil, persistence.MapError(err)
		}
	}
	return codes, nil
}

// RemainingRecoveryCodes conta os códigos ainda não usados.
func (a *Auth) RemainingRecoveryCodes(ctx context.Context, adminID string) (int, error) {
	var n int
	err := a.db.QueryRow(ctx, `SELECT count(*) FROM recovery_codes WHERE admin_id = $1 AND used_at IS NULL`, adminID).Scan(&n)
	return n, persistence.MapError(err)
}

// UseRecoveryCode conclui o login com um código de uso único. O código é
// consumido atomicamente: requisições concorrentes com o mesmo código resultam
// em exatamente uma sessão. Depois disso o 2FA deve ser substituído.
func (a *Auth) UseRecoveryCode(ctx context.Context, challengeToken, code string, m Meta) (string, Session, error) {
	c, err := a.loadChallenge(ctx, challengeToken)
	if err != nil {
		return "", Session{}, err
	}
	if !c.Confirmed {
		return "", Session{}, ErrTOTPState
	}
	if err := a.checkLock(ctx, acctKey(c.Username), ipKey(m.IP)); err != nil {
		a.Audit(ctx, c.AdminID, "login_blocked", m, nil)
		return "", Session{}, err
	}
	var token string
	var sess Session
	err = persistence.WithTx(ctx, a.db, func(tx pgx.Tx) error {
		if err := consumeChallenge(ctx, tx, c.ID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE recovery_codes SET used_at = now()
			WHERE admin_id = $1 AND code_hash = $2 AND used_at IS NULL`, c.AdminID, hashRecoveryCode(code))
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return ErrInvalidCode
		}
		if _, err := tx.Exec(ctx, `UPDATE admins SET totp_reset_required = true WHERE id = $1`, c.AdminID); err != nil {
			return persistence.MapError(err)
		}
		token, sess, err = createSession(ctx, tx, c.AdminID, a.cfg.SessionTTL, m, MethodRecovery)
		return err
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCode) {
			a.secondFactorFailed(ctx, c, m, "recovery_failed")
		}
		return "", Session{}, err
	}
	sess.Username = c.Username
	sess.TOTPResetRequired = true
	a.markFreshReauth(&sess)
	a.resetAccount(ctx, c.Username)
	a.Audit(ctx, c.AdminID, "recovery_used", m, nil)
	a.Audit(ctx, c.AdminID, "login_success", m, map[string]any{"method": MethodRecovery})
	return token, sess, nil
}

// --- Onboarding do TOTP (primeiro acesso) ------------------------------------

// Enrollment é o material para cadastrar o app autenticador.
type Enrollment struct {
	Secret     string `json:"secret"`
	OTPAuthURI string `json:"otpauth_uri"`
}

func (a *Auth) beginTOTP(ctx context.Context, adminID, username string, requireUnconfirmed bool) (Enrollment, error) {
	secret, err := NewTOTPSecret()
	if err != nil {
		return Enrollment{}, err
	}
	sealed, err := a.sealer.Seal(secret, totpPendingAAD(adminID))
	if err != nil {
		return Enrollment{}, err
	}
	cond := "totp_confirmed_at IS NOT NULL"
	if requireUnconfirmed {
		cond = "totp_confirmed_at IS NULL"
	}
	tag, err := a.db.Exec(ctx, `UPDATE admins SET totp_pending_secret = $2 WHERE id = $1 AND `+cond, adminID, sealed)
	if err != nil {
		return Enrollment{}, persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return Enrollment{}, ErrTOTPState
	}
	return Enrollment{Secret: EncodeTOTPSecret(secret), OTPAuthURI: OTPAuthURI(a.cfg.Issuer, username, secret)}, nil
}

// BeginEnrollment gera o segredo do primeiro acesso (admin ainda sem 2FA).
func (a *Auth) BeginEnrollment(ctx context.Context, challengeToken string) (Enrollment, error) {
	c, err := a.loadChallenge(ctx, challengeToken)
	if err != nil {
		return Enrollment{}, err
	}
	if c.Confirmed {
		return Enrollment{}, ErrTOTPState
	}
	return a.beginTOTP(ctx, c.AdminID, c.Username, true)
}

func (a *Auth) pendingSecret(ctx context.Context, adminID string) ([]byte, error) {
	var sealed []byte
	if err := a.db.QueryRow(ctx, `SELECT totp_pending_secret FROM admins WHERE id = $1`, adminID).Scan(&sealed); err != nil {
		return nil, persistence.MapError(err)
	}
	if sealed == nil {
		return nil, ErrTOTPState
	}
	secret, err := a.sealer.Open(sealed, totpPendingAAD(adminID))
	if err != nil {
		return nil, fmt.Errorf("abrir segredo TOTP pendente: %w", err)
	}
	return secret, nil
}

// ConfirmEnrollment ativa o 2FA com o primeiro código, gera os códigos de
// recuperação (mostrados uma única vez) e abre a sessão.
func (a *Auth) ConfirmEnrollment(ctx context.Context, challengeToken, code string, m Meta) (string, Session, []string, error) {
	c, err := a.loadChallenge(ctx, challengeToken)
	if err != nil {
		return "", Session{}, nil, err
	}
	if c.Confirmed {
		return "", Session{}, nil, ErrTOTPState
	}
	if err := a.checkLock(ctx, acctKey(c.Username), ipKey(m.IP)); err != nil {
		return "", Session{}, nil, err
	}
	secret, err := a.pendingSecret(ctx, c.AdminID)
	if err != nil {
		return "", Session{}, nil, err
	}
	step, ok := MatchTOTP(secret, code, a.now())
	if !ok {
		a.secondFactorFailed(ctx, c, m, "totp_enroll_failed")
		return "", Session{}, nil, ErrInvalidCode
	}
	sealed, err := a.sealer.Seal(secret, totpAAD(c.AdminID))
	if err != nil {
		return "", Session{}, nil, err
	}
	var token string
	var sess Session
	var codes []string
	err = persistence.WithTx(ctx, a.db, func(tx pgx.Tx) error {
		if err := consumeChallenge(ctx, tx, c.ID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE admins SET totp_secret = $2, totp_pending_secret = NULL, totp_confirmed_at = now(),
			       totp_last_step = $3, totp_reset_required = false
			WHERE id = $1 AND totp_confirmed_at IS NULL`, c.AdminID, sealed, step)
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return ErrTOTPState
		}
		if codes, err = replaceRecoveryCodes(ctx, tx, c.AdminID); err != nil {
			return err
		}
		token, sess, err = createSession(ctx, tx, c.AdminID, a.cfg.SessionTTL, m, MethodTOTP)
		return err
	})
	if err != nil {
		return "", Session{}, nil, err
	}
	sess.Username = c.Username
	a.markFreshReauth(&sess)
	a.resetAccount(ctx, c.Username)
	a.Audit(ctx, c.AdminID, "totp_enrolled", m, nil)
	a.Audit(ctx, c.AdminID, "login_success", m, map[string]any{"method": MethodTOTP})
	return token, sess, codes, nil
}

// --- Substituição do segundo fator (admin autenticado + reautenticado) ----------

// BeginReplacement gera um novo segredo pendente; o atual segue valendo até a confirmação.
func (a *Auth) BeginReplacement(ctx context.Context, s Session) (Enrollment, error) {
	return a.beginTOTP(ctx, s.AdminID, s.Username, false)
}

// ConfirmReplacement troca o segredo, emite novos códigos de recuperação
// (invalidando os anteriores) e revoga as demais sessões.
func (a *Auth) ConfirmReplacement(ctx context.Context, s Session, code string, m Meta) ([]string, error) {
	if err := a.checkLock(ctx, acctKey(s.Username), ipKey(m.IP)); err != nil {
		return nil, err
	}
	secret, err := a.pendingSecret(ctx, s.AdminID)
	if err != nil {
		return nil, err
	}
	step, ok := MatchTOTP(secret, code, a.now())
	if !ok {
		a.failBoth(ctx, s.Username, m)
		a.Audit(ctx, s.AdminID, "totp_replace_failed", m, nil)
		return nil, ErrInvalidCode
	}
	sealed, err := a.sealer.Seal(secret, totpAAD(s.AdminID))
	if err != nil {
		return nil, err
	}
	var codes []string
	err = persistence.WithTx(ctx, a.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `
			UPDATE admins SET totp_secret = $2, totp_pending_secret = NULL, totp_last_step = $3, totp_reset_required = false
			WHERE id = $1 AND totp_pending_secret IS NOT NULL AND totp_confirmed_at IS NOT NULL`, s.AdminID, sealed, step)
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return ErrTOTPState
		}
		if codes, err = replaceRecoveryCodes(ctx, tx, s.AdminID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE admin_sessions SET revoked_at = now() WHERE admin_id = $1 AND id <> $2 AND revoked_at IS NULL`, s.AdminID, s.ID)
		return persistence.MapError(err)
	})
	if err != nil {
		return nil, err
	}
	a.Audit(ctx, s.AdminID, "totp_replaced", m, nil)
	return codes, nil
}

// --- Reautenticação -----------------------------------------------------------

// Reauthenticate confirma senha + código TOTP e abre a janela de ações sensíveis.
func (a *Auth) Reauthenticate(ctx context.Context, s Session, password, code string, m Meta) error {
	if err := a.checkLock(ctx, acctKey(s.Username), ipKey(m.IP)); err != nil {
		return err
	}
	var hash string
	var sealed []byte
	if err := a.db.QueryRow(ctx, `SELECT password_hash, totp_secret FROM admins WHERE id = $1`, s.AdminID).Scan(&hash, &sealed); err != nil {
		return persistence.MapError(err)
	}
	passOK := len(password) <= maxPasswordBytes && VerifyPassword(hash, password)
	var step int64
	codeOK := false
	if sealed != nil {
		secret, err := a.sealer.Open(sealed, totpAAD(s.AdminID))
		if err != nil {
			return fmt.Errorf("abrir segredo TOTP: %w", err)
		}
		step, codeOK = MatchTOTP(secret, code, a.now())
	}
	if !passOK || !codeOK {
		a.failBoth(ctx, s.Username, m)
		a.Audit(ctx, s.AdminID, "reauth_failed", m, nil)
		return ErrInvalidCredentials
	}
	err := persistence.WithTx(ctx, a.db, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE admins SET totp_last_step = $2 WHERE id = $1 AND totp_last_step < $2`, s.AdminID, step)
		if err != nil {
			return persistence.MapError(err)
		}
		if tag.RowsAffected() == 0 {
			return ErrInvalidCode
		}
		_, err = tx.Exec(ctx, `UPDATE admin_sessions SET reauth_at = now() WHERE id = $1 AND revoked_at IS NULL`, s.ID)
		return persistence.MapError(err)
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCode) {
			a.failBoth(ctx, s.Username, m)
			a.Audit(ctx, s.AdminID, "reauth_failed", m, map[string]any{"reason": "replay"})
		}
		return err
	}
	a.resetAccount(ctx, s.Username)
	a.Audit(ctx, s.AdminID, "reauth_ok", m, nil)
	return nil
}

// --- Sessões --------------------------------------------------------------------

// SessionInfo descreve uma sessão ativa na lista de sessões.
type SessionInfo struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	AuthMethod string    `json:"auth_method"`
	Current    bool      `json:"current"`
}

// ListSessions devolve as sessões ativas (não revogadas, não expiradas).
func (a *Auth) ListSessions(ctx context.Context, adminID, currentID string) ([]SessionInfo, error) {
	rows, err := a.db.Query(ctx, `
		SELECT id, created_at, expires_at, ip, user_agent, auth_method FROM admin_sessions
		WHERE admin_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY created_at DESC, id DESC`, adminID)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []SessionInfo{}
	for rows.Next() {
		var si SessionInfo
		if err := rows.Scan(&si.ID, &si.CreatedAt, &si.ExpiresAt, &si.IP, &si.UserAgent, &si.AuthMethod); err != nil {
			return nil, persistence.MapError(err)
		}
		si.Current = si.ID == currentID
		out = append(out, si)
	}
	return out, persistence.MapError(rows.Err())
}

// RevokeSession revoga uma sessão do próprio administrador.
func (a *Auth) RevokeSession(ctx context.Context, s Session, sessionID string, m Meta) error {
	tag, err := a.db.Exec(ctx, `UPDATE admin_sessions SET revoked_at = now() WHERE id = $1 AND admin_id = $2 AND revoked_at IS NULL`, sessionID, s.AdminID)
	if err != nil {
		return persistence.MapError(err)
	}
	if tag.RowsAffected() == 0 {
		return persistence.ErrNotFound
	}
	a.Audit(ctx, s.AdminID, "session_revoked", m, map[string]any{"session_id": sessionID, "self": sessionID == s.ID})
	return nil
}

// RevokeOthers revoga todas as sessões exceto a atual.
func (a *Auth) RevokeOthers(ctx context.Context, s Session, m Meta) (int64, error) {
	tag, err := a.db.Exec(ctx, `UPDATE admin_sessions SET revoked_at = now() WHERE admin_id = $1 AND id <> $2 AND revoked_at IS NULL`, s.AdminID, s.ID)
	if err != nil {
		return 0, persistence.MapError(err)
	}
	a.Audit(ctx, s.AdminID, "sessions_revoked_others", m, map[string]any{"count": tag.RowsAffected()})
	return tag.RowsAffected(), nil
}

// Logout revoga a sessão atual.
func (a *Auth) Logout(ctx context.Context, s Session, m Meta) error {
	if err := Revoke(ctx, a.db, s.ID); err != nil && !errors.Is(err, persistence.ErrNotFound) {
		return err
	}
	a.Audit(ctx, s.AdminID, "logout", m, nil)
	return nil
}
