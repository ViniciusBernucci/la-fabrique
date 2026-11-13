package tasksync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/effects"
	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Estados da fábrica que viram estados da origem (via StatusMap).
const (
	StateInProgress    = "in_progress"
	StateAwaitingMerge = "awaiting_merge"
	StateDone          = "done"
)

// Erros com resultado definido.
var (
	ErrUnmapped          = errors.New("estado da fábrica sem mapa para a origem")
	ErrMergeNotConfirmed = errors.New("merge não confirmado: a tarefa não pode ser concluída")
	ErrPRClosed          = errors.New("a PR foi fechada sem merge: é preciso uma decisão")
	ErrNoVerifier        = errors.New("sem verificador de merge configurado")
)

// MergeState é o que se sabe da PR no GitHub.
type MergeState struct {
	Merged bool
	Closed bool
	SHA    string // commit de integração (merge confirmado)
}

// MergeVerifier consulta a PR no repositório (adaptador GitHub, T15-03).
type MergeVerifier interface {
	Merged(ctx context.Context, prURL string) (MergeState, error)
}

// Publisher escreve na origem pelo ledger.
type Publisher struct {
	Tracker integrations.Tracker
	Ledger  *effects.Ledger
	// StatusMap traduz estados da fábrica em nomes de estado da origem.
	StatusMap map[string]string
	Merges    MergeVerifier
}

// Marker é o marcador, embutido na descrição, que liga um item remoto à chave de
// negócio: serve de prova de existência em destinos sem chave de idempotência.
func Marker(businessKey string) string {
	sum := sha256.Sum256([]byte(businessKey))
	return "[lf:key=" + hex.EncodeToString(sum[:])[:16] + "]"
}

func (p *Publisher) run(ctx context.Context, project, key, op string, req any, ad effects.Adapter) (effects.Result, error) {
	return p.Ledger.Run(ctx, effects.Spec{ProjectID: project, BusinessKey: key, Operation: op, Request: req}, ad)
}

func inconclusive(format string, a ...any) effects.Verdict {
	return effects.Verdict{Status: effects.VerdictInconclusive, Evidence: fmt.Sprintf(format, a...)}
}

// --- subtarefa ---------------------------------------------------------------

type subtaskAdapter struct {
	t             integrations.Tracker
	parent, title string
	description   string
	marker        string
}

func refused(err error) error {
	var e *integrations.Error
	if errors.As(err, &e) && (e.Kind == integrations.ErrPermission || e.Kind == integrations.ErrAuth || e.Kind == integrations.ErrInvalid) {
		return fmt.Errorf("%w: %v", effects.ErrNotApplied, err)
	}
	return err
}

func (a subtaskAdapter) Apply(ctx context.Context, op effects.Op) (any, error) {
	desc := strings.TrimSpace(a.description)
	if desc != "" {
		desc += "\n\n"
	}
	it, err := a.t.CreateSubtask(ctx, a.parent, integrations.SubtaskInput{Title: a.title, Description: desc + a.marker})
	if err != nil {
		return nil, refused(err)
	}
	return map[string]any{"remote_id": it.ID, "remote_key": it.Key}, nil
}

func (a subtaskAdapter) Verify(ctx context.Context, op effects.Op) (effects.Verdict, error) {
	kids, err := a.t.Children(ctx, a.parent)
	if err != nil {
		return inconclusive("não foi possível listar as subtarefas de %s: %v", a.parent, err), nil
	}
	var found []integrations.Item
	for _, k := range kids {
		if strings.Contains(k.Description, a.marker) {
			found = append(found, k)
		}
	}
	if len(found) == 0 {
		return effects.Verdict{Status: effects.VerdictAbsent}, nil
	}
	return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{
		"remote_id": found[0].ID, "remote_key": found[0].Key, "matches": len(found)}}, nil
}

// CreateSubtask cria a subtarefa uma única vez. keyBase identifica a tarefa técnica
// de forma estável (ex.: projeto:feature:tentativa:tarefa-local).
func (p *Publisher) CreateSubtask(ctx context.Context, project, keyBase, parentID, title, description string) (effects.Result, error) {
	key := keyBase + ":subtask"
	ad := subtaskAdapter{t: p.Tracker, parent: parentID, title: title, description: description, marker: Marker(key)}
	return p.run(ctx, project, key, "create_subtask", map[string]any{"parent": parentID, "title": title}, ad)
}

// --- dependência -------------------------------------------------------------

type dependencyAdapter struct {
	t               integrations.Tracker
	item, dependsOn string
}

func (a dependencyAdapter) Apply(ctx context.Context, _ effects.Op) (any, error) {
	if err := a.t.AddDependency(ctx, a.item, a.dependsOn); err != nil {
		return nil, refused(err)
	}
	return map[string]any{"item": a.item, "depends_on": a.dependsOn}, nil
}

func (a dependencyAdapter) Verify(ctx context.Context, _ effects.Op) (effects.Verdict, error) {
	it, err := a.t.GetItem(ctx, a.item)
	if err != nil {
		return inconclusive("não foi possível ler %s: %v", a.item, err), nil
	}
	dep, err := a.t.GetItem(ctx, a.dependsOn)
	if err != nil {
		return inconclusive("não foi possível ler %s: %v", a.dependsOn, err), nil
	}
	for _, d := range it.Dependencies {
		if d == dep.ID || d == dep.Key || d == a.dependsOn {
			return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"item": a.item, "depends_on": a.dependsOn}}, nil
		}
	}
	return effects.Verdict{Status: effects.VerdictAbsent}, nil
}

// AddDependency registra, uma vez, que itemID depende de dependsOnID.
func (p *Publisher) AddDependency(ctx context.Context, project, keyBase, itemID, dependsOnID string) (effects.Result, error) {
	key := keyBase + ":dependency:" + itemID + ">" + dependsOnID
	return p.run(ctx, project, key, "add_dependency", map[string]any{"item": itemID, "depends_on": dependsOnID},
		dependencyAdapter{t: p.Tracker, item: itemID, dependsOn: dependsOnID})
}

// --- link --------------------------------------------------------------------

type linkAdapter struct {
	t                integrations.Tracker
	item, url, title string
}

func (a linkAdapter) Apply(ctx context.Context, _ effects.Op) (any, error) {
	if err := a.t.AddLink(ctx, a.item, a.url, a.title); err != nil {
		return nil, refused(err)
	}
	return map[string]any{"item": a.item, "url": a.url}, nil
}

func (a linkAdapter) Verify(ctx context.Context, _ effects.Op) (effects.Verdict, error) {
	ok, err := a.t.HasLink(ctx, a.item, a.url)
	if err != nil {
		return inconclusive("não foi possível listar os links de %s: %v", a.item, err), nil
	}
	if ok {
		return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"item": a.item, "url": a.url}}, nil
	}
	return effects.Verdict{Status: effects.VerdictAbsent}, nil
}

// AddLink anexa o link (ex.: Draft PR) ao item, uma vez.
func (p *Publisher) AddLink(ctx context.Context, project, keyBase, itemID, linkURL, title string) (effects.Result, error) {
	key := keyBase + ":link:" + linkURL
	return p.run(ctx, project, key, "add_link", map[string]any{"item": itemID, "url": linkURL},
		linkAdapter{t: p.Tracker, item: itemID, url: linkURL, title: title})
}

// --- estado ------------------------------------------------------------------

type transitionAdapter struct {
	t          integrations.Tracker
	item, want string
}

// Apply revalida o alvo antes de escrever: se o item já está no estado desejado,
// não escreve; se não há transição válida para ele agora, recusa sem efeito.
func (a transitionAdapter) Apply(ctx context.Context, _ effects.Op) (any, error) {
	cur, err := a.t.GetItem(ctx, a.item)
	if err != nil {
		return nil, refused(err)
	}
	if cur.Status == a.want {
		return map[string]any{"status": a.want, "already": true}, nil
	}
	ts, err := a.t.Transitions(ctx, a.item)
	if err != nil {
		return nil, refused(err)
	}
	for _, tr := range ts {
		if tr.ToStatus == a.want {
			if err := a.t.Transition(ctx, a.item, tr.ID); err != nil {
				return nil, refused(err)
			}
			return map[string]any{"status": a.want, "from": cur.Status}, nil
		}
	}
	return nil, fmt.Errorf("%w: não há transição de %q para %q neste momento", effects.ErrNotApplied, cur.Status, a.want)
}

func (a transitionAdapter) Verify(ctx context.Context, _ effects.Op) (effects.Verdict, error) {
	cur, err := a.t.GetItem(ctx, a.item)
	if err != nil {
		return inconclusive("não foi possível ler %s: %v", a.item, err), nil
	}
	if cur.Status == a.want {
		return effects.Verdict{Status: effects.VerdictConfirmed, Proof: map[string]any{"status": cur.Status}}, nil
	}
	return effects.Verdict{Status: effects.VerdictAbsent}, nil
}

// MoveTo leva o item ao estado da origem mapeado para o estado da fábrica.
// A chave de negócio inclui o estado: repetir o mesmo movimento é idempotente;
// um novo ciclo legítimo precisa de outro keyBase (ex.: outra tentativa).
func (p *Publisher) MoveTo(ctx context.Context, project, keyBase, itemID, factoryState string, extra map[string]any) (effects.Result, error) {
	want, ok := p.StatusMap[factoryState]
	if !ok || want == "" {
		return effects.Result{}, fmt.Errorf("%w: %q", ErrUnmapped, factoryState)
	}
	key := keyBase + ":state:" + factoryState
	req := map[string]any{"item": itemID, "state": factoryState, "status": want}
	for k, v := range extra {
		req[k] = v
	}
	return p.run(ctx, project, key, "transition", req, transitionAdapter{t: p.Tracker, item: itemID, want: want})
}

// PublishDraftPR anexa a Draft PR ao item e o leva a "aguardando merge". NÃO conclui.
func (p *Publisher) PublishDraftPR(ctx context.Context, project, keyBase, itemID, prURL string) error {
	if _, err := p.AddLink(ctx, project, keyBase, itemID, prURL, "Draft PR"); err != nil {
		return fmt.Errorf("link da PR: %w", err)
	}
	if _, err := p.MoveTo(ctx, project, keyBase, itemID, StateAwaitingMerge, nil); err != nil {
		return fmt.Errorf("estado aguardando merge: %w", err)
	}
	return nil
}

// Complete conclui o item SOMENTE com o merge confirmado no repositório. PR fechada
// sem merge devolve ErrPRClosed (o chamador abre uma decisão); sem prova de merge,
// ErrMergeNotConfirmed. Em ambos nada é escrito na origem.
func (p *Publisher) Complete(ctx context.Context, project, keyBase, itemID, prURL string) (effects.Result, error) {
	if p.Merges == nil {
		return effects.Result{}, ErrNoVerifier
	}
	st, err := p.Merges.Merged(ctx, prURL)
	if err != nil {
		return effects.Result{}, fmt.Errorf("%w: %v", ErrMergeNotConfirmed, err)
	}
	switch {
	case st.Closed && !st.Merged:
		return effects.Result{}, ErrPRClosed
	case !st.Merged || strings.TrimSpace(st.SHA) == "":
		return effects.Result{}, ErrMergeNotConfirmed
	}
	return p.MoveTo(ctx, project, keyBase, itemID, StateDone, map[string]any{"merge_sha": st.SHA, "pr": prURL})
}

// RemoteIDFor devolve o ID remoto associado a uma chave de negócio já confirmada
// (a prova do ledger guarda o `remote_id`). Vazio se ainda não existe.
func (p *Publisher) RemoteIDFor(ctx context.Context, project, businessKey string) (string, error) {
	var proof []byte
	err := p.Ledger.DB.QueryRow(ctx, `SELECT proof FROM outbox WHERE project_id = $1 AND business_key = $2 AND state = 'confirmed'`, project, businessKey).Scan(&proof)
	if errors.Is(persistence.MapError(err), persistence.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", persistence.MapError(err)
	}
	var m map[string]any
	_ = json.Unmarshal(proof, &m)
	id, _ := m["remote_id"].(string)
	return id, nil
}
