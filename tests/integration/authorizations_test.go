package integration

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/authorizations"
	"github.com/ViniciusBernucci/le-fabrique/internal/checkpoints"
	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/execution"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/policy"
)

type authEnvT struct {
	e     *authEnv
	store *authorizations.Store
	proj  string
	exec  string
	admin string
	now   *time.Time
	mu    sync.Mutex
}

func newAuthzEnv(t *testing.T) *authEnvT {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "autorizacoes")
	ex := waitingExecution(t, e, p.ID, "AZ-1")
	var admin string
	if err := e.db.Pool.QueryRow(context.Background(), `SELECT id FROM admins LIMIT 1`).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	a := &authEnvT{e: e, proj: p.ID, exec: ex.ID, admin: admin}
	start := time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)
	a.now = &start
	a.store = &authorizations.Store{DB: e.db.Pool, Now: func() time.Time { a.mu.Lock(); defer a.mu.Unlock(); return *a.now }}
	return a
}

func (a *authEnvT) advance(d time.Duration) { a.mu.Lock(); *a.now = a.now.Add(d); a.mu.Unlock() }

func pushAction() policy.Action {
	return policy.Action{Tool: policy.ToolGit, Operation: "push", Resource: "refs/heads/lf/f-1-a1", Target: "github.com",
		Params: map[string]any{"branch": "lf/f-1-a1", "sha": "abc123", "force": false, "opts": []any{"a", map[string]any{"b": 1, "c": 2}}}}
}

func (a *authEnvT) grant(t *testing.T, act policy.Action, uses int, ttl time.Duration) authorizations.Authorization {
	t.Helper()
	au, err := a.store.Create(context.Background(), authorizations.NewAuthorization{ProjectID: a.proj, ExecutionID: a.exec, NodeID: "dev", Action: act, MaxUses: uses, TTL: ttl, GrantedBy: a.admin})
	if err != nil {
		t.Fatal(err)
	}
	return au
}

func (a *authEnvT) uses(t *testing.T, id string) int {
	t.Helper()
	au, err := a.store.Get(context.Background(), a.proj, id)
	if err != nil {
		t.Fatal(err)
	}
	return au.Uses
}

// T13-03 — "Corrida de último uso autoriza um."
func TestRaceForTheLastUseAuthorizesExactlyOne(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	for _, c := range []struct{ max, workers int }{{1, 30}, {5, 50}} {
		au := a.grant(t, pushAction(), c.max, time.Hour)
		var ok, exhausted, other atomic.Int64
		var wg sync.WaitGroup
		start := make(chan struct{})
		for i := 0; i < c.workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				_, err := a.store.Consume(ctx, a.proj, au.ID, pushAction())
				switch {
				case err == nil:
					ok.Add(1)
				case errors.Is(err, authorizations.ErrExhausted):
					exhausted.Add(1)
				default:
					other.Add(1)
					t.Errorf("%v", err)
				}
			}()
		}
		close(start)
		wg.Wait()
		if int(ok.Load()) != c.max || int(exhausted.Load()) != c.workers-c.max || other.Load() != 0 || a.uses(t, au.ID) != c.max {
			t.Fatalf("max=%d: ok=%d esgotadas=%d usos=%d", c.max, ok.Load(), exhausted.Load(), a.uses(t, au.ID))
		}
	}
}

// T13-03 — argumento, destino, recurso e ferramenta divergentes impedem a operação.
func TestDivergentActionIsRefusedWithoutSpendingAUse(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	au := a.grant(t, pushAction(), 3, time.Hour)
	mut := func(f func(*policy.Action)) policy.Action {
		x := pushAction()
		x.Params = map[string]any{"branch": "lf/f-1-a1", "sha": "abc123", "force": false, "opts": []any{"a", map[string]any{"b": 1, "c": 2}}}
		f(&x)
		return x
	}
	for name, c := range map[string]struct {
		act   policy.Action
		field string
	}{
		"outro SHA":          {mut(func(x *policy.Action) { x.Params["sha"] = "def456" }), "params"},
		"força o push":       {mut(func(x *policy.Action) { x.Params["force"] = true }), "params"},
		"parâmetro a mais":   {mut(func(x *policy.Action) { x.Params["delete"] = true }), "params"},
		"parâmetro aninhado": {mut(func(x *policy.Action) { x.Params["opts"] = []any{"a", map[string]any{"b": 1, "c": 3}} }), "params"},
		"outro destino":      {mut(func(x *policy.Action) { x.Target = "gitlab.com" }), "target"},
		"destino interno":    {mut(func(x *policy.Action) { x.Target = "169.254.169.254" }), "target"},
		"outro recurso":      {mut(func(x *policy.Action) { x.Resource = "refs/heads/main" }), "resource"},
		"outra operação":     {mut(func(x *policy.Action) { x.Operation = "delete" }), "operation"},
		"outra ferramenta":   {mut(func(x *policy.Action) { x.Tool = policy.ToolScript }), "tool"},
	} {
		_, err := a.store.Consume(ctx, a.proj, au.ID, c.act)
		var mm *authorizations.MismatchError
		if !errors.As(err, &mm) || mm.Field != c.field {
			t.Errorf("%s: esperado divergência em %q, veio %v", name, c.field, err)
		}
	}
	if a.uses(t, au.ID) != 0 {
		t.Fatalf("nenhuma divergência pode gastar uso: %d", a.uses(t, au.ID))
	}
	// A ação exata (ordem de chaves e caixa do host não importam) passa.
	same := pushAction()
	same.Target = "GitHub.COM"
	if _, err := a.store.Consume(ctx, a.proj, au.ID, same); err != nil {
		t.Fatalf("a mesma ação em forma canônica diferente deve passar: %v", err)
	}
	evs, _ := a.store.Events(ctx, au.ID)
	refused, consumed := 0, 0
	for _, e := range evs {
		switch e.Type {
		case "refused":
			refused++
		case "consumed":
			consumed++
		}
	}
	if refused != 9 || consumed != 1 || evs[0].Type != "created" {
		t.Fatalf("a trilha guarda cada recusa: %d recusas, %d consumos", refused, consumed)
	}
	// Outro projeto não enxerga nem consome.
	if _, err := a.store.Consume(ctx, "00000000-0000-4000-8000-000000000000", au.ID, pushAction()); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("%v", err)
	}
}

// T13-03 — prazo expirado e autorização revogada impedem a operação.
func TestExpiredAndRevokedAuthorizationsBlockTheOperation(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	au := a.grant(t, pushAction(), 5, 10*time.Minute)
	if _, err := a.store.Consume(ctx, a.proj, au.ID, pushAction()); err != nil {
		t.Fatal(err)
	}
	a.advance(11 * time.Minute)
	if _, err := a.store.Consume(ctx, a.proj, au.ID, pushAction()); !errors.Is(err, authorizations.ErrExpired) {
		t.Fatalf("prazo vencido: %v", err)
	}
	if got, _ := a.store.Get(ctx, a.proj, au.ID); got.Remaining(a.store.Now()) != 0 {
		t.Fatalf("vencida não tem usos: %+v", got)
	}
	b := a.grant(t, pushAction(), 5, time.Hour)
	if _, err := a.store.Consume(ctx, a.proj, b.ID, pushAction()); err != nil {
		t.Fatal(err)
	}
	a.store.Halter = &haltRecorder{} // o uso consumido e não concluído está "em andamento"
	rev, err := a.store.Revoke(ctx, a.proj, b.ID, a.admin, "projeto suspenso")
	if err != nil || rev.Authorization.State != "revoked" || rev.AlreadyRevoked || rev.InFlight != 1 || !rev.Halted {
		t.Fatalf("%+v %v", rev, err)
	}
	if _, err := a.store.Consume(ctx, a.proj, b.ID, pushAction()); !errors.Is(err, authorizations.ErrRevoked) {
		t.Fatalf("revogada: %v", err)
	}
	if again, err := a.store.Revoke(ctx, a.proj, b.ID, a.admin, "de novo"); err != nil || !again.AlreadyRevoked {
		t.Fatalf("revogar de novo é inofensivo: %+v %v", again, err)
	}
	// Revogada e vencida: a revogação vale primeiro.
	a.advance(2 * time.Hour)
	if _, err := a.store.Consume(ctx, a.proj, b.ID, pushAction()); !errors.Is(err, authorizations.ErrRevoked) {
		t.Fatalf("%v", err)
	}
	// Validação do cadastro.
	for name, in := range map[string]authorizations.NewAuthorization{
		"sem ferramenta": {ProjectID: a.proj, ExecutionID: a.exec, Action: policy.Action{Operation: "x"}, MaxUses: 1, TTL: time.Hour},
		"sem operação":   {ProjectID: a.proj, ExecutionID: a.exec, Action: policy.Action{Tool: "git"}, MaxUses: 1, TTL: time.Hour},
		"usos zero":      {ProjectID: a.proj, ExecutionID: a.exec, Action: pushAction(), MaxUses: 0, TTL: time.Hour},
		"usos demais":    {ProjectID: a.proj, ExecutionID: a.exec, Action: pushAction(), MaxUses: 101, TTL: time.Hour},
		"sem prazo":      {ProjectID: a.proj, ExecutionID: a.exec, Action: pushAction(), MaxUses: 1},
		"prazo longo":    {ProjectID: a.proj, ExecutionID: a.exec, Action: pushAction(), MaxUses: 1, TTL: 25 * time.Hour},
		"sem execução":   {ProjectID: a.proj, Action: pushAction(), MaxUses: 1, TTL: time.Hour},
	} {
		if _, err := a.store.Create(ctx, in); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
}

func TestRevocationRacingWithConsumersNeverLetsAUseThrough(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	au := a.grant(t, pushAction(), 100, time.Hour)
	var revokedAt atomic.Int64 // marca (em ns) do retorno da revogação
	var late atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				started := time.Now().UnixNano()
				_, err := a.store.Consume(ctx, a.proj, au.ID, pushAction())
				if err == nil {
					if r := revokedAt.Load(); r != 0 && started > r {
						late.Add(1) // começou DEPOIS de a revogação voltar e passou: proibido
					}
				}
			}
		}()
	}
	time.Sleep(30 * time.Millisecond)
	if _, err := a.store.Revoke(ctx, a.proj, au.ID, a.admin, "corrida"); err != nil && !errors.Is(err, authorizations.ErrHaltFailed) {
		t.Fatal(err)
	}
	revokedAt.Store(time.Now().UnixNano())
	wg.Wait()
	if late.Load() != 0 {
		t.Fatalf("%d usos passaram depois da revogação", late.Load())
	}
	if _, err := a.store.Consume(ctx, a.proj, au.ID, pushAction()); !errors.Is(err, authorizations.ErrRevoked) {
		t.Fatalf("%v", err)
	}
}

type haltRecorder struct {
	calls atomic.Int64
	run   func(ctx context.Context) error
	err   error
}

func (h *haltRecorder) Halt(ctx context.Context, project, executionID string) error {
	h.calls.Add(1)
	if h.run != nil {
		return h.run(ctx)
	}
	return h.err
}

// T13-03 — "Revogar e disparar parada segura/checkpoint quando aplicável."
func TestRevokingWithAUseInFlightTriggersSafeStopAndCheckpoint(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	d := &execution.Docker{}
	if !d.Available(ctx) {
		t.Skip("Docker indisponível")
	}
	sb, err := d.Provision(ctx, execution.Spec{ProjectID: a.proj, FeatureID: "feat-az", Profile: execution.Profiles["base"]})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sb.Destroy(context.Background()) })
	if _, err := sb.Exec(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "echo trabalho > /workspace/progresso.txt"}}); err != nil {
		t.Fatal(err)
	}
	cks := &checkpoints.Store{DB: a.e.db.Pool}
	running := make(chan *execution.Running, 1)
	halter := &haltRecorder{}
	halter.run = func(ctx context.Context) error {
		r := <-running
		if err := r.Cancel(2 * time.Second); err != nil {
			return err
		}
		_, err := cks.CaptureAndRelease(ctx, sb, checkpoints.Input{ProjectID: a.proj, ExecutionID: a.exec, Label: "revogação"}, checkpoints.Options{})
		return err
	}
	a.store.Halter = halter
	au := a.grant(t, pushAction(), 2, time.Hour)

	// A operação autorizada roda no sandbox e fica em andamento.
	started, done := make(chan struct{}), make(chan error, 1)
	go func() {
		done <- a.store.Do(ctx, a.proj, au.ID, pushAction(), func(ctx context.Context) error {
			r, err := sb.Start(ctx, execution.ExecSpec{Cmd: []string{"sh", "-c", "sleep 300"}})
			if err != nil {
				return err
			}
			running <- r
			close(started)
			code, _ := r.Wait()
			if code != 0 {
				return errors.New("operação interrompida")
			}
			return nil
		})
	}()
	<-started
	if n, _ := a.store.InFlight(ctx, au.ID); n != 1 {
		t.Fatalf("uso em andamento: %d", n)
	}
	rev, err := a.store.Revoke(ctx, a.proj, au.ID, a.admin, "o responsável revogou")
	if err != nil || !rev.Halted || rev.InFlight != 1 || halter.calls.Load() != 1 {
		t.Fatalf("%+v %v chamadas=%d", rev, err, halter.calls.Load())
	}
	if err := <-done; err == nil {
		t.Fatal("a operação em andamento foi interrompida")
	}
	if n, _ := a.store.InFlight(ctx, au.ID); n != 0 {
		t.Fatalf("o uso foi encerrado como falho: %d", n)
	}
	cp, err := cks.Latest(ctx, a.proj, a.exec)
	if err != nil || cp.Label != "revogação" || sb.Exists(ctx) {
		t.Fatalf("checkpoint gravado e sandbox liberado: %+v %v", cp, err)
	}
	var paths []string
	for _, f := range cp.Manifest.Files {
		paths = append(paths, f.Path)
	}
	if len(paths) != 1 || paths[0] != "progresso.txt" {
		t.Fatalf("o progresso foi preservado: %v", paths)
	}
	if _, err := a.store.Consume(ctx, a.proj, au.ID, pushAction()); !errors.Is(err, authorizations.ErrRevoked) {
		t.Fatalf("%v", err)
	}
	evs, _ := a.store.Events(ctx, au.ID)
	types := []string{}
	for _, e := range evs {
		types = append(types, e.Type)
	}
	want := "created,consumed,revoked,halted,refused"
	got := ""
	for i, ty := range types {
		if i > 0 {
			got += ","
		}
		got += ty
	}
	if got != want {
		t.Fatalf("trilha: %s", got)
	}
}

func TestRevokingWithoutAUseInFlightDoesNotHaltAndAHaltFailureIsExplicit(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	halter := &haltRecorder{}
	a.store.Halter = halter
	// Sem uso em andamento (nem nunca usada): só revoga.
	idle := a.grant(t, pushAction(), 1, time.Hour)
	if rev, err := a.store.Revoke(ctx, a.proj, idle.ID, a.admin, ""); err != nil || rev.Halted || rev.InFlight != 0 || halter.calls.Load() != 0 {
		t.Fatalf("%+v %v", rev, err)
	}
	// Uso concluído também não dispara parada.
	done := a.grant(t, pushAction(), 2, time.Hour)
	if err := a.store.Do(ctx, a.proj, done.ID, pushAction(), func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if rev, err := a.store.Revoke(ctx, a.proj, done.ID, a.admin, ""); err != nil || rev.Halted || halter.calls.Load() != 0 {
		t.Fatalf("%+v %v", rev, err)
	}
	// Uso em andamento e a parada falha: a revogação vale e o erro é explícito.
	halter.err = errors.New("docker indisponível")
	stuck := a.grant(t, pushAction(), 2, time.Hour)
	if _, err := a.store.Consume(ctx, a.proj, stuck.ID, pushAction()); err != nil {
		t.Fatal(err)
	}
	rev, err := a.store.Revoke(ctx, a.proj, stuck.ID, a.admin, "")
	if !errors.Is(err, authorizations.ErrHaltFailed) || rev.Halted || rev.Authorization.State != "revoked" {
		t.Fatalf("%+v %v", rev, err)
	}
	if _, err := a.store.Consume(ctx, a.proj, stuck.ID, pushAction()); !errors.Is(err, authorizations.ErrRevoked) {
		t.Fatalf("a revogação vale mesmo com a parada falhando: %v", err)
	}
	// Sem executor de parada configurado, também é explícito.
	a.store.Halter = nil
	nohalt := a.grant(t, pushAction(), 2, time.Hour)
	_, _ = a.store.Consume(ctx, a.proj, nohalt.ID, pushAction())
	if _, err := a.store.Revoke(ctx, a.proj, nohalt.ID, a.admin, ""); !errors.Is(err, authorizations.ErrHaltFailed) {
		t.Fatalf("%v", err)
	}
	evs, _ := a.store.Events(ctx, stuck.ID)
	if evs[len(evs)-2].Type != "halt_failed" && evs[len(evs)-1].Type != "halt_failed" {
		t.Fatalf("a falha da parada fica na trilha: %+v", evs)
	}
}

func TestDoRunsTheOperationOnlyWhenAuthorizedAndDecisionLinkIsValidated(t *testing.T) {
	a := newAuthzEnv(t)
	ctx := context.Background()
	au := a.grant(t, pushAction(), 1, time.Hour)
	ran := 0
	op := func(context.Context) error { ran++; return nil }
	if err := a.store.Do(ctx, a.proj, au.ID, pushAction(), op); err != nil || ran != 1 {
		t.Fatalf("%v %d", err, ran)
	}
	if err := a.store.Do(ctx, a.proj, au.ID, pushAction(), op); !errors.Is(err, authorizations.ErrExhausted) || ran != 1 {
		t.Fatalf("a operação não roda sem uso: %v %d", err, ran)
	}
	bad := pushAction()
	bad.Params["sha"] = "outro"
	fresh := a.grant(t, pushAction(), 1, time.Hour)
	if err := a.store.Do(ctx, a.proj, fresh.ID, bad, op); err == nil || ran != 1 {
		t.Fatalf("ação divergente não roda: %v %d", err, ran)
	}
	// Vínculo com a decisão humana: precisa estar resolvida e ser da mesma execução.
	req, _, err := decisions.Create(ctx, a.e.db.Pool, nil, decisions.NewRequest{ProjectID: a.proj, ExecutionID: a.exec, NodeID: "dev", Kind: decisions.KindSensitiveOperation,
		Title: "Publicar a branch", Problem: "Push na branch da feature.", Options: []decisions.Option{{ID: "aprovar", Label: "Aprovar", Description: "Publica"}, {ID: "negar", Label: "Negar", Description: "Não publica"}}})
	if err != nil {
		t.Fatal(err)
	}
	link := authorizations.NewAuthorization{ProjectID: a.proj, ExecutionID: a.exec, DecisionID: req.ID, Action: pushAction(), MaxUses: 1, TTL: time.Hour}
	if _, err := a.store.Create(ctx, link); !errors.Is(err, authorizations.ErrDecision) {
		t.Fatalf("decisão pendente não autoriza: %v", err)
	}
	if _, err := decisions.Resolve(ctx, a.e.db.Pool, a.admin, req.ID, "aprovar", "ok"); err != nil {
		t.Fatal(err)
	}
	if got, err := a.store.Create(ctx, link); err != nil || got.DecisionID == nil || *got.DecisionID != req.ID {
		t.Fatalf("%+v %v", got, err)
	}
	other := waitingExecution(t, a.e, a.proj, "AZ-2")
	link.ExecutionID = other.ID
	if _, err := a.store.Create(ctx, link); !errors.Is(err, authorizations.ErrDecision) {
		t.Fatalf("decisão de outra execução: %v", err)
	}
	link.ExecutionID, link.DecisionID = a.exec, "00000000-0000-4000-8000-000000000000"
	if _, err := a.store.Create(ctx, link); !errors.Is(err, authorizations.ErrDecision) {
		t.Fatalf("decisão inexistente: %v", err)
	}
}
