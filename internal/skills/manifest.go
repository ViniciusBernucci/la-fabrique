package skills

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/ViniciusBernucci/le-fabrique/internal/workflows"
)

// Tipos de executor.
const (
	ExecAI            = "ai"
	ExecScript        = "script"
	ExecDeterministic = "deterministic"
	ExecHybrid        = "hybrid"
)

// Tipos de passo de uma skill híbrida ou determinística.
const (
	StepAI            = "ai"
	StepScript        = "script"
	StepDeterministic = "deterministic"
)

// Operações determinísticas.
const (
	OpSet      = "set"
	OpTemplate = "template"
	OpAssert   = "assert"
)

// Step é um passo do pipeline.
type Step struct {
	Type       string `json:"type"`
	Command    string `json:"command,omitempty"`    // script
	Op         string `json:"op,omitempty"`         // deterministic: set | template | assert
	Key        string `json:"key,omitempty"`        // set/template: variável de destino
	Value      string `json:"value,omitempty"`      // set: valor; template: modelo com ${variavel}
	Expression string `json:"expression,omitempty"` // assert
}

// Manifest descreve a skill (skill.json ou cabeçalho do SKILL.md).
type Manifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Executor    string   `json:"executor"`
	Permissions []string `json:"permissions"`
	Resources   []string `json:"resources"` // arquivos do pacote usados pela skill
	Commands    []string `json:"commands"`  // skill de script
	Steps       []Step   `json:"steps"`     // determinística e híbrida
}

var (
	namePattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	permPattern  = regexp.MustCompile(`^[a-z][a-z_:-]{0,30}$`)
	keyPattern   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]{0,63}$`)
	placeholders = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_.]*)\}`)
)

// Limites.
const (
	MaxBody     = 60000
	MaxSteps    = 30
	MaxCommands = 30
)

// ValidationError aponta o problema do manifesto.
type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

func bad(field, msg string) error { return &ValidationError{field, msg} }

// ParseFrontmatter lê um cabeçalho simples "---\nchave: valor\n---" (só valores
// de texto) e devolve os campos e o corpo restante.
func ParseFrontmatter(src string) (map[string]string, string, error) {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	if !strings.HasPrefix(src, "---\n") {
		return nil, src, nil
	}
	rest := src[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return nil, "", bad("frontmatter", "cabeçalho sem fechamento (---)")
	}
	head, body := rest[:end], strings.TrimPrefix(rest[end+4:], "\n")
	fields := map[string]string{}
	for _, line := range strings.Split(head, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(k) == "" {
			return nil, "", bad("frontmatter", fmt.Sprintf("linha inválida: %q", clip(line)))
		}
		fields[strings.TrimSpace(k)] = strings.Trim(strings.TrimSpace(v), `"'`)
	}
	return fields, body, nil
}

func clip(s string) string {
	if len(s) > 60 {
		return s[:60] + "…"
	}
	return s
}

// ParseManifestJSON lê skill.json.
func ParseManifestJSON(raw []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return m, bad("skill.json", "JSON inválido ou com campo desconhecido")
	}
	return m, nil
}

// Validate confere o manifesto e a coerência com o executor. body é o texto do
// SKILL.md (instruções para skills de IA).
func (m *Manifest) Validate(body string) error {
	m.Name, m.Description = strings.TrimSpace(m.Name), strings.TrimSpace(m.Description)
	if !namePattern.MatchString(m.Name) {
		return bad("name", "O nome deve usar minúsculas, números, hífen e sublinhado (até 64).")
	}
	if len(m.Description) > 500 {
		return bad("description", "A descrição deve ter até 500 caracteres.")
	}
	if len(body) > MaxBody {
		return bad("body", "As instruções são longas demais.")
	}
	for _, p := range m.Permissions {
		if !permPattern.MatchString(p) {
			return bad("permissions", fmt.Sprintf("Permissão inválida: %q.", clip(p)))
		}
	}
	if len(m.Permissions) > 20 || len(m.Resources) > 100 {
		return bad("permissions", "Itens demais.")
	}
	if len(m.Commands) > MaxCommands || len(m.Steps) > MaxSteps {
		return bad("steps", "Passos ou comandos demais.")
	}
	for _, c := range m.Commands {
		if strings.TrimSpace(c) == "" || len(c) > 500 {
			return bad("commands", "Comando vazio ou longo demais.")
		}
	}
	switch m.Executor {
	case ExecAI:
		if strings.TrimSpace(body) == "" {
			return bad("body", "Uma skill de IA precisa de instruções (SKILL.md).")
		}
		if len(m.Commands) > 0 || len(m.Steps) > 0 {
			return bad("executor", "Uma skill de IA não declara comandos nem passos.")
		}
	case ExecScript:
		if len(m.Commands) == 0 {
			return bad("commands", "Uma skill de script precisa de ao menos um comando.")
		}
		if len(m.Steps) > 0 {
			return bad("executor", "Uma skill de script usa `commands`, não `steps`.")
		}
	case ExecDeterministic, ExecHybrid:
		if len(m.Steps) == 0 {
			return bad("steps", "Esta skill precisa de passos.")
		}
		if len(m.Commands) > 0 {
			return bad("executor", "Use `steps` (com passos de script) em vez de `commands`.")
		}
	default:
		return bad("executor", "O executor deve ser ai, script, deterministic ou hybrid.")
	}
	kinds := map[string]bool{}
	for i, s := range m.Steps {
		f := fmt.Sprintf("steps[%d]", i)
		kinds[s.Type] = true
		switch s.Type {
		case StepAI:
			if m.Executor != ExecHybrid {
				return bad(f, "Passo de IA só existe em skill híbrida.")
			}
			if strings.TrimSpace(body) == "" {
				return bad("body", "Uma skill com passo de IA precisa de instruções (SKILL.md).")
			}
		case StepScript:
			if m.Executor != ExecHybrid {
				return bad(f, "Passo de script só existe em skill híbrida.")
			}
			if strings.TrimSpace(s.Command) == "" || len(s.Command) > 500 {
				return bad(f, "Passo de script sem comando.")
			}
		case StepDeterministic:
			if err := validateOp(f, s); err != nil {
				return err
			}
		default:
			return bad(f, "Tipo de passo desconhecido.")
		}
	}
	if m.Executor == ExecDeterministic && (kinds[StepAI] || kinds[StepScript]) {
		return bad("steps", "Uma skill determinística só tem passos determinísticos.")
	}
	if m.Executor == ExecHybrid && len(kinds) < 2 {
		return bad("steps", "Uma skill híbrida combina ao menos dois tipos de passo.")
	}
	if m.Permissions == nil {
		m.Permissions = []string{}
	}
	if m.Resources == nil {
		m.Resources = []string{}
	}
	if m.Commands == nil {
		m.Commands = []string{}
	}
	if m.Steps == nil {
		m.Steps = []Step{}
	}
	return nil
}

func validateOp(f string, s Step) error {
	switch s.Op {
	case OpSet, OpTemplate:
		if !keyPattern.MatchString(s.Key) {
			return bad(f, "Informe a variável de destino (key).")
		}
		if len(s.Value) > 2000 {
			return bad(f, "Valor longo demais.")
		}
	case OpAssert:
		if _, err := workflows.ParseExpr(s.Expression); err != nil {
			return bad(f, "Expressão da asserção inválida.")
		}
	default:
		return bad(f, "Operação determinística desconhecida (set, template ou assert).")
	}
	return nil
}
