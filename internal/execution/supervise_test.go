package execution

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func start(t *testing.T, s *Sandbox, script string) *Running {
	t.Helper()
	r, err := s.Start(t.Context(), ExecSpec{Cmd: []string{"sh", "-c", script}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func fastPolicy(max, idle time.Duration) SupervisePolicy {
	return SupervisePolicy{MaxDuration: max, IdleTimeout: idle, Interval: 500 * time.Millisecond, Grace: time.Second}
}

func procsLeft(t *testing.T, s *Sandbox) string {
	t.Helper()
	return strings.TrimSpace(sh(t, s, "ps -o args= | grep -c -E '^(sleep 30|sh -c (end=|while))' || true").Stdout)
}

// T11-03 — "Processo legítimo silencioso continua."
func TestSilentButWorkingProcessIsNotKilled(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	// 5 s de trabalho de CPU sem escrever nada; a inatividade é de 2 s.
	r := start(t, s, `end=$(( $(date +%s) + 5 )); while [ $(date +%s) -lt $end ]; do :; done`)
	rep := s.Supervise(t.Context(), r, fastPolicy(60*time.Second, 2*time.Second))
	if rep.Outcome != OutcomeCompleted || rep.ExitCode != 0 || !rep.Confirmed || rep.Duration < 4*time.Second {
		t.Fatalf("o processo ativo e silencioso deveria seguir até o fim: %+v", rep)
	}
	if rep.OutputBytes != 0 || rep.CPUTicks == 0 {
		t.Fatalf("o sinal vinha da CPU, não da saída: %+v", rep)
	}
}

func TestOutputCountsAsActivity(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r := start(t, s, `i=0; while [ $i -lt 5 ]; do echo tick; sleep 1; i=$((i+1)); done`)
	rep := s.Supervise(t.Context(), r, fastPolicy(60*time.Second, 2500*time.Millisecond))
	if rep.Outcome != OutcomeCompleted || rep.OutputBytes < 25 {
		t.Fatalf("%+v", rep)
	}
}

func TestIdleProcessIsStoppedWithConfirmation(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r := start(t, s, `sleep 300 & sleep 301 & wait`)
	started := time.Now()
	rep := s.Supervise(t.Context(), r, fastPolicy(120*time.Second, 2*time.Second))
	if rep.Outcome != OutcomeIdle || !rep.Confirmed || rep.ProcessesAlive != 0 || rep.IdleFor < 2*time.Second {
		t.Fatalf("%+v", rep)
	}
	if time.Since(started) > 30*time.Second {
		t.Fatal("demorou demais")
	}
	eventually(t, 20*time.Second, "os filhos pararem", func() bool { return procsLeft(t, s) == "0" })
	if q, _ := s.Quarantined(); q {
		t.Fatal("parada confirmada não coloca o sandbox em quarentena")
	}
}

// T11-03 — "Timeout com filhos encerra a árvore."
func TestAbsoluteTimeoutStopsTheWholeTreeEvenWhenItKeepsTalking(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r := start(t, s, `sleep 300 & sleep 301 & while true; do echo vivo; sleep 1; done`)
	rep := s.Supervise(t.Context(), r, fastPolicy(3*time.Second, 0))
	if rep.Outcome != OutcomeTimeout || !rep.Confirmed || rep.ProcessesAlive != 0 || rep.OutputBytes == 0 {
		t.Fatalf("a saída contínua não impede o tempo absoluto: %+v", rep)
	}
	if rep.Duration < 3*time.Second || rep.Duration > 20*time.Second {
		t.Fatalf("duração: %v", rep.Duration)
	}
	eventually(t, 20*time.Second, "a árvore inteira terminar", func() bool { return procsLeft(t, s) == "0" })
	// Confirmado: o workspace pode ser reutilizado.
	if _, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"echo", "ok"}}); err != nil {
		t.Fatalf("reuso após parada confirmada: %v", err)
	}
}

func TestContextCancellationIsReported(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r := start(t, s, `while true; do echo x; sleep 1; done`)
	ctx, cancel := context.WithCancel(t.Context())
	go func() { time.Sleep(2 * time.Second); cancel() }()
	rep := s.Supervise(ctx, r, fastPolicy(time.Minute, 0))
	if rep.Outcome != OutcomeCancelled || !rep.Confirmed {
		t.Fatalf("%+v", rep)
	}
}

// T11-03 — "Estado incerto bloqueia reutilização do workspace."
func TestUncertainStopQuarantinesTheWorkspaceUntilVerified(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	r := start(t, s, `while true; do echo x; sleep 1; done`)
	// A amostragem "enxerga" um processo vivo depois do encerramento: não dá para confirmar.
	pol := fastPolicy(2*time.Second, 0)
	pol.Probe = func(context.Context) (Sample, error) { return Sample{CPUTicks: 1, IOBytes: 1, Procs: 1}, nil }
	rep := s.Supervise(t.Context(), r, pol)
	if rep.Outcome != OutcomeUncertain || rep.Confirmed || rep.ProcessesAlive != 1 || !strings.Contains(rep.Reason, "não foi possível confirmar") {
		t.Fatalf("%+v", rep)
	}
	q, why := s.Quarantined()
	if !q || why == "" {
		t.Fatal("o sandbox deve ficar em quarentena")
	}
	// Reuso bloqueado (comandos e processos em streaming); inspeção e destruição continuam.
	if _, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"echo", "x"}}); !errors.Is(err, ErrQuarantined) {
		t.Fatalf("Exec deveria ser bloqueado: %v", err)
	}
	if _, err := s.Start(t.Context(), ExecSpec{Cmd: []string{"echo", "x"}}); !errors.Is(err, ErrQuarantined) {
		t.Fatalf("Start deveria ser bloqueado: %v", err)
	}
	if _, err := s.Inspect(t.Context(), "{{.State.Running}}"); err != nil {
		t.Fatalf("inspecionar continua permitido: %v", err)
	}
	// A verificação real confirma que não há nada vivo e libera o sandbox.
	var last string
	eventually(t, 20*time.Second, "nada vivo no sandbox", func() bool {
		ok, err := s.VerifyQuiescent(t.Context())
		if !ok {
			out, _, _ := s.d.run(t.Context(), nil, "exec", s.Container, "ps", "-o", "pid,ppid,stat,args")
			last = out
		}
		return err == nil && ok
	})
	_ = last
	if q, _ := s.Quarantined(); q {
		t.Fatal("a quarentena deveria ter sido liberada")
	}
	if res, err := s.Exec(t.Context(), ExecSpec{Cmd: []string{"echo", "livre"}}); err != nil || strings.TrimSpace(res.Stdout) != "livre" {
		t.Fatalf("depois de verificado o sandbox volta a servir: %v %+v", err, res)
	}
}

func TestVerifyQuiescentSeesLeftoverProcesses(t *testing.T) {
	d := runtimeOrSkip(t)
	s := provision(t, d, "base", nil)
	sh(t, s, `(sleep 301 >/dev/null 2>&1 &)`)
	eventually(t, 20*time.Second, "o resto aparecer", func() bool { return procsLeft(t, s) == "1" })
	s.Quarantine("teste")
	if ok, err := s.VerifyQuiescent(t.Context()); err != nil || ok {
		t.Fatalf("com processo vivo não pode liberar: %v %v", ok, err)
	}
	if q, _ := s.Quarantined(); !q {
		t.Fatal("deve continuar em quarentena")
	}
}
