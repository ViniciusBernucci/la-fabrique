package execution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

// Running é um comando em andamento dentro do sandbox, com a saída em fluxo e
// cancelamento que encerra a ÁRVORE de processos dentro do container (matar só o
// cliente do Docker deixaria os processos vivos).
type Running struct {
	// Stdout entrega a saída padrão à medida que chega.
	Stdout io.ReadCloser

	s       *Sandbox
	cmd     *exec.Cmd
	pidfile string
	stderr  *limitBuffer
	mu      sync.Mutex
	done    chan struct{}
	code    int
	err     error
	killed  bool
}

// Start inicia o comando dentro do sandbox. O PID do comando fica num arquivo
// para o cancelamento localizar a árvore.
func (s *Sandbox) Start(ctx context.Context, es ExecSpec) (*Running, error) {
	if len(es.Cmd) == 0 {
		return nil, ErrInvalidSpec
	}
	if err := s.usable(); err != nil {
		return nil, err
	}
	id := randomID()
	pidfile := "/tmp/run-" + id + ".pid"
	args := []string{"exec", "-i", "--user", sandboxUID + ":" + sandboxUID}
	if es.Dir != "" {
		args = append(args, "--workdir", es.Dir)
	}
	for k, v := range es.Env {
		if !envKey.MatchString(k) {
			return nil, ErrInvalidSpec
		}
		args = append(args, "--env", k+"="+v)
	}
	args = append(args, s.Container, "sh", "-c", `echo $$ > "$0"; exec "$@"`, pidfile)
	args = append(args, es.Cmd...)
	cmd := exec.Command(s.d.bin(), args...) // sem CommandContext: o cancelamento é nosso, para matar a árvore
	cmd.Stdin = es.Stdin
	r := &Running{s: s, cmd: cmd, pidfile: pidfile, stderr: &limitBuffer{}, done: make(chan struct{})}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = r.stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("exec: %w", err)
	}
	r.Stdout = out
	go func() {
		err := cmd.Wait()
		var ee *exec.ExitError
		r.mu.Lock()
		switch {
		case err == nil:
		case errors.As(err, &ee):
			r.code = ee.ExitCode()
		default:
			r.err = err
		}
		r.mu.Unlock()
		close(r.done)
	}()
	if ctx != nil {
		go func() {
			select {
			case <-ctx.Done():
				_ = r.Cancel(5 * time.Second)
			case <-r.done:
			}
		}()
	}
	return r, nil
}

// Wait espera o fim e devolve o código de saída. Chame depois de consumir Stdout.
func (r *Running) Wait() (int, error) {
	<-r.done
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.killed {
		return -1, r.err // cancelado: o código do cliente Docker não é confiável
	}
	return r.code, r.err
}

// Stderr devolve o que o comando escreveu em stderr (limitado).
func (r *Running) Stderr() string {
	<-r.done
	return r.stderr.String()
}

// killTree encerra o processo raiz e todos os descendentes (mais fundos primeiro),
// lendo a tabela de processos do container.
const killTree = `sig=$1; root=$(cat "$2" 2>/dev/null); [ -n "$root" ] || exit 0
kt() { for c in $(ps -o pid= -o ppid= | while read p pp; do [ "$pp" = "$1" ] && echo $p; done); do kt $c; done; kill -$sig $1 2>/dev/null; }
kt $root`

// Cancel envia SIGTERM à árvore de processos e, passado o prazo, SIGKILL.
// Confirma que o comando terminou antes de devolver.
func (r *Running) Cancel(grace time.Duration) error {
	select {
	case <-r.done:
		return nil
	default:
	}
	r.mu.Lock()
	r.killed = true
	r.mu.Unlock()
	signal := func(sig string) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _, _ = r.s.d.run(ctx, nil, "exec", "--user", sandboxUID+":"+sandboxUID, r.s.Container, "sh", "-c", killTree, "kt", sig, r.pidfile)
	}
	signal("TERM")
	select {
	case <-r.done:
	case <-time.After(grace):
		signal("KILL")
		select {
		case <-r.done:
		case <-time.After(10 * time.Second):
			_ = r.cmd.Process.Kill() // o cliente: o container será destruído pelo supervisor
			return errors.New("o comando não confirmou o encerramento")
		}
	}
	return nil
}

// ReadAll lê a saída inteira (atalho para testes e comandos curtos).
func (r *Running) ReadAll() (string, error) {
	var b bytes.Buffer
	_, err := io.Copy(&b, r.Stdout)
	return b.String(), err
}
