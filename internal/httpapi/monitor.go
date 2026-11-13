package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/guidance"
	"github.com/ViniciusBernucci/le-fabrique/internal/monitor"
	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// executionMonitor devolve os eventos do agente (CLI e API na mesma forma) após o cursor e o resumo da
// execução: cronômetro, ferramentas, subagentes e progresso informado pelo executor (nunca estimado).
func (s *Server) executionMonitor(w http.ResponseWriter, r *http.Request) {
	project, execution, ok := s.executionOf(w, r)
	if !ok {
		return
	}
	after, valid := cursorOf(r)
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Cursor inválido.", "Use o número do último evento recebido.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	v, err := monitor.Read(r.Context(), s.db, project, execution, after, limit, time.Now().UTC())
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	for i := range v.Events {
		v.Events[i].Text = s.redactor.Redact(v.Events[i].Text)
	}
	writeJSON(w, http.StatusOK, v)
}

// GuidanceRegistry expõe o registro de executores vivos para quem inicia as execuções (o worker ou o
// orquestrador no mesmo processo registra o entregador do executor; sem registro a orientação só fica gravada).
func (s *Server) GuidanceRegistry() *guidance.Registry { return &s.guidanceLive }

func (s *Server) guidanceService() *guidance.Service { return &guidance.Service{DB: s.db} }

// listGuidance devolve o histórico de orientações da execução, com o estado comprovado de cada entrega.
func (s *Server) listGuidance(w http.ResponseWriter, r *http.Request) {
	project, execution, ok := s.executionOf(w, r)
	if !ok {
		return
	}
	items, err := s.guidanceService().List(r.Context(), project, execution)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	if items == nil {
		items = []guidance.Item{}
	}
	_, live := s.guidanceLive.Lookup(execution)
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "live": live})
}

// sendGuidance registra a orientação (autoria da sessão) e a entrega conforme a capacidade do executor.
// O texto é só orientação: não altera contrato, permissões nem aprovações.
func (s *Server) sendGuidance(w http.ResponseWriter, r *http.Request) {
	project, execution, ok := s.executionOf(w, r)
	if !ok {
		return
	}
	sess, _ := sessionFrom(r.Context())
	var in struct {
		Text string `json:"text"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	var state string
	if err := s.db.QueryRow(r.Context(), `SELECT state FROM executions WHERE id = $1`, execution).Scan(&state); err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	ended := state == "succeeded" || state == "failed" || state == "cancelled"
	d, caps := s.guidanceLive.Resolve(execution)
	it, err := s.guidanceService().Send(r.Context(), d, caps, project, execution, sess.Username, in.Text, ended)
	switch {
	case errors.Is(err, guidance.ErrRejected):
		writeError(w, r, http.StatusUnprocessableEntity, "guidance_rejected", "A orientação foi recusada: "+strings.TrimPrefix(err.Error(), guidance.ErrRejected.Error()+": "), "Escreva só a orientação, sem imitar o contrato nem comandos da fábrica.")
		return
	case errors.Is(err, guidance.ErrEnded):
		writeError(w, r, http.StatusConflict, "execution_ended", "A execução já terminou.", "Registre a orientação na retomada.")
		return
	case err != nil:
		s.writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}
