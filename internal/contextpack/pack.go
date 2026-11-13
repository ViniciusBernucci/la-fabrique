package contextpack

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Tipos de fonte, na ordem em que aparecem no texto.
const (
	KindRequirement = "requirement"
	KindCriterion   = "criterion"
	KindDecision    = "decision"
	KindMemory      = "memory"
	KindFile        = "file"
	KindEvidence    = "evidence"
	KindOutput      = "output"
)

var kindOrder = map[string]int{KindRequirement: 0, KindCriterion: 1, KindDecision: 2, KindMemory: 3, KindFile: 4, KindEvidence: 5, KindOutput: 6}

// Motivos de omissão.
const (
	OmitLimit     = "limit"
	OmitEmpty     = "empty"
	OmitDuplicate = "duplicate"
)

// Erros.
var (
	ErrRequiredOmitted = errors.New("uma fonte obrigatória não cabe no limite de contexto")
	ErrSecretLeft      = errors.New("o contexto ainda contém um segredo reconhecível")
	ErrCorrupt         = errors.New("pacote de contexto corrompido: um hash não confere")
)

// FieldError é um erro de entrada.
type FieldError struct{ Field, Message string }

func (e *FieldError) Error() string { return e.Field + ": " + e.Message }
func (e *FieldError) Unwrap() error { return persistence.ErrInvalidInput }

// Source é uma fonte candidata ao contexto.
type Source struct {
	Kind     string
	Ref      string // identificador estável (arquivo, decisão, artefato…)
	Content  string
	Priority int  // maior entra primeiro
	Required bool // nunca descartada nem truncada
}

// Limits delimita o pacote.
type Limits struct {
	MaxTokens    int64 // limite do pacote inteiro (estimado)
	MaxItemBytes int   // limite por fonte não obrigatória; 0 = sem limite
}

// Item é a linha do manifesto de uma fonte.
type Item struct {
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	SHA256    string `json:"sha256,omitempty"` // do conteúdo incluído, já sem segredos
	Bytes     int    `json:"bytes"`
	Tokens    int64  `json:"tokens"`
	Included  bool   `json:"included"`
	Redacted  bool   `json:"redacted,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Required  bool   `json:"required,omitempty"`
	Reason    string `json:"reason,omitempty"` // motivo da omissão
}

// Manifest descreve o pacote: o que entrou, o que ficou de fora e por quê.
type Manifest struct {
	Version         int    `json:"version"`
	Method          string `json:"method"` // como os tokens foram contados
	LimitTokens     int64  `json:"limit_tokens"`
	EstimatedTokens int64  `json:"estimated_tokens"`
	ContentHash     string `json:"content_hash"`
	Items           []Item `json:"items"`
	Omitted         int    `json:"omitted"`
}

// Pack é o contexto pronto para a sessão.
type Pack struct {
	Manifest Manifest
	Content  string
}

func sha(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func canonical(v any) []byte {
	b, _ := json.Marshal(v)
	var g any
	if json.Unmarshal(b, &g) != nil {
		return b
	}
	out, _ := json.Marshal(g)
	return out
}

// Hash do manifesto.
func (m Manifest) Hash() string { return sha(string(canonical(m))) }

func section(kind, ref, content string) string {
	return "### " + kind + " " + ref + "\n" + content + "\n"
}

// Build compõe o pacote. Determinístico: mesmas fontes, mesmo limite, mesmo pacote.
func Build(sources []Source, lim Limits, red *security.Redactor) (Pack, error) {
	if lim.MaxTokens <= 0 {
		return Pack{}, &FieldError{"max_tokens", "informe o limite de tokens do contexto"}
	}
	type cand struct {
		Source
		idx      int
		text     string
		redacted bool
	}
	seen := map[string]bool{}
	var cands []cand
	items := make([]Item, 0, len(sources))
	for i, s := range sources {
		if _, ok := kindOrder[s.Kind]; !ok {
			return Pack{}, &FieldError{"kind", fmt.Sprintf("tipo de fonte desconhecido: %q", s.Kind)}
		}
		if strings.TrimSpace(s.Ref) == "" || len(s.Ref) > 500 || strings.ContainsAny(s.Ref, "\n\r") {
			return Pack{}, &FieldError{"ref", "referência da fonte inválida"}
		}
		key := s.Kind + "\x00" + s.Ref
		clean := red.Redact(s.Content)
		it := Item{Kind: s.Kind, Ref: s.Ref, Required: s.Required, Redacted: clean != s.Content}
		switch {
		case seen[key]:
			it.Reason = OmitDuplicate
			items = append(items, it)
			continue
		case strings.TrimSpace(clean) == "":
			it.Reason = OmitEmpty
			seen[key] = true
			items = append(items, it)
			continue
		}
		seen[key] = true
		cands = append(cands, cand{Source: s, idx: i, text: clean, redacted: it.Redacted})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].Required != cands[j].Required {
			return cands[i].Required
		}
		if cands[i].Priority != cands[j].Priority {
			return cands[i].Priority > cands[j].Priority
		}
		return cands[i].idx < cands[j].idx
	})

	var used int64
	type kept struct {
		it   Item
		text string
	}
	var keep []kept
	for _, c := range cands {
		text, truncated := c.text, false
		if !c.Required && lim.MaxItemBytes > 0 && len(text) > lim.MaxItemBytes {
			text, truncated = cutUTF8(text, lim.MaxItemBytes), true
		}
		body := section(c.Kind, c.Ref, text)
		tok := budget.EstimateTokens(body)
		it := Item{Kind: c.Kind, Ref: c.Ref, Required: c.Required, Redacted: c.redacted, Truncated: truncated, Bytes: len(text), Tokens: tok}
		if used+tok > lim.MaxTokens {
			if c.Required {
				return Pack{}, fmt.Errorf("%w: %s %s (%d tokens, restam %d)", ErrRequiredOmitted, c.Kind, c.Ref, tok, lim.MaxTokens-used)
			}
			it.Reason = OmitLimit
			it.Bytes, it.Tokens, it.Truncated = 0, 0, false
			items = append(items, it)
			continue
		}
		used += tok
		it.Included, it.SHA256 = true, sha(text)
		keep = append(keep, kept{it: it, text: body})
	}
	// Texto em ordem estável de tipo e referência, independente da prioridade.
	sort.SliceStable(keep, func(i, j int) bool {
		a, b := keep[i].it, keep[j].it
		if kindOrder[a.Kind] != kindOrder[b.Kind] {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		return a.Ref < b.Ref
	})
	var sb strings.Builder
	for _, k := range keep {
		sb.WriteString(k.text)
		items = append(items, k.it)
	}
	content := sb.String()
	if red.Redact(content) != content {
		return Pack{}, ErrSecretLeft
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.Included != b.Included {
			return a.Included
		}
		if kindOrder[a.Kind] != kindOrder[b.Kind] {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		return a.Ref < b.Ref
	})
	omitted := 0
	for _, it := range items {
		if !it.Included {
			omitted++
		}
	}
	return Pack{Content: content, Manifest: Manifest{
		Version: 1, Method: "estimate:bytes/3", LimitTokens: lim.MaxTokens,
		EstimatedTokens: used, ContentHash: sha(content), Items: items, Omitted: omitted,
	}}, nil
}

func cutUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && s[n]&0xC0 == 0x80 { // não corta no meio de um caractere
		n--
	}
	return s[:n]
}

// Verify confere o pacote: hash do texto, total de tokens e limite.
func (p Pack) Verify(red *security.Redactor) error {
	m := p.Manifest
	if m.Version != 1 || sha(p.Content) != m.ContentHash {
		return fmt.Errorf("%w: conteúdo", ErrCorrupt)
	}
	var tok int64
	for _, it := range m.Items {
		if it.Included {
			tok += it.Tokens
		}
	}
	if tok != m.EstimatedTokens || m.EstimatedTokens > m.LimitTokens || budget.EstimateTokens(p.Content) > m.LimitTokens {
		return fmt.Errorf("%w: total de tokens", ErrCorrupt)
	}
	if red != nil && red.Redact(p.Content) != p.Content {
		return ErrSecretLeft
	}
	return nil
}
