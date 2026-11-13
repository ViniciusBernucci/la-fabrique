package syncsource

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/integrations"
)

// Source é a configuração da origem usada na leitura.
type Source struct {
	Kind             string
	Container        string
	Marker           string // rótulo/tag de elegibilidade
	EligibleStatuses []string
	PriorityMap      map[string]int // prioridade da origem (minúsculas) → 0..4
}

// Item é o item normalizado e gravado.
type Item struct {
	RemoteID       string    `json:"remote_id"`
	Key            string    `json:"key"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Status         string    `json:"status"`
	Category       string    `json:"category"`
	Priority       string    `json:"priority"`
	MappedPriority *int      `json:"mapped_priority"`
	Labels         []string  `json:"labels"`
	Dependencies   []string  `json:"dependencies"`
	Subtask        bool      `json:"subtask"`
	Eligible       bool      `json:"eligible"`
	Version        string    `json:"version"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	LocalPriority  *int      `json:"local_priority,omitempty"` // edição humana
	Conflict       *Conflict `json:"conflict,omitempty"`
	Hash           string    `json:"content_hash"`
}

// Conflict é uma mudança remota que colide com edição humana local.
type Conflict struct {
	Field  string `json:"field"`
	Local  int    `json:"local"`
	Remote *int   `json:"remote"`
}

// EffectivePriority é a prioridade que vale na fábrica: a local, se houver.
func (i Item) EffectivePriority() *int {
	if i.LocalPriority != nil {
		return i.LocalPriority
	}
	return i.MappedPriority
}

func lower(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, strings.ToLower(strings.TrimSpace(s)))
	}
	return out
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == strings.ToLower(strings.TrimSpace(v)) {
			return true
		}
	}
	return false
}

// normalize converte o item remoto no formato gravado. checkMarker é verdadeiro
// quando o item não veio de uma listagem filtrada pelo marcador (preflight).
func normalize(it integrations.Item, src Source, checkMarker bool) Item {
	labels := append([]string{}, it.Labels...)
	sort.Strings(labels)
	deps := append([]string{}, it.Dependencies...)
	sort.Strings(deps)
	n := Item{RemoteID: it.ID, Key: it.Key, Title: it.Title, Description: it.Description, Status: it.Status,
		Category: it.Category, Priority: it.Priority, Labels: labels, Dependencies: deps, Subtask: it.Subtask,
		Version: it.Version, URL: it.URL, State: "active"}
	if p, ok := src.PriorityMap[strings.ToLower(strings.TrimSpace(it.Priority))]; ok {
		v := p
		n.MappedPriority = &v
	}
	eligible := !it.Subtask && it.Category != integrations.CategoryDone && contains(lower(src.EligibleStatuses), it.Status)
	if checkMarker && src.Marker != "" && !contains(lower(labels), src.Marker) {
		eligible = false
	}
	n.Eligible = eligible
	n.Hash = hashOf(n)
	return n
}

// hashOf é o hash do conteúdo normalizado (sem a versão remota nem estado local).
func hashOf(n Item) string {
	b, _ := json.Marshal(struct {
		T, D, S, C, P string
		M             *int
		L, Dep        []string
		Sub, E        bool
	}{n.Title, n.Description, n.Status, n.Category, n.Priority, n.MappedPriority, n.Labels, n.Dependencies, n.Subtask, n.Eligible})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// FieldChange é uma diferença de campo.
type FieldChange struct {
	From any `json:"from"`
	To   any `json:"to"`
}

// diff lista os campos que mudaram entre o gravado e o lido.
func diff(prev, cur Item) map[string]FieldChange {
	out := map[string]FieldChange{}
	add := func(name string, a, b any) {
		ja, _ := json.Marshal(a)
		jb, _ := json.Marshal(b)
		if string(ja) != string(jb) {
			out[name] = FieldChange{a, b}
		}
	}
	add("title", prev.Title, cur.Title)
	add("description", prev.Description, cur.Description)
	add("status", prev.Status, cur.Status)
	add("priority", prev.Priority, cur.Priority)
	add("mapped_priority", prev.MappedPriority, cur.MappedPriority)
	add("labels", prev.Labels, cur.Labels)
	add("dependencies", prev.Dependencies, cur.Dependencies)
	add("subtask", prev.Subtask, cur.Subtask)
	add("eligible", prev.Eligible, cur.Eligible)
	return out
}

func snapshotHash(items []Item) string {
	es := append([]Item{}, items...)
	sort.Slice(es, func(i, j int) bool { return es[i].RemoteID < es[j].RemoteID })
	h := sha256.New()
	for _, e := range es {
		h.Write([]byte(e.RemoteID + "\x00" + e.Hash + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}
