package executor

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/gateways"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
	"github.com/ViniciusBernucci/le-fabrique/internal/providers"
)

type capRunner struct {
	out string
	cmd providers.Command
}

type capProc struct{ r io.Reader }

func (p capProc) Stdout() io.Reader          { return p.r }
func (p capProc) Wait() (int, error)         { return 0, nil }
func (p capProc) Stderr() string             { return "" }
func (p capProc) Cancel(time.Duration) error { return nil }
func (r *capRunner) Start(_ context.Context, c providers.Command, _ string) (providers.Process, error) {
	r.cmd = c
	return capProc{strings.NewReader(r.out)}, nil
}

type fakeChat struct{ prompts []string }

func (f *fakeChat) Chat(_ context.Context, r gateways.ChatRequest) (gateways.ChatResponse, error) {
	for _, m := range r.Messages {
		if m.Role == "user" {
			f.prompts = append(f.prompts, m.Content)
		}
	}
	return gateways.ChatResponse{Text: "feito", Usage: gateways.Usage{PromptTokens: 7, CompletionTokens: 3}}, nil
}

func task() Task {
	return Task{Contract: ContractRef{ID: "c1", Scope: "subtask:s1", Version: 2, Hash: strings.Repeat("a", 64)}, Instructions: "implemente a tela",
		Skill: &Skill{Name: "react", Version: "1", Body: "use tokens"}, Mode: providers.ModeWorkspaceWrite, Dir: "/workspace"}
}

var acct = providers.Account{ID: "0f0f0f0f-0f0f-4f0f-8f0f-0f0f0f0f0f0f", Name: "c", CLI: providers.Claude}

func TestTaskNeedsAContractAndHeaderIsTheSameForBothKinds(t *testing.T) {
	bad := task()
	bad.Contract = ContractRef{}
	if err := bad.Validate(); !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("%v", err)
	}
	tk := task()
	if err := tk.Validate(); err != nil {
		t.Fatal(err)
	}
	cli := &capRunner{out: `{"type":"result","result":"ok","usage":{}}` + "\n"}
	ca := &CLIAdapter{Runner: cli, Account: acct, Caps: CLICapabilities(providers.Claude, nil)}
	if _, err := ca.Run(context.Background(), tk, nil, nil); err != nil {
		t.Fatal(err)
	}
	chat := &fakeChat{}
	aa := &APIAdapter{Gateway: chat, Model: models.Model{Provider: "api", State: "active", Name: "m"}}
	if _, err := aa.Run(context.Background(), tk, nil, nil); err != nil {
		t.Fatal(err)
	}
	if cli.cmd.Stdin != chat.prompts[0] || !strings.HasPrefix(cli.cmd.Stdin, tk.Header()) || !strings.Contains(cli.cmd.Stdin, "[skill react 1]") {
		t.Fatalf("formato diferente:\n%q\n%q", cli.cmd.Stdin, chat.prompts[0])
	}
}

func TestCLICapabilitiesAreConservativeAndConfigurable(t *testing.T) {
	c := CLICapabilities(providers.Claude, nil)
	if !c.NativeSession || !c.NativeResume || c.Interaction != InteractionNextPoint || !c.Verified || c.VerifiedWith != VerifiedClaude || c.Kind != CLI {
		t.Fatalf("%+v", c)
	}
	// O Codex expõe o thread_id e retoma por `exec resume` (verificado), mas não tem entrada em fluxo.
	x := CLICapabilities(providers.Codex, nil)
	if !x.NativeSession || !x.NativeResume || !x.Verified || x.VerifiedWith != VerifiedCodex {
		t.Fatalf("Codex: %+v", x)
	}
	if ic := CLIInteractiveCapabilities(providers.Claude, nil); ic.Interaction != InteractionImmediate {
		t.Fatalf("Claude em fluxo: %+v", ic)
	}
	if ic := CLIInteractiveCapabilities(providers.Codex, nil); ic.Interaction != InteractionNextPoint {
		t.Fatalf("Codex não tem entrada em fluxo: %+v", ic)
	}
	imm, yes, no := InteractionImmediate, true, false
	c = CLICapabilities(providers.Claude, &Override{Interaction: &imm, Verified: &yes, NativeResume: &no, Subagents: &yes})
	if c.Interaction != InteractionImmediate || !c.Verified || c.NativeResume || !c.Subagents {
		t.Fatalf("%+v", c)
	}
	// A configuração pode desligar a retomada, mas nunca ligá-la sem sessão nativa.
	c = CLICapabilities(providers.Codex, &Override{NativeSession: &no, NativeResume: &yes})
	if c.NativeResume {
		t.Fatalf("%+v", c)
	}
	if a := APICapabilities(); a.Kind != API || a.NativeSession || !a.StructuredEvents {
		t.Fatalf("%+v", a)
	}
}

func TestCLIAdapterExposesTheNativeSessionAndResumes(t *testing.T) {
	out := `{"type":"system","session_id":"sess-abcdef12"}
{"type":"assistant","message":{"content":[{"type":"text","text":"olá"},{"type":"tool_use","name":"Edit"}]}}
{"type":"result","result":"pronto","usage":{"input_tokens":3,"output_tokens":2}}
`
	rn := &capRunner{out: out}
	a := &CLIAdapter{Runner: rn, Account: acct, Caps: CLICapabilities(providers.Claude, nil)}
	var evs []Event
	res, err := a.Run(context.Background(), task(), nil, func(e Event) { evs = append(evs, e) })
	if err != nil || res.Session.ID != "sess-abcdef12" || !res.Session.Native || res.Final != "pronto" || res.Failed {
		t.Fatalf("%+v %v", res, err)
	}
	for _, arg := range rn.cmd.Args {
		if arg == "--no-session-persistence" {
			t.Fatalf("sessão nativa declarada exige persistência: %v", rn.cmd.Args)
		}
	}
	for i, e := range evs {
		if e.Seq != int64(i+1) || e.Source != CLI || e.At.IsZero() {
			t.Fatalf("sequência/origem: %+v", evs)
		}
	}
	if len(evs) != 5 || evs[2].Tool != "Edit" {
		t.Fatalf("%+v", evs)
	}
	// Retomada nativa passa --resume.
	if _, err := a.Run(context.Background(), task(), &Session{ID: "sess-abcdef12", Native: true}, nil); err != nil {
		t.Fatal(err)
	}
	found := false
	for i, arg := range rn.cmd.Args {
		if arg == "--resume" && rn.cmd.Args[i+1] == "sess-abcdef12" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%v", rn.cmd.Args)
	}
	// Executor sem retomada declarada recusa e a fábrica usa o checkpoint.
	no := false
	b := &CLIAdapter{Runner: rn, Account: acct, Caps: CLICapabilities(providers.Claude, &Override{NativeResume: &no})}
	if _, err := b.Run(context.Background(), task(), &Session{ID: "sess-abcdef12"}, nil); !errors.Is(err, ErrNoNativeResume) {
		t.Fatalf("%v", err)
	}
	// Sem declarar sessão, o id do cliente não é exposto como nativo.
	c := &CLIAdapter{Runner: &capRunner{out: out}, Account: acct, Caps: CLICapabilities(providers.Claude, &Override{NativeSession: &no})}
	if res, _ := c.Run(context.Background(), task(), nil, nil); res.Session.ID != "" {
		t.Fatalf("%+v", res.Session)
	}
}

func TestAPIAdapterEmitsStructuredEventsAndRefusesNativeResume(t *testing.T) {
	a := &APIAdapter{Gateway: &fakeChat{}, Model: models.Model{Provider: "api", State: "active", Name: "m"}}
	var kinds []string
	res, err := a.Run(context.Background(), task(), nil, func(e Event) {
		if e.Source != API {
			t.Fatalf("%+v", e)
		}
		kinds = append(kinds, e.Kind)
	})
	if err != nil || res.Failed || res.Final != "feito" || res.InputTokens != 7 || res.OutputTokens != 3 {
		t.Fatalf("%+v %v", res, err)
	}
	want := []string{EvStart, EvMessage, EvUsage, EvDone}
	if strings.Join(kinds, ",") != strings.Join(want, ",") {
		t.Fatalf("%v", kinds)
	}
	if _, err := a.Run(context.Background(), task(), &Session{ID: "x"}, nil); !errors.Is(err, ErrNoNativeResume) {
		t.Fatalf("%v", err)
	}
	// Modelo inelegível vira falha registrada (não pânico) e termina com evento de erro.
	bad := &APIAdapter{Gateway: &fakeChat{}, Model: models.Model{Provider: "api", State: "disabled", Name: "m"}}
	var last Event
	out, err := bad.Run(context.Background(), task(), nil, func(e Event) { last = e })
	if err != nil || !out.Failed || last.Kind != EvError {
		t.Fatalf("%+v %v %+v", out, err, last)
	}
}
