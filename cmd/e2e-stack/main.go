// Comando e2e-stack: sobe, para o ensaio com navegador real (`make e2e`), uma API descartável
// sobre um banco temporário, com um GitHub de mentira e dados de ensaio. Escreve UMA linha JSON
// com os endereços e a credencial efêmera do administrador quando está pronto e remove o banco ao
// receber SIGINT/SIGTERM. Não é um serviço do produto: nada aqui roda em produção.
//
// Uso: TEST_DATABASE_URL=<banco administrativo> go run ./cmd/e2e-stack
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/config"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/gitops"
	"github.com/ViniciusBernucci/le-fabrique/internal/httpapi"
	"github.com/ViniciusBernucci/le-fabrique/internal/memory"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/planning"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

const ghToken = "e2e-token-descartavel"

func main() { os.Exit(run()) }

func run() int {
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		fmt.Fprintln(os.Stderr, "e2e-stack: defina TEST_DATABASE_URL (banco administrativo)")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	admin, err := persistence.Connect(ctx, adminURL, 10*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: banco administrativo: "+err.Error())
		return 1
	}
	defer admin.Close()
	suffix := make([]byte, 6)
	_, _ = rand.Read(suffix)
	name := "lf_e2e_" + hex.EncodeToString(suffix)
	if _, err := admin.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: criar banco: "+err.Error())
		return 1
	}
	defer func() {
		_, _ = admin.Exec(context.Background(), `DROP DATABASE IF EXISTS `+name+` WITH (FORCE)`)
	}()
	u, _ := url.Parse(adminURL)
	u.Path = "/" + name
	m, err := persistence.NewMigrator(u.String())
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: "+err.Error())
		return 1
	}
	_, err = m.Up(ctx)
	m.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: migrations: "+err.Error())
		return 1
	}
	pool, err := persistence.Connect(ctx, u.String(), 10*time.Second)
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: "+err.Error())
		return 1
	}
	defer pool.Close()

	gh := newFakeGitHub()
	ghSrv := &http.Server{Handler: gh, ReadHeaderTimeout: 5 * time.Second}
	ghLn, _ := net.Listen("tcp", "127.0.0.1:0")
	go func() { _ = ghSrv.Serve(ghLn) }()
	defer ghSrv.Close()

	pw := make([]byte, 12)
	_, _ = rand.Read(pw)
	password := "Ensaio-" + hex.EncodeToString(pw)
	cfg := config.API{Common: config.Common{Env: config.Development, LogLevel: "error"}, Addr: "127.0.0.1:0", SessionTTL: time.Hour, SessionCookieName: "lf_session",
		VaultKey: []byte("0123456789abcdef0123456789abcdef"), VaultKeyID: "k1"}
	srv, err := httpapi.NewServer(cfg, pool, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: "+err.Error())
		return 1
	}
	srv.UseGitHubBase("http://" + ghLn.Addr().String())
	if err := srv.Prime(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: "+err.Error())
		return 1
	}
	if _, err := security.NewAuth(pool, nil, security.AuthConfig{}, nil).CreateAdmin(ctx, "ensaio", password); err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: administrador: "+err.Error())
		return 1
	}
	if err := seed(ctx, pool, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "e2e-stack: dados de ensaio: "+err.Error())
		return 1
	}
	apiLn, _ := net.Listen("tcp", "127.0.0.1:0")
	apiSrv := &http.Server{Handler: srv.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = apiSrv.Serve(apiLn) }()
	defer apiSrv.Close()

	out, _ := json.Marshal(map[string]string{"api": "http://" + apiLn.Addr().String(), "github": "http://" + ghLn.Addr().String(), "user": "ensaio", "password": password})
	fmt.Println(string(out))
	<-ctx.Done()
	return 0
}

// ---- GitHub de mentira ---------------------------------------------------------------

type fakeGitHub struct {
	mu    sync.Mutex
	state map[int]string // open | closed
	merge map[int]string // sha
}

func newFakeGitHub() *fakeGitHub {
	return &fakeGitHub{state: map[int]string{12: "open", 13: "closed", 14: "open"}, merge: map[int]string{}}
}

func (g *fakeGitHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if strings.HasPrefix(r.URL.Path, "/__") { // controle do ensaio
		var n int
		switch {
		case r.URL.Path == "/__merge/12":
			n = 12
			g.state[n], g.merge[n] = "closed", "c0ffee0123456789c0ffee0123456789c0ffee01"
		case r.URL.Path == "/__reset":
			g.state, g.merge = map[int]string{12: "open", 13: "closed", 14: "open"}, map[int]string{}
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+ghToken {
		w.WriteHeader(401)
		return
	}
	pr := func(n int) map[string]any {
		return map[string]any{"number": n, "html_url": fmt.Sprintf("https://github.com/org/app/pull/%d", n), "draft": g.state[n] == "open", "state": g.state[n],
			"merged": g.merge[n] != "", "merge_commit_sha": g.merge[n], "title": "Feature", "head": map[string]any{"ref": "lf/x"}, "base": map[string]any{"ref": "main"}}
	}
	var n int
	switch {
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls/") && strings.HasSuffix(r.URL.Path, "/comments"):
		_, _ = w.Write([]byte(`[]`)) // sem comentários de linha de revisão
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments"):
		_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 501, "body": "documente a coluna removido_em", "user": map[string]any{"login": "revisora"}},
			{"id": 502, "body": "intruso pedindo algo", "user": map[string]any{"login": "intruso"}}})
	case r.Method == http.MethodGet && sscan(r.URL.Path, "/repos/org/app/pulls/%d", &n):
		_ = json.NewEncoder(w).Encode(pr(n))
	case r.Method == http.MethodPatch && sscan(r.URL.Path, "/repos/org/app/pulls/%d", &n):
		if g.merge[n] == "" {
			g.state[n] = "open"
		}
		_ = json.NewEncoder(w).Encode(pr(n))
	default:
		w.WriteHeader(404)
	}
}

func sscan(path, format string, n *int) bool {
	_, err := fmt.Sscanf(path, format, n)
	return err == nil
}

// ---- dados de ensaio -----------------------------------------------------------------

func seed(ctx context.Context, pool *pgxpool.Pool, cfg config.API) error {
	exec := func(sql string, args ...any) error {
		_, err := pool.Exec(ctx, sql, args...)
		return persistence.MapError(err)
	}
	redactor := security.NewRedactor()
	sealer, err := security.NewSealer(cfg.VaultKey, cfg.VaultKeyID)
	if err != nil {
		return err
	}
	conn, err := security.NewVault(pool, sealer, redactor).Create(ctx, security.NewConnection{Kind: "github", Name: "github-ensaio", Target: "org/app", Secret: security.NewSecret(ghToken)})
	if err != nil {
		return err
	}
	p, err := projects.Create(ctx, pool, projects.NewProject{Slug: "ensaio", Name: "Projeto de ensaio"})
	if err != nil {
		return err
	}
	if err := exec(`UPDATE projects SET state = 'active', repo_owner = 'org', repo_name = 'app', base_branch = 'main', repo_connection_id = $2 WHERE id = $1`, p.ID, conn.ID); err != nil {
		return err
	}
	plan := func(keys ...string) planning.Plan {
		var ts []planning.Task
		for _, k := range keys {
			ts = append(ts, planning.Task{Key: k, Title: "Tarefa " + k, Description: "Faz " + k, Risk: "medium",
				Criteria: []planning.Criterion{{Text: "Os testes de " + k + " passam", Check: planning.CheckTest, Target: "./internal/" + k}}})
		}
		return planning.Plan{Summary: "ensaio", Tasks: ts}
	}
	plans := &planning.Store{DB: pool}
	qs := &queue.Queue{DB: pool}
	feat := func(ext, title string, keys ...string) (string, error) {
		f, err := persistence.CreateFeature(ctx, pool, p.ID, ext, title)
		if err != nil {
			return "", err
		}
		if _, err := plans.Propose(ctx, p.ID, f.ID, plan(keys...)); err != nil {
			return "", err
		}
		return f.ID, nil
	}
	sha := strings.Repeat("b", 40)
	f7, err := feat("PED-7", "Pedidos com soft delete", "schema", "api")
	if err != nil {
		return err
	}
	f8, err := feat("PED-8", "Relatório de pedidos", "relatorio")
	if err != nil {
		return err
	}
	f9, err := feat("PED-9", "Exportação de pedidos", "exportar")
	if err != nil {
		return err
	}
	if _, err := qs.EnqueuePlan(ctx, p.ID, f7, 2); err != nil {
		return err
	}
	b7, _ := gitops.FeatureBranch(f7, 1)
	b8, _ := gitops.FeatureBranch(f8, 1)
	b9, _ := gitops.FeatureBranch(f9, 1)
	for _, d := range []struct {
		feature, branch, state string
		pr                     int
	}{{f7, b7, "draft_open", 12}, {f8, b8, "closed", 13}, {f9, b9, "draft_open", 14}} {
		if err := exec(`INSERT INTO deliveries (project_id, feature_id, attempt, branch, revision, state, pr_number, pr_url) VALUES ($1, $2, 1, $3, $4, $5, $6, $7)`,
			p.ID, d.feature, d.branch, sha, d.state, d.pr, fmt.Sprintf("https://github.com/org/app/pull/%d", d.pr)); err != nil {
			return err
		}
	}
	mem := &memory.Store{DB: pool, Redactor: redactor}
	if _, _, err := mem.Record(ctx, memory.Input{ProjectID: p.ID, FeatureID: f7, Branch: b7, Key: "pedidos usam soft delete", Kind: "fact",
		Content: "a tabela de pedidos usa a coluna removido_em", Source: "tarefa schema", Revision: sha, Confidence: "high",
		Evidence: []memory.Evidence{{Label: "check", Ref: "make test"}}}); err != nil {
		return err
	}
	// Decisão sensível e autorização, em execuções que aguardam.
	waiting := func(ext string) (string, error) {
		f, err := persistence.CreateFeature(ctx, pool, p.ID, ext, "Feature "+ext)
		if err != nil {
			return "", err
		}
		ex, err := persistence.CreateExecution(ctx, pool, p.ID, f.ID)
		if err != nil {
			return "", err
		}
		ex, err = persistence.UpdateExecutionState(ctx, pool, p.ID, ex.ID, ex.Version, "waiting")
		return ex.ID, err
	}
	exD, err := waiting("DEC-1")
	if err != nil {
		return err
	}
	if _, _, err := decisions.Create(ctx, pool, redactor, decisions.NewRequest{ProjectID: p.ID, ExecutionID: exD, NodeID: "git", Kind: decisions.KindSensitiveOperation,
		Title: "Publicar a branch da feature", Problem: "A feature PED-7 quer fazer push da branch lf/ped-7-a1.", Impact: "Sem a decisão a feature fica parada.",
		Evidence: []decisions.Evidence{{Label: "Branch", Text: b7}},
		Options:  []decisions.Option{{ID: "approve", Label: "Aprovar", Description: "autoriza o push"}, {ID: "refuse", Label: "Recusar", Description: "não publica"}}}); err != nil {
		return err
	}
	exA, err := waiting("AUT-1")
	if err != nil {
		return err
	}
	_, err = (&authorizations.Store{DB: pool}).Create(ctx, authorizations.NewAuthorization{ProjectID: p.ID, ExecutionID: exA, NodeID: "git",
		Action: policy.Action{Tool: policy.ToolGit, Operation: "push", Resource: "refs/heads/" + b7, Target: "github.com", Params: map[string]any{"branch": b7}}, MaxUses: 2, TTL: time.Hour})
	return err
}
