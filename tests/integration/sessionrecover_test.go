package integration

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
	"github.com/ViniciusBernucci/le-fabrique/internal/sessionrecover"
	"github.com/ViniciusBernucci/le-fabrique/internal/worktrees"
)

type fakeResumer struct {
	lost    bool
	resumed []string
	started []string
}

func (f *fakeResumer) Resume(_ context.Context, id, brief string) error {
	if f.lost {
		return sessionrecover.ErrSessionLost
	}
	f.resumed = append(f.resumed, id+"|"+brief)
	return nil
}
func (f *fakeResumer) Start(_ context.Context, brief string) (string, error) {
	f.started = append(f.started, brief)
	return "sess-new", nil
}

// AJ03-04 / CT07, CT15 — processo longo saudável não é encerrado; retomada nativa; sessão perdida cai para
// nova sessão com checkpoint; escritor antigo perde a autoridade; patch, não rastreados autorizados, commits e
// efeitos externos são preservados e informados.
func TestSessionRecoveryResumesOrFallsBackAndPreservesWork(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "rec")
	var feature string
	_ = db.Pool.QueryRow(ctx, `INSERT INTO features (project_id, external_id, title) VALUES ($1,'RC-1','f') RETURNING id::text`, p.ID).Scan(&feature)
	repo, base := newRepoWithBase(t)
	wt := &worktrees.Manager{DB: db.Pool, Repo: repo, Root: t.TempDir()}
	a, err := wt.Provision(ctx, worktrees.Spec{ProjectID: p.ID, FeatureID: feature, Track: "FRONTEND", Kind: "implementation", Attempt: 1, BaseSHA: base, Paths: []string{"apps/web"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = wt.SetWriter(ctx, a.ID, true)
	// Trabalho em curso: um commit, uma alteração rastreada, um arquivo novo autorizado e outro fora do escopo.
	writeCommit(t, a.Path, "apps/web/feito.ts", "commit\n")
	_ = os.WriteFile(filepath.Join(a.Path, "apps/web/a.ts"), []byte("rascunho\n"), 0o644)
	_ = os.WriteFile(filepath.Join(a.Path, "apps/web/novo.ts"), []byte("novo\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(a.Path, "internal/api"), 0o755)
	_ = os.WriteFile(filepath.Join(a.Path, "internal/api/fora.go"), []byte("package api\n"), 0o644)
	// Efeitos externos: um confirmado e um incerto.
	_, _ = db.Pool.Exec(ctx, `INSERT INTO outbox (project_id, business_key, destination, state) VALUES ($1,'exec:E1:push:1','git','confirmed'), ($1,'exec:E1:pr:1','github','uncertain')`, p.ID)

	rs := &sentinel.Reserver{DB: db.Pool}
	old, err := rs.Acquire(ctx, p.ID, "worker-old", "FRONTEND")
	if err != nil {
		t.Fatal(err)
	}
	rec := &sessionrecover.Recoverer{DB: db.Pool, Wt: wt, Reserver: rs}
	caps := executor.Capabilities{NativeSession: true, NativeResume: true}
	req := sessionrecover.Request{ProjectID: p.ID, Slot: "FRONTEND", OldHolder: "worker-old", NewHolder: "worker-new", AttemptID: a.ID, EffectPrefix: "exec:E1:",
		Obs: sessionrecover.Observation{Caps: caps, SessionID: "sess-1", ProcessAlive: true, CommandHealthy: true, LeaseValid: true}}

	// 1) Comando longo e saudável, sem stdout: não se interrompe nada.
	fr := &fakeResumer{}
	res, err := rec.Recover(ctx, fr, req)
	if err != nil || res.Strategy != sessionrecover.Wait || len(fr.resumed)+len(fr.started) != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	if err := rs.Validate(ctx, p.ID, old[0]); err != nil {
		t.Fatalf("a reserva do titular segue válida enquanto espera: %v", err)
	}
	if cur, _ := wt.Get(ctx, a.ID); !cur.WriterActive {
		t.Fatal("o escritor não pode ser desligado durante a espera")
	}

	// 2) Processo morreu e a sessão nativa existe: retoma a original.
	req.Obs.ProcessAlive, req.Obs.CommandHealthy = false, false
	res, err = rec.Recover(ctx, fr, req)
	if err != nil || res.Strategy != sessionrecover.NativeResume || res.Session != "sess-1" || len(fr.resumed) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	// O escritor antigo perdeu a autoridade: escrita tardia é recusada.
	if err := rs.Validate(ctx, p.ID, old[0]); !errors.Is(err, sentinel.ErrStaleGeneration) {
		t.Fatalf("escritor atrasado deve ser recusado: %v", err)
	}
	pr := res.Preserved
	if pr.Checkpoint == "" || len(pr.Commits) != 1 || len(pr.OutOfScope) != 1 || pr.OutOfScope[0] != "internal/api/fora.go" ||
		len(pr.AppliedEffects) != 1 || len(pr.Uncertain) != 1 || !strings.Contains(pr.Uncertain[0], "uncertain") {
		t.Fatalf("%+v", pr)
	}
	// O checkpoint guarda o patch e o arquivo novo autorizado.
	if gitIn(t, repo, "show", pr.Checkpoint+":apps/web/novo.ts") != "novo" || gitIn(t, repo, "show", pr.Checkpoint+":apps/web/a.ts") != "rascunho" {
		t.Fatal("o checkpoint deve conter o patch e o não rastreado")
	}
	for _, want := range []string{"NÃO repetir", "exec:E1:push:1", "incerta", "fora do escopo"} {
		if !strings.Contains(res.Brief, want) {
			t.Fatalf("o resumo da retomada deve conter %q:\n%s", want, res.Brief)
		}
	}

	// 3) Sessão nativa perdida: nova sessão com checkpoint e reconciliação.
	fr2 := &fakeResumer{lost: true}
	req.NewHolder = "worker-new-2"
	res, err = rec.Recover(ctx, fr2, req)
	if err != nil || res.Strategy != sessionrecover.NewSession || res.Session != "sess-new" || len(fr2.started) != 1 || !strings.Contains(fr2.started[0], "nova sessão a partir do checkpoint") {
		t.Fatalf("%+v %v", res, err)
	}
	// 4) Cliente sem retomada nativa: vai direto à nova sessão; lease perdido idem.
	req.Obs.Caps = executor.Capabilities{}
	req.Obs.LeaseValid = false
	req.Obs.ProcessAlive, req.Obs.CommandHealthy = true, true
	fr3 := &fakeResumer{}
	req.NewHolder = "worker-new-3"
	if res, err = rec.Recover(ctx, fr3, req); err != nil || res.Strategy != sessionrecover.NewSession {
		t.Fatalf("lease perdido nunca vira espera: %+v %v", res, err)
	}
	// 5) Coleta de órfãos: sem escritor vivo, a tentativa é listada e a limpeza preserva o trabalho.
	_ = wt.SetWriter(ctx, a.ID, false)
	orphans, err := wt.Orphans(ctx, p.ID, map[string]bool{})
	if err != nil || len(orphans) != 1 || orphans[0].ID != a.ID {
		t.Fatalf("%+v %v", orphans, err)
	}
	if live, _ := wt.Orphans(ctx, p.ID, map[string]bool{a.ID: true}); len(live) != 0 {
		t.Fatal("tentativa com executor vivo não é órfã")
	}
	if err := wt.Release(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if got := gitIn(t, repo, "show", pr.Checkpoint+":apps/web/novo.ts"); got != "novo" {
		t.Fatalf("o trabalho segue recuperável depois da coleta: %q", got)
	}
}

func TestDecideNeverWaitsOnStaleLeaseNorKillsHealthySilence(t *testing.T) {
	healthy := sessionrecover.Observation{ProcessAlive: true, CommandHealthy: true, LeaseValid: true}
	if sessionrecover.Decide(healthy) != sessionrecover.Wait {
		t.Fatal("comando saudável sem stdout deve esperar")
	}
	healthy.LeaseValid = false
	if sessionrecover.Decide(healthy) == sessionrecover.Wait {
		t.Fatal("lease perdido não espera")
	}
}
