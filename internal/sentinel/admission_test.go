package sentinel

import (
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
)

var t0 = time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)

func cand(id, work string, tr flow.Track, prio int, age int) Candidate {
	return Candidate{ID: id, Work: work, Track: tr, Priority: prio, EnqueuedAt: t0.Add(time.Duration(age) * time.Minute),
		WorkflowPublished: true, BudgetOK: true, Authorized: true}
}

func decisions(ds []Decision) map[string]Decision {
	m := map[string]Decision{}
	for _, d := range ds {
		m[d.ID] = d
	}
	return m
}

func TestOrderCorrectionThenPriorityThenAgeThenStableID(t *testing.T) {
	a := cand("a", WorkImplementation, flow.Frontend, 1, 0)
	b := cand("b", WorkImplementation, flow.Frontend, 0, 5)
	c := cand("c", WorkImplementation, flow.Frontend, 3, 9)
	c.Correction = true
	d := cand("d", WorkImplementation, flow.Frontend, 0, 5) // empate exato com b
	e := cand("e", WorkImplementation, flow.Frontend, 0, 1)
	got := Order([]Candidate{a, d, b, c, e})
	ids := ""
	for _, x := range got {
		ids += x.ID
	}
	if ids != "cebda" {
		t.Fatalf("ordem %s", ids)
	}
}

func TestIndependentTracksRunInParallelAndSecondFrontWaits(t *testing.T) {
	cs := []Candidate{cand("fa", WorkImplementation, flow.Frontend, 1, 0), cand("fb", WorkImplementation, flow.Frontend, 1, 1),
		cand("ba", WorkImplementation, flow.Backend, 1, 2), cand("pl", WorkPlanning, "", 1, 3)}
	m := decisions(Admit(cs, nil, nil, Limits{PlanningSlots: 1, GlobalCapacity: 10}))
	if !m["fa"].Admitted || !m["ba"].Admitted || !m["pl"].Admitted || m["fb"].Admitted || m["fb"].Wait != WaitSlot {
		t.Fatalf("%+v", m)
	}
}

func TestReviewSharesTheTrackSlotWithImplementation(t *testing.T) {
	running := []Running{{ID: "impl", Work: WorkImplementation, Track: flow.Frontend}}
	m := decisions(Admit([]Candidate{cand("rev", WorkReview, flow.Frontend, 0, 0)}, running, nil, Limits{GlobalCapacity: 5}))
	if m["rev"].Admitted || m["rev"].Wait != WaitSlot {
		t.Fatalf("%+v", m)
	}
}

func TestBlockedCorrectionDoesNotStopIndependentWork(t *testing.T) {
	fix := cand("fix", WorkImplementation, flow.Backend, 0, 0)
	fix.Correction, fix.DependsOn = true, []string{"api"}
	m := decisions(Admit([]Candidate{fix, cand("other", WorkImplementation, flow.Backend, 4, 9)}, nil, map[string]bool{}, Limits{GlobalCapacity: 5}))
	if m["fix"].Admitted || m["fix"].Wait != WaitDependency || !m["other"].Admitted {
		t.Fatalf("%+v", m)
	}
}

func TestCorrectionTakesNextFreeSlotWithoutPreemption(t *testing.T) {
	running := []Running{{ID: "feat", Work: WorkImplementation, Track: flow.Frontend}}
	fix := cand("fix", WorkImplementation, flow.Frontend, 4, 50)
	fix.Correction = true
	waiting := cand("w", WorkImplementation, flow.Frontend, 0, 0)
	m := decisions(Admit([]Candidate{waiting, fix}, running, nil, Limits{GlobalCapacity: 5}))
	if m["fix"].Admitted || m["fix"].Position != 1 || m["w"].Position != 2 {
		t.Fatalf("%+v", m) // a execução ativa segue; a correção é a primeira da fila
	}
	m = decisions(Admit([]Candidate{waiting, fix}, nil, nil, Limits{GlobalCapacity: 5}))
	if !m["fix"].Admitted || m["w"].Admitted {
		t.Fatalf("%+v", m)
	}
}

func TestSharedResourceConflictBlocksEvenWithFreeSlots(t *testing.T) {
	f := cand("f", WorkImplementation, flow.Frontend, 0, 0)
	b := cand("b", WorkImplementation, flow.Backend, 0, 1)
	f.Resources, b.Resources = []string{"db/schema"}, []string{"db/schema"}
	m := decisions(Admit([]Candidate{f, b}, nil, nil, Limits{GlobalCapacity: 5}))
	if !m["f"].Admitted || m["b"].Admitted || m["b"].Wait != WaitResource {
		t.Fatalf("%+v", m)
	}
	b.Resources = []string{"outro"}
	if m = decisions(Admit([]Candidate{f, b}, nil, nil, Limits{GlobalCapacity: 5})); !m["b"].Admitted {
		t.Fatalf("%+v", m)
	}
}

func TestBothTracksNeedsBothSlotsAtomically(t *testing.T) {
	x := cand("x", WorkImplementation, flow.Frontend, 0, 0)
	x.NeedsBoth = true
	running := []Running{{ID: "be", Work: WorkImplementation, Track: flow.Backend}}
	if m := decisions(Admit([]Candidate{x}, running, nil, Limits{GlobalCapacity: 5})); m["x"].Admitted || m["x"].Wait != WaitSlot {
		t.Fatalf("%+v", m)
	}
	if m := decisions(Admit([]Candidate{x}, nil, nil, Limits{GlobalCapacity: 5})); !m["x"].Admitted {
		t.Fatalf("%+v", m)
	}
}

func TestGatesPlanningLimitAndGlobalCapacity(t *testing.T) {
	p1, p2 := cand("p1", WorkPlanning, "", 0, 0), cand("p2", WorkPlanning, "", 0, 1)
	m := decisions(Admit([]Candidate{p1, p2}, nil, nil, Limits{PlanningSlots: 1, GlobalCapacity: 5}))
	if !m["p1"].Admitted || m["p2"].Wait != WaitPlanning {
		t.Fatalf("%+v", m)
	}
	m = decisions(Admit([]Candidate{cand("a", WorkImplementation, flow.Frontend, 0, 0), cand("b", WorkImplementation, flow.Backend, 0, 1)}, nil, nil, Limits{GlobalCapacity: 1}))
	if !m["a"].Admitted || m["b"].Wait != WaitCapacity {
		t.Fatalf("%+v", m)
	}
	paused := cand("z", WorkPlanning, "", 0, 0)
	paused.Paused = true
	noWF := cand("y", WorkPlanning, "", 0, 0)
	noWF.WorkflowPublished = false
	noBudget := cand("w", WorkPlanning, "", 0, 0)
	noBudget.BudgetOK = false
	noAuth := cand("v", WorkPlanning, "", 0, 0)
	noAuth.Authorized = false
	m = decisions(Admit([]Candidate{paused, noWF, noBudget, noAuth}, nil, nil, Limits{PlanningSlots: 9, GlobalCapacity: 9}))
	if m["z"].Wait != WaitPaused || m["y"].Wait != WaitWorkflow || m["w"].Wait != WaitBudget || m["v"].Wait != WaitAuth {
		t.Fatalf("%+v", m)
	}
}
