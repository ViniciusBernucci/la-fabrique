package browserenv

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

// MCPDriver fala com o servidor MCP do Playwright (Chromium) que roda DENTRO do container isolado da
// execução, por stdio via `docker exec -i`: nada é publicado no host e a rede da execução continua interna.
type MCPDriver struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
	mu  sync.Mutex
	id  int
}

// MCPArgs são os argumentos do servidor MCP do navegador: headless, perfil em memória e só as origens
// autorizadas (defesa adicional: a barreira real é a rede interna e a lista de origens das ferramentas).
func MCPArgs(allowed []Origin) []string {
	var hosts []string
	for _, o := range allowed {
		hosts = append(hosts, o.Scheme+"://"+o.Host)
	}
	args := []string{"/app/cli.js", "--headless", "--browser", "chromium", "--no-sandbox", "--isolated", "--output-dir", "/tmp/out", "--block-service-workers"}
	if len(hosts) > 0 {
		args = append(args, "--allowed-origins", strings.Join(hosts, ";"))
	}
	return args
}

// StartMCP inicia o servidor MCP no container do navegador e faz o handshake.
func StartMCP(ctx context.Context, bin string, e *Env) (*MCPDriver, error) {
	if bin == "" {
		bin = "docker"
	}
	args := append([]string{"exec", "-i", e.Container, "node"}, MCPArgs(e.Origins)...)
	cmd := exec.Command(bin, args...)
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("iniciar o servidor MCP: %w", err)
	}
	d := &MCPDriver{cmd: cmd, in: in, out: bufio.NewReaderSize(out, 1<<20)}
	if _, err := d.call(ctx, "initialize", map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "la-fabrique", "version": "1"}}); err != nil {
		d.Close()
		return nil, fmt.Errorf("handshake MCP: %w", err)
	}
	if err := d.send(map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"}); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func (d *MCPDriver) send(msg map[string]any) error {
	b, _ := json.Marshal(msg)
	_, err := d.in.Write(append(b, '\n'))
	return err
}

// call envia uma requisição JSON-RPC e espera a resposta de mesmo id (ignora notificações do servidor).
func (d *MCPDriver) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.id++
	id := d.id
	if err := d.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		return nil, err
	}
	type resp struct {
		ID     *int            `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	done := make(chan error, 1)
	var result json.RawMessage
	go func() {
		for {
			line, err := d.out.ReadBytes('\n')
			if err != nil {
				done <- err
				return
			}
			var r resp
			if json.Unmarshal(bytes.TrimSpace(line), &r) != nil || r.ID == nil || *r.ID != id {
				continue
			}
			if r.Error != nil {
				done <- errors.New(r.Error.Message)
				return
			}
			result = r.Result
			done <- nil
			return
		}
	}()
	select {
	case err := <-done:
		return result, err
	case <-ctx.Done():
		d.Close()
		return nil, ctx.Err()
	case <-time.After(90 * time.Second):
		d.Close()
		return nil, errors.New("o servidor MCP não respondeu a tempo")
	}
}

type toolResult struct {
	IsError bool `json:"isError"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Data string `json:"data"`
	} `json:"content"`
}

func (d *MCPDriver) tool(ctx context.Context, name string, args map[string]any) (toolResult, error) {
	raw, err := d.call(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return toolResult{}, err
	}
	var tr toolResult
	if err := json.Unmarshal(raw, &tr); err != nil {
		return tr, err
	}
	if tr.IsError {
		return tr, errors.New(strings.TrimSpace(tr.text()))
	}
	return tr, nil
}

func (t toolResult) text() string {
	var parts []string
	for _, c := range t.Content {
		if c.Type == "text" {
			parts = append(parts, c.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// Navigate implementa Driver.
func (d *MCPDriver) Navigate(ctx context.Context, url string) error {
	_, err := d.tool(ctx, "browser_navigate", map[string]any{"url": url})
	return err
}

var actionPat = regexp.MustCompile(`^[a-z_]{2,32}$`)

// Interact implementa Driver: a ação vira a ferramenta `browser_<ação>` do servidor MCP.
func (d *MCPDriver) Interact(ctx context.Context, action string, args map[string]string) (string, error) {
	if !actionPat.MatchString(action) {
		return "", ErrInvalidSpec
	}
	a := map[string]any{}
	for k, v := range args {
		a[k] = v
	}
	tr, err := d.tool(ctx, "browser_"+action, a)
	if err != nil {
		return "", err
	}
	return tr.text(), nil
}

// Screenshot implementa Driver (PNG).
func (d *MCPDriver) Screenshot(ctx context.Context, name string) ([]byte, error) {
	tr, err := d.tool(ctx, "browser_take_screenshot", map[string]any{})
	if err != nil {
		return nil, err
	}
	for _, c := range tr.Content {
		if c.Type == "image" && c.Data != "" {
			return base64.StdEncoding.DecodeString(c.Data)
		}
	}
	return nil, errors.New("o servidor MCP não devolveu a imagem")
}

// ConsoleLogs implementa Driver.
func (d *MCPDriver) ConsoleLogs(ctx context.Context) ([]string, error) {
	tr, err := d.tool(ctx, "browser_console_messages", map[string]any{})
	if err != nil {
		return nil, err
	}
	t := strings.TrimSpace(tr.text())
	if t == "" {
		return []string{}, nil
	}
	return strings.Split(t, "\n"), nil
}

// Close encerra o servidor MCP.
func (d *MCPDriver) Close() {
	_ = d.in.Close()
	if d.cmd.Process != nil {
		_ = d.cmd.Process.Kill()
	}
	_ = d.cmd.Wait()
}
