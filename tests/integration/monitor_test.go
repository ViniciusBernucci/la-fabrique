package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/monitor"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func monitorExec(t *testing.T) (testDB, string, string) {
	t.Helper()
	db := newMigratedDB(t)
	p := mustProject(t, db.Pool, "mon")
	f, err := persistence.CreateFeature(context.Background(), db.Pool, p.ID, "MN-1", "f")
	if err != nil {
		t.Fatal(err)
	}
	ex, err := persistence.CreateExecution(context.Background(), db.Pool, p.ID, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	return db, p.ID, ex.ID
}

// AJ07-02 / CT10 — a saída aparece enquanto o processo roda, o cursor retoma sem duplicar, CLI e API
// compartilham a interface, e subagente sem telemetria não recebe progresso fabricado.
func TestMonitorShowsLiveOutputCursorSubagentsAndNeverInventsProgress(t *testing.T) {
	db, project, exec := monitorExec(t)
	ctx := context.Background()
	red := security.NewRedactor()
	red.Register("s3cr3t-token-value")
	rec := &monitor.Recorder{DB: db.Pool, Redactor: red}
	emit, finish := rec.Sink(ctx, project, exec)
	t0 := time.Date(2026, 5, 1, 10, 0, 0, 0, time.UTC)
	pct := 40
	emit(executor.Event{At: t0, Source: executor.CLI, Kind: executor.EvStart, Text: "iniciando"})
	emit(executor.Event{At: t0.Add(2 * time.Second), Source: executor.CLI, Kind: executor.EvMessage, Text: "usando s3cr3t-token-value no teste"})
	emit(executor.Event{At: t0.Add(3 * time.Second), Source: executor.CLI, Kind: executor.EvTool, Tool: "bash", Text: "go test ./..."})
	emit(executor.Event{At: t0.Add(4 * time.Second), Source: executor.CLI, Kind: executor.EvMessage, Subagent: "explorador", Text: "lendo"})
	emit(executor.Event{At: t0.Add(5 * time.Second), Source: executor.CLI, Kind: executor.EvMessage, Subagent: "revisor", Text: "analisando", Progress: &pct})
	finish()

	// Em andamento (sem `done`): conteúdo já visível e cronômetro correndo.
	v, err := monitor.Read(ctx, db.Pool, project, exec, 0, 100, t0.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Events) != 5 || v.Summary.Done || v.Summary.ElapsedSeconds != 10 || v.Summary.ToolCalls != 1 {
		t.Fatalf("%+v", v.Summary)
	}
	for _, e := range v.Events {
		if strings.Contains(e.Text, "s3cr3t-token-value") {
			t.Fatalf("segredo na saída: %q", e.Text)
		}
	}
	sub := map[string]monitor.Subagent{}
	for _, s := range v.Summary.Subagents {
		sub[s.Name] = s
	}
	if sub["explorador"].Progress != nil || sub["revisor"].Progress == nil || *sub["revisor"].Progress != 40 || v.Summary.Progress != nil {
		t.Fatalf("progresso só do que o executor informou: %+v", v.Summary)
	}
	// Reconexão pelo cursor: só o que é novo, sem repetir.
	cursor := v.Cursor
	emit2, finish2 := rec.Sink(ctx, project, exec)
	emit2(executor.Event{At: t0.Add(11 * time.Second), Source: executor.API, Kind: executor.EvMessage, Text: "resposta da API"})
	emit2(executor.Event{At: t0.Add(12 * time.Second), Source: executor.API, Kind: executor.EvDone})
	finish2()
	v2, err := monitor.Read(ctx, db.Pool, project, exec, cursor, 100, t0.Add(20*time.Second))
	if err != nil || len(v2.Events) != 2 || v2.Events[0].Seq != cursor+1 || !v2.Summary.Done || v2.Summary.ElapsedSeconds != 12 {
		t.Fatalf("%+v %v", v2, err)
	}
	if again, _ := monitor.Read(ctx, db.Pool, project, exec, v2.Cursor, 100, t0); len(again.Events) != 0 {
		t.Fatalf("sem eventos novos: %+v", again.Events)
	}
}

// AJ07-02 — sob pressão o descarte é explícito: eventos gravados + descartados = emitidos.
func TestMonitorMakesBackpressureGapsExplicit(t *testing.T) {
	db, project, exec := monitorExec(t)
	ctx := context.Background()
	rec := &monitor.Recorder{DB: db.Pool, Buffer: 1}
	emit, finish := rec.Sink(ctx, project, exec)
	const total = 300
	for i := 0; i < total; i++ {
		emit(executor.Event{At: time.Now().UTC(), Source: executor.CLI, Kind: executor.EvMessage, Text: "linha"})
	}
	dropped := finish()
	v, err := monitor.Read(ctx, db.Pool, project, exec, 0, 1000, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var msgs int
	for _, e := range v.Events {
		if e.Kind == "message" {
			msgs++
		}
	}
	if dropped == 0 || int64(msgs)+dropped != total || v.Summary.Dropped != dropped {
		t.Fatalf("gravados=%d descartados=%d (resumo %d)", msgs, dropped, v.Summary.Dropped)
	}
}
