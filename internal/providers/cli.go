package providers

import (
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
}

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
	ModelSelect    bool `json:"model_select"`
	Cancel         bool `json:"cancel"`
}

// CapabilitiesOf devolve as capacidades do cliente.
func CapabilitiesOf(c CLI) Capabilities {
	switch c {
	case Claude:
		return Capabilities{Streaming: true, ReadOnly: true, WorkspaceWrite: true, ToolAllowlist: true, ModelSelect: true, Cancel: true}
	case Codex:
		return Capabilities{Streaming: true, ReadOnly: true, WorkspaceWrite: true, ModelSelect: true, Cancel: true}
	}
	return Capabilities{}
}

// BuildCommand valida o pedido e monta o comando do cliente. As flags seguem o
// `--help` das versões instaladas em 2026-11-12; a execução real contra a conta
// de um responsável não foi feita (NOT_RUN).
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
		cmd.Args = []string{bin, "-p", "--output-format", "stream-json", "--verbose", "--no-session-persistence",
			"--permission-mode", perm, "--allowedTools", strings.Join(tools, ",")}
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
		cmd.Args = []string{bin, "exec", "--json", "--skip-git-repo-check", "--ephemeral", "--sandbox", sandbox, "-C", dir}
		if req.Model != "" {
			cmd.Args = append(cmd.Args, "--model", req.Model)
		}
		cmd.Args = append(cmd.Args, "-") // o prompt vem de stdin
	}
	return cmd, nil
}
