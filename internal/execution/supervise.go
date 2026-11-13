package execution

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Resultados da supervisão.
const (
	OutcomeCompleted = "completed" // terminou por conta própria
	OutcomeTimeout   = "timeout"   // estourou o tempo absoluto
	OutcomeIdle      = "idle"      // sem nenhum sinal de atividade pelo prazo
	OutcomeCancelled = "cancelled" // o contexto foi cancelado
	OutcomeUncertain = "uncertain" // não foi possível confirmar que tudo parou
)

// Sample é uma amostra da árvore de processos do comando.
type Sample struct {
	CPUTicks uint64 // tempo de CPU acumulado (usuário + sistema) de toda a árvore
	IOBytes  uint64 // bytes lidos + escritos acumulados de toda a árvore
	Procs    int    // processos vivos na árvore
}

// SupervisePolicy define os prazos. MaxDuration e IdleTimeout são independentes.
type SupervisePolicy struct {
	MaxDuration time.Duration // 0 = sem limite absoluto
	IdleTimeout time.Duration // 0 = sem limite de inatividade
	Interval    time.Duration // intervalo entre amostras (padrão 1 s)
	Grace       time.Duration // espera entre SIGTERM e SIGKILL (padrão 3 s)
	// Probe substitui a amostragem real (testes).
	Probe func(ctx context.Context) (Sample, error)
}

// Report é o resultado auditável da supervisão.
type Report struct {
	Outcome        string        `json:"outcome"`
	Reason         string        `json:"reason"`
	ExitCode       int           `json:"exit_code"`
	StartedAt      time.Time     `json:"started_at"`
	EndedAt        time.Time     `json:"ended_at"`
	Duration       time.Duration `json:"duration"`
	IdleFor        time.Duration `json:"idle_for"`        // inatividade no momento da decisão
	OutputBytes    int64         `json:"output_bytes"`    // bytes vistos na saída
	CPUTicks       uint64        `json:"cpu_ticks"`       // maior valor observado (contador acumulado da árvore)
	IOBytes        uint64        `json:"io_bytes"`        // maior valor observado
	Confirmed      bool          `json:"confirmed"`       // a árvore de processos terminou, confirmado por amostra
	ProcessesAlive int           `json:"processes_alive"` // vivos na última amostra após o encerramento
}

type countingReader struct {
	r    io.ReadCloser
	n    atomic.Int64
	last atomic.Int64 // unix nano da última leitura com dados
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	if n > 0 {
		c.n.Add(int64(n))
		c.last.Store(time.Now().UnixNano())
	}
	return n, err
}
func (c *countingReader) Close() error { return c.r.Close() }

const probeScript = `root=$(cat "$1" 2>/dev/null); [ -n "$root" ] || { echo "0 0 0"; exit 0; }
pids="$root"; frontier="$root"
while [ -n "$frontier" ]; do
  next=""
  for p in $frontier; do
    for c in $(ps -o pid= -o ppid= | while read a b; do [ "$b" = "$p" ] && echo $a; done); do next="$next $c"; pids="$pids $c"; done
  done
  frontier="$next"
done
cpu=0; io=0; n=0
for p in $pids; do
  [ -r /proc/$p/stat ] || continue
  n=$((n+1))
  set -- $(sed 's/.*) //' /proc/$p/stat 2>/dev/null)
  cpu=$((cpu + ${12:-0} + ${13:-0}))
  b=$(awk '/^(rchar|wchar):/{s+=$2} END{print s+0}' /proc/$p/io 2>/dev/null)
  io=$((io + ${b:-0}))
done
echo "$cpu $io $n"`

// probe amostra a árvore de processos do comando dentro do container.
func (r *Running) probe(ctx context.Context) (Sample, error) {
	out, _, err := r.s.d.run(ctx, nil, "exec", "--user", sandboxUID+":"+sandboxUID, r.s.Container, "sh", "-c", probeScript, "probe", r.pidfile)
	if err != nil {
		return Sample{}, fmt.Errorf("amostragem falhou: %w", err)
	}
	f := strings.Fields(out)
	if len(f) != 3 {
		return Sample{}, fmt.Errorf("amostragem com formato inesperado")
	}
	cpu, e1 := strconv.ParseUint(f[0], 10, 64)
	io, e2 := strconv.ParseUint(f[1], 10, 64)
	n, e3 := strconv.Atoi(f[2])
	if e1 != nil || e2 != nil || e3 != nil {
		return Sample{}, fmt.Errorf("amostragem com números inválidos")
	}
	return Sample{CPUTicks: cpu, IOBytes: io, Procs: n}, nil
}

// Supervise acompanha o comando até o fim, o tempo absoluto, a inatividade ou o
// cancelamento do contexto. Atividade é qualquer um destes sinais: saída nova,
// aumento de CPU ou de I/O da árvore, ou mudança no número de processos. Um
// comando legítimo que fica em silêncio mas segue trabalhando continua; um que
// não dá nenhum sinal pelo prazo é encerrado.
//
// Ao encerrar, o grupo recebe SIGTERM e depois SIGKILL e a parada é CONFIRMADA
// por amostra (nenhum processo vivo). Sem confirmação o resultado é `uncertain`
// e o sandbox entra em quarentena: não pode ser reutilizado até VerifyQuiescent.
// Consome r.Stdout até o fim.
func (s *Sandbox) Supervise(ctx context.Context, r *Running, p SupervisePolicy) Report {
	if p.Interval <= 0 {
		p.Interval = time.Second
	}
	if p.Grace <= 0 {
		p.Grace = 3 * time.Second
	}
	probe := p.Probe
	if probe == nil {
		probe = r.probe
	}
	started := time.Now()
	rep := Report{StartedAt: started}
	cr := &countingReader{r: r.Stdout}
	cr.last.Store(started.UnixNano())
	r.Stdout = cr
	drained := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, cr); close(drained) }()

	finished := make(chan struct{})
	go func() { _, _ = r.Wait(); close(finished) }()

	lastActivity := started
	var prev Sample
	var prevOut int64
	tick := time.NewTicker(p.Interval)
	defer tick.Stop()

	stop := func(outcome, reason string) {
		rep.IdleFor = time.Since(lastActivity)
		rep.Outcome, rep.Reason = outcome, reason
		cerr := r.Cancel(p.Grace)
		// Confirma a parada: nenhum processo vivo na árvore.
		pctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		smp, perr := probe(pctx)
		rep.ProcessesAlive = smp.Procs
		if cerr == nil && perr == nil && smp.Procs == 0 {
			rep.Confirmed = true
		} else {
			rep.Outcome = OutcomeUncertain
			rep.Reason = reason + "; não foi possível confirmar o encerramento da árvore de processos"
			s.Quarantine(rep.Reason)
		}
	}

loop:
	for {
		select {
		case <-finished:
			rep.Outcome, rep.Reason, rep.Confirmed = OutcomeCompleted, "o comando terminou", true
			break loop
		case <-ctx.Done():
			stop(OutcomeCancelled, "execução cancelada")
			break loop
		case <-tick.C:
			now := time.Now()
			if p.MaxDuration > 0 && now.Sub(started) >= p.MaxDuration {
				stop(OutcomeTimeout, fmt.Sprintf("tempo máximo de %s excedido", p.MaxDuration))
				break loop
			}
			cur := Sample{}
			if smp, err := probe(ctx); err == nil {
				cur = smp
				rep.CPUTicks, rep.IOBytes = max(rep.CPUTicks, smp.CPUTicks), max(rep.IOBytes, smp.IOBytes)
			}
			out := cr.n.Load()
			if out > prevOut || cur.CPUTicks > prev.CPUTicks || cur.IOBytes > prev.IOBytes || (cur.Procs != prev.Procs && cur.Procs > 0) {
				lastActivity = now
			}
			prev, prevOut = cur, out
			if p.IdleTimeout > 0 && now.Sub(lastActivity) >= p.IdleTimeout {
				stop(OutcomeIdle, fmt.Sprintf("sem atividade por %s", p.IdleTimeout))
				break loop
			}
		}
	}
	<-finished
	<-drained
	code, _ := r.Wait()
	rep.ExitCode = code
	rep.OutputBytes = cr.n.Load()
	rep.EndedAt = time.Now()
	rep.Duration = rep.EndedAt.Sub(started)
	return rep
}

// quiescentScript conta os processos que não são o init, o sleep de manutenção do
// sandbox nem o próprio comando de verificação.
const quiescentScript = `ps -o pid=,args= | awk '{pid=$1; $1=""; sub(/^ /,"",$0); a=$0} pid==1{next} a=="sleep 2147483647"{next} index(a,"lf-quiescence-check")>0{next} index(a,"ps -o pid=")==1{next} index(a,"awk ")==1{next} {n++} END{print n+0}'`

// VerifyQuiescent confirma que não há processo do projeto vivo no sandbox e, se
// confirmar, libera a quarentena.
func (s *Sandbox) VerifyQuiescent(ctx context.Context) (bool, error) {
	out, _, err := s.d.run(ctx, nil, "exec", "--user", sandboxUID+":"+sandboxUID, s.Container, "sh", "-c", quiescentScript, "lf-quiescence-check")
	if err != nil {
		return false, fmt.Errorf("verificação falhou: %w", err)
	}
	n, perr := strconv.Atoi(strings.TrimSpace(out))
	if perr != nil {
		return false, fmt.Errorf("verificação com formato inesperado")
	}
	if n == 0 {
		s.Quarantine("")
		return true, nil
	}
	return false, nil
}
