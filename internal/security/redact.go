package security

import (
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
)

const redacted = "[REDACTED]"

// Secret embrulha um valor sensível. Qualquer formatação, serialização JSON ou
// log produz [REDACTED]; só Reveal devolve o valor, e só deve ser chamado no
// ponto de uso (cliente HTTP autorizado, processo filho, etc.).
type Secret struct{ v string }

// NewSecret cria um Secret.
func NewSecret(v string) Secret { return Secret{v: v} }

// Reveal devolve o valor em claro.
func (s Secret) Reveal() string { return s.v }

// IsZero informa se não há valor.
func (s Secret) IsZero() bool { return s.v == "" }

func (Secret) String() string               { return redacted }
func (Secret) GoString() string             { return redacted }
func (Secret) MarshalText() ([]byte, error) { return []byte(redacted), nil }
func (Secret) MarshalJSON() ([]byte, error) { return []byte(`"` + redacted + `"`), nil }

// LogValue implementa slog.LogValuer.
func (Secret) LogValue() slog.Value { return slog.StringValue(redacted) }

// Padrões de credenciais conhecidas, redigidos mesmo quando o valor exato não foi registrado.
var tokenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`github_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]{10,}`),
	regexp.MustCompile(`(?i)\bbasic\s+[A-Za-z0-9+/=]{12,}`),
	regexp.MustCompile(`sk-[A-Za-z0-9_-]{20,}`),
	regexp.MustCompile(`pk_[0-9]+_[A-Z0-9]{20,}`),
	regexp.MustCompile(`ATATT[A-Za-z0-9_=-]{20,}`),
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`),
}

// Redactor substitui segredos em texto: os valores exatos registrados (inclusive
// versões antigas após rotação, enquanto o processo viver) e padrões conhecidos.
type Redactor struct {
	mu     sync.RWMutex
	values map[string]struct{}
	sorted []string // mais longos primeiro, para não deixar sufixo exposto
}

// NewRedactor cria um redator vazio.
func NewRedactor() *Redactor { return &Redactor{values: map[string]struct{}{}} }

// minRedactLen evita redigir fragmentos curtos que colidiriam com texto comum.
const minRedactLen = 6

// Register passa a redigir o valor. Nunca é removido (valor antigo segue protegido).
func (r *Redactor) Register(secret string) {
	if r == nil || len(secret) < minRedactLen {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[secret]; ok {
		return
	}
	r.values[secret] = struct{}{}
	r.sorted = append(r.sorted, secret)
	sort.Slice(r.sorted, func(i, j int) bool { return len(r.sorted[i]) > len(r.sorted[j]) })
}

// Redact devolve s sem nenhum segredo conhecido.
func (r *Redactor) Redact(s string) string {
	if r == nil || s == "" {
		return s
	}
	r.mu.RLock()
	for _, v := range r.sorted {
		if strings.Contains(s, v) {
			s = strings.ReplaceAll(s, v, redacted)
		}
	}
	r.mu.RUnlock()
	for _, p := range tokenPatterns {
		s = p.ReplaceAllString(s, redacted)
	}
	return s
}

// RedactBytes é Redact para []byte.
func (r *Redactor) RedactBytes(b []byte) []byte {
	if r == nil {
		return b
	}
	return []byte(r.Redact(string(b)))
}

// RedactHeader redige valores de cabeçalhos HTTP (ex.: ao registrar uma chamada de saída).
func (r *Redactor) RedactHeader(h http.Header) http.Header {
	out := http.Header{}
	for k, vs := range h {
		lk := strings.ToLower(k)
		for _, v := range vs {
			if lk == "authorization" || lk == "cookie" || lk == "set-cookie" || lk == "x-csrf-token" || strings.Contains(lk, "token") || strings.Contains(lk, "key") {
				out.Add(k, redacted)
			} else {
				out.Add(k, r.Redact(v))
			}
		}
	}
	return out
}
