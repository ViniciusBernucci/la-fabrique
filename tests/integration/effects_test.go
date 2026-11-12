package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

// fakeRemote simula um destino sem chave de idempotência: guarda quantas vezes
// cada efeito aconteceu de verdade, independente do que o chamador soube.
type fakeRemote struct {
	mu          sync.Mutex
	effects     map[string]int
	applyCalls  int
	verifyCalls int
	delay       time.Duration
	// apply decide, a cada chamada, se o efeito acontece e qual erro volta.
	apply  func(call int) (lands bool, err error)
	verify func(key string, landed int) (effects.Verdict, error)
}

func newRemote() *fakeRemote { return &fakeRemote{effects: map[string]int{}} }

func (f *fakeRemote) Apply(_ context.Context, op effects.Op) (any, error) {
	f.mu.Lock()
	f.applyCalls++
	call := f.applyCalls
	f.mu.Unlock()
	time.Sleep(f.delay)
	lands, err := true, error(nil)
	if f.apply != nil {
		lands, err = f.apply(call)
	}
	f.mu.Lock()
	if lands {
		f.effects[op.Key]++
	}
	f.mu.Unlock()
	return map[string]any{"remote_id": "R-" + op.Key}, err
}

func (f *fakeRemote) Verify(_ context.Context, op effects.Op) (effects.Verdict, error) {
	f.mu.Lock()
	f.verifyCalls++
	landed := f.effects[op.Key]
	f.mu.Unlock()
	if f.verify != nil {
		return f.verify(op.Key, landed)
	}
	if landed > 0 {
		return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"found": landed}}, nil
	}
	return effects.Verdict{Status: effects.VerdictAbsent}, nil
}

func (f *fakeRemote) count(key string) int { f.mu.Lock(); defer f.mu.Unlock(); return f.effects[key] }

var errTimeout = errors.New("timeout aguardando a resposta")

func spec(project, key string) effects.Spec {
	return effects.Spec{ProjectID: project, BusinessKey: key, Operation: "create_subtask", Request: map[string]any{"parent": "ABC-1", "title": "Subtarefa"}}
}

func ledgerFor(e *authEnv) *effects.Ledger {
	return &effects.Ledger{DB: e.db.Pool, Lease: time.Minute, MaxAttempts: 3}
}

// T13-04 — "Timeout após efeito confirmado não duplica."
func TestTimeoutAfterTheEffectLandedDoesNotDuplicate(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "efeito-ok")
	rm := newRemote()
	rm.apply = func(int) (bool, error) { return true, errTimeout } // o efeito aconteceu, a resposta se perdeu
	l := ledgerFor(e)

	res, err := l.Run(t.Context(), spec(p.ID, "feat1:subtarefa:1"), rm)
	if err != nil || res.Outcome != effects.OutcomeReconciled || res.Entry.State != "confirmed" {
		t.Fatalf("%+v %v", res, err)
	}
	if rm.count("feat1:subtarefa:1") != 1 || rm.applyCalls != 1 || rm.verifyCalls != 1 {
		t.Fatalf("o efeito não pode ser repetido: efeitos=%d envios=%d verificações=%d", rm.count("feat1:subtarefa:1"), rm.applyCalls, rm.verifyCalls)
	}
	var proof map[string]any
	_ = json.Unmarshal(res.Entry.Proof, &proof)
	if proof["found"] != float64(1) {
		t.Fatalf("a prova encontrada fica registrada: %s", res.Entry.Proof)
	}
	// Repetir a operação (nova tentativa, reinício) não reenvia nada.
	again, err := l.Run(t.Context(), spec(p.ID, "feat1:subtarefa:1"), rm)
	if err != nil || again.Outcome != effects.OutcomeAlreadyConfirmed || rm.applyCalls != 1 || rm.verifyCalls != 1 || rm.count("feat1:subtarefa:1") != 1 {
		t.Fatalf("já confirmado: %+v %v envios=%d", again, err, rm.applyCalls)
	}
}

// T13-04 — "Ausência segura permite repetir."
func TestProvenAbsenceAllowsASafeRetry(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "efeito-ausente")
	rm := newRemote()
	rm.apply = func(call int) (bool, error) {
		if call == 1 {
			return false, errTimeout // a primeira nem chegou ao destino
		}
		return true, nil
	}
	res, err := ledgerFor(e).Run(t.Context(), spec(p.ID, "k1"), rm)
	if err != nil || res.Outcome != effects.OutcomeApplied || rm.count("k1") != 1 || rm.applyCalls != 2 || res.Entry.Attempts != 2 {
		t.Fatalf("%+v %v efeitos=%d envios=%d", res, err, rm.count("k1"), rm.applyCalls)
	}

	// Ausência persistente: para depois do limite de tentativas, sem efeito, e continua repetível.
	down := newRemote()
	down.apply = func(int) (bool, error) { return false, errTimeout }
	res, err = ledgerFor(e).Run(t.Context(), spec(p.ID, "k2"), down)
	if err == nil || res.Outcome != effects.OutcomeFailed || res.Entry.State != "failed" || down.applyCalls != 3 || down.count("k2") != 0 {
		t.Fatalf("%+v %v envios=%d", res, err, down.applyCalls)
	}
	healthy := newRemote()
	res, err = ledgerFor(e).Run(t.Context(), spec(p.ID, "k2"), healthy)
	if err != nil || res.Outcome != effects.OutcomeApplied || healthy.count("k2") != 1 {
		t.Fatalf("depois de falhar, a operação pode ser repetida: %+v %v", res, err)
	}
}

func TestDefinitiveRefusalFailsWithoutVerifying(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "efeito-recusado")
	rm := newRemote()
	rm.apply = func(int) (bool, error) { return false, fmt.Errorf("403 sem permissão: %w", effects.ErrNotApplied) }
	res, err := ledgerFor(e).Run(t.Context(), spec(p.ID, "k"), rm)
	if !errors.Is(err, effects.ErrNotApplied) || res.Outcome != effects.OutcomeFailed || rm.verifyCalls != 0 || rm.applyCalls != 1 {
		t.Fatalf("%+v %v verificações=%d", res, err, rm.verifyCalls)
	}
}

// T13-04 — "Estado inconclusivo abre bloqueio com evidência."
func TestInconclusiveStateBlocksAndOpensADecisionWithEvidence(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "efeito-incerto")
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	rm := newRemote()
	rm.apply = func(int) (bool, error) { return true, errTimeout } // aconteceu, mas ninguém sabe
	rm.verify = func(string, int) (effects.Verdict, error) {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "a busca de subtarefas do pai ABC-1 devolveu erro 502"}, nil
	}
	l := ledgerFor(e)
	l.Escalate = effects.DecisionEscalator(e.db.Pool, nil, ex.ID)

	res, err := l.Run(ctx, spec(p.ID, "k-incerto"), rm)
	if !errors.Is(err, effects.ErrBlocked) || res.Entry.State != "uncertain" {
		t.Fatalf("%+v %v", res, err)
	}
	// Repetir automaticamente continua bloqueado: nenhum novo envio.
	for i := 0; i < 3; i++ {
		if _, err := l.Run(ctx, spec(p.ID, "k-incerto"), rm); !errors.Is(err, effects.ErrBlocked) {
			t.Fatalf("deveria seguir bloqueado: %v", err)
		}
	}
	if rm.applyCalls != 1 || rm.count("k-incerto") != 1 {
		t.Fatalf("o bloqueio impede duplicar: envios=%d efeitos=%d", rm.applyCalls, rm.count("k-incerto"))
	}

	// Um pedido na Central, com a evidência.
	var list struct{ Items []decisions.Request }
	c.expect(c.get("/api/decisions?state=pending"), 200).JSON(&list)
	if len(list.Items) != 1 {
		t.Fatalf("pedidos: %d (repetir não cria outro)", len(list.Items))
	}
	d := list.Items[0]
	body, _ := json.Marshal(d)
	if d.Kind != "uncertain_result" || !strings.HasPrefix(d.NodeID, "effect:") || !strings.Contains(string(body), "erro 502") || !strings.Contains(string(body), "k-incerto") {
		t.Fatalf("evidência do pedido: %s", body)
	}

	// A pessoa confirma que o efeito existe: o ledger fecha como confirmado, sem reenviar.
	c.expect(c.post("/api/decisions/"+d.ID+"/resolve", map[string]any{"option": "applied", "note": "Conferi no Jira"}), 200)
	entry, err := persistence.GetOutbox(ctx, e.db.Pool, p.ID, res.Entry.ID)
	if err != nil || entry.State != "confirmed" {
		t.Fatalf("%+v %v", entry, err)
	}
	var proof map[string]any
	_ = json.Unmarshal(entry.Proof, &proof)
	if proof["manual"] != true || proof["resolution"] != "applied" {
		t.Fatalf("prova manual: %s", entry.Proof)
	}
	if again, err := l.Run(ctx, spec(p.ID, "k-incerto"), rm); err != nil || again.Outcome != effects.OutcomeAlreadyConfirmed || rm.applyCalls != 1 {
		t.Fatalf("depois de confirmado: %+v %v", again, err)
	}
	// Repetir a decisão é inofensivo.
	c.expect(c.post("/api/decisions/"+d.ID+"/resolve", map[string]any{"option": "applied"}), 200)
}

func TestHumanSaysTheEffectDoesNotExistAndTheRetryIsAllowed(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "efeito-nao-existe")
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	rm := newRemote()
	rm.apply = func(call int) (bool, error) {
		if call == 1 {
			return false, errTimeout
		}
		return true, nil
	}
	rm.verify = func(string, int) (effects.Verdict, error) {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "o destino não respondeu à consulta"}, nil
	}
	l := ledgerFor(e)
	l.Escalate = effects.DecisionEscalator(e.db.Pool, nil, ex.ID)
	if _, err := l.Run(ctx, spec(p.ID, "k"), rm); !errors.Is(err, effects.ErrBlocked) {
		t.Fatal(err)
	}
	var list struct{ Items []decisions.Request }
	c.expect(c.get("/api/decisions?state=pending"), 200).JSON(&list)
	c.expect(c.post("/api/decisions/"+list.Items[0].ID+"/resolve", map[string]any{"option": "not_applied"}), 200)
	rm.verify = nil
	res, err := l.Run(ctx, spec(p.ID, "k"), rm)
	if err != nil || res.Outcome != effects.OutcomeApplied || rm.count("k") != 1 {
		t.Fatalf("liberada a repetição: %+v %v", res, err)
	}
}

func TestVerificationErrorCountsAsInconclusiveNotAsAbsent(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "efeito-verif-erro")
	rm := newRemote()
	rm.apply = func(int) (bool, error) { return true, errTimeout }
	rm.verify = func(string, int) (effects.Verdict, error) { return effects.Verdict{}, errors.New("conexão recusada") }
	if _, err := ledgerFor(e).Run(t.Context(), spec(p.ID, "k"), rm); !errors.Is(err, effects.ErrBlocked) || rm.applyCalls != 1 {
		t.Fatalf("falha na verificação não prova ausência: %v envios=%d", err, rm.applyCalls)
	}
}

func TestConcurrentRunsOfTheSameOperationSendOnce(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "efeito-concorrente")
	rm := newRemote()
	rm.delay = 150 * time.Millisecond
	l := ledgerFor(e)
	const n = 8
	var wg sync.WaitGroup
	results := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := l.Run(context.Background(), spec(p.ID, "k-conc"), rm)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	ok, inflight := 0, 0
	for err := range results {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, effects.ErrInFlight):
			inflight++
		default:
			t.Fatalf("erro inesperado: %v", err)
		}
	}
	if rm.applyCalls != 1 || rm.count("k-conc") != 1 || ok < 1 || ok+inflight != n {
		t.Fatalf("um único envio: envios=%d efeitos=%d ok=%d em-andamento=%d", rm.applyCalls, rm.count("k-conc"), ok, inflight)
	}
}

func TestAbandonedSendingIsReconciledBeforeAnyRetry(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "efeito-queda")
	age := func(key string) {
		if _, err := e.db.Pool.Exec(ctx, `UPDATE outbox SET updated_at = now() - interval '10 minutes' WHERE project_id = $1 AND business_key = $2`, p.ID, key); err != nil {
			t.Fatal(err)
		}
	}
	crash := func(key string) {
		entry, _, _ := persistence.EnqueueOutbox(ctx, e.db.Pool, p.ID, key, "create_subtask", map[string]any{})
		if _, err := persistence.TransitionOutbox(ctx, e.db.Pool, p.ID, entry.ID, entry.Version, "sending", nil); err != nil {
			t.Fatal(err)
		}
	}
	l := ledgerFor(e)

	// Dentro da concessão: outro processo pode estar enviando; nem verifica nem envia.
	rm := newRemote()
	crash("k-vivo")
	if _, err := l.Run(ctx, spec(p.ID, "k-vivo"), rm); !errors.Is(err, effects.ErrInFlight) || rm.applyCalls != 0 || rm.verifyCalls != 0 {
		t.Fatalf("envio em andamento: %v %d %d", err, rm.applyCalls, rm.verifyCalls)
	}
	// Queda depois do efeito: a prova confirma e nada é reenviado.
	crash("k-caiu-depois")
	age("k-caiu-depois")
	rm.effects["k-caiu-depois"] = 1
	res, err := l.Run(ctx, spec(p.ID, "k-caiu-depois"), rm)
	if err != nil || res.Outcome != effects.OutcomeReconciled || rm.applyCalls != 0 || rm.count("k-caiu-depois") != 1 {
		t.Fatalf("%+v %v envios=%d", res, err, rm.applyCalls)
	}
	// Queda antes do efeito: a ausência é comprovada e o envio é refeito uma vez.
	crash("k-caiu-antes")
	age("k-caiu-antes")
	res, err = l.Run(ctx, spec(p.ID, "k-caiu-antes"), rm)
	if err != nil || res.Outcome != effects.OutcomeApplied || rm.applyCalls != 1 || rm.count("k-caiu-antes") != 1 {
		t.Fatalf("%+v %v envios=%d", res, err, rm.applyCalls)
	}
	// Queda com prova inconclusiva: bloqueia.
	crash("k-caiu-duvida")
	age("k-caiu-duvida")
	rm.verify = func(string, int) (effects.Verdict, error) {
		return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: "sem resposta"}, nil
	}
	if _, err := l.Run(ctx, spec(p.ID, "k-caiu-duvida"), rm); !errors.Is(err, effects.ErrBlocked) || rm.applyCalls != 1 {
		t.Fatalf("%v envios=%d", err, rm.applyCalls)
	}
}

func TestIntentIsRecordedBeforeSendingAndRequestsAreRedacted(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	ctx := context.Background()
	p1, p2 := mustProject(t, e.db.Pool, "efeito-a"), mustProject(t, e.db.Pool, "efeito-b")
	sentinel := "SENTINELA-EFEITO-5b1c-nao-pode-vazar"
	rd := security.NewRedactor()
	rd.Register(sentinel)
	l := &effects.Ledger{DB: e.db.Pool, Redactor: rd}

	// A intenção já existe (com o pedido redigido) quando o adaptador é chamado.
	var seen persistence.OutboxEntry
	spy := &spyRemote{fakeRemote: newRemote(), onApply: func() {
		seen, _ = persistence.GetOutbox(ctx, e.db.Pool, p1.ID, mustEntryID(t, e, p1.ID, "k"))
	}}
	s := spec(p1.ID, "k")
	s.Request = map[string]any{"title": "Subtarefa", "nota": "usando " + sentinel}
	if _, err := l.Run(ctx, s, spy); err != nil {
		t.Fatal(err)
	}
	if seen.State != "sending" || seen.Attempts != 1 {
		t.Fatalf("a intenção deve estar gravada antes do envio: %+v", seen)
	}
	var found int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM outbox o WHERE o::text LIKE '%'||$1||'%'`, sentinel).Scan(&found)
	if found != 0 {
		t.Fatal("o pedido do efeito vazou segredo para o ledger")
	}
	// A mesma chave de negócio em outro projeto é outro efeito.
	rm := newRemote()
	if res, err := l.Run(ctx, spec(p2.ID, "k"), rm); err != nil || res.Outcome != effects.OutcomeApplied {
		t.Fatalf("%+v %v", res, err)
	}
	// Entradas inválidas.
	for _, bad := range []effects.Spec{{BusinessKey: "k", Operation: "x"}, {ProjectID: p1.ID, Operation: "x"}, {ProjectID: p1.ID, BusinessKey: "k"}} {
		if _, err := l.Run(ctx, bad, rm); err == nil {
			t.Errorf("%+v deveria ser recusada", bad)
		}
	}
}

type spyRemote struct {
	*fakeRemote
	onApply func()
}

func (s *spyRemote) Apply(ctx context.Context, op effects.Op) (any, error) {
	s.onApply()
	return s.fakeRemote.Apply(ctx, op)
}

func mustEntryID(t *testing.T, e *authEnv, project, key string) string {
	t.Helper()
	var id string
	if err := e.db.Pool.QueryRow(context.Background(), `SELECT id FROM outbox WHERE project_id = $1 AND business_key = $2`, project, key).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}
