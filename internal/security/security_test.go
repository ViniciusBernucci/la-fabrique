package security

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPasswordHashingAndPolicy(t *testing.T) {
	h, err := HashPassword("uma senha longa e boa")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Fatalf("formato PHC inesperado: %s", h)
	}
	if !VerifyPassword(h, "uma senha longa e boa") || VerifyPassword(h, "outra senha qualquer") {
		t.Fatal("verificação incorreta")
	}
	h2, _ := HashPassword("uma senha longa e boa")
	if h == h2 {
		t.Fatal("salt deve ser aleatório")
	}
	for _, bad := range []string{"", "$argon2id$", "$argon2id$v=19$m=99999999,t=1,p=1$AAAA$AAAA", "texto-puro", "dev-auth-no-password"} {
		if VerifyPassword(bad, "x") {
			t.Fatalf("hash inválido nunca valida: %q", bad)
		}
	}
	if err := ValidatePassword("curta"); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("senha curta: %v", err)
	}
	if err := ValidatePassword(strings.Repeat("a", 129)); !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("senha longa demais: %v", err)
	}
	if err := ValidatePassword("doze caracteres"); err != nil {
		t.Fatal(err)
	}
}

// Vetor de teste do RFC 6238 (SHA-1, segredo ASCII "12345678901234567890").
func TestTOTPMatchesRFC6238(t *testing.T) {
	secret := []byte("12345678901234567890")
	if got := totpCode(secret, 59/30); got != "287082" {
		t.Fatalf("T=59: %s (esperado 287082)", got)
	}
	if got := totpCode(secret, 1111111109/30); got != "081804" {
		t.Fatalf("T=1111111109: %s (esperado 081804)", got)
	}
	now := time.Unix(1111111109, 0)
	step, ok := MatchTOTP(secret, "081 804", now)
	if !ok || step != 1111111109/30 {
		t.Fatalf("deve aceitar com espaço: %d %v", step, ok)
	}
	// Tolera ±1 passo de relógio, mas não 2.
	if _, ok := MatchTOTP(secret, "081804", now.Add(30*time.Second)); !ok {
		t.Fatal("deveria tolerar +1 passo")
	}
	if _, ok := MatchTOTP(secret, "081804", now.Add(90*time.Second)); ok {
		t.Fatal("não deveria aceitar código 3 passos depois")
	}
	for _, bad := range []string{"", "12345", "1234567", "abcdef"} {
		if _, ok := MatchTOTP(secret, bad, now); ok {
			t.Fatalf("formato inválido aceito: %q", bad)
		}
	}
}

func TestOTPAuthURI(t *testing.T) {
	uri := OTPAuthURI("La fabrique", "admin", []byte("12345678901234567890"))
	for _, want := range []string{"otpauth://totp/La%20fabrique:admin?", "secret=GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", "issuer=La+fabrique", "digits=6", "period=30"} {
		if !strings.Contains(uri, want) {
			t.Errorf("URI sem %q: %s", want, uri)
		}
	}
}

func TestSealerRoundTripAADAndTamper(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	s, err := NewSealer(key, "k1")
	if err != nil {
		t.Fatal(err)
	}
	sealed, _ := s.Seal([]byte("segredo-xyz"), []byte("connection:1"))
	if bytes.Contains(sealed, []byte("segredo-xyz")) {
		t.Fatal("texto claro no cifrado")
	}
	if got, err := s.Open(sealed, []byte("connection:1")); err != nil || string(got) != "segredo-xyz" {
		t.Fatalf("round trip: %q %v", got, err)
	}
	if _, err := s.Open(sealed, []byte("connection:2")); !errors.Is(err, ErrSealed) {
		t.Fatal("AAD diferente deve falhar")
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := s.Open(sealed, []byte("connection:1")); !errors.Is(err, ErrSealed) {
		t.Fatal("adulteração deve falhar")
	}
	other, _ := NewSealer([]byte("fedcba9876543210fedcba9876543210"), "k2")
	fresh, _ := s.Seal([]byte("x"), nil)
	if _, err := other.Open(fresh, nil); !errors.Is(err, ErrSealed) {
		t.Fatal("outra chave deve falhar")
	}
	if _, err := NewSealer([]byte("curta"), "k"); err == nil {
		t.Fatal("chave curta deve ser recusada")
	}
	a, _ := s.Seal([]byte("x"), nil)
	b, _ := s.Seal([]byte("x"), nil)
	if bytes.Equal(a, b) {
		t.Fatal("nonce deve ser aleatório")
	}
}

func TestRecoveryCodes(t *testing.T) {
	codes, err := newRecoveryCodes(10)
	if err != nil || len(codes) != 10 {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range codes {
		if len(c) != 19 || strings.Count(c, "-") != 3 || seen[c] {
			t.Fatalf("código inválido ou repetido: %q", c)
		}
		seen[c] = true
	}
	c := codes[0]
	variants := []string{strings.ToLower(c), strings.ReplaceAll(c, "-", ""), " " + c + " ", strings.ReplaceAll(c, "-", " ")}
	for _, v := range variants {
		if !bytes.Equal(hashRecoveryCode(v), hashRecoveryCode(c)) {
			t.Fatalf("variante de digitação deve gerar o mesmo hash: %q", v)
		}
	}
	if bytes.Equal(hashRecoveryCode(codes[0]), hashRecoveryCode(codes[1])) {
		t.Fatal("códigos distintos, hashes distintos")
	}
}

const sentinel = "sentinel-7c2e91d4-valor-secreto"

func TestSecretNeverFormatsItsValue(t *testing.T) {
	s := NewSecret(sentinel)
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	log.Info("teste", "s", s, "grupo", slog.GroupValue(slog.Any("x", s)))
	j, _ := json.Marshal(map[string]any{"s": s})
	outs := []string{
		fmt.Sprint(s), fmt.Sprintf("%v %+v %#v %s %q", s, s, s, s, s), string(j), buf.String(),
		fmt.Sprint(struct{ S Secret }{s}), fmt.Sprintf("%+v", &struct{ S Secret }{s}),
	}
	for _, o := range outs {
		if strings.Contains(o, sentinel) {
			t.Fatalf("Secret vazou o valor: %s", o)
		}
	}
	if s.Reveal() != sentinel {
		t.Fatal("Reveal deve devolver o valor")
	}
}

func TestRedactor(t *testing.T) {
	r := NewRedactor()
	r.Register(sentinel)
	r.Register(sentinel + "-longo")
	r.Register("ab") // curto demais: ignorado
	in := "falha com " + sentinel + " e " + sentinel + "-longo; ab continua"
	out := r.Redact(in)
	if strings.Contains(out, sentinel) || !strings.Contains(out, "ab continua") {
		t.Fatalf("redação incorreta: %s", out)
	}
	if strings.Contains(out, "-longo") {
		t.Fatalf("o valor mais longo deve ser redigido por inteiro: %s", out)
	}
	for _, tc := range []string{
		"token ghp_abcdefghijklmnopqrstuvwxyz0123456789",
		"github_pat_11AAAAAAA0abcdefghijklmnop_qrstuvwxyz",
		"Authorization: Bearer abc.def-ghi_jkl123456",
		"Authorization: Basic dXNlcjpwYXNzd29yZDEyMw==",
		"pk_12345678_ABCDEFGHIJKLMNOPQRSTUVWXYZ0",
		"ATATT3xFfGF0abcdefghijklmnopqrstuvwx",
		"-----BEGIN OPENSSH PRIVATE KEY-----",
	} {
		if got := r.Redact(tc); !strings.Contains(got, redacted) || got == tc {
			t.Errorf("padrão não redigido: %q -> %q", tc, got)
		}
	}
	// Valor antigo (pós-rotação) continua protegido: Register nunca remove.
	r.Register("valor-antigo-123")
	r.Register("valor-novo-456")
	if strings.Contains(r.Redact("valor-antigo-123"), "antigo") {
		t.Fatal("valor anterior deve seguir redigido")
	}
	var nilR *Redactor
	if nilR.Redact("x") != "x" {
		t.Fatal("redator nulo é passagem direta")
	}
	h := r.RedactHeader(http.Header{"Authorization": {"Bearer " + sentinel}, "X-Api-Key": {"k"}, "Accept": {"text/" + sentinel}})
	if strings.Contains(fmt.Sprint(h), sentinel) || h.Get("Authorization") != redacted || h.Get("X-Api-Key") != redacted {
		t.Fatalf("cabeçalhos: %v", h)
	}
}

func TestConnectionRef(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	if ConnectionRef(id) != "connection:"+id || !refPattern.MatchString(ConnectionRef(id)) {
		t.Fatal("referência malformada")
	}
	for _, bad := range []string{"connection:x", "secret:" + id, id, "connection:" + id + "/extra"} {
		if refPattern.MatchString(bad) {
			t.Fatalf("referência inválida aceita: %q", bad)
		}
	}
}
