package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/executor"
	"github.com/ViniciusBernucci/le-fabrique/internal/guidance"
)

// AJ07-03 — o entregador ligado ao laço do executor: com Ack vira `acknowledged`; sem Ack no prazo, `uncertain`;
// o registro de executores vivos decide entre entregar agora e só registrar.
func TestGuidanceInboxConnectsTheRunningExecutorLoop(t *testing.T) {
	db, project, exec := monitorExec(t)
	ctx := context.Background()
	svc := &guidance.Service{DB: db.Pool}
	reg := &guidance.Registry{}
	inbox := guidance.NewInbox()
	inbox.Timeout = 300 * time.Millisecond
	reg.Register(exec, guidance.Live{Caps: executor.Capabilities{Interaction: executor.InteractionImmediate}, Deliverer: inbox})

	// O laço do executor consome e confirma.
	done := make(chan string, 1)
	go func() {
		for i := 0; i < 100; i++ {
			if m, ok := inbox.Next(); ok {
				done <- m.Text
				m.Ack()
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()
	d, caps := reg.Resolve(exec)
	it, err := svc.Send(ctx, d, caps, project, exec, "admin", "Prefira o pacote existente", false)
	if err != nil || it.State != guidance.Acknowledged {
		t.Fatalf("%+v %v", it, err)
	}
	if got := <-done; got == "" {
		t.Fatal("o laço deveria ter recebido o envelope")
	}
	// Ninguém consome: sem Ack a entrega é incerta, não "entregue".
	it2, err := svc.Send(ctx, d, caps, project, exec, "admin", "Segunda", false)
	if err != nil || it2.State != guidance.Uncertain {
		t.Fatalf("%+v %v", it2, err)
	}
	// Executor encerrado: volta a só registrar para a retomada.
	reg.Unregister(exec)
	d, caps = reg.Resolve(exec)
	it3, _ := svc.Send(ctx, d, caps, project, exec, "admin", "Terceira", false)
	if it3.State != guidance.Queued || it3.Mode != "none" {
		t.Fatalf("%+v", it3)
	}
}
