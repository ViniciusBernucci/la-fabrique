package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

func ckSandbox(t *testing.T, project, feature string) *execution.Sandbox {
	t.Helper()
	d := &execution.Docker{}
	if !d.Available(t.Context()) {
		t.Skip("Docker indisponível")
	}
	sb, err := d.Provision(t.Context(), execution.Spec{ProjectID: project, FeatureID: feature, Profile: execution.Profiles["base"]})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sb.Destroy(context.Background()) })
	return sb
}

func ckRun(t *testing.T, sb *execution.Sandbox, script string) string {
	t.Helper()
	r, err := sb.Exec(t.Context(), execution.ExecSpec{Cmd: []string{"sh", "-c", script}})
	if err != nil || r.ExitCode != 0 {
		t.Fatalf("%q: %+v %v", script, r, err)
	}
	return r.Stdout
}

const ckWorkspace = `cd /workspace &&
mkdir -p src/pkg vazio docs &&
printf 'package main\n' > src/main.go &&
printf 'helper\n' > src/pkg/helper.go &&
printf '#!/bin/sh\necho oi\n' > run.sh && chmod 755 run.sh &&
head -c 70000 /dev/urandom > blob.bin &&
printf 'igual' > docs/a.txt && printf 'igual' > docs/b.txt &&
ln -s src/main.go link_interno && ln -s /etc/passwd link_hostil && ln -s nao/existe pendente`

type ckEnv struct {
	e     *authEnv
	store *checkpoints.Store
	proj  string
	exec  string
	cl    *client
}

func newCkEnv(t *testing.T, name string) ckEnv {
	t.Helper()
	e, cl, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, name)
	ex := waitingExecution(t, e, p.ID, "CK-1")
	return ckEnv{cl: cl, e: e, store: &checkpoints.Store{DB: e.db.Pool}, proj: p.ID, exec: ex.ID}
}

func (c ckEnv) input(label string) checkpoints.Input {
	return checkpoints.Input{ProjectID: c.proj, ExecutionID: c.exec, Label: label,
		State: json.RawMessage(`{"status":"waiting","queue":[],"vars":{"n":1},"steps":4}`),
		Refs:  map[string]string{"branch": "feature/ck", "pr": "42"}}
}

func (c ckEnv) objects(t *testing.T) (n int) {
	t.Helper()
	if err := c.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM checkpoint_objects WHERE project_id = $1`, c.proj).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return
}

func (c ckEnv) count(t *testing.T) (n int) {
	t.Helper()
	if err := c.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM checkpoints WHERE execution_id = $1`, c.exec).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return
}

// T12-01 — "Restaurar em container novo e comparar hashes."
func TestCheckpointRestoresInANewContainerAndHashesMatch(t *testing.T) {
	c := newCkEnv(t, "ck-restore")
	ctx := context.Background()
	a := ckSandbox(t, c.proj, "feat-a")
	ckRun(t, a, ckWorkspace)
	before := ckRun(t, a, `cd /workspace && find . -type f | sort | xargs sha256sum`)

	cp, err := c.store.CaptureAndRelease(ctx, a, c.input("antes da pausa"), checkpoints.Options{})
	if err != nil {
		t.Fatal(err)
	}
	// Recursos liberados só depois do checkpoint durável.
	if a.Exists(ctx) {
		t.Fatal("o sandbox de origem deveria ter sido destruído")
	}
	if cp.Seq != 1 || cp.Manifest.TreeHash != cp.TreeHash || cp.Refs["pr"] != "42" || string(cp.State) == "" || cp.Manifest.Git.Available {
		t.Fatalf("%+v", cp)
	}
	by := map[string]checkpoints.FileEntry{}
	for _, f := range cp.Manifest.Files {
		by[f.Path] = f
	}
	if by["run.sh"].Mode != 0o755 || by["link_hostil"].Link != "/etc/passwd" || by["link_hostil"].Kind != checkpoints.KindSymlink || by["vazio"].Kind != checkpoints.KindDir || by["docs/a.txt"].SHA256 != by["docs/b.txt"].SHA256 {
		t.Fatalf("manifesto: %+v", by)
	}
	// O conteúdo apontado pelo link hostil NÃO foi lido nem gravado.
	if n := c.objects(t); n != 5 { // main.go, helper.go, run.sh, blob.bin e "igual" (a.txt e b.txt dividem um objeto)
		t.Fatalf("objetos gravados: %d", n)
	}

	// Container NOVO, outro feature: restaura e compara.
	b := ckSandbox(t, c.proj, "feat-b")
	if err := c.store.Restore(ctx, cp, checkpoints.SandboxWorkspace{S: b}, checkpoints.Options{}); err != nil {
		t.Fatal(err)
	}
	after := ckRun(t, b, `cd /workspace && find . -type f | sort | xargs sha256sum`)
	if before != after {
		t.Fatalf("hashes diferem:\n%s\n---\n%s", before, after)
	}
	if got := strings.TrimSpace(ckRun(t, b, `readlink /workspace/link_hostil; readlink /workspace/pendente; [ -x /workspace/run.sh ] && echo exec; [ -d /workspace/vazio ] && echo vazio`)); got != "/etc/passwd\nnao/existe\nexec\nvazio" {
		t.Fatalf("links, modo e diretório vazio: %q", got)
	}
	// Restaurar de novo no mesmo destino (já com conteúdo) é recusado.
	if err := c.store.Restore(ctx, cp, checkpoints.SandboxWorkspace{S: b}, checkpoints.Options{}); !errors.Is(err, checkpoints.ErrNotEmpty) {
		t.Fatalf("destino não vazio: %v", err)
	}
	// Qualquer divergência depois é detectada, com o caminho.
	ckRun(t, b, `echo mudou >> /workspace/src/main.go`)
	if err := checkpoints.Verify(ctx, cp, checkpoints.SandboxWorkspace{S: b}, checkpoints.Options{}); !errors.Is(err, checkpoints.ErrMismatch) || !strings.Contains(err.Error(), "src/main.go") {
		t.Fatalf("divergência: %v", err)
	}
}

// T12-01 — a parada só vale com o sandbox parado.
func TestCheckpointRefusesToCaptureWhileProcessesAreAlive(t *testing.T) {
	c := newCkEnv(t, "ck-quiescent")
	ctx := context.Background()
	sb := ckSandbox(t, c.proj, "feat-q")
	ckRun(t, sb, `echo dado > /workspace/a.txt`)
	run, err := sb.Start(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "sleep 300"}})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		if quiet, _ := sb.VerifyQuiescent(ctx); !quiet {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("o processo não apareceu")
		}
	}
	if _, err := c.store.CaptureAndRelease(ctx, sb, c.input("x"), checkpoints.Options{}); !errors.Is(err, checkpoints.ErrNotQuiescent) {
		t.Fatalf("%v", err)
	}
	if !sb.Exists(ctx) || c.count(t) != 0 {
		t.Fatal("nada é gravado nem liberado com processo vivo")
	}
	if err := run.Cancel(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := c.store.CaptureAndRelease(ctx, sb, c.input("parado"), checkpoints.Options{}); err != nil {
		t.Fatalf("parado: %v", err)
	}
	if sb.Exists(ctx) || c.count(t) != 1 {
		t.Fatal("depois do checkpoint o sandbox é liberado")
	}
}

// T12-01 — "Interromper gravação de snapshot mantém checkpoint anterior consistente."
func TestInterruptedSnapshotWriteKeepsThePreviousCheckpointConsistent(t *testing.T) {
	c := newCkEnv(t, "ck-interrupt")
	ctx := context.Background()
	sb := ckSandbox(t, c.proj, "feat-i")
	ckRun(t, sb, ckWorkspace)
	ws := checkpoints.SandboxWorkspace{S: sb}
	snap1, err := checkpoints.Capture(ctx, ws, checkpoints.Options{})
	if err != nil {
		t.Fatal(err)
	}
	cp1, err := c.store.Save(ctx, snap1, c.input("primeiro"))
	if err != nil {
		t.Fatal(err)
	}
	objs1 := c.objects(t)

	// O workspace muda e a gravação do segundo snapshot é interrompida em cada estágio.
	ckRun(t, sb, `echo novo-1 > /workspace/n1.txt; echo novo-2 > /workspace/n2.txt; echo novo-3 > /workspace/n3.txt`)
	snap2, err := checkpoints.Capture(ctx, ws, checkpoints.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, stage := range []struct {
		name string
		at   string
		i    int
	}{{"no meio dos objetos", "object", 2}, {"antes de inserir o checkpoint", "checkpoint", 0}} {
		failing := &checkpoints.Store{DB: c.e.db.Pool, Hook: func(s string, i int) error {
			if s == stage.at && i == stage.i {
				return errors.New("queda simulada")
			}
			return nil
		}}
		if _, err := failing.Save(ctx, snap2, c.input("interrompido")); err == nil {
			t.Fatalf("%s: a gravação deveria falhar", stage.name)
		}
		got, err := c.store.Latest(ctx, c.proj, c.exec)
		if err != nil || got.ID != cp1.ID || got.Seq != 1 || got.ManifestHash != cp1.ManifestHash || c.count(t) != 1 {
			t.Fatalf("%s: o checkpoint anterior deve seguir intacto: %+v %v", stage.name, got, err)
		}
		if n := c.objects(t); n != objs1 {
			t.Fatalf("%s: objetos órfãos da gravação interrompida: %d → %d", stage.name, objs1, n)
		}
	}
	// Cancelamento do contexto no meio da gravação.
	cctx, cancel := context.WithCancel(ctx)
	cancelling := &checkpoints.Store{DB: c.e.db.Pool, Hook: func(s string, i int) error {
		if s == "object" && i == 1 {
			cancel()
		}
		return nil
	}}
	if _, err := cancelling.Save(cctx, snap2, c.input("cancelado")); err == nil {
		t.Fatal("contexto cancelado deve falhar")
	}
	if c.count(t) != 1 || c.objects(t) != objs1 {
		t.Fatal("o cancelamento não deixa rastro")
	}
	// O anterior ainda restaura em container novo e confere.
	got, _ := c.store.Latest(ctx, c.proj, c.exec)
	nb := ckSandbox(t, c.proj, "feat-i2")
	if err := c.store.Restore(ctx, got, checkpoints.SandboxWorkspace{S: nb}, checkpoints.Options{}); err != nil {
		t.Fatalf("restaurar o anterior: %v", err)
	}
	// Captura interrompida (tar cortado) nem chega ao banco.
	if _, err := checkpoints.Capture(ctx, brokenWS{ws}, checkpoints.Options{}); err == nil {
		t.Fatal("captura cortada deve falhar")
	}
	// Depois de tudo isso, o próximo checkpoint sai normalmente, sem buraco na sequência.
	cp2, err := c.store.Save(ctx, snap2, c.input("segundo"))
	if err != nil || cp2.Seq != 2 || c.count(t) != 2 {
		t.Fatalf("%+v %v", cp2, err)
	}
}

// brokenWS entrega só o começo do tar e depois falha (queda no meio da leitura).
type brokenWS struct{ checkpoints.SandboxWorkspace }

func (b brokenWS) Tar(ctx context.Context) (io.ReadCloser, func() error, error) {
	rc, wait, err := b.SandboxWorkspace.Tar(ctx)
	if err != nil {
		return nil, nil, err
	}
	return cutReader{rc: rc, left: 3000}, wait, nil
}

type cutReader struct {
	rc   io.ReadCloser
	left int
}

func (c cutReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (c cutReader) Close() error               { return c.rc.Close() }

func TestCheckpointCorruptionIsDetectedOnReadAndRestore(t *testing.T) {
	c := newCkEnv(t, "ck-corrupt")
	ctx := context.Background()
	sb := ckSandbox(t, c.proj, "feat-c")
	ckRun(t, sb, ckWorkspace)
	snap, _ := checkpoints.Capture(ctx, checkpoints.SandboxWorkspace{S: sb}, checkpoints.Options{})
	cp, err := c.store.Save(ctx, snap, c.input("íntegro"))
	if err != nil {
		t.Fatal(err)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := c.e.db.Pool.Exec(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	restore := func() error {
		return c.store.Restore(ctx, cp, checkpoints.SandboxWorkspace{S: ckSandbox(t, c.proj, fmt.Sprintf("feat-c-%d", time.Now().UnixNano()%100000))}, checkpoints.Options{})
	}
	main := ""
	for _, f := range snap.Manifest.Files {
		if f.Path == "src/main.go" {
			main = f.SHA256
		}
	}
	// Conteúdo adulterado com o mesmo tamanho: o hash do objeto não confere.
	exec(`UPDATE checkpoint_objects SET data = decode(repeat('41', size), 'hex') WHERE project_id = $1 AND hash = $2`, c.proj, main)
	if _, err := c.store.Object(ctx, c.proj, main); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("objeto adulterado: %v", err)
	}
	if err := restore(); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("restaurar com objeto adulterado: %v", err)
	}
	// Objeto ausente.
	exec(`DELETE FROM checkpoint_objects WHERE project_id = $1 AND hash = $2`, c.proj, main)
	if err := restore(); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("restaurar com objeto ausente: %v", err)
	}
	// Manifesto e estado adulterados no banco.
	exec(`UPDATE checkpoints SET manifest = jsonb_set(manifest, '{total_bytes}', '1') WHERE id = $1`, cp.ID)
	if _, err := c.store.Get(ctx, c.proj, cp.ID); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("manifesto adulterado: %v", err)
	}
	exec(`UPDATE checkpoints SET manifest = $2::jsonb WHERE id = $1`, cp.ID, mustJSON(t, cp.Manifest))
	if _, err := c.store.Get(ctx, c.proj, cp.ID); err != nil {
		t.Fatalf("manifesto restaurado deveria ler: %v", err)
	}
	exec(`UPDATE checkpoints SET state = '{"status":"succeeded"}'::jsonb WHERE id = $1`, cp.ID)
	if _, err := c.store.Latest(ctx, c.proj, c.exec); !errors.Is(err, checkpoints.ErrCorrupt) {
		t.Fatalf("estado adulterado: %v", err)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCheckpointsAreScopedByProjectAndPrunedSafely(t *testing.T) {
	c := newCkEnv(t, "ck-scope")
	ctx := context.Background()
	other := mustProject(t, c.e.db.Pool, "ck-outro")
	sb := ckSandbox(t, c.proj, "feat-s")
	ckRun(t, sb, `cd /workspace && echo comum > comum.txt && echo v1 > v.txt`)
	ws := checkpoints.SandboxWorkspace{S: sb}
	snap1, _ := checkpoints.Capture(ctx, ws, checkpoints.Options{})
	cp1, err := c.store.Save(ctx, snap1, c.input("1"))
	if err != nil {
		t.Fatal(err)
	}
	ckRun(t, sb, `cd /workspace && echo v2 > v.txt`)
	snap2, _ := checkpoints.Capture(ctx, ws, checkpoints.Options{})
	cp2, _ := c.store.Save(ctx, snap2, c.input("2"))
	ckRun(t, sb, `cd /workspace && echo v3 > v.txt`)
	snap3, _ := checkpoints.Capture(ctx, ws, checkpoints.Options{})
	cp3, _ := c.store.Save(ctx, snap3, c.input("3"))
	if cp1.Seq != 1 || cp2.Seq != 2 || cp3.Seq != 3 {
		t.Fatalf("sequência: %d %d %d", cp1.Seq, cp2.Seq, cp3.Seq)
	}
	// Outro projeto não lê o checkpoint nem os objetos.
	if _, err := c.store.Get(ctx, other.ID, cp1.ID); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("outro projeto: %v", err)
	}
	if _, err := c.store.Object(ctx, other.ID, cp1.Manifest.Files[0].SHA256); err == nil {
		t.Fatal("objeto de outro projeto")
	}
	if _, err := c.store.Latest(ctx, other.ID, c.exec); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("latest de outro projeto: %v", err)
	}
	// Poda: fica o mais recente; objetos só do passado somem, os compartilhados ficam.
	before := c.objects(t) // comum, v1, v2, v3
	removed, err := c.store.Prune(ctx, c.proj, c.exec, 1)
	if err != nil || removed != 2 {
		t.Fatalf("%d %v", removed, err)
	}
	if after := c.objects(t); after != before-2 {
		t.Fatalf("objetos: %d → %d", before, after)
	}
	latest, err := c.store.Latest(ctx, c.proj, c.exec)
	if err != nil || latest.ID != cp3.ID {
		t.Fatalf("%+v %v", latest, err)
	}
	nb := ckSandbox(t, c.proj, "feat-s2")
	if err := c.store.Restore(ctx, latest, checkpoints.SandboxWorkspace{S: nb}, checkpoints.Options{}); err != nil {
		t.Fatalf("o que sobrou da poda ainda restaura: %v", err)
	}
	if _, err := c.store.Prune(ctx, c.proj, c.exec, 0); err == nil {
		t.Fatal("keep=0 não é permitido")
	}
}
