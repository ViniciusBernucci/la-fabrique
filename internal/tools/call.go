package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

// Erros que impedem a chamada antes de qualquer efeito.
var (
	ErrUnknownTool = errors.New("ferramenta desconhecida")
	ErrNotGranted  = errors.New("ferramenta não concedida a este agente")
	ErrScope       = errors.New("escopo insuficiente para a ferramenta")
	ErrNotReady    = errors.New("o ambiente não está configurado para esta chamada")
	ErrEscape      = errors.New("o caminho resolvido (link simbólico) sai do workspace")
	ErrIdentity    = errors.New("chamada sem identidade (agente e execução)")
)

// Grant é o que foi concedido ao agente: quais ferramentas e quais permissões.
// Uma ferramenta recém-registrada ou descoberta não está em nenhum Grant.
type Grant struct {
	Tools       []string
	Permissions []string
}

func in(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// Identity diz QUEM faz a chamada: o agente e a execução. Chamadas brokeradas
// (integrações) exigem identidade; as demais a registram na trilha da política.
type Identity struct{ Agent, Run string }

func (i Identity) valid() bool { return i.Agent != "" && i.Run != "" }

func (i Identity) String() string { return "agent:" + i.Agent + " run:" + i.Run }

type identityKey struct{}

// WithIdentity anexa a identidade ao contexto da chamada (o conector a lê com IdentityFrom).
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// IdentityFrom lê a identidade do contexto.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok && id.valid()
}

// ConnectorRunner executa a operação de uma ferramenta de integração.
type ConnectorRunner func(ctx context.Context, operation string, args map[string]any) (map[string]any, error)

// Env é o ambiente de execução. Sandbox e Guard são obrigatórios: sem política
// nenhuma ferramenta roda.
type Env struct {
	Sandbox   *execution.Sandbox
	Guard     *policy.Guard
	Project   string
	Connector ConnectorRunner
}

// Result é a saída tipada de uma chamada.
type Result struct {
	Output    map[string]any `json:"output"`
	Truncated bool           `json:"truncated"`
}

// Registry reúne as ferramentas nativas e as personalizadas (via Loader).
type Registry struct {
	natives map[string]Tool
	custom  func(ctx context.Context, name string) (Tool, error)
}

// NewRegistry cria o registro. custom pode ser nil (só nativas).
func NewRegistry(custom func(ctx context.Context, name string) (Tool, error)) *Registry {
	r := &Registry{natives: map[string]Tool{}, custom: custom}
	for _, t := range Natives() {
		if err := t.Compile(); err != nil {
			panic("ferramenta nativa inválida: " + t.Name + ": " + err.Error())
		}
		r.natives[t.Name] = t
	}
	return r
}

// Get localiza a ferramenta pelo nome (nativas primeiro).
func (r *Registry) Get(ctx context.Context, name string) (Tool, error) {
	if t, ok := r.natives[name]; ok {
		return t, nil
	}
	if r.custom != nil {
		if t, err := r.custom(ctx, name); err == nil {
			if cerr := t.Compile(); cerr != nil {
				return Tool{}, cerr
			}
			return t, nil
		}
	}
	return Tool{}, fmt.Errorf("%w: %q", ErrUnknownTool, name)
}

// Check faz tudo o que vem antes do efeito: existência, concessão, escopo e
// validação dos argumentos. Não executa nada.
func (r *Registry) Check(ctx context.Context, name string, args map[string]any, g Grant) (Tool, map[string]any, error) {
	t, err := r.Get(ctx, name)
	if err != nil {
		return Tool{}, nil, err
	}
	if !in(g.Tools, name) {
		return Tool{}, nil, fmt.Errorf("%w: %q", ErrNotGranted, name)
	}
	for _, p := range t.Permissions {
		if !in(g.Permissions, p) {
			return Tool{}, nil, fmt.Errorf("%w: exige a permissão %q", ErrScope, p)
		}
	}
	clean, err := t.Validate(args)
	if err != nil {
		return Tool{}, nil, err
	}
	return t, clean, nil
}

func str(m map[string]any, k string) string { s, _ := m[k].(string); return s }

// action traduz a chamada na ação que a política avalia.
func action(project string, t Tool, args map[string]any) policy.Action {
	a := policy.Action{Project: project, Resource: str(args, "path"), Params: map[string]any{"tool": t.Name}}
	switch t.Name {
	case "read_file":
		a.Tool, a.Operation = policy.ToolFS, "read"
	case "write_file", "edit_file":
		a.Tool, a.Operation = policy.ToolFS, "write"
	case "list_dir":
		a.Tool, a.Operation = policy.ToolFS, "list"
	case "search":
		a.Tool, a.Operation = policy.ToolFS, "search"
	case "run_command":
		a.Tool, a.Operation = policy.ToolScript, "run"
		a.Params["command"] = str(args, "command")
	case "git_status":
		a.Tool, a.Operation = policy.ToolGit, "status"
	case "git_diff":
		a.Tool, a.Operation = policy.ToolGit, "diff"
	default:
		switch t.Kind {
		case KindScript:
			a.Tool, a.Operation = policy.ToolScript, "run"
			a.Params["command"] = t.Command
		case KindIntegration:
			a.Tool, a.Operation = policy.ToolConnector, t.Operation
		}
	}
	return a
}

// Call executa a ferramenta: Check, política (via Guard) e execução com os
// limites; a saída volta validada contra o esquema.
func (r *Registry) Call(ctx context.Context, env Env, name string, args map[string]any, g Grant) (Result, error) {
	t, clean, err := r.Check(ctx, name, args, g)
	if err != nil {
		return Result{}, err
	}
	if env.Guard == nil {
		return Result{}, fmt.Errorf("%w: política ausente", ErrNotReady)
	}
	if t.Kind != KindIntegration && env.Sandbox == nil {
		return Result{}, fmt.Errorf("%w: sandbox ausente", ErrNotReady)
	}
	id, hasID := IdentityFrom(ctx)
	if t.Kind == KindIntegration && !hasID {
		return Result{}, ErrIdentity
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(t.Limits.TimeoutSeconds)*time.Second)
	defer cancel()
	var res Result
	act := action(env.Project, t, clean)
	if hasID {
		act.Params["actor"] = id.String()
	}
	_, err = env.Guard.Do(ctx, act, func() error {
		var rerr error
		res, rerr = r.run(ctx, env, t, clean)
		return rerr
	})
	return res, err
}

func capText(s string, max int) (string, bool) {
	if len(s) > max {
		return s[:max], true
	}
	return s, false
}

func lines(s string) []string {
	out := []string{}
	for _, l := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

// exec roda um comando por argv (sem shell), com o limite de tempo da ferramenta.
func exec(ctx context.Context, env Env, argv []string, stdin []byte, extraEnv map[string]string) (execution.ExecResult, error) {
	spec := execution.ExecSpec{Cmd: argv, Env: extraEnv}
	if stdin != nil {
		spec.Stdin = bytes.NewReader(stdin)
	}
	res, err := env.Sandbox.Exec(ctx, spec)
	if err == nil && res.TimedOut {
		err = errors.New("tempo limite excedido")
	}
	return res, err
}

// confineSh resolve o caminho ($0) seguindo links simbólicos (inclusive o trecho
// que ainda não existe, a partir do ancestral existente mais próximo) e SÓ segue
// se o resultado estiver dentro do workspace; os comandos seguintes usam o caminho
// já resolvido ($rp). A resolução e o uso acontecem no mesmo processo, o que
// estreita a janela de corrida. Código 77 = o caminho sai do workspace.
const confineSh = `resolve() { p="$1"; while [ ! -e "$p" ] && [ ! -L "$p" ]; do p=$(dirname -- "$p"); done; b=$(readlink -f -- "$p") || return 1; printf '%s%s' "$b" "${1#"$p"}"; }
rp=$(resolve "$0") || exit 77
case "$rp" in /workspace|/workspace/*) ;; *) exit 77;; esac
`

const exitEscape = 77

// fsExec roda o corpo confinado: $0 = caminho, $1... = extras.
func fsExec(ctx context.Context, env Env, body, path string, stdin []byte, extra ...string) (execution.ExecResult, error) {
	return exec(ctx, env, append([]string{"sh", "-c", confineSh + body, path}, extra...), stdin, nil)
}

// fsFail traduz a saída de fsExec: escape de workspace vira ErrEscape; o resto, a mensagem do chamador.
func fsFail(out execution.ExecResult, err error, msg string) error {
	if err == nil && out.ExitCode == exitEscape {
		return ErrEscape
	}
	if err != nil || out.ExitCode != 0 {
		return errors.New(msg)
	}
	return nil
}

func (r *Registry) run(ctx context.Context, env Env, t Tool, a map[string]any) (Result, error) {
	max := t.Limits.MaxOutputBytes
	switch t.Name {
	case "read_file":
		limit := 262144
		if v, ok := a["max_bytes"].(int); ok {
			limit = v
		}
		out, err := fsExec(ctx, env, `head -c "$1" -- "$rp"`, str(a, "path"), nil, fmt.Sprint(limit+1))
		if ferr := fsFail(out, err, "não foi possível ler o arquivo"); ferr != nil {
			return Result{}, ferr
		}
		content, trunc := capText(out.Stdout, limit)
		return typed(t, map[string]any{"content": content, "truncated": trunc, "bytes": len(content)}, trunc)
	case "write_file":
		content := str(a, "content")
		out, err := fsExec(ctx, env, `mkdir -p -- "$(dirname -- "$rp")" && cat > "$rp"`, str(a, "path"), []byte(content))
		if ferr := fsFail(out, err, "não foi possível gravar o arquivo"); ferr != nil {
			return Result{}, ferr
		}
		return typed(t, map[string]any{"path": a["path"], "bytes": len(content)}, false)
	case "edit_file":
		cur, err := fsExec(ctx, env, `head -c "$1" -- "$rp"`, str(a, "path"), nil, fmt.Sprint(1<<20+1))
		if ferr := fsFail(cur, err, "não foi possível ler o arquivo"); ferr != nil {
			return Result{}, ferr
		}
		if len(cur.Stdout) > 1<<20 {
			return Result{}, errors.New("o arquivo é grande demais para edição")
		}
		old := str(a, "old")
		if old == "" {
			return Result{}, errors.New("o trecho a trocar não pode ser vazio")
		}
		switch n := strings.Count(cur.Stdout, old); {
		case n == 0:
			return Result{}, errors.New("o trecho não foi encontrado no arquivo")
		case n > 1:
			return Result{}, fmt.Errorf("o trecho aparece %d vezes; informe um trecho único", n)
		}
		updated := strings.Replace(cur.Stdout, old, str(a, "new"), 1)
		out, err := fsExec(ctx, env, `cat > "$rp"`, str(a, "path"), []byte(updated))
		if ferr := fsFail(out, err, "não foi possível gravar o arquivo"); ferr != nil {
			return Result{}, ferr
		}
		return typed(t, map[string]any{"path": a["path"], "replacements": 1}, false)
	case "list_dir":
		out, err := fsExec(ctx, env, `ls -1Ap -- "$rp"`, str(a, "path"), nil)
		if ferr := fsFail(out, err, "não foi possível listar o diretório"); ferr != nil {
			return Result{}, ferr
		}
		return typed(t, map[string]any{"entries": lines(out.Stdout)}, false)
	case "search":
		p := str(a, "path")
		if p == "" {
			p = workspace
		}
		out, err := fsExec(ctx, env, `grep -rnI -m 50 -e "$1" -- "$rp"`, p, nil, str(a, "pattern"))
		if err == nil && out.ExitCode == exitEscape {
			return Result{}, ErrEscape
		}
		if err != nil || out.ExitCode > 1 { // 1 = nenhuma ocorrência
			return Result{}, errors.New("a busca falhou")
		}
		ls := lines(out.Stdout)
		trunc := len(ls) > 200
		if trunc {
			ls = ls[:200]
		}
		return typed(t, map[string]any{"matches": ls, "truncated": trunc}, trunc)
	case "run_command":
		out, err := exec(ctx, env, []string{"sh", "-c", str(a, "command")}, nil, nil)
		if err != nil {
			if strings.Contains(err.Error(), "tempo limite") {
				return Result{}, err
			}
			return Result{}, errors.New("não foi possível executar o comando")
		}
		so, t1 := capText(out.Stdout, max)
		se, t2 := capText(out.Stderr, max)
		return typed(t, map[string]any{"exit_code": out.ExitCode, "stdout": so, "stderr": se}, t1 || t2)
	case "git_status":
		out, err := exec(ctx, env, []string{"git", "status", "--porcelain"}, nil, nil)
		return gitResult(t, "output", out, err, max)
	case "git_diff":
		var out execution.ExecResult
		var err error
		if p := str(a, "path"); p != "" {
			out, err = fsExec(ctx, env, `git diff -- "$rp"`, p, nil)
			if err == nil && out.ExitCode == exitEscape {
				return Result{}, ErrEscape
			}
		} else {
			out, err = exec(ctx, env, []string{"git", "diff"}, nil, nil)
		}
		return gitResult(t, "diff", out, err, max)
	}
	switch t.Kind {
	case KindScript:
		return r.runScript(ctx, env, t, a)
	case KindIntegration:
		if env.Connector == nil {
			return Result{}, fmt.Errorf("%w: conector ausente", ErrNotReady)
		}
		raw, err := env.Connector(ctx, t.Operation, a)
		if err != nil {
			return Result{}, err
		}
		return typed(t, raw, false)
	}
	return Result{}, ErrUnknownTool
}

func gitResult(t Tool, field string, out execution.ExecResult, err error, max int) (Result, error) {
	if err != nil || out.ExitCode != 0 {
		return Result{}, errors.New("o comando git falhou (o repositório ou o git podem não existir neste ambiente)")
	}
	s, trunc := capText(out.Stdout, max)
	return typed(t, map[string]any{field: s}, trunc)
}

// runScript roda a ferramenta de script: os parâmetros vão em PARAM_<nome> (nunca
// substituídos no texto do comando, o que evita injeção). A saída é um objeto
// JSON no stdout, conferido contra o esquema; sem esquema, volta exit_code e stdout.
func (r *Registry) runScript(ctx context.Context, env Env, t Tool, a map[string]any) (Result, error) {
	envs := map[string]string{}
	for k, v := range a {
		envs["PARAM_"+k] = fmt.Sprint(v)
	}
	out, err := exec(ctx, env, []string{"sh", "-c", t.Command}, nil, envs)
	if err != nil {
		if strings.Contains(err.Error(), "tempo limite") {
			return Result{}, err
		}
		return Result{}, errors.New("não foi possível executar a ferramenta")
	}
	stdout, trunc := capText(out.Stdout, t.Limits.MaxOutputBytes)
	if out.ExitCode != 0 {
		return Result{}, fmt.Errorf("a ferramenta terminou com código %d", out.ExitCode)
	}
	if len(t.Output) == 0 {
		return typed(t, map[string]any{"exit_code": 0, "stdout": stdout}, trunc, true)
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		return Result{}, errors.New("a saída da ferramenta não é um objeto JSON")
	}
	for k, v := range raw { // listas JSON viram []string
		if arr, ok := v.([]any); ok {
			ss := make([]string, 0, len(arr))
			for _, e := range arr {
				s, ok := e.(string)
				if !ok {
					return Result{}, fmt.Errorf("o campo %q da saída deve ser uma lista de textos", k)
				}
				ss = append(ss, s)
			}
			raw[k] = ss
		}
	}
	return typed(t, raw, trunc)
}

func typed(t Tool, raw map[string]any, truncated bool, loose ...bool) (Result, error) {
	if len(loose) > 0 && loose[0] { // sem esquema declarado: devolve como está
		return Result{Output: raw, Truncated: truncated}, nil
	}
	out, err := t.TypedOutput(raw)
	if err != nil {
		return Result{}, err
	}
	return Result{Output: out, Truncated: truncated}, nil
}
