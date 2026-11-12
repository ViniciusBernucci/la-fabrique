package integration

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

var ctx = context.Background()

func mustProject(t *testing.T, db persistence.DBTX, slug string) projects.Project {
	t.Helper()
	p, err := projects.Create(ctx, db, projects.NewProject{Slug: slug, Name: "Projeto " + slug})
	if err != nil {
		t.Fatalf("criar projeto %s: %v", slug, err)
	}
	return p
}

func TestMigrationPathPreservesData(t *testing.T) {
	dbURL := newDatabase(t)
	m, err := persistence.NewMigrator(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()

	// Schema anterior: só a migration 1, com fixture.
	if _, err := m.UpTo(ctx, 1); err != nil {
		t.Fatalf("migrate até 1: %v", err)
	}
	pool := connect(t, dbURL)
	// Insert direto: no schema 1 a tabela ainda não tem as colunas do cadastro (T05-01).
	var p projects.Project
	if err := pool.QueryRow(ctx, `INSERT INTO projects (slug, name) VALUES ('legado', 'Projeto legado') RETURNING id`).Scan(&p.ID); err != nil {
		t.Fatalf("criar projeto legado: %v", err)
	}

	// Upgrade para o schema atual.
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	got, err := projects.Get(ctx, pool, p.ID)
	if err != nil || got.Slug != "legado" || got.Version != 1 {
		t.Fatalf("projeto deveria sobreviver ao upgrade: %+v err=%v", got, err)
	}
	if got.WizardStep != "basics" || got.Source.Kind != "" || got.Repository.ConnectionID != nil {
		t.Fatalf("projeto legado deve ganhar rascunho vazio de vínculos: %+v", got)
	}
	if _, err := persistence.CreateFeature(ctx, pool, p.ID, "EXT-1", "Feature nova"); err != nil {
		t.Fatalf("tabelas da migration 2 deveriam existir: %v", err)
	}

	// Rollback e reaplicação (dev): as migrations 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3 e 2 saem e voltam.
	for i := 0; i < 15; i++ {
		if _, err := m.Down(ctx); err != nil {
			t.Fatalf("down %d: %v", i+1, err)
		}
	}
	if _, err := persistence.CreateFeature(ctx, pool, p.ID, "EXT-2", "x"); err == nil {
		t.Fatal("features não deveria existir após o down")
	}
	if _, err := m.Up(ctx); err != nil {
		t.Fatalf("reaplicar: %v", err)
	}
}

func TestDataSurvivesRestart(t *testing.T) {
	db := newMigratedDB(t)
	p := mustProject(t, db.Pool, "persistente")
	f, err := persistence.CreateFeature(ctx, db.Pool, p.ID, "EXT-9", "Persistir")
	if err != nil {
		t.Fatal(err)
	}
	ex, err := persistence.CreateExecution(ctx, db.Pool, p.ID, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistence.WithTx(ctx, db.Pool, func(tx pgx.Tx) error {
		_, err := persistence.AppendEvent(ctx, tx, p.ID, ex.ID, "started", map[string]string{"k": "v"})
		return err
	}); err != nil {
		t.Fatal(err)
	}

	db.Pool.Close() // "restart" do serviço: novo pool, mesmo banco
	pool := connect(t, db.URL)

	if got, err := projects.Get(ctx, pool, p.ID); err != nil || got.Name != p.Name {
		t.Fatalf("projeto após restart: %+v %v", got, err)
	}
	events, err := persistence.EventsAfter(ctx, pool, p.ID, ex.ID, 0, 10)
	if err != nil || len(events) != 1 || events[0].Type != "started" {
		t.Fatalf("eventos após restart: %+v %v", events, err)
	}
}

func TestDuplicatesAreRejected(t *testing.T) {
	db := newMigratedDB(t)
	a := mustProject(t, db.Pool, "alpha")
	b := mustProject(t, db.Pool, "beta")

	if _, err := projects.Create(ctx, db.Pool, projects.NewProject{Slug: "alpha", Name: "outro"}); !errors.Is(err, persistence.ErrDuplicate) {
		t.Fatalf("slug repetido deveria ser ErrDuplicate: %v", err)
	}
	if _, err := persistence.CreateFeature(ctx, db.Pool, a.ID, "EXT-1", "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := persistence.CreateFeature(ctx, db.Pool, a.ID, "EXT-1", "y"); !errors.Is(err, persistence.ErrDuplicate) {
		t.Fatalf("external_id repetido no mesmo projeto: %v", err)
	}
	if _, err := persistence.CreateFeature(ctx, db.Pool, b.ID, "EXT-1", "y"); err != nil {
		t.Fatalf("mesmo external_id em outro projeto é válido: %v", err)
	}

	first, created, err := persistence.EnqueueOutbox(ctx, db.Pool, a.ID, "pr:feat-1:1", "github", map[string]string{"a": "1"})
	if err != nil || !created {
		t.Fatalf("primeira intenção: %v created=%v", err, created)
	}
	again, created, err := persistence.EnqueueOutbox(ctx, db.Pool, a.ID, "pr:feat-1:1", "github", map[string]string{"a": "2"})
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("mesma chave deve devolver a entrada original: %+v created=%v err=%v", again, created, err)
	}

	if _, err := persistence.CreateExecution(ctx, db.Pool, a.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, persistence.ErrInvalidScope) {
		t.Fatalf("feature inexistente: %v", err)
	}
}

func TestSingleAdminInvariant(t *testing.T) {
	db := newMigratedDB(t)
	_, err := db.Pool.Exec(ctx, `INSERT INTO admins (username, password_hash) VALUES ('admin', 'x')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Pool.Exec(ctx, `INSERT INTO admins (username, password_hash) VALUES ('outro', 'x')`)
	if !errors.Is(persistence.MapError(err), persistence.ErrDuplicate) {
		t.Fatalf("segundo administrador deveria ser recusado: %v", err)
	}
}

func TestScopeIsEnforcedAcrossProjects(t *testing.T) {
	db := newMigratedDB(t)
	a := mustProject(t, db.Pool, "alpha")
	b := mustProject(t, db.Pool, "beta")
	fa, _ := persistence.CreateFeature(ctx, db.Pool, a.ID, "EXT-1", "x")
	ea, err := persistence.CreateExecution(ctx, db.Pool, a.ID, fa.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Execução de A usando o escopo de B.
	if _, err := persistence.CreateExecution(ctx, db.Pool, b.ID, fa.ID); !errors.Is(err, persistence.ErrInvalidScope) {
		t.Fatalf("feature de A em B: %v", err)
	}
	err = persistence.WithTx(ctx, db.Pool, func(tx pgx.Tx) error {
		_, err := persistence.AppendEvent(ctx, tx, b.ID, ea.ID, "x", nil)
		return err
	})
	if !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("evento da execução de A com escopo B: %v", err)
	}
	if _, err := persistence.UpdateExecutionState(ctx, db.Pool, b.ID, ea.ID, 1, "running"); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("atualizar execução de A com escopo B: %v", err)
	}
	sha := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	_, _, err = persistence.RegisterArtifact(ctx, db.Pool, persistence.Artifact{
		ProjectID: b.ID, ExecutionID: ea.ID, Kind: "patch", SHA256: sha, StorageRef: "b/patch.bin"})
	if !errors.Is(err, persistence.ErrInvalidScope) {
		t.Fatalf("artefato de B ligado à execução de A: %v", err)
	}
}

func TestOptimisticVersioning(t *testing.T) {
	db := newMigratedDB(t)
	p := mustProject(t, db.Pool, "versionado")
	name := "Novo nome"
	updated, err := projects.Update(ctx, db.Pool, p.ID, 1, projects.Patch{Name: &name})
	if err != nil || updated.Version != 2 || updated.Name != name {
		t.Fatalf("update: %+v %v", updated, err)
	}
	if _, err := projects.Update(ctx, db.Pool, p.ID, 1, projects.Patch{Name: &name}); !errors.Is(err, persistence.ErrVersionConflict) {
		t.Fatalf("versão obsoleta deveria conflitar: %v", err)
	}
	if _, err := projects.Update(ctx, db.Pool, "00000000-0000-0000-0000-000000000000", 1, projects.Patch{Name: &name}); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("projeto inexistente: %v", err)
	}

	// Escritores concorrentes com a mesma versão: exatamente um vence.
	const n = 12
	var wg sync.WaitGroup
	results := make(chan error, n)
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nm := fmt.Sprintf("concorrente-%d", i)
			_, err := projects.Update(ctx, db.Pool, p.ID, 2, projects.Patch{Name: &nm})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	wins, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			wins++
		case errors.Is(err, persistence.ErrVersionConflict):
			conflicts++
		default:
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	if wins != 1 || conflicts != n-1 {
		t.Fatalf("esperava 1 vencedor e %d conflitos, veio %d/%d", n-1, wins, conflicts)
	}
}

func TestEventSequenceIsMonotonicUnderConcurrency(t *testing.T) {
	db := newMigratedDB(t)
	p := mustProject(t, db.Pool, "eventos")
	f, _ := persistence.CreateFeature(ctx, db.Pool, p.ID, "EXT-1", "x")
	ex, _ := persistence.CreateExecution(ctx, db.Pool, p.ID, f.ID)

	const n = 25
	var wg sync.WaitGroup
	seqs := make(chan int64, n)
	errs := make(chan error, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- persistence.WithTx(ctx, db.Pool, func(tx pgx.Tx) error {
				ev, err := persistence.AppendEvent(ctx, tx, p.ID, ex.ID, "tick", nil)
				seqs <- ev.Seq
				return err
			})
		}()
	}
	wg.Wait()
	close(errs)
	close(seqs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var got []int64
	for s := range seqs {
		got = append(got, s)
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	for i, s := range got {
		if s != int64(i+1) {
			t.Fatalf("sequência com lacuna ou repetição: %v", got)
		}
	}
	tail, err := persistence.EventsAfter(ctx, db.Pool, p.ID, ex.ID, n-3, 100)
	if err != nil || len(tail) != 3 || tail[0].Seq != n-2 {
		t.Fatalf("cursor de reconexão: %+v %v", tail, err)
	}
}

func TestOutboxTransitions(t *testing.T) {
	db := newMigratedDB(t)
	p := mustProject(t, db.Pool, "outbox")
	e, _, _ := persistence.EnqueueOutbox(ctx, db.Pool, p.ID, "k1", "github", nil)

	if _, err := persistence.TransitionOutbox(ctx, db.Pool, p.ID, e.ID, e.Version, persistence.OutboxConfirmed, nil); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("intent → confirmed deveria ser recusado: %v", err)
	}
	sending, err := persistence.TransitionOutbox(ctx, db.Pool, p.ID, e.ID, e.Version, persistence.OutboxSending, nil)
	if err != nil || sending.Attempts != 1 || sending.Version != 2 {
		t.Fatalf("sending: %+v %v", sending, err)
	}
	if _, err := persistence.TransitionOutbox(ctx, db.Pool, p.ID, e.ID, e.Version, persistence.OutboxUncertain, nil); !errors.Is(err, persistence.ErrVersionConflict) {
		t.Fatalf("versão obsoleta: %v", err)
	}
	done, err := persistence.TransitionOutbox(ctx, db.Pool, p.ID, e.ID, sending.Version, persistence.OutboxConfirmed, map[string]string{"pr": "12"})
	if err != nil || done.State != persistence.OutboxConfirmed || string(done.Proof) == "" {
		t.Fatalf("confirmed: %+v %v", done, err)
	}
}

func TestArtifactsAreIdempotentAndReferencesAreSafe(t *testing.T) {
	db := newMigratedDB(t)
	p := mustProject(t, db.Pool, "artefatos")
	sha := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	a := persistence.Artifact{ProjectID: p.ID, Kind: "checkpoint", SHA256: sha, SizeBytes: 10, StorageRef: "checkpoints/ab/cd"}
	first, created, err := persistence.RegisterArtifact(ctx, db.Pool, a)
	if err != nil || !created {
		t.Fatalf("registrar: %v created=%v", err, created)
	}
	again, created, err := persistence.RegisterArtifact(ctx, db.Pool, a)
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("repetição deveria devolver o existente: %+v created=%v err=%v", again, created, err)
	}
	for _, ref := range []string{"/etc/passwd", "../fora", "a/../../b", "a//b", `a\b`, ""} {
		bad := a
		bad.SHA256 = "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
		bad.StorageRef = ref
		if _, _, err := persistence.RegisterArtifact(ctx, db.Pool, bad); !errors.Is(err, persistence.ErrInvalidInput) {
			t.Errorf("storage_ref %q deveria ser recusada: %v", ref, err)
		}
	}
	bad := a
	bad.SHA256 = "XYZ"
	if _, _, err := persistence.RegisterArtifact(ctx, db.Pool, bad); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Errorf("sha256 inválido: %v", err)
	}
	// A CHECK do banco é a segunda barreira, mesmo se o código for contornado.
	_, err = db.Pool.Exec(ctx, `INSERT INTO artifacts (project_id, kind, sha256, size_bytes, storage_ref)
		VALUES ($1, 'x', $2, 1, '../x')`, p.ID, "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	if !errors.Is(persistence.MapError(err), persistence.ErrInvalidInput) {
		t.Fatalf("CHECK do banco deveria recusar traversal: %v", err)
	}
}

func TestSessions(t *testing.T) {
	db := newMigratedDB(t)
	var adminID string
	if err := db.Pool.QueryRow(ctx, `INSERT INTO admins (username, password_hash) VALUES ('admin', 'x') RETURNING id`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	token, sess, err := security.CreateSession(ctx, db.Pool, adminID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := security.Authenticate(ctx, db.Pool, token, 5*time.Minute)
	if err != nil || got.Username != "admin" || got.ID != sess.ID {
		t.Fatalf("autenticar: %+v %v", got, err)
	}
	if _, err := security.Authenticate(ctx, db.Pool, token+"x", 5*time.Minute); !errors.Is(err, security.ErrNoSession) {
		t.Fatalf("token alterado: %v", err)
	}
	if _, err := security.Authenticate(ctx, db.Pool, "", 5*time.Minute); !errors.Is(err, security.ErrNoSession) {
		t.Fatalf("token vazio: %v", err)
	}
	var plain int
	_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM admin_sessions WHERE token_hash = convert_to($1, 'UTF8')`, token).Scan(&plain)
	if plain != 0 {
		t.Fatal("token em claro não pode ser persistido")
	}
	if err := security.Revoke(ctx, db.Pool, sess.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := security.Authenticate(ctx, db.Pool, token, 5*time.Minute); !errors.Is(err, security.ErrNoSession) {
		t.Fatalf("sessão revogada deveria falhar: %v", err)
	}

	expired, _, err := security.CreateSession(ctx, db.Pool, adminID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `UPDATE admin_sessions SET created_at = now() - interval '2 hours', expires_at = now() - interval '1 hour' WHERE token_hash = $1`, security.HashToken(expired)); err != nil {
		t.Fatal(err)
	}
	if _, err := security.Authenticate(ctx, db.Pool, expired, 5*time.Minute); !errors.Is(err, security.ErrNoSession) {
		t.Fatalf("sessão expirada deveria falhar: %v", err)
	}
}

func TestProjectListPagination(t *testing.T) {
	db := newMigratedDB(t)
	for i := range 5 {
		mustProject(t, db.Pool, fmt.Sprintf("proj-%d", i))
	}
	paused := "paused"
	first, _ := projects.List(ctx, db.Pool, projects.ListParams{Limit: 5})
	if _, err := projects.Update(ctx, db.Pool, first.Items[0].ID, 1, projects.Patch{State: &paused}); err != nil {
		t.Fatal(err)
	}

	var seen []string
	cursor := ""
	for range 10 {
		page, err := projects.List(ctx, db.Pool, projects.ListParams{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range page.Items {
			seen = append(seen, p.Slug)
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if len(seen) != 5 {
		t.Fatalf("paginação deveria cobrir os 5 projetos sem repetição: %v", seen)
	}
	only, _ := projects.List(ctx, db.Pool, projects.ListParams{State: "paused"})
	if len(only.Items) != 1 {
		t.Fatalf("filtro por estado: %+v", only)
	}
	if _, err := projects.List(ctx, db.Pool, projects.ListParams{Cursor: "lixo"}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("cursor inválido: %v", err)
	}
	if _, err := projects.List(ctx, db.Pool, projects.ListParams{Limit: 1000}); !errors.Is(err, persistence.ErrInvalidInput) {
		t.Fatalf("limit excessivo: %v", err)
	}
}

func TestWithTxRollsBack(t *testing.T) {
	db := newMigratedDB(t)
	boom := errors.New("boom")
	err := persistence.WithTx(ctx, db.Pool, func(tx pgx.Tx) error {
		if _, err := projects.Create(ctx, tx, projects.NewProject{Slug: "fantasma", Name: "x"}); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("esperava boom: %v", err)
	}
	page, _ := projects.List(ctx, db.Pool, projects.ListParams{})
	if len(page.Items) != 0 {
		t.Fatalf("rollback deveria desfazer o projeto: %+v", page)
	}
}

// A migration 3 apaga o admin do acesso de dev provisório (ADR-0004) e mantém uma conta real.
func TestAuthMigrationRemovesDevAdminButKeepsRealAdmin(t *testing.T) {
	for _, tc := range []struct {
		name, hash string
		kept       bool
	}{
		{"dev", "dev-auth-no-password", false},
		{"real", "$argon2id$v=19$m=65536,t=3,p=2$c2FsdA$aGFzaA", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dbURL := newDatabase(t)
			m, err := persistence.NewMigrator(dbURL)
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			if _, err := m.UpTo(ctx, 2); err != nil {
				t.Fatal(err)
			}
			pool := connect(t, dbURL)
			var id string
			if err := pool.QueryRow(ctx, `INSERT INTO admins (username, password_hash) VALUES ('dono', $1) RETURNING id`, tc.hash).Scan(&id); err != nil {
				t.Fatal(err)
			}
			if _, err := pool.Exec(ctx, `INSERT INTO admin_sessions (admin_id, token_hash, expires_at) VALUES ($1, $2, now() + interval '1 hour')`, id, security.HashToken("t")); err != nil {
				t.Fatal(err)
			}
			if _, err := m.Up(ctx); err != nil {
				t.Fatalf("migrate up: %v", err)
			}
			var admins, sessions int
			_ = pool.QueryRow(ctx, `SELECT count(*) FROM admins`).Scan(&admins)
			_ = pool.QueryRow(ctx, `SELECT count(*) FROM admin_sessions`).Scan(&sessions)
			want := 0
			if tc.kept {
				want = 1
			}
			if admins != want || sessions != want {
				t.Fatalf("admins=%d sessões=%d, esperado %d", admins, sessions, want)
			}
		})
	}
}
