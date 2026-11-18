package runner

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

func ev(seq int64, typ string, p any) persistence.Event {
	b, _ := json.Marshal(p)
	return persistence.Event{Seq: seq, Type: typ, Payload: b}
}

func TestHintsRecognizeTheUsualFailures(t *testing.T) {
	cases := map[string]string{
		"não há conta claude conectada: conecte em Conexões":        "Nenhuma conta de IA ativa",
		"falhou (código 127): make lint — sh: make: not found":      "FACTORY_TOOL_IMAGE",
		"Cannot connect to the Docker daemon at unix:///x":          "Docker",
		"Unable to find image 'le-fabrique/cli:dev' locally":        "make cli-image",
		"Invalid API key · Please run /login":                       "login",
		"o envio ao GitHub está desligado: ligue FACTORY_PUSH":      "FACTORY_PUSH",
		"falhou (código 1): test -f fabrica-teste/backend/saude.md": "arquivos da feature fictícia",
		"error: unknown option '--session-id'":                      "versão",
	}
	for text, want := range cases {
		hs := HintsFor(text)
		if len(hs) == 0 {
			t.Errorf("sem dica para %q", text)
			continue
		}
		all := ""
		for _, h := range hs {
			all += h.Cause + " " + h.Fix + " "
		}
		if !strings.Contains(strings.ToLower(all), strings.ToLower(want)) {
			t.Errorf("dica para %q deveria mencionar %q: %s", text, want, all)
		}
	}
	if len(HintsFor("tudo certo")) != 0 {
		t.Error("texto sem erro não deveria gerar dica")
	}
}

func TestReportShowsWhereItFailedWhyHintsAndPreflight(t *testing.T) {
	now := time.Now()
	run := Run{ExecutionID: "e1", ProjectID: "p1", Kind: "test", Title: "Teste", Status: "completed", Outcome: "failure", CreatedAt: now, UpdatedAt: now.Add(42 * time.Second)}
	events := []persistence.Event{
		ev(1, EventStarted, map[string]any{"title": "Teste"}),
		ev(2, EventDiag, map[string]any{"step": "preflight", "check": "docker", "ok": true, "detail": "o Docker responde"}),
		ev(3, EventDiag, map[string]any{"step": "preflight", "check": "conta claude", "ok": false, "detail": "0 contas ativas", "dica": "Conecte a conta"}),
		ev(4, EventDiag, map[string]any{"node": "dev", "step": "cli.run", "ms": 1200, "exit_code": 1, "stderr": "Invalid API key"}),
		ev(5, EventNode, map[string]any{"node": "dev", "kind": "error", "text": "o cliente terminou com código de saída diferente de zero: Invalid API key"}),
		ev(6, EventStep, map[string]any{"node": "dev", "type": "agent", "port": "error", "ms": 1500}),
		ev(7, EventFinished, map[string]any{"status": "completed", "outcome": "failure"}),
	}
	r := Report(run, events)
	for _, want := range []string{"## Onde falhou", "- Nó: `dev`", "Invalid API key", "## Causas prováveis e o que fazer", "login",
		"✖ conta claude", "✔ docker", "dica: Conecte a conta", "cli.run", "1500 ms", "duração: 42s"} {
		if !strings.Contains(r, want) {
			t.Errorf("o relatório deveria conter %q:\n%s", want, r)
		}
	}
	ok := Report(Run{ExecutionID: "e2", Status: "completed", Outcome: "success", CreatedAt: now, UpdatedAt: now}, nil)
	if !strings.Contains(ok, "Nenhuma falha registrada.") || strings.Contains(ok, "Causas prováveis") {
		t.Errorf("execução sem falha:\n%s", ok)
	}
}
