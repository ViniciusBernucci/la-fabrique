package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/tasksync"
)

// memTracker é uma origem de tarefas em memória, com falhas injetáveis, que
// guarda o que de fato aconteceu (independente do que o chamador ficou sabendo).
type memTracker struct {
	mu       sync.Mutex
	items    map[string]*integrations.Item
	children map[string][]string
	links    map[string][]string
	next     int

	creates, transitions, linkAdds, depAdds int
	createFn                                func(call int) (lands bool, err error)
	transitionFn                            func(call int) (lands bool, err error)
	childrenErr                             error
}

var errTrackerTimeout = errors.New("timeout")

func newMem() *memTracker {
	m := &memTracker{items: map[string]*integrations.Item{}, children: map[string][]string{}, links: map[string][]string{}}
	m.items["P1"] = &integrations.Item{Provider: "mem", ID: "P1", Key: "ABC-1", Title: "Feature", Status: "To Do", Category: "todo", Dependencies: []string{}}
	m.items["P2"] = &integrations.Item{Provider: "mem", ID: "P2", Key: "ABC-2", Title: "Outra", Status: "To Do", Category: "todo", Dependencies: []string{}}
	return m
}

func (m *memTracker) Provider() string { return "mem" }
func (m *memTracker) Identity(context.Context) (integrations.Identity, error) {
	return integrations.Identity{Account: "1", Name: "bot"}, nil
}
func (m *memTracker) ItemTypes(context.Context, string) ([]integrations.ItemType, error) {
	return nil, nil
}
func (m *memTracker) Statuses(context.Context, string) ([]integrations.Status, error) {
	return nil, nil
}
func (m *memTracker) Fields(context.Context, string) ([]integrations.Field, error) { return nil, nil }
func (m *memTracker) Priorities(context.Context) ([]string, error)                 { return nil, nil }
func (m *memTracker) ListItems(context.Context, string, integrations.ListOptions) (integrations.ItemPage, error) {
	return integrations.ItemPage{}, nil
}

func (m *memTracker) GetItem(_ context.Context, id string) (integrations.Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it, ok := m.items[id]
	if !ok {
		return integrations.Item{}, &integrations.Error{Kind: integrations.ErrNotFound, Op: "ler item"}
	}
	return *it, nil
}

var flow = map[string][]string{"To Do": {"In Progress"}, "In Progress": {"In Review", "To Do"}, "In Review": {"Done", "In Progress"}}

func (m *memTracker) Transitions(_ context.Context, id string) ([]integrations.Transition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	it, ok := m.items[id]
	if !ok {
		return nil, &integrations.Error{Kind: integrations.ErrNotFound, Op: "transições"}
	}
	var out []integrations.Transition
	for _, to := range flow[it.Status] {
		out = append(out, integrations.Transition{ID: to, Name: to, ToStatus: to})
	}
	return out, nil
}

func (m *memTracker) Transition(_ context.Context, id, transitionID string) error {
	m.mu.Lock()
	m.transitions++
	call := m.transitions
	m.mu.Unlock()
	lands, err := true, error(nil)
	if m.transitionFn != nil {
		lands, err = m.transitionFn(call)
	}
	if lands {
		m.mu.Lock()
		m.items[id].Status = transitionID
		m.mu.Unlock()
	}
	return err
}

func (m *memTracker) CreateSubtask(_ context.Context, parent string, in integrations.SubtaskInput) (integrations.Item, error) {
	m.mu.Lock()
	m.creates++
	call := m.creates
	m.mu.Unlock()
	lands, err := true, error(nil)
	if m.createFn != nil {
		lands, err = m.createFn(call)
	}
	if !lands {
		return integrations.Item{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	id := fmt.Sprintf("S%d", m.next)
	it := &integrations.Item{Provider: "mem", ID: id, Key: "ABC-" + id, Title: in.Title, Description: in.Description, Subtask: true, ParentID: parent, Status: "To Do", Dependencies: []string{}}
	m.items[id] = it
	m.children[parent] = append(m.children[parent], id)
	return *it, err
}

func (m *memTracker) AddDependency(_ context.Context, item, dependsOn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.depAdds++
	m.items[item].Dependencies = append(m.items[item].Dependencies, m.items[dependsOn].ID)
	return nil
}

func (m *memTracker) AddLink(_ context.Context, item, url, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.linkAdds++
	m.links[item] = append(m.links[item], url)
	return nil
}

func (m *memTracker) Children(_ context.Context, parent string) ([]integrations.Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.childrenErr != nil {
		return nil, m.childrenErr
	}
	var out []integrations.Item
	for _, id := range m.children[parent] {
		out = append(out, *m.items[id])
	}
	return out, nil
}

func (m *memTracker) HasLink(_ context.Context, item, url string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.links[item] {
		if u == url {
			return true, nil
		}
	}
	return false, nil
}

func (m *memTracker) count(parent string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.children[parent])
}

type fakeMerges struct {
	state tasksync.MergeState
	err   error
}

func (f fakeMerges) Merged(context.Context, string) (tasksync.MergeState, error) {
	return f.state, f.err
}

func publisher(e *authEnv, tr integrations.Tracker, merges tasksync.MergeVerifier) *tasksync.Publisher {
	return &tasksync.Publisher{
		Tracker: tr, Ledger: &effects.Ledger{DB: e.db.Pool, Lease: time.Minute, MaxAttempts: 3}, Merges: merges,
		StatusMap: map[string]string{tasksync.StateInProgress: "In Progress", tasksync.StateAwaitingMerge: "In Review", tasksync.StateDone: "Done"},
	}
}

// T06-03 — "Timeout após criação não duplica subtarefa."
func TestSubtaskCreationTimeoutDoesNotDuplicate(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "sync-subtarefa")
	ctx := context.Background()

	// A subtarefa é criada na origem, mas a resposta se perde.
	tr := newMem()
	tr.createFn = func(int) (bool, error) { return true, errTrackerTimeout }
	pub := publisher(e, tr, nil)
	key := "feat1:1:tarefa-7"
	res, err := pub.CreateSubtask(ctx, p.ID, key, "P1", "Validar CPF", "Detalhes")
	if err != nil || res.Outcome != effects.OutcomeReconciled || tr.count("P1") != 1 || tr.creates != 1 {
		t.Fatalf("%+v %v filhos=%d criações=%d", res, err, tr.count("P1"), tr.creates)
	}
	id, _ := pub.RemoteIDFor(ctx, p.ID, key+":subtask")
	kid, _ := tr.GetItem(ctx, id)
	if id == "" || !kid.Subtask || kid.ParentID != "P1" || !strings.Contains(kid.Description, tasksync.Marker(key+":subtask")) || !strings.HasPrefix(kid.Description, "Detalhes") {
		t.Fatalf("ID remoto associado e marcador na descrição: id=%q %+v", id, kid)
	}
	// Repetir (nova tentativa, reinício do worker) não cria outra.
	if again, err := pub.CreateSubtask(ctx, p.ID, key, "P1", "Validar CPF", "Detalhes"); err != nil || again.Outcome != effects.OutcomeAlreadyConfirmed || tr.count("P1") != 1 || tr.creates != 1 {
		t.Fatalf("%+v %v filhos=%d", again, err, tr.count("P1"))
	}

	// A chamada nem chegou à origem: a ausência é comprovada e a repetição é segura.
	tr2 := newMem()
	tr2.createFn = func(call int) (bool, error) {
		if call == 1 {
			return false, errTrackerTimeout
		}
		return true, nil
	}
	if res, err := publisher(e, tr2, nil).CreateSubtask(ctx, p.ID, "feat2:1:tarefa-1", "P1", "Outra", ""); err != nil || res.Outcome != effects.OutcomeApplied || tr2.count("P1") != 1 || tr2.creates != 2 {
		t.Fatalf("%+v %v filhos=%d criações=%d", res, err, tr2.count("P1"), tr2.creates)
	}

	// Sem conseguir listar as subtarefas não dá para provar nada: bloqueia em vez de arriscar duplicar.
	tr3 := newMem()
	tr3.createFn = func(int) (bool, error) { return true, errTrackerTimeout }
	tr3.childrenErr = &integrations.Error{Kind: integrations.ErrUnavailable, Op: "listar"}
	if _, err := publisher(e, tr3, nil).CreateSubtask(ctx, p.ID, "feat3:1:tarefa-1", "P1", "Incerta", ""); !errors.Is(err, effects.ErrBlocked) || tr3.creates != 1 {
		t.Fatalf("sem prova o ledger bloqueia: %v criações=%d", err, tr3.creates)
	}
	// A mesma tarefa em outra tentativa da feature (outra chave) é outro efeito.
	tr4 := newMem()
	pub4 := publisher(e, tr4, nil)
	for _, k := range []string{"featX:1:tarefa-1", "featX:2:tarefa-1"} {
		if _, err := pub4.CreateSubtask(ctx, p.ID, k, "P1", "Mesma tarefa", ""); err != nil {
			t.Fatal(err)
		}
	}
	if tr4.count("P1") != 2 {
		t.Fatalf("chaves diferentes são efeitos diferentes: %d", tr4.count("P1"))
	}
}

func TestDependenciesAndLinksAreWrittenOnce(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "sync-dep")
	ctx := context.Background()
	tr := newMem()
	pub := publisher(e, tr, nil)
	for i := 0; i < 3; i++ {
		if _, err := pub.AddDependency(ctx, p.ID, "feat:1", "P1", "P2"); err != nil {
			t.Fatal(err)
		}
		if _, err := pub.AddLink(ctx, p.ID, "feat:1", "P1", "https://github.com/o/r/pull/7", "Draft PR"); err != nil {
			t.Fatal(err)
		}
	}
	if tr.depAdds != 1 || tr.linkAdds != 1 || len(tr.items["P1"].Dependencies) != 1 {
		t.Fatalf("dependências=%d links=%d", tr.depAdds, tr.linkAdds)
	}
	// A resposta se perde depois de gravar: a prova encontra a dependência.
	tr2 := newMem()
	tr2.items["P1"].Dependencies = []string{"P2"} // já existe na origem, ledger não sabe
	if res, err := publisher(e, tr2, nil).AddDependency(ctx, p.ID, "feat:2", "P1", "P2"); err != nil || res.Outcome != effects.OutcomeApplied {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestStatusMappingRevalidatesTheTargetBeforeRepeating(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "sync-estado")
	ctx := context.Background()

	// Resposta perdida depois de mover: não move de novo.
	tr := newMem()
	tr.transitionFn = func(int) (bool, error) { return true, errTrackerTimeout }
	pub := publisher(e, tr, nil)
	res, err := pub.MoveTo(ctx, p.ID, "feat:1", "P1", tasksync.StateInProgress, nil)
	if err != nil || res.Outcome != effects.OutcomeReconciled || tr.items["P1"].Status != "In Progress" || tr.transitions != 1 {
		t.Fatalf("%+v %v %s %d", res, err, tr.items["P1"].Status, tr.transitions)
	}
	// O item já estava no estado desejado (alguém moveu à mão): nenhuma escrita.
	tr2 := newMem()
	tr2.items["P1"].Status = "In Progress"
	if res, err := publisher(e, tr2, nil).MoveTo(ctx, p.ID, "feat:2", "P1", tasksync.StateInProgress, nil); err != nil || res.Outcome != effects.OutcomeApplied || tr2.transitions != 0 {
		t.Fatalf("já no estado: %+v %v escritas=%d", res, err, tr2.transitions)
	}
	// Sem transição válida agora (To Do → Done não existe): recusa sem efeito e sem tentar de novo.
	tr3 := newMem()
	res, err = publisher(e, tr3, nil).MoveTo(ctx, p.ID, "feat:3", "P1", tasksync.StateDone, nil)
	if !errors.Is(err, effects.ErrNotApplied) || res.Outcome != effects.OutcomeFailed || tr3.transitions != 0 || tr3.items["P1"].Status != "To Do" {
		t.Fatalf("%+v %v", res, err)
	}
	// Estado sem mapa para a origem.
	pubNoMap := publisher(e, newMem(), nil)
	pubNoMap.StatusMap = map[string]string{}
	if _, err := pubNoMap.MoveTo(ctx, p.ID, "feat:4", "P1", tasksync.StateInProgress, nil); !errors.Is(err, tasksync.ErrUnmapped) {
		t.Fatalf("%v", err)
	}
}

// T06-03 — "Draft PR publica aguardando merge e conclusão depende de merge confirmado."
func TestDraftPRWaitsForMergeAndCompletionNeedsAConfirmedMerge(t *testing.T) {
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, "sync-pr")
	ctx := context.Background()
	const pr = "https://github.com/o/r/pull/7"
	newTracker := func() *memTracker {
		tr := newMem()
		tr.items["P1"].Status = "In Progress" // a feature estava em desenvolvimento
		return tr
	}

	tr := newTracker()
	pub := publisher(e, tr, fakeMerges{state: tasksync.MergeState{}}) // PR aberta, sem merge
	if err := pub.PublishDraftPR(ctx, p.ID, "feat:1", "P1", pr); err != nil {
		t.Fatal(err)
	}
	if tr.items["P1"].Status != "In Review" || len(tr.links["P1"]) != 1 || tr.links["P1"][0] != pr {
		t.Fatalf("Draft PR publicada: aguardando merge, com o link: %s %v", tr.items["P1"].Status, tr.links["P1"])
	}
	// Publicar de novo não duplica link nem muda nada.
	if err := pub.PublishDraftPR(ctx, p.ID, "feat:1", "P1", pr); err != nil || tr.linkAdds != 1 || tr.transitions != 1 {
		t.Fatalf("%v links=%d transições=%d", err, tr.linkAdds, tr.transitions)
	}

	// Sem merge confirmado a tarefa NÃO conclui.
	if _, err := pub.Complete(ctx, p.ID, "feat:1", "P1", pr); !errors.Is(err, tasksync.ErrMergeNotConfirmed) {
		t.Fatalf("PR aberta: %v", err)
	}
	// Merge declarado sem o commit de integração também não vale.
	pub.Merges = fakeMerges{state: tasksync.MergeState{Merged: true}}
	if _, err := pub.Complete(ctx, p.ID, "feat:1", "P1", pr); !errors.Is(err, tasksync.ErrMergeNotConfirmed) {
		t.Fatalf("merge sem SHA: %v", err)
	}
	// Falha ao consultar o repositório: não conclui.
	pub.Merges = fakeMerges{err: errors.New("GitHub fora do ar")}
	if _, err := pub.Complete(ctx, p.ID, "feat:1", "P1", pr); !errors.Is(err, tasksync.ErrMergeNotConfirmed) {
		t.Fatalf("consulta falhou: %v", err)
	}
	// PR fechada sem merge: pede decisão, não conclui.
	pub.Merges = fakeMerges{state: tasksync.MergeState{Closed: true}}
	if _, err := pub.Complete(ctx, p.ID, "feat:1", "P1", pr); !errors.Is(err, tasksync.ErrPRClosed) {
		t.Fatalf("PR fechada: %v", err)
	}
	if tr.items["P1"].Status != "In Review" || tr.transitions != 1 {
		t.Fatalf("nada foi escrito sem merge confirmado: %s transições=%d", tr.items["P1"].Status, tr.transitions)
	}

	// Merge confirmado: conclui uma única vez e guarda o commit de integração.
	pub.Merges = fakeMerges{state: tasksync.MergeState{Merged: true, SHA: "abc123"}}
	res, err := pub.Complete(ctx, p.ID, "feat:1", "P1", pr)
	if err != nil || res.Outcome != effects.OutcomeApplied || tr.items["P1"].Status != "Done" || tr.transitions != 2 {
		t.Fatalf("%+v %v %s", res, err, tr.items["P1"].Status)
	}
	if !strings.Contains(string(res.Entry.Request), "abc123") {
		t.Fatalf("o commit de integração fica na intenção do efeito: %s", res.Entry.Request)
	}
	if again, err := pub.Complete(ctx, p.ID, "feat:1", "P1", pr); err != nil || again.Outcome != effects.OutcomeAlreadyConfirmed || tr.transitions != 2 {
		t.Fatalf("%+v %v transições=%d", again, err, tr.transitions)
	}
	if _, err := (&tasksync.Publisher{Tracker: tr, Ledger: pub.Ledger, StatusMap: pub.StatusMap}).Complete(ctx, p.ID, "feat:9", "P1", pr); !errors.Is(err, tasksync.ErrNoVerifier) {
		t.Fatalf("sem verificador: %v", err)
	}
}
