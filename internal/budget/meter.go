package budget

import (
	"context"
	"fmt"

	"github.com/ViniciusBernucci/le-fabrique/internal/gateways"
	"github.com/ViniciusBernucci/le-fabrique/internal/models"
)

// Meter liga o orçamento ao laço do gateway (gateways.Meter).
//
// Conta de API: reserva custo estimado (preço do modelo × tokens estimados de
// entrada + teto de saída) antes de cada chamada e liquida com o consumo
// informado pelo gateway; sem consumo informado, liquida pelo valor reservado
// (fonte "estimated", pessimista). Conta de assinatura: não há custo nem teto
// em dinheiro; cada chamada é registrada como cota (unidade "calls"), em
// separado do gasto em API.
type Meter struct {
	Store        *Store
	AgentID      string
	ConnectionID string
	Subscription bool
}

type handle struct {
	res   Reservation
	price Price
	model string
}

// Reserve implementa gateways.Meter.
func (m Meter) Reserve(ctx context.Context, model models.Model, msgs []gateways.Message, maxOutput int64) (gateways.Handle, error) {
	if m.Subscription {
		return handle{model: model.Name}, nil
	}
	price, err := PriceOf(model)
	if err != nil {
		return nil, err
	}
	texts := make([]string, 0, len(msgs))
	for _, ms := range msgs {
		texts = append(texts, ms.Content)
		for _, tc := range ms.ToolCalls {
			texts = append(texts, tc.Name, fmt.Sprint(tc.Args))
		}
	}
	in := EstimateTokens(texts...) + 16*int64(len(msgs))
	res, err := m.Store.Reserve(ctx, ReserveRequest{AgentID: m.AgentID, ConnectionID: m.ConnectionID, AmountMicros: CostOf(price, in, maxOutput)})
	if err != nil {
		return nil, err
	}
	return handle{res: res, price: price, model: model.Name}, nil
}

// Settle implementa gateways.Meter.
func (m Meter) Settle(ctx context.Context, h gateways.Handle, u gateways.Usage) error {
	hd, ok := h.(handle)
	if !ok {
		return fmt.Errorf("reserva desconhecida")
	}
	if m.Subscription {
		src := SourceProvider
		if u.PromptTokens == 0 && u.CompletionTokens == 0 {
			src = SourceEstimated
		}
		_, err := m.Store.RecordSubscription(ctx, SubscriptionUse{AgentID: m.AgentID, ConnectionID: m.ConnectionID, Model: hd.model, Units: 1, Unit: "calls",
			InputTokens: int64(u.PromptTokens), OutputTokens: int64(u.CompletionTokens), Source: src})
		return err
	}
	a := Actual{Model: hd.model, InputTokens: int64(u.PromptTokens), OutputTokens: int64(u.CompletionTokens), Source: SourceProvider}
	if u.PromptTokens == 0 && u.CompletionTokens == 0 {
		a.Source, a.CostMicros = SourceEstimated, hd.res.AmountMicros
	} else {
		a.CostMicros = CostOf(hd.price, a.InputTokens, a.OutputTokens)
	}
	_, err := m.Store.Settle(ctx, hd.res.ID, a)
	return err
}

// Release implementa gateways.Meter.
func (m Meter) Release(ctx context.Context, h gateways.Handle) {
	if hd, ok := h.(handle); ok && hd.res.ID != "" {
		_ = m.Store.Release(ctx, hd.res.ID)
	}
}
