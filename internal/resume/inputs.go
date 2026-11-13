package resume

import (
	"fmt"
	"sort"
	"strings"
)

// Requirement é o requisito da feature como a origem o expõe (normalizado).
type Requirement struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	Priority     string   `json:"priority"`
	Labels       []string `json:"labels"`
	Dependencies []string `json:"dependencies"`
}

// Inputs são os insumos de uma execução num ponto reconciliado.
type Inputs struct {
	Refs        map[string]string `json:"refs"` // ramo, SHA da base, PR…
	Requirement Requirement       `json:"requirement"`
	ConfigHash  string            `json:"config_hash"` // hash do workflow congelado + configuração efetiva
}

// Classes de diferença.
const (
	Compatible = "compatible"
	Conflict   = "conflict"
)

// Diff é uma diferença entre o baseline e o estado atual.
type Diff struct {
	Kind   string `json:"kind"` // ref | requirement | config
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
	Class  string `json:"class"`
	Reason string `json:"reason"`
}

// Options define como julgar as diferenças.
type Options struct {
	// FastForward diz se `after` descende de `before` (ref avançou sem reescrever
	// histórico). Nil ou erro tratam a mudança de ref como conflito (falha fechada).
	FastForward func(ref, before, after string) (bool, error)
}

func sorted(in []string) []string {
	out := append([]string{}, in...)
	for i := range out {
		out[i] = strings.TrimSpace(out[i])
	}
	sort.Strings(out)
	return out
}

func eq(a, b []string) bool {
	a, b = sorted(a), sorted(b)
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Reconcile compara o baseline com o estado atual e classifica cada diferença.
// Prioridade, rótulos e estado (enquanto o item segue elegível) são compatíveis;
// título, descrição e dependências do requisito, ref divergente e configuração
// congelada diferente são conflitos.
func Reconcile(base, cur Inputs, opt Options) []Diff {
	var out []Diff
	names := map[string]bool{}
	for k := range base.Refs {
		names[k] = true
	}
	for k := range cur.Refs {
		names[k] = true
	}
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b, a := base.Refs[k], cur.Refs[k]
		if b == a {
			continue
		}
		d := Diff{Kind: "ref", Field: k, Before: b, After: a, Class: Conflict, Reason: "a ref mudou e não foi possível provar que é um avanço compatível"}
		if b != "" && a != "" && opt.FastForward != nil {
			if ok, err := opt.FastForward(k, b, a); err == nil && ok {
				d.Class, d.Reason = Compatible, "a ref avançou sem reescrever o histórico (fast-forward)"
			} else if err == nil {
				d.Reason = "a ref divergiu do ponto em que o trabalho começou"
			}
		} else if b == "" || a == "" {
			d.Reason = "a ref passou a existir ou deixou de existir"
		}
		out = append(out, d)
	}
	br, cr := base.Requirement, cur.Requirement
	add := func(field, before, after, class, reason string) {
		out = append(out, Diff{Kind: "requirement", Field: field, Before: before, After: after, Class: class, Reason: reason})
	}
	if br.Title != cr.Title {
		add("title", br.Title, cr.Title, Conflict, "o título do requisito mudou: o plano pode não valer mais")
	}
	if br.Description != cr.Description {
		add("description", br.Description, cr.Description, Conflict, "a descrição do requisito mudou: o plano pode não valer mais")
	}
	if !eq(br.Dependencies, cr.Dependencies) {
		add("dependencies", strings.Join(sorted(br.Dependencies), ","), strings.Join(sorted(cr.Dependencies), ","), Conflict, "as dependências do requisito mudaram")
	}
	if br.Priority != cr.Priority {
		add("priority", br.Priority, cr.Priority, Compatible, "a prioridade só afeta a próxima seleção, sem preempção")
	}
	if !eq(br.Labels, cr.Labels) {
		add("labels", strings.Join(sorted(br.Labels), ","), strings.Join(sorted(cr.Labels), ","), Compatible, "rótulos não alteram o trabalho em andamento")
	}
	if br.Status != cr.Status {
		add("status", br.Status, cr.Status, Compatible, "o estado na origem mudou; a elegibilidade é reavaliada no preflight")
	}
	if base.ConfigHash != cur.ConfigHash {
		out = append(out, Diff{Kind: "config", Field: "config_hash", Before: base.ConfigHash, After: cur.ConfigHash, Class: Conflict,
			Reason: "a configuração efetiva congelada da execução é diferente da esperada"})
	}
	if out == nil {
		out = []Diff{}
	}
	return out
}

// HasConflict informa se alguma diferença é conflitante.
func HasConflict(ds []Diff) bool {
	for _, d := range ds {
		if d.Class == Conflict {
			return true
		}
	}
	return false
}

func (d Diff) String() string {
	return fmt.Sprintf("%s.%s: %q → %q (%s)", d.Kind, d.Field, d.Before, d.After, d.Class)
}
