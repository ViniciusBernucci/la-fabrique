package integration

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/envconfig"
)

type fakeEnvRunner struct {
	ready     map[string]bool
	browserOK bool
	cleanupOK bool
	calls     int
}

func (f *fakeEnvRunner) Run(_ context.Context, _ envconfig.Config, sha string) (envconfig.Evidence, error) {
	f.calls++
	return envconfig.Evidence{SHA: sha, Ready: f.ready, BrowserOK: f.browserOK, CleanupOK: f.cleanupOK}, nil
}

func goodEnv() envconfig.Config {
	return envconfig.Config{
		Services: []envconfig.Service{
			{Name: "web", Kind: "frontend", Ports: []int{5173}, Readiness: "wget -q http://localhost:5173", DependsOn: []string{"api"}},
			{Name: "api", Kind: "api", Ports: []int{8080}, Readiness: "wget -q http://localhost:8080/health", DependsOn: []string{"db"}},
			{Name: "db", Kind: "database", Image: "postgres:17-alpine", Readiness: "pg_isready"},
		},
		Start: "docker compose up -d", Test: "make test", DataSetup: "make migrate && make seed",
		Credentials: []envconfig.Credential{{Name: "DB_PASSWORD", Ref: "env:DB_PASSWORD"}}, BrowserTarget: "http://web:5173",
	}
}

// AJ08-04 / CT08, CT09 — o administrador salva, valida e aprova; a versão reaparece após reinício e é
// reutilizada; mudança relevante exige revalidação sem derrubar a aprovada; readiness com falha, segredo
// por valor e destino de produção impedem a promoção.
func TestEnvConfigVersioningValidationAndReuse(t *testing.T) {
	db := newMigratedDB(t)
	ctx := context.Background()
	p := mustProject(t, db.Pool, "envcfg")
	st := &envconfig.Store{DB: db.Pool}
	ok := &fakeEnvRunner{ready: map[string]bool{"web": true, "api": true, "db": true}, browserOK: true, cleanupOK: true}

	v1, err := st.Save(ctx, p.ID, goodEnv(), "sha1")
	if err != nil || v1.Version != 1 || v1.State != envconfig.StateProposal {
		t.Fatalf("%+v %v", v1, err)
	}
	if _, err := st.Approve(ctx, p.ID, 1, "admin"); !errors.Is(err, envconfig.ErrNotValidated) {
		t.Fatalf("não validada não é promovida: %v", err)
	}
	v1, err = st.Validate(ctx, p.ID, 1, "sha1", ok)
	if err != nil || v1.State != envconfig.StateValidated || v1.Evidence == nil || !v1.Evidence.BrowserOK {
		t.Fatalf("%+v %v", v1, err)
	}
	if v1, err = st.Approve(ctx, p.ID, 1, "admin"); err != nil || !v1.Active || v1.ApprovedBy != "admin" {
		t.Fatalf("%+v %v", v1, err)
	}
	// "Reinício": outro armazenamento sobre o mesmo banco enxerga a versão e outro review a reutiliza.
	st2 := &envconfig.Store{DB: db.Pool}
	got, err := st2.Resolve(ctx, p.ID, "sha2", func(envconfig.Config, string) error { return nil })
	if err != nil || got.Version != 1 || got.Config.BrowserTarget != "http://web:5173" {
		t.Fatalf("%+v %v", got, err)
	}
	// Mudança só de notas: herda a validação.
	c := goodEnv()
	c.Notes = "ajuste de texto"
	v2, err := st.Save(ctx, p.ID, c, "sha2")
	if err != nil || v2.State != envconfig.StateValidated {
		t.Fatalf("mudança não material herda a validação: %+v %v", v2, err)
	}
	// Mudança relevante (comando de subida): exige revalidação, a aprovada segue valendo.
	c = goodEnv()
	c.Start = "docker compose up -d --build"
	v3, err := st.Save(ctx, p.ID, c, "sha2")
	if err != nil || v3.State != envconfig.StateRevalidation || !strings.Contains(v3.Diagnostic, "mudança relevante") {
		t.Fatalf("%+v %v", v3, err)
	}
	view, _ := st.View(ctx, p.ID)
	if view.Active == nil || view.Active.Version != 1 || view.Candidate == nil || view.Candidate.Version != 3 {
		t.Fatalf("o painel distingue a validada da candidata: %+v", view)
	}
	// Readiness com falha impede promover.
	bad := &fakeEnvRunner{ready: map[string]bool{"web": true, "api": false, "db": true}, browserOK: true, cleanupOK: true}
	v3, _ = st.Validate(ctx, p.ID, 3, "sha2", bad)
	if v3.State != envconfig.StateInvalid || !strings.Contains(v3.Diagnostic, "api não ficou pronto") {
		t.Fatalf("%+v", v3)
	}
	if _, err := st.Approve(ctx, p.ID, 3, "admin"); !errors.Is(err, envconfig.ErrNotValidated) {
		t.Fatalf("%v", err)
	}
	// Limpeza não comprovada também impede.
	noClean := &fakeEnvRunner{ready: ok.ready, browserOK: true, cleanupOK: false}
	if v, _ := st.Validate(ctx, p.ID, 3, "sha2", noClean); v.State != envconfig.StateInvalid {
		t.Fatalf("%+v", v)
	}
	// Política: segredo por valor e destino de produção são recusados sem rodar nada.
	c = goodEnv()
	c.Credentials = []envconfig.Credential{{Name: "DB_PASSWORD", Ref: "hunter2"}}
	c.Test = "curl https://api.minhaempresa.com/health"
	v4, _ := st.Save(ctx, p.ID, c, "sha2")
	calls := ok.calls
	v4, _ = st.Validate(ctx, p.ID, v4.Version, "sha2", ok)
	if v4.State != envconfig.StateInvalid || ok.calls != calls || !strings.Contains(v4.Diagnostic, "referência env:NOME") || !strings.Contains(v4.Diagnostic, "produção") {
		t.Fatalf("%+v", v4)
	}
	// Preflight que falha na revisão testada: a execução não usa a versão e ela volta a exigir revalidação.
	if _, err := st.Resolve(ctx, p.ID, "sha9", func(envconfig.Config, string) error { return errors.New("docker-compose.yml mudou") }); err == nil {
		t.Fatal("preflight falho deve recusar")
	}
	view, _ = st.View(ctx, p.ID)
	if view.Active.State != envconfig.StateRevalidation {
		t.Fatalf("%+v", view.Active)
	}
	if _, err := (&envconfig.Store{DB: db.Pool}).Resolve(ctx, mustProject(t, db.Pool, "envcfg2").ID, "x", nil); !errors.Is(err, envconfig.ErrNoActive) {
		t.Fatalf("%v", err)
	}
}
