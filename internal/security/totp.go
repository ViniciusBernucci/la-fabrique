package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // TOTP (RFC 6238) usa HMAC-SHA1 por compatibilidade com apps autenticadores.
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	totpPeriod = 30
	totpDigits = 6
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewTOTPSecret gera um segredo de 160 bits.
func NewTOTPSecret() ([]byte, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// EncodeTOTPSecret devolve o segredo em base32 (formato dos apps autenticadores).
func EncodeTOTPSecret(secret []byte) string { return b32.EncodeToString(secret) }

// OTPAuthURI monta o URI otpauth:// para QR/entrada manual.
func OTPAuthURI(issuer, account string, secret []byte) string {
	label := url.PathEscape(issuer + ":" + account)
	q := url.Values{}
	q.Set("secret", EncodeTOTPSecret(secret))
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(totpDigits))
	q.Set("period", fmt.Sprint(totpPeriod))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

func totpStep(t time.Time) int64 { return t.Unix() / totpPeriod }

func totpCode(secret []byte, step int64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", totpDigits, bin%1_000_000)
}

// NormalizeCode remove espaços e hífens de um código digitado.
func NormalizeCode(code string) string {
	return strings.NewReplacer(" ", "", "-", "").Replace(strings.TrimSpace(code))
}

// MatchTOTP devolve o passo (30 s) em que o código confere, tolerando ±1 passo
// de relógio. O chamador deve registrar o passo no banco para impedir replay.
func MatchTOTP(secret []byte, code string, now time.Time) (int64, bool) {
	code = NormalizeCode(code)
	if len(code) != totpDigits {
		return 0, false
	}
	cur := totpStep(now)
	matched := int64(0)
	ok := false
	for _, step := range []int64{cur - 1, cur, cur + 1} {
		if subtle.ConstantTimeCompare([]byte(totpCode(secret, step)), []byte(code)) == 1 {
			matched, ok = step, true
		}
	}
	return matched, ok
}

// TOTPCodeAt devolve o código válido no instante t (usado por testes e ferramentas de diagnóstico).
func TOTPCodeAt(secret []byte, t time.Time) string { return totpCode(secret, totpStep(t)) }
