package browserenv

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/qualitypolicy"
)

// Driver é o cliente do servidor MCP do navegador dentro do container (navegação, interação, capturas).
type Driver interface {
	Navigate(ctx context.Context, url string) error
	Interact(ctx context.Context, action string, args map[string]string) (string, error)
	Screenshot(ctx context.Context, name string) ([]byte, error)
	ConsoleLogs(ctx context.Context) ([]string, error)
}

// Item é um registro de evidência (já sanitizado).
type Item struct {
	Kind string    `json:"kind"` // screenshot | log | error | step
	Name string    `json:"name"`
	Text string    `json:"text,omitempty"`
	Data []byte    `json:"-"`
	At   time.Time `json:"at"`
}

var secretPattern = regexp.MustCompile(`(?i)(bearer\s+[A-Za-z0-9._~+/=-]{8,}|(password|passwd|secret|token|api[_-]?key)\s*[=:]\s*\S+|sk-[A-Za-z0-9]{10,}|ghp_[A-Za-z0-9]{10,})`)

// Sanitize remove segredos de textos de evidência.
func Sanitize(s string) string { return secretPattern.ReplaceAllString(s, "[REDIGIDO]") }

// Tools é a superfície de ferramentas que o revisor recebe: só navega nas origens autorizadas e tudo o
// que registra passa pela sanitização.
type Tools struct {
	env   *Env
	drv   Driver
	mu    sync.Mutex
	items []Item
	Now   func() time.Time
}

func (t *Tools) now() time.Time {
	if t.Now != nil {
		return t.Now().UTC()
	}
	return time.Now().UTC()
}

func (t *Tools) add(it Item) {
	t.mu.Lock()
	defer t.mu.Unlock()
	it.At = t.now()
	it.Text = Sanitize(it.Text)
	t.items = append(t.items, it)
}

// Navigate abre uma URL da aplicação autorizada.
func (t *Tools) Navigate(ctx context.Context, url string) error {
	if err := t.env.Check(url); err != nil {
		t.add(Item{Kind: "error", Name: "navegação bloqueada", Text: err.Error()})
		return err
	}
	if err := t.drv.Navigate(ctx, url); err != nil {
		t.add(Item{Kind: "error", Name: "navegação", Text: err.Error()})
		return err
	}
	t.add(Item{Kind: "step", Name: "navegar", Text: url})
	return nil
}

// Interact executa uma ação (clicar, digitar...) no navegador. Entrada bruta de shell não existe aqui.
func (t *Tools) Interact(ctx context.Context, action string, args map[string]string) (string, error) {
	if !regexp.MustCompile(`^[a-z_]{2,32}$`).MatchString(action) {
		return "", fmt.Errorf("%w: ação %q", ErrInvalidSpec, action)
	}
	out, err := t.drv.Interact(ctx, action, args)
	if err != nil {
		t.add(Item{Kind: "error", Name: action, Text: err.Error()})
		return "", err
	}
	t.add(Item{Kind: "step", Name: action, Text: out})
	return out, nil
}

// Screenshot registra uma captura de tela.
func (t *Tools) Screenshot(ctx context.Context, name string) error {
	b, err := t.drv.Screenshot(ctx, name)
	if err != nil {
		t.add(Item{Kind: "error", Name: "screenshot " + name, Text: err.Error()})
		return err
	}
	t.add(Item{Kind: "screenshot", Name: name, Data: b})
	return nil
}

// Logs registra o console do navegador.
func (t *Tools) Logs(ctx context.Context) error {
	ls, err := t.drv.ConsoleLogs(ctx)
	if err != nil {
		t.add(Item{Kind: "error", Name: "console", Text: err.Error()})
		return err
	}
	t.add(Item{Kind: "log", Name: "console", Text: strings.Join(ls, "\n")})
	return nil
}

// Evidence devolve os registros.
func (t *Tools) Evidence() []Item {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]Item{}, t.items...)
}

// RunReview executa o review visual de uma revisão do código: provisiona, espera a prontidão, entrega as
// ferramentas ao corpo do review e SEMPRE recolhe os recursos (sucesso, falha ou cancelamento). Qualquer
// falha de provisionamento, prontidão, MCP ou do próprio review deixa o gate PENDENTE: nunca aprova.
func RunReview(ctx context.Context, rt Runtime, spec Spec, revision string, ready time.Duration, newDriver func(*Env) Driver, body func(*Tools) (bool, error)) (qualitypolicy.Review, []Item) {
	pending := qualitypolicy.Review{Revision: revision, Pending: true}
	env, err := Provision(ctx, rt, spec)
	if err != nil {
		return pending, []Item{{Kind: "error", Name: "provisionamento", Text: Sanitize(err.Error()), At: time.Now().UTC()}}
	}
	defer func() { _ = env.Cleanup(context.WithoutCancel(ctx)) }()
	if err := env.Ready(ctx, ready); err != nil {
		return pending, []Item{{Kind: "error", Name: "prontidão", Text: err.Error(), At: time.Now().UTC()}}
	}
	tools := &Tools{env: env, drv: newDriver(env)}
	approved, err := body(tools)
	if err != nil {
		tools.add(Item{Kind: "error", Name: "review", Text: err.Error()})
		return pending, tools.Evidence()
	}
	return qualitypolicy.Review{Revision: revision, Approved: approved, Independent: true}, tools.Evidence()
}
