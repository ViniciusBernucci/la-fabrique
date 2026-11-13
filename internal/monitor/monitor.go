// Package monitor torna observável a execução de agentes CLI e API na mesma interface (AJ07-02):
// grava os eventos normalizados do executor com sequência e cursor de reconexão (sem duplicar), mostra a
// saída enquanto o processo roda, o cronômetro, as ferramentas e os subagentes. O progresso só aparece
// quando o executor o informa: nada é estimado. Sob pressão (leitor lento) lacunas são explicitadas por
// um evento `agent.gap`, e todo texto passa pela sanitização antes de ser gravado.
package monitor

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// Prefixo dos tipos de evento do monitor em execution_events.
const TypePrefix = "agent."

// DB é o que o monitor precisa do PostgreSQL.
type DB interface {
	persistence.DBTX
	persistence.TxStarter
}

// Recorder grava eventos do executor.
type Recorder struct {
	DB       DB
	Redactor *security.Redactor
	Buffer   int // capacidade do canal (padrão 256): acima disso, lacuna explícita
}

// Sink devolve o `emit` para o executor e o `finish` que aguarda a gravação e devolve quantos eventos
// foram descartados por pressão. O `emit` nunca bloqueia o agente.
func (r *Recorder) Sink(ctx context.Context, projectID, executionID string) (emit func(executor.Event), finish func() (dropped int64)) {
	size := r.Buffer
	if size <= 0 {
		size = 256
	}
	ch := make(chan executor.Event, size)
	var dropped, total atomic.Int64
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for ev := range ch {
			// Lacuna pendente é registrada antes do próximo evento, na ordem real.
			if n := dropped.Swap(0); n > 0 {
				r.append(ctx, projectID, executionID, "gap", map[string]any{"dropped": n, "at": time.Now().UTC()})
			}
			r.append(ctx, projectID, executionID, ev.Kind, r.payload(ev))
		}
		if n := dropped.Swap(0); n > 0 {
			r.append(ctx, projectID, executionID, "gap", map[string]any{"dropped": n, "at": time.Now().UTC()})
		}
	}()
	emit = func(ev executor.Event) {
		select {
		case ch <- ev:
		default:
			dropped.Add(1)
			total.Add(1)
		}
	}
	finish = func() int64 {
		close(ch)
		wg.Wait()
		return total.Load()
	}
	return emit, finish
}

func (r *Recorder) payload(ev executor.Event) map[string]any {
	text := ev.Text
	if r.Redactor != nil {
		text = r.Redactor.Redact(text)
	}
	m := map[string]any{"source": ev.Source, "at": ev.At, "text": text}
	if ev.Tool != "" {
		m["tool"] = ev.Tool
	}
	if ev.Outcome != "" {
		m["outcome"] = ev.Outcome
	}
	if ev.Subagent != "" {
		m["subagent"] = ev.Subagent
	}
	if ev.InputTokens != 0 || ev.OutputTokens != 0 {
		m["input_tokens"], m["output_tokens"] = ev.InputTokens, ev.OutputTokens
	}
	if ev.Progress != nil {
		m["progress"] = *ev.Progress
	}
	return m
}

func (r *Recorder) append(ctx context.Context, projectID, executionID, kind string, payload any) {
	_ = persistence.WithTx(ctx, r.DB, func(tx pgx.Tx) error {
		_, err := persistence.AppendEvent(ctx, tx, projectID, executionID, TypePrefix+kind, payload)
		return err
	})
}
