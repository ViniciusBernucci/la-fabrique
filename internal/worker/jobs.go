package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/opslog"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/recovery"
	"github.com/ViniciusBernucci/le-fabrique/internal/retention"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// opslogSource mapeia o nome do job (T16-05) para a origem do evento operacional. Nome não
// listado aqui cai no padrão "sentinela" (ver 00-plano-T16-05, seção "Regras de negócio").
func opslogSource(jobName string) string {
	switch jobName {
	case "retencao", "retencao-simulada":
		return opslog.SourceRetencao
	case "login-contas-ia":
		return opslog.SourceCLILogin
	case "fabrica":
		return opslog.SourceFabrica
	case "sentinela-eventos": // reavaliação da admissão por evento (cmd/worker/admission.go)
		return opslog.SourceAdmissao
	default: // "sentinela-rotinas", "liberar-decisoes", "recuperar-tentativas", etc.
		return opslog.SourceSentinela
	}
}

// Job é uma tarefa periódica do worker. Run devolve um resumo curto do que fez ("" = nada) ou erro.
type Job struct {
	Name  string
	Every time.Duration
	Run   func(ctx context.Context) (string, error)
}

// JobStatus é o que o worker mostra de cada tarefa periódica.
type JobStatus struct {
	Name      string    `json:"name"`
	Every     string    `json:"every"`
	Runs      int64     `json:"runs"`
	LastRun   time.Time `json:"last_run,omitempty"`
	LastError string    `json:"last_error,omitempty"`
	LastNote  string    `json:"last_note,omitempty"`
}

type jobState struct {
	Job
	mu     sync.Mutex
	status JobStatus
}

// AddJob registra uma tarefa periódica (antes de Run).
func (w *Worker) AddJob(j Job) {
	if j.Name == "" || j.Every <= 0 || j.Run == nil {
		return
	}
	w.jobs = append(w.jobs, &jobState{Job: j, status: JobStatus{Name: j.Name, Every: j.Every.String()}})
}

// Jobs devolve a situação das tarefas.
func (w *Worker) Jobs() []JobStatus {
	out := make([]JobStatus, 0, len(w.jobs))
	for _, j := range w.jobs {
		j.mu.Lock()
		out = append(out, j.status)
		j.mu.Unlock()
	}
	return out
}

// RunJobOnce executa a tarefa agora (para testes e para a primeira rodada).
func (w *Worker) RunJobOnce(ctx context.Context, name string) (string, error) {
	for _, j := range w.jobs {
		if j.Name == name {
			return w.exec(ctx, j), nil
		}
	}
	return "", fmt.Errorf("tarefa %q não registrada", name)
}

func (w *Worker) exec(ctx context.Context, j *jobState) (err string) {
	note, runErr := j.Run(ctx)
	j.mu.Lock()
	defer j.mu.Unlock()
	j.status.Runs++
	j.status.LastRun = time.Now().UTC()
	j.status.LastNote = note
	j.status.LastError = ""
	if runErr != nil {
		j.status.LastError = runErr.Error()
		w.log.Warn("tarefa do worker falhou", "job", j.Name, "error", runErr.Error())
		w.emitOps(ctx, opslog.LevelError, opslogSource(j.Name), runErr.Error(), map[string]any{"job": j.Name})
		return runErr.Error()
	}
	if note != "" {
		w.log.Info("tarefa do worker", "job", j.Name, "resumo", note)
		w.emitOps(ctx, opslog.LevelInfo, opslogSource(j.Name), note, map[string]any{"job": j.Name})
	}
	return ""
}

// emitOps grava o evento operacional (T16-05). Sem conexão disponível (ex.: testes com um Pinger
// simples) ou falha na gravação, só avisa no log: nunca afeta o resultado do job que chamou.
func (w *Worker) emitOps(ctx context.Context, level, source, message string, payload map[string]any) {
	if w.ops == nil {
		return
	}
	if err := opslog.Emit(ctx, w.ops, level, source, message, payload); err != nil {
		w.log.Warn("evento operacional não gravado", "job", payload["job"], "error", err.Error())
	}
}

func (w *Worker) runJobs(ctx context.Context) {
	for _, j := range w.jobs {
		go func(j *jobState) {
			t := time.NewTicker(j.Every)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					w.exec(ctx, j)
				}
			}
		}(j)
	}
}

// ReleaseDecisionsJob devolve à fila os itens que esperavam uma decisão já resolvida. Quem libera é
// a porta de decisão: item com decisão pendente continua esperando.
func ReleaseDecisionsJob(db persistence.DBTX, q *queue.Queue, every time.Duration) Job {
	return Job{Name: "liberar-decisoes", Every: every, Run: func(ctx context.Context) (string, error) {
		rows, err := db.Query(ctx, `SELECT id::text FROM queue_items WHERE state = 'waiting' AND wait_reason = 'decision' ORDER BY seq LIMIT 200`)
		if err != nil {
			return "", persistence.MapError(err)
		}
		var ids []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return "", persistence.MapError(err)
			}
			ids = append(ids, id)
		}
		rows.Close()
		released := 0
		for _, id := range ids {
			if _, err := q.Resume(ctx, id); err == nil {
				released++
			}
			// Recusa da porta (decisão ainda pendente) não é erro: o item segue esperando.
		}
		if released == 0 {
			return "", nil
		}
		return fmt.Sprintf("%d item(ns) devolvido(s) à fila", released), nil
	}}
}

// RecoverJob recupera tentativas interrompidas (lease vencida): volta à fila quando é seguro e abre
// decisão quando há dúvida. Sem inspeção possível, adia sem alterar nada.
func RecoverJob(r *recovery.Recoverer, every time.Duration) Job {
	return Job{Name: "recuperar-tentativas", Every: every, Run: func(ctx context.Context) (string, error) {
		outs, err := r.RecoverAll(ctx, nil)
		if err != nil || len(outs) == 0 {
			return "", err
		}
		counts := map[string]int{}
		for _, o := range outs {
			counts[o.Action]++
		}
		return fmt.Sprintf("%d tentativa(s): %v", len(outs), counts), nil
	}}
}

// RetentionJob aplica a retenção. Sem `apply`, só simula e registra o que removeria.
func RetentionJob(r *retention.Retention, pol retention.Policy, apply bool, every time.Duration) Job {
	name := "retencao-simulada"
	if apply {
		name = "retencao"
	}
	return Job{Name: name, Every: every, Run: func(ctx context.Context) (string, error) {
		rep, err := r.Run(ctx, pol, !apply)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("apply=%v %+v", apply, rep), nil
	}}
}

// SentinelEventsJob entrega os eventos internos pendentes ao coordenador da Sentinela (reavaliação
// imediata após conclusão, falha, mudança de estado ou vaga liberada). Pendentes sobrevivem a reinício.
func SentinelEventsJob(db persistence.DBTX, handle func(context.Context, sentinel.Event) error, every time.Duration) Job {
	return Job{Name: "sentinela-eventos", Every: every, Run: func(ctx context.Context) (string, error) {
		h, f, err := sentinel.Drain(ctx, db, 200, handle)
		if err != nil || h+f == 0 {
			return "", err
		}
		return fmt.Sprintf("%d tratado(s), %d pendente(s) por falha", h, f), nil
	}}
}

// SentinelRoutinesJob executa as rotinas agendadas vencidas (sync, correções, PRs, manutenção),
// cada uma com relógio e cursor próprios. scopes lista os projetos ("" = global).
func SentinelRoutinesJob(s *sentinel.Scheduler, scopes func(context.Context) ([]string, error), every time.Duration) Job {
	return Job{Name: "sentinela-rotinas", Every: every, Run: func(ctx context.Context) (string, error) {
		sc, err := scopes(ctx)
		if err != nil {
			return "", err
		}
		res, err := s.Tick(ctx, sc)
		if err != nil || len(res) == 0 {
			return "", err
		}
		failed := 0
		for _, r := range res {
			if r.Err != nil {
				failed++
			}
		}
		return fmt.Sprintf("%d rotina(s) executada(s), %d com falha", len(res), failed), nil
	}}
}
