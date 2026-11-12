package providers

import (
	"archive/tar"
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
)

// Estes testes usam Docker de verdade com um "CLI" de mentira (um script que
// imita a saída do Claude Code). O CLI real NÃO é executado: exigiria a conta de
// um responsável.

const fakeClaude = `#!/bin/sh
d="$CLAUDE_CONFIG_DIR"
printf '%s\n' "$@" > "$d/argv"
cat > "$d/stdin"
tok=$(cat "$d/token" 2>/dev/null)
echo '{"type":"system","subtype":"init"}'
echo "{\"type\":\"assistant\",\"message\":{\"content\":[{\"type\":\"text\",\"text\":\"conta=$tok\"},{\"type\":\"tool_use\",\"name\":\"Edit\"}]}}"
echo '{"type":"result","is_error":false,"result":"pronto","usage":{"input_tokens":10,"output_tokens":5}}'
`

const fakeSlow = `#!/bin/sh
sleep 300 &
sleep 301 &
echo '{"type":"system"}'
wait
`

func script(files map[string]string) *bytes.Reader {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for name, body := range files {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))})
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	return bytes.NewReader(buf.Bytes())
}

func dockerOrSkip(t *testing.T) *execution.Docker {
	t.Helper()
	d := &execution.Docker{}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if !d.Available(ctx) {
		t.Skip("Docker indisponível")
	}
	return d
}

// sandboxFor provisiona um sandbox que monta só o volume da conta (já com o "login").
func sandboxFor(t *testing.T, d *execution.Docker, acct Account, token string) *execution.Sandbox {
	t.Helper()
	if err := d.EnsureVolume(t.Context(), acct.VolumeName()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.RemoveVolume(context.Background(), acct.VolumeName()) })
	s, err := d.Provision(t.Context(), execution.Spec{
		ProjectID: "p", FeatureID: "f", Profile: execution.Profiles["base"],
		Mounts: []execution.Mount{{Volume: acct.VolumeName(), Target: AccountMount}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Destroy(context.Background()) })
	if err := s.Seed(t.Context(), script(map[string]string{"fake-claude.sh": fakeClaude, "fake-slow.sh": fakeSlow})); err != nil {
		t.Fatal(err)
	}
	if token != "" { // simula o login oficial feito pelo responsável nesta conta
		r, err := s.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", "echo " + token + " > " + AccountMount + "/token"}})
		if err != nil || r.ExitCode != 0 {
			t.Fatalf("login simulado: %+v %v", r, err)
		}
	}
	return s
}

func read(t *testing.T, s *execution.Sandbox, path string) string {
	t.Helper()
	r, err := s.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"cat", path}})
	if err != nil {
		t.Fatal(err)
	}
	return r.Stdout
}

func TestTwoAccountsOfTheSameProviderAreDistinct(t *testing.T) {
	d := dockerOrSkip(t)
	a := Account{ID: acctA, Name: "conta A", CLI: Claude}
	b := Account{ID: acctB, Name: "conta B", CLI: Claude}
	if a.VolumeName() == b.VolumeName() {
		t.Fatal("cada conta tem o seu volume")
	}
	sa, sb := sandboxFor(t, d, a, "token-da-A"), sandboxFor(t, d, b, "token-da-B")

	for _, c := range []struct {
		s    *execution.Sandbox
		acct Account
		want string
	}{{sa, a, "conta=token-da-A"}, {sb, b, "conta=token-da-B"}} {
		r := Request{Account: c.acct, Prompt: "Implemente a feature X", Mode: ModeReadOnly, Binary: "/workspace/fake-claude.sh"}
		var msgs []string
		res, err := Run(t.Context(), SandboxRunner{S: c.s}, r, func(e Event) {
			if e.Kind == EvMessage {
				msgs = append(msgs, e.Text)
			}
		})
		if err != nil || res.Failed || res.Final != "pronto" || res.InputTokens != 10 || len(msgs) != 1 || msgs[0] != c.want {
			t.Fatalf("%s: %+v %v %v", c.acct.Name, res, err, msgs)
		}
		// O modo autorizado chegou ao cliente e o prompt chegou por stdin.
		argv := read(t, c.s, AccountMount+"/argv")
		if !strings.Contains(argv, "--permission-mode\ndontAsk") || !strings.Contains(argv, "--allowedTools\nRead,Grep,Glob") || strings.Contains(argv, "bypass") {
			t.Fatalf("argv recebido pelo cliente: %q", argv)
		}
		if strings.Contains(argv, "feature X") || strings.TrimSpace(read(t, c.s, AccountMount+"/stdin")) != "Implemente a feature X" {
			t.Fatalf("o prompt deve chegar por stdin: argv=%q", argv)
		}
	}
	// Uma conta não enxerga o login da outra.
	if got := strings.TrimSpace(read(t, sa, AccountMount+"/token")); got != "token-da-A" {
		t.Fatalf("A: %q", got)
	}
	if got := strings.TrimSpace(read(t, sb, AccountMount+"/token")); got != "token-da-B" {
		t.Fatalf("B: %q", got)
	}
}

func TestCancellingARunKillsTheClientAndItsChildren(t *testing.T) {
	d := dockerOrSkip(t)
	a := Account{ID: acctA, Name: "conta A", CLI: Claude}
	s := sandboxFor(t, d, a, "token-da-A")
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan Result, 1)
	go func() {
		res, _ := Run(ctx, SandboxRunner{S: s}, Request{Account: a, Prompt: "x", Mode: ModeReadOnly, Binary: "/workspace/fake-slow.sh"}, nil)
		done <- res
	}()
	count := func() string {
		r, _ := s.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", "ps -o args= | grep -c '^sleep 30' || true"}})
		return strings.TrimSpace(r.Stdout)
	}
	deadline := time.Now().Add(20 * time.Second)
	for count() != "2" && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	if count() != "2" {
		t.Fatal("os filhos do cliente deveriam estar rodando antes do cancelamento")
	}
	cancel()
	select {
	case res := <-done:
		if !res.Cancelled || !res.Failed {
			t.Fatalf("%+v", res)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("o cancelamento não encerrou a execução")
	}
	deadline = time.Now().Add(20 * time.Second)
	for count() != "0" && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	if got := count(); got != "0" {
		t.Fatalf("os filhos do cliente deveriam ter terminado: %q", got)
	}
}

// "CLI" de mentira que faz o que um agente faz dentro do container: operações
// autorizadas no workspace e tentativas de fugir dele.
const fakeAgent = `#!/bin/sh
cat > /dev/null
out=""
say() { out="$out$1;"; }
mkdir -p /workspace/feature && echo "linha" > /workspace/feature/a.txt && grep -q linha /workspace/feature/a.txt && say "workspace=ok" || say "workspace=falhou"
( cd /workspace/feature && echo 'teste ok' > t.log && cat t.log >/dev/null ) && say "teste=ok" || say "teste=falhou"
echo x > /etc/escapou 2>/dev/null && say "rootfs=ESCAPOU" || say "rootfs=bloqueado"
echo x > /workspace/../etc/escapou2 2>/dev/null && say "traversal=ESCAPOU" || say "traversal=bloqueado"
[ -e /var/run/docker.sock ] && say "socket=EXISTE" || say "socket=ausente"
command -v docker >/dev/null && say "docker=EXISTE" || say "docker=ausente"
[ "$(id -u)" = "0" ] && say "usuario=ROOT" || say "usuario=sem-privilegio"
echo '{"type":"system","subtype":"init"}'
echo "{\"type\":\"result\",\"is_error\":false,\"result\":\"$out\",\"usage\":{\"input_tokens\":1,\"output_tokens\":1}}"
`

// T11-02 — modo CLI: o agente trabalha no workspace e não consegue sair do container.
func TestCLIModeWorksInTheWorkspaceAndCannotEscapeTheContainer(t *testing.T) {
	d := dockerOrSkip(t)
	a := Account{ID: acctA, Name: "conta A", CLI: Claude}
	s := sandboxFor(t, d, a, "")
	if err := s.Seed(t.Context(), script(map[string]string{"fake-agent.sh": fakeAgent})); err != nil {
		t.Fatal(err)
	}
	res, err := Run(t.Context(), SandboxRunner{S: s}, Request{Account: a, Prompt: "implemente", Mode: ModeWorkspaceWrite, Binary: "/workspace/fake-agent.sh"}, func(Event) {})
	if err != nil || res.Failed {
		t.Fatalf("%+v %v", res, err)
	}
	want := "workspace=ok;teste=ok;rootfs=bloqueado;traversal=bloqueado;socket=ausente;docker=ausente;usuario=sem-privilegio;"
	if res.Final != want {
		t.Fatalf("\n got: %s\nwant: %s", res.Final, want)
	}
}
