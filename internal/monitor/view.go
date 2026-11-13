package monitor

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/ViniciusBernucci/le-fabrique/internal/persistence"
)

// Event é um evento do monitor para a tela.
type Event struct {
	Seq      int64          `json:"seq"`
	Kind     string         `json:"kind"`
	At       time.Time      `json:"at"`
	Text     string         `json:"text,omitempty"`
	Tool     string         `json:"tool,omitempty"`
	Subagent string         `json:"subagent,omitempty"`
	Progress *int           `json:"progress,omitempty"`
	Raw      map[string]any `json:"-"`
}

// Subagent resume um subagente identificado pelo executor.
type Subagent struct {
	Name   string `json:"name"`
	Events int    `json:"events"`
	State  string `json:"state"` // running | done
	// Progress só existe se o subagente emitiu telemetria de progresso; senão é nulo (nada é fabricado).
	Progress *int `json:"progress"`
}

// Summary resume a execução até o cursor lido.
type Summary struct {
	StartedAt      *time.Time `json:"started_at"`
	LastEventAt    *time.Time `json:"last_event_at"`
	ElapsedSeconds int64      `json:"elapsed_seconds"`
	Done           bool       `json:"done"`
	ToolCalls      int        `json:"tool_calls"`
	Errors         int        `json:"errors"`
	Dropped        int64      `json:"dropped"`  // eventos perdidos por pressão (lacunas)
	Progress       *int       `json:"progress"` // só o que o executor informou
	Subagents      []Subagent `json:"subagents"`
}

// View é a resposta do monitor.
type View struct {
	Events  []Event `json:"events"`
	Cursor  int64   `json:"cursor"` // último seq entregue: a reconexão continua dele
	Summary Summary `json:"summary"`
}

// Read devolve os eventos do monitor com seq > after e o resumo de TODA a execução. Reconectar com o
// cursor devolvido continua sem repetir eventos.
func Read(ctx context.Context, db persistence.DBTX, projectID, executionID string, after int64, limit int, now time.Time) (View, error) {
	v := View{Events: []Event{}, Cursor: after, Summary: Summary{Subagents: []Subagent{}}}
	all, err := persistence.EventsAfter(ctx, db, projectID, executionID, 0, 1000)
	if err != nil {
		return v, err
	}
	subs := map[string]*Subagent{}
	for _, pe := range all {
		if !strings.HasPrefix(pe.Type, TypePrefix) {
			continue
		}
		kind := strings.TrimPrefix(pe.Type, TypePrefix)
		var p map[string]any
		_ = json.Unmarshal(pe.Payload, &p)
		at, _ := time.Parse(time.RFC3339Nano, str(p["at"]))
		ev := Event{Seq: pe.Seq, Kind: kind, At: at, Text: str(p["text"]), Tool: str(p["tool"]), Subagent: str(p["subagent"])}
		if f, ok := p["progress"].(float64); ok {
			n := int(f)
			ev.Progress = &n
		}
		// Resumo (sobre todos os eventos).
		s := &v.Summary
		if !at.IsZero() {
			if s.StartedAt == nil {
				t := at
				s.StartedAt = &t
			}
			t := at
			s.LastEventAt = &t
		}
		switch kind {
		case "tool":
			s.ToolCalls++
		case "error":
			s.Errors++
		case "done":
			s.Done = true
		case "gap":
			if f, ok := p["dropped"].(float64); ok {
				s.Dropped += int64(f)
			}
		}
		if ev.Progress != nil && ev.Subagent == "" {
			s.Progress = ev.Progress
		}
		if ev.Subagent != "" {
			sa := subs[ev.Subagent]
			if sa == nil {
				sa = &Subagent{Name: ev.Subagent, State: "running"}
				subs[ev.Subagent] = sa
			}
			sa.Events++
			if kind == "done" {
				sa.State = "done"
			}
			if ev.Progress != nil {
				sa.Progress = ev.Progress
			}
		}
		if pe.Seq > after && len(v.Events) < max(limit, 1) {
			v.Events = append(v.Events, ev)
			v.Cursor = pe.Seq
		}
	}
	s := &v.Summary
	if s.StartedAt != nil {
		end := now
		if s.Done && s.LastEventAt != nil {
			end = *s.LastEventAt
		}
		s.ElapsedSeconds = int64(end.Sub(*s.StartedAt).Seconds())
	}
	for _, sa := range subs {
		if s.Done {
			sa.State = "done"
		}
		s.Subagents = append(s.Subagents, *sa)
	}
	sort.Slice(s.Subagents, func(i, j int) bool { return s.Subagents[i].Name < s.Subagents[j].Name })
	return v, nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
