package guidance

import (
	"context"
	"sync"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
)

// Live é um executor em andamento que aceita orientação: as capacidades que ele declara e como entregar.
type Live struct {
	Caps      executor.Capabilities
	Deliverer Deliverer
}

// Registry guarda os executores vivos por execução. Quem inicia o executor o registra e remove ao terminar;
// sem registro a execução não aceita entrega agora (a orientação fica registrada para a retomada).
type Registry struct {
	mu   sync.RWMutex
	live map[string]Live
}

// Register marca a execução como viva.
func (r *Registry) Register(executionID string, l Live) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.live == nil {
		r.live = map[string]Live{}
	}
	r.live[executionID] = l
}

// Unregister remove a execução (terminou ou caiu).
func (r *Registry) Unregister(executionID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.live, executionID)
}

// Lookup devolve o executor vivo, se houver.
func (r *Registry) Lookup(executionID string) (Live, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	l, ok := r.live[executionID]
	return l, ok
}

// Resolve devolve o entregador e as capacidades para enviar: o vivo, ou um "nenhum" que só registra.
func (r *Registry) Resolve(executionID string) (Deliverer, executor.Capabilities) {
	if l, ok := r.Lookup(executionID); ok {
		return l.Deliverer, l.Caps
	}
	return noDeliverer{}, executor.Capabilities{Interaction: executor.InteractionNone}
}

type noDeliverer struct{}

func (noDeliverer) Deliver(context.Context, string, string) (Confirmation, error) {
	return Unconfirmed, nil
}
