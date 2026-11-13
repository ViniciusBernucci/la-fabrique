package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/contract"
	"github.com/ViniciusBernucci/le-fabrique/internal/gateways"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/providers"
)

// Kind é o tipo de executor.
type Kind string

// Tipos.
const (
	CLI Kind = "cli"
	API Kind = "api"
)

// Interaction é como o executor aceita orientação durante a execução.
type Interaction string

// Modos de interação.
const (
	InteractionNone      Interaction = "none"            // não aceita; fica registrada para a retomada
	InteractionNextPoint Interaction = "next_safe_point" // enfileira para o próximo ponto seguro
	InteractionImmediate Interaction = "immediate"       // entrega na hora, com segurança
)

// Capabilities é o que o adaptador declara. Verified diz se a capacidade foi comprovada contra o
// cliente real; declarada e não verificada continua utilizável, mas a seleção manual a vê assim.
type Capabilities struct {
	Kind             Kind        `json:"kind"`
	Streaming        bool        `json:"streaming"`
	StructuredEvents bool        `json:"structured_events"`
	NativeSession    bool        `json:"native_session"` // expõe o id da sessão nativa
	NativeResume     bool        `json:"native_resume"`  // retoma a sessão nativa
	Interaction      Interaction `json:"interaction"`
	Subagents        bool        `json:"subagents"` // identifica subagentes quando o executor os usa
	Skill            bool        `json:"skill"`
	Cancel           bool        `json:"cancel"`
	Verified         bool        `json:"verified"`
}

// Override ajusta as capacidades de um CLI configurável (ex.: depois de verificar o cliente).
type Override struct {
	NativeSession *bool
	NativeResume  *bool
	Interaction   *Interaction
	Subagents     *bool
	Verified      *bool
}

// CLICapabilities devolve as capacidades declaradas do cliente, com ajustes da configuração.
// Os padrões são conservadores e NÃO verificados contra a execução real (NOT_RUN).
func CLICapabilities(c providers.CLI, o *Override) Capabilities {
	base := providers.CapabilitiesOf(c)
	caps := Capabilities{Kind: CLI, Streaming: base.Streaming, StructuredEvents: true, Interaction: InteractionNextPoint, Skill: true, Cancel: base.Cancel}
	if c == providers.Claude {
		caps.NativeSession, caps.NativeResume = true, true
	}
	if o != nil {
		if o.NativeSession != nil {
			caps.NativeSession = *o.NativeSession
		}
		if o.NativeResume != nil {
			caps.NativeResume = *o.NativeResume && caps.NativeSession
		}
		if o.Interaction != nil {
			caps.Interaction = *o.Interaction
		}
		if o.Subagents != nil {
			caps.Subagents = *o.Subagents
		}
		if o.Verified != nil {
			caps.Verified = *o.Verified
		}
	}
	return caps
}

// APICapabilities são as do adaptador de API: eventos estruturados, sem sessão nativa.
func APICapabilities() Capabilities {
	return Capabilities{Kind: API, Streaming: false, StructuredEvents: true, Interaction: InteractionNextPoint, Skill: true, Cancel: true, Verified: true}
}

// ContractRef é a referência ao snapshot imutável do contrato.
type ContractRef struct {
	ID      string `json:"id"`
	Scope   string `json:"scope"`
	Version int    `json:"version"`
	Hash    string `json:"hash"`
}

// RefOf extrai a referência de um snapshot.
func RefOf(s contract.Snapshot) ContractRef {
	return ContractRef{ID: s.ID, Scope: s.Scope, Version: s.Version, Hash: s.Hash}
}

// Skill é a skill entregue ao executor.
type Skill struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Body    string `json:"body"`
}

// Task é o formato único de tarefa, igual para CLI e API.
type Task struct {
	Contract     ContractRef     `json:"contract"`
	Instructions string          `json:"instructions"`
	Skill        *Skill          `json:"skill,omitempty"`
	Mode         providers.Mode  `json:"mode"`
	Model        string          `json:"model"`
	Dir          string          `json:"dir"`
	AllowedTools []string        `json:"allowed_tools,omitempty"`
	Limits       contract.Limits `json:"limits"`
}

// ErrInvalidTask indica tarefa recusada antes de qualquer execução.
var ErrInvalidTask = errors.New("tarefa inválida")

// Validate confere a tarefa. Sem contrato não há despacho.
func (t Task) Validate() error {
	switch {
	case t.Contract.ID == "" || t.Contract.Hash == "" || t.Contract.Version < 1:
		return fmt.Errorf("%w: o contrato é obrigatório no cabeçalho", ErrInvalidTask)
	case strings.TrimSpace(t.Instructions) == "":
		return fmt.Errorf("%w: instruções vazias", ErrInvalidTask)
	case t.Mode != providers.ModeReadOnly && t.Mode != providers.ModeWorkspaceWrite:
		return fmt.Errorf("%w: modo desconhecido", ErrInvalidTask)
	}
	return nil
}

// Header é o cabeçalho determinístico (o mesmo para CLI e API) que referencia o contrato.
func (t Task) Header() string {
	return fmt.Sprintf("[contrato %s v%d hash=%s escopo=%s]", t.Contract.ID, t.Contract.Version, t.Contract.Hash, t.Contract.Scope)
}

// Prompt monta o texto entregue ao executor: cabeçalho, skill e instruções.
func (t Task) Prompt() string {
	var b strings.Builder
	b.WriteString(t.Header())
	b.WriteString("\n")
	if t.Skill != nil {
		fmt.Fprintf(&b, "\n[skill %s %s]\n%s\n", t.Skill.Name, t.Skill.Version, t.Skill.Body)
	}
	b.WriteString("\n")
	b.WriteString(t.Instructions)
	return b.String()
}

// Eventos normalizados.
const (
	EvStart   = "start"
	EvMessage = "message"
	EvTool    = "tool"
	EvUsage   = "usage"
	EvError   = "error"
	EvDone    = "done"
	EvRaw     = "raw"
)

// Event é o evento único dos dois tipos de executor. Seq é monotônico dentro da execução.
type Event struct {
	Seq          int64     `json:"seq"`
	At           time.Time `json:"at"`
	Source       Kind      `json:"source"`
	Kind         string    `json:"kind"`
	Text         string    `json:"text,omitempty"`
	Tool         string    `json:"tool,omitempty"`
	Outcome      string    `json:"outcome,omitempty"`
	Subagent     string    `json:"subagent,omitempty"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
}

// Session é a sessão nativa do executor, quando existe.
type Session struct {
	ID     string `json:"id"`
	Native bool   `json:"native"`
}

// Outcome é o resultado de uma execução.
type Outcome struct {
	Session      Session `json:"session"`
	Final        string  `json:"final"`
	Failed       bool    `json:"failed"`
	Cancelled    bool    `json:"cancelled"`
	Failure      string  `json:"failure,omitempty"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Events       int64   `json:"events"`
}

// Adapter é o contrato dos executores.
type Adapter interface {
	Capabilities() Capabilities
	// Run executa a tarefa. resume (opcional) pede a retomada da sessão nativa; o adaptador que não a
	// declara recusa com ErrNoNativeResume e quem chama usa o checkpoint.
	Run(ctx context.Context, t Task, resume *Session, emit func(Event)) (Outcome, error)
}

// ErrNoNativeResume indica que o executor não retoma sessão nativa.
var ErrNoNativeResume = errors.New("o executor não declara retomada de sessão nativa")

type sequencer struct {
	n    atomic.Int64
	emit func(Event)
	now  func() time.Time
	src  Kind
}

func (s *sequencer) send(e Event) {
	e.Seq, e.Source = s.n.Add(1), s.src
	if s.now != nil {
		e.At = s.now().UTC()
	} else {
		e.At = time.Now().UTC()
	}
	if s.emit != nil {
		s.emit(e)
	}
}

// CLIAdapter executa por um cliente de linha de comando (no sandbox, via Runner).
type CLIAdapter struct {
	Runner  providers.Runner
	Account providers.Account
	Caps    Capabilities
	Now     func() time.Time
}

// Capabilities implementa Adapter.
func (a *CLIAdapter) Capabilities() Capabilities { return a.Caps }

// Run implementa Adapter.
func (a *CLIAdapter) Run(ctx context.Context, t Task, resume *Session, emit func(Event)) (Outcome, error) {
	if err := t.Validate(); err != nil {
		return Outcome{}, err
	}
	req := providers.Request{Account: a.Account, Prompt: t.Prompt(), Mode: t.Mode, Model: t.Model, Dir: t.Dir, AllowedTools: t.AllowedTools,
		Persist: a.Caps.NativeSession}
	if resume != nil && resume.ID != "" {
		if !a.Caps.NativeResume {
			return Outcome{}, ErrNoNativeResume
		}
		req.ResumeSession = resume.ID
	}
	sq := &sequencer{emit: emit, now: a.Now, src: CLI}
	res, err := providers.Run(ctx, a.Runner, req, func(e providers.Event) {
		sq.send(Event{Kind: e.Kind, Text: e.Text, Tool: e.Tool, InputTokens: e.InputTokens, OutputTokens: e.OutputTokens})
	})
	out := Outcome{Final: res.Final, Failed: res.Failed, Cancelled: res.Cancelled, Failure: res.Failure,
		InputTokens: res.InputTokens, OutputTokens: res.OutputTokens, Events: sq.n.Load()}
	if res.Session != "" && a.Caps.NativeSession {
		out.Session = Session{ID: res.Session, Native: true}
	}
	return out, err
}

// APIAdapter executa por gateway de API, sob os mesmos limites e o mesmo formato de tarefa.
type APIAdapter struct {
	Gateway gateways.Chatter
	Model   models.Model
	Exec    gateways.Executor
	Tools   []gateways.ToolSpec
	Meter   gateways.Meter
	Redact  func(string) string
	Now     func() time.Time
}

// Capabilities implementa Adapter.
func (a *APIAdapter) Capabilities() Capabilities { return APICapabilities() }

// Run implementa Adapter. A API não tem sessão nativa: resume é recusado.
func (a *APIAdapter) Run(ctx context.Context, t Task, resume *Session, emit func(Event)) (Outcome, error) {
	if err := t.Validate(); err != nil {
		return Outcome{}, err
	}
	if resume != nil && resume.ID != "" {
		return Outcome{}, ErrNoNativeResume
	}
	sq := &sequencer{emit: emit, now: a.Now, src: API}
	sq.send(Event{Kind: EvStart})
	maxTurns := t.Limits.MaxRounds
	res, err := gateways.Run(ctx, a.Gateway, gateways.RunConfig{Model: a.Model, Prompt: t.Prompt(), Tools: a.Tools, Exec: a.Exec, Meter: a.Meter,
		Redact: a.Redact, MaxTurns: maxTurns, OnEvent: func(e gateways.Event) {
			switch e.Kind {
			case gateways.EventMessage:
				sq.send(Event{Kind: EvMessage, Text: e.Text})
			case gateways.EventTool:
				sq.send(Event{Kind: EvTool, Tool: e.Tool, Outcome: e.Outcome})
			case gateways.EventUsage:
				sq.send(Event{Kind: EvUsage, InputTokens: e.InputTokens, OutputTokens: e.OutputTokens})
			}
		}})
	out := Outcome{Final: res.Text, InputTokens: res.Usage.PromptTokens, OutputTokens: res.Usage.CompletionTokens}
	if err != nil {
		out.Failed, out.Failure = true, err.Error()
		sq.send(Event{Kind: EvError, Text: err.Error()})
		out.Events = sq.n.Load()
		if errors.Is(err, context.Canceled) {
			out.Cancelled = true
		}
		return out, nil
	}
	sq.send(Event{Kind: EvDone, Text: res.Text})
	out.Events = sq.n.Load()
	return out, nil
}
