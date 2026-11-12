package providers

import (
	"bufio"
	"context"
	"errors"
	"io"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
)

// Process é uma execução em andamento.
type Process interface {
	Stdout() io.Reader
	Wait() (int, error)
	Stderr() string
	Cancel(grace time.Duration) error
}

// Runner inicia o comando (no sandbox, em produção).
type Runner interface {
	Start(ctx context.Context, cmd Command, dir string) (Process, error)
}

// SandboxRunner roda o comando dentro de um sandbox provisionado.
type SandboxRunner struct{ S *execution.Sandbox }

type sandboxProcess struct{ r *execution.Running }

func (p sandboxProcess) Stdout() io.Reader            { return p.r.Stdout }
func (p sandboxProcess) Wait() (int, error)           { return p.r.Wait() }
func (p sandboxProcess) Stderr() string               { return p.r.Stderr() }
func (p sandboxProcess) Cancel(g time.Duration) error { return p.r.Cancel(g) }

// Start implementa Runner.
func (sr SandboxRunner) Start(ctx context.Context, cmd Command, dir string) (Process, error) {
	r, err := sr.S.Start(ctx, execution.ExecSpec{Cmd: cmd.Args, Env: cmd.Env, Dir: dir, Stdin: stringsReader(cmd.Stdin)})
	if err != nil {
		return nil, err
	}
	return sandboxProcess{r}, nil
}

// Result resume uma execução.
type Result struct {
	ExitCode     int
	Failed       bool // saída diferente de zero, evento de erro ou cancelamento
	Cancelled    bool
	Final        string // texto do evento `done`, ou a última mensagem
	Failure      string
	InputTokens  int
	OutputTokens int
	Events       int
}

const maxLine = 1 << 20

// Run executa o pedido, entrega cada evento em onEvent e devolve o resumo.
// Cancelar o contexto encerra a árvore de processos. O erro devolvido é de
// infraestrutura; falha do próprio cliente vem em Result.Failed.
func Run(ctx context.Context, rn Runner, req Request, onEvent func(Event)) (Result, error) {
	cmd, err := BuildCommand(req)
	if err != nil {
		return Result{}, err
	}
	dir := req.Dir
	if dir == "" {
		dir = "/workspace"
	}
	proc, err := rn.Start(ctx, cmd, dir)
	if err != nil {
		return Result{}, err
	}
	var res Result
	sc := bufio.NewScanner(proc.Stdout())
	sc.Buffer(make([]byte, 64<<10), maxLine)
	var lastMessage string
	for sc.Scan() {
		for _, ev := range ParseLine(req.Account.CLI, sc.Text()) {
			res.Events++
			switch ev.Kind {
			case EvMessage:
				lastMessage = ev.Text
			case EvUsage:
				res.InputTokens += ev.InputTokens
				res.OutputTokens += ev.OutputTokens
			case EvDone:
				if ev.Text != "" {
					res.Final = ev.Text
				}
			case EvError:
				res.Failed = true
				res.Failure = ev.Text
			}
			if onEvent != nil {
				onEvent(ev)
			}
		}
	}
	if errors.Is(sc.Err(), bufio.ErrTooLong) {
		// Linha enorme: descarta o resto e segue (o processo não pode travar escrevendo).
		_, _ = io.Copy(io.Discard, proc.Stdout())
		res.Failed, res.Failure = true, "linha de saída acima do limite"
	}
	code, werr := proc.Wait()
	res.ExitCode = code
	if ctx.Err() != nil || code == -1 {
		res.Cancelled, res.Failed = true, true
		if res.Failure == "" {
			res.Failure = "execução cancelada"
		}
	} else if code != 0 {
		res.Failed = true
		if res.Failure == "" {
			res.Failure = "o cliente terminou com código de saída diferente de zero"
		}
	}
	if res.Final == "" {
		res.Final = lastMessage
	}
	if werr != nil {
		return res, werr
	}
	return res, nil
}
