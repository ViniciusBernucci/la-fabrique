package gateways

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Endereços oficiais (compatíveis com a API de chat completions).
var defaultBase = map[string]string{
	models.GatewayOpenRouter: "https://openrouter.ai/api/v1",
	models.GatewayVercel:     "https://ai-gateway.vercel.sh/v1",
}

const maxBody = 4 << 20

// Error é uma falha do gateway; a mensagem nunca carrega corpo remoto, URL ou token.
type Error struct {
	Kind   string // auth | rate_limit | unavailable | protocol | invalid | limit
	Status int
}

func (e *Error) Error() string {
	switch e.Kind {
	case "auth":
		return "gateway: credencial recusada"
	case "rate_limit":
		return "gateway: limite de uso atingido"
	case "unavailable":
		return "gateway: indisponível ou sem resposta"
	case "protocol":
		return "gateway: resposta fora do formato esperado"
	case "limit":
		return "gateway: resposta acima dos limites"
	}
	return "gateway: requisição inválida"
}

func fail(kind string, status int) *Error { return &Error{Kind: kind, Status: status} }

// ToolCall é um pedido de ferramenta feito pelo modelo (dado não confiável).
type ToolCall struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
	// BadArgs marca argumentos que não são um objeto JSON.
	BadArgs bool `json:"bad_args,omitempty"`
}

// Message é uma mensagem da conversa.
type Message struct {
	Role       string // system | user | assistant | tool
	Content    string
	ToolCallID string
	ToolCalls  []ToolCall
}

// ToolSpec anuncia uma ferramenta ao modelo.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]any // JSON Schema
}

// ChatRequest é um pedido de conclusão.
type ChatRequest struct {
	Model    string
	Messages []Message
	Tools    []ToolSpec
	Params   models.Params
}

// Usage é o consumo informado pelo gateway.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

// ChatResponse é a resposta normalizada.
type ChatResponse struct {
	Text         string
	ToolCalls    []ToolCall
	FinishReason string
	Usage        Usage
}

// Chatter é o que o laço de agente precisa de um gateway.
type Chatter interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
}

// Client é um gateway OpenAI-compatível. Não segue redirecionamentos.
type Client struct {
	Kind    string
	BaseURL string
	Token   security.Secret
	HTTP    *http.Client
}

// New valida o destino. baseURL vazio usa o endereço oficial do gateway;
// allowPrivate (só testes) permite http e destinos internos.
func New(kind string, token security.Secret, baseURL string, allowPrivate bool) (*Client, error) {
	def, ok := defaultBase[kind]
	if !ok {
		return nil, fail("invalid", 0)
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = def
	}
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return nil, fail("invalid", 0)
	}
	if u.Scheme != "https" && !(allowPrivate && u.Scheme == "http") {
		return nil, fail("invalid", 0)
	}
	if !allowPrivate && policy.IsInternalHost(u.Hostname()) {
		return nil, fail("invalid", 0)
	}
	if token.IsZero() {
		return nil, fail("auth", 0)
	}
	return &Client{
		Kind: kind, BaseURL: strings.TrimRight(u.String(), "/"), Token: token,
		HTTP: &http.Client{Timeout: 120 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

type wireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type wireMessage struct {
	Role       string         `json:"role"`
	Content    *string        `json:"content"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolCalls  []wireToolCall `json:"tool_calls,omitempty"`
}

func toWire(ms []Message) []wireMessage {
	out := make([]wireMessage, 0, len(ms))
	for _, m := range ms {
		c := m.Content
		w := wireMessage{Role: m.Role, Content: &c, ToolCallID: m.ToolCallID}
		for _, tc := range m.ToolCalls {
			var wt wireToolCall
			wt.ID, wt.Type, wt.Function.Name = tc.ID, "function", tc.Name
			args, _ := json.Marshal(tc.Args)
			if tc.Args == nil {
				args = []byte("{}")
			}
			wt.Function.Arguments = string(args)
			w.ToolCalls = append(w.ToolCalls, wt)
		}
		out = append(out, w)
	}
	return out
}

// Chat envia a conversa e normaliza a resposta.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	body := map[string]any{"model": req.Model, "messages": toWire(req.Messages)}
	if len(req.Tools) > 0 {
		var ts []map[string]any
		for _, t := range req.Tools {
			ts = append(ts, map[string]any{"type": "function", "function": map[string]any{"name": t.Name, "description": t.Description, "parameters": t.Parameters}})
		}
		body["tools"] = ts
	}
	if p := req.Params; p.Temperature != nil {
		body["temperature"] = *p.Temperature
	}
	if p := req.Params; p.TopP != nil {
		body["top_p"] = *p.TopP
	}
	if p := req.Params; p.MaxTokens != nil {
		body["max_tokens"] = *p.MaxTokens
	}
	raw, _ := json.Marshal(body)
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return ChatResponse{}, fail("invalid", 0)
	}
	hr.Header.Set("Content-Type", "application/json")
	hr.Header.Set("Authorization", "Bearer "+c.Token.Reveal())
	resp, err := c.HTTP.Do(hr)
	if err != nil {
		return ChatResponse{}, fail("unavailable", 0) // o erro do cliente HTTP contém a URL
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return ChatResponse{}, fail("auth", resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests:
		return ChatResponse{}, fail("rate_limit", resp.StatusCode)
	case resp.StatusCode >= 500:
		return ChatResponse{}, fail("unavailable", resp.StatusCode)
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return ChatResponse{}, fail("invalid", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if err != nil || len(data) > maxBody {
		return ChatResponse{}, fail("limit", 0)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content   *string        `json:"content"`
				ToolCalls []wireToolCall `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage Usage `json:"usage"`
	}
	if json.Unmarshal(data, &out) != nil || len(out.Choices) == 0 {
		return ChatResponse{}, fail("protocol", 0)
	}
	ch := out.Choices[0]
	res := ChatResponse{FinishReason: ch.FinishReason, Usage: out.Usage}
	if ch.Message.Content != nil {
		res.Text = *ch.Message.Content
	}
	for i, tc := range ch.Message.ToolCalls {
		call := ToolCall{ID: tc.ID, Name: tc.Function.Name}
		if call.ID == "" {
			call.ID = fmt.Sprintf("call_%d", i)
		}
		var args map[string]any
		if s := strings.TrimSpace(tc.Function.Arguments); s == "" {
			args = map[string]any{}
		} else if json.Unmarshal([]byte(s), &args) != nil || args == nil {
			call.BadArgs = true
			args = map[string]any{}
		}
		call.Args = args
		res.ToolCalls = append(res.ToolCalls, call)
	}
	return res, nil
}

// Open resolve a conta no cofre e cria o cliente do gateway. A conta precisa ser
// do tipo api, ativa, e seu destino define o gateway (openrouter ou vercel).
func Open(ctx context.Context, vault *security.Vault, connectionID, baseURL string, allowPrivate bool) (*Client, error) {
	secret, conn, err := vault.Resolve(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	if conn.Kind != "api" {
		return nil, fail("invalid", 0)
	}
	return New(conn.Target, secret, baseURL, allowPrivate)
}
