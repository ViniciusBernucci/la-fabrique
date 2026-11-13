package contextpack

import (
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func srcs() []Source {
	return []Source{
		{Kind: KindOutput, Ref: "build", Content: "saída do build", Priority: 1},
		{Kind: KindRequirement, Ref: "R1", Content: "O sistema deve exportar CSV.", Required: true},
		{Kind: KindCriterion, Ref: "C1", Content: "Exporta 1000 linhas em menos de 2s.", Required: true},
		{Kind: KindMemory, Ref: "mem-1", Content: "Usamos pgx, não database/sql.", Priority: 5},
	}
}

func TestBuildIsDeterministicAndRecordsSources(t *testing.T) {
	a, err := Build(srcs(), Limits{MaxTokens: 1000}, nil)
	if err != nil {
		t.Fatal(err)
	}
	rev := srcs()
	rev[0], rev[3] = rev[3], rev[0]
	b, err := Build(rev, Limits{MaxTokens: 1000}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Content != b.Content || a.Manifest.ContentHash != b.Manifest.ContentHash {
		t.Fatal("a ordem de entrada não pode mudar o pacote")
	}
	if !strings.HasPrefix(a.Content, "### requirement R1") || a.Manifest.Method == "" || len(a.Manifest.Items) != 4 || a.Manifest.Omitted != 0 {
		t.Fatalf("%+v\n%s", a.Manifest, a.Content)
	}
	for _, it := range a.Manifest.Items {
		if !it.Included || len(it.SHA256) != 64 || it.Tokens <= 0 {
			t.Fatalf("%+v", it)
		}
	}
	if err := a.Verify(nil); err != nil {
		t.Fatal(err)
	}
}

func TestBuildRemovesSecretsBeforeHashing(t *testing.T) {
	red := security.NewRedactor()
	red.Register("s3gredo-interno-xyz")
	pat := "ghp_" + strings.Repeat("a", 30)
	p, err := Build([]Source{
		{Kind: KindFile, Ref: "a.env", Content: "TOKEN=s3gredo-interno-xyz e " + pat},
		{Kind: KindEvidence, Ref: "e1", Content: "limpo"},
	}, Limits{MaxTokens: 500}, red)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.Content, "s3gredo-interno-xyz") || strings.Contains(p.Content, pat) || !strings.Contains(p.Content, "[REDACTED]") {
		t.Fatal(p.Content)
	}
	var red1, red2 bool
	for _, it := range p.Manifest.Items {
		if it.Ref == "a.env" {
			red1 = it.Redacted
		}
		if it.Ref == "e1" {
			red2 = it.Redacted
		}
	}
	if !red1 || red2 {
		t.Fatal("o manifesto deve indicar só a fonte redigida")
	}
	// Um pacote adulterado com segredo é recusado.
	p.Content += "\nTOKEN=s3gredo-interno-xyz"
	if err := p.Verify(red); err == nil {
		t.Fatal("deveria recusar")
	}
}

func TestBuildOmitsByLimitAndRecordsWhy(t *testing.T) {
	big := strings.Repeat("x", 3000) // ~1000 tokens
	p, err := Build([]Source{
		{Kind: KindRequirement, Ref: "R1", Content: "curto", Required: true},
		{Kind: KindFile, Ref: "grande.txt", Content: big, Priority: 1},
		{Kind: KindMemory, Ref: "m", Content: "cabe", Priority: 9},
		{Kind: KindMemory, Ref: "m", Content: "duplicada", Priority: 9},
		{Kind: KindMemory, Ref: "vazia", Content: "   "},
	}, Limits{MaxTokens: 100}, nil)
	if err != nil {
		t.Fatal(err)
	}
	why := map[string]string{}
	for _, it := range p.Manifest.Items {
		if !it.Included {
			why[it.Ref] = it.Reason
		}
	}
	if why["grande.txt"] != OmitLimit || why["vazia"] != OmitEmpty || p.Manifest.Omitted != 3 {
		t.Fatalf("%v %+v", why, p.Manifest)
	}
	if !strings.Contains(p.Content, "cabe") || strings.Contains(p.Content, "duplicada") || p.Manifest.EstimatedTokens > 100 {
		t.Fatal(p.Content)
	}
}

func TestBuildTruncatesOptionalButNeverRequired(t *testing.T) {
	long := strings.Repeat("é", 400)
	p, err := Build([]Source{{Kind: KindFile, Ref: "f", Content: long}}, Limits{MaxTokens: 1000, MaxItemBytes: 101}, nil)
	if err != nil {
		t.Fatal(err)
	}
	it := p.Manifest.Items[0]
	if !it.Truncated || it.Bytes > 101 || it.Bytes%2 != 0 {
		t.Fatalf("truncou no meio de um caractere: %+v", it)
	}
	_, err = Build([]Source{{Kind: KindCriterion, Ref: "C", Content: strings.Repeat("a", 900), Required: true}}, Limits{MaxTokens: 100}, nil)
	if !errors.Is(err, ErrRequiredOmitted) {
		t.Fatalf("obrigatória nunca é descartada em silêncio: %v", err)
	}
}

func TestBuildValidatesInput(t *testing.T) {
	for _, s := range []Source{{Kind: "x", Ref: "r", Content: "c"}, {Kind: KindFile, Ref: "", Content: "c"}, {Kind: KindFile, Ref: "a\nb", Content: "c"}} {
		if _, err := Build([]Source{s}, Limits{MaxTokens: 10}, nil); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Fatalf("%+v: %v", s, err)
		}
	}
	if _, err := Build(nil, Limits{}, nil); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatal(err)
	}
}

func TestVerifyDetectsTamper(t *testing.T) {
	p, _ := Build(srcs(), Limits{MaxTokens: 1000}, nil)
	q := p
	q.Content = strings.Replace(q.Content, "CSV", "XML", 1)
	if !errors.Is(q.Verify(nil), ErrCorrupt) {
		t.Fatal("conteúdo adulterado")
	}
	m := p
	m.Manifest.EstimatedTokens++
	if !errors.Is(m.Verify(nil), ErrCorrupt) {
		t.Fatal("total adulterado")
	}
}

func ptr(n int64) *int64 { return &n }

func TestDecideOfficialAndEstimate(t *testing.T) {
	d, err := Decide(Usage{Official: ptr(70_000), Window: 100_000}, Policy{})
	if err != nil || d.Renew || d.Method != UsageOfficial || d.Used != 70_000 {
		t.Fatalf("%+v %v", d, err)
	}
	d, _ = Decide(Usage{Official: ptr(80_000), Window: 100_000}, Policy{})
	if !d.Renew {
		t.Fatalf("80%% da janela deve renovar: %+v", d)
	}
	// Sem métrica oficial: estimativa com margem renova ANTES do que o número cru.
	d, _ = Decide(Usage{Estimated: 70_000, Window: 100_000}, Policy{})
	if !d.Renew || d.Method != UsageEstimate || d.Used != 87_500 {
		t.Fatalf("%+v", d)
	}
	d, _ = Decide(Usage{Estimated: 10_000, Window: 100_000}, Policy{})
	if d.Renew {
		t.Fatalf("%+v", d)
	}
	// O oficial, quando existe, vence a estimativa.
	d, _ = Decide(Usage{Official: ptr(1000), Estimated: 99_000, Window: 100_000}, Policy{})
	if d.Renew || d.Method != UsageOfficial {
		t.Fatalf("%+v", d)
	}
	if _, err := Decide(Usage{Window: 0}, Policy{}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatal("sem janela deve falhar")
	}
	if _, err := Decide(Usage{Official: ptr(-1), Window: 10}, Policy{}); err == nil {
		t.Fatal("uso negativo")
	}
}
