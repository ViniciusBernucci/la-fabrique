package providers

import (
	"encoding/json"
	"strings"
)

// Tipos de evento normalizados.
const (
	EvStart   = "start"
	EvMessage = "message"
	EvTool    = "tool"
	EvUsage   = "usage"
	EvError   = "error"
	EvDone    = "done"
	EvRaw     = "raw" // linha que não é um evento conhecido
)

// Event é um evento da execução, igual para os dois clientes.
type Event struct {
	Kind         string `json:"kind"`
	Text         string `json:"text,omitempty"`
	Tool         string `json:"tool,omitempty"`
	InputTokens  int    `json:"input_tokens,omitempty"`
	OutputTokens int    `json:"output_tokens,omitempty"`
}

// ParseLine converte uma linha JSON do cliente em eventos. Os formatos seguem a
// documentação dos clientes e NÃO foram confirmados contra execução real: linhas
// desconhecidas viram EvRaw em vez de falhar.
func ParseLine(cli CLI, line string) []Event {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	var m map[string]any
	if json.Unmarshal([]byte(line), &m) != nil {
		return []Event{{Kind: EvRaw, Text: clip(line)}}
	}
	switch cli {
	case Claude:
		return parseClaude(m, line)
	case Codex:
		return parseCodex(m, line)
	}
	return []Event{{Kind: EvRaw, Text: clip(line)}}
}

func clip(s string) string {
	if len(s) > 2000 {
		return s[:2000] + "…"
	}
	return s
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }
func num(m map[string]any, k string) int {
	f, _ := m[k].(float64)
	return int(f)
}
func sub(m map[string]any, k string) map[string]any { s, _ := m[k].(map[string]any); return s }

func parseClaude(m map[string]any, raw string) []Event {
	switch str(m, "type") {
	case "system":
		return []Event{{Kind: EvStart}}
	case "assistant":
		var out []Event
		content, _ := sub(m, "message")["content"].([]any)
		for _, c := range content {
			b, _ := c.(map[string]any)
			switch str(b, "type") {
			case "text":
				out = append(out, Event{Kind: EvMessage, Text: clip(str(b, "text"))})
			case "tool_use":
				out = append(out, Event{Kind: EvTool, Tool: str(b, "name")})
			}
		}
		return out
	case "result":
		u := sub(m, "usage")
		out := []Event{{Kind: EvUsage, InputTokens: num(u, "input_tokens"), OutputTokens: num(u, "output_tokens")}}
		if failed, _ := m["is_error"].(bool); failed {
			return append(out, Event{Kind: EvError, Text: clip(str(m, "result"))})
		}
		return append(out, Event{Kind: EvDone, Text: clip(str(m, "result"))})
	case "user":
		return nil // resultados de ferramentas: não são eventos da fábrica
	}
	return []Event{{Kind: EvRaw, Text: clip(raw)}}
}

func parseCodex(m map[string]any, raw string) []Event {
	switch str(m, "type") {
	case "thread.started":
		return []Event{{Kind: EvStart}}
	case "item.completed", "item.started":
		it := sub(m, "item")
		switch str(it, "type") {
		case "agent_message":
			if str(m, "type") == "item.completed" {
				return []Event{{Kind: EvMessage, Text: clip(str(it, "text"))}}
			}
			return nil
		case "command_execution":
			if str(m, "type") == "item.started" {
				return []Event{{Kind: EvTool, Tool: "shell", Text: clip(str(it, "command"))}}
			}
			return nil
		}
		return nil
	case "turn.completed":
		u := sub(m, "usage")
		return []Event{{Kind: EvUsage, InputTokens: num(u, "input_tokens"), OutputTokens: num(u, "output_tokens")}, {Kind: EvDone}}
	case "turn.failed":
		return []Event{{Kind: EvError, Text: clip(str(sub(m, "error"), "message"))}}
	case "error":
		return []Event{{Kind: EvError, Text: clip(str(m, "message"))}}
	}
	return []Event{{Kind: EvRaw, Text: clip(raw)}}
}
