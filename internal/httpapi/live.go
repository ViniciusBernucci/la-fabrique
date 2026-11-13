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
	"github.com/ViniciusBernucci/le-fabrique/internal/projects"
	"github.com/ViniciusBernucci/le-fabrique/internal/queue"
	"github.com/ViniciusBernucci/le-fabrique/internal/syncsource"
)

// Parâmetros do fluxo ao vivo (variáveis para os testes).
var (
	StreamPoll      = time.Second
	StreamHeartbeat = 15 * time.Second
	StreamMaxAge    = 5 * time.Minute
)

func (s *Server) queueStore() *queue.Queue { return &queue.Queue{DB: s.db} }

type itemView struct {
	ID          string     `json:"id"`
	FeatureID   string     `json:"feature_id"`
	TaskKey     string     `json:"task_key"`
	Kind        string     `json:"kind"`
	Priority    int        `json:"priority"`
	State       string     `json:"state"`
	WaitReason  string     `json:"wait_reason,omitempty"`
	WaitRef     string     `json:"wait_ref,omitempty"`
	Blocked     string     `json:"blocked,omitempty"` // causa em linguagem do painel
	ExecutionID string     `json:"execution_id,omitempty"`
	EnqueuedAt  time.Time  `json:"enqueued_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	Seconds     int64      `json:"seconds"` // tempo desde o início (ou até o fim)
}

type executionView struct {
	ID        string `json:"id"`
	FeatureID string `json:"feature_id"`
	State     string `json:"state"`
	Attempt   int    `json:"attempt"`
	LastSeq   int64  `json:"last_seq"`
}

func waitText(it queue.Item) string {
	switch it.WaitReason {
	case queue.WaitMerge:
		return "Aguardando o merge da PR " + it.WaitRef
	case queue.WaitProvider:
		return "Aguardando a disponibilidade do provedor de IA"
	case queue.WaitDecision:
		return "Aguardando uma decisão humana na Central de Decisões"
	case queue.WaitResources:
		return "Aguardando recursos livres"
	case queue.WaitSuspended:
		return "Suspensa pelo controle operacional; retome o projeto para continuar"
	}
	return ""
}

// projectOverview resume o projeto para o painel: estado, controles persistidos (global e do
// projeto), contagem e lista de tarefas da fila com a CAUSA de cada bloqueio, execuções com o
// cursor do último evento e a última sincronização.
func (s *Server) projectOverview(w http.ResponseWriter, r *http.Request) {
	id, ok := s.projectID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	p, err := projects.Get(ctx, s.db, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	q := s.queueStore()
	global, err := q.ControlOf(ctx, queue.Scope{Global: true})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	proj, err := q.ControlOf(ctx, queue.Scope{ProjectID: id})
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	items, err := q.List(ctx, id)
	if err != nil {
		s.writeDomainError(w, r, err)
		return
	}
	counts := map[string]int{"queued": 0, "running": 0, "waiting": 0, "done": 0, "cancelled": 0}
	views := make([]itemView, 0, len(items))
	now := time.Now().UTC()
	for _, it := range items {
		counts[it.State]++
		v := itemView{ID: it.ID, FeatureID: it.FeatureID, TaskKey: it.TaskKey, Kind: it.Kind, Priority: it.Priority, State: it.State,
			WaitReason: it.WaitReason, WaitRef: it.WaitRef, ExecutionID: it.ExecutionID, EnqueuedAt: it.EnqueuedAt, StartedAt: it.StartedAt, FinishedAt: it.FinishedAt}
		if it.StartedAt != nil {
			end := now
			if it.FinishedAt != nil {
				end = *it.FinishedAt
			}
			v.Seconds = int64(end.Sub(*it.StartedAt).Seconds())
		}
		switch it.State {
		case queue.StateWaiting:
			v.Blocked = waitText(it)
		case queue.StateQueued:
			if reasons, err := q.Explain(ctx, it.ID); err == nil && len(reasons) > 0 {
				v.Blocked = strings.Join(reasons, "; ")
			}
		}
		views = append(views, v)
	}
	rows, err := s.db.Query(ctx, `SELECT e.id, e.feature_id, e.state, e.attempt, COALESCE((SELECT max(seq) FROM execution_events ev WHERE ev.execution_id = e.id), 0)
		FROM executions e WHERE e.project_id = $1 ORDER BY e.created_at DESC LIMIT 20`, id)
	if err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return
	}
	execs := []executionView{}
	for rows.Next() {
		var e executionView
		if err := rows.Scan(&e.ID, &e.FeatureID, &e.State, &e.Attempt, &e.LastSeq); err != nil {
			rows.Close()
			s.writeDomainError(w, r, persistence.MapError(err))
			return
		}
		execs = append(execs, e)
	}
	rows.Close()
	var lastSync *syncsource.Result
	if res, err := s.syncer().Last(ctx, id); err == nil {
		res.Changes = nil
		lastSync = &res
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"project": p, "controls": map[string]any{"global": global, "project": proj},
		"counts": counts, "items": views, "executions": execs, "last_sync": lastSync,
	})
}

type eventView struct {
	Seq       int64           `json:"seq"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
}

func (s *Server) executionOf(w http.ResponseWriter, r *http.Request) (project, execution string, ok bool) {
	project, ok = s.projectID(w, r)
	if !ok {
		return "", "", false
	}
	execution = r.PathValue("executionId")
	if !uuidPattern.MatchString(execution) {
		writeError(w, r, http.StatusNotFound, "not_found", "Recurso não encontrado.", "Verifique o identificador.")
		return "", "", false
	}
	var one int
	if err := s.db.QueryRow(r.Context(), `SELECT 1 FROM executions WHERE id = $1 AND project_id = $2`, execution, project).Scan(&one); err != nil {
		s.writeDomainError(w, r, persistence.MapError(err))
		return "", "", false
	}
	return project, execution, true
}

func cursorOf(r *http.Request) (int64, bool) {
	raw := r.Header.Get("Last-Event-ID")
	if raw == "" {
		raw = r.URL.Query().Get("after")
	}
	if raw == "" {
		return 0, true
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n < 0 {
		return 0, false
	}
	return n, true
}

// fetchEvents lê eventos após o cursor, com busca opcional por texto no tipo ou no conteúdo.
func (s *Server) fetchEvents(ctx context.Context, project, execution string, after int64, q string, limit int) ([]eventView, error) {
	if limit < 1 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.Query(ctx, `SELECT seq, type, payload, created_at FROM execution_events
		WHERE project_id = $1 AND execution_id = $2 AND seq > $3 AND ($4 = '' OR type ILIKE '%' || $4 || '%' OR payload::text ILIKE '%' || $4 || '%')
		ORDER BY seq LIMIT $5`, project, execution, after, q, limit)
	if err != nil {
		return nil, persistence.MapError(err)
	}
	defer rows.Close()
	out := []eventView{}
	for rows.Next() {
		var e eventView
		var payload []byte
		if err := rows.Scan(&e.Seq, &e.Type, &payload, &e.CreatedAt); err != nil {
			return nil, persistence.MapError(err)
		}
		e.Payload = json.RawMessage(s.redactor.RedactBytes(payload))
		out = append(out, e)
	}
	return out, persistence.MapError(rows.Err())
}

// executionEvents é o histórico pesquisável: eventos após o cursor, com `q` e `limit`.
func (s *Server) executionEvents(w http.ResponseWriter, r *http.Request) {
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
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 100 {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "A busca é longa demais.", "Use até 100 caracteres.")
		return
	}
	evs, err := s.fetchEvents(r.Context(), project, execution, after, q, limit)
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

// executionStream envia os eventos ao vivo (SSE). A reconexão recupera o que faltou pelo cursor
// (cabeçalho Last-Event-ID ou ?after=): nada se perde e nada se repete. Cada evento leva
// `id: <seq>` e o tipo dentro do JSON (evento padrão `message`, para o EventSource do navegador); há heartbeat e a conexão é encerrada após StreamMaxAge para o cliente reconectar
// com o cursor. O conteúdo passa pela redação de segredos.
func (s *Server) executionStream(w http.ResponseWriter, r *http.Request) {
	project, execution, ok := s.executionOf(w, r)
	if !ok {
		return
	}
	after, valid := cursorOf(r)
	if !valid {
		writeError(w, r, http.StatusBadRequest, "invalid_request", "Cursor inválido.", "Use o número do último evento recebido.")
		return
	}
	rc := http.NewResponseController(w)
	_ = rc.SetWriteDeadline(time.Time{}) // o fluxo é longo: sem o prazo de escrita padrão
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
			evs, err := s.fetchEvents(r.Context(), project, execution, after, "", 200)
			if err != nil {
				return false
			}
			for _, e := range evs {
				data, _ := json.Marshal(map[string]any{"seq": e.Seq, "type": e.Type, "payload": e.Payload, "created_at": e.CreatedAt})
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
