package integration

import (
	"context"
	"sync"
	"testing"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	statusflow "github.com/ViniciusBernucci/le-fabrique/internal/flow"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/projection"
	"github.com/ViniciusBernucci/le-fabrique/internal/sentinel"
)

// moveTracker é uma origem em memória: só o status muda; guarda um campo humano que não pode ser tocado.
type moveTracker struct {
	integrations.Tracker
	mu           sync.Mutex
	status       string
	human        string
	writes       int
	failAfter    bool // grava e devolve indisponível (timeout depois do efeito remoto)
	noTransition bool // o humano moveu a Issue: não há transição válida agora
	names        map[string]string
}

func (m *moveTracker) Provider() string { return "jira" }
func (m *moveTracker) GetItem(context.Context, string) (integrations.Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return integrations.Item{ID: "r1", Key: "K-1", Status: m.status, Description: m.human}, nil
}
func (m *moveTracker) Transitions(context.Context, string) ([]integrations.Transition, error) {
	if m.noTransition {
		return nil, nil
	}
	var ts []integrations.Transition
	for id, name := range m.names {
		ts = append(ts, integrations.Transition{ID: id, ToStatus: name})
	}
	return ts, nil
}
func (m *moveTracker) Transition(_ context.Context, _ string, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writes++
	m.status = m.names[id]
	if m.failAfter {
		return &integrations.Error{Kind: integrations.ErrUnavailable, Op: "transição"}
	}
	return nil
}

type projEnv struct {
	db   testDB
	p    string
	co   *sentinel.Coordinator
	pr   *projection.Projector
	tr   *moveTracker
	m    statusflow.ProjectMap
	step int
}

func newProjEnv(t *testing.T, slug string) *projEnv {
	t.Helper()
	db := newMigratedDB(t)
	ctx := context.Background()
	if err := (&statusflow.Store{DB: db.Pool}).EnsureCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	p := mustProject(t, db.Pool, slug)
	co := &sentinel.Coordinator{DB: db.Pool}
	if _, err := co.Track(ctx, sentinel.NewIssue{ProjectID: p.ID, Key: "K-1", Kind: "feature", Status: statusflow.PRDDoc}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `INSERT INTO issue_registry (project_id, source, remote_id, issue_key, raw_status, status_code, kind) VALUES ($1,'jira','r1','K-1','To Do','PRD-DOC','feature')`, p.ID); err != nil {
		t.Fatal(err)
	}
	m := statusflow.ProjectMap{Source: "jira", Remote: map[string]string{"To Do": statusflow.PRDDoc, "Spec OK": statusflow.PRDDocConcluido, "Planning": statusflow.Planejamento},
		Moves: map[string]string{statusflow.PRDDocConcluido: "t-spec", statusflow.Planejamento: "t-plan"}}
	tr := &moveTracker{status: "To Do", human: "texto escrito por uma pessoa", names: map[string]string{"t-spec": "Spec OK", "t-plan": "Planning"}}
	return &projEnv{db: db, p: p.ID, co: co, pr: &projection.Projector{DB: db.Pool, Ledger: &effects.Ledger{DB: db.Pool}}, tr: tr, m: m}
}

func (e *projEnv) move(t *testing.T, to string, ev statusflow.Evidence) {
	t.Helper()
	e.step++
	if _, err := e.co.Transition(context.Background(), sentinel.Request{ProjectID: e.p, IssueKey: "K-1", To: to, Actor: sentinel.ActorSentinel,
		ObservationKey: "o" + string(rune('0'+e.step)), Evidence: ev}); err != nil {
		t.Fatal(err)
	}
}

var toPlanning = statusflow.Evidence{ProjectActive: true, WorkflowPublished: true, InputsValid: true, PlanningSlot: true}

// AJ01-04 — mapeamento pendente é exposto e se resolve; campo humano preservado.
func TestProjectionExposesMappingPendingThenConfirms(t *testing.T) {
	e := newProjEnv(t, "proj-map")
	ctx := context.Background()
	e.move(t, statusflow.PRDDocConcluido, statusflow.Evidence{PRDComplete: true, InitialContract: true})
	noMove := e.m
	noMove.Moves = map[string]string{}
	res, err := e.pr.Run(ctx, e.p, e.tr, noMove)
	if err != nil || len(res) != 1 || res[0].Outcome != "pending" || res[0].Reason != projection.ReasonMapping || e.tr.writes != 0 {
		t.Fatalf("%+v %v", res, err)
	}
	pend, _ := e.pr.PendingFor(ctx, e.p)
	if len(pend) != 1 || pend[0].Reason != "mapping" || pend[0].Target != statusflow.PRDDocConcluido || pend[0].Version != 2 {
		t.Fatalf("%+v", pend)
	}
	res, err = e.pr.Run(ctx, e.p, e.tr, e.m)
	if err != nil || res[0].Outcome != "applied" || e.tr.writes != 1 || e.tr.status != "Spec OK" || e.tr.human != "texto escrito por uma pessoa" {
		t.Fatalf("%+v %v writes=%d", res, err, e.tr.writes)
	}
	if pend, _ = e.pr.PendingFor(ctx, e.p); len(pend) != 0 {
		t.Fatalf("%+v", pend)
	}
	// Repetir não escreve de novo.
	if res, _ = e.pr.Run(ctx, e.p, e.tr, e.m); len(res) != 0 || e.tr.writes != 1 {
		t.Fatalf("%+v writes=%d", res, e.tr.writes)
	}
}

// AJ01-04 / CT15 — timeout depois da gravação remota é reconciliado por leitura, sem novo envio.
func TestProjectionReconcilesATimeoutAfterTheRemoteWrite(t *testing.T) {
	e := newProjEnv(t, "proj-timeout")
	ctx := context.Background()
	e.move(t, statusflow.PRDDocConcluido, statusflow.Evidence{PRDComplete: true, InitialContract: true})
	e.tr.failAfter = true
	res, err := e.pr.Run(ctx, e.p, e.tr, e.m)
	if err != nil || len(res) != 1 || res[0].Outcome != effects.OutcomeReconciled || e.tr.writes != 1 {
		t.Fatalf("%+v %v writes=%d", res, err, e.tr.writes)
	}
	if pend, _ := e.pr.PendingFor(ctx, e.p); len(pend) != 0 {
		t.Fatalf("%+v", pend)
	}
}

// AJ01-04 — conflito (transição indisponível) fica pendente; intenção antiga é superada pela nova.
func TestProjectionKeepsConflictsPendingAndSupersedesOldIntents(t *testing.T) {
	e := newProjEnv(t, "proj-conflict")
	ctx := context.Background()
	e.move(t, statusflow.PRDDocConcluido, statusflow.Evidence{PRDComplete: true, InitialContract: true})
	e.tr.noTransition = true
	res, _ := e.pr.Run(ctx, e.p, e.tr, e.m)
	if len(res) != 1 || res[0].Reason != projection.ReasonConflict || e.tr.writes != 0 {
		t.Fatalf("%+v", res)
	}
	// A fábrica avança de novo antes de o conflito ser resolvido: só a intenção mais nova é projetada.
	e.move(t, statusflow.Planejamento, toPlanning)
	e.tr.noTransition = false
	res, err := e.pr.Run(ctx, e.p, e.tr, e.m)
	if err != nil || len(res) != 2 || res[0].Outcome != "superseded" || res[1].Outcome != "applied" || e.tr.writes != 1 || e.tr.status != "Planning" {
		t.Fatalf("%+v %v writes=%d status=%s", res, err, e.tr.writes, e.tr.status)
	}
	if pend, _ := e.pr.PendingFor(ctx, e.p); len(pend) != 0 {
		t.Fatalf("%+v", pend)
	}
	if res, _ = e.pr.Run(ctx, e.p, e.tr, e.m); len(res) != 0 {
		t.Fatalf("superada reapareceu: %+v", res)
	}
}
