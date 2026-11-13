package guidance

import (
	"context"
	"errors"
	"time"
)

// ErrNoAck: o laço do executor não confirmou a mensagem a tempo (a entrega fica incerta).
var ErrNoAck = errors.New("o executor não confirmou o recebimento")

// Message é uma orientação entregue ao laço do executor.
type Message struct {
	Text string
	ack  chan bool
}

// Ack confirma que o executor RECEBEU a orientação (a entregou ao cliente/modelo). Sem Ack a entrega fica incerta.
func (m Message) Ack() {
	select {
	case m.ack <- true:
	default:
	}
}

// Inbox liga o serviço de orientações ao laço do executor em andamento (CLI com stdin seguro ou API): o laço
// chama Next nos pontos em que aceita entrada e Ack depois de entregar. Implementa Deliverer.
type Inbox struct {
	ch      chan Message
	Timeout time.Duration // quanto esperar o Ack (padrão 10 s)
}

// NewInbox cria a caixa de entrada.
func NewInbox() *Inbox { return &Inbox{ch: make(chan Message, 16)} }

// Deliver entrega ao laço e espera o Ack: com Ack é Confirmed; sem Ack no prazo é erro (entrega incerta).
func (i *Inbox) Deliver(ctx context.Context, _ string, text string) (Confirmation, error) {
	m := Message{Text: text, ack: make(chan bool, 1)}
	select {
	case i.ch <- m:
	case <-ctx.Done():
		return Unconfirmed, ctx.Err()
	default:
		return Unconfirmed, ErrNoAck // fila cheia: não há como garantir a entrega
	}
	wait := i.Timeout
	if wait <= 0 {
		wait = 10 * time.Second
	}
	select {
	case <-m.ack:
		return Confirmed, nil
	case <-time.After(wait):
		return Unconfirmed, ErrNoAck
	case <-ctx.Done():
		return Unconfirmed, ctx.Err()
	}
}

// Next devolve a próxima orientação, se houver (não bloqueia).
func (i *Inbox) Next() (Message, bool) {
	select {
	case m := <-i.ch:
		return m, true
	default:
		return Message{}, false
	}
}
