package planning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Limites do plano.
const (
	MaxTasks      = 50
	MaxCriteria   = 12
	MaxDepends    = 20
	minTextLen    = 8
	minEvidence   = 15
	maxTitleLen   = 200
	maxDescLen    = 4000
	maxPathLength = 300
)

// Tipos de verificação de um critério.
const (
	CheckCommand = "command" // comando cujo código de saída decide
	CheckTest    = "test"    // teste automatizado (alvo)
	CheckFile    = "file"    // arquivo que deve existir (caminho relativo)
	CheckReview  = "review"  // revisão humana/agente de uma evidência observável
)

// Riscos iniciais.
var risks = map[string]bool{"low": true, "medium": true, "high": true}

var keyPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,39}$`)

// Criterion é um critério de aceite verificável.
type Criterion struct {
	Text     string `json:"text"`
	Check    string `json:"check"`
	Command  string `json:"command,omitempty"`
	Target   string `json:"target,omitempty"`
	Path     string `json:"path,omitempty"`
	Evidence string `json:"evidence,omitempty"`
}

// Task é uma tarefa técnica do plano.
type Task struct {
	Key         string      `json:"key"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Criteria    []Criterion `json:"criteria"`
	Risk        string      `json:"risk"`
	DependsOn   []string    `json:"depends_on"`
	Order       int         `json:"order"` // estimativa de ordem (desempate)
	Paths       []string    `json:"paths,omitempty"`
	// Trilha e recursos (adequação, LF-AJ-05): classificação, indivisibilidade e recursos compartilhados.
	Track           string   `json:"track,omitempty"`
	Indivisible     bool     `json:"indivisible,omitempty"`
	ExclusiveBoth   bool     `json:"exclusive_both,omitempty"`
	SharedResources []string `json:"shared_resources,omitempty"`
	// Refs liga a tarefa de uma correção aos comentários da rodada que ela trata (comment:<id>, review:<id>…).
	Refs []string `json:"refs,omitempty"`
}

// Plan é o plano de uma feature.
type Plan struct {
	Summary string `json:"summary"`
	Tasks   []Task `json:"tasks"`
}

// Problem é um defeito do plano.
type Problem struct {
	Task    string `json:"task,omitempty"`
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (p Problem) String() string {
	if p.Task != "" {
		return p.Task + "." + p.Field + ": " + p.Message
	}
	return p.Field + ": " + p.Message
}

func validRelPath(p string) bool {
	if p == "" || len(p) > maxPathLength || strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\x00\n\r\\") {
		return false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." || seg == "" {
			return false
		}
	}
	return true
}

func validCriterion(c Criterion) string {
	if len(strings.TrimSpace(c.Text)) < minTextLen {
		return "descreva o critério (mínimo de 8 caracteres)"
	}
	switch c.Check {
	case CheckCommand:
		if strings.TrimSpace(c.Command) == "" || strings.ContainsAny(c.Command, "\x00\r\n") {
			return "critério de comando exige o comando (uma linha)"
		}
	case CheckTest:
		if strings.TrimSpace(c.Target) == "" {
			return "critério de teste exige o teste ou pacote alvo"
		}
	case CheckFile:
		if !validRelPath(c.Path) {
			return "critério de arquivo exige um caminho relativo válido"
		}
	case CheckReview:
		if len(strings.TrimSpace(c.Evidence)) < minEvidence {
			return "critério de revisão exige a evidência observável esperada (mínimo de 15 caracteres)"
		}
	default:
		return "o critério não é verificável: informe check = command, test, file ou review"
	}
	return ""
}

// Validate recusa o que não pode virar trabalho verificável. Pura e determinística.
func Validate(p Plan) []Problem {
	var out []Problem
	add := func(task, field, msg string) { out = append(out, Problem{task, field, msg}) }
	if len(p.Tasks) == 0 {
		add("", "tasks", "o plano precisa de ao menos uma tarefa")
		return out
	}
	if len(p.Tasks) > MaxTasks {
		add("", "tasks", fmt.Sprintf("o plano tem %d tarefas (máximo %d): divida a feature", len(p.Tasks), MaxTasks))
	}
	keys := map[string]bool{}
	for _, t := range p.Tasks {
		if !keyPattern.MatchString(t.Key) {
			add(t.Key, "key", "chave inválida (minúsculas, números, - e _; até 40 caracteres)")
			continue
		}
		if keys[t.Key] {
			add(t.Key, "key", "chave repetida")
		}
		keys[t.Key] = true
	}
	for _, t := range p.Tasks {
		if !keyPattern.MatchString(t.Key) {
			continue
		}
		if title := strings.TrimSpace(t.Title); title == "" || len(title) > maxTitleLen {
			add(t.Key, "title", "título obrigatório (até 200 caracteres)")
		}
		if len(t.Description) > maxDescLen {
			add(t.Key, "description", "descrição acima de 4000 caracteres")
		}
		if !risks[t.Risk] {
			add(t.Key, "risk", "informe o risco inicial: low, medium ou high")
		}
		if t.Order < 0 {
			add(t.Key, "order", "a ordem estimada não pode ser negativa")
		}
		if len(t.Criteria) == 0 {
			add(t.Key, "criteria", "a tarefa precisa de ao menos um critério verificável")
		}
		if len(t.Criteria) > MaxCriteria {
			add(t.Key, "criteria", fmt.Sprintf("mais de %d critérios: divida a tarefa", MaxCriteria))
		}
		for i, c := range t.Criteria {
			if msg := validCriterion(c); msg != "" {
				add(t.Key, fmt.Sprintf("criteria[%d]", i), msg)
			}
		}
		for _, pth := range t.Paths {
			if !validRelPath(pth) {
				add(t.Key, "paths", fmt.Sprintf("caminho inválido: %q", pth))
			}
		}
		if len(t.DependsOn) > MaxDepends {
			add(t.Key, "depends_on", "dependências demais")
		}
		seen := map[string]bool{}
		for _, d := range t.DependsOn {
			switch {
			case d == t.Key:
				add(t.Key, "depends_on", "a tarefa não pode depender de si mesma")
			case !keys[d]:
				add(t.Key, "depends_on", fmt.Sprintf("referência ausente: %q", d))
			case seen[d]:
				add(t.Key, "depends_on", fmt.Sprintf("dependência repetida: %q", d))
			}
			seen[d] = true
		}
	}
	if len(out) == 0 {
		if cycle := findCycle(p); cycle != nil {
			add(cycle[0], "depends_on", "ciclo de dependências: "+strings.Join(cycle, " → "))
		}
	}
	return out
}

// findCycle devolve um ciclo (a → b → … → a) ou nil.
func findCycle(p Plan) []string {
	deps := map[string][]string{}
	keys := make([]string, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		d := append([]string{}, t.DependsOn...)
		sort.Strings(d)
		deps[t.Key] = d
		keys = append(keys, t.Key)
	}
	sort.Strings(keys)
	const (
		white = 0
		grey  = 1
		black = 2
	)
	color := map[string]int{}
	var stack []string
	var cycle []string
	var visit func(k string) bool
	visit = func(k string) bool {
		color[k] = grey
		stack = append(stack, k)
		for _, d := range deps[k] {
			if color[d] == grey {
				i := len(stack) - 1
				for stack[i] != d {
					i--
				}
				cycle = append(append([]string{}, stack[i:]...), d)
				return true
			}
			if color[d] == white && visit(d) {
				return true
			}
		}
		stack = stack[:len(stack)-1]
		color[k] = black
		return false
	}
	for _, k := range keys {
		if color[k] == white && visit(k) {
			return cycle
		}
	}
	return nil
}

// Order devolve as chaves em ordem topológica estável: só libera uma tarefa
// depois de suas dependências; entre as liberadas, menor Order e depois a chave.
// Pressupõe plano válido.
func Order(p Plan) []string {
	by := map[string]Task{}
	indeg := map[string]int{}
	next := map[string][]string{}
	for _, t := range p.Tasks {
		by[t.Key] = t
		indeg[t.Key] += 0
	}
	for _, t := range p.Tasks {
		for _, d := range t.DependsOn {
			indeg[t.Key]++
			next[d] = append(next[d], t.Key)
		}
	}
	less := func(a, b string) bool {
		if by[a].Order != by[b].Order {
			return by[a].Order < by[b].Order
		}
		return a < b
	}
	var ready []string
	for k, n := range indeg {
		if n == 0 {
			ready = append(ready, k)
		}
	}
	var out []string
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return less(ready[i], ready[j]) })
		k := ready[0]
		ready = ready[1:]
		out = append(out, k)
		for _, n := range next[k] {
			if indeg[n]--; indeg[n] == 0 {
				ready = append(ready, n)
			}
		}
	}
	return out
}

func canonical(v any) []byte {
	b, _ := json.Marshal(v)
	var g any
	if json.Unmarshal(b, &g) != nil {
		return b
	}
	out, _ := json.Marshal(g)
	return out
}

func sha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// normalizeTask ordena o que não tem ordem própria, para o hash não depender dela.
func normalizeTask(t Task) Task {
	t.DependsOn = append([]string{}, t.DependsOn...)
	sort.Strings(t.DependsOn)
	t.Paths = append([]string{}, t.Paths...)
	sort.Strings(t.Paths)
	if t.SharedResources != nil {
		t.SharedResources = append([]string{}, t.SharedResources...)
		sort.Strings(t.SharedResources)
	}
	if t.Refs != nil {
		t.Refs = append([]string{}, t.Refs...)
		sort.Strings(t.Refs)
	}
	if t.Criteria == nil {
		t.Criteria = []Criterion{}
	}
	return t
}

// TaskHash é o hash do conteúdo de uma tarefa (sem a ordem estimada).
func TaskHash(t Task) string {
	t = normalizeTask(t)
	t.Order = 0
	return sha(canonical(t))
}

// Normalize devolve o plano em forma canônica (tarefas por chave).
func Normalize(p Plan) Plan {
	ts := make([]Task, 0, len(p.Tasks))
	for _, t := range p.Tasks {
		ts = append(ts, normalizeTask(t))
	}
	sort.Slice(ts, func(i, j int) bool { return ts[i].Key < ts[j].Key })
	return Plan{Summary: strings.TrimSpace(p.Summary), Tasks: ts}
}

// Hash é o hash do plano normalizado.
func Hash(p Plan) string { return sha(canonical(Normalize(p))) }
