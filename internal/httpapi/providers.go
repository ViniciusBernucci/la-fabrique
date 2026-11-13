package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/availability"
	"github.com/ViniciusBernucci/le-fabrique/internal/budget"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

type quotaView struct {
	Used       int64      `json:"used"`
	Limit      int64      `json:"limit"`
	Unit       string     `json:"unit"`
	ResetsAt   *time.Time `json:"resets_at"`
	Source     string     `json:"source"`
	ObservedAt time.Time  `json:"observed_at"`
}

type providerView struct {
	ConnectionID string              `json:"connection_id"`
	Name         string              `json:"name"`
	Kind         string              `json:"kind"` // cli (assinatura) | api
	Target       string              `json:"target"`
	State        string              `json:"state"` // active | revoked (da conexão)
	Health       availability.Health `json:"health"`
	Quota        *quotaView          `json:"quota"` // nil = cota não observada
}

// listProviders mostra, por conta de IA (assinatura/CLI e API), a disponibilidade
// (sinal oficial + histórico) e a última cota OBSERVADA. Sem observação, `quota` é
// null: o painel diz "não observada", nunca inventa um valor. Sem segredos.
func (s *Server) listProviders(w http.ResponseWriter, r *http.Request) {
	conns, err := s.vault.List(r.Context())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	tracker := &availability.Tracker{DB: s.db}
	bs := &budget.Store{DB: s.db}
	items := []providerView{}
	for _, c := range conns {
		if c.Kind != "cli" && c.Kind != "api" {
			continue
		}
		h, err := tracker.Get(r.Context(), c.ID)
		if err != nil {
			s.writeDomainError(w, r, err)
			return
		}
		v := providerView{ConnectionID: c.ID, Name: c.Name, Kind: c.Kind, Target: c.Target, State: c.State, Health: h}
		q, err := bs.LatestQuota(r.Context(), c.ID)
		switch {
		case err == nil:
			v.Quota = &quotaView{Used: q.Used, Limit: q.Limit, Unit: q.Unit, ResetsAt: q.ResetsAt, Source: q.Source, ObservedAt: q.ObservedAt}
		case !errors.Is(err, persistence.ErrNotFound):
			s.writeDomainError(w, r, err)
			return
		}
		items = append(items, v)
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
