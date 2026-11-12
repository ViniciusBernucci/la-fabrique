package workflows

import "sort"

// Tipos de nó. O catálogo abaixo é a fonte única de portas e campos usada pelo
// editor visual, pelo validador do servidor e, depois, pelo motor (T07-02).
const (
	TStart     = "start"
	TEnd       = "end"
	TAgent     = "agent"
	TSkill     = "skill"
	TScript    = "script"
	TChecks    = "checks"
	TGit       = "git"
	TDocs      = "docs"
	TCondition = "condition"
	TLoop      = "loop"
	TWait      = "wait"
	THuman     = "human_decision"
	TSubflow   = "subflow"
	TConverge  = "converge"
)

// Port names.
const (
	PortIn    = "in"
	PortOut   = "out"
	PortError = "error"
)

// Tipos de campo.
const (
	KindText     = "text"
	KindTextarea = "textarea"
	KindInt      = "int"
	KindEnum     = "enum"
)

// FieldSpec descreve uma propriedade editável de um nó.
type FieldSpec struct {
	Name     string   `json:"name"`
	Label    string   `json:"label"`
	Kind     string   `json:"kind"`
	Required bool     `json:"required"`
	Options  []string `json:"options,omitempty"`
	Min      int      `json:"min,omitempty"`
	Max      int      `json:"max,omitempty"`
	Hint     string   `json:"hint,omitempty"`
}

// NodeType descreve um tipo de nó: portas e propriedades.
type NodeType struct {
	Type     string      `json:"type"`
	Label    string      `json:"label"`
	Category string      `json:"category"` // fluxo | execução | controle
	Inputs   []string    `json:"inputs"`
	Outputs  []string    `json:"outputs"`
	MultiIn  bool        `json:"multi_in"` // aceita várias conexões de entrada
	Fields   []FieldSpec `json:"fields"`
}

var (
	profiles = []string{"base", "go", "node"}
	gitOps   = []string{"branch", "commit", "push", "draft_pr"}
)

// Catalog devolve os tipos de nó na ordem de exibição.
func Catalog() []NodeType {
	exec := []string{PortOut, PortError}
	return []NodeType{
		{Type: TStart, Label: "Início", Category: "fluxo", Inputs: nil, Outputs: []string{PortOut}},
		{Type: TEnd, Label: "Fim", Category: "fluxo", Inputs: []string{PortIn}, MultiIn: true, Fields: []FieldSpec{
			{Name: "outcome", Label: "Resultado", Kind: KindEnum, Required: true, Options: []string{"success", "failure"}},
		}},
		{Type: TAgent, Label: "Agente (IA)", Category: "execução", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "agent", Label: "Agente", Kind: KindText, Required: true, Hint: "Nome do agente no catálogo"},
			{Name: "prompt", Label: "Instrução", Kind: KindTextarea, Required: true},
			{Name: "timeout_minutes", Label: "Tempo máximo (min)", Kind: KindInt, Min: 1, Max: 720},
		}},
		{Type: TSkill, Label: "Skill", Category: "execução", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "skill", Label: "Skill", Kind: KindText, Required: true, Hint: "Nome da skill no catálogo"},
			{Name: "input", Label: "Entrada", Kind: KindTextarea},
		}},
		{Type: TScript, Label: "Script", Category: "execução", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "command", Label: "Comando", Kind: KindTextarea, Required: true},
			{Name: "profile", Label: "Perfil do sandbox", Kind: KindEnum, Required: true, Options: profiles},
			{Name: "timeout_minutes", Label: "Tempo máximo (min)", Kind: KindInt, Min: 1, Max: 720},
		}},
		{Type: TChecks, Label: "Testes e checks", Category: "execução", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "commands", Label: "Comandos (um por linha)", Kind: KindTextarea, Required: true},
			{Name: "profile", Label: "Perfil do sandbox", Kind: KindEnum, Required: true, Options: profiles},
		}},
		{Type: TGit, Label: "Git", Category: "execução", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "operation", Label: "Operação", Kind: KindEnum, Required: true, Options: gitOps},
			{Name: "message", Label: "Mensagem", Kind: KindText},
		}},
		{Type: TDocs, Label: "Documentação", Category: "execução", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "target", Label: "Arquivo ou pasta", Kind: KindText, Required: true},
			{Name: "instruction", Label: "Instrução", Kind: KindTextarea, Required: true},
		}},
		{Type: TCondition, Label: "Condição (IF/ELSE)", Category: "controle", Inputs: []string{PortIn}, Outputs: []string{"true", "false"}, Fields: []FieldSpec{
			{Name: "expression", Label: "Expressão", Kind: KindText, Required: true, Hint: "ex.: checks.passed == true"},
		}},
		{Type: TLoop, Label: "Loop limitado", Category: "controle", Inputs: []string{PortIn}, Outputs: []string{"body", "done"}, MultiIn: true, Fields: []FieldSpec{
			{Name: "max_iterations", Label: "Máximo de iterações", Kind: KindInt, Required: true, Min: 1, Max: 100},
			{Name: "condition", Label: "Continuar enquanto", Kind: KindText, Required: true},
		}},
		{Type: TWait, Label: "Espera durável", Category: "controle", Inputs: []string{PortIn}, Outputs: []string{PortOut}, Fields: []FieldSpec{
			{Name: "minutes", Label: "Aguardar (min)", Kind: KindInt, Required: true, Min: 1, Max: 10080},
		}},
		{Type: THuman, Label: "Decisão humana", Category: "controle", Inputs: []string{PortIn}, Outputs: []string{"approved", "rejected"}, Fields: []FieldSpec{
			{Name: "question", Label: "Pergunta", Kind: KindTextarea, Required: true},
		}},
		{Type: TSubflow, Label: "Subfluxo", Category: "controle", Inputs: []string{PortIn}, Outputs: exec, Fields: []FieldSpec{
			{Name: "workflow_id", Label: "Subfluxo (ID)", Kind: KindText, Required: true, Hint: "ID de um workflow do tipo subfluxo"},
		}},
		{Type: TConverge, Label: "Convergência", Category: "controle", Inputs: []string{PortIn}, MultiIn: true, Outputs: []string{PortOut}, Fields: []FieldSpec{
			{Name: "mode", Label: "Modo", Kind: KindEnum, Required: true, Options: []string{"all", "any"}},
		}},
	}
}

var catalogByType = func() map[string]NodeType {
	m := map[string]NodeType{}
	for _, t := range Catalog() {
		m[t.Type] = t
	}
	return m
}()

// TypeOf devolve a definição de um tipo de nó.
func TypeOf(t string) (NodeType, bool) {
	nt, ok := catalogByType[t]
	return nt, ok
}

// Node é um nó do grafo. Props guarda valores de texto ou número.
type Node struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"`
	X     float64        `json:"x"`
	Y     float64        `json:"y"`
	Props map[string]any `json:"props"`
}

// Edge liga a porta de saída de um nó à porta de entrada de outro.
type Edge struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	FromPort string `json:"from_port"`
	To       string `json:"to"`
	ToPort   string `json:"to_port"`
}

// Graph é o documento de um workflow.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Limites do documento.
const (
	MaxNodes = 200
	MaxEdges = 400
)

// Issue localiza um problema: nó, campo (ou porta) e motivo. Node vazio indica
// problema do grafo inteiro.
type Issue struct {
	Node    string `json:"node"`
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func sortIssues(is []Issue) {
	sort.SliceStable(is, func(a, b int) bool {
		if is[a].Node != is[b].Node {
			return is[a].Node < is[b].Node
		}
		if is[a].Field != is[b].Field {
			return is[a].Field < is[b].Field
		}
		return is[a].Code < is[b].Code
	})
}
