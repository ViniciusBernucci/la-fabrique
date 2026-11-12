package integration

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/syncsource"
)

// srcTracker é uma origem em memória que conta cada leitura.
type srcTracker struct {
	integrations.Tracker
	mu       sync.Mutex
	items    []integrations.Item
	pageSize int
	calls    int
	err      error
}

func (m *srcTracker) Provider() string { return "jira" }

func (m *srcTracker) ListItems(_ context.Context, _ string, o integrations.ListOptions) (integrations.ItemPage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return integrations.ItemPage{}, m.err
	}
	start := 0
	if o.Cursor != "" {
		start, _ = strconv.Atoi(o.Cursor)
	}
	size := m.pageSize
	if size == 0 {
		size = 100
	}
	end := start + size
	next := strconv.Itoa(end)
	if end >= len(m.items) {
		end, next = len(m.items), ""
	}
	return integrations.ItemPage{Items: append([]integrations.Item{}, m.items[start:end]...), NextCursor: next}, nil
}

func (m *srcTracker) GetItem(_ context.Context, id string) (integrations.Item, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.err != nil {
		return integrations.Item{}, m.err
	}
	for _, it := range m.items {
		if it.ID == id {
			return it, nil
		}
	}
	return integrations.Item{}, &integrations.Error{Kind: integrations.ErrNotFound, Op: "ler item"}
}

func (m *srcTracker) set(id string, f func(*integrations.Item)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.items {
		if m.items[i].ID == id {
			f(&m.items[i])
			m.items[i].Version = m.items[i].Version + "+"
		}
	}
}

func (m *srcTracker) remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.items {
		if m.items[i].ID == id {
			m.items = append(m.items[:i], m.items[i+1:]...)
			return
		}
	}
}

func issue(id, status, cat, prio string) integrations.Item {
	return integrations.Item{Provider: "jira", ID: id, Key: "ABC-" + id, Title: "Item " + id, Description: "desc " + id, Status: status, Category: cat,
		Priority: prio, Labels: []string{"lf-ready"}, Dependencies: []string{}, Version: "v1", URL: "https://x/" + id}
}

var syncSrc = syncsource.Source{Kind: "jira", Container: "ABC", Marker: "lf-ready", EligibleStatuses: []string{"To Do"},
	PriorityMap: map[string]int{"highest": 0, "high": 1, "medium": 2, "low": 3}}

type syEnv struct {
	e    *authEnv
	s    *syncsource.Syncer
	proj string
	now  time.Time
	tr   *srcTracker
}

func newSyEnv(t *testing.T, name string) *syEnv {
	t.Helper()
	e, _, _ := newProjectEnv(t)
	p := mustProject(t, e.db.Pool, name)
	y := &syEnv{e: e, proj: p.ID, now: time.Date(2026, 11, 12, 15, 0, 0, 0, time.UTC)}
	y.s = &syncsource.Syncer{DB: e.db.Pool, Now: func() time.Time { return y.now }}
	y.tr = &srcTracker{items: []integrations.Item{
		issue("1", "To Do", "todo", "High"), issue("2", "In Progress", "in_progress", "Medium"),
		issue("3", "Done", "done", "High"), issue("4", "To Do", "todo", "Highest"),
	}}
	return y
}

func (y *syEnv) run(t *testing.T) syncsource.Result {
	t.Helper()
	r, err := y.s.Run(context.Background(), y.proj, y.tr, syncSrc)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func (y *syEnv) syncs(t *testing.T) (n int) {
	t.Helper()
	_ = y.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM source_syncs WHERE project_id = $1`, y.proj).Scan(&n)
	return
}

func findChange(r syncsource.Result, id string) *syncsource.Change {
	for i := range r.Changes {
		if r.Changes[i].RemoteID == id {
			return &r.Changes[i]
		}
	}
	return nil
}

// T06-04 — "Relógio avança ocioso sem leitura periódica de mudanças."
func TestIdleClockNeverReadsTheSource(t *testing.T) {
	y := newSyEnv(t, "sync-idle")
	y.run(t)
	calls, syncs := y.tr.calls, y.syncs(t)
	y.tr.set("1", func(i *integrations.Item) { i.Title = "Mudou na origem" })
	y.now = y.now.Add(30 * 24 * time.Hour) // um mês ocioso
	time.Sleep(50 * time.Millisecond)
	if y.tr.calls != calls || y.syncs(t) != syncs {
		t.Fatalf("nada pode ler a origem sozinho: %d→%d leituras", calls, y.tr.calls)
	}
	// A mudança só é vista na próxima consulta permitida.
	r := y.run(t)
	if c := findChange(r, "1"); c == nil || c.Kind != syncsource.KindChanged || c.Fields["title"].To != "Mudou na origem" {
		t.Fatalf("%+v", r.Changes)
	}
}

// T06-04 — "Sync manual ou preflight detecta a alteração uma vez."
func TestAChangeIsDetectedExactlyOnce(t *testing.T) {
	y := newSyEnv(t, "sync-once")
	r := y.run(t)
	if r.New != 4 || r.Changed != 0 || r.Read != 4 || r.Outcome != "ok" || r.SnapshotHash == "" {
		t.Fatalf("%+v", r)
	}
	if r2 := y.run(t); len(r2.Changes) != 0 || r2.SnapshotHash != r.SnapshotHash {
		t.Fatalf("repetir sem mudança não gera nada: %+v", r2.Changes)
	}
	y.tr.set("2", func(i *integrations.Item) { i.Status, i.Category = "To Do", "todo"; i.Priority = "Highest" })
	r3 := y.run(t)
	c := findChange(r3, "2")
	if len(r3.Changes) != 1 || c == nil || c.BeforeVersion != "v1" || c.AfterVersion != "v1+" ||
		c.Fields["status"].To != "To Do" || c.Fields["eligible"].To != true || c.Fields["mapped_priority"].To == nil || r3.SnapshotHash == r.SnapshotHash {
		t.Fatalf("%+v", r3.Changes)
	}
	if r4 := y.run(t); len(r4.Changes) != 0 {
		t.Fatalf("a mesma mudança não pode ser detectada de novo: %+v", r4.Changes)
	}
	// O preflight do mesmo item, já registrado, também não a repete.
	pf, err := y.s.Preflight(context.Background(), y.proj, y.tr, syncSrc, "2")
	if err != nil || len(pf.Changes) != 0 || !pf.Proceed {
		t.Fatalf("%+v %v", pf, err)
	}
	// Mudança vista primeiro pelo preflight não aparece de novo na sync manual.
	y.tr.set("4", func(i *integrations.Item) { i.Title = "Novo título" })
	pf, _ = y.s.Preflight(context.Background(), y.proj, y.tr, syncSrc, "4")
	if len(pf.Changes) != 1 || pf.Changes[0].Fields["title"].To != "Novo título" {
		t.Fatalf("%+v", pf.Changes)
	}
	if r5 := y.run(t); len(r5.Changes) != 0 {
		t.Fatalf("já registrada pelo preflight: %+v", r5.Changes)
	}
}

func TestImportIsDeduplicatedAndPaginated(t *testing.T) {
	y := newSyEnv(t, "sync-dedup")
	y.tr.pageSize = 2
	y.tr.items = append(y.tr.items, y.tr.items[0]) // o mesmo ID repetido numa página
	r := y.run(t)
	if r.New != 4 || r.Read != 5 {
		t.Fatalf("%+v", r)
	}
	items, _ := y.s.Items(context.Background(), y.proj)
	if len(items) != 4 {
		t.Fatalf("um item por identidade remota: %d", len(items))
	}
	if r := y.run(t); len(r.Changes) != 0 {
		t.Fatalf("%+v", r.Changes)
	}
	by := map[string]syncsource.Item{}
	for _, it := range items {
		by[it.RemoteID] = it
	}
	if !by["1"].Eligible || by["2"].Eligible || by["3"].Eligible || by["4"].MappedPriority == nil || *by["4"].MappedPriority != 0 {
		t.Fatalf("elegibilidade e prioridade reavaliadas: %+v", by)
	}
}

func TestItemsLeavingAndReturningAreRecordedOnce(t *testing.T) {
	y := newSyEnv(t, "sync-gone")
	y.run(t)
	saved := y.tr.items[3]
	y.tr.remove("4")
	r := y.run(t)
	if c := findChange(r, "4"); len(r.Changes) != 1 || c == nil || c.Kind != syncsource.KindGone {
		t.Fatalf("%+v", r.Changes)
	}
	if r := y.run(t); len(r.Changes) != 0 {
		t.Fatalf("saída detectada uma vez: %+v", r.Changes)
	}
	y.tr.items = append(y.tr.items, saved)
	if r := y.run(t); len(r.Changes) != 1 || r.Changes[0].Kind != syncsource.KindReappeared {
		t.Fatalf("%+v", r.Changes)
	}
}

func TestAFailedReadChangesNothing(t *testing.T) {
	y := newSyEnv(t, "sync-fail")
	y.run(t)
	y.tr.err = &integrations.Error{Kind: integrations.ErrRateLimited, Op: "listar", RetryAfter: time.Minute}
	r, err := y.s.Run(context.Background(), y.proj, y.tr, syncSrc)
	if err != nil || r.Outcome != "failed" || !r.Retryable || r.Error == "" || len(r.Changes) != 0 {
		t.Fatalf("%+v %v", r, err)
	}
	items, _ := y.s.Items(context.Background(), y.proj)
	for _, it := range items {
		if it.State != "active" {
			t.Fatalf("leitura incompleta não pode dar item como removido: %+v", it)
		}
	}
	last, _ := y.s.Last(context.Background(), y.proj)
	if last.Outcome != "failed" || last.SyncID != r.SyncID {
		t.Fatalf("a falha fica registrada: %+v", last)
	}
	y.tr.err = nil
	if r := y.run(t); r.Outcome != "ok" || len(r.Changes) != 0 {
		t.Fatalf("%+v", r)
	}
}

// T06-04 — "Validar ... mudança de prioridade sem preempção" e "rejeitar atualização que sobrescreva silenciosamente edição humana".
func TestRemotePriorityChangeNeverOverwritesALocalEdit(t *testing.T) {
	y := newSyEnv(t, "sync-conflict")
	ctx := context.Background()
	y.run(t)
	three := 3
	if _, err := y.s.SetLocalPriority(ctx, y.proj, "1", &three); err != nil {
		t.Fatal(err)
	}
	if _, err := y.s.SetLocalPriority(ctx, y.proj, "1", func() *int { v := 9; return &v }()); err == nil {
		t.Fatal("prioridade fora da faixa")
	}
	y.tr.set("1", func(i *integrations.Item) { i.Priority = "Highest" })
	r := y.run(t)
	c := findChange(r, "1")
	if c == nil || c.Kind != syncsource.KindConflict || r.Conflicts != 1 || c.Detail == "" {
		t.Fatalf("%+v", r.Changes)
	}
	items, _ := y.s.Items(ctx, y.proj)
	var it syncsource.Item
	for _, x := range items {
		if x.RemoteID == "1" {
			it = x
		}
	}
	if it.LocalPriority == nil || *it.LocalPriority != 3 || *it.EffectivePriority() != 3 || it.Conflict == nil || *it.Conflict.Remote != 0 || *it.MappedPriority != 0 {
		t.Fatalf("o valor local fica e o da origem é guardado: %+v", it)
	}
	if r := y.run(t); len(r.Changes) != 0 {
		t.Fatalf("o conflito não se repete: %+v", r.Changes)
	}
	pf, _ := y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "1")
	if pf.Proceed || pf.Item.Conflict == nil {
		t.Fatalf("conflito aberto bloqueia a execução: %+v", pf)
	}
	if _, err := y.s.ResolveConflict(ctx, y.proj, "1", "outra"); err == nil {
		t.Fatal("escolha inválida")
	}
	got, err := y.s.ResolveConflict(ctx, y.proj, "1", "use_remote")
	if err != nil || got.LocalPriority != nil || got.Conflict != nil || *got.EffectivePriority() != 0 {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err := y.s.ResolveConflict(ctx, y.proj, "1", "keep_local"); err == nil {
		t.Fatal("sem conflito aberto")
	}
	if pf, _ := y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "1"); !pf.Proceed {
		t.Fatalf("%+v", pf)
	}
}

func TestPreflightBlocksAnItemThatStoppedBeingEligible(t *testing.T) {
	y := newSyEnv(t, "sync-preflight")
	ctx := context.Background()
	y.run(t)
	pf, err := y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "1")
	if err != nil || !pf.Proceed || pf.Sync.Trigger != "preflight" || pf.Sync.Read != 1 {
		t.Fatalf("%+v %v", pf, err)
	}
	for name, mut := range map[string]func(*integrations.Item){
		"concluído":     func(i *integrations.Item) { i.Status, i.Category = "Done", "done" },
		"fora do fluxo": func(i *integrations.Item) { i.Status, i.Category = "Blocked", "in_progress" },
		"sem marcador":  func(i *integrations.Item) { i.Status, i.Category = "To Do", "todo"; i.Labels = nil },
		"subtarefa":     func(i *integrations.Item) { i.Labels = []string{"lf-ready"}; i.Subtask = true },
	} {
		y.tr.set("1", func(i *integrations.Item) {
			i.Status, i.Category, i.Labels, i.Subtask = "To Do", "todo", []string{"lf-ready"}, false
		})
		y.tr.set("1", mut)
		pf, err := y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "1")
		if err != nil || pf.Proceed || pf.Reason == "" {
			t.Fatalf("%s: %+v %v", name, pf, err)
		}
	}
	y.tr.remove("1")
	pf, _ = y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "1")
	if pf.Proceed || pf.Item == nil || pf.Item.State != "gone" {
		t.Fatalf("%+v", pf)
	}
	pf, _ = y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "999")
	if pf.Proceed || pf.Item != nil {
		t.Fatalf("item desconhecido: %+v", pf)
	}
	// Falha de leitura: não libera e fica registrada, sem tocar no item.
	y.tr.err = &integrations.Error{Kind: integrations.ErrUnavailable, Op: "ler"}
	pf, err = y.s.Preflight(ctx, y.proj, y.tr, syncSrc, "4")
	if err != nil || pf.Proceed || pf.Sync.Outcome != "failed" {
		t.Fatalf("%+v %v", pf, err)
	}
}

func TestConcurrentSyncsRecordTheChangeOnce(t *testing.T) {
	y := newSyEnv(t, "sync-race")
	y.run(t)
	y.tr.set("3", func(i *integrations.Item) { i.Title = "Mudou" })
	var wg sync.WaitGroup
	results := make([]syncsource.Result, 4)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := y.s.Run(context.Background(), y.proj, y.tr, syncSrc)
			if err != nil {
				t.Error(err)
			}
			results[i] = r
		}()
	}
	wg.Wait()
	total := 0
	for _, r := range results {
		total += len(r.Changes)
	}
	var n int
	_ = y.e.db.Pool.QueryRow(context.Background(), `SELECT count(*) FROM source_changes WHERE project_id = $1 AND kind = 'changed'`, y.proj).Scan(&n)
	if total != 1 || n != 1 {
		t.Fatalf("a mudança deve ser registrada uma vez: %d / %d", total, n)
	}
}

// API: sincronização manual pelo painel, com a conexão do cofre e a origem configurada.
func TestManualSyncThroughTheAPI(t *testing.T) {
	f := newSourceFixture(t, "sync-api")
	var first syncsource.Result
	f.c.expect(f.c.post("/api/projects/"+f.p.ID+"/sync", nil), 200).JSON(&first)
	if first.Outcome != "ok" || first.New != 4 || first.Trigger != "manual" {
		t.Fatalf("%+v", first)
	}
	var again syncsource.Result
	f.c.expect(f.c.post("/api/projects/"+f.p.ID+"/sync", nil), 200).JSON(&again)
	if len(again.Changes) != 0 || again.SyncID == first.SyncID {
		t.Fatalf("%+v", again)
	}
	var view struct {
		Last  *syncsource.Result `json:"last"`
		Items []syncsource.Item  `json:"items"`
	}
	f.c.expect(f.c.get("/api/projects/"+f.p.ID+"/sync"), 200).JSON(&view)
	if view.Last == nil || view.Last.SyncID != again.SyncID || len(view.Items) != 4 {
		t.Fatalf("%+v", view)
	}
	f.c.expect(f.c.do("PUT", "/api/projects/"+f.p.ID+"/sync/items/1/priority", map[string]any{"priority": 4}, nil), 200)
	if resp := f.c.do("PUT", "/api/projects/"+f.p.ID+"/sync/items/1/priority", map[string]any{"priority": 7}, nil); resp.StatusCode != 400 && resp.StatusCode != 422 {
		t.Fatalf("faixa: %d", resp.StatusCode)
	}
	if resp := f.c.post("/api/projects/"+f.p.ID+"/sync/items/1/resolve", map[string]any{"choice": "keep_local"}); resp.StatusCode != 409 {
		t.Fatalf("sem conflito: %d", resp.StatusCode)
	}
	if resp := f.c.do("PUT", "/api/projects/"+f.p.ID+"/sync/items/zzz/priority", map[string]any{"priority": 1}, nil); resp.StatusCode != 404 {
		t.Fatalf("item inexistente: %d", resp.StatusCode)
	}
	// Projeto sem origem completa.
	p2 := createProject(t, f.c, "sync-api-vazio")
	if resp := f.c.post("/api/projects/"+p2.ID+"/sync", nil); resp.StatusCode != 422 {
		t.Fatalf("origem incompleta: %d", resp.StatusCode)
	}
	var _ = projects.Project{}
}
