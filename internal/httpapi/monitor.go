package httpapi

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/monitor"
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
