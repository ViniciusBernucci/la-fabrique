package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// T16-05: eventos operacionais da fábrica (jobs/rotinas de fundo do worker), fora do escopo por
// projeto já coberto por live.go (execution_events). Mesmo padrão de cursor/SSE/redação.

var operationalLevels = map[string]bool{"info": true, "warn": true, "error": true}

var operationalSources = map[string]bool{
	"sentinela": true, "admissao": true, "retencao": true, "sincronizacao": true, "clilogin": true, "fabrica": true,
}

type operationalEventView struct {
	Seq       int64           `json:"seq"`
	Level     string          `json:"level"`
	Source    string          `json:"source"`
	Message   string          `json:"message"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

// fetchOperationalEvents lê eventos após o cursor, com filtro opcional por nível, origem e busca.
func (s *Server) fetchOperationalEvents(ctx context.Context, after int64, level, source, q string, limit int) ([]operationalEventView, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(ctx, `SELECT id, level, source, message, payload, created_at FROM operational_events
		WHERE id > $1 AND ($2 = '' OR level = $2) AND ($3 = '' OR source = $3)
		AND ($4 = '' OR message ILIKE '%' || $4 || '%' OR payload::text ILIKE '%' || $4 || '%')
		ORDER BY id LIMIT $5`, after, level, source, q, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []operationalEventView{}
	for rows.Next() {
		var e operationalEventView
		var payload []byte
		if err := rows.Scan(&e.Seq, &e.Level, &e.Source, &e.Message, &payload, &e.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		e.Payload = json.RawMessage(s.redactor.RedactBytes(payload))
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}

// operationsEvents é o histórico pesquisável: eventos após o cursor, com `level`, `source` e `q`.
func (s *Server) operationsEvents(w http.ResponseWriter, r *http.Request) {
	after, valid := cursorOf(r)
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Cursor inválido.", "Use o número do último evento recebido.")
		return
	}
	level := strings.TrimSpace(r.URL.Query().Get("level"))
	if level != "" && !operationalLevels[level] {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Nível inválido.", "Use info, warn ou error.")
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if source != "" && !operationalSources[source] {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Origem inválida.", "Use sentinela, admissao, retencao, sincronizacao, clilogin ou fabrica.")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 100 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "A busca é longa demais.", "Use até 100 caracteres.")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	evs, err := s.fetchOperationalEvents(r.Context(), after, level, source, q, limit)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	next := after
	if len(evs) > 0 {
		next = evs[len(evs)-1].Seq
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": evs, "next_cursor": next})
}

// operationsStream envia os eventos operacionais ao vivo (SSE), sem filtro no servidor — mesmo
// padrão de reconexão por cursor de executionStream (live.go).
func (s *Server) operationsStream(w http.ResponseWriter, r *http.Request) {
	after, valid := cursorOf(r)
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Cursor inválido.", "Use o número do último evento recebido.")
		return
	}
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{})
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	bw := bufio.NewWriter(w)
	flush := func() error {
		if err := bw.Flush(); err != nil {
			return err
		}
		return rc.Flush()
	}
	fmt.Fprint(bw, "retry: 3000\n\n")
	if flush() != nil {
		return
	}
	poll := time.NewTicker(StreamPoll)
	defer poll.Stop()
	beat := time.NewTicker(StreamHeartbeat)
	defer beat.Stop()
	deadline := time.NewTimer(StreamMaxAge)
	defer deadline.Stop()
	send := func() bool {
		for {
			evs, err := s.fetchOperationalEvents(r.Context(), after, "", "", "", 200)
			if err != nil {
				return false
			}
			for _, e := range evs {
				data, _ := json.Marshal(e)
				fmt.Fprintf(bw, "id: %d\ndata: %s\n\n", e.Seq, data)
				after = e.Seq
			}
			if flush() != nil {
				return false
			}
			if len(evs) < 200 {
				return true
			}
		}
	}
	if !send() {
		return
	}
	for {
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			fmt.Fprint(bw, "event: reconnect\ndata: {}\n\n")
			_ = flush()
			return
		case <-beat.C:
			fmt.Fprint(bw, ": ping\n\n")
			if flush() != nil {
				return
			}
		case <-poll.C:
			if !send() {
				return
			}
		}
	}
}
