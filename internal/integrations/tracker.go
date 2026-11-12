package integrations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Categorias normalizadas de estado, iguais para Jira e ClickUp.
const (
	CategoryTodo       = "todo"
	CategoryInProgress = "in_progress"
	CategoryDone       = "done"
)

// ErrKind classifica falhas das plataformas de forma estável para o chamador
// decidir (repetir, pedir permissão, abrir decisão).
type ErrKind string

const (
	ErrAuth        ErrKind = "auth"         // credencial recusada (401)
	ErrPermission  ErrKind = "permission"   // sem permissão para o recurso (403)
	ErrNotFound    ErrKind = "not_found"    // recurso inexistente (ou invisível à credencial)
	ErrRateLimited ErrKind = "rate_limited" // 429; veja RetryAfter
	ErrUnavailable ErrKind = "unavailable"  // rede, timeout ou 5xx: repetir exige reconciliar antes de escritas
	ErrInvalid     ErrKind = "invalid"      // requisição recusada (400/422) ou entrada inválida
)

// Error é a falha de uma operação. A mensagem é fixa: nunca carrega corpo
// remoto, URL, cabeçalhos nem o segredo.
type Error struct {
	Kind       ErrKind
	Op         string
	Status     int
	RetryAfter time.Duration // 0 = desconhecido
}

func (e *Error) Error() string {
	switch e.Kind {
	case ErrAuth:
		return e.Op + ": credencial recusada pela plataforma"
	case ErrPermission:
		return e.Op + ": sem permissão para este recurso"
	case ErrNotFound:
		return e.Op + ": recurso não encontrado"
	case ErrRateLimited:
		return e.Op + ": limite de requisições da plataforma atingido"
	case ErrUnavailable:
		return e.Op + ": plataforma indisponível ou sem resposta"
	default:
		return e.Op + ": requisição inválida para a plataforma"
	}
}

// Retryable indica falhas transitórias. Para escritas, confirme o estado no
// destino antes de repetir (ledger de efeitos, T13-04).
func (e *Error) Retryable() bool { return e.Kind == ErrRateLimited || e.Kind == ErrUnavailable }

// Identity é quem a credencial representa (sem e-mail nem token).
type Identity struct {
	Account string `json:"account"`
	Name    string `json:"name"`
}

// Item é um item de trabalho normalizado.
type Item struct {
	Provider     string   `json:"provider"`
	ID           string   `json:"id"`  // identidade remota estável
	Key          string   `json:"key"` // chave legível (Jira: ABC-1; ClickUp: custom_id ou id)
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	Subtask      bool     `json:"subtask"`
	ParentID     string   `json:"parent_id,omitempty"`
	Status       string   `json:"status"`
	Category     string   `json:"category"`
	Priority     string   `json:"priority"`
	Labels       []string `json:"labels"`
	Dependencies []string `json:"dependencies"` // IDs dos itens dos quais este depende
	Version      string   `json:"version"`      // marca de atualização remota (revisão observada)
	URL          string   `json:"url"`
}

// ItemPage é uma página de itens; NextCursor vazio indica a última.
type ItemPage struct {
	Items      []Item `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// ListOptions filtra a listagem. Label é o marcador de elegibilidade.
type ListOptions struct {
	Label       string
	IncludeDone bool
	Cursor      string
	Limit       int // Jira: 1..100 (padrão 50); ClickUp tem página fixa
}

// ItemType é um tipo de item (Jira). ClickUp não tem tipos e devolve lista vazia.
type ItemType struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
}

// Status é um estado possível na origem.
type Status struct {
	Name     string `json:"name"`
	Category string `json:"category"`
}

// Field é um campo da origem (inclusive personalizados).
type Field struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Custom bool   `json:"custom"`
}

// Transition é uma mudança de estado válida para um item agora.
type Transition struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ToStatus string `json:"to_status"`
	Category string `json:"category"`
}

// SubtaskInput descreve a subtarefa a criar sob um item.
type SubtaskInput struct {
	Title       string
	Description string
}

// Tracker é a interface simétrica dos adaptadores Jira e ClickUp. A conexão é
// de escopo reduzido e o segredo nunca sai do adaptador.
type Tracker interface {
	Provider() string
	// Identity valida a credencial.
	Identity(ctx context.Context) (Identity, error)
	// Container é a chave do projeto Jira ou o ID da lista ClickUp.
	ItemTypes(ctx context.Context, container string) ([]ItemType, error)
	Statuses(ctx context.Context, container string) ([]Status, error)
	Fields(ctx context.Context, container string) ([]Field, error)
	Priorities(ctx context.Context) ([]string, error)
	ListItems(ctx context.Context, container string, opts ListOptions) (ItemPage, error)
	GetItem(ctx context.Context, id string) (Item, error)
	// Transitions lista só mudanças válidas agora; Transition recusa as demais
	// sem chamar a escrita remota.
	Transitions(ctx context.Context, itemID string) ([]Transition, error)
	Transition(ctx context.Context, itemID, transitionID string) error
	CreateSubtask(ctx context.Context, parentID string, in SubtaskInput) (Item, error)
	// AddDependency registra que itemID depende de dependsOnID.
	AddDependency(ctx context.Context, itemID, dependsOnID string) error
	// AddLink anexa um link externo (ex.: Draft PR) ao item.
	AddLink(ctx context.Context, itemID, linkURL, title string) error
	// Children lista as subtarefas de um item (inclui a descrição, que pode carregar
	// o marcador da chave de negócio). Base da prova de existência (T13-04).
	Children(ctx context.Context, parentID string) ([]Item, error)
	// HasLink informa se o link já está anexado ao item.
	HasLink(ctx context.Context, itemID, linkURL string) (bool, error)
}

// Config cria um Tracker. Bases só são sobrescritas em testes.
type Config struct {
	Kind   string // "jira" | "clickup"
	Target string // Jira: https://<site>.atlassian.net
	Secret security.Secret
	Client *http.Client
	// Testes.
	BaseURL        string
	AllowAnyTarget bool
}

// NewHTTPClient usa timeout curto e NÃO segue redirecionamentos: o cabeçalho
// de autorização nunca é reenviado a outro host.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

// NewTracker devolve o adaptador do tipo pedido.
func NewTracker(cfg Config) (Tracker, error) {
	if cfg.Client == nil {
		cfg.Client = NewHTTPClient()
	}
	switch cfg.Kind {
	case "jira":
		return newJira(cfg)
	case "clickup":
		return newClickUp(cfg)
	default:
		return nil, &Error{Kind: ErrInvalid, Op: "criar adaptador"}
	}
}

// apiClient concentra HTTP, limites e tradução de erros dos dois adaptadores.
type apiClient struct {
	hc    *http.Client
	base  string
	auth  func(*http.Request)
	reset string // cabeçalho extra com reset do limite (ClickUp)
	now   func() time.Time
}

const maxBody = 4 << 20

func (c *apiClient) do(ctx context.Context, op, method, path string, query url.Values, body, out any) *Error {
	u := c.base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return &Error{Kind: ErrInvalid, Op: op}
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, u, rdr)
	if err != nil {
		return &Error{Kind: ErrInvalid, Op: op}
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.auth(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		// Não repassa err: o erro do cliente HTTP contém a URL.
		return &Error{Kind: ErrUnavailable, Op: op}
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if out == nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBody))
			return nil
		}
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(out); err != nil {
			return &Error{Kind: ErrUnavailable, Op: op, Status: resp.StatusCode}
		}
		return nil
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBody))
	e := &Error{Op: op, Status: resp.StatusCode}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		e.Kind = ErrAuth
	case resp.StatusCode == http.StatusForbidden:
		e.Kind = ErrPermission
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		e.Kind = ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		e.Kind = ErrRateLimited
		e.RetryAfter = c.retryAfter(resp.Header)
	case resp.StatusCode >= 500:
		e.Kind = ErrUnavailable
	default:
		e.Kind = ErrInvalid
	}
	return e
}

// retryAfter lê Retry-After (segundos ou data) ou o cabeçalho de reset (epoch).
func (c *apiClient) retryAfter(h http.Header) time.Duration {
	now := c.now()
	clamp := func(d time.Duration) time.Duration {
		if d < 0 {
			return 0
		}
		if d > time.Hour {
			return time.Hour
		}
		return d
	}
	if v := strings.TrimSpace(h.Get("Retry-After")); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return clamp(time.Duration(n) * time.Second)
		}
		if t, err := http.ParseTime(v); err == nil {
			return clamp(t.Sub(now))
		}
	}
	if c.reset != "" {
		if n, err := strconv.ParseInt(strings.TrimSpace(h.Get(c.reset)), 10, 64); err == nil {
			return clamp(time.Unix(n, 0).Sub(now))
		}
	}
	return 0
}

func invalid(op string) *Error { return &Error{Kind: ErrInvalid, Op: op} }

// asError evita o nil tipado: converte *Error em error preservando nil.
func asError(e *Error) error {
	if e == nil {
		return nil
	}
	return e
}

func pageLimit(n int) int {
	if n <= 0 {
		return 50
	}
	if n > 100 {
		return 100
	}
	return n
}

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return fmt.Sprintf("%s…", s[:n])
}
