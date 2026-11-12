package integration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/decisions"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
	"github.com/ViniciusBernucci/le-fabrique/internal/security"
)

func waitingExecution(t *testing.T, e *authEnv, projectID, ext string) persistence.Execution {
	t.Helper()
	ctx := context.Background()
	f, err := persistence.CreateFeature(ctx, e.db.Pool, projectID, ext, "Feature "+ext)
	if err != nil {
		t.Fatal(err)
	}
	ex, err := persistence.CreateExecution(ctx, e.db.Pool, projectID, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	ex, err = persistence.UpdateExecutionState(ctx, e.db.Pool, projectID, ex.ID, ex.Version, "waiting")
	if err != nil {
		t.Fatal(err)
	}
	return ex
}

func newRequest(project, execution, node, kind string) decisions.NewRequest {
	return decisions.NewRequest{
		ProjectID: project, ExecutionID: execution, NodeID: node, Kind: kind,
		Title: "Conflito de requisitos", Problem: "Duas tarefas pedem comportamentos incompatíveis.", Impact: "A feature fica parada até a decisão.",
		Evidence: []decisions.Evidence{{Label: "Tarefa A", Text: "Pede X"}},
		Options:  []decisions.Option{{ID: "seguir-a", Label: "Seguir A", Description: "Ignora B"}, {ID: "seguir-b", Label: "Seguir B", Description: "Ignora A"}},
	}
}

func execRow(t *testing.T, e *authEnv, id string) (state string, version int, events int) {
	t.Helper()
	ctx := context.Background()
	if err := e.db.Pool.QueryRow(ctx, `SELECT state, version FROM executions WHERE id = $1`, id).Scan(&state, &version); err != nil {
		t.Fatal(err)
	}
	if err := e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM execution_events WHERE execution_id = $1 AND type IN ('decision_resolved','resume_requested')`, id).Scan(&events); err != nil {
		t.Fatal(err)
	}
	return
}

func TestDecisionRequestCreationRules(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "decisoes")
	ex := waitingExecution(t, e, p.ID, "EXT-1")

	req, created, err := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "dev", decisions.KindConflict))
	if err != nil || !created || req.State != "pending" || req.ExecutionVersion != ex.Version {
		t.Fatalf("%+v %v %v", req, created, err)
	}
	// O mesmo problema pendente não vira dois pedidos.
	again, created, err := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "dev", decisions.KindConflict))
	if err != nil || created || again.ID != req.ID {
		t.Fatalf("duplicado: %+v %v %v", again, created, err)
	}
	// Outro nó ou outro tipo é outro pedido.
	if other, created, err := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "docs", decisions.KindConflict)); err != nil || !created || other.ID == req.ID {
		t.Fatalf("outro nó: %v %v", created, err)
	}
	bad := map[string]func(*decisions.NewRequest){
		"tipo":           func(r *decisions.NewRequest) { r.Kind = "magia" },
		"título":         func(r *decisions.NewRequest) { r.Title = " " },
		"problema":       func(r *decisions.NewRequest) { r.Problem = "" },
		"uma opção":      func(r *decisions.NewRequest) { r.Options = r.Options[:1] },
		"sete opções":    func(r *decisions.NewRequest) { r.Options = make([]decisions.Option, 7) },
		"opção repetida": func(r *decisions.NewRequest) { r.Options[1].ID = r.Options[0].ID },
		"id de opção":    func(r *decisions.NewRequest) { r.Options[0].ID = "Com Espaço" },
		"evidência":      func(r *decisions.NewRequest) { r.Evidence = make([]decisions.Evidence, 21) },
	}
	for name, mut := range bad {
		r := newRequest(p.ID, ex.ID, "x-"+name, decisions.KindOther)
		mut(&r)
		if _, _, err := decisions.Create(ctx, e.db.Pool, nil, r); err == nil {
			t.Errorf("%s deveria ser recusado", name)
		}
	}
	// A execução precisa existir, ser do projeto e estar aguardando.
	other := mustProject(t, e.db.Pool, "outro-projeto")
	if _, _, err := decisions.Create(ctx, e.db.Pool, nil, newRequest(other.ID, ex.ID, "dev", decisions.KindOther)); err == nil {
		t.Error("execução de outro projeto")
	}
	running, _ := persistence.UpdateExecutionState(ctx, e.db.Pool, p.ID, ex.ID, ex.Version, "running")
	if _, _, err := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, running.ID, "novo", decisions.KindOther)); err == nil {
		t.Error("execução que não está aguardando")
	}
}

func TestEvidenceIsRedactedBeforeItIsStored(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "evidencia")
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	sentinel := "SENTINELA-DECISAO-8d2f-nao-pode-vazar"
	rd := security.NewRedactor()
	rd.Register(sentinel)
	r := newRequest(p.ID, ex.ID, "dev", decisions.KindUncertainResult)
	r.Evidence = []decisions.Evidence{{Label: "Log", Text: "falhou com " + sentinel + " e Bearer abc.def-ghi_jkl123456"}}
	req, _, err := decisions.Create(ctx, e.db.Pool, rd, r)
	if err != nil {
		t.Fatal(err)
	}
	body := string(c.get("/api/decisions/" + req.ID).Body)
	var inDB int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM decision_requests d WHERE d::text LIKE '%'||$1||'%'`, sentinel).Scan(&inDB)
	if strings.Contains(body, sentinel) || strings.Contains(body, "abc.def-ghi") || inDB != 0 {
		t.Fatalf("a evidência vazou: %s (no banco: %d)", body, inDB)
	}
}

// T13-02 — "Resolver pendência por UI altera só execução afetada."
func TestResolvingChangesOnlyTheAffectedExecution(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "so-a")
	exA, exB := waitingExecution(t, e, p.ID, "EXT-A"), waitingExecution(t, e, p.ID, "EXT-B")
	reqA, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, exA.ID, "dev", decisions.KindConflict))
	reqB, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, exB.ID, "dev", decisions.KindConflict))
	_, vB, evB := execRow(t, e, exB.ID)

	var list struct{ Items []decisions.Request }
	c.expect(c.get("/api/decisions?state=pending"), 200).JSON(&list)
	if len(list.Items) != 2 {
		t.Fatalf("pendentes: %d", len(list.Items))
	}

	var done decisions.Request
	r := c.expect(c.post("/api/decisions/"+reqA.ID+"/resolve", map[string]any{"option": "seguir-a", "note": "Cliente confirmou"}), 200)
	r.JSON(&done)
	if done.State != "resolved" || done.ResolutionOption != "seguir-a" || done.ResolutionNote != "Cliente confirmou" || done.ResolvedAt == nil || r.Header.Get("X-Decision-Replayed") != "false" {
		t.Fatalf("%+v", done)
	}
	// A execução A recebe o sinal de retomada: eventos e estado.
	stateA, vA, evA := execRow(t, e, exA.ID)
	if stateA != "queued" || vA != exA.Version+1 || evA != 2 {
		t.Fatalf("execução A: %s v%d eventos %d", stateA, vA, evA)
	}
	// A execução B e o pedido B ficam exatamente como estavam.
	stateB, vB2, evB2 := execRow(t, e, exB.ID)
	if stateB != "waiting" || vB2 != vB || evB2 != evB {
		t.Fatalf("a execução B não pode mudar: %s v%d eventos %d", stateB, vB2, evB2)
	}
	var gotB decisions.Request
	c.expect(c.get("/api/decisions/"+reqB.ID), 200).JSON(&gotB)
	if gotB.State != "pending" || gotB.ResolutionOption != "" || gotB.Version != reqB.Version {
		t.Fatalf("pedido B: %+v", gotB)
	}
	// Histórico: resolvido de um lado, pendente do outro.
	c.expect(c.get("/api/decisions?state=resolved"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].ID != reqA.ID {
		t.Fatalf("resolvidos: %+v", list.Items)
	}
	c.expect(c.get("/api/decisions?state=pending"), 200).JSON(&list)
	if len(list.Items) != 1 || list.Items[0].ID != reqB.ID {
		t.Fatalf("pendentes: %+v", list.Items)
	}
	// Trilha de auditoria da decisão.
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM auth_events WHERE type = 'decision_resolved'`).Scan(&n)
	if n != 1 {
		t.Fatalf("auditoria: %d", n)
	}
}

// T13-02 — "Decisão repetida ... tratada de forma consistente."
func TestRepeatedDecisionIsIdempotentAndAConflictingOneIsRefused(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "repetida")
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	req, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "dev", decisions.KindConflict))
	c.expect(c.post("/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-a"}), 200)
	_, v1, ev1 := execRow(t, e, ex.ID)

	r := c.expect(c.post("/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-a"}), 200)
	if r.Header.Get("X-Decision-Replayed") != "true" {
		t.Fatal("a mesma decisão deve ser reconhecida como repetida")
	}
	if _, v2, ev2 := execRow(t, e, ex.ID); v2 != v1 || ev2 != ev1 {
		t.Fatalf("repetir não pode gerar novo sinal: v%d→v%d eventos %d→%d", v1, v2, ev1, ev2)
	}
	if r := c.post("/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-b"}); r.StatusCode != 409 || r.ErrCode() != "decision_conflict" {
		t.Fatalf("outra opção: %d %s", r.StatusCode, r.Body)
	}
	var got decisions.Request
	c.expect(c.get("/api/decisions/"+req.ID), 200).JSON(&got)
	if got.ResolutionOption != "seguir-a" {
		t.Fatalf("a decisão registrada não pode mudar: %+v", got)
	}
	var n int
	_ = e.db.Pool.QueryRow(ctx, `SELECT count(*) FROM auth_events WHERE type = 'decision_resolved'`).Scan(&n)
	if n != 1 {
		t.Fatalf("a repetição não gera nova auditoria: %d", n)
	}
}

func TestConcurrentResolutionsProduceExactlyOneResumeSignal(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "corrida-decisao")

	// Mesma opção em paralelo: todas dão certo, só uma muda algo.
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	req, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "dev", decisions.KindConflict))
	const n = 8
	var wg sync.WaitGroup
	codes := make(chan int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			codes <- c.post("/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-a"}).StatusCode
		}()
	}
	wg.Wait()
	close(codes)
	for code := range codes {
		if code != 200 {
			t.Fatalf("mesma opção em paralelo deve dar 200, veio %d", code)
		}
	}
	if state, v, ev := execRow(t, e, ex.ID); state != "queued" || v != ex.Version+1 || ev != 2 {
		t.Fatalf("um único sinal: %s v%d eventos %d", state, v, ev)
	}

	// Opções diferentes em paralelo: uma vence, as outras recebem conflito.
	ex2 := waitingExecution(t, e, p.ID, "EXT-2")
	req2, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex2.ID, "dev", decisions.KindConflict))
	results := make(chan [2]any, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		opt := []string{"seguir-a", "seguir-b"}[i%2]
		go func() {
			defer wg.Done()
			r := c.post("/api/decisions/"+req2.ID+"/resolve", map[string]any{"option": opt})
			results <- [2]any{opt, r.StatusCode}
		}()
	}
	wg.Wait()
	close(results)
	winners := map[any]bool{}
	for r := range results {
		if r[1] == 200 {
			winners[r[0]] = true
		} else if r[1] != 409 {
			t.Fatalf("esperado 200 ou 409: %v", r)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("só uma opção pode vencer: %v", winners)
	}
	if _, _, ev := execRow(t, e, ex2.ID); ev != 2 {
		t.Fatalf("um único sinal: %d", ev)
	}
}

// T13-02 — "... decisão sobre versão obsoleta é tratada de forma consistente."
func TestDecisionOnAnObsoleteExecutionIsRefusedAndMarkedStale(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "obsoleta")

	// A execução foi cancelada depois do pedido.
	exA := waitingExecution(t, e, p.ID, "EXT-A")
	reqA, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, exA.ID, "dev", decisions.KindConflict))
	if _, err := persistence.UpdateExecutionState(ctx, e.db.Pool, p.ID, exA.ID, exA.Version, "cancelled"); err != nil {
		t.Fatal(err)
	}
	if r := c.post("/api/decisions/"+reqA.ID+"/resolve", map[string]any{"option": "seguir-a"}); r.StatusCode != 409 || r.ErrCode() != "decision_stale" {
		t.Fatalf("cancelada: %d %s", r.StatusCode, r.Body)
	}
	// A execução mudou de versão (ainda aguardando, mas já não é a mesma situação).
	exB := waitingExecution(t, e, p.ID, "EXT-B")
	reqB, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, exB.ID, "dev", decisions.KindConflict))
	if _, err := persistence.UpdateExecutionState(ctx, e.db.Pool, p.ID, exB.ID, exB.Version, "waiting"); err != nil {
		t.Fatal(err)
	}
	if r := c.post("/api/decisions/"+reqB.ID+"/resolve", map[string]any{"option": "seguir-a"}); r.StatusCode != 409 || r.ErrCode() != "decision_stale" {
		t.Fatalf("versão mudou: %d %s", r.StatusCode, r.Body)
	}
	for _, id := range []string{reqA.ID, reqB.ID} {
		var got decisions.Request
		c.expect(c.get("/api/decisions/"+id), 200).JSON(&got)
		if got.State != "stale" || got.ResolutionOption != "" {
			t.Fatalf("o pedido obsoleto fica marcado: %+v", got)
		}
		// Tentar de novo dá o mesmo resultado.
		if r := c.post("/api/decisions/"+id+"/resolve", map[string]any{"option": "seguir-b"}); r.StatusCode != 409 || r.ErrCode() != "decision_stale" {
			t.Fatalf("de novo: %d %s", r.StatusCode, r.Body)
		}
	}
	// Nada foi sinalizado: as execuções não receberam eventos de decisão nem foram retomadas.
	if state, _, ev := execRow(t, e, exA.ID); state != "cancelled" || ev != 0 {
		t.Fatalf("A: %s %d", state, ev)
	}
	if state, _, ev := execRow(t, e, exB.ID); state != "waiting" || ev != 0 {
		t.Fatalf("B: %s %d", state, ev)
	}
	var list struct{ Items []decisions.Request }
	c.expect(c.get("/api/decisions?state=stale"), 200).JSON(&list)
	if len(list.Items) != 2 {
		t.Fatalf("obsoletos: %d", len(list.Items))
	}
}

func TestDecisionInputValidationAndSensitiveReauth(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "validacao")
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	req, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "dev", decisions.KindConflict))
	for name, body := range map[string]map[string]any{
		"opção inexistente": {"option": "nao-existe"},
		"sem opção":         {},
		"nota longa":        {"option": "seguir-a", "note": strings.Repeat("a", 501)},
		"campo extra":       {"option": "seguir-a", "x": 1},
	} {
		if r := c.post("/api/decisions/"+req.ID+"/resolve", body); r.StatusCode != 400 {
			t.Errorf("%s: %d %s", name, r.StatusCode, r.Body)
		}
	}
	if r := c.post("/api/decisions/00000000-0000-4000-8000-000000000000/resolve", map[string]any{"option": "a"}); r.StatusCode != 404 {
		t.Errorf("inexistente: %d", r.StatusCode)
	}
	if r := c.get("/api/decisions?state=talvez"); r.StatusCode != 400 {
		t.Errorf("estado inválido: %d", r.StatusCode)
	}
	if state, _, ev := execRow(t, e, ex.ID); state != "waiting" || ev != 0 {
		t.Fatalf("entradas inválidas não podem sinalizar nada: %s %d", state, ev)
	}

	// Operação sensível exige reautenticação recente; as demais não.
	ex2 := waitingExecution(t, e, p.ID, "EXT-2")
	sens, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex2.ID, "git", decisions.KindSensitiveOperation))
	if _, err := e.db.Pool.Exec(ctx, `UPDATE admin_sessions SET reauth_at = NULL`); err != nil {
		t.Fatal(err)
	}
	if r := c.post("/api/decisions/"+sens.ID+"/resolve", map[string]any{"option": "seguir-a"}); r.StatusCode != 403 || r.ErrCode() != "reauth_required" {
		t.Fatalf("sensível sem reautenticação: %d %s", r.StatusCode, r.Body)
	}
	if state, _, ev := execRow(t, e, ex2.ID); state != "waiting" || ev != 0 {
		t.Fatal("recusado por falta de reautenticação: nada pode mudar")
	}
	c.expect(c.post("/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-a"}), 200) // comum não exige
	c.expect(c.post("/api/auth/reauth", map[string]string{"password": testPassword, "code": e.code(c.secret)}), 200)
	c.expect(c.post("/api/decisions/"+sens.ID+"/resolve", map[string]any{"option": "seguir-a"}), 200)
}

func TestDecisionRoutesRequireSessionAndCSRF(t *testing.T) {
	e, c, _ := newProjectEnv(t)
	ctx := context.Background()
	p := mustProject(t, e.db.Pool, "seguranca-decisao")
	ex := waitingExecution(t, e, p.ID, "EXT-1")
	req, _, _ := decisions.Create(ctx, e.db.Pool, nil, newRequest(p.ID, ex.ID, "dev", decisions.KindConflict))
	anon := e.newClient()
	for _, r := range []apiResp{anon.get("/api/decisions"), anon.get("/api/decisions/" + req.ID), anon.post("/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-a"})} {
		if r.StatusCode != 401 {
			t.Errorf("sem sessão: %d", r.StatusCode)
		}
	}
	if r := c.do("POST", "/api/decisions/"+req.ID+"/resolve", map[string]any{"option": "seguir-a"}, map[string]string{"X-CSRF-Token": ""}); r.StatusCode != 403 {
		t.Errorf("sem CSRF: %d", r.StatusCode)
	}
	if state, _, ev := execRow(t, e, ex.ID); state != "waiting" || ev != 0 {
		t.Fatal("requisição forjada alterou a execução")
	}
	_ = fmt.Sprint
}
