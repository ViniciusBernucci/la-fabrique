package main

import (
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/admission"
	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/worker"
)

// registerAdmission liga o laço de admissão (opt-in: SENTINEL_ADMISSION=true): a varredura periódica e a
// reavaliação por evento durável. Ainda não há executor de trilhas conectado ao worker, então o laço roda em
// observação (Starter nulo): decide e audita em admission_decisions sem reservar nem mudar estado.
func registerAdmission(w *worker.Worker, sch *sentinel.Scheduler, pool *pgxpool.Pool, cfg config.Worker) {
	if !cfg.Admission {
		return
	}
	loop := &admission.Loop{DB: pool, PlanningSlots: cfg.PlanningSlots}
	if err := sch.Register(loop.Routine(cfg.AdmissionInterval)); err != nil {
		fmt.Fprintln(os.Stderr, "worker: "+err.Error())
		return
	}
	w.AddJob(worker.SentinelEventsJob(pool, loop.HandleEvent, cfg.JobsInterval))
}
