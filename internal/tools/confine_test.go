package tools

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

type actorRec struct {
	mu     sync.Mutex
	actors []string
}

func (a *actorRec) Record(_ context.Context, act policy.Action, _ policy.Decision) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	s, _ := act.Params["actor"].(string)
	a.actors = append(a.actors, s)
	return nil
}

// sh prepara o cenário DIRETO no sandbox (como faria um repositório hostil ou um
// CLI de IA dentro do container), sem passar pela política das ferramentas.
func sh(t *testing.T, env Env, cmd string) {
	t.Helper()
	res, err := env.Sandbox.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", cmd}})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("preparo %q: %+v %v", cmd, res, err)
	}
}

// T11-02 — "Path traversal, symlinks e tentativa de acesso ao socket falham."
func TestSymlinksCannotLeaveTheWorkspaceInAnyFileTool(t *testing.T) {
	sb := dockerSandbox(t)
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: &countRec{}}}
	r := NewRegistry(nil)
	call(t, r, env, "write_file", map[string]any{"path": "dentro/ok.txt", "content": "conteudo interno\n"})
	// O que o agente (ou um repositório hostil) pode deixar no workspace.
	sh(t, env, `cd /workspace &&
		ln -s /etc/passwd link_arquivo &&
		ln -s / link_raiz &&
		ln -s /etc link_dir &&
		ln -s /nao/existe/fora link_pendente &&
		ln -s /var/run/docker.sock link_socket &&
		ln -s link_arquivo link_cadeia &&
		ln -s dentro link_interno &&
		ln -s dentro/ok.txt link_arquivo_interno &&
		ln -s ../../etc link_relativo`)

	escapes := []struct {
		name string
		tool string
		args map[string]any
	}{
		{"ler link para arquivo fora", "read_file", map[string]any{"path": "link_arquivo"}},
		{"ler pela raiz", "read_file", map[string]any{"path": "link_raiz/etc/passwd"}},
		{"ler por diretório linkado", "read_file", map[string]any{"path": "link_dir/passwd"}},
		{"ler link em cadeia", "read_file", map[string]any{"path": "link_cadeia"}},
		{"ler link relativo", "read_file", map[string]any{"path": "link_relativo/passwd"}},
		{"ler socket do runtime", "read_file", map[string]any{"path": "link_socket"}},
		{"ler link pendente", "read_file", map[string]any{"path": "link_pendente"}},
		{"gravar por link de arquivo", "write_file", map[string]any{"path": "link_arquivo", "content": "x"}},
		{"gravar por diretório linkado", "write_file", map[string]any{"path": "link_dir/novo.txt", "content": "x"}},
		{"gravar criando diretórios além do link", "write_file", map[string]any{"path": "link_dir/a/b/c.txt", "content": "x"}},
		{"gravar por link pendente", "write_file", map[string]any{"path": "link_pendente", "content": "x"}},
		{"editar por link", "edit_file", map[string]any{"path": "link_arquivo", "old": "root", "new": "x"}},
		{"listar a raiz pelo link", "list_dir", map[string]any{"path": "link_raiz"}},
		{"listar /etc pelo link", "list_dir", map[string]any{"path": "link_dir"}},
		{"buscar pelo link", "search", map[string]any{"pattern": "root", "path": "link_dir"}},
		{"git diff pelo link", "git_diff", map[string]any{"path": "link_arquivo"}},
	}
	for _, e := range escapes {
		if _, err := call(t, r, env, e.tool, e.args); !errors.Is(err, ErrEscape) {
			t.Errorf("%s: esperado ErrEscape, veio %v", e.name, err)
		}
	}
	// Nada foi criado fora do workspace.
	sh(t, env, `test ! -e /etc/novo.txt && test ! -e /etc/a && ! grep -q "^x$" /etc/passwd`)

	// Links que ficam DENTRO do workspace continuam funcionando.
	if rd, err := call(t, r, env, "read_file", map[string]any{"path": "link_arquivo_interno"}); err != nil || rd.Output["content"] != "conteudo interno\n" {
		t.Fatalf("link interno de arquivo: %+v %v", rd, err)
	}
	if ls, err := call(t, r, env, "list_dir", map[string]any{"path": "link_interno"}); err != nil || len(ls.Output["entries"].([]string)) != 1 {
		t.Fatalf("link interno de diretório: %+v %v", ls, err)
	}
	if _, err := call(t, r, env, "write_file", map[string]any{"path": "link_interno/novo.txt", "content": "novo"}); err != nil {
		t.Fatalf("gravar por link interno: %v", err)
	}
	if rd, _ := call(t, r, env, "read_file", map[string]any{"path": "dentro/novo.txt"}); rd.Output["content"] != "novo" {
		t.Fatalf("a gravação por link interno cai no destino: %+v", rd)
	}
	if _, err := call(t, r, env, "edit_file", map[string]any{"path": "link_arquivo_interno", "old": "interno", "new": "INTERNO"}); err != nil {
		t.Fatal(err)
	}
	if rd, _ := call(t, r, env, "read_file", map[string]any{"path": "dentro/ok.txt"}); rd.Output["content"] != "conteudo INTERNO\n" {
		t.Fatalf("%+v", rd)
	}
	// Caminhos novos em diretórios novos (mkdir -p) seguem válidos.
	if _, err := call(t, r, env, "write_file", map[string]any{"path": "novo/dir/fundo.txt", "content": "ok"}); err != nil {
		t.Fatalf("criar diretórios novos: %v", err)
	}
}

func TestSearchDoesNotFollowLinksInsideTheTree(t *testing.T) {
	sb := dockerSandbox(t)
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: &countRec{}}}
	r := NewRegistry(nil)
	call(t, r, env, "write_file", map[string]any{"path": "a.txt", "content": "palavra-unica-xyz\n"})
	sh(t, env, `cd /workspace && echo "root:x:0:0" > /tmp/fora.txt && ln -s /tmp tmp_link && ln -s /etc etc_link`)
	res, err := call(t, r, env, "search", map[string]any{"pattern": "root"})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range res.Output["matches"].([]string) {
		if !strings.HasPrefix(m, "/workspace/") || strings.Contains(m, "tmp_link") || strings.Contains(m, "etc_link") {
			t.Fatalf("a busca seguiu um link para fora: %v", res.Output["matches"])
		}
	}
	if own, err := call(t, r, env, "search", map[string]any{"pattern": "palavra-unica-xyz"}); err != nil || len(own.Output["matches"].([]string)) != 1 {
		t.Fatalf("a busca normal continua achando: %+v %v", own, err)
	}
}

func TestTraversalAndRuntimeSocketAreUnreachable(t *testing.T) {
	sb := dockerSandbox(t)
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: &countRec{}}}
	r := NewRegistry(nil)
	for _, p := range []string{"../etc/passwd", "/etc/passwd", "/workspace/../etc/passwd", "/var/run/docker.sock", "/run/docker.sock", "/proc/self/environ", "dentro/../../etc"} {
		for _, tool := range []string{"read_file", "list_dir"} {
			if _, err := call(t, r, env, tool, map[string]any{"path": p}); err == nil {
				t.Errorf("%s %q deveria falhar", tool, p)
			}
		}
		if _, err := call(t, r, env, "write_file", map[string]any{"path": p, "content": "x"}); err == nil {
			t.Errorf("write_file %q deveria falhar", p)
		}
	}
	// Por comando: a política recusa até mencionar o runtime de containers.
	for _, cmd := range []string{"docker ps", "ls -l /var/run/docker.sock", "curl --unix-socket /var/run/docker.sock http://x/containers/json"} {
		if _, err := call(t, r, env, "run_command", map[string]any{"command": cmd}); !errors.Is(err, policy.ErrForbidden) {
			t.Errorf("%q deveria ser proibido pela política: %v", cmd, err)
		}
	}
	// E, mesmo sem a política (CLI de IA dentro do container), o ambiente não o oferece.
	sh(t, env, `test ! -e /var/run/docker.sock && test ! -e /run/docker.sock && ! command -v docker >/dev/null`)
	// Fora do workspace só se escreve no tmpfs /tmp; o rootfs é somente leitura.
	res, err := env.Sandbox.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", "echo x > /etc/escrita"}})
	if err != nil || res.ExitCode == 0 {
		t.Fatalf("rootfs deve ser somente leitura: %+v %v", res, err)
	}
}

// T11-02 — "chamadas brokeradas com identidade e política por operação".
func TestBrokeredCallsRequireIdentityAndRecordTheActor(t *testing.T) {
	rec := &actorRec{}
	var gotID Identity
	var gotOK bool
	env := Env{Guard: &policy.Guard{Policy: pol(), Rec: rec}, Connector: func(ctx context.Context, op string, args map[string]any) (map[string]any, error) {
		gotID, gotOK = IdentityFrom(ctx)
		return map[string]any{"key": "ABC-1"}, nil
	}}
	tool := Tool{Name: "criar_sub", Kind: KindIntegration, Operation: "create_subtask", Description: "Cria subtarefa.", Permissions: []string{PermAPI}, Limits: Limits{20, 1024},
		Input:  []Field{{Name: "titulo", Type: TypeString, Required: true, Max: 200}},
		Output: []Field{{Name: "key", Type: TypeString, Required: true}}}
	r := NewRegistry(func(_ context.Context, name string) (Tool, error) { return tool, nil })
	args := map[string]any{"titulo": "Nova"}

	// Sem identidade (ou incompleta): recusada antes da política e do conector.
	for _, id := range []Identity{{}, {Agent: "ag"}, {Run: "r"}} {
		ctx := t.Context()
		if id != (Identity{}) {
			ctx = WithIdentity(ctx, id)
		}
		if _, err := r.Call(ctx, env, "criar_sub", args, grantAll("criar_sub")); !errors.Is(err, ErrIdentity) {
			t.Fatalf("%+v: %v", id, err)
		}
	}
	if gotOK || len(rec.actors) != 0 {
		t.Fatalf("nada pode ter chegado ao conector nem à trilha: %v %v", gotOK, rec.actors)
	}
	ctx := WithIdentity(t.Context(), Identity{Agent: "agente-7", Run: "exec-9"})
	if _, err := r.Call(ctx, env, "criar_sub", args, grantAll("criar_sub")); err != nil {
		t.Fatal(err)
	}
	if !gotOK || gotID != (Identity{Agent: "agente-7", Run: "exec-9"}) {
		t.Fatalf("o conector recebe a identidade: %v %+v", gotOK, gotID)
	}
	if len(rec.actors) != 1 || rec.actors[0] != "agent:agente-7 run:exec-9" {
		t.Fatalf("a trilha da política registra o ator: %v", rec.actors)
	}
	// Política por operação: a mesma ferramenta, com outra operação, é recusada.
	tool.Operation = "rename_project"
	gotOK = false
	if _, err := r.Call(ctx, env, "criar_sub", args, grantAll("criar_sub")); !errors.Is(err, policy.ErrForbidden) || gotOK {
		t.Fatalf("%v", err)
	}
}

// Ferramentas locais também registram o ator quando há identidade.
func TestLocalToolsRecordTheActorWhenIdentified(t *testing.T) {
	sb := dockerSandbox(t)
	rec := &actorRec{}
	env := Env{Sandbox: sb, Guard: &policy.Guard{Policy: pol(), Rec: rec}}
	r := NewRegistry(nil)
	ctx := WithIdentity(t.Context(), Identity{Agent: "a1", Run: "r1"})
	if _, err := r.Call(ctx, env, "list_dir", map[string]any{"path": "."}, grantAll("list_dir")); err != nil {
		t.Fatal(err)
	}
	if len(rec.actors) != 1 || rec.actors[0] != "agent:a1 run:r1" {
		t.Fatalf("%v", rec.actors)
	}
}
