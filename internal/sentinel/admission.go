package sentinel

import (
	"sort"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/flow"
)

// Tipos de trabalho admitido.
const (
	WorkPlanning       = "planning"
	WorkImplementation = "implementation"
	WorkReview         = "review"
)

// Motivos de espera.
const (
	WaitDependency = "dependency"
	WaitPaused     = "paused"
	WaitWorkflow   = "workflow_not_published"
	WaitBudget     = "budget"
	WaitAuth       = "authorization"
	WaitCapacity   = "global_capacity"
	WaitSlot       = "track_slot"
	WaitPlanning   = "planning_limit"
	WaitResource   = "resource_conflict"
)

// Candidate é um trabalho elegível a admissão.
type Candidate struct {
	ID         string
	Work       string     // planning | implementation | review
	Track      flow.Track // vazio para planejamento
	NeedsBoth  bool       // acesso exclusivo às duas trilhas (adquire ambas atomicamente)
	Correction bool       // correção liberada: categoria mais prioritária
	Priority   int        // oficial: 0 é o mais urgente
	EnqueuedAt time.Time  // entrada na fila vigente (antiguidade)
	DependsOn  []string
	Resources  []string // arquivos/schema/recursos compartilhados reservados com exclusividade

	Paused, WorkflowPublished, BudgetOK, Authorized bool
}

// Running é um trabalho em execução.
type Running struct {
	ID        string
	Work      string
	Track     flow.Track
	NeedsBoth bool
	Resources []string
}

// Limits são os limites de admissão do projeto e globais.
type Limits struct {
	PlanningSlots  int // limite próprio de planejamento
	GlobalCapacity int // capacidade global restante da VPS (já inclui o que roda)
}

// Decision é o resultado para um candidato.
type Decision struct {
	ID       string `json:"id"`
	Admitted bool   `json:"admitted"`
	Wait     string `json:"wait,omitempty"`
	Position int    `json:"position"` // ordem na fila vigente (1 = primeiro)
}

func slotsOf(work string, track flow.Track, both bool) []string {
	if work == WorkPlanning {
		return []string{"PLANNING"}
	}
	if both {
		return []string{string(flow.Frontend), string(flow.Backend)}
	}
	return []string{string(track)}
}

// Order devolve os candidatos na ordem de despacho: correção liberada, prioridade oficial (0 primeiro),
// antiguidade na fila vigente e, em empate exato, o ID estável.
func Order(cs []Candidate) []Candidate {
	out := append([]Candidate{}, cs...)
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Correction != b.Correction {
			return a.Correction
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		if !a.EnqueuedAt.Equal(b.EnqueuedAt) {
			return a.EnqueuedAt.Before(b.EnqueuedAt)
		}
		return a.ID < b.ID
	})
	return out
}

// Admit decide quem entra agora. Cada candidato é avaliado pela ordem; um bloqueado (dependência,
// recurso, vaga) não impede o trabalho independente que vem depois. Implementação e review da
// mesma trilha compartilham a vaga; planejamento tem limite próprio; todos consomem capacidade global.
// Prioridade vale só para a próxima admissão: nada em execução é preemptado.
func Admit(cs []Candidate, running []Running, done map[string]bool, lim Limits) []Decision {
	busy := map[string]int{} // vagas ocupadas
	planning := 0
	res := map[string]bool{} // recursos reservados
	capLeft := lim.GlobalCapacity
	for _, r := range running {
		for _, s := range slotsOf(r.Work, r.Track, r.NeedsBoth) {
			busy[s]++
		}
		if r.Work == WorkPlanning {
			planning++
		}
		for _, x := range r.Resources {
			res[x] = true
		}
	}
	ordered := Order(cs)
	out := make([]Decision, 0, len(ordered))
	for i, c := range ordered {
		d := Decision{ID: c.ID, Position: i + 1}
		wait := func(w string) { d.Wait = w; out = append(out, d) }
		switch {
		case c.Paused:
			wait(WaitPaused)
			continue
		case !c.WorkflowPublished:
			wait(WaitWorkflow)
			continue
		case !c.BudgetOK:
			wait(WaitBudget)
			continue
		case !c.Authorized:
			wait(WaitAuth)
			continue
		}
		ready := true
		for _, dep := range c.DependsOn {
			if !done[dep] {
				ready = false
			}
		}
		if !ready {
			wait(WaitDependency)
			continue
		}
		if capLeft <= 0 {
			wait(WaitCapacity)
			continue
		}
		if c.Work == WorkPlanning {
			if planning >= lim.PlanningSlots {
				wait(WaitPlanning)
				continue
			}
		} else {
			free := true
			for _, s := range slotsOf(c.Work, c.Track, c.NeedsBoth) {
				if busy[s] > 0 {
					free = false
				}
			}
			if !free {
				wait(WaitSlot)
				continue
			}
		}
		conflict := false
		for _, x := range c.Resources {
			if res[x] {
				conflict = true
			}
		}
		if conflict {
			wait(WaitResource)
			continue
		}
		// admitido: reserva vagas, recursos e capacidade
		for _, s := range slotsOf(c.Work, c.Track, c.NeedsBoth) {
			busy[s]++
		}
		if c.Work == WorkPlanning {
			planning++
		}
		for _, x := range c.Resources {
			res[x] = true
		}
		capLeft--
		d.Admitted = true
		out = append(out, d)
	}
	return out
}
