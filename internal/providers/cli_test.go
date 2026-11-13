package providers

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	acctA = "11111111-1111-4111-8111-111111111111"
	acctB = "22222222-2222-4222-8222-222222222222"
)

func req(cli CLI, mode Mode) Request {
	return Request{Account: Account{ID: acctA, Name: "conta A", CLI: cli}, Prompt: "Implemente a feature X", Mode: mode}
}

func has(args []string, want ...string) bool {
	for i := range args {
		if i+len(want) <= len(args) && reflect.DeepEqual(args[i:i+len(want)], want) {
			return true
		}
	}
	return false
}

func TestClaudeCommandUsesAuthorizedModeAndKeepsPromptOffTheCommandLine(t *testing.T) {
	c, err := BuildCommand(req(Claude, ModeReadOnly))
	if err != nil {
		t.Fatal(err)
	}
	if !has(c.Args, "--permission-mode", "dontAsk") || !has(c.Args, "--allowedTools", "Read,Grep,Glob") ||
		!has(c.Args, "--output-format", "stream-json") || !has(c.Args, "-p") || !has(c.Args, "--no-session-persistence") {
		t.Fatalf("modo somente leitura: %v", c.Args)
	}
	if c.Stdin != "Implemente a feature X" || strings.Contains(strings.Join(c.Args, " "), "feature X") {
		t.Fatal("o prompt vai por stdin, nunca por argumento (apareceria na lista de processos)")
	}
	if c.Env["CLAUDE_CONFIG_DIR"] != AccountMount || len(c.Env) != 2 {
		t.Fatalf("ambiente só com caminhos: %v", c.Env)
	}

	r := req(Claude, ModeWorkspaceWrite)
	r.Model, r.AllowedTools = "claude-opus-5-5", []string{"Bash(make test)"}
	c, err = BuildCommand(r)
	if err != nil {
		t.Fatal(err)
	}
	if !has(c.Args, "--permission-mode", "acceptEdits") || !has(c.Args, "--allowedTools", "Read,Grep,Glob,Edit,Write,Bash(make test)") || !has(c.Args, "--model", "claude-opus-5-5") {
		t.Fatalf("modo de escrita: %v", c.Args)
	}
	for _, a := range c.Args {
		if strings.Contains(a, "bypass") {
			t.Fatalf("o modo que desliga as permissões nunca é usado: %v", c.Args)
		}
	}
}

func TestCodexCommandUsesSandboxModesOnly(t *testing.T) {
	for mode, want := range map[Mode]string{ModeReadOnly: "read-only", ModeWorkspaceWrite: "workspace-write"} {
		c, err := BuildCommand(req(Codex, mode))
		if err != nil {
			t.Fatal(err)
		}
		if !has(c.Args, "--sandbox", want) || !has(c.Args, "--json") || !has(c.Args, "-C", "/workspace") ||
			!has(c.Args, "--ephemeral") || c.Args[len(c.Args)-1] != "-" || c.Env["CODEX_HOME"] != AccountMount {
			t.Fatalf("%s: %v %v", mode, c.Args, c.Env)
		}
		if strings.Contains(strings.Join(c.Args, " "), "danger") || strings.Contains(strings.Join(c.Args, " "), "feature X") {
			t.Fatalf("sem danger-full-access e sem prompt no argv: %v", c.Args)
		}
	}
}

func TestInvalidRequestsAreRefusedBeforeAnyExecution(t *testing.T) {
	long := strings.Repeat("a", maxPrompt+1)
	cases := map[string]func(*Request){
		"conta sem UUID":         func(r *Request) { r.Account.ID = "../etc" },
		"cliente desconhecido":   func(r *Request) { r.Account.CLI = "gemini" },
		"instrução vazia":        func(r *Request) { r.Prompt = "  " },
		"instrução enorme":       func(r *Request) { r.Prompt = long },
		"modo que desliga tudo":  func(r *Request) { r.Mode = "bypassPermissions" },
		"modelo com injeção":     func(r *Request) { r.Model = "x; rm -rf /" },
		"ferramenta com injeção": func(r *Request) { r.Mode = ModeWorkspaceWrite; r.AllowedTools = []string{"Bash;rm"} },
		"extras em somente leitura": func(r *Request) {
			r.AllowedTools = []string{"Edit"}
		},
		"diretório com ..":   func(r *Request) { r.Dir = "/workspace/../etc" },
		"diretório relativo": func(r *Request) { r.Dir = "workspace" },
	}
	for name, mut := range cases {
		r := req(Claude, ModeReadOnly)
		mut(&r)
		if _, err := BuildCommand(r); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: deveria ser recusado (%v)", name, err)
		}
	}
	r := req(Codex, ModeWorkspaceWrite)
	r.AllowedTools = []string{"Edit"}
	if _, err := BuildCommand(r); err == nil {
		t.Error("Codex não tem lista de ferramentas")
	}
	if !CapabilitiesOf(Claude).ToolAllowlist || CapabilitiesOf(Codex).ToolAllowlist || CapabilitiesOf("x").Streaming {
		t.Error("capacidades")
	}
}

func TestParsersNormalizeBothClients(t *testing.T) {
	claude := map[string][]Event{
		`{"type":"system","subtype":"init"}`: {{Kind: EvStart}},
		`{"type":"assistant","message":{"content":[{"type":"text","text":"olá"},{"type":"tool_use","name":"Edit"}]}}`: {{Kind: EvMessage, Text: "olá"}, {Kind: EvTool, Tool: "Edit"}},
		`{"type":"result","is_error":false,"result":"pronto","usage":{"input_tokens":10,"output_tokens":5}}`:          {{Kind: EvUsage, InputTokens: 10, OutputTokens: 5}, {Kind: EvDone, Text: "pronto"}},
		`{"type":"result","is_error":true,"result":"sem cota","usage":{"input_tokens":1}}`:                            {{Kind: EvUsage, InputTokens: 1}, {Kind: EvError, Text: "sem cota"}},
		`{"type":"user","message":{}}`: nil,
		`{"type":"novo-tipo"}`:         {{Kind: EvRaw, Text: `{"type":"novo-tipo"}`}},
		`isto não é json`:              {{Kind: EvRaw, Text: "isto não é json"}},
		`   `:                          nil,
	}
	codex := map[string][]Event{
		`{"type":"thread.started","thread_id":"t1"}`:                                        {{Kind: EvStart, Session: "t1"}},
		`{"type":"item.completed","item":{"type":"agent_message","text":"feito"}}`:          {{Kind: EvMessage, Text: "feito"}},
		`{"type":"item.started","item":{"type":"command_execution","command":"make test"}}`: {{Kind: EvTool, Tool: "shell", Text: "make test"}},
		`{"type":"item.completed","item":{"type":"command_execution","command":"x"}}`:       nil,
		`{"type":"turn.completed","usage":{"input_tokens":7,"output_tokens":3}}`:            {{Kind: EvUsage, InputTokens: 7, OutputTokens: 3}, {Kind: EvDone}},
		`{"type":"turn.failed","error":{"message":"quota"}}`:                                {{Kind: EvError, Text: "quota"}},
		`{"type":"error","message":"falha de rede"}`:                                        {{Kind: EvError, Text: "falha de rede"}},
	}
	for cli, table := range map[CLI]map[string][]Event{Claude: claude, Codex: codex} {
		for line, want := range table {
			if got := ParseLine(cli, line); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %q:\n got %+v\nwant %+v", cli, line, got, want)
			}
		}
	}
	if got := ParseLine(Claude, `{"type":"assistant","message":{"content":[{"type":"text","text":"`+strings.Repeat("x", 5000)+`"}]}}`); len(got[0].Text) > 2100 {
		t.Error("texto longo deve ser truncado")
	}
}

// fakeRunner devolve uma saída pronta sem processo real.
type fakeRunner struct {
	out    string
	code   int
	block  bool // fica pendente até Cancel
	killed chan struct{}
}

type fakeProc struct {
	r *fakeRunner
	p *io.PipeReader
}

func newFake(out string, code int, block bool) *fakeRunner {
	return &fakeRunner{out: out, code: code, block: block, killed: make(chan struct{})}
}

func (f *fakeRunner) Start(_ context.Context, _ Command, _ string) (Process, error) {
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.WriteString(pw, f.out)
		if f.block {
			<-f.killed
		}
		_ = pw.Close()
	}()
	return &fakeProc{f, pr}, nil
}
func (p *fakeProc) Stdout() io.Reader { return p.p }
func (p *fakeProc) Wait() (int, error) {
	if p.r.block {
		return -1, nil
	}
	return p.r.code, nil
}
func (p *fakeProc) Stderr() string { return "" }
func (p *fakeProc) Cancel(time.Duration) error {
	close(p.r.killed)
	return nil
}

func TestRunSummarizesEventsAndFailures(t *testing.T) {
	out := `{"type":"system"}
{"type":"assistant","message":{"content":[{"type":"text","text":"parcial"}]}}
{"type":"result","is_error":false,"result":"pronto","usage":{"input_tokens":10,"output_tokens":5}}
`
	var kinds []string
	res, err := Run(context.Background(), newFake(out, 0, false), req(Claude, ModeReadOnly), func(e Event) { kinds = append(kinds, e.Kind) })
	if err != nil || res.Failed || res.Final != "pronto" || res.InputTokens != 10 || res.OutputTokens != 5 || res.Events != 4 {
		t.Fatalf("%+v %v", res, err)
	}
	if !reflect.DeepEqual(kinds, []string{EvStart, EvMessage, EvUsage, EvDone}) {
		t.Fatalf("eventos: %v", kinds)
	}
	// Código de saída diferente de zero e evento de erro falham a execução.
	if res, _ := Run(context.Background(), newFake("", 3, false), req(Claude, ModeReadOnly), nil); !res.Failed || res.ExitCode != 3 {
		t.Fatalf("saída não zero: %+v", res)
	}
	errOut := `{"type":"turn.failed","error":{"message":"sem cota"}}` + "\n"
	if res, _ := Run(context.Background(), newFake(errOut, 0, false), req(Codex, ModeReadOnly), nil); !res.Failed || res.Failure != "sem cota" {
		t.Fatalf("evento de erro: %+v", res)
	}
	// Pedido inválido não inicia nada.
	bad := req(Claude, ModeReadOnly)
	bad.Mode = "x"
	if _, err := Run(context.Background(), newFake("", 0, false), bad, nil); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("pedido inválido: %v", err)
	}
	// Linha acima do limite não trava nem derruba o processo.
	huge := strings.Repeat("x", maxLine+10) + "\n"
	if res, err := Run(context.Background(), newFake(huge, 0, false), req(Claude, ModeReadOnly), nil); err != nil || !res.Failed || !strings.Contains(res.Failure, "limite") {
		t.Fatalf("linha enorme: %+v %v", res, err)
	}
}

func TestRunCancelsWhenContextEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fr := newFake(`{"type":"system"}`+"\n", 0, true)
	done := make(chan Result, 1)
	go func() {
		res, _ := Run(ctx, fr, req(Claude, ModeReadOnly), nil)
		done <- res
	}()
	time.Sleep(100 * time.Millisecond)
	cancel()
	go func() { // o runner de verdade cancela pelo contexto; aqui o teste simula
		time.Sleep(50 * time.Millisecond)
		close(fr.killed)
	}()
	select {
	case res := <-done:
		if !res.Cancelled || !res.Failed {
			t.Fatalf("%+v", res)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run não terminou após o cancelamento")
	}
}

func TestNativeSessionIsOptInAndResumeIsClaudeOnly(t *testing.T) {
	acct := Account{ID: "0f0f0f0f-0f0f-4f0f-8f0f-0f0f0f0f0f0f", Name: "c", CLI: Claude}
	base := Request{Account: acct, Prompt: "faça", Mode: ModeReadOnly}
	c, err := BuildCommand(base)
	if err != nil || !has(c.Args, "--no-session-persistence") || has(c.Args, "--resume") {
		t.Fatalf("%v %v", c.Args, err)
	}
	base.Persist = true
	if c, _ = BuildCommand(base); has(c.Args, "--no-session-persistence") {
		t.Fatalf("persistente não pode desligar a sessão: %v", c.Args)
	}
	base.ResumeSession = "sess-12345678"
	if c, err = BuildCommand(base); err != nil || !has(c.Args, "--resume", "sess-12345678") || has(c.Args, "--no-session-persistence") {
		t.Fatalf("%v %v", c.Args, err)
	}
	base.ResumeSession = "inválida; rm -rf"
	if _, err := BuildCommand(base); err == nil {
		t.Fatal("sessão inválida deve ser recusada")
	}
	codex := Request{Account: Account{ID: acct.ID, Name: "x", CLI: Codex}, Prompt: "x", Mode: ModeReadOnly, ResumeSession: "sess-12345678"}
	if _, err := BuildCommand(codex); err == nil {
		t.Fatal("retomada nativa não é declarada para o Codex")
	}
}
