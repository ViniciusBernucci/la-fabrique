package mcp

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Versões do protocolo aceitas, da mais nova para a mais antiga.
var SupportedVersions = []string{"2025-11-25", "2025-06-18", "2025-03-26"}

// Limites do cliente.
const (
	maxBody      = 4 << 20
	maxPages     = 10
	maxTools     = 500
	maxResultTxt = 256 << 10
)

// Error é uma falha do cliente; a mensagem nunca carrega corpo remoto, URL ou token.
type Error struct {
	Op     string
	Kind   string // auth | permission | not_found | unavailable | protocol | rpc | invalid | limit
	Status int
	Code   int // código JSON-RPC
}

func (e *Error) Error() string {
	switch e.Kind {
	case "auth":
		return e.Op + ": credencial recusada pelo servidor MCP"
	case "permission":
		return e.Op + ": sem permissão no servidor MCP"
	case "not_found":
		return e.Op + ": recurso não encontrado no servidor MCP"
	case "unavailable":
		return e.Op + ": servidor MCP indisponível ou sem resposta"
	case "protocol":
		return e.Op + ": resposta fora do protocolo MCP"
	case "rpc":
		return fmt.Sprintf("%s: o servidor MCP recusou a chamada (código %d)", e.Op, e.Code)
	case "limit":
		return e.Op + ": resposta acima dos limites"
	}
	return e.Op + ": requisição inválida para o servidor MCP"
}

func fail(op, kind string) *Error { return &Error{Op: op, Kind: kind} }

// ToolDef é uma ferramenta anunciada pelo servidor. Descrição e anotações são DADOS.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// SchemaHash identifica a revisão do contrato (nome + esquema de entrada canônico).
// Mudar descrição não muda o hash; mudar o contrato muda.
func (t ToolDef) SchemaHash() string {
	h := sha256.Sum256(append([]byte(t.Name+"\x00"), canonical(t.InputSchema)...))
	return hex.EncodeToString(h[:])
}

func canonical(v any) []byte {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b bytes.Buffer
		b.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				b.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			b.Write(kb)
			b.WriteByte(':')
			b.Write(canonical(x[k]))
		}
		b.WriteByte('}')
		return b.Bytes()
	case []any:
		var b bytes.Buffer
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			b.Write(canonical(e))
		}
		b.WriteByte(']')
		return b.Bytes()
	}
	b, _ := json.Marshal(v)
	return b
}

// CleanText remove caracteres de controle e limita o tamanho: descrições e
// resultados de ferramentas são dados não confiáveis.
func CleanText(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, s)
	if len(s) > max {
		s = s[:max]
	}
	return s
}

// ServerInfo é o resultado da negociação.
type ServerInfo struct {
	Name     string
	Version  string
	Protocol string
}

// CallResult é o resultado de uma chamada: texto limitado e sinal de erro da ferramenta.
type CallResult struct {
	Text      string `json:"text"`
	IsError   bool   `json:"is_error"`
	Blocks    int    `json:"blocks"`
	Truncated bool   `json:"truncated"`
}

// Client fala Streamable HTTP com um servidor MCP. Não segue redirecionamentos e
// recusa destinos internos (a menos que AllowPrivate, só em testes).
type Client struct {
	URL          string
	Token        security.Secret
	HTTP         *http.Client
	AllowPrivate bool

	session  string
	protocol string
	nextID   atomic.Int64
}

// NewClient valida o destino e prepara o cliente.
func NewClient(rawURL string, token security.Secret, allowPrivate bool) (*Client, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" {
		return nil, fail("conectar", "invalid")
	}
	if u.Scheme != "https" && !(allowPrivate && u.Scheme == "http") {
		return nil, fail("conectar", "invalid")
	}
	if !allowPrivate && policy.IsInternalHost(u.Hostname()) {
		return nil, fail("conectar", "invalid")
	}
	return &Client{
		URL: u.String(), Token: token, AllowPrivate: allowPrivate,
		HTTP: &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}, nil
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type rpcResponse struct {
	ID     *json.RawMessage `json:"id"`
	Result json.RawMessage  `json:"result"`
	Error  *struct {
		Code int `json:"code"`
	} `json:"error"`
}

func (c *Client) post(ctx context.Context, op string, body rpcRequest, wantID int64) (json.RawMessage, error) {
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return nil, fail(op, "invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if c.protocol != "" {
		req.Header.Set("MCP-Protocol-Version", c.protocol)
	}
	if c.session != "" {
		req.Header.Set("Mcp-Session-Id", c.session)
	}
	if tok := c.Token.Reveal(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fail(op, "unavailable") // o erro do cliente HTTP contém a URL
	}
	defer resp.Body.Close()
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" && c.session == "" {
		c.session = sid
	}
	switch {
	case resp.StatusCode == http.StatusAccepted && body.ID == nil:
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBody))
		return nil, nil
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, &Error{Op: op, Kind: "auth", Status: 401}
	case resp.StatusCode == http.StatusForbidden:
		return nil, &Error{Op: op, Kind: "permission", Status: 403}
	case resp.StatusCode == http.StatusNotFound:
		return nil, &Error{Op: op, Kind: "not_found", Status: 404}
	case resp.StatusCode >= 500 || resp.StatusCode == http.StatusTooManyRequests:
		return nil, &Error{Op: op, Kind: "unavailable", Status: resp.StatusCode}
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, &Error{Op: op, Kind: "invalid", Status: resp.StatusCode}
	}
	ct := resp.Header.Get("Content-Type")
	var msg rpcResponse
	switch {
	case strings.HasPrefix(ct, "text/event-stream"):
		found, err := readSSE(io.LimitReader(resp.Body, maxBody+1), wantID, &msg)
		if err != nil {
			return nil, fail(op, "protocol")
		}
		if !found {
			return nil, fail(op, "protocol")
		}
	case strings.HasPrefix(ct, "application/json"):
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
		if err != nil || len(data) > maxBody {
			return nil, fail(op, "limit")
		}
		if json.Unmarshal(data, &msg) != nil {
			return nil, fail(op, "protocol")
		}
	default:
		return nil, fail(op, "protocol")
	}
	if msg.Error != nil {
		return nil, &Error{Op: op, Kind: "rpc", Code: msg.Error.Code}
	}
	return msg.Result, nil
}

// readSSE lê eventos até achar a resposta do pedido wantID.
func readSSE(r io.Reader, wantID int64, out *rpcResponse) (bool, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), maxBody)
	var data bytes.Buffer
	flush := func() bool {
		if data.Len() == 0 {
			return false
		}
		var m rpcResponse
		err := json.Unmarshal(data.Bytes(), &m)
		data.Reset()
		if err != nil || m.ID == nil {
			return false
		}
		var id int64
		if json.Unmarshal(*m.ID, &id) == nil && id == wantID {
			*out = m
			return true
		}
		return false
	}
	for sc.Scan() {
		line := sc.Text()
		switch {
		case line == "":
			if flush() {
				return true, nil
			}
		case strings.HasPrefix(line, "data:"):
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := sc.Err(); err != nil {
		return false, err
	}
	return flush(), nil
}

func (c *Client) call(ctx context.Context, op, method string, params any) (json.RawMessage, error) {
	id := c.nextID.Add(1)
	return c.post(ctx, op, rpcRequest{JSONRPC: "2.0", ID: &id, Method: method, Params: params}, id)
}

// Initialize negocia a versão do protocolo e abre a sessão.
func (c *Client) Initialize(ctx context.Context) (ServerInfo, error) {
	const op = "mcp: inicializar"
	raw, err := c.call(ctx, op, "initialize", map[string]any{
		"protocolVersion": SupportedVersions[0],
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "la-fabrique", "version": "1"},
	})
	if err != nil {
		return ServerInfo{}, err
	}
	var res struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	if json.Unmarshal(raw, &res) != nil {
		return ServerInfo{}, fail(op, "protocol")
	}
	ok := false
	for _, v := range SupportedVersions {
		ok = ok || v == res.ProtocolVersion
	}
	if !ok {
		return ServerInfo{}, fail(op, "protocol") // versão que não sabemos falar
	}
	c.protocol = res.ProtocolVersion
	if _, err := c.post(ctx, op, rpcRequest{JSONRPC: "2.0", Method: "notifications/initialized"}, 0); err != nil {
		return ServerInfo{}, err
	}
	return ServerInfo{Name: CleanText(res.ServerInfo.Name, 100), Version: CleanText(res.ServerInfo.Version, 50), Protocol: res.ProtocolVersion}, nil
}

// ListTools pagina tools/list (até 10 páginas e 500 ferramentas).
func (c *Client) ListTools(ctx context.Context) ([]ToolDef, error) {
	const op = "mcp: listar ferramentas"
	var all []ToolDef
	cursor := ""
	for page := 0; page < maxPages; page++ {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := c.call(ctx, op, "tools/list", params)
		if err != nil {
			return nil, err
		}
		var res struct {
			Tools []struct {
				Name        string         `json:"name"`
				Description string         `json:"description"`
				InputSchema map[string]any `json:"inputSchema"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor"`
		}
		if json.Unmarshal(raw, &res) != nil {
			return nil, fail(op, "protocol")
		}
		for _, t := range res.Tools {
			if !validToolName(t.Name) {
				return nil, fail(op, "protocol")
			}
			schema := t.InputSchema
			if schema == nil {
				schema = map[string]any{"type": "object"}
			}
			all = append(all, ToolDef{Name: t.Name, Description: CleanText(t.Description, 1000), InputSchema: schema})
			if len(all) > maxTools {
				return nil, fail(op, "limit")
			}
		}
		if res.NextCursor == "" {
			return all, nil
		}
		cursor = res.NextCursor
	}
	return nil, fail(op, "limit")
}

func validToolName(n string) bool {
	if n == "" || len(n) > 64 {
		return false
	}
	for _, r := range n {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

// CallTool chama uma ferramenta. O resultado é DADO não confiável: só texto,
// sem caracteres de controle e limitado.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (CallResult, error) {
	const op = "mcp: chamar ferramenta"
	if !validToolName(name) {
		return CallResult{}, fail(op, "invalid")
	}
	if args == nil {
		args = map[string]any{}
	}
	raw, err := c.call(ctx, op, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return CallResult{}, err
	}
	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if json.Unmarshal(raw, &res) != nil {
		return CallResult{}, fail(op, "protocol")
	}
	out := CallResult{IsError: res.IsError, Blocks: len(res.Content)}
	var b strings.Builder
	for _, blk := range res.Content {
		if blk.Type != "text" {
			continue // imagens, recursos e outros blocos não entram no contexto do agente
		}
		if b.Len()+len(blk.Text) > maxResultTxt {
			out.Truncated = true
			break
		}
		b.WriteString(blk.Text)
		b.WriteByte('\n')
	}
	out.Text = CleanText(strings.TrimRight(b.String(), "\n"), maxResultTxt)
	return out, nil
}
