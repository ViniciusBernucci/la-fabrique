package providers

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// CLI identifica o cliente de IA.
type CLI string

const (
	Claude CLI = "claude"
	Codex  CLI = "codex"
)

// Mode é o modo de acesso pedido ao cliente. Os modos que desligam a proteção
// do próprio cliente (bypassPermissions, danger-full-access) NÃO existem aqui:
// a fronteira é o sandbox, e a política decide o que cada agente pode.
type Mode string

const (
	ModeReadOnly       Mode = "read_only"
	ModeWorkspaceWrite Mode = "workspace_write"
)

// AccountMount é onde o volume da conta aparece dentro do sandbox.
const AccountMount = "/accounts/current"

// Account é uma conta nomeada de um CLI. ID é o da conexão no cofre (tipo cli).
// Duas contas do mesmo provedor têm volumes (e portanto logins) distintos.
type Account struct {
	ID   string
	Name string
	CLI  CLI
}

var accountID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// VolumeName é o volume Docker que guarda a configuração e o login da conta.
func (a Account) VolumeName() string { return "lf-acct-" + a.ID }

// Valid confere o ID (UUID) e o cliente.
func (a Account) Valid() bool {
	return accountID.MatchString(a.ID) && (a.CLI == Claude || a.CLI == Codex)
}

// Request descreve uma execução.
type Request struct {
	Account      Account
	Prompt       string
	Mode         Mode
	Model        string
	Dir          string   // diretório de trabalho no sandbox (padrão /workspace)
	AllowedTools []string // Claude: ferramentas extras no modo de escrita, ex.: "Bash(make test)"
	Binary       string   // sobrescreve o executável (testes)
	// Persist mantém a sessão nativa do cliente (padrão: efêmera, como antes). ResumeSession retoma
	// uma sessão nativa já existente (implica Persist). Só o Claude declara suporte à retomada.
	Persist       bool
	ResumeSession string
	// NewSessionID (só Claude, UUID) fixa o id da sessão nativa desde o início (`--session-id`), verificado
	// contra o cliente real: o init devolve exatamente esse id. O Codex não aceita; o id vem de `thread.started`.
	NewSessionID string
	// Interactive (só Claude) abre a entrada em fluxo (`--input-format stream-json`): orientações entram no meio
	// da execução e o cliente as reconhece (`--replay-user-messages`). O prompt segue por stdin como mensagem JSON
	// e o stdin permanece aberto para o chamador.
	Interactive bool
}

var uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// UserMessage é a mensagem de usuário (uma linha JSON) aceita pelo Claude em `--input-format stream-json`.
func UserMessage(text string) string {
	b, _ := json.Marshal(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": text}})
	return string(b)
}

var sessionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{7,63}$`)

// Command é o que o executor roda: argumentos, ambiente (só caminhos, nenhum
// segredo) e o prompt, entregue por stdin.
type Command struct {
	Args  []string
	Env   map[string]string
	Stdin string
}

var (
	modelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]{0,79}$`)
	toolPattern  = regexp.MustCompile(`^[A-Za-z]+(\([A-Za-z0-9 :*._/-]{1,80}\))?$`)
	dirPattern   = regexp.MustCompile(`^/[A-Za-z0-9_./-]{0,200}$`)
)

// ErrInvalidRequest indica pedido recusado antes de qualquer execução.
var ErrInvalidRequest = errors.New("pedido de execução inválido")

const maxPrompt = 100 << 10

// Capabilities descreve o que o adaptador de cada cliente oferece.
type Capabilities struct {
	Streaming      bool `json:"streaming"`
	ReadOnly       bool `json:"read_only"`
	WorkspaceWrite bool `json:"workspace_write"`
	ToolAllowlist  bool `json:"tool_allowlist"`
	NativeSession  bool `json:"native_session"` // informa o id da sessão nativa (verificado)
	NativeResume   bool `json:"native_resume"`  // retoma a sessão nativa (verificado)
	InteractiveIn  bool `json:"interactive_in"` // aceita entrada no meio da execução com reconhecimento (verificado)
	ModelSelect    bool `json:"model_select"`
	Cancel         bool `json:"cancel"`
}

// CapabilitiesOf devolve as capacidades do cliente.
func CapabilitiesOf(c CLI) Capabilities {
	switch c {
	case Claude:
		return Capabilities{Streaming: true, ReadOnly: true, WorkspaceWrite: true, ToolAllowlist: true, ModelSelect: true, Cancel: true, NativeSession: true, NativeResume: true, InteractiveIn: true}
	case Codex:
		return Capabilities{Streaming: true, ReadOnly: true, WorkspaceWrite: true, ModelSelect: true, Cancel: true, NativeSession: true, NativeResume: true}
	}
	return Capabilities{}
}

// BuildCommand valida o pedido e monta o comando do cliente. As flags de sessão, retomada e entrada em fluxo
// foram verificadas contra os clientes reais (claude 2.1.323 e codex-cli 0.164.0; ver live_test.go).
func BuildCommand(req Request) (Command, error) {
	bad := func(why string) (Command, error) { return Command{}, fmt.Errorf("%w: %s", ErrInvalidRequest, why) }
	if !req.Account.Valid() {
		return bad("conta inválida")
	}
	if strings.TrimSpace(req.Prompt) == "" || len(req.Prompt) > maxPrompt {
		return bad("instrução vazia ou grande demais")
	}
	if req.Mode != ModeReadOnly && req.Mode != ModeWorkspaceWrite {
		return bad("modo desconhecido")
	}
	if req.Model != "" && !modelPattern.MatchString(req.Model) {
		return bad("modelo inválido")
	}
	dir := req.Dir
	if dir == "" {
		dir = "/workspace"
	}
	if !dirPattern.MatchString(dir) || strings.Contains(dir, "..") {
		return bad("diretório inválido")
	}
	for _, t := range req.AllowedTools {
		if !toolPattern.MatchString(t) {
			return bad("ferramenta inválida")
		}
	}
	if req.ResumeSession != "" {
		if !sessionPattern.MatchString(req.ResumeSession) {
			return bad("sessão inválida")
		}
	}
	if req.NewSessionID != "" {
		if req.Account.CLI != Claude || req.ResumeSession != "" || !uuidPattern.MatchString(req.NewSessionID) {
			return bad("session id fixo: só Claude, UUID, e não junto da retomada")
		}
	}
	if req.Interactive && req.Account.CLI != Claude {
		return bad("a entrada em fluxo só foi verificada para o Claude")
	}
	persist := req.Persist || req.ResumeSession != ""
	bin := req.Binary
	if bin == "" {
		bin = string(req.Account.CLI)
	}
	cmd := Command{Stdin: req.Prompt, Env: map[string]string{"HOME": "/tmp"}}
	switch req.Account.CLI {
	case Claude:
		cmd.Env["CLAUDE_CONFIG_DIR"] = AccountMount
		tools := []string{"Read", "Grep", "Glob"}
		perm := "dontAsk" // nega tudo o que não estiver na lista
		if req.Mode == ModeWorkspaceWrite {
			perm = "acceptEdits"
			tools = append(tools, "Edit", "Write")
			tools = append(tools, req.AllowedTools...)
		} else if len(req.AllowedTools) > 0 {
			return bad("ferramentas extras exigem o modo de escrita")
		}
		cmd.Args = []string{bin, "-p", "--output-format", "stream-json", "--verbose"}
		if req.Interactive {
			cmd.Args = append(cmd.Args, "--input-format", "stream-json", "--replay-user-messages")
			cmd.Stdin = UserMessage(req.Prompt)
		}
		if !persist {
			cmd.Args = append(cmd.Args, "--no-session-persistence")
		}
		if req.ResumeSession != "" {
			cmd.Args = append(cmd.Args, "--resume", req.ResumeSession)
		}
		if req.NewSessionID != "" {
			cmd.Args = append(cmd.Args, "--session-id", req.NewSessionID)
		}
		cmd.Args = append(cmd.Args, "--permission-mode", perm, "--allowedTools", strings.Join(tools, ","))
		if req.Model != "" {
			cmd.Args = append(cmd.Args, "--model", req.Model)
		}
	case Codex:
		if len(req.AllowedTools) > 0 {
			return bad("o Codex não tem lista de ferramentas")
		}
		cmd.Env["CODEX_HOME"] = AccountMount
		sandbox := "read-only"
		if req.Mode == ModeWorkspaceWrite {
			sandbox = "workspace-write"
		}
		if req.ResumeSession != "" {
			// `codex exec resume` NÃO aceita --sandbox nem -C (verificado): o modo vai por -c sandbox_mode e o
			// diretório é o do processo (o runner usa Request.Dir).
			cmd.Args = []string{bin, "exec", "resume", req.ResumeSession, "--json", "--skip-git-repo-check", "-c", `sandbox_mode="` + sandbox + `"`}
			if req.Model != "" {
				cmd.Args = append(cmd.Args, "--model", req.Model)
			}
			cmd.Args = append(cmd.Args, "-")
			return cmd, nil
		}
		cmd.Args = []string{bin, "exec", "--json", "--skip-git-repo-check"}
		if !persist {
			cmd.Args = append(cmd.Args, "--ephemeral")
		}
		cmd.Args = append(cmd.Args, "--sandbox", sandbox, "-C", dir)
		if req.Model != "" {
			cmd.Args = append(cmd.Args, "--model", req.Model)
		}
		cmd.Args = append(cmd.Args, "-") // o prompt vem de stdin
	}
	return cmd, nil
}
